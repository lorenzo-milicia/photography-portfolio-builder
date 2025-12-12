package images

import (
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/image/draw"
)

// Processor handles image processing operations
type Processor struct {
	cacheDir string
}

// NewProcessor creates a new image processor
func NewProcessor(cacheDir string) *Processor {
	return &Processor{cacheDir: cacheDir}
}

// ProcessedImage contains information about processed image variants
type ProcessedImage struct {
	Original   string         `json:"original"`
	Thumbnails map[int]string `json:"thumbnails"` // size -> path
	Variants   map[int]string `json:"variants"`   // width -> path
	WebP       map[int]string `json:"webp"`       // width -> webp path
}

// ProcessImage processes an image to create thumbnails and responsive variants
func (p *Processor) ProcessImage(sourcePath, destDir string) (*ProcessedImage, error) {
	// Ensure destination directory exists
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Open source image
	src, err := os.Open(sourcePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open source image: %w", err)
	}
	defer src.Close()

	// Decode image
	img, format, err := image.Decode(src)
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	// Get filename without extension
	filename := filepath.Base(sourcePath)
	ext := filepath.Ext(filename)
	nameOnly := strings.TrimSuffix(filename, ext)

	result := &ProcessedImage{
		Original:   sourcePath,
		Thumbnails: make(map[int]string),
		Variants:   make(map[int]string),
		WebP:       make(map[int]string),
	}

	// Generate thumbnail (300px)
	thumbPath := filepath.Join(destDir, fmt.Sprintf("%s-thumb%s", nameOnly, ext))
	if err := p.resizeImage(img, thumbPath, 300, format); err != nil {
		return nil, fmt.Errorf("failed to create thumbnail: %w", err)
	}
	result.Thumbnails[300] = thumbPath

	// Generate responsive variants (480, 800, 1200px)
	sizes := []int{480, 800, 1200}
	for _, size := range sizes {
		variantPath := filepath.Join(destDir, fmt.Sprintf("%s-%d%s", nameOnly, size, ext))
		if err := p.resizeImage(img, variantPath, size, format); err != nil {
			return nil, fmt.Errorf("failed to create variant %d: %w", size, err)
		}
		result.Variants[size] = variantPath
	}

	return result, nil
}

// resizeImage resizes an image to the specified max width while maintaining aspect ratio
func (p *Processor) resizeImage(src image.Image, destPath string, maxWidth int, format string) error {
	bounds := src.Bounds()
	srcWidth := bounds.Dx()
	srcHeight := bounds.Dy()

	// If image is smaller than target, just copy it
	if srcWidth <= maxWidth {
		return p.saveImage(src, destPath, format)
	}

	// Calculate new dimensions maintaining aspect ratio
	ratio := float64(maxWidth) / float64(srcWidth)
	newHeight := int(float64(srcHeight) * ratio)

	// Create destination image
	dst := image.NewRGBA(image.Rect(0, 0, maxWidth, newHeight))

	// Resize using high-quality algorithm
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, bounds, draw.Over, nil)

	return p.saveImage(dst, destPath, format)
}

// saveImage saves an image to disk
func (p *Processor) saveImage(img image.Image, path string, format string) error {
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	switch format {
	case "jpeg", "jpg":
		return jpeg.Encode(file, img, &jpeg.Options{Quality: 90})
	case "png":
		return png.Encode(file, img)
	default:
		// Default to JPEG
		return jpeg.Encode(file, img, &jpeg.Options{Quality: 90})
	}
}

// GenerateThumbnail creates a single thumbnail of specified size
func (p *Processor) GenerateThumbnail(sourcePath, destPath string, size int) error {
	src, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to open source image: %w", err)
	}
	defer src.Close()

	img, format, err := image.Decode(src)
	if err != nil {
		return fmt.Errorf("failed to decode image: %w", err)
	}

	return p.resizeImage(img, destPath, size, format)
}
