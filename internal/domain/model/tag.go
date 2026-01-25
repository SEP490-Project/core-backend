package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Tag struct {
	ID          uuid.UUID      `json:"id" gorm:"type:uuid;column:id;primaryKey;default:gen_random_uuid()"`
	Name        string         `json:"name" gorm:"type:varchar(255);column:name;unique;not null"`
	Description *string        `json:"description" gorm:"type:text;column:description"`
	UsageCount  int            `json:"usage_count" gorm:"type:integer;column:usage_count;default:0"`
	CreatedAt   time.Time      `json:"created_at" gorm:"type:timestamptz;column:created_at;autoCreateTime"`
	CreatedByID *uuid.UUID     `json:"created_by" gorm:"type:uuid;column:created_by"`
	UpdatedAt   time.Time      `json:"updated_at" gorm:"type:timestamptz;column:updated_at;autoUpdateTime"`
	UpdatedByID *uuid.UUID     `json:"updated_by" gorm:"type:uuid;column:updated_by"`
	DeletedAt   gorm.DeletedAt `json:"deleted_at" gorm:"type:timestamptz;column:deleted_at;index"`

	// Relationships
	// Tags []Blog `json:"blogs" gorm:"many2many:blog_tags;joinForeignKey:tag_id;joinReferences:blog_id"`
	Blogs []Blog `json:"blogs" gorm:"many2many:blog_tags;joinForeignKey:TagID;JoinReferences:BlogID"`
}

func (Tag) TableName() string { return "tags" }

func (t *Tag) BeforeCreate(tx *gorm.DB) error {
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}
	return nil
}
