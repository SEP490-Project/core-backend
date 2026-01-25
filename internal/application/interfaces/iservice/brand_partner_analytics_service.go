package iservice

import (
	"context"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/domain/constant"

	"github.com/google/uuid"
)

// BrandPartnerAnalyticsService defines the interface for Brand Partner analytics operations
type BrandPartnerAnalyticsService interface {
	// GetDashboard returns the complete Brand Partner dashboard for a specific brand
	GetDashboard(ctx context.Context, brandUserID uuid.UUID, req *requests.BrandPartnerDashboardRequest) (*responses.BrandPartnerDashboardResponse, error)

	// GetTopProducts returns top products by revenue for a brand
	GetTopProducts(ctx context.Context, brandUserID uuid.UUID, req *requests.BrandTopProductsRequest) ([]responses.BrandProductMetric, error)

	// GetCampaignMetrics returns campaign performance metrics for a brand
	GetCampaignMetrics(ctx context.Context, brandUserID uuid.UUID, req *requests.BrandCampaignsRequest) ([]responses.BrandCampaignMetric, error)

	// GetContentMetrics returns content performance metrics for a brand
	GetContentMetrics(ctx context.Context, brandUserID uuid.UUID, req *requests.BrandContentMetricsRequest) (*responses.BrandContentMetric, error)

	// GetRevenueTrend returns revenue time-series for a brand
	GetRevenueTrend(ctx context.Context, brandUserID uuid.UUID, req *requests.BrandRevenueTrendRequest) ([]responses.BrandRevenueTrendPoint, error)

	// GetAffiliateMetrics returns affiliate link performance for a brand
	GetAffiliateMetrics(ctx context.Context, brandUserID uuid.UUID, req *requests.BrandAffiliateMetricsRequest) (*responses.BrandAffiliateMetric, error)

	// GetContractDetails returns contract details for a brand
	GetContractDetails(ctx context.Context, brandUserID uuid.UUID, req *requests.BrandContractsRequest) ([]responses.BrandContractDetail, error)

	// GetBrandTopRatingProduct return top rating products for a brand
	GetBrandTopRatingProduct(ctx context.Context, brandUserID uuid.UUID, req *requests.BrandTopRatingProductRequest) ([]responses.BrandProductRating, error)

	// GetBrandTopSoldProduct return top sold products for a brand
	GetBrandTopSoldProduct(ctx context.Context, brandUserID uuid.UUID, req *requests.BrandTopSoldProductRequest) ([]responses.BrandTopSoldProducts, error)

	// GetBrandContractStatusDistribution returns contract status distribution for a brand
	GetBrandContractStatusDistribution(ctx context.Context, brandUserID uuid.UUID, filter *requests.DashboardFilterRequest) (*responses.ContractStatusDistributionResponse, error)

	// GetBrandTaskStatusDistribution returns task status distribution for a brand
	GetBrandTaskStatusDistribution(ctx context.Context, brandUserID uuid.UUID, filter *requests.DashboardFilterRequest) (*responses.TaskStatusDistributionResponse, error)

	// GetBrandRevenueOverTime returns revenue trend over time for a brand (Limited Products only)
	GetBrandRevenueOverTime(ctx context.Context, brandUserID uuid.UUID, filter *requests.DashboardFilterRequest, granularity constant.TrendGranularity) (*responses.BrandRevenueOverTimeResponse, error)

	// GetBrandRefundViolationStats returns refund and contract violation stats for a brand
	GetBrandRefundViolationStats(ctx context.Context, brandUserID uuid.UUID, filter *requests.DashboardFilterRequest) (*responses.RefundViolationStatsResponse, error)

	// GetBrandGrossIncome returns brand's gross income (limited product revenue × brand share percentage)
	GetBrandGrossIncome(ctx context.Context, brandUserID uuid.UUID, filter *requests.DashboardFilterRequest) (*responses.BrandIncomeResponse, error)

	// GetBrandNetIncome returns brand's net income (gross income - paid contract payments)
	GetBrandNetIncome(ctx context.Context, brandUserID uuid.UUID, filter *requests.DashboardFilterRequest) (*responses.BrandNetIncomeResponse, error)
}
