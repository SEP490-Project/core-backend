package consumers

type VideoUploadMessage struct {
	UserID   string
	FilePath string
	Key      string
	Action   *string
}
