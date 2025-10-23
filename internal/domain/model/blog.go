package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// Blog is a weak entity extending Content for blog-specific attributes (Type=POST only)
type Blog struct {
	ContentID uuid.UUID      `json:"content_id" gorm:"type:uuid;primaryKey"`
	AuthorID  uuid.UUID      `json:"author_id" gorm:"type:uuid;not null"`
	Tags      datatypes.JSON `json:"tags" gorm:"type:jsonb"`
	Excerpt   *string        `json:"excerpt,omitempty" gorm:"type:text"`
	ReadTime  *int           `json:"read_time,omitempty" gorm:"type:integer"`
	CreatedAt *time.Time     `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt *time.Time     `json:"updated_at" gorm:"autoUpdateTime"`

	// Relationships
	Content *Content `json:"content,omitempty" gorm:"foreignKey:ContentID;constraint:OnDelete:CASCADE"`
	Author  *User    `json:"author,omitempty" gorm:"foreignKey:AuthorID"`
}

// TableName specifies the table name for Blog
func (Blog) TableName() string {
	return "blogs"
}
