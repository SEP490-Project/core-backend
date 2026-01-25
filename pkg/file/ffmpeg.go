// Package file provides utilities to interact with media files using FFmpeg tools or pure Go implementations.
package file

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// FFProbeRawOutput represents the JSON structure returned by ffprobe.
type FFProbeRawOutput struct {
	Streams []struct {
		CodecName    string `json:"codec_name"` // e.g., h264, hevc
		CodecType    string `json:"codec_type"` // video or audio
		Width        int    `json:"width"`
		Height       int    `json:"height"`
		AvgFrameRate string `json:"avg_frame_rate"` // e.g., "30/1"
		Duration     string `json:"duration"`       // e.g., "10.000"
	} `json:"streams"`
	Format struct {
		FormatName string `json:"format_name"` // e.g., "mov,mp4,m4a"
		Duration   string `json:"duration"`
		Size       string `json:"size"` // File size in bytes as string
	} `json:"format"`
}

// RunFFprobe executes the ffprobe command on the file path and returns the raw struct.
// Requirement: FFmpeg must be installed on the system.
func RunFFprobe(filePath string) (*FFProbeRawOutput, error) {
	cmd := exec.Command("ffprobe",
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		filePath,
	)

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to execute ffprobe: %w", err)
	}

	var data FFProbeRawOutput
	if err := json.Unmarshal(output, &data); err != nil {
		return nil, fmt.Errorf("failed to parse ffprobe json: %w", err)
	}

	return &data, nil
}

func ExtractThumbnailFromVideo(opt struct {
	VideoPath          string
	ThumbnailPath      string
	TimestampSeconds   int
	UseThumbnailFilter bool
	Width              int
	Height             int
}) error {
	// 1. Sanitize Inputs
	if opt.TimestampSeconds < 0 {
		opt.TimestampSeconds = 0
	}

	// 2. Build Arguments
	args := []string{}

	// FAST SEEK: Put -ss before -i
	args = append(args, "-ss", fmt.Sprintf("%d", opt.TimestampSeconds))

	// INPUT
	args = append(args, "-i", opt.VideoPath)

	// 3. Construct Filter Chain
	// We need to join multiple filters (thumbnail, scale) with a comma
	var filters []string

	if opt.UseThumbnailFilter {
		filters = append(filters, "thumbnail")
	}

	// Dynamic Scaling logic
	// If User provided Width OR Height, add scale filter
	if opt.Width > 0 || opt.Height > 0 {
		w := opt.Width
		if w == 0 {
			w = -1 // -1 tells ffmpeg to keep aspect ratio based on height
		}
		h := opt.Height
		if h == 0 {
			h = -1 // -1 tells ffmpeg to keep aspect ratio based on width
		}
		filters = append(filters, fmt.Sprintf("scale=%d:%d", w, h))
	}

	// If we have any filters, add the -vf flag
	if len(filters) > 0 {
		args = append(args, "-vf", strings.Join(filters, ","))
	}

	// 4. Output Settings
	args = append(args, "-frames:v", "1") // Extract 1 frame
	args = append(args, "-y")             // Overwrite output if exists

	// Quality settings (optional, but recommended for JPG)
	if strings.HasSuffix(strings.ToLower(opt.ThumbnailPath), ".jpg") {
		args = append(args, "-q:v", "2") // Best balance for JPG
	}

	// OUTPUT FILE
	args = append(args, opt.ThumbnailPath)

	// 5. Execute
	cmd := exec.Command("ffmpeg", args...)

	// Capture Stderr to debug FFmpeg errors
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		// Return the FFmpeg error message, not just "exit status 1"
		return fmt.Errorf("ffmpeg failed: %s | %w", stderr.String(), err)
	}

	return nil
}
