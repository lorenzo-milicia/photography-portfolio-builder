package images

import (
	"fmt"
	"image"
	"image/jpeg"
	"os"
	"path/filepath"

	"github.com/disintegration/imaging"
	"go.lorenzomilicia.dev/photography-portfolio-builder/internal/content"
)

// Processor handles image processing operations
type Processor struct {
	cacheDir string
}

// NewProcessor creates a new image processor
func NewProcessor(cacheDir string) *Processor {
	return &Processor{cacheDir: cacheDir}
}

// OptimizationConfig holds optimization parameters for an image
type OptimizationConfig struct {
	MaxWidth      int
	MaxHeight     int
	Quality       int
	StripMetadata bool
}

// ImageVariants holds information about responsive image variants
type ImageVariants struct {
	BaseFilename string
	Variants     []ImageVariant
}

// ImageVariant represents a single responsive image variant
type ImageVariant struct {
	Filename string
	Width    int
}

// ProcessProjectImages processes all images for a project with optimization
// It strips metadata, renames files sequentially, and generates responsive variants
func (p *Processor) ProcessProjectImages(sourceDir, destDir string, layout *content.LayoutConfig) (map[string]*ImageVariants, error) {
	// Map from original filename to responsive variants
	variantsMap := make(map[string]*ImageVariants)

	if err := os.MkdirAll(destDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create destination directory: %w", err)
	}

	// First pass: Calculate optimization config for all images (Desktop + Mobile)
	// Map from original filename to aggregated optimization config
	fileConfigs := make(map[string]OptimizationConfig)

	// Helper to merge configs
	mergeConfig := func(filename string, placement content.PhotoPlacement, isMobile bool) {
		currentConfig, exists := fileConfigs[filename]
		newConfig := p.calculateOptimalSize(placement) // Note: currently this assumes standard grid logic

		// If mobile, generic logic might underestimate.
		// For now, we trust calculateOptimalSize, but ensure we take the MAX dimensions if reused.
		if !exists {
			fileConfigs[filename] = newConfig
		} else {
			// Take maximum dimensions to satisfy both layouts
			if newConfig.MaxWidth > currentConfig.MaxWidth {
				currentConfig.MaxWidth = newConfig.MaxWidth
			}
			if newConfig.MaxHeight > currentConfig.MaxHeight {
				currentConfig.MaxHeight = newConfig.MaxHeight
			}
			fileConfigs[filename] = currentConfig
		}
	}

	// Collect Desktop requirements
	for _, placement := range layout.Placements {
		mergeConfig(placement.Filename, placement, false)
	}
	// Collect Mobile requirements
	for _, placement := range layout.MobilePlacements {
		mergeConfig(placement.Filename, placement, true) // isMobile flag could be used for diff logic later
	}

	// Deterministic iteration order (not strictly required for correctness but good for testing)
	// Actually, the original code used index-based filenames ("1", "2").
	// To support disjoint sets, we need a stable mapping strategy.
	// Let's use a stable index based on sorted presence? Or just hash?
	// The previous logic was: `baseFilename := fmt.Sprintf("%d", idx+1)`
	// If we change this, we change existing file URLs.
	// To maintain backward compatibility somewhat:
	// We can assign IDs based on order of appearance in Desktop, then append new ones from Mobile?

	// Collect unique filenames in order
	var uniqueFilenames []string
	seen := make(map[string]bool)

	// Desktop first (preserve existing order mainly)
	for _, p := range layout.Placements {
		if !seen[p.Filename] {
			uniqueFilenames = append(uniqueFilenames, p.Filename)
			seen[p.Filename] = true
		}
	}
	// Mobile next
	for _, p := range layout.MobilePlacements {
		if !seen[p.Filename] {
			uniqueFilenames = append(uniqueFilenames, p.Filename)
			seen[p.Filename] = true
		}
	}

	// Process all files
	for idx, filename := range uniqueFilenames {
		sourcePath := filepath.Join(sourceDir, filename)
		baseFilename := fmt.Sprintf("%d", idx+1)
		config := fileConfigs[filename]

		// Generate responsive variants for this image
		variants, err := p.generateResponsiveVariants(sourcePath, destDir, baseFilename, config)
		if err != nil {
			return nil, fmt.Errorf("failed to generate variants for %s: %w", filename, err)
		}

		variantsMap[filename] = variants
	}

	return variantsMap, nil
}

