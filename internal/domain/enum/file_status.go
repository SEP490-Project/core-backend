package enum

import (
	"database/sql/driver"
	"fmt"
)

type FileStatus string

const (
	FileStatusPending   FileStatus = "PENDING"   // Upload initiated, not yet complete
	FileStatusUploading FileStatus = "UPLOADING" // Chunks being uploaded
	FileStatusUploaded  FileStatus = "UPLOADED"  // Successfully uploaded to storage
	FileStatusFailed    FileStatus = "FAILED"    // Upload failed
)

func (s FileStatus) IsValid() bool {
	switch s {
	case FileStatusPending, FileStatusUploading, FileStatusUploaded, FileStatusFailed:
		return true
	}
	return false
}

func (s FileStatus) Value() (driver.Value, error) {
	if !s.IsValid() {
		return nil, fmt.Errorf("invalid file status: %s", s)
	}
	return string(s), nil
}

func (s *FileStatus) Scan(value any) error {
	if value == nil {
		*s = FileStatusPending
		return nil
	}
	strVal, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to scan FileStatus")
	}
	*s = FileStatus(strVal)
	return nil
}

func (s FileStatus) String() string { return string(s) }
