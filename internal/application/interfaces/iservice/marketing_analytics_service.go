package iservice

import (
	"context"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/domain/constant"
)

// MarketingAnalyticsService defines analytics operations for Marketing Staff
type MarketingAnalyticsService interface {
	// GetActiveBrandsCount returns the count of active brands
	GetActiveBrandsCount(ctx context.Context) (int64, error)

	// GetActiveCampaignsCount returns the count of active campaigns
	GetActiveCampaignsCount(ctx context.Context) (int64, error)

	// GetDraftCampaignsCount returns the count of draft campaigns with contracts
	GetDraftCampaignsCount(ctx context.Context) (int64, error)

	// GetGrossContractRevenue returns total revenue from paid contract payments (before refund deductions)
	// Includes PAID + KOL_REFUND_APPROVED payments (full amounts)
	GetGrossContractRevenue(ctx context.Context, filter *requests.DashboardFilterRequest) (float64, error)

	// GetNetContractRevenue returns net revenue after refund deductions
	// Returns: grossRevenue (PAID + KOL_REFUND_APPROVED full amounts), netRevenue (after refund), totalRefunds
	GetNetContractRevenue(ctx context.Context, filter *requests.DashboardFilterRequest) (grossRevenue, netRevenue, totalRefunds float64, err error)

	// GetTopBrandsByRevenue returns top brands by total revenue (contracts + products)
	GetTopBrandsByRevenue(ctx context.Context, filter *requests.DashboardFilterRequest) ([]responses.BrandRevenueResponse, error)

	// GetRevenueByContractType returns revenue breakdown by contract type and standard products
	GetRevenueByContractType(ctx context.Context, filter *requests.DashboardFilterRequest) (*responses.RevenueByTypeResponse, error)

	// GetUpcomingDeadlineCampaigns returns campaigns approaching deadline
	GetUpcomingDeadlineCampaigns(ctx context.Context, filter *requests.UpcomingDeadlineFilter) ([]responses.UpcomingCampaignResponse, error)

	// GetDashboard returns aggregated analytics data for marketing dashboard
	GetDashboard(ctx context.Context, filter *requests.DashboardFilterRequest) (*responses.MarketingDashboardResponse, error)

	// NEW METHODS FOR DASHBOARD REFACTOR

	// GetContractStatusDistribution returns contract counts grouped by status categories
	GetContractStatusDistribution(ctx context.Context, filter *requests.DashboardFilterRequest) (*responses.ContractStatusDistributionResponse, error)

	// GetTaskStatusDistribution returns task counts grouped by status
	GetTaskStatusDistribution(ctx context.Context, filter *requests.DashboardFilterRequest) (*responses.TaskStatusDistributionResponse, error)

	// GetRevenueOverTime returns revenue breakdown by source over time for combo chart
	GetRevenueOverTime(ctx context.Context, filter *requests.DashboardFilterRequest) (*responses.RevenueOverTimeResponse, error)

	// GetRefundViolationStats returns system-wide refund and violation statistics
	GetRefundViolationStats(ctx context.Context, filter *requests.DashboardFilterRequest) (*responses.RefundViolationStatsResponse, error)

	// GetContractRevenueBreakdown returns detailed revenue breakdown for ComposedChart
	// Components: Base Cost, Affiliate Revenue (tiered), Limited Product Brand/System Shares
	GetContractRevenueBreakdown(ctx context.Context, filter *requests.DashboardFilterRequest, granularity constant.TrendGranularity) (*responses.ContractRevenueBreakdownResponse, error)
}
