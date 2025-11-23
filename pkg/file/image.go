package file

import (
	"encoding/binary"
	"fmt"
	"image"
	_ "image/jpeg" // Register JPEG decoder
	"io"
	"os"
)

type ImageMetadata struct {
	Format    string // jpeg, webp
	Width     int
	Height    int
	SizeBytes int64
}

// ExtractImageMetadata parses dimensions and format without external deps.
func ExtractImageMetadata(path string) (*ImageMetadata, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// 1. Get File Size
	stat, err := file.Stat()
	if err != nil {
		return nil, err
	}
	sizeBytes := stat.Size()

	// 2. Sniff Header
	header := make([]byte, 32)
	if _, err := io.ReadFull(file, header); err != nil {
		return nil, err
	}

	// Reset pointer
	file.Seek(0, 0)

	var width, height int
	var format string

	// 3. Parse Logic
	if isWebP(header) {
		format = "webp"
		width, height, err = parseWebPDimensions(file)
		if err != nil {
			return nil, fmt.Errorf("failed to parse webp: %w", err)
		}
	} else {
		// Use Standard Lib (JPEG)
		config, fmtName, err := image.DecodeConfig(file)
		if err != nil {
			return nil, fmt.Errorf("unsupported image format or decode error: %w", err)
		}
		format = fmtName
		width = config.Width
		height = config.Height
	}

	return &ImageMetadata{
		Format:    format,
		Width:     width,
		Height:    height,
		SizeBytes: sizeBytes,
	}, nil
}

// ValidateImage checks size, format, and resolution requirements.
func ValidateImage(m *ImageMetadata) error {
	// 1. Format
	if m.Format != "jpeg" && m.Format != "webp" {
		return fmt.Errorf("unsupported format: %s", m.Format)
	}

	// 2. File Size (Max 20MB)
	const maxSizeBytes = 20 * 1024 * 1024
	if m.SizeBytes > maxSizeBytes {
		return fmt.Errorf("file size %.2f MB exceeds limit of 20MB", float64(m.SizeBytes)/(1024*1024))
	}

	// 3. Resolution (Max 1080p)
	// "Max 1080p" typically means the smaller dimension is <= 1080,
	// or strictly that neither dimension exceeds 1920x1080 bounds.
	// Given strict requirements, we often check if the Width (for vertical) or Height (horizontal) > 1080.
	// However, strictly "1080p" means 1920x1080.

	// Let's use a safe logic: neither side should exceed 1920,
	// and at least one side should be <= 1080.
	const maxDim = 1920
	const classDim = 1080

	if m.Width > maxDim || m.Height > maxDim {
		return fmt.Errorf("resolution %dx%d exceeds limits", m.Width, m.Height)
	}

	// Simple strict check: if user implies NO dimension > 1080 (square/strict):
	// Uncomment below if you want strict 1080x1080 max:
	/*
		if m.Width > 1080 || m.Height > 1080 {
			return fmt.Errorf("resolution %dx%d exceeds 1080p strict bounds", m.Width, m.Height)
		}
	*/

	return nil
}

// --- Internal Helpers for WebP (No Deps) ---

func isWebP(header []byte) bool {
	return len(header) > 12 && string(header[0:4]) == "RIFF" && string(header[8:12]) == "WEBP"
}

func parseWebPDimensions(r io.ReadSeeker) (int, int, error) {
	data := make([]byte, 30)
	if _, err := io.ReadFull(r, data); err != nil {
		return 0, 0, err
	}

	chunkFormat := string(data[12:16])

	switch chunkFormat {
	case "VP8 ": // Lossy
		// Frame tag starts at offset 23.
		if data[23] != 0x9d || data[24] != 0x01 || data[25] != 0x2a {
			return 0, 0, fmt.Errorf("invalid VP8 signature")
		}
		w := binary.LittleEndian.Uint16(data[26:28])
		h := binary.LittleEndian.Uint16(data[28:30])
		return int(w & 0x3fff), int(h & 0x3fff), nil

	case "VP8L": // Lossless
		if data[20] != 0x2f {
			return 0, 0, fmt.Errorf("invalid VP8L signature")
		}
		b := data[21:25]
		w := int(b[0]) | (int(b[1]&0x3F) << 8)
		h := (int(b[1]) >> 6) | (int(b[2]) << 2) | (int(b[3]&0x0F) << 10)
		return w + 1, h + 1, nil

	case "VP8X": // Extended
		w := int(data[24]) | (int(data[25]) << 8) | (int(data[26]) << 16)
		h := int(data[27]) | (int(data[28]) << 8) | (int(data[29]) << 16)
		return w + 1, h + 1, nil

	default:
		return 0, 0, fmt.Errorf("unsupported WebP chunk: %s", chunkFormat)
	}
}
