package service

import (
	"context"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/domain/model"
	"core-backend/pkg/utils"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type AffiliateLinkAnalyticsService struct {
	clickEventRepo    irepository.ClickEventRepository
	kpiMetricsRepo    irepository.GenericRepository[model.KPIMetrics]
	affiliateLinkRepo irepository.AffiliateLinkRepository
	contractRepo      irepository.GenericRepository[model.Contract]
}

func NewAffiliateLinkAnalyticsService(
	clickEventRepo irepository.ClickEventRepository,
	kpiMetricsRepo irepository.GenericRepository[model.KPIMetrics],
	affiliateLinkRepo irepository.AffiliateLinkRepository,
	contractRepo irepository.GenericRepository[model.Contract],
) *AffiliateLinkAnalyticsService {
	return &AffiliateLinkAnalyticsService{
		clickEventRepo:    clickEventRepo,
		kpiMetricsRepo:    kpiMetricsRepo,
		affiliateLinkRepo: affiliateLinkRepo,
		contractRepo:      contractRepo,
	}
}

// GetMetricsByContract retrieves analytics metrics for a specific contract
func (s *AffiliateLinkAnalyticsService) GetMetricsByContract(
	ctx context.Context,
	req *requests.ContractMetricsRequest,
) (*responses.ContractMetricsResponse, error) {
	// Set default time range if not provided (last 30 days)
	startDate, endDate := s.getDefaultTimeRange(req.StartDate, req.EndDate)

	// Get contract details
	contract, err := s.contractRepo.GetByID(ctx, req.ContractID, []string{"Brand"})
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("contract not found")
		}
		return nil, fmt.Errorf("failed to get contract: %w", err)
	}

	// Get all affiliate links for this contract
	links, _, err := s.affiliateLinkRepo.GetAll(ctx,
		func(db *gorm.DB) *gorm.DB {
			return db.Where("contract_id = ?", req.ContractID)
		},
		nil, 0, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to get affiliate links: %w", err)
	}

	if len(links) == 0 {
		// Return empty metrics if no links exist
		return &responses.ContractMetricsResponse{
			ContractID:   contract.ID,
			ContractName: s.getStringValue(contract.ContractNumber),
			BrandName:    contract.Brand.Name,
			TotalClicks:  0,
			UniqueUsers:  0,
			CTR:          0,
			TopChannels:  []responses.ChannelMetricItem{},
			TopLinks:     []responses.AffiliateLinkMetric{},
			Period: responses.PeriodInfo{
				StartDate: startDate,
				EndDate:   endDate,
			},
		}, nil
	}

	// Extract link IDs
	linkIDs := make([]uuid.UUID, len(links))
	for i, link := range links {
		linkIDs[i] = link.ID
	}

	// Get click metrics for all links in this contract
	totalClicks, uniqueUsers, err := s.getClickMetrics(ctx, linkIDs, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get click metrics: %w", err)
	}

	// Get top channels for this contract
	topChannels, err := s.getTopChannelsByLinks(ctx, linkIDs, startDate, endDate)
	if err != nil {
		zap.L().Error("Failed to get top channels", zap.Error(err))
		topChannels = []responses.ChannelMetricItem{} // Default to empty
	}

	// Get top performing links for this contract
	topLinks, err := s.getTopLinksByIDs(ctx, linkIDs, startDate, endDate, 5)
	if err != nil {
		zap.L().Error("Failed to get top links", zap.Error(err))
		topLinks = []responses.AffiliateLinkMetric{} // Default to empty
	}

	return &responses.ContractMetricsResponse{
		ContractID:   contract.ID,
		ContractName: s.getStringValue(contract.ContractNumber),
		BrandName:    contract.Brand.Name,
		TotalClicks:  totalClicks,
		UniqueUsers:  uniqueUsers,
		CTR:          s.calculateCTR(totalClicks, uniqueUsers),
		TopChannels:  topChannels,
		TopLinks:     topLinks,
		Period: responses.PeriodInfo{
			StartDate: startDate,
			EndDate:   endDate,
		},
	}, nil
}

