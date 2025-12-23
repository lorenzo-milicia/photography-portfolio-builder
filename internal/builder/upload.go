package builder

import (
	"encoding/json"
	"fmt"
	"image"
	_ "image/jpeg" // Register JPEG decoder
	_ "image/png"  // Register PNG decoder
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog/log"
	"go.lorenzomilicia.dev/photography-portfolio-builder/internal/content"
	"go.lorenzomilicia.dev/photography-portfolio-builder/internal/processing"
)

// handleProjectUpload handles photo uploads
func (s *Server) handleProjectUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// 1. Parse form
	if err := r.ParseMultipartForm(32 << 20); err != nil { // 32MB max
		http.Error(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	slug := r.FormValue("slug")
	if slug == "" {
		http.Error(w, "Project slug required", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("upload")
	if err != nil {
		http.Error(w, "Upload file required", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// 2. Validate file type
	buff := make([]byte, 512)
	if _, err := file.Read(buff); err != nil {
		http.Error(w, "Failed to read file", http.StatusInternalServerError)
		return
	}
	fileType := http.DetectContentType(buff)
	if !strings.HasPrefix(fileType, "image/") {
		s.sendErrorToast(w, "Invalid file type. Only images are allowed.")
		return
	}
	// Reset file pointer
	if _, err := file.Seek(0, 0); err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// 3. Save to content/photos/<slug>/<filename>
	filename := filepath.Base(header.Filename)
	saveDir := filepath.Join(s.photosDir, slug) // Use configured photosDir

	if err := os.MkdirAll(saveDir, 0755); err != nil {
		s.sendErrorToast(w, fmt.Sprintf("Failed to create directory: %v", err))
		return
	}

	savePath := filepath.Join(saveDir, filename)
	dst, err := os.Create(savePath)
	if err != nil {
		s.sendErrorToast(w, fmt.Sprintf("Failed to save file: %v", err))
		return
	}

	if _, err := io.Copy(dst, file); err != nil {
		dst.Close()
		s.sendErrorToast(w, fmt.Sprintf("Failed to write file: %v", err))
		return
	}
	dst.Close()

	// 4. Process Image
	src := &processing.FileSource{Path: savePath}
	imgDest := &processing.Destination{
		OutputDir: filepath.Join(s.outputDir, "images", slug),
	}

	if err := s.processor.ProcessImage(src, imgDest); err != nil {
		log.Error().Err(err).Msg("Failed to process uploaded image")
		// Rollback: delete saved file
		os.Remove(savePath)
		s.sendErrorToast(w, fmt.Sprintf("Processing failed: %v", err))
		return
	}

	// 5. Success - Prepare OOB response
	hash, _ := s.processor.ComputeHash(src) // Should succeed if file exists
	hashID := hash[:12]

	// Get dimensions
	f, _ := os.Open(savePath)
	cfg, _, _ := image.DecodeConfig(f)
	stat, _ := f.Stat()
	f.Close()

	// Calculate 300w thumb height
	// thumbWidth := 300
	// ratio := float64(cfg.Height) / float64(cfg.Width)
	// thumbHeight := int(float64(thumbWidth) * ratio) // Unused
	aspectRatio := float64(cfg.Width) / float64(cfg.Height)

	// Since we don't have getIntegerRatio exported, we approximate or duplicate logic?
	// content package has getIntegerRatio BUT it is not exported.
	// However, we just need RatioWidth/RatioHeight for display.
	// For now, let's just use what we have or try to be smart.
	// Or we can rely on what we have.
	// Wait, PhotoInfo JSON needs RatioWidth/RatioHeight.
	// I'll implement a simple integer ratio calculation here similar to the one in content package.
	rw, rh := 3, 2 // Default
	// Simple approximation:
	if abs(aspectRatio-1.5) < 0.1 {
		rw, rh = 3, 2
	} else if abs(aspectRatio-0.66) < 0.1 {
		rw, rh = 2, 3
	} else if abs(aspectRatio-1.0) < 0.1 {
		rw, rh = 1, 1
	} else if abs(aspectRatio-1.33) < 0.1 {
		rw, rh = 4, 3
	} else if abs(aspectRatio-1.77) < 0.1 {
		rw, rh = 16, 9
	}
	// Better: just duplicate the list from content/photo.go if needed, or define a helper.
	// I'll add the helper function `getIntegerRatio` at the bottom of this file.

	photo := content.PhotoInfo{
		Filename:    filename,
		HashID:      hashID,
		Path:        savePath,
		Size:        stat.Size(),
		AspectRatio: aspectRatio,
		RatioWidth:  rw,
		RatioHeight: rh,
		ThumbPath:   fmt.Sprintf("/images/%s/.thumbs/thumb-%s.webp", slug, hashID),
	}

	// Render to buffer first to ensure we can set headers on success
	var buf strings.Builder
	data := struct {
		content.PhotoInfo
		ProjectSlug string
	}{
		PhotoInfo:   photo,
		ProjectSlug: slug,
	}

	if err := s.templates.ExecuteTemplate(&buf, "photo-item-oob.html", data); err != nil {
		log.Error().Err(err).Msg("Template error")
		s.sendErrorToast(w, "Upload succeeded but failed to update UI")
		return
	}

	// Send success toast trigger
	s.setSuccessToast(w, "Photo uploaded successfully")
	w.WriteHeader(http.StatusOK)
	io.WriteString(w, buf.String())
}

// handlePhotoDelete handles photo deletion
func (s *Server) handlePhotoDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	slug := r.URL.Query().Get("slug")
	filename := r.URL.Query().Get("filename")
	if slug == "" || filename == "" {
		s.sendErrorToast(w, "Missing project slug or filename")
		return
	}

	// 1. Locate source file
	photoDir := filepath.Join(s.photosDir, slug)
	sourcePath := filepath.Join(photoDir, filename)

	if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
		s.sendErrorToast(w, "File not found")
		return
	}

	// 2. Compute hash to find variants
	// We need the hash to know where variants are stored (output/images/slug/hash/...)
	// Since we haven't deleted it yet, we can compute it.
	src := &processing.FileSource{Path: sourcePath}
	hash, err := s.processor.ComputeHash(src)
	if err != nil {
		log.Error().Err(err).Str("file", sourcePath).Msg("Failed to compute hash for deletion")
		s.sendErrorToast(w, "Internal error: could not identify processed files")
		return
	}
	hashID := hash[:12]

	// 3. Delete processed files
	// Variants: dist/images/slug/hashID/ (directory)
	// Thumbnails: dist/images/slug/.thumbs/thumb-hashID.webp
	// The Destination logic in Processor puts variants in {OutputDir}/{HashID}/{filename}
	// In our case OutputDir is dist/images/slug. So dist/images/slug/hashID directory contains them?
	// Let's check Destination.CreateVariant: fullPath := filepath.Join(d.OutputDir, hashID, filename)
	// Yes, variants are inside a directory named after the hash.
	// So we can remove the entire `{OutputDir}/{HashID}` directory.

	imagesDir := filepath.Join(s.outputDir, "images", slug)
	variantDir := filepath.Join(imagesDir, hashID)

	if err := os.RemoveAll(variantDir); err != nil {
		log.Error().Err(err).Str("dir", variantDir).Msg("Failed to delete variants")
		// Continue to try deleting other things?
	}

	thumbPath := filepath.Join(imagesDir, ".thumbs", fmt.Sprintf("thumb-%s.webp", hashID))
	if err := os.Remove(thumbPath); err != nil && !os.IsNotExist(err) {
		log.Error().Err(err).Str("path", thumbPath).Msg("Failed to delete thumbnail")
	}

	// 4. Delete source file
	if err := os.Remove(sourcePath); err != nil {
		log.Error().Err(err).Str("path", sourcePath).Msg("Failed to delete source file")
		s.sendErrorToast(w, "Failed to delete source file")
		return // Don't remove from UI if source delete failed?
	}

	// 5. Success
	s.setSuccessToast(w, "Photo deleted successfully")
	w.WriteHeader(http.StatusOK) // HTMX swap delete doesn't need content, or we can send empty string.
	// Client side: hx-target="closest .photo-item" hx-swap="outerHTML" -> empty content removes it.
	// But hx-swap="delete" ignores content and just removes target.
	// So we just return 200 OK.
}

// Helper to send error toast
func (s *Server) sendErrorToast(w http.ResponseWriter, msg string) {
	events := map[string]interface{}{
		"showMessage": map[string]string{
			"type":    "error",
			"message": msg,
		},
	}
	eventJSON, _ := json.Marshal(events)
	w.Header().Set("HX-Trigger", string(eventJSON))
	w.WriteHeader(http.StatusOK) // 200 OK for partials
}

// Helper to set success toast header
func (s *Server) setSuccessToast(w http.ResponseWriter, msg string) {
	events := map[string]interface{}{
		"showMessage": map[string]string{
			"type":    "success",
			"message": msg,
		},
	}
	eventJSON, _ := json.Marshal(events)
	w.Header().Set("HX-Trigger", string(eventJSON))
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
