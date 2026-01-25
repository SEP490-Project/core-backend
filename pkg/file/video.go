package file

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// VideoMetadata contains the cleaned, parsed data ready for your application.
type VideoMetadata struct {
	Format    string  // mp4, webm, mov
	Codec     string  // h264, h265, vp8, vp9
	Width     int     // px
	Height    int     // px
	FPS       float64 // frames per second
	Duration  float64 // seconds
	SizeBytes int64   // bytes
}

// ExtractVideoMetadata gets the raw data and cleans it up.
func ExtractVideoMetadata(filePath string) (*VideoMetadata, error) {
	// 1. Get Raw Data
	raw, err := RunFFprobe(filePath)
	if err != nil {
		return nil, err
	}

	// 2. Find Video Stream
	var videoStream *struct {
		CodecName    string `json:"codec_name"`
		CodecType    string `json:"codec_type"`
		Width        int    `json:"width"`
		Height       int    `json:"height"`
		AvgFrameRate string `json:"avg_frame_rate"`
		Duration     string `json:"duration"`
	}

	for i, s := range raw.Streams {
		if s.CodecType == "video" {
			videoStream = &raw.Streams[i]
			break
		}
	}

	if videoStream == nil {
		return nil, errors.New("no video stream found")
	}

	// 3. Parse Fields
	// Size
	sizeBytes, _ := strconv.ParseInt(raw.Format.Size, 10, 64)
	if sizeBytes == 0 {
		// Fallback to OS stat if ffprobe fails to read size
		info, _ := os.Stat(filePath)
		if info != nil {
			sizeBytes = info.Size()
		}
	}

	// FPS
	fps := parseFPS(videoStream.AvgFrameRate)

	// Duration
	duration, _ := strconv.ParseFloat(videoStream.Duration, 64)

	// Format (Pick the first match if multiple are returned, e.g. "mov,mp4,m4a")
	fmtName := strings.Split(raw.Format.FormatName, ",")[0]

	// Codec Normalization
	codec := videoStream.CodecName
	if codec == "hevc" {
		codec = "h265" // Normalize for consistency
	}

	return &VideoMetadata{
		Format:    fmtName,
		Codec:     codec,
		Width:     videoStream.Width,
		Height:    videoStream.Height,
		FPS:       fps,
		Duration:  duration,
		SizeBytes: sizeBytes,
	}, nil
}

// ValidateVideo checks if the video meets the specific project requirements.
func ValidateVideo(m *VideoMetadata) error {
	// 1. Size Restriction (Max 4GB)
	const maxSizeBytes = 4 * 1024 * 1024 * 1024
	if m.SizeBytes > maxSizeBytes {
		return fmt.Errorf("file size %.2f GB exceeds limit of 4GB", float64(m.SizeBytes)/(1024*1024*1024))
	}

	// 2. Format Restriction
	validFormats := map[string]bool{"mp4": true, "webm": true, "mov": true}
	if !validFormats[m.Format] {
		// FFprobe sometimes returns "mov,mp4". Check substring or mapped name.
		isMovOrMp4 := strings.Contains(m.Format, "mp4") || strings.Contains(m.Format, "mov")
		if !isMovOrMp4 && !validFormats[m.Format] {
			return fmt.Errorf("unsupported format: %s", m.Format)
		}
	}

	// 3. Codec Restriction
	validCodecs := map[string]bool{"h264": true, "h265": true, "vp8": true, "vp9": true, "hevc": true}
	if !validCodecs[m.Codec] {
		return fmt.Errorf("unsupported codec: %s", m.Codec)
	}

	// 4. Framerate Restriction (23 - 60 FPS)
	// We use a small buffer (epsilon) for floating point comparison
	if m.FPS < 22.9 || m.FPS > 60.1 {
		return fmt.Errorf("fps %.2f is out of range (23-60)", m.FPS)
	}

	// 5. Picture Size (360 - 4096)
	if m.Width < 360 || m.Height < 360 {
		return fmt.Errorf("resolution %dx%d too small (min 360px)", m.Width, m.Height)
	}
	if m.Width > 4096 || m.Height > 4096 {
		return fmt.Errorf("resolution %dx%d too large (max 4096px)", m.Width, m.Height)
	}

	// 6. Duration (Max 10 minutes = 600 seconds)
	if m.Duration > 600.5 {
		return fmt.Errorf("duration %.2fs exceeds 10 minutes", m.Duration)
	}

	return nil
}

// Helper to convert "60/1" or "30000/1001" to float
func parseFPS(fraction string) float64 {
	parts := strings.Split(fraction, "/")
	if len(parts) == 2 {
		num, _ := strconv.ParseFloat(parts[0], 64)
		den, _ := strconv.ParseFloat(parts[1], 64)
		if den > 0 {
			return num / den
		}
	}
	return 0.0
}
