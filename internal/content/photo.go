package content

import (
	"fmt"
	"image"
	"image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/image/draw"
)

// PhotoInfo holds information about a photo
type PhotoInfo struct {
	Filename    string  `json:"filename"`
	Path        string  `json:"path"`
	Size        int64   `json:"size"`
	AspectRatio float64 `json:"aspectRatio"` // width / height (deprecated, use RatioWidth/RatioHeight)
	RatioWidth  int     `json:"ratioWidth"`  // integer width of aspect ratio (e.g., 3 for 3:2)
	RatioHeight int     `json:"ratioHeight"` // integer height of aspect ratio (e.g., 2 for 3:2)
	ThumbPath   string  `json:"thumbPath"`   // path to thumbnail for builder UI
}

// ListPhotos returns all photos for a project
func (m *Manager) ListPhotos(slug string) ([]*PhotoInfo, error) {
	photosDir := m.ProjectPhotosDir(slug)

	entries, err := os.ReadDir(photosDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []*PhotoInfo{}, nil
		}
		return nil, fmt.Errorf("failed to read photos directory: %w", err)
	}

	// Ensure thumbs directory exists
	thumbsDir := filepath.Join(photosDir, ".thumbs")
	os.MkdirAll(thumbsDir, 0755)

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
		thumbPath := filepath.Join(thumbsDir, entry.Name())

		// Compute aspect ratio
		aspectRatio, err := getImageAspectRatio(photoPath)
		if err != nil {
			continue // Skip invalid images
		}

		// Convert to integer ratio
		ratioW, ratioH := getIntegerRatio(aspectRatio)

		// Generate thumbnail if it doesn't exist
		if _, err := os.Stat(thumbPath); os.IsNotExist(err) {
			if err := generateThumbnail(photoPath, thumbPath, 400); err != nil {
				// If thumbnail generation fails, use original
				thumbPath = photoPath
			}
		}

		// Convert thumb path to web URL
		thumbURL := fmt.Sprintf("/content/photos/%s/.thumbs/%s", slug, entry.Name())
		// If thumbnail generation failed, use original photo URL
		if thumbPath == photoPath {
			thumbURL = fmt.Sprintf("/content/photos/%s/%s", slug, entry.Name())
		}

		photo := &PhotoInfo{
			Filename:    entry.Name(),
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
