package iservice

import (
	"context"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"

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

	GetBrandTopRatingProduct(ctx context.Context, brandUserID uuid.UUID, req *requests.BrandTopRatingProductRequest) ([]responses.BrandProductRating, error)

	GetBrandTopSoldProduct(ctx context.Context, brandUserID uuid.UUID, req *requests.BrandTopSoldProductRequest) ([]responses.BrandTopSoldProducts, error)
}
