package model

import (
	"core-backend/internal/domain/enum"
	"core-backend/pkg/utils"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type Content struct {
	ID                uuid.UUID          `json:"id" gorm:"type:uuid;primaryKey"`
	TaskID            *uuid.UUID         `json:"task_id,omitempty" gorm:"type:uuid"`
	Title             string             `json:"title" gorm:"type:varchar(500);not null"`
	Description       *string            `json:"description,omitempty" gorm:"type:text"`
	Body              datatypes.JSON     `json:"body" gorm:"type:text;not null"`
	Type              enum.ContentType   `json:"type" gorm:"type:varchar(50);not null"`
	Status            enum.ContentStatus `json:"status" gorm:"type:varchar(50);not null"`
	ThumbnailURL      *string            `json:"thumbnail_url,omitempty" gorm:"type:varchar(100)"`
	PublishDate       *time.Time         `json:"publish_date,omitempty" gorm:"type:timestamp"`
	AIGeneratedText   *string            `json:"ai_generated_text,omitempty" gorm:"type:text"`
	RejectionFeedback *string            `json:"rejection_feedback,omitempty" gorm:"type:text"`
	CreatedAt         *time.Time         `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt         *time.Time         `json:"updated_at" gorm:"autoUpdateTime"`
	DeletedAt         gorm.DeletedAt     `json:"deleted_at" gorm:"index"`
	Tags              []string           `json:"tags,omitempty" gorm:"type:text[]"`

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

type ContentBodyVideo struct {
	VideoURL      string `json:"video_url" gorm:"type:varchar(100);not null"`
	S3Key         string `json:"s3_key" gorm:"type:varchar(255)"`
	Title         string `json:"title" gorm:"type:varchar(255)"`
	Description   string `json:"description" gorm:"type:text"`
	OriginalURL   string `json:"original_url" gorm:"type:text"`
	AffiliateLink string `json:"affiliate_link" gorm:"type:text"`
}

func (c *Content) GetVideoBody(channelID uuid.UUID, baseAffiliateLinkURL string) (*ContentBodyVideo, error) {
	var videoBody ContentBodyVideo
	if err := json.Unmarshal(c.Body, &videoBody); err != nil {
		return nil, err
	}

	if videoBody.VideoURL == "" {
		panic("content body must contain 'video_url' field for VIDEO type")
	}

	videoURL, err := url.Parse(videoBody.VideoURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse video URL: %w", err)
	}
	videoS3Key := strings.Trim(videoURL.Path, "/")
	var encodedURL string
	if encodedURL, err = utils.EncodeIndividualPathSegments(videoURL.String()); err != nil {
		zap.L().Error("Failed to encode video S3 URL",
			zap.Error(err))
		return nil, fmt.Errorf("failed to encode video S3 URL: %w", err)
	}
	videoBody.VideoURL = encodedURL
	videoBody.S3Key = videoS3Key

	if videoBody.Title == "" {
		videoBody.Title = "Untitled Video"
	}

	for i := range c.ContentChannels {
		cc := c.ContentChannels[i]
		if cc.ChannelID == channelID && cc.AffiliateLinkID != nil && cc.AffiliateLink != nil {
			videoBody.OriginalURL = cc.AffiliateLink.TrackingURL
			videoBody.AffiliateLink = cc.AffiliateLink.GetFullLink(baseAffiliateLinkURL)
			break
		}
	}
	if videoBody.AffiliateLink != "" && !strings.Contains(videoBody.Description, videoBody.AffiliateLink) {
		if videoBody.Description != "" && strings.Contains(videoBody.Description, videoBody.OriginalURL) {
			videoBody.Description = strings.ReplaceAll(videoBody.Description, videoBody.OriginalURL, videoBody.AffiliateLink)
		} else if videoBody.Description != "" && !strings.Contains(videoBody.Description, videoBody.OriginalURL) {
			videoBody.Description = fmt.Sprintf("%s\n\nFound out more at: %s", videoBody.Description, videoBody.AffiliateLink)
		} else if videoBody.Description == "" {
			videoBody.Description = fmt.Sprintf("Found out more at: %s", videoBody.AffiliateLink)
		}
	}
	return &videoBody, nil
}
