package layouts

import (
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"os"
)

// ImageDimensions represents image dimensions
type ImageDimensions struct {
	Width  int     `json:"width"`
	Height int     `json:"height"`
	Ratio  float64 `json:"ratio"`
}

// LayoutItem represents a positioned image in the layout
type LayoutItem struct {
	Filename string `json:"filename"`
	X        int    `json:"x"`
	Y        int    `json:"y"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
	ColSpan  int    `json:"col_span,omitempty"`
	RowSpan  int    `json:"row_span,omitempty"`
}

// GetImageDimensions reads image dimensions from file
func GetImageDimensions(path string) (*ImageDimensions, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open image: %w", err)
	}
	defer file.Close()

	config, _, err := image.DecodeConfig(file)
	if err != nil {
		return nil, fmt.Errorf("failed to decode image config: %w", err)
	}

	ratio := float64(config.Width) / float64(config.Height)

	return &ImageDimensions{
		Width:  config.Width,
		Height: config.Height,
		Ratio:  ratio,
	}, nil
}

// JustifiedLayout calculates a justified layout
func JustifiedLayout(imagePaths []string, rowHeight, gap int) ([]*LayoutItem, error) {
	var items []*LayoutItem
	var currentRow []*LayoutItem
	var currentRowWidth float64
	const containerWidth = 1200 // Default container width

	y := 0

	for _, path := range imagePaths {
		dims, err := GetImageDimensions(path)
		if err != nil {
			continue // Skip images we can't read
		}

		// Calculate width for this image at target row height
		width := float64(rowHeight) * dims.Ratio

		item := &LayoutItem{
			Filename: path,
			Width:    int(width),
			Height:   rowHeight,
		}

		currentRow = append(currentRow, item)
		currentRowWidth += width + float64(gap)

		// Check if row is full
		if currentRowWidth >= float64(containerWidth) {
			// Scale row to fit exactly
			scaleFactor := float64(containerWidth) / (currentRowWidth - float64(gap))
			x := 0

			for _, rowItem := range currentRow {
				rowItem.Width = int(float64(rowItem.Width) * scaleFactor)
				rowItem.Height = int(float64(rowItem.Height) * scaleFactor)
				rowItem.X = x
				rowItem.Y = y
				x += rowItem.Width + gap
			}

			items = append(items, currentRow...)
			y += int(float64(rowHeight)*scaleFactor) + gap
			currentRow = nil
			currentRowWidth = 0
		}
	}

	// Handle remaining items in last row
	if len(currentRow) > 0 {
		x := 0
		for _, item := range currentRow {
			item.X = x
			item.Y = y
			x += item.Width + gap
		}
		items = append(items, currentRow...)
	}

	return items, nil
}

// GridLayout calculates a simple grid layout
func GridLayout(imagePaths []string, columns, gap int) ([]*LayoutItem, error) {
	var items []*LayoutItem
	const containerWidth = 1200

	// Calculate column width
	totalGap := gap * (columns - 1)
	colWidth := (containerWidth - totalGap) / columns

	for i, path := range imagePaths {
		dims, err := GetImageDimensions(path)
		if err != nil {
			continue
		}

		// Calculate height maintaining aspect ratio
		height := int(float64(colWidth) / dims.Ratio)

		row := i / columns
		col := i % columns

		item := &LayoutItem{
			Filename: path,
			X:        col * (colWidth + gap),
			Y:        row * (height + gap),
			Width:    colWidth,
			Height:   height,
			ColSpan:  1,
			RowSpan:  1,
		}

		items = append(items, item)
	}

	return items, nil
}

// ManualGridLayout uses explicit positioning
func ManualGridLayout(positions map[string]map[string]int, columns, gap int) ([]*LayoutItem, error) {
	var items []*LayoutItem
	const containerWidth = 1200

	totalGap := gap * (columns - 1)
	colWidth := (containerWidth - totalGap) / columns
	rowHeight := colWidth // Square cells by default

	for filename, pos := range positions {
		colSpan := 1
		rowSpan := 1
		if span, ok := pos["col_span"]; ok {
			colSpan = span
		}
		if span, ok := pos["row_span"]; ok {
			rowSpan = span
		}

		width := colWidth*colSpan + gap*(colSpan-1)
		height := rowHeight*rowSpan + gap*(rowSpan-1)

		item := &LayoutItem{
			Filename: filename,
			X:        pos["col"] * (colWidth + gap),
			Y:        pos["row"] * (rowHeight + gap),
			Width:    width,
			Height:   height,
			ColSpan:  colSpan,
			RowSpan:  rowSpan,
		}

		items = append(items, item)
	}

	return items, nil
}
