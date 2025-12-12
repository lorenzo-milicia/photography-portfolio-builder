package content

import (
	"fmt"
	"image"
	"image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"strings"

	"go.lorenzomilicia.dev/photography-portfolio-builder/internal/util"
	"golang.org/x/image/draw"
)

// PhotoInfo holds information about a photo
type PhotoInfo struct {
	Filename    string  `json:"filename"`
	Path        string  `json:"path"`
	Size        int64   `json:"size"`
	Selected    bool    `json:"selected"`
	AspectRatio float64 `json:"aspectRatio"` // width / height (deprecated, use RatioWidth/RatioHeight)
	RatioWidth  int     `json:"ratioWidth"`  // integer width of aspect ratio (e.g., 3 for 3:2)
	RatioHeight int     `json:"ratioHeight"` // integer height of aspect ratio (e.g., 2 for 3:2)
	ThumbPath   string  `json:"thumbPath"`   // path to thumbnail for builder UI
}

// PhotoSelection stores which photos are selected for a project
type PhotoSelection struct {
	Selected []string `yaml:"selected"`
}

// ProjectPhotosSelectionPath returns the photos selection file path for a project
func (m *Manager) ProjectPhotosSelectionPath(slug string) string {
	return filepath.Join(m.ProjectDir(slug), "photos.yaml")
}

// GetPhotoSelection retrieves the selected photos for a project
func (m *Manager) GetPhotoSelection(slug string) (*PhotoSelection, error) {
	selectionPath := m.ProjectPhotosSelectionPath(slug)

	// If file doesn't exist, return empty selection
	if _, err := os.Stat(selectionPath); os.IsNotExist(err) {
		return &PhotoSelection{Selected: []string{}}, nil
	}

	var selection PhotoSelection
	if err := util.LoadYAML(selectionPath, &selection); err != nil {
		return nil, fmt.Errorf("failed to load photo selection: %w", err)
	}

	return &selection, nil
}

// SavePhotoSelection saves the selected photos for a project
func (m *Manager) SavePhotoSelection(slug string, selection *PhotoSelection) error {
	selectionPath := m.ProjectPhotosSelectionPath(slug)
	return util.SaveYAML(selectionPath, selection)
}

// ListPhotos returns all photos for a project with selection state
func (m *Manager) ListPhotos(slug string) ([]*PhotoInfo, error) {
	photosDir := m.ProjectPhotosDir(slug)

	// Get current selection
	selection, err := m.GetPhotoSelection(slug)
	if err != nil {
		return nil, fmt.Errorf("failed to get photo selection: %w", err)
	}

	// Create a map for quick lookup
	selectedMap := make(map[string]bool)
	for _, filename := range selection.Selected {
		selectedMap[filename] = true
	}

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
			Selected:    selectedMap[entry.Name()],
			AspectRatio: aspectRatio,
			RatioWidth:  ratioW,
			RatioHeight: ratioH,
			ThumbPath:   thumbURL,
		}
		photos = append(photos, photo)
	}

	return photos, nil
}

// ListSelectedPhotos returns only the selected photos for a project
func (m *Manager) ListSelectedPhotos(slug string) ([]*PhotoInfo, error) {
	allPhotos, err := m.ListPhotos(slug)
	if err != nil {
		return nil, err
	}

	var selectedPhotos []*PhotoInfo
	for _, photo := range allPhotos {
		if photo.Selected {
			selectedPhotos = append(selectedPhotos, photo)
		}
	}

	return selectedPhotos, nil
}

// TogglePhotoSelection toggles the selection state of a photo
func (m *Manager) TogglePhotoSelection(slug, filename string) error {
	selection, err := m.GetPhotoSelection(slug)
	if err != nil {
		return err
	}

	// Check if photo exists in selection
	index := -1
	for i, selected := range selection.Selected {
		if selected == filename {
			index = i
			break
		}
	}

	if index >= 0 {
		// Remove from selection
		selection.Selected = append(selection.Selected[:index], selection.Selected[index+1:]...)
	} else {
		// Add to selection
		selection.Selected = append(selection.Selected, filename)
	}

	return m.SavePhotoSelection(slug, selection)
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

// SetPhotoSelection sets the complete selection for a project
func (m *Manager) SetPhotoSelection(slug string, filenames []string) error {
	selection := &PhotoSelection{Selected: filenames}
	return m.SavePhotoSelection(slug, selection)
}
