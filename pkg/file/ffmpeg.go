// Package file provides utilities to interact with media files using FFmpeg tools or pure Go implementations.
package file

import (
	"encoding/json"
	"fmt"
	"os/exec"
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
