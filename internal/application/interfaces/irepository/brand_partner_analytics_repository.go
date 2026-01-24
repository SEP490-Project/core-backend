package irepository

import (
	"context"
	"core-backend/internal/application/dto/dtos"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/domain/constant"
	"time"

	"github.com/google/uuid"
)

// BrandPartnerAnalyticsRepository defines the interface for brand partner analytics data access
type BrandPartnerAnalyticsRepository interface {
	// Brand Overview
	GetBrandContractCount(ctx context.Context, brandUserID uuid.UUID, status *string) (int64, error)
	GetBrandCampaignCount(ctx context.Context, brandUserID uuid.UUID, status *string) (int64, error)
	GetBrandProductCount(ctx context.Context, brandUserID uuid.UUID, status *string) (int64, error)
	GetBrandOrderCount(ctx context.Context, brandUserID uuid.UUID, status *string, startDate, endDate *time.Time) (int64, error)
	GetBrandTotalRevenue(ctx context.Context, brandUserID uuid.UUID, startDate, endDate *time.Time) (float64, error)
	GetBrandTotalPayments(ctx context.Context, brandUserID uuid.UUID, startDate, endDate *time.Time) (float64, error)
	GetBrandPendingPayments(ctx context.Context, brandUserID uuid.UUID) (float64, error)

	// Top Products
	GetBrandTopProducts(ctx context.Context, brandUserID uuid.UUID, limit int, startDate, endDate *time.Time) ([]dtos.BrandProductMetrics, error)
	GetBrandTopSoldProduct(ctx context.Context, brandUserID uuid.UUID, limit int, startDate, endDate *time.Time) ([]dtos.BrandTopSoldProducts, error)

	// Campaign Metrics
	GetBrandCampaignMetrics(ctx context.Context, brandUserID uuid.UUID, limit int, startDate, endDate *time.Time) ([]dtos.BrandCampaignMetrics, error)

	// Content Metrics
	GetBrandContentMetrics(ctx context.Context, brandUserID uuid.UUID, startDate, endDate *time.Time) (*dtos.BrandContentMetrics, error)

	// Revenue Trend
	GetBrandRevenueTrend(ctx context.Context, brandUserID uuid.UUID, granularity string, startDate, endDate *time.Time) ([]dtos.BrandRevenueTrendResult, error)

	// Affiliate Metrics
	GetBrandAffiliateMetrics(ctx context.Context, brandUserID uuid.UUID, startDate, endDate *time.Time) (*dtos.BrandAffiliateMetrics, error)

	// Contract Details
	GetBrandContractDetails(ctx context.Context, brandUserID uuid.UUID, limit int) ([]dtos.BrandContractDetails, error)

	// Rating
	GetBrandTopRatingProduct(ctx context.Context, brandUserID uuid.UUID, limit int, startDate, endDate *time.Time) ([]dtos.BrandProductRating, error)

	// NEW METHODS FOR DASHBOARD REFACTOR

	// GetBrandContractStatusDistribution returns contract counts grouped by status for a brand
	GetBrandContractStatusDistribution(ctx context.Context, brandUserID uuid.UUID, filter *requests.DashboardFilterRequest) (*responses.ContractStatusDistributionResponse, error)

	// GetBrandTaskStatusDistribution returns task counts grouped by status for a brand's campaigns
	GetBrandTaskStatusDistribution(ctx context.Context, brandUserID uuid.UUID, filter *requests.DashboardFilterRequest) (*responses.TaskStatusDistributionResponse, error)

	// GetBrandLimitedProductRevenueOverTime returns brand's LIMITED product revenue over time
	GetBrandLimitedProductRevenueOverTime(ctx context.Context, brandUserID uuid.UUID, filter *requests.DashboardFilterRequest, granularity constant.TrendGranularity) ([]responses.BrandRevenueOverTimePoint, error)

	// GetBrandRefundViolationStats returns refund and violation statistics for a brand
	GetBrandRefundViolationStats(ctx context.Context, brandUserID uuid.UUID, filter *requests.DashboardFilterRequest) (*responses.RefundViolationStatsResponse, error)

	// GetBrandGrossIncome returns brand's gross income (limited product revenue × brand share percentage)
	GetBrandGrossIncome(ctx context.Context, brandUserID uuid.UUID, filter *requests.DashboardFilterRequest) (*responses.BrandIncomeResponse, error)

	// GetBrandNetIncome returns brand's net income (gross income - paid contract payments)
	GetBrandNetIncome(ctx context.Context, brandUserID uuid.UUID, filter *requests.DashboardFilterRequest) (*responses.BrandNetIncomeResponse, error)
}
