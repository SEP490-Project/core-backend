package consumers

type VideoUploadMessage struct {
	UserID   string  `json:"user_id"`
	FilePath string  `json:"file_path"`
	Key      string  `json:"key"`
	Action   *string `json:"action,omitempty"`
	FileID   string  `json:"file_id"`
}
