package model

import (
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type Blog struct {
	ContentID uuid.UUID      `json:"content_id" gorm:"type:uuid;column:content_id;primaryKey"`
	AuthorID  uuid.UUID      `json:"author_id" gorm:"type:uuid;column:author_id;not null"`
	Tags      datatypes.JSON `json:"tags" gorm:"type:jsonb;column:tags;default:'[]'::jsonb"`
	Excerpt   *string        `json:"excerpt" gorm:"type:text;column:excerpt;default:''"`
	ReadTime  int            `json:"read_time" gorm:"type:int;column:read_time;default:0"`

	// Relationships
	Content *Content `json:"content" gorm:"foreignKey:ContentID"`
	User    *User    `json:"user" gorm:"foreignKey:AuthorID"`
}

func (Blog) TableName() string { return "blogs" }

func (b *Blog) BeforeCreate(tx *gorm.DB) error {
	if b.Tags == nil {
		b.Tags = datatypes.JSON([]byte("[]"))
	}
	if b.ReadTime < 0 {
		b.ReadTime = 0
	}
	return nil
}
