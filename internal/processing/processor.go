package processing

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"image"
	"io"

	"github.com/chai2010/webp"
	"github.com/disintegration/imaging"
)

// ImageSource abstracts the reading of an image
type ImageSource interface {
	Open() (io.ReadCloser, error)
	Name() string
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
func (p *Processor) ProcessImage(src ImageSource, dst *Destination) error {
	// 1. Compute Hash
	hash, err := p.ComputeHash(src)
	if err != nil {
		return err
	}

	hashID := hash[:12]

	// 2. Check if processing is needed (skip if all files exist and not forcing)
	if !p.Config.Force {
		allExist := true

		// Check all variant files
		for _, width := range p.Config.Widths {
			filename := fmt.Sprintf("%s-%dw.webp", hashID, width)
			if !dst.VariantExists(hashID, filename) {
				allExist = false
				break
			}
		}

		// Check thumbnail if enabled
		if allExist && p.Config.GenerateThumbnails {
			thumbFilename := fmt.Sprintf("thumb-%s.webp", hashID)
			if !dst.ThumbnailExists(thumbFilename) {
				allExist = false
			}
		}

		if allExist {
			// All files exist, skip processing
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

	// 4. Generate and save all image variants
	for _, width := range p.Config.Widths {
		// Calculate height maintaining aspect ratio
		bounds := img.Bounds()
		ratio := float64(bounds.Dy()) / float64(bounds.Dx())
		height := int(float64(width) * ratio)

		// Resize image
		resized := p.resizeImage(img, width, height)

		// Save variant: dist/images/project/{hashID}/{hashID}-{width}w.webp
		filename := fmt.Sprintf("%s-%dw.webp", hashID, width)
		if err := p.saveVariant(resized, dst, hashID, filename); err != nil {
			return fmt.Errorf("failed to save variant %s: %w", filename, err)
		}
	}

	// 5. Generate and save thumbnail
	if p.Config.GenerateThumbnails {
		width := p.Config.ThumbnailWidth
		bounds := img.Bounds()
		ratio := float64(bounds.Dy()) / float64(bounds.Dx())
		height := int(float64(width) * ratio)

		// Resize for thumbnail
		resized := p.resizeImage(img, width, height)

		// Save thumbnail: photos/project/.thumbs/thumb-{hashID}.webp
		filename := fmt.Sprintf("thumb-%s.webp", hashID)
		if err := p.saveThumbnail(resized, dst, filename); err != nil {
			return fmt.Errorf("failed to save thumbnail %s: %w", filename, err)
		}
	}

	return nil
}

// resizeImage resizes the image to the specified dimensions using Lanczos filter
func (p *Processor) resizeImage(img image.Image, width, height int) image.Image {
	return imaging.Resize(img, width, height, imaging.Lanczos)
}

// saveVariant saves an image variant to the output directory
func (p *Processor) saveVariant(img image.Image, dst *Destination, hashID, filename string) error {
	writer, err := dst.CreateVariant(hashID, filename)
	if err != nil {
		return fmt.Errorf("failed to create variant file: %w", err)
	}
	defer writer.Close()

	if err := webp.Encode(writer, img, &webp.Options{Quality: float32(p.Config.Quality)}); err != nil {
		return fmt.Errorf("failed to encode WebP: %w", err)
	}

	return nil
}

// saveThumbnail saves a thumbnail to the source directory
func (p *Processor) saveThumbnail(img image.Image, dst *Destination, filename string) error {
	writer, err := dst.CreateThumbnail(filename)
	if err != nil {
		return fmt.Errorf("failed to create thumbnail file: %w", err)
	}
	defer writer.Close()

	if err := webp.Encode(writer, img, &webp.Options{Quality: float32(p.Config.Quality)}); err != nil {
		return fmt.Errorf("failed to encode WebP: %w", err)
	}

	return nil
}
