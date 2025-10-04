package model

import (
	"core-backend/internal/domain/enum"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Content struct {
	ID              uuid.UUID          `json:"id" gorm:"column:id;primaryKey"`
	TaskID          uuid.UUID          `json:"task_id" gorm:"type:uuid;not null;index"`
	Title           string             `json:"title" gorm:"column:title;not null;type:varchar(255)"`
	Type            enum.ContentType   `json:"type" gorm:"column:type;not null;type:varchar(35);check:type in ('POST', 'VIDEO')"`
	Body            string             `json:"body" gorm:"column:body;not null;type:text"`
	PublishDate     *time.Time         `json:"publish_date" gorm:"column:publish_date"`
	AffiliateLink   *string            `json:"affiliate_link" gorm:"column:affiliate_link;type:text"`
	ContentStatus   enum.ContentStatus `json:"content_status" gorm:"column:content_status;not null;type:varchar(35);check:content_status in ('DRAFT', 'AWAIT_STAFF', 'AWAIT_BRAND', 'REJECTED', 'APPROVED', 'POSTED')"`
	AIGeneratedText *string            `json:"ai_generated_text" gorm:"column:ai_generated_text;type:text"`
	CreatedAt       time.Time          `json:"created_at" gorm:"column:created_at;autoCreateTime"`
	UpdatedAt       time.Time          `json:"updated_at" gorm:"column:updated_at;autoUpdateTime"`
	DeletedAt       gorm.DeletedAt     `json:"deleted_at" gorm:"column:deleted_at;index"`

	// Relationships
	Task *Task `json:"-" gorm:"foreignKey:TaskID"`
}

func (Content) TableName() string { return "contents" }

func (c *Content) BeforeCreate(tx *gorm.DB) error {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	return nil
}
