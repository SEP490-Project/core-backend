package irepository

import (
	"context"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
)

// MarketingAnalyticsRepository provides analytics data for marketing staff dashboard
type MarketingAnalyticsRepository interface {
	// GetActiveBrandsCount returns the count of brands with status = 'ACTIVE'
	GetActiveBrandsCount(ctx context.Context) (int64, error)

	// GetActiveCampaignsCount returns the count of campaigns with status = 'RUNNING'
	GetActiveCampaignsCount(ctx context.Context) (int64, error)

	// GetDraftCampaignsCount returns the count of campaigns with status = 'DRAFT' AND contract_id IS NOT NULL
	GetDraftCampaignsCount(ctx context.Context) (int64, error)

	// GetMonthlyContractRevenue returns sum of PAID contract payments for specified month
	GetMonthlyContractRevenue(ctx context.Context, year, month int) (float64, error)

	// GetTopBrandsByRevenue returns top 4 brands by total revenue (contract + product sales)
	GetTopBrandsByRevenue(ctx context.Context, filter *requests.TimeFilter) ([]responses.BrandRevenueResponse, error)

	// GetRevenueByContractType returns revenue breakdown: 4 contract types + standard products
	GetRevenueByContractType(ctx context.Context, filter *requests.TimeFilter) (*responses.RevenueByTypeResponse, error)

	// GetUpcomingDeadlineCampaigns returns campaigns with end_date within X days and status = 'RUNNING'
	GetUpcomingDeadlineCampaigns(ctx context.Context, daysBeforeDeadline int) ([]responses.UpcomingCampaignResponse, error)
}
