package responses

import (
	"core-backend/internal/domain/enum"
	"time"

	"github.com/google/uuid"
)

// ContentResponse DTO for content API responses
type ContentResponse struct {
	ID                uuid.UUID             `json:"id"`
	TaskID            *uuid.UUID            `json:"task_id,omitempty"`
	Title             string                `json:"title"`
	Description       *string               `json:"description,omitempty"`
	ThumbnailURL      *string               `json:"thumbnail_url,omitempty"`
	Body              any                   `json:"body"`
	Type              enum.ContentType      `json:"type"`
	Status            enum.ContentStatus    `json:"status"`
	PublishDate       *time.Time            `json:"publish_date,omitempty"`
	AIGeneratedText   *string               `json:"ai_generated_text,omitempty"`
	RejectionFeedback *string               `json:"rejection_feedback,omitempty"`
	CreatedAt         string                `json:"created_at"`
	UpdatedAt         string                `json:"updated_at"`
	Blog              *BlogResponse         `json:"blog,omitempty"`
	ContentChannels   []ContentChannelBrief `json:"content_channels,omitempty"`
	// AffiliateLink     *string               `json:"affiliate_link,omitempty"`
}

// ContentChannelBrief for nested content channel info
type ContentChannelBrief struct {
	ID             uuid.UUID  `json:"id"`
	ChannelID      uuid.UUID  `json:"channel_id"`
	ChannelName    string     `json:"channel_name"`
	PostDate       *time.Time `json:"post_date,omitempty"`
	AutoPostStatus string     `json:"auto_post_status"`
	AffiliateLink  *string    `json:"affiliate_link,omitempty"`
}

// ContentPaginationResponse for paginated content responses
// Only used for Swaggo swagger docs generation
type ContentPaginationResponse PaginationResponse[ContentResponse]
