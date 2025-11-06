package iservice

import (
	"context"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
)

// MarketingAnalyticsService defines analytics operations for Marketing Staff
type MarketingAnalyticsService interface {
	// GetActiveBrandsCount returns the count of active brands
	GetActiveBrandsCount(ctx context.Context) (int64, error)

	// GetActiveCampaignsCount returns the count of active campaigns
	GetActiveCampaignsCount(ctx context.Context) (int64, error)

	// GetDraftCampaignsCount returns the count of draft campaigns with contracts
	GetDraftCampaignsCount(ctx context.Context) (int64, error)

	// GetMonthlyContractRevenue returns total revenue from paid contract payments for specified month
	GetMonthlyContractRevenue(ctx context.Context, req *requests.MonthlyRevenueRequest) (float64, error)

	// GetTopBrandsByRevenue returns top 4 brands by total revenue (contracts + products)
	GetTopBrandsByRevenue(ctx context.Context, filter *requests.TimeFilter) ([]responses.BrandRevenueResponse, error)

	// GetRevenueByContractType returns revenue breakdown by contract type and standard products
	GetRevenueByContractType(ctx context.Context, filter *requests.TimeFilter) (*responses.RevenueByTypeResponse, error)

	// GetUpcomingDeadlineCampaigns returns campaigns approaching deadline
	GetUpcomingDeadlineCampaigns(ctx context.Context, filter *requests.UpcomingDeadlineFilter) ([]responses.UpcomingCampaignResponse, error)

	// GetDashboard returns aggregated analytics data for marketing dashboard
	GetDashboard(ctx context.Context, filter *requests.DashboardFilter) (*responses.MarketingDashboardResponse, error)
}