// GetMetricsByChannel retrieves analytics metrics grouped by channel
func (s *AffiliateLinkAnalyticsService) GetMetricsByChannel(
	ctx context.Context,
	req *requests.ChannelMetricsRequest,
) (*responses.ChannelMetricsResponse, error) {
	startDate, endDate := s.getDefaultTimeRange(req.StartDate, req.EndDate)

	// Get all active affiliate links
	links, _, err := s.affiliateLinkRepo.GetActiveLinks(ctx, 0, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to get active links: %w", err)
	}

	// Group links by channel
	channelLinks := make(map[string][]uuid.UUID)
	for _, link := range links {
		channelStr := s.getChannelString(link.Channel)
		channelLinks[channelStr] = append(channelLinks[channelStr], link.ID)
	}

	// Get metrics for each channel
	var channels []responses.ChannelMetricItem
	var totalClicksAllChannels int64

	for channel, linkIDs := range channelLinks {
		clicks, users, err := s.getClickMetrics(ctx, linkIDs, startDate, endDate)
		if err != nil {
			zap.L().Warn("Failed to get metrics for channel", zap.String("channel", channel), zap.Error(err))
			continue
		}

		channels = append(channels, responses.ChannelMetricItem{
			Channel:      channel,
			TotalClicks:  clicks,
			UniqueUsers:  users,
			CTR:          s.calculateCTR(clicks, users),
			LinkCount:    len(linkIDs),
			PercentTotal: 0, // Will be calculated after we have total
		})

		totalClicksAllChannels += clicks
	}

	// Calculate percentages
	for i := range channels {
		if totalClicksAllChannels > 0 {
			channels[i].PercentTotal = float64(channels[i].TotalClicks) / float64(totalClicksAllChannels) * 100
		}
	}

	return &responses.ChannelMetricsResponse{
		Channels: channels,
		Period: responses.PeriodInfo{
			StartDate: startDate,
			EndDate:   endDate,
		},
	}, nil
}

// GetTimeSeriesData retrieves time-series data for a specific affiliate link
func (s *AffiliateLinkAnalyticsService) GetTimeSeriesData(
	ctx context.Context,
	req *requests.TimeSeriesRequest,
) (*responses.TimeSeriesDataResponse, error) {
	startDate, endDate := s.getDefaultTimeRange(req.StartDate, req.EndDate)

	// Get affiliate link details
	link, err := s.affiliateLinkRepo.GetByID(ctx, req.AffiliateLinkID, nil)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("affiliate link not found")
		}
		return nil, fmt.Errorf("failed to get affiliate link: %w", err)
	}

	// Determine granularity (default to DAY)
	granularity := req.Granularity
	if granularity == "" {
		granularity = "DAY"
	}

	// Get time-bucketed data from click_events
	dataPoints, err := s.getTimeSeriesPoints(ctx, req.AffiliateLinkID, startDate, endDate, granularity)
	if err != nil {
		return nil, fmt.Errorf("failed to get time series data: %w", err)
	}

	return &responses.TimeSeriesDataResponse{
		AffiliateLinkID: link.ID,
		LinkName:        link.Hash, // Use hash as name since Name field doesn't exist
		TrackingURL:     link.TrackingURL,
		Channel:         s.getChannelString(link.Channel),
		DataPoints:      dataPoints,
		Granularity:     granularity,
		Period: responses.PeriodInfo{
			StartDate: startDate,
			EndDate:   endDate,
		},
	}, nil
}

// GetTopPerformers retrieves top performing affiliate links
func (s *AffiliateLinkAnalyticsService) GetTopPerformers(
	ctx context.Context,
	req *requests.TopPerformersRequest,
) (*responses.TopPerformerResponse, error) {
	startDate, endDate := s.getDefaultTimeRange(req.StartDate, req.EndDate)

	// Default sort and limit
	sortBy := req.SortBy
	if sortBy == "" {
		sortBy = "CLICKS"
	}
	limit := req.Limit
	if limit <= 0 {
		limit = 10
	}

	// Get all active links
	links, _, err := s.affiliateLinkRepo.GetActiveLinks(ctx, 0, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to get active links: %w", err)
	}

	// Get metrics for each link
	linkMetrics := make([]responses.AffiliateLinkMetric, 0, len(links))
	for _, link := range links {
		clicks, users, err := s.getClickMetrics(ctx, []uuid.UUID{link.ID}, startDate, endDate)
		if err != nil {
			zap.L().Warn("Failed to get metrics for link", zap.String("link_id", link.ID.String()), zap.Error(err))
			continue
		}

		linkMetrics = append(linkMetrics, responses.AffiliateLinkMetric{
			AffiliateLinkID: link.ID,
			LinkName:        link.Hash, // Use hash as identifier
			ShortHash:       link.Hash,
			TrackingURL:     link.TrackingURL,
			Channel:         s.getChannelString(link.Channel),
			TotalClicks:     clicks,
			UniqueUsers:     users,
			CTR:             s.calculateCTR(clicks, users),
		})
	}

	// Sort based on criteria
	s.sortLinkMetrics(linkMetrics, sortBy)

	// Limit results
	if len(linkMetrics) > limit {
		linkMetrics = linkMetrics[:limit]
	}

	// Add rank
	for i := range linkMetrics {
		linkMetrics[i].Rank = i + 1
	}

	return &responses.TopPerformerResponse{
		TopLinks: linkMetrics,
		SortBy:   sortBy,
		Period: responses.PeriodInfo{
			StartDate: startDate,
			EndDate:   endDate,
		},
	}, nil
}

