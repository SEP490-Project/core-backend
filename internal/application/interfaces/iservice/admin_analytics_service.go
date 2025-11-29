package iservice

import (
	"context"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
)

// AdminAnalyticsService defines the interface for Admin analytics operations
type AdminAnalyticsService interface {
	// GetDashboard returns the complete Admin dashboard with platform-wide metrics
	GetDashboard(ctx context.Context, req *requests.AdminDashboardRequest) (*responses.AdminDashboardResponse, error)

	// GetUsersOverview returns user statistics and growth
	GetUsersOverview(ctx context.Context, req *requests.UsersOverviewRequest) (*responses.UsersOverviewResponse, error)

	// GetPlatformRevenue returns platform-wide revenue analytics
	GetPlatformRevenue(ctx context.Context, req *requests.PlatformRevenueRequest) (*responses.PlatformRevenueResponse, error)

	// GetSystemHealth returns system health metrics
	GetSystemHealth(ctx context.Context) (*responses.SystemHealthResponse, error)

	// GetUserGrowth returns user growth over time
	GetUserGrowth(ctx context.Context, req *requests.UserGrowthRequest) ([]responses.UserGrowthPoint, error)

	// GetContractsSummary returns contract statistics
	GetContractsSummary(ctx context.Context, req *requests.DashboardRequest) (*responses.ContractsSummary, error)

	// GetCampaignsSummary returns campaign statistics
	GetCampaignsSummary(ctx context.Context, req *requests.DashboardRequest) (*responses.AdminCampaignsSummary, error)
}
