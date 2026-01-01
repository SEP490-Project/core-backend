package model

import (
	"core-backend/internal/domain/enum"
	"core-backend/pkg/utils"
	"database/sql/driver"
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
	Tags              ContentTag         `json:"tags,omitempty" gorm:"type:jsonb"`
	CreatedBy         *uuid.UUID         `json:"created_by,omitempty" gorm:"type:uuid"`
	UpdatedBy         *uuid.UUID         `json:"updated_by,omitempty" gorm:"type:uuid"`

	// Relationships
	Task            *Task             `json:"task,omitempty" gorm:"foreignKey:TaskID;constraint:OnDelete:SET NULL"`
	Blog            *Blog             `json:"blog,omitempty" gorm:"foreignKey:ContentID;constraint:OnDelete:CASCADE"`
	ContentChannels []*ContentChannel `json:"content_channels,omitempty" gorm:"foreignKey:ContentID"`
	CreatedUser     *User             `json:"created_user,omitempty" gorm:"foreignKey:CreatedBy;constraint:OnDelete:SET NULL"`
	UpdatedUser     *User             `json:"updated_user,omitempty" gorm:"foreignKey:UpdatedBy;constraint:OnDelete:SET NULL"`
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

func (c *Content) GetRenderedBody(baseURL string, channelCode string) datatypes.JSON {
	if len(c.ContentChannels) == 0 {
		return c.Body
	}

	var contentChannel *ContentChannel
	for i := range c.ContentChannels {
		cc := c.ContentChannels[i]
		if cc.Channel == nil || !strings.EqualFold(cc.Channel.Code, channelCode) || cc.AffiliateLink == nil {
			continue
		}

		contentChannel = cc
	}
	if contentChannel == nil || contentChannel.AffiliateLink == nil || contentChannel.AffiliateLinkID == nil {
		return c.Body
	}

	// Replace tracking URL with the full affiliate short URL
	fullAffiliateURL := contentChannel.AffiliateLink.AffiliateURL
	if contentChannel.AffiliateLink.Hash != "" {
		fullAffiliateURL = contentChannel.AffiliateLink.GetFullLink(baseURL)
	}

	renderedBody, err := renderBodyWithAffiliateLink(c.Body, c.Type, contentChannel.AffiliateLink.TrackingURL, fullAffiliateURL)
	if err != nil {
		return c.Body // Fallback to original on error
	}

	return renderedBody
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

// region: ======== ContentTag ========

type ContentTag []string

func (ct *ContentTag) Scan(value any) error {
	if value == nil {
		*ct = ContentTag{}
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to scan ContentTag: %v", value)
	}
	return json.Unmarshal(bytes, ct)
}

func (ct ContentTag) Value() (driver.Value, error) {
	if len(ct) == 0 {
		return "[]", nil
	}
	return json.Marshal(ct)
}

// endregion
