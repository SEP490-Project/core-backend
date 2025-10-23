package model

import (
	"core-backend/internal/domain/enum"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ContentChannel struct {
	ID             uuid.UUID           `json:"id" gorm:"type:uuid;column:id;primaryKey"`
	ContentID      uuid.UUID           `json:"content_id" gorm:"type:uuid;column:content_id;not null;index"`
	ChannelID      uuid.UUID           `json:"channel_id" gorm:"type:uuid;column:channel_id;not null;index"`
	PostDate       *time.Time          `json:"post_date" gorm:"column:post_date"`
	AutoPostStatus enum.AutoPostStatus `json:"auto_post_status" gorm:"column:auto_post_status;not null;type:varchar(35);check:auto_post_status in ('PENDING', 'POSTED', 'FAILED', 'SKIPPED')"`
	CreatedAt      time.Time           `json:"created_at" gorm:"column:created_at;autoCreateTime"`
	UpdatedAt      time.Time           `json:"updated_at" gorm:"column:updated_at;autoUpdateTime"`

	// Relationships
	Content *Content `json:"-" gorm:"foreignKey:ContentID"`
	Channel *Channel `json:"-" gorm:"foreignKey:ChannelID"`
}

func (ContentChannel) TableName() string { return "content_channels" }

func (cc *ContentChannel) BeforeCreate(tx *gorm.DB) error {
	if cc.ID == uuid.Nil {
		cc.ID = uuid.New()
	}
	if cc.AutoPostStatus == "" {
		cc.AutoPostStatus = enum.AutoPostStatusPending
	}
	return nil
}
