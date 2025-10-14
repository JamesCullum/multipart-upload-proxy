package main

import (
	"os"
	"testing"
)

func TestNewConfigFromEnv_Defaults(t *testing.T) {
	clearAllTestEnvVars()
	defer clearAllTestEnvVars()

	cfg := NewConfigFromEnv()

	if cfg.ImgMaxWidth != DEFAULT_IMG_MAX_WIDTH {
		t.Errorf("ImgMaxWidth = %d, want %d", cfg.ImgMaxWidth, DEFAULT_IMG_MAX_WIDTH)
	}
	
	if cfg.ImgMaxHeight != DEFAULT_IMG_MAX_HEIGHT {
		t.Errorf("ImgMaxHeight = %d, want %d", cfg.ImgMaxHeight, DEFAULT_IMG_MAX_HEIGHT)
	}
	
	if cfg.JpegQuality != DEFAULT_JPEG_QUALITY {
		t.Errorf("JpegQuality = %d, want %d", cfg.JpegQuality, DEFAULT_JPEG_QUALITY)
	}
	
	if cfg.ConvertToFormat != DEFAULT_CONVERT_TO_FORMAT {
		t.Errorf("ConvertToFormat = %q, want %q", cfg.ConvertToFormat, DEFAULT_CONVERT_TO_FORMAT)
	}

	expectedMaxPixels := int64(DEFAULT_IMG_MAX_WIDTH) * int64(DEFAULT_IMG_MAX_HEIGHT)
	if cfg.ImgMaxPixels != expectedMaxPixels {
		t.Errorf("ImgMaxPixels = %d, want %d", cfg.ImgMaxPixels, expectedMaxPixels)
	}
}

func TestNewConfigFromEnv_ValidIntegerValues(t *testing.T) {
	clearAllTestEnvVars()
	defer clearAllTestEnvVars()

	os.Setenv("IMG_MAX_WIDTH", "1920")
	os.Setenv("IMG_MAX_HEIGHT", "1080")
	os.Setenv("IMG_MAX_NARROW_SIDE", "720")
	os.Setenv("JPEG_QUALITY", "85")
	os.Setenv("WEBP_QUALITY", "90")
	os.Setenv("NORMALIZE_EXTENSIONS", "1")

	cfg := NewConfigFromEnv()

	if cfg.ImgMaxWidth != 1920 {
		t.Errorf("ImgMaxWidth = %d, want 1920", cfg.ImgMaxWidth)
	}
	
	if cfg.ImgMaxHeight != 1080 {
		t.Errorf("ImgMaxHeight = %d, want 1080", cfg.ImgMaxHeight)
	}
	
	if cfg.ImgMaxNarrowSide != 720 {
		t.Errorf("ImgMaxNarrowSide = %d, want 720", cfg.ImgMaxNarrowSide)
	}
	
	if cfg.JpegQuality != 85 {
		t.Errorf("JpegQuality = %d, want 85", cfg.JpegQuality)
	}
	
	if cfg.WebpQuality != 90 {
		t.Errorf("WebpQuality = %d, want 90", cfg.WebpQuality)
	}
	
	if !cfg.NormalizeExt {
		t.Errorf("NormalizeExt = %t, want true", cfg.NormalizeExt)
	}

	expectedMaxPixels := int64(1920) * int64(1080)
	if cfg.ImgMaxPixels != expectedMaxPixels {
		t.Errorf("ImgMaxPixels = %d, want %d", cfg.ImgMaxPixels, expectedMaxPixels)
	}
}

func TestNewConfigFromEnv_InvalidIntegerValues(t *testing.T) {
	clearAllTestEnvVars()
	defer clearAllTestEnvVars()

	os.Setenv("IMG_MAX_WIDTH", "not-a-number")
	os.Setenv("IMG_MAX_HEIGHT", "-100")  // negative
	os.Setenv("JPEG_QUALITY", "150")     // too high
	os.Setenv("WEBP_QUALITY", "0")
	os.Setenv("NORMALIZE_EXTENSIONS", "2")

	cfg := NewConfigFromEnv()

	if cfg.ImgMaxWidth != DEFAULT_IMG_MAX_WIDTH {
		t.Errorf("ImgMaxWidth = %d, want default %d", cfg.ImgMaxWidth, DEFAULT_IMG_MAX_WIDTH)
	}
	
	if cfg.ImgMaxHeight != DEFAULT_IMG_MAX_HEIGHT {
		t.Errorf("ImgMaxHeight = %d, want default %d", cfg.ImgMaxHeight, DEFAULT_IMG_MAX_HEIGHT)
	}
	
	if cfg.JpegQuality != DEFAULT_JPEG_QUALITY {
		t.Errorf("JpegQuality = %d, want default %d", cfg.JpegQuality, DEFAULT_JPEG_QUALITY)
	}
	
	if cfg.WebpQuality != DEFAULT_WEBP_QUALITY {
		t.Errorf("WebpQuality = %d, want default %d", cfg.WebpQuality, DEFAULT_WEBP_QUALITY)
	}

	expectedNormalizeExt := DEFAULT_NORMALIZE_EXTENSIONS == 1
	if cfg.NormalizeExt != expectedNormalizeExt {
		t.Errorf("NormalizeExt = %t, want default %t", cfg.NormalizeExt, expectedNormalizeExt)
	}
}

