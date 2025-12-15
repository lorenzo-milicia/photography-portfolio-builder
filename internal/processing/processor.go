package processing

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"image"
	"io"
	"path/filepath"

	"github.com/chai2010/webp"
	"github.com/disintegration/imaging"
)

// ImageSource abstracts the reading of an image
type ImageSource interface {
	Open() (io.ReadCloser, error)
	Name() string
}

// ImageDestination abstracts the writing of an image
type ImageDestination interface {
	Create(filename string) (io.WriteCloser, error)
	Exists(filename string) bool
}

// ProcessConfig holds configuration for the processor
type ProcessConfig struct {
	Widths             []int
	Quality            int
	Force              bool // Overwrite existing files
	GenerateThumbnails bool
	ThumbnailWidth     int
}

// Processor handles the image processing pipeline
type Processor struct {
	Config ProcessConfig
}

// NewProcessor creates a new processor
func NewProcessor(config ProcessConfig) *Processor {
	if len(config.Widths) == 0 {
		config.Widths = []int{480, 800, 1200, 1920}
	}
	if config.Quality == 0 {
		config.Quality = 80
	}
	if config.GenerateThumbnails && config.ThumbnailWidth == 0 {
		config.ThumbnailWidth = 300 // Default thumbnail width
	}
	return &Processor{Config: config}
}

// ComputeHash computes a SHA256 hash of the image source content
func (p *Processor) ComputeHash(src ImageSource) (string, error) {
	reader, err := src.Open()
	if err != nil {
		return "", fmt.Errorf("failed to open source for hashing: %w", err)
	}
	defer reader.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, reader); err != nil {
		return "", fmt.Errorf("failed to compute hash: %w", err)
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

// ProcessImage processes a single image: hash -> resize -> convert -> save
func (p *Processor) ProcessImage(src ImageSource, dst ImageDestination) error {
	// 1. Compute Hash
	hash, err := p.ComputeHash(src)
	if err != nil {
		return err
	}

	// 2. Check if processing is needed
	photoDir := hash[:12]
	if !p.Config.Force {
		allExist := true
		// Check variants (now inside photoDir)
		for _, width := range p.Config.Widths {
			filename := filepath.Join(photoDir, fmt.Sprintf("%s-%dw.webp", hash[:12], width))
			if !dst.Exists(filename) {
				allExist = false
				break
			}
		}
		// Check thumbnail if enabled (top-level, routed to .thumbs by destination)
		if allExist && p.Config.GenerateThumbnails {
			thumbFilename := fmt.Sprintf("thumb-%s.webp", hash[:12])
			if !dst.Exists(thumbFilename) {
				allExist = false
			}
		}

		if allExist {
			// Skip processing
			return nil
		}
	}

	// 3. Decode Image
	reader, err := src.Open()
	if err != nil {
		return fmt.Errorf("failed to open source for decoding: %w", err)
	}
	defer reader.Close()

	img, _, err := image.Decode(reader)
	if err != nil {
		return fmt.Errorf("failed to decode image: %w", err)
	}

	// 4. Resize and Save Variants — store variants inside a per-photo directory
	for _, width := range p.Config.Widths {
		// Calculate height maintaining aspect ratio
		bounds := img.Bounds()
		ratio := float64(bounds.Dy()) / float64(bounds.Dx())
		height := int(float64(width) * ratio)

		// Resize
		resized := p.resizeImage(img, width, height)

		// Save as WebP inside the photoDir
		filename := filepath.Join(photoDir, fmt.Sprintf("%s-%dw.webp", hash[:12], width))
		if err := p.saveAsWebP(resized, dst, filename); err != nil {
			return fmt.Errorf("failed to save variant %s: %w", filename, err)
		}
	}

	// 5. Generate Thumbnail
	if p.Config.GenerateThumbnails {
		width := p.Config.ThumbnailWidth
		bounds := img.Bounds()
		ratio := float64(bounds.Dy()) / float64(bounds.Dx())
		height := int(float64(width) * ratio)

		resized := p.resizeImage(img, width, height)
		filename := fmt.Sprintf("thumb-%s.webp", hash[:12])
		if err := p.saveAsWebP(resized, dst, filename); err != nil {
			return fmt.Errorf("failed to save thumbnail %s: %w", filename, err)
		}
	}

	return nil
}

// resizeImage resizes the image to the specified dimensions using Lanczos filter
func (p *Processor) resizeImage(img image.Image, width, height int) image.Image {
	return imaging.Resize(img, width, height, imaging.Lanczos)
}

// saveAsWebP saves the image as a WebP file to the destination
func (p *Processor) saveAsWebP(img image.Image, dst ImageDestination, filename string) error {
	writer, err := dst.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer writer.Close()

	if err := webp.Encode(writer, img, &webp.Options{Quality: float32(p.Config.Quality)}); err != nil {
		return fmt.Errorf("failed to encode WebP: %w", err)
	}

	return nil
}
