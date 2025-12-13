package iservice

import (
	"context"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"

	"github.com/google/uuid"
)

// ContentEngagementService defines the interface for content engagement operations
// This handles reactions, shares, comments for WEBSITE channel only.
// All engagement data is stored in ContentChannel.Metrics (JSONB) using ContentChannelMetrics struct.
// Comments are stored in the ContentComment model with relationship to ContentChannel.
type ContentEngagementService interface {
	// RecordEngagement handles all engagement actions in a single API endpoint.
	// Supported actions:
	// - add_reaction: Add or update user's reaction (requires reaction_type)
	// - remove_reaction: Remove user's reaction
	// - share: Record a share action
	// - add_comment: Add a comment (requires comment_text)
	// - edit_comment: Edit a comment (requires comment_id and comment_text)
	// - delete_comment: Delete a comment (requires comment_id)
	// - add_comment_reaction: React to a comment (requires comment_id and reaction_type)
	// - remove_comment_reaction: Remove reaction from comment (requires comment_id)
	//
	// Note: content_id and user_id should be set in context by the handler before calling this method.
	RecordEngagement(ctx context.Context, req *requests.ContentEngagementRequest) (*responses.ContentEngagementResponse, error)

	// GetEngagementSummary returns engagement summary for a content on WEBSITE channel
	// Returns reaction counts, comment count, share count, and current user's reaction
	GetEngagementSummary(ctx context.Context, contentID uuid.UUID) (*responses.WebsiteEngagementSummary, error)

	// GetUserEngagementStatus returns whether the current user has liked/shared the content
	GetUserEngagementStatus(ctx context.Context, contentID, userID uuid.UUID) (*responses.UserEngagementStatus, error)
}
