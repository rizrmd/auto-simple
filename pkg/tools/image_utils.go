package tools

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/image/draw"
	"golang.org/x/image/webp"
)

const (
	MaxImageSize     = 20 * 1024 * 1024 // 20MB max file size
	MaxImageWidth    = 2048             // Max width for optimization
	MaxImageHeight   = 2048             // Max height for optimization
	OptimizedQuality = 85               // JPEG quality for optimization
	LLMMaxWidth      = 250              // Max width for LLM processing
	LLMMaxHeight     = 250              // Max height for LLM processing
	LLMQuality       = 75               // JPEG quality for LLM processing
)

// DetectImageType detects the image type from file extension and magic bytes
func DetectImageType(filename string, data []byte) string {
	// First try to detect from file extension
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".webp":
		return "image/webp"
	case ".gif":
		return "image/gif"
	}

	// Fallback to magic bytes detection
	if len(data) >= 8 {
		// JPEG
		if data[0] == 0xFF && data[1] == 0xD8 && data[2] == 0xFF {
			return "image/jpeg"
		}
		// PNG
		if bytes.HasPrefix(data, []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}) {
			return "image/png"
		}
		// WebP
		if bytes.HasPrefix(data, []byte("RIFF")) && len(data) > 12 &&
			bytes.HasPrefix(data[8:12], []byte("WEBP")) {
			return "image/webp"
		}
		// GIF
		if bytes.HasPrefix(data, []byte("GIF87a")) || bytes.HasPrefix(data, []byte("GIF89a")) {
			return "image/gif"
		}
	}

	// Default to JPEG if we can't detect
	return "image/jpeg"
}

// decodeImage decodes an image from byte data based on MIME type
func decodeImage(data []byte, mimeType string) (image.Image, error) {
	switch mimeType {
	case "image/jpeg":
		return jpeg.Decode(bytes.NewReader(data))
	case "image/png":
		return png.Decode(bytes.NewReader(data))
	case "image/webp":
		return webp.Decode(bytes.NewReader(data))
	default:
		// Try JPEG as fallback
		return jpeg.Decode(bytes.NewReader(data))
	}
}

// resizeImage resizes an image to fit within the specified dimensions while maintaining aspect ratio
func resizeImage(img image.Image, maxWidth, maxHeight int) image.Image {
	// Get original dimensions
	bounds := img.Bounds()
	originalWidth := bounds.Dx()
	originalHeight := bounds.Dy()

	// Calculate scaling factor
	scaleX := float64(maxWidth) / float64(originalWidth)
	scaleY := float64(maxHeight) / float64(originalHeight)
	scale := scaleX
	if scaleY < scaleX {
		scale = scaleY
	}

	// If image is already small enough, return as-is
	if scale >= 1.0 {
		return img
	}

	// Calculate new dimensions
	newWidth := int(float64(originalWidth) * scale)
	newHeight := int(float64(originalHeight) * scale)

	// Create new image
	newImg := image.NewRGBA(image.Rect(0, 0, newWidth, newHeight))

	// Resize using bilinear interpolation
	draw.BiLinear.Scale(newImg, newImg.Bounds(), img, bounds, draw.Over, nil)

	return newImg
}

// encodeImage encodes an image to JPEG format with specified quality
func encodeImage(img image.Image, quality int) ([]byte, error) {
	var buf bytes.Buffer
	err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: quality})
	if err != nil {
		return nil, fmt.Errorf("failed to encode image as JPEG: %w", err)
	}
	return buf.Bytes(), nil
}

// ResizeImageForLLM resizes an image specifically for LLM processing
func ResizeImageForLLM(data []byte, mimeType string) ([]byte, error) {
	// Decode the image
	img, err := decodeImage(data, mimeType)
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	// Resize for LLM processing
	resizedImg := resizeImage(img, LLMMaxWidth, LLMMaxHeight)

	// Encode as JPEG with appropriate quality
	return encodeImage(resizedImg, LLMQuality)
}

// OptimizeImage optimizes an image if it's too large
func OptimizeImage(data []byte, mimeType string) ([]byte, error) {
	// If image is already small enough, return as-is
	if len(data) < 5*1024*1024 { // 5MB threshold
		return data, nil
	}

	// Decode the image
	img, err := decodeImage(data, mimeType)
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	// Resize if dimensions are too large
	resizedImg := resizeImage(img, MaxImageWidth, MaxImageHeight)

	// Encode with optimized quality
	return encodeImage(resizedImg, OptimizedQuality)
}

// ValidateImage checks if an image meets size requirements
func ValidateImage(data []byte) error {
	if len(data) > MaxImageSize {
		return fmt.Errorf("image too large (%.2fMB), max size is %.2fMB",
			float64(len(data))/1024/1024, float64(MaxImageSize)/1024/1024)
	}
	return nil
}

// SaveImageToFile saves image data to a file with the appropriate extension
func SaveImageToFile(data []byte, filename string, mimeType string) (string, error) {
	// Determine appropriate file extension
	ext := ".jpg"
	switch mimeType {
	case "image/png":
		ext = ".png"
	case "image/webp":
		ext = ".webp"
	case "image/gif":
		ext = ".gif"
	}

	// Ensure filename has the correct extension
	if !strings.HasSuffix(strings.ToLower(filename), ext) {
		filename = strings.TrimSuffix(filename, filepath.Ext(filename)) + ext
	}

	// Create data directory if it doesn't exist
	if err := os.MkdirAll("data", 0755); err != nil {
		return "", fmt.Errorf("failed to create data directory: %w", err)
	}

	// Save the file
	filePath := filepath.Join("data", filename)
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return "", fmt.Errorf("failed to save image file: %w", err)
	}

	return filePath, nil
}
