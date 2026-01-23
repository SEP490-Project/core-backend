package model

import (
	"bytes"
	"core-backend/internal/domain/enum"
	"core-backend/pkg/tiptap"
	"core-backend/pkg/utils"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strings"
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
	ExternalPostID   *string                 `json:"external_post_id" gorm:"type:varchar(255);column:external_post_id"`
	ExternalPostURL  *string                 `json:"external_post_url" gorm:"type:text;column:external_post_url"`
	ExternalPostType *enum.ExternalPostType  `json:"external_post_type" gorm:"column:external_post_type;type:varchar(50)"`
	PublishedAt      *time.Time              `json:"published_at" gorm:"column:published_at"`
	LastError        *string                 `json:"last_error" gorm:"type:text;column:last_error"`
	Metrics          *ContentChannelMetrics  `json:"metrics" gorm:"type:jsonb;column:metrics"`
	Metadata         *ContentChannelMetadata `json:"metadata" gorm:"type:jsonb;column:metadata"`

	CreatedAt time.Time `json:"created_at" gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time `json:"updated_at" gorm:"column:updated_at;autoUpdateTime"`

	// Relationships
	Content         *Content          `json:"-" gorm:"foreignKey:ContentID"`
	ContentComments []*ContentComment `json:"content_comments,omitempty" gorm:"foreignKey:ContentChannelID"`
	Channel         *Channel          `json:"-" gorm:"foreignKey:ChannelID"`
	AffiliateLink   *AffiliateLink    `json:"-" gorm:"foreignKey:AffiliateLinkID"`
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
	return cc.Metrics, nil
}

// GetRenderedBody returns the content body with tracking URLs replaced by this channel's affiliate link.
// Must preload Content and AffiliateLink before calling this method.
// Returns original body if no affiliate link is configured.
func (cc *ContentChannel) GetRenderedBody(baseURL string) datatypes.JSON {
	if cc.Content == nil {
		return nil
	}

	body := cc.Content.Body

	// If no affiliate link, return original body
	if cc.AffiliateLink == nil || cc.AffiliateLinkID == nil {
		return body
	}

	// Replace tracking URL with the full affiliate short URL
	fullAffiliateURL := cc.AffiliateLink.AffiliateURL
	if cc.AffiliateLink.Hash != "" {
		fullAffiliateURL = cc.AffiliateLink.GetFullLink(baseURL)
	}

	renderedBody, err := renderBodyWithAffiliateLink(body, cc.Content.Type, cc.AffiliateLink.TrackingURL, fullAffiliateURL)
	if err != nil {
		return body // Fallback to original on error
	}

	return renderedBody
}

// renderBodyWithAffiliateLink replaces tracking URLs with affiliate URLs in TipTap JSON body
func renderBodyWithAffiliateLink(body []byte, contentType enum.ContentType, trackingURL, affiliateURL string) ([]byte, error) {
	switch contentType {
	case enum.ContentTypePost:
		return tiptap.RenderWithAffiliateLink(body, trackingURL, affiliateURL)
	case enum.ContentTypeVideo:
		if bytes.Contains(body, []byte(trackingURL)) {
			var videoBody map[string]any
			if err := json.Unmarshal(body, &videoBody); err != nil {
				return nil, err
			}
			videoBody["description"] = strings.ReplaceAll(utils.ToString(videoBody["description"]), trackingURL, affiliateURL)
			return json.Marshal(videoBody)
		}

	}
	return body, nil
}

// region: ======== ContentChannelMetrics ========

type ContentChannelMetrics struct {
	CurrentFetched map[string]any                `json:"current_fetched"` // Latest fetched values
	CurrentMapped  map[enum.KPIValueType]float64 `json:"current_mapped"`  // Mapped metrics for KPIs
	LastFetched    map[string]any                `json:"last_fetched"`    // Values from previous fetch
	LastMapped     map[enum.KPIValueType]float64 `json:"last_mapped"`     // Mapped metrics from previous fetch
	LastUpdatedAt  *time.Time                    `json:"last_updated_at"` // When metrics were last updated
}

// region: ======== Website-Specific Metrics Helpers ========
// These structs are used to store detailed engagement data in CurrentFetched for WEBSITE channel.
// Other channels (Facebook, TikTok) use their own format in CurrentFetched.

// WebsiteReactionEntry represents a single user reaction stored in CurrentFetched["reactions"]
type WebsiteReactionEntry struct {
	UserID    uuid.UUID         `json:"user_id"`
	Type      enum.ReactionType `json:"type"`
	ReactedAt time.Time         `json:"reacted_at"`
}

// WebsiteEngagementData represents the structure of CurrentFetched for WEBSITE channel
// Access via GetWebsiteEngagement() helper method
type WebsiteEngagementData struct {
	Reactions       []WebsiteReactionEntry `json:"reactions"`        // Detailed reaction list with user info
	ReactionSummary map[string]int64       `json:"reaction_summary"` // Aggregated counts by type: {"LIKE": 5, "LOVE": 3}
	SharesCount     int64                  `json:"shares_count"`
	ViewsCount      int64                  `json:"views_count"`
}

// GetWebsiteEngagement parses CurrentFetched into WebsiteEngagementData struct
// Returns empty struct if CurrentFetched is nil or not in website format
func (ccm *ContentChannelMetrics) GetWebsiteEngagement() *WebsiteEngagementData {
	if ccm == nil || ccm.CurrentFetched == nil {
		return &WebsiteEngagementData{
			Reactions:       make([]WebsiteReactionEntry, 0),
			ReactionSummary: make(map[string]int64),
		}
	}

	data := &WebsiteEngagementData{
		Reactions:       make([]WebsiteReactionEntry, 0),
		ReactionSummary: make(map[string]int64),
	}

	// Parse reactions array
	if reactionsRaw, ok := ccm.CurrentFetched["reactions"]; ok {
		if reactionsJSON, err := json.Marshal(reactionsRaw); err == nil {
			_ = json.Unmarshal(reactionsJSON, &data.Reactions)
		}
	}

	// Parse reaction summary
	if summaryRaw, ok := ccm.CurrentFetched["reaction_summary"]; ok {
		if summaryJSON, err := json.Marshal(summaryRaw); err == nil {
			_ = json.Unmarshal(summaryJSON, &data.ReactionSummary)
		}
	}

	// Parse shares count
	if sharesRaw, ok := ccm.CurrentFetched["shares_count"]; ok {
		switch v := sharesRaw.(type) {
		case float64:
			data.SharesCount = int64(v)
		case int64:
			data.SharesCount = v
		case int:
			data.SharesCount = int64(v)
		}
	}

	// Parse views count
	if viewsRaw, ok := ccm.CurrentFetched["views_count"]; ok {
		switch v := viewsRaw.(type) {
		case float64:
			data.ViewsCount = int64(v)
		case int64:
			data.ViewsCount = v
		case int:
			data.ViewsCount = int64(v)
		}
	}

	return data
}

// SetWebsiteEngagement updates CurrentFetched with WebsiteEngagementData
// Also updates CurrentMapped with uniform KPI metrics
func (ccm *ContentChannelMetrics) SetWebsiteEngagement(data *WebsiteEngagementData) {
	if ccm.CurrentFetched == nil {
		ccm.CurrentFetched = make(map[string]any)
	}
	if ccm.CurrentMapped == nil {
		ccm.CurrentMapped = make(map[enum.KPIValueType]float64)
	}

	ccm.CurrentFetched["reactions"] = data.Reactions
	ccm.CurrentFetched["reaction_summary"] = data.ReactionSummary
	ccm.CurrentFetched["shares_count"] = data.SharesCount
	ccm.CurrentFetched["views_count"] = data.ViewsCount

	// Update CurrentMapped (uniform KPI format)
	var totalReactions int64
	for _, count := range data.ReactionSummary {
		totalReactions += count
	}
	ccm.CurrentMapped[enum.KPIValueTypeLikes] = float64(totalReactions)
	ccm.CurrentMapped[enum.KPIValueTypeShares] = float64(data.SharesCount)
	ccm.CurrentMapped[enum.KPIValueTypeEngagement] = float64(totalReactions + data.SharesCount)
}

// endregion

func (ccm *ContentChannelMetrics) Value() (driver.Value, error) {
	return json.Marshal(ccm)
}

func (ccm *ContentChannelMetrics) Scan(value any) error {
	if value == nil {
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to unmarshal ContentChannelMetrics value: %v", value)
	}
	return json.Unmarshal(bytes, ccm)
}

// endregion

// region: ======== ContentChannelMetadata ========

// ContentChannelMetadata represents metadata for content channel publishing
// Used to track upload status for async video publishing (TikTok, Facebook)
type ContentChannelMetadata struct {
	// For TikTok video publishing
	UploadID     *string `json:"upload_id,omitempty"`     // TikTok publish_id for tracking upload status
	UploadStatus *string `json:"upload_status,omitempty"` // "pending", "processing", "completed", "failed"

	// For Facebook video publishing
	PostID   *string  `json:"post_id,omitempty"`  // Facebook post ID (after published)
	VideoID  *string  `json:"video_id,omitempty"` // Facebook video ID (before post is published)
	PhotoIDs []string `json:"photo_id,omitempty"`

	// Generic metadata
	Type *string `json:"type,omitempty"` // "video", "photo", "text", etc.
}

func (ccm *ContentChannelMetadata) Value() (driver.Value, error) {
	return json.Marshal(ccm)
}

func (ccm *ContentChannelMetadata) Scan(value any) error {
	if value == nil {
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("failed to unmarshal ContentChannelMetadata value: %v", value)
	}
	return json.Unmarshal(bytes, ccm)
}

func (cc *ContentChannel) GetMetadata() (*ContentChannelMetadata, error) {
	return cc.Metadata, nil
}

func (cc *ContentChannel) SetMetadata(metadata *ContentChannelMetadata) error {
	cc.Metadata = metadata
	return nil
}

// endregion
