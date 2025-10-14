package main

import (
	"bytes"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"path/filepath"
	"strings"
)

func reformatMultipart(w http.ResponseWriter, r *http.Request, cfg *Config) (string, *bytes.Buffer, error) {
	r.ParseMultipartForm(cfg.UploadMaxSize)
	file, handler, err := r.FormFile(cfg.FileUploadField)
	if err != nil {
		return "", nil, err
	}
	defer file.Close()

	byteContainer, err := io.ReadAll(file)
	if err != nil {
		log.Printf("Failed to read file: %v", err)
		return "", nil, err
	}

	settings := ImageProcessingSettings{
		MaxWidth:        cfg.ImgMaxWidth,
		MaxHeight:       cfg.ImgMaxHeight,
		MaxNarrowSide:   cfg.ImgMaxNarrowSide,
		JpegQuality:     cfg.JpegQuality,
		WebpQuality:     cfg.WebpQuality,
		ConvertToFormat: cfg.ConvertToFormat,
	}

	result, err := processImageWithStrategy(byteContainer, settings)

	var wasImageProcessed bool
	var actuallyCompressed bool
	var wasResized bool

	if err == nil {
		wasImageProcessed = true
		actuallyCompressed = result.WasCompressed
		wasResized = result.WasResized
		byteContainer = result.ProcessedData
	} else {
		log.Printf("Image processing error: %v", err)
		wasImageProcessed = false
		actuallyCompressed = false
		wasResized = false
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	for formKey := range r.Form {
		formValue := r.Form.Get(formKey)
		fw, _ := writer.CreateFormField(formKey)
		io.Copy(fw, strings.NewReader(formValue))
	}

	var finalFilename string
	var finalMimeType string

	convertFormat := cfg.ConvertToFormat

	if wasImageProcessed && actuallyCompressed {
		switch convertFormat {
		case "JPEG":
			finalMimeType = JPEG_MIME_TYPE
			if cfg.NormalizeExt {
				finalFilename = changeExtensionToJPG(handler.Filename)
				log.Printf("Converted to JPEG with normalized filename: %s -> %s", handler.Filename, finalFilename)
			} else {
				finalFilename = handler.Filename
				log.Printf("Converted to JPEG but keeping original filename: %s", finalFilename)
			}
		case "WEBP":
			finalMimeType = WEBP_MIME_TYPE
			if cfg.NormalizeExt {
				finalFilename = changeExtensionToWebP(handler.Filename)
				log.Printf("Converted to WebP with normalized filename: %s -> %s", handler.Filename, finalFilename)
			} else {
				finalFilename = handler.Filename
				log.Printf("Converted to WebP but keeping original filename: %s", finalFilename)
			}
		default:
			// Fallback (shouldn't happen)
			finalMimeType = JPEG_MIME_TYPE
			finalFilename = handler.Filename
			log.Printf("Unknown convert format, defaulting to JPEG MIME: %s", finalFilename)
		}
	} else if wasImageProcessed && !actuallyCompressed {
		finalFilename = handler.Filename
		finalMimeType = handler.Header.Get("Content-Type")
		if finalMimeType == "" {
			finalMimeType = DEFAULT_MIME_TYPE
		}
		if convertFormat == "" {
			if wasResized {
				log.Printf("Image resized but format conversion disabled: %s (%s)", finalFilename, finalMimeType)
			} else {
				log.Printf("Image processed but no changes needed: %s (%s)", finalFilename, finalMimeType)
			}
		} else {
			log.Printf("Image processed but original kept (better compression): %s (%s)", finalFilename, finalMimeType)
		}
	} else {
		finalFilename = handler.Filename
		finalMimeType = handler.Header.Get("Content-Type")
		if finalMimeType == "" {
			finalMimeType = DEFAULT_MIME_TYPE
		}
		log.Printf("Non-image file or processing failed, keeping original: %s (%s)", finalFilename, finalMimeType)
	}

	fw, _ := CreateFormFileWithMime(writer, cfg.FileUploadField, finalFilename, finalMimeType)
	io.Copy(fw, bytes.NewReader(byteContainer))
	writer.Close()

	contentType := writer.FormDataContentType()

	return contentType, body, nil
}

var quoteEscaper = strings.NewReplacer("\\", "\\\\", `"`, "\\\"")

func escapeQuotes(s string) string {
	return quoteEscaper.Replace(s)
}

func changeExtensionToJPG(filename string) string {
	extension := filepath.Ext(filename)

	if extension == "" {
		return filename + ".JPG"
	}

	nameWithoutExt := strings.TrimSuffix(filename, extension)
	return nameWithoutExt + ".JPG"
}

func changeExtensionToWebP(filename string) string {
	extension := filepath.Ext(filename)

	if extension == "" {
		return filename + ".WEBP"
	}

	nameWithoutExt := strings.TrimSuffix(filename, extension)
	return nameWithoutExt + ".WEBP"
}

func CreateFormFileWithMime(w *multipart.Writer, fieldname, filename, mimeType string) (io.Writer, error) {
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", `form-data; name="`+escapeQuotes(fieldname)+`"; filename="`+escapeQuotes(filename)+`"`)
	h.Set("Content-Type", mimeType)
	return w.CreatePart(h)
}
