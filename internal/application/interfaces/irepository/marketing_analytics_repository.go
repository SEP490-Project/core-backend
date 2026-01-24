package irepository

import (
	"context"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/domain/constant"
)

// MarketingAnalyticsRepository provides analytics data for marketing staff dashboard
type MarketingAnalyticsRepository interface {
	// GetActiveBrandsCount returns the count of brands with status = 'ACTIVE'
	GetActiveBrandsCount(ctx context.Context) (int64, error)

	// GetActiveCampaignsCount returns the count of campaigns with status = 'RUNNING'
	GetActiveCampaignsCount(ctx context.Context) (int64, error)

	// GetDraftCampaignsCount returns the count of campaigns with status = 'DRAFT' AND contract_id IS NOT NULL
	GetDraftCampaignsCount(ctx context.Context) (int64, error)

	// GetGrossContractRevenue returns sum of contract payment amounts (before refunds) for specified period
	// Includes PAID and KOL_REFUND_APPROVED statuses, uses paid_at for accurate timing
	GetGrossContractRevenue(ctx context.Context, filter *requests.DashboardFilterRequest) (float64, error)

	// GetNetContractRevenue returns net contract revenue (gross - refunds) for specified period
	// Includes PAID and KOL_REFUND_APPROVED statuses, subtracts refund_amount for KOL_REFUND_APPROVED
	GetNetContractRevenue(ctx context.Context, filter *requests.DashboardFilterRequest) (grossRevenue float64, netRevenue float64, totalRefunds float64, err error)

	// GetTopBrandsByRevenue returns top brands by total revenue (contract + product sales)
	GetTopBrandsByRevenue(ctx context.Context, filter *requests.DashboardFilterRequest) ([]responses.BrandRevenueResponse, error)

	// GetRevenueByContractType returns revenue breakdown: 4 contract types + standard products
	GetRevenueByContractType(ctx context.Context, filter *requests.DashboardFilterRequest) (*responses.RevenueByTypeResponse, error)

	// GetUpcomingDeadlineCampaigns returns campaigns with end_date within X days and status = 'RUNNING'
	GetUpcomingDeadlineCampaigns(ctx context.Context, daysBeforeDeadline int) ([]responses.UpcomingCampaignResponse, error)

	// NEW METHODS FOR DASHBOARD REFACTOR

	// GetContractStatusDistribution returns contract counts grouped by status categories
	GetContractStatusDistribution(ctx context.Context, filter *requests.DashboardFilterRequest) (*responses.ContractStatusDistributionResponse, error)

	// GetTaskStatusDistribution returns task counts grouped by status
	GetTaskStatusDistribution(ctx context.Context, filter *requests.DashboardFilterRequest) (*responses.TaskStatusDistributionResponse, error)

	// GetContractBaseRevenueOverTime returns contract base revenue grouped by time periods
	GetContractBaseRevenueOverTime(ctx context.Context, filter *requests.DashboardFilterRequest, granularity constant.TrendGranularity) ([]responses.RevenueOverTimePoint, error)

	// GetLimitedProductRevenueOverTime returns limited product revenue grouped by time periods
	GetLimitedProductRevenueOverTime(ctx context.Context, filter *requests.DashboardFilterRequest, granularity constant.TrendGranularity) ([]responses.RevenueOverTimePoint, error)

	// GetRefundViolationStats returns system-wide refund and violation statistics
	GetRefundViolationStats(ctx context.Context, filter *requests.DashboardFilterRequest) (*responses.RefundViolationStatsResponse, error)

	// GetAffiliateClicksOverTime returns click counts per contract per period for tiered calculation
	GetAffiliateClicksOverTime(ctx context.Context, filter *requests.DashboardFilterRequest, granularity constant.TrendGranularity) ([]responses.AffiliateClicksPeriod, error)

	// GetLimitedProductRevenueWithSharesOverTime returns limited product revenue with brand/system shares
	GetLimitedProductRevenueWithSharesOverTime(ctx context.Context, filter *requests.DashboardFilterRequest, granularity constant.TrendGranularity) ([]responses.LimitedProductSharePeriod, error)

	// GetTotalRefundsPaid returns total refunds paid during the period
	GetTotalRefundsPaid(ctx context.Context, filter *requests.DashboardFilterRequest) (float64, error)

	// GetDetailedContractRevenueBreakdown returns aggregated revenue breakdown from contract payments
	GetDetailedContractRevenueBreakdown(ctx context.Context, filter *requests.DashboardFilterRequest, granularity constant.TrendGranularity) ([]responses.ContractRevenueBreakdownPoint, float64, error)
}