// GetDashboardMetrics retrieves overall dashboard metrics with parallel aggregation
func (s *AffiliateLinkAnalyticsService) GetDashboardMetrics(
	ctx context.Context,
	req *requests.DashboardRequest,
) (*responses.DashboardMetricsResponse, error) {
	startDate, endDate := s.getDefaultTimeRange(req.StartDate, req.EndDate)

	// Use parallel execution for independent queries
	type dashboardData struct {
		overview       *responses.OverviewMetrics
		topContracts   []responses.ContractAnalyticsSummary
		topChannels    []responses.ChannelMetricItem
		recentActivity []responses.RecentActivityItem
		trendData      []responses.TrendDataPoint
	}

	var result dashboardData

	// Execute queries in parallel using RunParallel
	err := utils.RunParallel(ctx, 5,
		func(ctx context.Context) error {
			var err error
			result.overview, err = s.calculateOverview(ctx, startDate, endDate)
			return err
		},
		func(ctx context.Context) error {
			var err error
			result.topContracts, err = s.getTopContracts(ctx, startDate, endDate, 5)
			if err != nil {
				zap.L().Warn("Failed to get top contracts", zap.Error(err))
				result.topContracts = []responses.ContractAnalyticsSummary{}
				return nil // Don't fail entire request
			}
			return nil
		},
		func(ctx context.Context) error {
			var err error
			result.topChannels, err = s.getTopChannels(ctx, startDate, endDate, 5)
			if err != nil {
				zap.L().Warn("Failed to get top channels", zap.Error(err))
				result.topChannels = []responses.ChannelMetricItem{}
				return nil // Don't fail entire request
			}
			return nil
		},
		func(ctx context.Context) error {
			var err error
			result.recentActivity, err = s.getRecentActivity(ctx, 10)
			if err != nil {
				zap.L().Warn("Failed to get recent activity", zap.Error(err))
				result.recentActivity = []responses.RecentActivityItem{}
				return nil // Don't fail entire request
			}
			return nil
		},
		func(ctx context.Context) error {
			var err error
			result.trendData, err = s.getTrendData(ctx, startDate, endDate)
			if err != nil {
				zap.L().Warn("Failed to get trend data", zap.Error(err))
				result.trendData = []responses.TrendDataPoint{}
				return nil // Don't fail entire request
			}
			return nil
		},
	)

	// Check for critical errors (only overview is critical)
	if err != nil {
		return nil, fmt.Errorf("failed to aggregate dashboard metrics: %w", err)
	}

	return &responses.DashboardMetricsResponse{
		Overview:       *result.overview,
		TopContracts:   result.topContracts,
		TopChannels:    result.topChannels,
		RecentActivity: result.recentActivity,
		TrendData:      result.trendData,
		Period: responses.PeriodInfo{
			StartDate: startDate,
			EndDate:   endDate,
		},
	}, nil
}

// ValidateContractAccess validates that the user has access to the contract's analytics
func (s *AffiliateLinkAnalyticsService) ValidateContractAccess(
	ctx context.Context,
	userID uuid.UUID,
	contractID uuid.UUID,
) error {
	// Get contract with brand relationship
	contract, err := s.contractRepo.GetByID(ctx, contractID, []string{"Brand"})
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("contract not found")
		}
		return fmt.Errorf("failed to get contract: %w", err)
	}

	// For now, allow all authenticated users to access analytics
	// TODO: Implement proper RBAC - check if user is BRAND_PARTNER and owns the brand
	// This would require checking user's role and brand ownership
	_ = contract // Use contract to avoid unused variable error
	_ = userID   // Use userID to avoid unused variable error

	return nil
}

// Helper methods

func (s *AffiliateLinkAnalyticsService) getStringValue(ptr *string) string {
	if ptr == nil {
		return ""
	}
	return *ptr
}

func (s *AffiliateLinkAnalyticsService) getChannelString(channel *model.Channel) string {
	if channel == nil {
		return ""
	}
	return channel.Name
}

