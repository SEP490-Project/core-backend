package iservice

import (
	"context"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"

	"github.com/google/uuid"
)

// AffiliateLinkAnalyticsService defines methods for affiliate link analytics and reporting
type AffiliateLinkAnalyticsService interface {
	// GetMetricsByContract retrieves analytics metrics for a specific contract
	GetMetricsByContract(ctx context.Context, req *requests.ContractMetricsRequest) (*responses.ContractMetricsResponse, error)

	// GetMetricsByChannel retrieves analytics metrics grouped by channel with comparison
	GetMetricsByChannel(ctx context.Context, req *requests.ChannelMetricsRequest) (*responses.ChannelMetricsResponse, error)

	// GetTimeSeriesData retrieves time-series data for a specific affiliate link
	GetTimeSeriesData(ctx context.Context, req *requests.TimeSeriesRequest) (*responses.TimeSeriesDataResponse, error)

	// GetTopPerformers retrieves top performing affiliate links based on sorting criteria
	GetTopPerformers(ctx context.Context, req *requests.TopPerformersRequest) (*responses.TopPerformerResponse, error)

	// GetDashboardMetrics retrieves overall dashboard metrics with parallel aggregation
	GetDashboardMetrics(ctx context.Context, req *requests.DashboardRequest) (*responses.DashboardMetricsResponse, error)

	// ValidateContractAccess validates that the user has access to the contract's analytics
	// Returns error if user doesn't have permission (for BRAND_PARTNER role)
	ValidateContractAccess(ctx context.Context, userID uuid.UUID, contractID uuid.UUID) error
}
