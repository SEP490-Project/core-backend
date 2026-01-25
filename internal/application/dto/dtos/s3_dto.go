package dtos

// BatchUploadItem represents a single item to be uploaded in a batch to S3
type BatchUploadItem struct {
	LocalPath   string
	S3Key       string
	ContentType string
}