func TestNewConfigFromEnv_StringValues(t *testing.T) {
	clearAllTestEnvVars()
	defer clearAllTestEnvVars()

	os.Setenv("FORWARD_DESTINATION", "https://api.example.com/upload")
	os.Setenv("FILE_UPLOAD_FIELD", "image")
	os.Setenv("LISTEN_PATH", "/v1/upload")

	cfg := NewConfigFromEnv()

	if cfg.ForwardDestination != "https://api.example.com/upload" {
		t.Errorf("ForwardDestination = %q, want %q", cfg.ForwardDestination, "https://api.example.com/upload")
	}
	
	if cfg.FileUploadField != "image" {
		t.Errorf("FileUploadField = %q, want %q", cfg.FileUploadField, "image")
	}
	
	if cfg.ListenPath != "/v1/upload" {
		t.Errorf("ListenPath = %q, want %q", cfg.ListenPath, "/v1/upload")
	}
}

func TestNewConfigFromEnv_ConvertToFormat(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		expected string
	}{
		{
			name:     "Empty string",
			envValue: "",
			expected: "",
		},
		{
			name:     "JPEG uppercase",
			envValue: "JPEG",
			expected: "JPEG",
		},
		{
			name:     "jpeg lowercase - should normalize",
			envValue: "jpeg",
			expected: "JPEG",
		},
		{
			name:     "JPG uppercase - should normalize to JPEG",
			envValue: "JPG",
			expected: "JPEG",
		},
		{
			name:     "jpg lowercase - should normalize to JPEG",
			envValue: "jpg",
			expected: "JPEG",
		},
		{
			name:     "WEBP uppercase",
			envValue: "WEBP",
			expected: "WEBP",
		},
		{
			name:     "webp lowercase - should normalize",
			envValue: "webp",
			expected: "WEBP",
		},
		{
			name:     "WebP mixed case - should normalize",
			envValue: "WebP",
			expected: "WEBP",
		},
		{
			name:     "With whitespace - should trim and normalize",
			envValue: " jpeg ",
			expected: "JPEG",
		},
		{
			name:     "JPG with whitespace - should trim and normalize",
			envValue: "  JPG  ",
			expected: "JPEG",
		},
		{
			name:     "Invalid format - should use default",
			envValue: "PNG",
			expected: DEFAULT_CONVERT_TO_FORMAT,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearAllTestEnvVars()
			
			if tt.envValue != "default" {
				os.Setenv("CONVERT_TO_FORMAT", tt.envValue)
			}
			
			cfg := NewConfigFromEnv()
			
			if cfg.ConvertToFormat != tt.expected {
				t.Errorf("ConvertToFormat = %q, want %q", cfg.ConvertToFormat, tt.expected)
			}
			
			clearAllTestEnvVars()
		})
	}
}

func TestNewConfigFromEnv_Int64Values(t *testing.T) {
	clearAllTestEnvVars()
	defer clearAllTestEnvVars()

	os.Setenv("UPLOAD_MAX_SIZE", "209715200")

	cfg := NewConfigFromEnv()

	if cfg.UploadMaxSize != 209715200 {
		t.Errorf("UploadMaxSize = %d, want 209715200", cfg.UploadMaxSize)
	}
}

func TestNewConfigFromEnv_InvalidInt64Values(t *testing.T) {
	clearAllTestEnvVars()
	defer clearAllTestEnvVars()

	tests := []struct {
		name     string
		envValue string
	}{
		{"Not a number", "not-a-number"},
		{"Negative", "-100"},
		{"Zero", "0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearAllTestEnvVars()
			os.Setenv("UPLOAD_MAX_SIZE", tt.envValue)

			cfg := NewConfigFromEnv()

			expectedDefault := int64(100 << 20)
			if cfg.UploadMaxSize != expectedDefault {
				t.Errorf("UploadMaxSize = %d, want default %d", cfg.UploadMaxSize, expectedDefault)
			}

			clearAllTestEnvVars()
		})
	}
}

func TestNewConfigFromEnv_NarrowSideZeroAllowed(t *testing.T) {
	clearAllTestEnvVars()
	defer clearAllTestEnvVars()

	os.Setenv("IMG_MAX_NARROW_SIDE", "0")

	cfg := NewConfigFromEnv()

	if cfg.ImgMaxNarrowSide != 0 {
		t.Errorf("ImgMaxNarrowSide = %d, want 0", cfg.ImgMaxNarrowSide)
	}
}

func clearAllTestEnvVars() {
	envVars := []string{
		"IMG_MAX_WIDTH",
		"IMG_MAX_HEIGHT",
		"IMG_MAX_NARROW_SIDE",
		"JPEG_QUALITY",
		"WEBP_QUALITY",
		"NORMALIZE_EXTENSIONS",
		"UPLOAD_MAX_SIZE",
		"FORWARD_DESTINATION",
		"FILE_UPLOAD_FIELD",
		"LISTEN_PATH",
		"CONVERT_TO_FORMAT",
	}
	
	for _, envVar := range envVars {
		os.Unsetenv(envVar)
	}
}