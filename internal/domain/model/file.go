package model

import (
	"core-backend/internal/domain/enum"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type File struct {
	ID          uuid.UUID       `json:"id" gorm:"type:uuid;column:id;primaryKey"`
	Name        string          `json:"file_name" gorm:"type:varchar(255);column:file_name;not null"`
	AltText     *string         `json:"alt_text" gorm:"type:varchar(255);column:alt_text"`
	URL         string          `json:"url" gorm:"type:text;column:url"`
	StorageKey  string          `json:"storage_key" gorm:"type:text;column:storage_key;not null"` // S3 key
	MimeType    string          `json:"mime_type" gorm:"type:varchar(100);column:mime_type;not null"`
	Size        int64           `json:"size" gorm:"type:bigint;column:size;default:0"`
	Metadata    datatypes.JSON  `json:"metadata" gorm:"type:jsonb;column:metadata"`
	Status      enum.FileStatus `json:"status" gorm:"type:varchar(50);column:status;default:'PENDING';not null"`
	ErrorReason *string         `json:"error_reason,omitempty" gorm:"type:text;column:error_reason"`
	UploadedAt  *time.Time      `json:"uploaded_at" gorm:"column:uploaded_at"`
	UploadedBy  *uuid.UUID      `json:"uploaded_by" gorm:"type:uuid;column:uploaded_by"`
	CreatedAt   time.Time       `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt   time.Time       `json:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt   gorm.DeletedAt  `json:"deleted_at" gorm:"index"`
}

func (File) TableName() string { return "files" }

func (f *File) BeforeCreate(tx *gorm.DB) error {
	if f.ID == uuid.Nil {
		f.ID = uuid.New()
	}
	return nil
}
