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

func (fs FileStatus) IsValid() bool {
	switch fs {
	case FileStatusPending, FileStatusUploading, FileStatusUploaded, FileStatusFailed:
		return true
	}
	return false
}

func (fs *FileStatus) Scan(value any) error {
	s, ok := value.([]byte)
	if !ok {
		// It might also be a string
		str, ok := value.(string)
		if !ok {
			return fmt.Errorf("failed to scan FileStatus: invalid type %T", value)
		}
		s = []byte(str)
	}

	// Convert the byte slice to our type.
	*fs = FileStatus(s)
	return nil
}

func (fs FileStatus) Value() (driver.Value, error) {
	return string(fs), nil
}

func (fs FileStatus) String() string { return string(fs) }
