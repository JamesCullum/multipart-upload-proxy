package main

import (
	"log"

	"github.com/h2non/bimg"
)

type ImageSize struct {
	Width  int
	Height int
}

type ImageProcessingSettings struct {
	MaxWidth        int
	MaxHeight       int
	MaxNarrowSide   int
	JpegQuality     int
	WebpQuality     int
	ConvertToFormat string
}

type ImageProcessingResult struct {
	ProcessedData   []byte
	WasCompressed   bool
	WasResized      bool
	NewDimensions   ImageSize
	ProcessingError error
}

func detectImageTransparency(imageData []byte) (bool, error) {
	image := bimg.NewImage(imageData)
	metadata, err := image.Metadata()
	if err != nil {
		return false, err
	}
	return metadata.Alpha, nil
}

func processImageWithStrategy(originalData []byte, settings ImageProcessingSettings) (*ImageProcessingResult, error) {
	convertFormat := settings.ConvertToFormat
	if convertFormat != "" {
		hasTransparency, err := detectImageTransparency(originalData)
		if err == nil && hasTransparency {
			log.Printf("Skipping %s conversion - image has transparency", convertFormat)
			return &ImageProcessingResult{
				ProcessedData:   originalData,
				WasCompressed:   false,
				WasResized:      false,
				NewDimensions:   ImageSize{},
				ProcessingError: nil,
			}, nil
		}
	}

	workingImage, rotatedData, err := handleEXIFOrientation(originalData)
	if err != nil {
		return &ImageProcessingResult{
			ProcessedData:   originalData,
			WasCompressed:   false,
			WasResized:      false,
			NewDimensions:   ImageSize{},
			ProcessingError: err,
		}, err
	}

	// Get size from properly oriented image
	oldImageSize, err := workingImage.Size()
	if err != nil {
		return &ImageProcessingResult{
			ProcessedData:   rotatedData,
			WasCompressed:   false,
			WasResized:      false,
			NewDimensions:   ImageSize{},
			ProcessingError: err,
		}, err
	}

	// Calculate resize dimensions
	newDimensions := calculateResizeDimensions(
		ImageSize{Width: oldImageSize.Width, Height: oldImageSize.Height},
		settings,
	)

	// Check if format conversion is enabled (reuse convertFormat from earlier)
	if convertFormat == "" {
		// No format conversion - just resize if needed (backwards compatible behavior)
		needsResize := newDimensions.Width != oldImageSize.Width || newDimensions.Height != oldImageSize.Height
		if !needsResize {
			// No processing needed
			return &ImageProcessingResult{
				ProcessedData:   rotatedData,
				WasCompressed:   false,
				WasResized:      false,
				NewDimensions:   ImageSize{Width: oldImageSize.Width, Height: oldImageSize.Height},
				ProcessingError: nil,
			}, nil
		}

		// Resize only (preserve original format)
		options := bimg.Options{
			Width:  newDimensions.Width,
			Height: newDimensions.Height,
		}

		processedData, err := workingImage.Process(options)
		if err != nil {
			return &ImageProcessingResult{
				ProcessedData:   rotatedData,
				WasCompressed:   false,
				WasResized:      false,
				NewDimensions:   ImageSize{Width: oldImageSize.Width, Height: oldImageSize.Height},
				ProcessingError: err,
			}, err
		}

		return &ImageProcessingResult{
			ProcessedData: processedData,
			WasCompressed: false, // We didn't change format, just resized
			WasResized:    true,  // We did resize the image
			NewDimensions: newDimensions,
		}, nil
	}

	// Format conversion is enabled - process the image with format conversion
	var targetType bimg.ImageType
	var quality int

	switch convertFormat {
	case "JPEG":
		targetType = bimg.JPEG
		quality = settings.JpegQuality
	case "WEBP":
		targetType = bimg.WEBP
		quality = settings.WebpQuality
	default:
		// Shouldn't happen due to validation, but fallback to JPEG
		targetType = bimg.JPEG
		quality = settings.JpegQuality
	}

	options := bimg.Options{
		Width:   newDimensions.Width,
		Height:  newDimensions.Height,
		Quality: quality,
		Type:    targetType,
	}

	// Note: WebP transparency fallback is now handled in early detection phase

	processedData, err := workingImage.Process(options)
	if err != nil {
		return &ImageProcessingResult{
			ProcessedData:   rotatedData,  // Return rotated data even if processing fails
			WasCompressed:   false,
			NewDimensions:   ImageSize{Width: oldImageSize.Width, Height: oldImageSize.Height},
			WasResized:      false,
			ProcessingError: err,
		}, err
	}

	// Only use converted data if it's actually smaller (the whole point of conversion is optimization)
	wasCompressed := len(processedData) < len(rotatedData)
	wasResized := newDimensions.Width != oldImageSize.Width || newDimensions.Height != oldImageSize.Height
	
	var finalData []byte
	if wasCompressed {
		finalData = processedData
		log.Printf("Conversion to %s successful: %d → %d bytes", convertFormat, len(rotatedData), len(processedData))
	} else {
		finalData = rotatedData  // Use rotated data (preserves EXIF rotation)
		log.Printf("Conversion to %s skipped - would increase size: %d → %d bytes", convertFormat, len(rotatedData), len(processedData))
	}

	return &ImageProcessingResult{
		ProcessedData: finalData,
		WasCompressed: wasCompressed,
		WasResized:    wasResized,
		NewDimensions: newDimensions,
	}, nil
}

