package model

import (
	"core-backend/internal/domain/enum"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type Content struct {
	ID                uuid.UUID          `json:"id" gorm:"type:uuid;primaryKey"`
	TaskID            *uuid.UUID         `json:"task_id,omitempty" gorm:"type:uuid"`
	Title             string             `json:"title" gorm:"type:varchar(500);not null"`
	Body              datatypes.JSON     `json:"body" gorm:"type:text;not null"`
	Type              enum.ContentType   `json:"type" gorm:"type:varchar(50);not null"`
	Status            enum.ContentStatus `json:"status" gorm:"type:varchar(50);not null"`
	ThumbnailURL      *string            `json:"thumbnail_url,omitempty" gorm:"type:varchar(100)"`
	PublishDate       *time.Time         `json:"publish_date,omitempty" gorm:"type:timestamp"`
	AffiliateLink     *string            `json:"affiliate_link,omitempty" gorm:"type:varchar(1000)"`
	AIGeneratedText   *string            `json:"ai_generated_text,omitempty" gorm:"type:text"`
	RejectionFeedback *string            `json:"rejection_feedback,omitempty" gorm:"type:text"`
	CreatedAt         *time.Time         `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt         *time.Time         `json:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt         gorm.DeletedAt     `json:"deleted_at" gorm:"index"`

	// Relationships
	Task            *Task             `json:"task,omitempty" gorm:"foreignKey:TaskID;constraint:OnDelete:SET NULL"`
	Blog            *Blog             `json:"blog,omitempty" gorm:"foreignKey:ContentID;constraint:OnDelete:CASCADE"`
	ContentChannels []*ContentChannel `json:"content_channels,omitempty" gorm:"foreignKey:ContentID"`
}

func (Content) TableName() string { return "contents" }

func (c *Content) BeforeCreate(_ *gorm.DB) error {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	// Default status to DRAFT if not set
	if c.Status == "" {
		c.Status = enum.ContentStatusDraft
	}
	return nil
}
