package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type File struct {
	ID         uuid.UUID  `json:"id" gorm:"type:uuid;column:id;primaryKey:gen_random_uuid()"`
	Name       string     `json:"file_name" gorm:"type:varchar(255);column:file_name;not null"`
	AltTest    *string    `json:"alt_text" gorm:"type:varchar(255);column:alt_text"`
	URL        string     `json:"url" gorm:"type:text;column:url;not null"`
	MimeType   string     `json:"mime_type" gorm:"type:varchar(100);column:mime_type;not null"`
	Size       int64      `json:"size" gorm:"type:bigint;column:size;not null"`
	UploadedAt time.Time  `json:"uploaded_at" gorm:"column:uploaded_at;autoCreateTime"`
	UploadedBy *uuid.UUID `json:"uploaded_by" gorm:"type:uuid;column:uploaded_by;"`
}

func (File) TableName() string { return "files" }

func (f *File) BeforeCreate(tx *gorm.DB) error {
	if f.ID == uuid.Nil {
		f.ID = uuid.New()
	}
	return nil
}
