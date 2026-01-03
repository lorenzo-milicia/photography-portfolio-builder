package content

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"image"
	"image/jpeg"
	_ "image/png"
	"io"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/image/draw"
)

// PhotoInfo holds information about a photo
type PhotoInfo struct {
	Filename    string  `json:"filename"` // Original filename
	HashID      string  `json:"hashId"`   // 12-char hash ID used in layout.yaml and processed images
	Path        string  `json:"path"`
	Size        int64   `json:"size"`
	AspectRatio float64 `json:"aspectRatio"` // width / height (deprecated, use RatioWidth/RatioHeight)
	RatioWidth  int     `json:"ratioWidth"`  // integer width of aspect ratio (e.g., 3 for 3:2)
	RatioHeight int     `json:"ratioHeight"` // integer height of aspect ratio (e.g., 2 for 3:2)
	ThumbPath   string  `json:"thumbPath"`   // path to thumbnail for builder UI
}

// ListPhotos returns all photos for a project by reading from the source photos directory
func (m *Manager) ListPhotos(slug string) ([]*PhotoInfo, error) {
	photosDir := m.ProjectPhotosDir(slug)

	entries, err := os.ReadDir(photosDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []*PhotoInfo{}, nil
		}
		return nil, fmt.Errorf("failed to read photos directory: %w", err)
	}

	var photos []*PhotoInfo
	for _, entry := range entries {
		if entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if ext != ".jpg" && ext != ".jpeg" && ext != ".png" && ext != ".webp" {
			continue
		}

		photoPath := filepath.Join(photosDir, entry.Name())

		// Compute hash ID from photo content
		hashID, err := computePhotoHash(photoPath)
		if err != nil {
			continue // Skip if we can't compute hash
		}

		// Compute aspect ratio
		aspectRatio, err := getImageAspectRatio(photoPath)
		if err != nil {
			continue // Skip invalid images
		}

		// Convert to integer ratio
		ratioW, ratioH := getIntegerRatio(aspectRatio)

		// Build thumbnail URL - thumbnails are in dist/images/{project}/.thumbs/thumb-{hashID}.webp
		// The builder server serves /images/ from dist/images/
		thumbFilename := fmt.Sprintf("thumb-%s.webp", hashID)
		thumbURL := fmt.Sprintf("/images/%s/.thumbs/%s", slug, thumbFilename)

		photo := &PhotoInfo{
			Filename:    entry.Name(),
			HashID:      hashID,
			Path:        photoPath,
			Size:        info.Size(),
			AspectRatio: aspectRatio,
			RatioWidth:  ratioW,
			RatioHeight: ratioH,
			ThumbPath:   thumbURL,
		}
		photos = append(photos, photo)
	}

	return photos, nil
}

// ListProcessedPhotos returns all processed photos for a project by reading from the processed images directory (dist/images).
// This is used by the builder UI to display thumbnails that have been processed.
func (m *Manager) ListProcessedPhotos(slug string, processedDir string) ([]*PhotoInfo, error) {
	// Read from dist/images/{slug}/.thumbs/ directory
	thumbsDir := filepath.Join(processedDir, slug, ".thumbs")

	entries, err := os.ReadDir(thumbsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []*PhotoInfo{}, nil
		}
		return nil, fmt.Errorf("failed to read processed thumbs directory: %w", err)
	}

	var photos []*PhotoInfo
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasPrefix(entry.Name(), "thumb-") {
			continue
		}

		// Extract hashID from filename: thumb-{hashID}.webp
		name := entry.Name()
		if !strings.HasSuffix(name, ".webp") {
			continue
		}

		hashID := strings.TrimPrefix(name, "thumb-")
		hashID = strings.TrimSuffix(hashID, ".webp")

		if len(hashID) != 12 {
			continue // Invalid hash ID length
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		// Try to get aspect ratio from the thumbnail itself
		thumbPath := filepath.Join(thumbsDir, entry.Name())
		aspectRatio, err := getImageAspectRatio(thumbPath)
		if err != nil {
			// Default to square if we can't read the aspect ratio
			aspectRatio = 1.0
		}

		// Convert to integer ratio
		ratioW, ratioH := getIntegerRatio(aspectRatio)

		// Build thumbnail URL - will be updated by server based on imageURLPrefix
		thumbURL := fmt.Sprintf("/images/%s/.thumbs/%s", slug, entry.Name())

		photo := &PhotoInfo{
			Filename:    entry.Name(), // Will be updated with original filename if needed
			HashID:      hashID,
			Path:        thumbPath,
			Size:        info.Size(),
			AspectRatio: aspectRatio,
			RatioWidth:  ratioW,
			RatioHeight: ratioH,
			ThumbPath:   thumbURL,
		}
		photos = append(photos, photo)
	}

	return photos, nil
}

// getImageAspectRatio returns the aspect ratio (width/height) of an image
func getImageAspectRatio(path string) (float64, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	img, _, err := image.DecodeConfig(file)
	if err != nil {
		return 0, err
	}

	if img.Height == 0 {
		return 0, fmt.Errorf("invalid image dimensions")
	}

	return float64(img.Width) / float64(img.Height), nil
}

// getIntegerRatio converts a decimal aspect ratio to the nearest common integer ratio
func getIntegerRatio(aspectRatio float64) (width, height int) {
	// Common photo ratios to check
	commonRatios := []struct {
		w, h int
	}{
		{1, 1},  // 1:1 square
		{3, 2},  // 3:2 landscape
		{2, 3},  // 2:3 portrait
		{4, 3},  // 4:3
		{3, 4},  // 3:4
		{16, 9}, // 16:9
		{9, 16}, // 9:16
		{5, 4},  // 5:4
		{4, 5},  // 4:5
		{7, 5},  // 7:5
		{5, 7},  // 5:7
	}

	// Find the closest match
	bestW, bestH := 3, 2
	bestDiff := abs(aspectRatio - float64(bestW)/float64(bestH))

	for _, ratio := range commonRatios {
		ratioValue := float64(ratio.w) / float64(ratio.h)
		diff := abs(aspectRatio - ratioValue)
		if diff < bestDiff {
			bestDiff = diff
			bestW = ratio.w
			bestH = ratio.h
		}
	}

	return bestW, bestH
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

// computePhotoHash computes the SHA256 hash (first 12 characters) of a photo file
func computePhotoHash(photoPath string) (string, error) {
	file, err := os.Open(photoPath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	fullHash := hex.EncodeToString(hash.Sum(nil))
	return fullHash[:12], nil
}

// generateThumbnail creates a compressed thumbnail for the builder UI
func generateThumbnail(sourcePath, thumbPath string, maxWidth int) error {
	// Open source image
	file, err := os.Open(sourcePath)
	if err != nil {
		return err
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		return err
	}

	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// Skip if already small enough
	if width <= maxWidth {
		return nil
	}

	// Calculate new dimensions
	newWidth := maxWidth
	newHeight := int(float64(height) * (float64(maxWidth) / float64(width)))

	// Create thumbnail
	thumb := image.NewRGBA(image.Rect(0, 0, newWidth, newHeight))
	draw.CatmullRom.Scale(thumb, thumb.Bounds(), img, bounds, draw.Over, nil)

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(thumbPath), 0755); err != nil {
		return err
	}

	// Save as JPEG with high compression
	outFile, err := os.Create(thumbPath)
	if err != nil {
		return err
	}
	defer outFile.Close()

	return jpeg.Encode(outFile, thumb, &jpeg.Options{Quality: 60})
}
