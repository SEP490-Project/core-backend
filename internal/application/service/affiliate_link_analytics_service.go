package service

import (
	"context"
	"core-backend/internal/application/dto/dtos"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/application/interfaces/iservice"
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
	"core-backend/pkg/utils"
	"errors"
	"fmt"
	"sort"
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
) iservice.AffiliateLinkAnalyticsService {
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
			return db.Where("contract_id = ? OR metadata->>'contract_id' = ?", req.ContractID, req.ContractID.String())
		},
		[]string{"Channel"}, 0, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to get affiliate links: %w", err)
	}

	if len(links) == 0 {
		return &responses.ContractMetricsResponse{
			ContractID:   contract.ID,
			ContractName: s.getStringValue(contract.ContractNumber),
			BrandName:    contract.Brand.Name,
			Period: responses.PeriodInfo{
				StartDate: startDate,
				EndDate:   endDate,
			},
			TopChannels: []responses.ChannelMetricItem{},
			TopLinks:    []responses.AffiliateLinkMetric{},
		}, nil
	}

	// Get clicks for the contract
	clicks, err := s.clickEventRepo.GetClicksByContract(ctx, req.ContractID, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get clicks: %w", err)
	}

	totalClicks := int64(len(clicks))
	uniqueUsers := s.countUniqueUsers(clicks)
	ctr := s.calculateCTR(totalClicks, uniqueUsers)

	// Aggregate by channel and link
	channelStats := make(map[string]*responses.ChannelMetricItem)
	linkStats := make(map[uuid.UUID]*responses.AffiliateLinkMetric)

	// Initialize link stats
	for _, link := range links {
		channelName := "UNKNOWN"
		if link.Channel != nil {
			channelName = link.Channel.Name
		}
		linkStats[link.ID] = &responses.AffiliateLinkMetric{
			AffiliateLinkID: link.ID,
			LinkName:        link.Hash,
			ShortHash:       link.Hash,
			TrackingURL:     link.TrackingURL,
			Channel:         channelName,
		}
	}

	// Process clicks
	linkUniqueUsers := make(map[uuid.UUID]map[string]struct{})
	channelUniqueUsers := make(map[string]map[string]struct{})

	for _, click := range clicks {
		// Link Stats
		if stat, ok := linkStats[click.AffiliateLinkID]; ok {
			stat.TotalClicks++

			userKey := s.getUserKey(click)

			// Link Unique
			if _, ok := linkUniqueUsers[click.AffiliateLinkID]; !ok {
				linkUniqueUsers[click.AffiliateLinkID] = make(map[string]struct{})
			}
			linkUniqueUsers[click.AffiliateLinkID][userKey] = struct{}{}

			// Channel Stats
			channelName := stat.Channel
			if _, exists := channelStats[channelName]; !exists {
				channelStats[channelName] = &responses.ChannelMetricItem{
					Channel: channelName,
				}
			}
			channelStats[channelName].TotalClicks++

			// Channel Unique
			if _, ok := channelUniqueUsers[channelName]; !ok {
				channelUniqueUsers[channelName] = make(map[string]struct{})
			}
			channelUniqueUsers[channelName][userKey] = struct{}{}
		}
	}

	// Finalize Link Stats
	var topLinks []responses.AffiliateLinkMetric
	for id, stat := range linkStats {
		stat.UniqueUsers = int64(len(linkUniqueUsers[id]))
		stat.CTR = s.calculateCTR(stat.TotalClicks, stat.UniqueUsers)
		if stat.TotalClicks > 0 {
			topLinks = append(topLinks, *stat)
		}
	}
	// Sort links by clicks
	sort.Slice(topLinks, func(i, j int) bool {
		return topLinks[i].TotalClicks > topLinks[j].TotalClicks
	})
	if len(topLinks) > 5 {
		topLinks = topLinks[:5]
	}

	// Finalize Channel Stats
	var topChannels []responses.ChannelMetricItem
	for name, stat := range channelStats {
		stat.UniqueUsers = int64(len(channelUniqueUsers[name]))
		stat.CTR = s.calculateCTR(stat.TotalClicks, stat.UniqueUsers)
		if totalClicks > 0 {
			stat.PercentTotal = float64(stat.TotalClicks) / float64(totalClicks) * 100
		}
		topChannels = append(topChannels, *stat)
	}
	// Sort channels by clicks
	sort.Slice(topChannels, func(i, j int) bool {
		return topChannels[i].TotalClicks > topChannels[j].TotalClicks
	})

	return &responses.ContractMetricsResponse{
		ContractID:   contract.ID,
		ContractName: s.getStringValue(contract.ContractNumber),
		BrandName:    contract.Brand.Name,
		TotalClicks:  totalClicks,
		UniqueUsers:  uniqueUsers,
		CTR:          ctr,
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

	// Use GetTopChannels from ClickEventRepo
	topChannels, err := s.clickEventRepo.GetTopChannels(ctx, startDate, endDate, 100)
	if err != nil {
		return nil, fmt.Errorf("failed to get channel metrics: %w", err)
	}

	var channels []responses.ChannelMetricItem
	var totalClicksAllChannels int64

	for _, p := range topChannels {
		channels = append(channels, responses.ChannelMetricItem{
			Channel:      p.ChannelName,
			TotalClicks:  p.TotalClicks,
			UniqueUsers:  p.UniqueUsers,
			CTR:          s.calculateCTR(p.TotalClicks, p.UniqueUsers),
			PercentTotal: 0,
		})
		totalClicksAllChannels += p.TotalClicks
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
	link, err := s.affiliateLinkRepo.GetByID(ctx, req.AffiliateLinkID, []string{"Channel"})
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
		LinkName:        link.Hash,
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

	// Use optimized query from ClickEventRepo
	perfs, err := s.clickEventRepo.GetTopPerformingLinks(ctx, startDate, endDate, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get top performers: %w", err)
	}

	var topLinks []responses.AffiliateLinkMetric
	for i, p := range perfs {
		topLinks = append(topLinks, responses.AffiliateLinkMetric{
			AffiliateLinkID: p.AffiliateLinkID,
			LinkName:        p.Hash,
			ShortHash:       p.Hash,
			TrackingURL:     "", // Not returned by aggregation query
			Channel:         p.ChannelName,
			TotalClicks:     p.TotalClicks,
			UniqueUsers:     p.UniqueUsers,
			CTR:             s.calculateCTR(p.TotalClicks, p.UniqueUsers),
			Rank:            i + 1,
		})
	}

	// If sorting by other than CLICKS (which is default in repo), we might need to resort
	if sortBy != "CLICKS" {
		s.sortLinkMetrics(topLinks, sortBy)
		// Re-assign ranks
		for i := range topLinks {
			topLinks[i].Rank = i + 1
		}
	}

	return &responses.TopPerformerResponse{
		TopLinks: topLinks,
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
	_ = contract
	_ = userID

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

func (s *AffiliateLinkAnalyticsService) countUniqueUsers(clicks []model.ClickEvent) int64 {
	unique := make(map[string]struct{})
	for _, c := range clicks {
		unique[s.getUserKey(c)] = struct{}{}
	}
	return int64(len(unique))
}

func (s *AffiliateLinkAnalyticsService) getUserKey(c model.ClickEvent) string {
	if c.UserID != nil {
		return c.UserID.String()
	}
	if c.IPAddress != nil {
		return *c.IPAddress
	}
	return "unknown"
}

func (s *AffiliateLinkAnalyticsService) getTimeSeriesPoints(
	ctx context.Context,
	linkID uuid.UUID,
	startDate, endDate time.Time,
	granularity string,
) ([]responses.TimeSeriesPoint, error) {
	var points []responses.TimeSeriesPoint

	if granularity == "HOUR" {
		stats, err := s.clickEventRepo.GetHourlyStats(ctx, linkID, startDate, endDate)
		if err != nil {
			return nil, err
		}
		for _, s := range stats {
			points = append(points, responses.TimeSeriesPoint{
				Timestamp:   s.Hour,
				Clicks:      s.TotalClicks,
				UniqueUsers: s.UniqueUsers,
			})
		}
	} else {
		// Default to DAY
		stats, err := s.clickEventRepo.GetDailyStats(ctx, linkID, startDate, endDate)
		if err != nil {
			return nil, err
		}
		for _, s := range stats {
			points = append(points, responses.TimeSeriesPoint{
				Timestamp:   s.Date,
				Clicks:      s.TotalClicks,
				UniqueUsers: s.UniqueUsers,
			})
		}
	}

	return points, nil
}

func (s *AffiliateLinkAnalyticsService) sortLinkMetrics(metrics []responses.AffiliateLinkMetric, sortBy string) {
	sort.Slice(metrics, func(i, j int) bool {
		switch sortBy {
		case "CTR":
			return metrics[i].CTR > metrics[j].CTR
		case "ENGAGEMENT":
			return metrics[i].UniqueUsers > metrics[j].UniqueUsers
		default: // CLICKS
			return metrics[i].TotalClicks > metrics[j].TotalClicks
		}
	})
}

func (s *AffiliateLinkAnalyticsService) calculateOverview(
	ctx context.Context,
	startDate, endDate time.Time,
) (*responses.OverviewMetrics, error) {
	// Get global click stats
	stats, err := s.clickEventRepo.GetGlobalOverview(ctx, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get global overview: %w", err)
	}

	// Get all active links count
	totalLinks, err := s.affiliateLinkRepo.Count(ctx, func(db *gorm.DB) *gorm.DB {
		return db.Where("status = ?", enum.AffiliateLinkStatusActive)
	})
	if err != nil {
		return nil, fmt.Errorf("failed to count links: %w", err)
	}

	// Get active contracts count
	activeContracts, err := s.contractRepo.Count(ctx, func(db *gorm.DB) *gorm.DB {
		return db.Where("status = ?", enum.ContractStatusActive)
	})
	if err != nil {
		return nil, fmt.Errorf("failed to count contracts: %w", err)
	}

	// Calculate growth
	duration := endDate.Sub(startDate)
	prevEndDate := startDate
	prevStartDate := startDate.Add(-duration)

	prevStats, err := s.clickEventRepo.GetGlobalOverview(ctx, prevStartDate, prevEndDate)
	if err != nil {
		zap.L().Warn("Failed to get previous period stats for growth calculation", zap.Error(err))
		prevStats = dtos.GlobalClickStats{}
	}

	clickGrowth := s.calculateGrowth(float64(stats.TotalClicks), float64(prevStats.TotalClicks))
	userGrowth := s.calculateGrowth(float64(stats.UniqueUsers), float64(prevStats.UniqueUsers))

	return &responses.OverviewMetrics{
		TotalClicks:     stats.TotalClicks,
		UniqueUsers:     stats.UniqueUsers,
		TotalLinks:      int(totalLinks),
		ActiveContracts: int(activeContracts),
		AverageCTR:      s.calculateCTR(stats.TotalClicks, stats.UniqueUsers),
		ClickGrowth:     clickGrowth,
		UserGrowth:      userGrowth,
	}, nil
}

func (s *AffiliateLinkAnalyticsService) getTopContracts(
	ctx context.Context,
	startDate, endDate time.Time,
	limit int,
) ([]responses.ContractAnalyticsSummary, error) {
	perfs, err := s.clickEventRepo.GetTopContracts(ctx, startDate, endDate, limit)
	if err != nil {
		return nil, err
	}

	var summaries []responses.ContractAnalyticsSummary
	for _, p := range perfs {
		summaries = append(summaries, responses.ContractAnalyticsSummary{
			ContractID:   p.ContractID,
			ContractName: p.ContractName,
			TotalClicks:  p.TotalClicks,
			UniqueUsers:  p.UniqueUsers,
			CTR:          s.calculateCTR(p.TotalClicks, p.UniqueUsers),
		})
	}
	return summaries, nil
}

func (s *AffiliateLinkAnalyticsService) getTopChannels(
	ctx context.Context,
	startDate, endDate time.Time,
	limit int,
) ([]responses.ChannelMetricItem, error) {
	perfs, err := s.clickEventRepo.GetTopChannels(ctx, startDate, endDate, limit)
	if err != nil {
		return nil, err
	}

	var channels []responses.ChannelMetricItem
	for _, p := range perfs {
		channels = append(channels, responses.ChannelMetricItem{
			Channel:     p.ChannelName,
			TotalClicks: p.TotalClicks,
			UniqueUsers: p.UniqueUsers,
			CTR:         s.calculateCTR(p.TotalClicks, p.UniqueUsers),
		})
	}
	return channels, nil
}

func (s *AffiliateLinkAnalyticsService) getRecentActivity(
	ctx context.Context,
	limit int,
) ([]responses.RecentActivityItem, error) {
	since := time.Now().Add(-24 * time.Hour)
	events, err := s.clickEventRepo.GetRecentClicks(ctx, since, 100)
	if err != nil {
		return nil, err
	}

	// Sort DESC
	sort.Slice(events, func(i, j int) bool {
		return events[i].ClickedAt.After(events[j].ClickedAt)
	})

	if len(events) > limit {
		events = events[:limit]
	}

	var items []responses.RecentActivityItem
	for _, e := range events {
		// Need to fetch link to get name
		link, err := s.affiliateLinkRepo.GetByID(ctx, e.AffiliateLinkID, nil)
		linkName := "Unknown"
		if err == nil {
			linkName = link.Hash
		}

		items = append(items, responses.RecentActivityItem{
			AffiliateLinkID: e.AffiliateLinkID,
			LinkName:        linkName,
			Channel:         s.getChannelString(link.Channel),
			ClickCount:      1,
			Timestamp:       e.ClickedAt,
		})
	}
	return items, nil
}

func (s *AffiliateLinkAnalyticsService) getTrendData(
	ctx context.Context,
	startDate, endDate time.Time,
) ([]responses.TrendDataPoint, error) {
	stats, err := s.clickEventRepo.GetGlobalTrendData(ctx, startDate, endDate)
	if err != nil {
		return nil, err
	}

	var points []responses.TrendDataPoint
	for _, s := range stats {
		points = append(points, responses.TrendDataPoint{
			Date:        s.Date,
			Clicks:      s.TotalClicks,
			UniqueUsers: s.UniqueUsers,
		})
	}
	return points, nil
}

func (s *AffiliateLinkAnalyticsService) calculateGrowth(current, previous float64) float64 {
	if previous == 0 {
		if current > 0 {
			return 100 // 100% growth if started from 0
		}
		return 0
	}
	return ((current - previous) / previous) * 100
}
