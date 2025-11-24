package model

import (
	"core-backend/internal/domain/enum"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type ContentChannel struct {
	ID              uuid.UUID           `json:"id" gorm:"type:uuid;column:id;primaryKey"`
	ContentID       uuid.UUID           `json:"content_id" gorm:"type:uuid;column:content_id;not null;index"`
	ChannelID       uuid.UUID           `json:"channel_id" gorm:"type:uuid;column:channel_id;not null;index"`
	AffiliateLinkID *uuid.UUID          `json:"affiliate_link_id" gorm:"type:uuid;column:affiliate_link_id;index"`
	PostDate        *time.Time          `json:"post_date" gorm:"column:post_date"`
	AutoPostStatus  enum.AutoPostStatus `json:"auto_post_status" gorm:"column:auto_post_status;not null;type:varchar(35)"`

	// Publishing fields
	ExternalPostID   *string                `json:"external_post_id" gorm:"type:varchar(255);column:external_post_id"`
	ExternalPostURL  *string                `json:"external_post_url" gorm:"type:text;column:external_post_url"`
	ExternalPostType *enum.ExternalPostType `json:"external_post_type" gorm:"column:external_post_type;type:varchar(50)"`
	PublishedAt      *time.Time             `json:"published_at" gorm:"column:published_at"`
	LastError        *string                `json:"last_error" gorm:"type:text;column:last_error"`
	Metrics          datatypes.JSON         `json:"metrics" gorm:"type:jsonb;column:metrics"`

	CreatedAt time.Time `json:"created_at" gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"column:updated_at;autoUpdateTime"`

	// Relationships
	Content       *Content       `json:"-" gorm:"foreignKey:ContentID"`
	Channel       *Channel       `json:"-" gorm:"foreignKey:ChannelID"`
	AffiliateLink *AffiliateLink `json:"-" gorm:"foreignKey:ContentID;references:ContentID"`
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
