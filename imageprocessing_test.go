package main

import (
	"testing"
)

func createImageProcessingSettings(maxWidth, maxHeight, maxNarrowSide, jpegQuality, webpQuality int, convertToFormat string) ImageProcessingSettings {
	return ImageProcessingSettings{
		MaxWidth:        maxWidth,
		MaxHeight:       maxHeight,
		MaxNarrowSide:   maxNarrowSide,
		JpegQuality:     jpegQuality,
		WebpQuality:     webpQuality,
		ConvertToFormat: convertToFormat,
	}
}

func TestCalculateNarrowSideResize(t *testing.T) {
	tests := []struct {
		name           string
		original       ImageSize
		maxNarrowSide  int
		expected       ImageSize
	}{
		{
			name:          "No resize needed - within limit",
			original:      ImageSize{Width: 800, Height: 600},
			maxNarrowSide: 600,
			expected:      ImageSize{Width: 800, Height: 600},
		},
		{
			name:          "Resize landscape image - height is narrow side",
			original:      ImageSize{Width: 1600, Height: 800},
			maxNarrowSide: 400,
			expected:      ImageSize{Width: 800, Height: 400},
		},
		{
			name:          "Resize portrait image - width is narrow side",
			original:      ImageSize{Width: 600, Height: 1200},
			maxNarrowSide: 300,
			expected:      ImageSize{Width: 300, Height: 600},
		},
		{
			name:          "Square image resize",
			original:      ImageSize{Width: 1000, Height: 1000},
			maxNarrowSide: 500,
			expected:      ImageSize{Width: 500, Height: 500},
		},
		{
			name:          "Very narrow image",
			original:      ImageSize{Width: 2000, Height: 100},
			maxNarrowSide: 50,
			expected:      ImageSize{Width: 1000, Height: 50},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateNarrowSideResize(tt.original, tt.maxNarrowSide)
			if result.Width != tt.expected.Width || result.Height != tt.expected.Height {
				t.Errorf("calculateNarrowSideResize() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestCalculateBoundingBoxResize(t *testing.T) {
	tests := []struct {
		name      string
		original  ImageSize
		maxWidth  int
		maxHeight int
		expected  ImageSize
	}{
		{
			name:      "No resize needed - landscape within bounds",
			original:  ImageSize{Width: 500, Height: 300},
			maxWidth:  800,
			maxHeight: 600,
			expected:  ImageSize{Width: 500, Height: 300},
		},
		{
			name:      "No resize needed - portrait within bounds",
			original:  ImageSize{Width: 300, Height: 500},
			maxWidth:  800,
			maxHeight: 600,
			expected:  ImageSize{Width: 300, Height: 500},
		},
		{
			name:      "Landscape: resize by width constraint",
			original:  ImageSize{Width: 1600, Height: 800},
			maxWidth:  800,
			maxHeight: 1000,
			// Config 800x1000 interpreted as: short edge ≤ 800, long edge ≤ 1000
			// Landscape image → needs long×short boundary box = 1000×800
			// Scale: min(1000/1600, 800/800) = min(0.625, 1.0) = 0.625
			// Result: 1600×0.625 = 1000, 800×0.625 = 500
			expected:  ImageSize{Width: 1000, Height: 500},
		},
		{
			name:      "Portrait: orientation-aware resize (swapped limits)",
			original:  ImageSize{Width: 800, Height: 1200},
			maxWidth:  1000,
			maxHeight: 600,
			expected:  ImageSize{Width: 600, Height: 900},
		},
		{
			name:      "Landscape: both constraints - width tighter",
			original:  ImageSize{Width: 2000, Height: 1000},
			maxWidth:  400,
			maxHeight: 800,
			// Config 400x800 interpreted as: short edge ≤ 400, long edge ≤ 800
			// Landscape image → needs long×short boundary box = 800×400
			// Scale: min(800/2000, 400/1000) = min(0.4, 0.4) = 0.4
			// Result: 2000×0.4 = 800, 1000×0.4 = 400
			expected:  ImageSize{Width: 800, Height: 400},
		},
		{
			name:      "Portrait: orientation-aware with swapped constraints",
			original:  ImageSize{Width: 1000, Height: 2000},
			maxWidth:  800,
			maxHeight: 400,
			expected:  ImageSize{Width: 400, Height: 800},
		},
		{
			name:      "Real example: 3000x2000 landscape with 1920x1080 config",
			original:  ImageSize{Width: 3000, Height: 2000},
			maxWidth:  1920,
			maxHeight: 1080,
			expected:  ImageSize{Width: 1620, Height: 1080},
		},
		{
			name:      "Real example: 2000x3000 portrait with 1920x1080 config",
			original:  ImageSize{Width: 2000, Height: 3000},
			maxWidth:  1920,
			maxHeight: 1080,
			expected:  ImageSize{Width: 1080, Height: 1620},
		},
		{
			name:      "Portrait config: 2000x3000 portrait with 1080x1920 config",
			original:  ImageSize{Width: 2000, Height: 3000},
			maxWidth:  1080,
			maxHeight: 1920,
			expected:  ImageSize{Width: 1080, Height: 1620},
		},
		{
			name:      "Portrait config: 3000x2000 landscape with 1080x1920 config",
			original:  ImageSize{Width: 3000, Height: 2000},
			maxWidth:  1080,
			maxHeight: 1920,
			expected:  ImageSize{Width: 1620, Height: 1080},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateBoundingBoxResize(tt.original, tt.maxWidth, tt.maxHeight)
			if result.Width != tt.expected.Width || result.Height != tt.expected.Height {
				t.Errorf("calculateBoundingBoxResize() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestCalculateResizeDimensions(t *testing.T) {
	tests := []struct {
		name     string
		original ImageSize
		settings ImageProcessingSettings
		expected ImageSize
	}{
		{
			name:     "Use narrow side strategy",
			original: ImageSize{Width: 1200, Height: 800},
			settings: createImageProcessingSettings(1000, 1000, 400, 80, 85, ""),
			expected: ImageSize{Width: 600, Height: 400},
		},
		{
			name:     "Use bounding box strategy - no narrow side set",
			original: ImageSize{Width: 1200, Height: 800},
			settings: createImageProcessingSettings(600, 1000, 0, 80, 85, ""), // MaxNarrowSide=0 means not set
			// Config 600x1000 interpreted as: short edge ≤ 600, long edge ≤ 1000
			// Landscape image 1200x800 → needs long×short boundary box = 1000×600
			// Scale: min(1000/1200, 600/800) = min(0.833, 0.75) = 0.75
			// Result: 1200×0.75 = 900, 800×0.75 = 600
			expected: ImageSize{Width: 900, Height: 600},
		},
		{
			name:     "Narrow side strategy takes precedence",
			original: ImageSize{Width: 1000, Height: 2000},
			settings: createImageProcessingSettings(500, 500, 800, 80, 85, ""), // Narrow side looser but takes precedence
			expected: ImageSize{Width: 800, Height: 1600},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateResizeDimensions(tt.original, tt.settings)
			if result.Width != tt.expected.Width || result.Height != tt.expected.Height {
				t.Errorf("calculateResizeDimensions() = %v, want %v", result, tt.expected)
			}
		})
	}
}
