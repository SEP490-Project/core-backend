package model

import (
	"core-backend/internal/domain/enum"
	"encoding/json"
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
	Metadata         datatypes.JSON         `json:"metadata" gorm:"type:jsonb;column:metadata"`

	CreatedAt time.Time `json:"created_at" gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"column:updated_at;autoUpdateTime"`

	// Relationships
	Content       *Content       `json:"-" gorm:"foreignKey:ContentID"`
	Channel       *Channel       `json:"-" gorm:"foreignKey:ChannelID"`
	AffiliateLink *AffiliateLink `json:"-" gorm:"foreignKey:ContentID;references:ContentID"`
}

type ContentChannelMetrics struct {
	CurrentFetched map[string]any     `json:"current_fetched"` // Latest fetched values
	CurrentMapped  map[string]float64 `json:"current_mapped"`  // Mapped metrics for KPIs
	LastFetched    map[string]any     `json:"last_fetched"`    // Values from previous fetch
	LastMapped     map[string]float64 `json:"last_mapped"`     // Mapped metrics from previous fetch
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

func (cc ContentChannel) GetMetrics() (*ContentChannelMetrics, error) {
	var metrics ContentChannelMetrics
	if len(cc.Metrics) > 0 {
		if err := json.Unmarshal(cc.Metrics, &metrics); err != nil {
			return nil, err
		}
	}

	return &metrics, nil
}

// ContentChannelMetadata represents metadata for content channel publishing
// Used to track upload status for async video publishing (TikTok, Facebook)
type ContentChannelMetadata struct {
	// For TikTok video publishing
	UploadID     *string `json:"upload_id,omitempty"`     // TikTok publish_id for tracking upload status
	UploadStatus *string `json:"upload_status,omitempty"` // "pending", "processing", "completed", "failed"

	// For Facebook video publishing
	VideoID *string `json:"video_id,omitempty"` // Facebook video ID (before post is published)

	// Generic metadata
	Type *string `json:"type,omitempty"` // "video", "photo", "text", etc.
}

func (cc *ContentChannel) GetMetadata() (*ContentChannelMetadata, error) {
	var metadata ContentChannelMetadata
	if len(cc.Metadata) > 0 {
		if err := json.Unmarshal(cc.Metadata, &metadata); err != nil {
			return nil, err
		}
	}
	return &metadata, nil
}

func (cc *ContentChannel) SetMetadata(metadata *ContentChannelMetadata) error {
	data, err := json.Marshal(metadata)
	if err != nil {
		return err
	}
	cc.Metadata = data
	return nil
}
