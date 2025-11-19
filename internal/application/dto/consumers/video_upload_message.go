package consumers

type VideoUploadMessage struct {
	UserID          string   `json:"user_id"`
	FilePath        string   `json:"file_path"`
	Key             string   `json:"key"`
	Action          *string  `json:"action,omitempty"`
	FileID          string   `json:"file_id"`
	IsHLS           bool     `json:"is_hls"`
	Resolutions     []string `json:"resolutions,omitempty"`
	SegmentDuration int      `json:"segment_duration,omitempty"` // Duration of each segment in seconds (default 10)
}
