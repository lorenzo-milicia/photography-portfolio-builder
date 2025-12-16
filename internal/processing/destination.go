package processing

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// Destination provides explicit methods for writing processed images and thumbnails
// to their respective locations
type Destination struct {
	// OutputDir is where processed image variants and thumbnails are stored (e.g., dist/images/project-name)
	OutputDir string
}

// CreateVariant creates a file for an image variant in OutputDir/{hashID}/{filename}
func (d *Destination) CreateVariant(hashID, filename string) (io.WriteCloser, error) {
	fullPath := filepath.Join(d.OutputDir, hashID, filename)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create variant directory: %w", err)
	}
	return os.Create(fullPath)
}

// VariantExists checks if a variant file exists
func (d *Destination) VariantExists(hashID, filename string) bool {
	path := filepath.Join(d.OutputDir, hashID, filename)
	_, err := os.Stat(path)
	return err == nil
}

// CreateThumbnail creates a file for a thumbnail in OutputDir/.thumbs/{filename}
func (d *Destination) CreateThumbnail(filename string) (io.WriteCloser, error) {
	fullPath := filepath.Join(d.OutputDir, ".thumbs", filename)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create thumbnail directory: %w", err)
	}
	return os.Create(fullPath)
}

// ThumbnailExists checks if a thumbnail file exists
func (d *Destination) ThumbnailExists(filename string) bool {
	path := filepath.Join(d.OutputDir, ".thumbs", filename)
	_, err := os.Stat(path)
	return err == nil
}
