package file

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go.uber.org/zap"
)

// BaseFileMetadata contains metadata applicable to any file type.
type BaseFileMetadata struct {
	FileName     string    `json:"file_name"`
	Extension    string    `json:"extension"`
	MIMEType     string    `json:"mime_type"`
	SizeBytes    int64     `json:"size_bytes"`
	Hash         string    `json:"hash_sha256"` // Unique file fingerprint
	LastModified time.Time `json:"last_modified"`
}

// ExtractBaseMetadata analyzes any file to get size, hash, and MIME type.
func ExtractBaseMetadata(filePath string) (*BaseFileMetadata, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// 1. Get Basic Stats (Size, Time, Name)
	stat, err := file.Stat()
	if err != nil {
		return nil, err
	}

	// 2. Detect MIME Type
	// We read the first 512 bytes to sniff the content type.
	buffer := make([]byte, 512)
	_, err = file.Read(buffer)
	if err != nil && err != io.EOF {
		return nil, err
	}

	// Reset file pointer to beginning after reading header
	if _, err := file.Seek(0, 0); err != nil {
		return nil, err
	}

	// DetectContentType returns "application/octet-stream" if unknown.
	mimeType := http.DetectContentType(buffer)
	ext := strings.ToLower(filepath.Ext(filePath))

	// Refinement: If http detects generic octet-stream, or if we want to be specific
	// (e.g., css, js, json), fallback to extension detection.
	if mimeType == "application/octet-stream" || mimeType == "text/plain" {
		byExt := mime.TypeByExtension(ext)
		if byExt != "" {
			mimeType = byExt
		}
	}

	// 3. Calculate SHA256 Hash (Useful for deduplication/integrity)
	// Note: For extremely large files (e.g. >2GB), you might want to skip this
	// or do it in a separate background process to avoid blocking.
	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return nil, err
	}
	fileHash := hex.EncodeToString(hasher.Sum(nil))

	return &BaseFileMetadata{
		FileName:     stat.Name(),
		Extension:    ext,
		MIMEType:     mimeType,
		SizeBytes:    stat.Size(),
		Hash:         fileHash,
		LastModified: stat.ModTime(),
	}, nil
}

func ExtractFileMetadata(filePath string) (map[string]any, error) {
	// 1. Run General Extraction (Applies to ALL files)
	genMeta, err := ExtractBaseMetadata(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to extract general metadata: %w", err)
	}

	// Populate basic info immediately
	data := make(map[string]any)
	{
		data["file_name"] = genMeta.FileName
		data["extension"] = genMeta.Extension
		data["mime_type"] = genMeta.MIMEType
		data["size_bytes"] = genMeta.SizeBytes
		data["hash_sha256"] = genMeta.Hash
		data["last_modified"] = genMeta.LastModified
	}

	// 2. Prepare for specific extraction
	ext := genMeta.Extension
	videoExts := map[string]bool{".mp4": true, ".mov": true, ".webm": true, ".mkv": true}
	imageExts := map[string]bool{".jpg": true, ".jpeg": true, ".webp": true, ".png": true}
	if videoExts[ext] {
		// --- Video Specifics ---
		vidMeta, err := ExtractVideoMetadata(filePath)
		if err != nil {
			zap.L().Warn("Failed to extract specific video details", zap.Error(err))
		} else {
			if vidMeta.Format == "mov" {
				data["mime_type"] = "video/quicktime"
			}
			data["extension"] = vidMeta.Format
			data["codec"] = vidMeta.Codec
			data["image_width"] = vidMeta.Width
			data["image_height"] = vidMeta.Height
			data["fps"] = vidMeta.FPS
			data["duration_seconds"] = vidMeta.Duration
		}

	} else if imageExts[ext] {
		// --- Image Specifics ---
		imgMeta, err := ExtractImageMetadata(filePath)
		if err != nil {
			zap.L().Warn("Failed to extract specific image details", zap.Error(err))
		} else {
			data["mime_type"] = "image/" + imgMeta.Format
			data["image_width"] = imgMeta.Width
			data["image_height"] = imgMeta.Height
		}

	}

	return data, nil
}
