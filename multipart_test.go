package main

import (
	"bytes"
	"mime/multipart"
	"strings"
	"testing"
)

func TestEscapeQuotes(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "No quotes or backslashes",
			input:    "simple_filename.jpg",
			expected: "simple_filename.jpg",
		},
		{
			name:     "Single quote",
			input:    `file"name.jpg`,
			expected: `file\"name.jpg`,
		},
		{
			name:     "Multiple quotes",
			input:    `file"with"quotes.jpg`,
			expected: `file\"with\"quotes.jpg`,
		},
		{
			name:     "Backslash",
			input:    `file\name.jpg`,
			expected: `file\\name.jpg`,
		},
		{
			name:     "Mixed quotes and backslashes",
			input:    `file\"name.jpg`,
			expected: `file\\\"name.jpg`,
		},
		{
			name:     "Empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := escapeQuotes(tt.input)
			if result != tt.expected {
				t.Errorf("escapeQuotes(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestChangeExtensionToJPG(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "PNG to JPG",
			input:    "photo.png",
			expected: "photo.JPG",
		},
		{
			name:     "JPEG to JPG",
			input:    "image.jpeg",
			expected: "image.JPG",
		},
		{
			name:     "Uppercase extension",
			input:    "picture.PNG",
			expected: "picture.JPG",
		},
		{
			name:     "No extension",
			input:    "filename",
			expected: "filename.JPG",
		},
		{
			name:     "Multiple dots",
			input:    "file.with.dots.png",
			expected: "file.with.dots.JPG",
		},
		{
			name:     "Dot at end",
			input:    "filename.",
			expected: "filename.JPG",
		},
		{
			name:     "Empty filename",
			input:    "",
			expected: ".JPG",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := changeExtensionToJPG(tt.input)
			if result != tt.expected {
				t.Errorf("changeExtensionToJPG(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestChangeExtensionToWebP(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "PNG to WebP",
			input:    "photo.png",
			expected: "photo.WEBP",
		},
		{
			name:     "JPEG to WebP",
			input:    "image.jpeg",
			expected: "image.WEBP",
		},
		{
			name:     "JPG to WebP",
			input:    "picture.jpg",
			expected: "picture.WEBP",
		},
		{
			name:     "No extension",
			input:    "filename",
			expected: "filename.WEBP",
		},
		{
			name:     "Multiple dots",
			input:    "file.with.dots.png",
			expected: "file.with.dots.WEBP",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := changeExtensionToWebP(tt.input)
			if result != tt.expected {
				t.Errorf("changeExtensionToWebP(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestCreateFormFileWithMime(t *testing.T) {
	tests := []struct {
		name      string
		fieldname string
		filename  string
		mimeType  string
		wantErr   bool
	}{
		{
			name:      "Simple JPEG file",
			fieldname: "file",
			filename:  "photo.jpg",
			mimeType:  "image/jpeg",
			wantErr:   false,
		},
		{
			name:      "PNG with quotes in filename",
			fieldname: "upload",
			filename:  `photo"with"quotes.png`,
			mimeType:  "image/png",
			wantErr:   false,
		},
		{
			name:      "WebP file",
			fieldname: "image",
			filename:  "picture.webp",
			mimeType:  "image/webp",
			wantErr:   false,
		},
		{
			name:      "File with backslashes",
			fieldname: "file",
			filename:  `path\to\file.jpg`,
			mimeType:  "image/jpeg",
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := &bytes.Buffer{}
			writer := multipart.NewWriter(body)

			_, err := CreateFormFileWithMime(writer, tt.fieldname, tt.filename, tt.mimeType)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateFormFileWithMime() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Close writer to finalize the multipart form
			writer.Close()

			content := body.String()
			
			expectedFieldname := escapeQuotes(tt.fieldname)
			expectedFilename := escapeQuotes(tt.filename)
			
			if !strings.Contains(content, `name="`+expectedFieldname+`"`) {
				t.Errorf("CreateFormFileWithMime() missing field name %q in content: %s", expectedFieldname, content)
			}
			
			if !strings.Contains(content, `filename="`+expectedFilename+`"`) {
				t.Errorf("CreateFormFileWithMime() missing filename %q in content: %s", expectedFilename, content)
			}
			
			if !strings.Contains(content, "Content-Type: "+tt.mimeType) {
				t.Errorf("CreateFormFileWithMime() missing MIME type %q in content: %s", tt.mimeType, content)
			}
		})
	}
}