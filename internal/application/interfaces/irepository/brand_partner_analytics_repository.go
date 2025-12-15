package irepository

import (
	"context"
	"core-backend/internal/application/dto/dtos"
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
	GetBrandTopBoughtProducts(ctx context.Context, brandUserID uuid.UUID, limit int, startDate, endDate *time.Time) ([]dtos.BrandTopSoldProducts, error)

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
}