// handleEXIFOrientation handles EXIF orientation correction
// Returns both the corrected image and the corrected bytes
func handleEXIFOrientation(originalData []byte) (*bimg.Image, []byte, error) {
	image := bimg.NewImage(originalData)
	metadata, err := image.Metadata()
	needsRotation := err == nil && metadata.Orientation > EXIF_ORIENTATION_NORMAL

	if needsRotation {
		log.Println("EXIF orientation detected, applying rotation")
		rotatedBytes, err := image.AutoRotate()
		if err != nil {
			log.Printf("EXIF rotation failed: %v", err)
			return image, originalData, nil // Return original if rotation fails
		}
		return bimg.NewImage(rotatedBytes), rotatedBytes, nil
	}

	return image, originalData, nil
}

// calculateResizeDimensions determines the final dimensions for image resizing
func calculateResizeDimensions(original ImageSize, settings ImageProcessingSettings) ImageSize {
	if settings.MaxNarrowSide > 0 {
		return calculateNarrowSideResize(original, settings.MaxNarrowSide)
	} else {
		return calculateBoundingBoxResize(original, settings.MaxWidth, settings.MaxHeight)
	}
}

// calculateNarrowSideResize calculates new dimensions based on narrow side constraint
func calculateNarrowSideResize(original ImageSize, maxNarrowSide int) ImageSize {
	narrowSide := original.Width
	if original.Height < original.Width {
		narrowSide = original.Height
	}

	if narrowSide <= maxNarrowSide {
		return original // No resize needed
	}

	scale := float64(maxNarrowSide) / float64(narrowSide)
	return ImageSize{
		Width:  int(float64(original.Width) * scale),
		Height: int(float64(original.Height) * scale),
	}
}

// calculateBoundingBoxResize calculates new dimensions based on bounding box constraints
// Uses orientation-aware logic: interprets config as long/short edge limits,
// then applies them based on image orientation
func calculateBoundingBoxResize(original ImageSize, maxWidth, maxHeight int) ImageSize {
	// Interpret config as long edge and short edge limits (orientation-agnostic)
	// Note: maxWidth/maxHeight are legacy parameter names; semantically they represent
	// edge limits that should be applied based on image orientation, not literal width/height
	configLongEdge := maxWidth
	configShortEdge := maxHeight
	if maxHeight > maxWidth {
		configLongEdge = maxHeight
		configShortEdge = maxWidth
	}

	// Apply limits based on image orientation
	var effectiveMaxWidth, effectiveMaxHeight int
	isLandscape := original.Width >= original.Height

	if isLandscape {
		// Landscape: long edge constrains width, short edge constrains height
		effectiveMaxWidth = configLongEdge
		effectiveMaxHeight = configShortEdge
	} else {
		// Portrait: short edge constrains width, long edge constrains height
		effectiveMaxWidth = configShortEdge
		effectiveMaxHeight = configLongEdge
	}

	if original.Width <= effectiveMaxWidth && original.Height <= effectiveMaxHeight {
		return original // No resize needed
	}

	scaleWidth := float64(effectiveMaxWidth) / float64(original.Width)
	scaleHeight := float64(effectiveMaxHeight) / float64(original.Height)

	// Use the smaller scale factor to ensure both dimensions fit
	scale := scaleWidth
	if scaleHeight < scaleWidth {
		scale = scaleHeight
	}

	return ImageSize{
		Width:  int(float64(original.Width) * scale),
		Height: int(float64(original.Height) * scale),
	}
}
