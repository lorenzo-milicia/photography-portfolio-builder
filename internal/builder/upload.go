package builder

import (
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

	rw := NewResponseWriter(w)

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

	log.Info().Str("slug", slug).Str("filename", header.Filename).Msg("Processing photo upload")

	// 2. Validate file type
	buff := make([]byte, 512)
	if _, err := file.Read(buff); err != nil {
		http.Error(w, "Failed to read file", http.StatusInternalServerError)
		return
	}
	fileType := http.DetectContentType(buff)
	if !strings.HasPrefix(fileType, "image/") {
		rw.Error("Invalid file type. Only images are allowed.")
		w.WriteHeader(http.StatusOK)
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
		log.Error().Err(err).Str("dir", saveDir).Msg("Failed to create directory")
		rw.Error(fmt.Sprintf("Failed to create directory: %v", err))
		w.WriteHeader(http.StatusOK)
		return
	}

	savePath := filepath.Join(saveDir, filename)
	dst, err := os.Create(savePath)
	if err != nil {
		log.Error().Err(err).Str("path", savePath).Msg("Failed to save file")
		rw.Error(fmt.Sprintf("Failed to save file: %v", err))
		w.WriteHeader(http.StatusOK)
		return
	}

	if _, err := io.Copy(dst, file); err != nil {
		dst.Close()
		log.Error().Err(err).Str("path", savePath).Msg("Failed to write file")
		rw.Error(fmt.Sprintf("Failed to write file: %v", err))
		w.WriteHeader(http.StatusOK)
		return
	}
	dst.Close()

	// 4. Process Image
	src := &processing.FileSource{Path: savePath}
	imgDest := &processing.Destination{
		OutputDir: filepath.Join(s.outputDir, "images", slug),
	}

	if err := s.processor.ProcessImage(src, imgDest); err != nil {
		log.Error().Err(err).Str("file", savePath).Msg("Failed to process uploaded image")
		// Rollback: delete saved file
		os.Remove(savePath)
		rw.Error(fmt.Sprintf("Processing failed: %v", err))
		w.WriteHeader(http.StatusOK)
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

	aspectRatio := float64(cfg.Width) / float64(cfg.Height)

	// Simple ratio approximation
	ratioWidth, ratioHeight := 3, 2 // Default
	if abs(aspectRatio-1.5) < 0.1 {
		ratioWidth, ratioHeight = 3, 2
	} else if abs(aspectRatio-0.66) < 0.1 {
		ratioWidth, ratioHeight = 2, 3
	} else if abs(aspectRatio-1.0) < 0.1 {
		ratioWidth, ratioHeight = 1, 1
	} else if abs(aspectRatio-1.33) < 0.1 {
		ratioWidth, ratioHeight = 4, 3
	} else if abs(aspectRatio-1.77) < 0.1 {
		ratioWidth, ratioHeight = 16, 9
	}

	photo := content.PhotoInfo{
		Filename:    filename,
		HashID:      hashID,
		Path:        savePath,
		Size:        stat.Size(),
		AspectRatio: aspectRatio,
		RatioWidth:  ratioWidth,
		RatioHeight: ratioHeight,
		ThumbPath:   fmt.Sprintf("/images/%s/.thumbs/thumb-%s.webp", slug, hashID),
	}

	log.Info().Str("slug", slug).Str("filename", filename).Str("hashID", hashID).Msg("Photo uploaded and processed successfully")

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
		respWriter := NewResponseWriter(w)
		respWriter.Error("Upload succeeded but failed to update UI")
		w.WriteHeader(http.StatusOK)
		return
	}

	// Send success toast trigger
	respWriter := NewResponseWriter(w)
	respWriter.Success("Photo uploaded successfully")
	w.WriteHeader(http.StatusOK)
	io.WriteString(w, buf.String())
}

// handlePhotoDelete handles photo deletion
func (s *Server) handlePhotoDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	rw := NewResponseWriter(w)

	slug := r.URL.Query().Get("slug")
	filename := r.URL.Query().Get("filename")
	if slug == "" || filename == "" {
		rw.Error("Missing project slug or filename")
		w.WriteHeader(http.StatusOK)
		return
	}

	log.Info().Str("slug", slug).Str("filename", filename).Msg("Deleting photo")

	// 1. Locate source file
	photoDir := filepath.Join(s.photosDir, slug)
	sourcePath := filepath.Join(photoDir, filename)

	if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
		rw.Error("File not found")
		w.WriteHeader(http.StatusOK)
		return
	}

	// 2. Compute hash to find variants
	src := &processing.FileSource{Path: sourcePath}
	hash, err := s.processor.ComputeHash(src)
	if err != nil {
		log.Error().Err(err).Str("file", sourcePath).Msg("Failed to compute hash for deletion")
		rw.Error("Internal error: could not identify processed files")
		w.WriteHeader(http.StatusOK)
		return
	}
	hashID := hash[:12]

	// 3. Delete processed files
	imagesDir := filepath.Join(s.outputDir, "images", slug)
	variantDir := filepath.Join(imagesDir, hashID)

	if err := os.RemoveAll(variantDir); err != nil {
		log.Error().Err(err).Str("dir", variantDir).Msg("Failed to delete variants")
	}

	thumbPath := filepath.Join(imagesDir, ".thumbs", fmt.Sprintf("thumb-%s.webp", hashID))
	if err := os.Remove(thumbPath); err != nil && !os.IsNotExist(err) {
		log.Error().Err(err).Str("path", thumbPath).Msg("Failed to delete thumbnail")
	}

	// 4. Delete source file
	if err := os.Remove(sourcePath); err != nil {
		log.Error().Err(err).Str("path", sourcePath).Msg("Failed to delete source file")
		rw.Error("Failed to delete source file")
		w.WriteHeader(http.StatusOK)
		return
	}

	log.Info().Str("slug", slug).Str("filename", filename).Msg("Photo deleted successfully")

	// 5. Success
	rw.Success("Photo deleted successfully")
	w.WriteHeader(http.StatusOK)
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