// generateResponsiveVariants creates multiple size variants for responsive images
func (p *Processor) generateResponsiveVariants(sourcePath, destDir, baseFilename string, config OptimizationConfig) (*ImageVariants, error) {
	// Open source image once
	img, err := imaging.Open(sourcePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open image: %w", err)
	}

	bounds := img.Bounds()
	origWidth := bounds.Dx()
	origHeight := bounds.Dy()
	aspectRatio := float64(origWidth) / float64(origHeight)

	// Calculate target widths for responsive variants
	// Based on the max width, create variants at 0.5x, 0.75x, and 1x
	maxWidth := config.MaxWidth
	targetWidths := []int{
		maxWidth / 2,     // 0.5x for mobile
		maxWidth * 3 / 4, // 0.75x for tablets
		maxWidth,         // 1x for desktop
	}

	// Remove duplicates and ensure we don't exceed original image size
	uniqueWidths := make(map[int]bool)
	var finalWidths []int
	for _, w := range targetWidths {
		if w > origWidth {
			w = origWidth
		}
		if !uniqueWidths[w] && w >= 320 { // Minimum practical width
			uniqueWidths[w] = true
			finalWidths = append(finalWidths, w)
		}
	}

	// Ensure at least one variant exists (for very small grid cells)
	if len(finalWidths) == 0 {
		// Use the original width or maxWidth, whichever is smaller
		fallbackWidth := maxWidth
		if origWidth < fallbackWidth {
			fallbackWidth = origWidth
		}
		if fallbackWidth < 320 {
			fallbackWidth = 320 // Absolute minimum
		}
		finalWidths = append(finalWidths, fallbackWidth)
	}

	variants := &ImageVariants{
		BaseFilename: baseFilename,
		Variants:     make([]ImageVariant, 0, len(finalWidths)),
	}

	// Generate each variant
	for _, targetWidth := range finalWidths {
		targetHeight := int(float64(targetWidth) / aspectRatio)

		// Determine quality based on size
		quality := p.calculateQuality(targetWidth * targetHeight)

		// Generate filename with width suffix: 1-480w.jpg, 1-800w.jpg, etc.
		filename := fmt.Sprintf("%s-%dw.jpg", baseFilename, targetWidth)
		destPath := filepath.Join(destDir, filename)

		// Resize and save
		resized := imaging.Resize(img, targetWidth, targetHeight, imaging.Lanczos)
		if err := p.saveOptimizedJPEG(resized, destPath, quality); err != nil {
			return nil, fmt.Errorf("failed to save variant %s: %w", filename, err)
		}

		variants.Variants = append(variants.Variants, ImageVariant{
			Filename: filename,
			Width:    targetWidth,
		})
	}

	return variants, nil
}

// calculateQuality determines JPEG quality based on total pixels
func (p *Processor) calculateQuality(totalPixels int) int {
	if totalPixels > 4000000 { // > 4MP
		return 85
	} else if totalPixels > 2000000 { // > 2MP
		return 87
	}
	return 90
}

// calculateOptimalSize determines optimal image dimensions based on grid placement
func (p *Processor) calculateOptimalSize(placement content.PhotoPlacement) OptimizationConfig {
	// Calculate grid cell dimensions
	colSpan := placement.Position.BottomRightX - placement.Position.TopLeftX + 1
	rowSpan := placement.Position.BottomRightY - placement.Position.TopLeftY + 1

	// Assume grid width is 12 columns across ~1400px max container
	// Each column is ~116px, so calculate pixel width
	const maxContainerWidth = 1400
	const gridColumns = 12
	const baseRowHeight = 100 // Base row height in pixels

	pixelWidth := (colSpan * maxContainerWidth) / gridColumns
	pixelHeight := rowSpan * baseRowHeight

	// Determine target size and quality based on display dimensions
	// Use 2x for retina/high DPI displays, but cap at reasonable limits
	maxWidth := pixelWidth * 2
	maxHeight := pixelHeight * 2

	// Cap at reasonable maximum (4K width)
	if maxWidth > 3840 {
		maxWidth = 3840
	}
	if maxHeight > 2160 {
		maxHeight = 2160
	}

	return OptimizationConfig{
		MaxWidth:      maxWidth,
		MaxHeight:     maxHeight,
		Quality:       90, // Default, will be adjusted per variant
		StripMetadata: true,
	}
}

