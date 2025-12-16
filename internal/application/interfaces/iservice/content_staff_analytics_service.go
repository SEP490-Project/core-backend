package iservice

import (
	"context"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"

	"github.com/google/uuid"
)

// ContentStaffAnalyticsService defines the interface for the content staff dashboard
// This is the consolidated service for content staff analytics
type ContentStaffAnalyticsService interface {
	// GetDashboard returns the complete content dashboard with all metrics, charts, and lists
	// userID is used to check which alerts have been acknowledged by the user
	GetDashboard(ctx context.Context, filter *requests.ContentDashboardFilterRequest, userID uuid.UUID) (*responses.ContentDashboardResponse, error)

	// GetChannelDetails returns detailed metrics for a specific channel
	GetChannelDetails(ctx context.Context, channelID uuid.UUID, filter *requests.ChannelDetailsRequest) (*responses.ChannelDetailsResponse, error)
}
