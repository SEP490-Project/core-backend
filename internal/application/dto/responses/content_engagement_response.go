package responses

import "github.com/google/uuid"

// ContentEngagementResponse represents the response after an engagement action
type ContentEngagementResponse struct {
	Success   bool                      `json:"success"`
	Message   string                    `json:"message"`
	Metrics   *WebsiteEngagementSummary `json:"metrics,omitempty"`
	CommentID *uuid.UUID                `json:"comment_id,omitempty"` // For add_comment action
}

// WebsiteEngagementSummary represents the engagement summary for website channel content
type WebsiteEngagementSummary struct {
	TotalReactions  int64            `json:"total_reactions"`
	ReactionsByType map[string]int64 `json:"reactions_by_type"` // {"LIKE": 10, "LOVE": 5}
	TotalComments   int64            `json:"total_comments"`
	TotalShares     int64            `json:"total_shares"`
	UserReaction    *string          `json:"user_reaction,omitempty"` // Current user's reaction type (nil if not reacted)
}

// UserEngagementStatus represents the current user's engagement status with a content
type UserEngagementStatus struct {
	HasLiked  bool    `json:"has_liked"`
	HasShared bool    `json:"has_shared"`
	LikeType  *string `json:"like_type,omitempty"` // The type of reaction if liked
	SharedAt  *string `json:"shared_at,omitempty"` // When the user last shared
}

// ReactionResponse represents the response after a reaction action
type ReactionResponse struct {
	ID           uuid.UUID `json:"id"`
	ContentID    uuid.UUID `json:"content_id"`
	ReactionType string    `json:"reaction_type"`
	UserID       uuid.UUID `json:"user_id"`
	CreatedAt    string    `json:"created_at"`
}

// CommentResponse represents the response after adding a comment
type CommentResponse struct {
	ID          uuid.UUID  `json:"id"`
	ContentID   uuid.UUID  `json:"content_id"`
	ParentID    *uuid.UUID `json:"parent_id,omitempty"`
	CommentText string     `json:"comment_text"`
	UserID      uuid.UUID  `json:"user_id"`
	Username    string     `json:"username"`
	AvatarURL   *string    `json:"avatar_url,omitempty"`
	CreatedAt   string     `json:"created_at"`
}

// ShareResponse represents the response after a share action
type ShareResponse struct {
	ID          uuid.UUID `json:"id"`
	ContentID   uuid.UUID `json:"content_id"`
	Platform    *string   `json:"platform,omitempty"`
	TotalShares int64     `json:"total_shares"` // Updated total shares count
	CreatedAt   string    `json:"created_at"`
}

// EngagementStatsResponse represents detailed engagement statistics for content
type EngagementStatsResponse struct {
	ContentID       uuid.UUID        `json:"content_id"`
	ChannelCode     string           `json:"channel_code"`
	TotalReactions  int64            `json:"total_reactions"`
	ReactionsByType map[string]int64 `json:"reactions_by_type"`
	TotalComments   int64            `json:"total_comments"`
	TotalShares     int64            `json:"total_shares"`
	TotalViews      int64            `json:"total_views,omitempty"`
	CTR             float64          `json:"ctr,omitempty"`
}