// optimizeImage processes and optimizes a single image
func (p *Processor) optimizeImage(sourcePath, destPath string, config OptimizationConfig) error {
	// Open and decode the source image
	// The imaging library automatically strips EXIF metadata when encoding
	img, err := imaging.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to open image: %w", err)
	}

	// Get original dimensions
	bounds := img.Bounds()
	origWidth := bounds.Dx()
	origHeight := bounds.Dy()

	// Calculate aspect ratio
	aspectRatio := float64(origWidth) / float64(origHeight)

	// Determine final dimensions while maintaining aspect ratio
	finalWidth := config.MaxWidth
	finalHeight := config.MaxHeight

	// Fit within both max width and max height constraints
	if float64(finalWidth)/float64(finalHeight) > aspectRatio {
		// Height is the limiting factor
		finalWidth = int(float64(finalHeight) * aspectRatio)
	} else {
		// Width is the limiting factor
		finalHeight = int(float64(finalWidth) / aspectRatio)
	}

	// Only resize if image is larger than target
	var processedImg image.Image
	if origWidth > finalWidth || origHeight > finalHeight {
		// Use Lanczos resampling for high-quality downscaling
		processedImg = imaging.Resize(img, finalWidth, finalHeight, imaging.Lanczos)
	} else {
		// Image is already smaller than target, use as-is
		processedImg = img
	}

	// Save as JPEG with specified quality
	// This automatically strips all EXIF metadata
	return p.saveOptimizedJPEG(processedImg, destPath, config.Quality)
}

// saveOptimizedJPEG saves an image as optimized JPEG
func (p *Processor) saveOptimizedJPEG(img image.Image, path string, quality int) error {
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	// Encode as JPEG with specified quality
	// The jpeg.Encode does not preserve EXIF metadata, effectively stripping it
	options := &jpeg.Options{Quality: quality}
	if err := jpeg.Encode(file, img, options); err != nil {
		return fmt.Errorf("failed to encode JPEG: %w", err)
	}

	return nil
}

// Legacy functions below for backward compatibility

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
	img, err := imaging.Open(sourcePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open source image: %w", err)
	}

	// Get filename without extension
	filename := filepath.Base(sourcePath)
	ext := filepath.Ext(filename)
	nameOnly := filename[:len(filename)-len(ext)]

	result := &ProcessedImage{
		Original:   sourcePath,
		Thumbnails: make(map[int]string),
		Variants:   make(map[int]string),
		WebP:       make(map[int]string),
	}

	// Generate thumbnail (300px)
	thumbPath := filepath.Join(destDir, fmt.Sprintf("%s-thumb%s", nameOnly, ext))
	thumb := imaging.Resize(img, 300, 0, imaging.Lanczos)
	if err := imaging.Save(thumb, thumbPath); err != nil {
		return nil, fmt.Errorf("failed to create thumbnail: %w", err)
	}
	result.Thumbnails[300] = thumbPath

	// Generate responsive variants (480, 800, 1200px)
	sizes := []int{480, 800, 1200}
	for _, size := range sizes {
		variantPath := filepath.Join(destDir, fmt.Sprintf("%s-%d%s", nameOnly, size, ext))
		variant := imaging.Resize(img, size, 0, imaging.Lanczos)
		if err := imaging.Save(variant, variantPath); err != nil {
			return nil, fmt.Errorf("failed to create variant %d: %w", size, err)
		}
		result.Variants[size] = variantPath
	}

	return result, nil
}

// GenerateThumbnail creates a single thumbnail of specified size
func (p *Processor) GenerateThumbnail(sourcePath, destPath string, size int) error {
	img, err := imaging.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to open source image: %w", err)
	}

	thumb := imaging.Resize(img, size, 0, imaging.Lanczos)
	return imaging.Save(thumb, destPath)
}