func (s *AffiliateLinkAnalyticsService) getDefaultTimeRange(startDate, endDate *time.Time) (time.Time, time.Time) {
	now := time.Now()
	if endDate == nil {
		endDate = &now
	}
	if startDate == nil {
		thirtyDaysAgo := now.AddDate(0, 0, -30)
		startDate = &thirtyDaysAgo
	}
	return *startDate, *endDate
}

func (s *AffiliateLinkAnalyticsService) calculateCTR(clicks, users int64) float64 {
	if users == 0 {
		return 0
	}
	return float64(clicks) / float64(users)
}

func (s *AffiliateLinkAnalyticsService) getClickMetrics(
	ctx context.Context,
	linkIDs []uuid.UUID,
	startDate, endDate time.Time,
) (totalClicks int64, uniqueUsers int64, err error) {
	// This would use a custom repository method to query click_events
	// For now, return placeholder implementation
	// TODO: Implement actual query using TimescaleDB
	return 0, 0, nil
}

func (s *AffiliateLinkAnalyticsService) getTopChannelsByLinks(
	ctx context.Context,
	linkIDs []uuid.UUID,
	startDate, endDate time.Time,
) ([]responses.ChannelMetricItem, error) {
	// TODO: Implement channel aggregation from click_events
	return []responses.ChannelMetricItem{}, nil
}

func (s *AffiliateLinkAnalyticsService) getTopLinksByIDs(
	ctx context.Context,
	linkIDs []uuid.UUID,
	startDate, endDate time.Time,
	limit int,
) ([]responses.AffiliateLinkMetric, error) {
	// TODO: Implement top links query
	return []responses.AffiliateLinkMetric{}, nil
}

func (s *AffiliateLinkAnalyticsService) getTimeSeriesPoints(
	ctx context.Context,
	linkID uuid.UUID,
	startDate, endDate time.Time,
	granularity string,
) ([]responses.TimeSeriesPoint, error) {
	// TODO: Implement time_bucket query using TimescaleDB
	return []responses.TimeSeriesPoint{}, nil
}

func (s *AffiliateLinkAnalyticsService) sortLinkMetrics(metrics []responses.AffiliateLinkMetric, sortBy string) {
	// TODO: Implement sorting logic based on sortBy (CLICKS, CTR, ENGAGEMENT)
}

func (s *AffiliateLinkAnalyticsService) calculateOverview(
	ctx context.Context,
	startDate, endDate time.Time,
) (*responses.OverviewMetrics, error) {
	// Get all active links
	links, _, err := s.affiliateLinkRepo.GetActiveLinks(ctx, 0, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to get active links: %w", err)
	}

	// Get all contracts with affiliate links
	contracts, _, err := s.contractRepo.GetAll(ctx,
		func(db *gorm.DB) *gorm.DB {
			return db.Where("id IN (SELECT DISTINCT contract_id FROM affiliate_links WHERE deleted_at IS NULL)")
		},
		nil, 0, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to get contracts: %w", err)
	}

	// TODO: Calculate actual metrics from click_events
	// For now, return structure with placeholder values
	return &responses.OverviewMetrics{
		TotalClicks:     0,
		UniqueUsers:     0,
		TotalLinks:      len(links),
		ActiveContracts: len(contracts),
		AverageCTR:      0,
		ClickGrowth:     0,
		UserGrowth:      0,
	}, nil
}

func (s *AffiliateLinkAnalyticsService) getTopContracts(
	ctx context.Context,
	startDate, endDate time.Time,
	limit int,
) ([]responses.ContractAnalyticsSummary, error) {
	// TODO: Implement top contracts query
	return []responses.ContractAnalyticsSummary{}, nil
}

func (s *AffiliateLinkAnalyticsService) getTopChannels(
	ctx context.Context,
	startDate, endDate time.Time,
	limit int,
) ([]responses.ChannelMetricItem, error) {
	// TODO: Implement top channels aggregation
	return []responses.ChannelMetricItem{}, nil
}

func (s *AffiliateLinkAnalyticsService) getRecentActivity(
	ctx context.Context,
	limit int,
) ([]responses.RecentActivityItem, error) {
	// TODO: Implement recent activity query from click_events
	return []responses.RecentActivityItem{}, nil
}

func (s *AffiliateLinkAnalyticsService) getTrendData(
	ctx context.Context,
	startDate, endDate time.Time,
) ([]responses.TrendDataPoint, error) {
	// TODO: Implement trend data using time_bucket for daily aggregation
	return []responses.TrendDataPoint{}, nil
}
