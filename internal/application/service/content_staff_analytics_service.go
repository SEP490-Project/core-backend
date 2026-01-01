package service

import (
	"context"
	"core-backend/internal/application/dto/dtos"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/application/interfaces/iservice"
	"core-backend/internal/domain/constant"
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
	gormrepository "core-backend/internal/infrastructure/gorm_repository"
	"core-backend/pkg/utils"
	"encoding/json"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type contentStaffAnalyticsService struct {
	dashboardRepo      irepository.ContentStaffAnalyticsRepository
	scheduleRepo       irepository.ScheduleRepository
	alertRepo          irepository.SystemAlertRepository
	channelRepo        irepository.GenericRepository[model.Channel]
	kpiMetricsRepo     irepository.KPIMetricsRepository
	contentChannelRepo irepository.GenericRepository[model.ContentChannel]
	clickEventRepo     irepository.ClickEventRepository
}

// NewContentStaffAnalyticsService creates a new content staff analytics service
func NewContentStaffAnalyticsService(
	dbReg *gormrepository.DatabaseRegistry,
) iservice.ContentStaffAnalyticsService {
	return &contentStaffAnalyticsService{
		dashboardRepo:      dbReg.ContentStaffAnalyticsRepository,
		scheduleRepo:       dbReg.ScheduleRepository,
		alertRepo:          dbReg.SystemAlertRepository,
		channelRepo:        dbReg.ChannelRepository,
		kpiMetricsRepo:     dbReg.KPIMetricsRepository,
		contentChannelRepo: dbReg.ContentChannelRepository,
		clickEventRepo:     dbReg.ClickEventRepository,
	}
}

// GetDashboard returns the complete content dashboard data
func (s *contentStaffAnalyticsService) GetDashboard(ctx context.Context, filter *requests.ContentDashboardFilterRequest, userID uuid.UUID) (*responses.ContentDashboardResponse, error) {
	// Get date ranges
	currentRange, previousRange := filter.GetDateRanges()

	// Parse channel filter
	var channelID *uuid.UUID
	if filter.ChannelID != nil {
		parsed, err := uuid.Parse(*filter.ChannelID)
		if err == nil {
			channelID = &parsed
		}
	}

	dashboard := &responses.ContentDashboardResponse{
		Period: responses.DashboardPeriodInfo{
			PresetLabel:   filter.GetPresetLabel(),
			CompareLabel:  filter.GetCompareLabel(),
			CurrentStart:  currentRange.Start,
			CurrentEnd:    currentRange.End,
			PreviousStart: previousRange.Start,
			PreviousEnd:   previousRange.End,
		},
	}

	var mu sync.Mutex

	// Run all queries in parallel
	tasks := []func(ctx context.Context) error{
		// Quick Stats
		func(ctx context.Context) error {
			stats, err := s.getQuickStats(ctx, currentRange, previousRange, channelID, filter.GetCompareLabel())
			if err != nil {
				return err
			}
			mu.Lock()
			dashboard.QuickStats = *stats
			mu.Unlock()
			return nil
		},

		// Channel Metrics
		func(ctx context.Context) error {
			metrics, err := s.getChannelMetrics(ctx, currentRange, previousRange)
			if err != nil {
				return err
			}
			mu.Lock()
			dashboard.ChannelMetrics = metrics
			mu.Unlock()
			return nil
		},

		// Charts
		func(ctx context.Context) error {
			charts, err := s.getCharts(ctx, currentRange, filter.GetTrendGranularity(), channelID)
			if err != nil {
				return err
			}
			mu.Lock()
			dashboard.Charts = *charts
			mu.Unlock()
			return nil
		},

		// Top Content
		func(ctx context.Context) error {
			topContent, err := s.getTopContent(ctx, filter.GetTopContentLimit(), currentRange, channelID)
			if err != nil {
				return err
			}
			mu.Lock()
			dashboard.TopContent = topContent
			mu.Unlock()
			return nil
		},

		// Bottom Content
		func(ctx context.Context) error {
			bottomContent, err := s.getBottomContent(ctx, filter.GetBottomContentLimit(), currentRange, channelID)
			if err != nil {
				return err
			}
			mu.Lock()
			dashboard.BottomContent = bottomContent
			mu.Unlock()
			return nil
		},

		// Upcoming Schedule
		func(ctx context.Context) error {
			schedule, err := s.getUpcomingSchedule(ctx)
			if err != nil {
				return err
			}
			mu.Lock()
			dashboard.UpcomingSchedule = schedule
			mu.Unlock()
			return nil
		},

		// Active Alerts
		func(ctx context.Context) error {
			alerts, err := s.getActiveAlerts(ctx, userID)
			if err != nil {
				return err
			}
			mu.Lock()
			dashboard.Alerts = alerts
			mu.Unlock()
			return nil
		},
	}

	// Execute tasks in parallel with concurrency limit
	if err := utils.RunParallel(ctx, 7, tasks...); err != nil {
		zap.L().Error("Error fetching dashboard data", zap.Error(err))
	}

	return dashboard, nil
}

// getQuickStats fetches quick stats section data
func (s *contentStaffAnalyticsService) getQuickStats(
	ctx context.Context,
	currentRange, previousRange constant.DateRange,
	channelID *uuid.UUID,
	compareLabel string,
) (*responses.QuickStatsSection, error) {
	stats := &responses.QuickStatsSection{}

	var mu sync.Mutex

	tasks := []func(ctx context.Context) error{
		// Posts count
		func(ctx context.Context) error {
			current, _ := s.dashboardRepo.GetPostCountByDateRange(ctx, currentRange.Start, currentRange.End, channelID)
			previous, _ := s.dashboardRepo.GetPostCountByDateRange(ctx, previousRange.Start, previousRange.End, channelID)
			mu.Lock()
			stats.PostsThisWeek = buildMetricWithGrowth(current, previous, compareLabel)
			mu.Unlock()
			return nil
		},

		// Total Views
		func(ctx context.Context) error {
			current, _ := s.dashboardRepo.GetTotalViews(ctx, currentRange.Start, currentRange.End, channelID)
			previous, _ := s.dashboardRepo.GetTotalViews(ctx, previousRange.Start, previousRange.End, channelID)
			mu.Lock()
			stats.TotalViews = buildMetricWithGrowth(current, previous, compareLabel)
			mu.Unlock()
			return nil
		},

		// Total Engagement
		func(ctx context.Context) error {
			current, _ := s.dashboardRepo.GetTotalEngagement(ctx, currentRange.Start, currentRange.End, channelID)
			previous, _ := s.dashboardRepo.GetTotalEngagement(ctx, previousRange.Start, previousRange.End, channelID)
			mu.Lock()
			stats.TotalEngagement = buildMetricWithGrowth(current, previous, compareLabel)
			mu.Unlock()
			return nil
		},

		// Average CTR
		func(ctx context.Context) error {
			current, _ := s.dashboardRepo.GetAverageCTR(ctx, currentRange.Start, currentRange.End, channelID)
			previous, _ := s.dashboardRepo.GetAverageCTR(ctx, previousRange.Start, previousRange.End, channelID)
			mu.Lock()
			stats.AverageCTR = buildMetricWithGrowthFloat(current, previous, compareLabel)
			mu.Unlock()
			return nil
		},

		// Pending Content count
		func(ctx context.Context) error {
			count, _ := s.dashboardRepo.GetPendingContentCount(ctx, currentRange.Start, currentRange.End)
			mu.Lock()
			stats.PendingContent = count
			mu.Unlock()
			return nil
		},
	}

	_ = utils.RunParallel(ctx, 5, tasks...)

	// Calculate posting frequency (expected vs actual)
	stats.PostingFrequency = s.calculatePostingFrequency(ctx, currentRange)

	return stats, nil
}

// calculatePostingFrequency calculates the posting frequency metrics
// Priority: 1) Scheduled content, 2) Task deliverables, 3) Daily average
func (s *contentStaffAnalyticsService) calculatePostingFrequency(ctx context.Context, currentRange constant.DateRange) responses.PostingFrequencyMetric {
	// Get actual posts published in the period
	actual, _ := s.dashboardRepo.GetPostCountByDateRange(ctx, currentRange.Start, currentRange.End, nil)

	var expected int64
	var source string

	// Priority 1: Check for scheduled content in the period
	scheduledCount, _ := s.dashboardRepo.GetScheduledContentCount(ctx, currentRange.Start, currentRange.End)
	if scheduledCount > 0 {
		expected = scheduledCount
		source = "schedule"
	} else {
		// Priority 2: Check for task content deliverables in the period
		taskCount, _ := s.dashboardRepo.GetTaskContentDeliverableCount(ctx, currentRange.Start, currentRange.End)
		if taskCount > 0 {
			expected = taskCount
			source = "tasks"
		} else {
			// Priority 3: Fall back to daily average (1 post per day target)
			days := max(int64(currentRange.End.Sub(currentRange.Start).Hours()/24), 1)
			expected = days
			source = "average"
		}
	}

	ratio := float64(0)
	if expected > 0 {
		ratio = float64(actual) / float64(expected)
	}

	status := "on_track"
	if ratio < 0.8 {
		status = "behind"
	} else if ratio > 1.2 {
		status = "ahead"
	}

	return responses.PostingFrequencyMetric{
		Actual:   actual,
		Expected: expected,
		Ratio:    ratio,
		Status:   status,
		Source:   source,
	}
}

// getChannelMetrics fetches channel metrics data
func (s *contentStaffAnalyticsService) getChannelMetrics(
	ctx context.Context,
	currentRange, previousRange constant.DateRange,
) ([]responses.ChannelMetricsCard, error) {
	// Get current period metrics
	currentMetrics, err := s.dashboardRepo.GetChannelMetrics(ctx, currentRange.Start, currentRange.End)
	if err != nil {
		return nil, err
	}

	// Get previous period metrics for comparison
	previousMetrics, _ := s.dashboardRepo.GetChannelMetrics(ctx, previousRange.Start, previousRange.End)
	previousMap := make(map[uuid.UUID]*dtos.ChannelMetricsDTO)
	for i := range previousMetrics {
		previousMap[previousMetrics[i].ChannelID] = &previousMetrics[i]
	}

	// Build response
	result := make([]responses.ChannelMetricsCard, 0, len(currentMetrics))
	for _, m := range currentMetrics {
		card := responses.ChannelMetricsCard{
			ChannelID:       m.ChannelID,
			ChannelName:     m.ChannelName,
			ChannelCode:     m.ChannelCode,
			PostCount:       m.PostCount,
			TotalReach:      m.TotalReach,
			TotalEngagement: m.TotalEngagement,
			CTR:             m.AverageCTR,
			FollowersCount:  m.FollowersCount,
			FetchedMetrics:  m.FetchedMetrics,
			MappedMetrics:   m.MappedMetrics,
		}
		zap.L().Debug("Channel fetched metrics", zap.String("channel", m.ChannelName),
			zap.Any("fetched_metrics", m.FetchedMetrics), zap.Any("mapped_metrics", m.MappedMetrics))

		// Calculate growth compared to previous period
		if prev, ok := previousMap[m.ChannelID]; ok {
			card.ReachGrowth = calculateGrowth(m.TotalReach, prev.TotalReach)
			card.EngagementGrowth = calculateGrowth(m.TotalEngagement, prev.TotalEngagement)

			// Calculate followers trend
			if m.FollowersCount > 0 || prev.FollowersCount > 0 {
				followersChange := float64(m.FollowersCount - prev.FollowersCount)
				followersPercent := calculateGrowth(m.FollowersCount, prev.FollowersCount)
				card.FollowersTrend = &responses.TrendIndicator{
					Value:      followersChange,
					Percentage: followersPercent,
					Direction:  getGrowthDirection(followersPercent),
				}
			}
		}

		// Get top post for this channel
		topPost, err := s.dashboardRepo.GetTopPostForChannel(ctx, m.ChannelID, currentRange.Start, currentRange.End)
		if err == nil && topPost != nil {
			card.TopPost = &responses.TopPostBrief{
				ContentID: topPost.ContentID,
				Title:     topPost.Title,
				Views:     topPost.Views,
				Likes:     topPost.Likes,
			}
		}

		result = append(result, card)
	}

	return result, nil
}

// getCharts fetches chart data
func (s *contentStaffAnalyticsService) getCharts(
	ctx context.Context,
	currentRange constant.DateRange,
	granularity constant.TrendGranularity,
	channelID *uuid.UUID,
) (*responses.ChartsSection, error) {
	charts := &responses.ChartsSection{}

	var mu sync.Mutex

	tasks := []func(ctx context.Context) error{
		// Reach by Channel (bar chart)
		func(ctx context.Context) error {
			metrics, _ := s.dashboardRepo.GetChannelMetrics(ctx, currentRange.Start, currentRange.End)
			reachData := make([]responses.BarChartDataPoint, 0, len(metrics))
			for _, m := range metrics {
				reachData = append(reachData, responses.BarChartDataPoint{
					Label:      m.ChannelName,
					Reach:      m.TotalReach,
					Engagement: m.TotalEngagement,
				})
			}
			mu.Lock()
			charts.ReachByChannel = reachData
			mu.Unlock()
			return nil
		},

		// Trend Data (line chart)
		func(ctx context.Context) error {
			trendData, _ := s.dashboardRepo.GetTrendData(ctx, currentRange.Start, currentRange.End, granularity, channelID)
			points := make([]responses.DashboardTimeSeriesPoint, 0, len(trendData))
			for _, t := range trendData {
				points = append(points, responses.DashboardTimeSeriesPoint{
					Date:        t.Date,
					Views:       t.Views,
					Likes:       t.Likes,
					Comments:    t.Comments,
					Shares:      t.Shares,
					Engagements: t.Engagements,
				})
			}
			mu.Lock()
			charts.TrendData = points
			mu.Unlock()
			return nil
		},

		// Content Type Distribution (pie chart)
		func(ctx context.Context) error {
			distribution, _ := s.dashboardRepo.GetContentTypeDistribution(ctx, currentRange.Start, currentRange.End)
			pieData := make([]responses.PieChartDataPoint, 0, len(distribution))
			for _, d := range distribution {
				pieData = append(pieData, responses.PieChartDataPoint{
					Label: d.ContentType,
					Value: d.Count,
					Ratio: d.Percentage,
				})
			}
			mu.Lock()
			charts.ContentTypeDistribution = pieData
			mu.Unlock()
			return nil
		},

		// Channel Distribution (pie chart - posts per channel)
		func(ctx context.Context) error {
			distribution, _ := s.dashboardRepo.GetChannelDistribution(ctx, currentRange.Start, currentRange.End)
			pieData := make([]responses.PieChartDataPoint, 0, len(distribution))
			for _, d := range distribution {
				pieData = append(pieData, responses.PieChartDataPoint{
					Label: d.ChannelName,
					Value: d.Count,
					Ratio: d.Percentage,
				})
			}
			mu.Lock()
			charts.ChannelDistribution = pieData
			mu.Unlock()
			return nil
		},
	}

	_ = utils.RunParallel(ctx, 4, tasks...)

	return charts, nil
}

// getTopContent fetches top performing content
func (s *contentStaffAnalyticsService) getTopContent(
	ctx context.Context,
	limit int,
	currentRange constant.DateRange,
	channelID *uuid.UUID,
) ([]responses.ContentPerformanceItem, error) {
	content, err := s.dashboardRepo.GetTopContentByPerformance(ctx, limit, currentRange.Start, currentRange.End, channelID)
	if err != nil {
		return nil, err
	}

	result := make([]responses.ContentPerformanceItem, 0, len(content))
	for i, c := range content {
		result = append(result, responses.ContentPerformanceItem{
			ContentID:        c.ContentID,
			Title:            c.Title,
			Type:             c.ContentType,
			ChannelName:      c.ChannelName,
			Views:            c.Views,
			Engagement:       c.Engagement,
			CTR:              c.CTR,
			PerformanceScore: c.PerformanceScore,
			PublishedAt:      c.PublishedAt,
			Thumbnail:        c.ThumbnailURL,
			Rank:             i + 1,
		})
	}

	return result, nil
}

// getBottomContent fetches bottom performing content
func (s *contentStaffAnalyticsService) getBottomContent(
	ctx context.Context,
	limit int,
	currentRange constant.DateRange,
	channelID *uuid.UUID,
) ([]responses.ContentPerformanceItem, error) {
	content, err := s.dashboardRepo.GetBottomContentByPerformance(ctx, limit, currentRange.Start, currentRange.End, channelID)
	if err != nil {
		return nil, err
	}

	result := make([]responses.ContentPerformanceItem, 0, len(content))
	for i, c := range content {
		result = append(result, responses.ContentPerformanceItem{
			ContentID:        c.ContentID,
			Title:            c.Title,
			Type:             c.ContentType,
			ChannelName:      c.ChannelName,
			Views:            c.Views,
			Engagement:       c.Engagement,
			CTR:              c.CTR,
			PerformanceScore: c.PerformanceScore,
			PublishedAt:      c.PublishedAt,
			Thumbnail:        c.ThumbnailURL,
			Rank:             i + 1,
		})
	}

	return result, nil
}

// getUpcomingSchedule fetches upcoming scheduled content
func (s *contentStaffAnalyticsService) getUpcomingSchedule(ctx context.Context) ([]responses.ScheduledContentItem, error) {
	// Get schedules for the next 7 days
	from := time.Now()
	to := from.AddDate(0, 0, 7)

	schedules, err := s.scheduleRepo.GetUpcomingSchedules(ctx, from, to, 10)
	if err != nil {
		return nil, err
	}

	result := make([]responses.ScheduledContentItem, 0, len(schedules))
	for _, sch := range schedules {
		// Get schedule details using content-specific method
		detail, err := s.scheduleRepo.GetContentScheduleByIDWithDetails(ctx, sch.ID)
		if err != nil || detail == nil {
			continue
		}

		var contentID uuid.UUID
		var contentTitle, channelName string
		if detail.ContentDetails != nil {
			contentID = detail.ContentDetails.ContentID
			contentTitle = detail.ContentDetails.ContentTitle
			channelName = detail.ContentDetails.ChannelName
		}

		result = append(result, responses.ScheduledContentItem{
			ScheduleID:  detail.ScheduleID,
			ContentID:   contentID,
			Title:       contentTitle,
			ChannelName: channelName,
			ScheduledAt: detail.ScheduledAt,
			Status:      detail.Status.String(),
			CreatedBy:   detail.CreatedByName,
			CreatedByID: detail.CreatedBy,
		})
	}

	return result, nil
}

// getActiveAlerts fetches active alerts for the user
func (s *contentStaffAnalyticsService) getActiveAlerts(ctx context.Context, userID uuid.UUID) ([]responses.AlertItem, error) {
	alerts, err := s.alertRepo.GetActiveAlerts(ctx, nil, nil)
	if err != nil {
		return nil, err
	}

	result := make([]responses.AlertItem, 0, len(alerts))
	for _, alert := range alerts {
		// Check if user has acknowledged this alert
		isAcknowledged, _ := s.alertRepo.IsAlertAcknowledgedByUser(ctx, alert.ID, userID)

		item := responses.AlertItem{
			ID:          alert.ID,
			Type:        string(alert.Type),
			Category:    string(alert.Category),
			Severity:    string(alert.Severity),
			Title:       alert.Title,
			Description: alert.Description,
			CreatedAt:   *alert.CreatedAt,
			IsRead:      isAcknowledged,
		}

		if alert.ReferenceID != nil {
			item.ReferenceID = alert.ReferenceID
		}
		if alert.ReferenceType != nil {
			refType := string(*alert.ReferenceType)
			item.ReferenceType = &refType
		}
		if alert.ActionURL != nil {
			item.ActionURL = alert.ActionURL
		}

		result = append(result, item)
	}

	return result, nil
}

// Helper functions

func buildMetricWithGrowth(current, previous int64, compareLabel string) responses.MetricWithGrowth {
	growth := calculateGrowth(current, previous)
	status := "stable"
	if growth > 0 {
		status = "up"
	} else if growth < 0 {
		status = "down"
	}

	return responses.MetricWithGrowth{
		Value:         current,
		PreviousValue: previous,
		Growth:        growth,
		GrowthStatus:  status,
		CompareLabel:  compareLabel,
	}
}

func buildMetricWithGrowthFloat(current, previous float64, compareLabel string) responses.MetricWithGrowth {
	growth := calculateGrowthFloat(current, previous)
	status := "stable"
	if growth > 0 {
		status = "up"
	} else if growth < 0 {
		status = "down"
	}

	return responses.MetricWithGrowth{
		Value:         current,
		PreviousValue: previous,
		Growth:        growth,
		GrowthStatus:  status,
		CompareLabel:  compareLabel,
	}
}

func calculateGrowth(current, previous int64) float64 {
	if previous == 0 {
		if current > 0 {
			return 100.0
		}
		return 0.0
	}
	return float64(current-previous) / float64(previous) * 100
}

func calculateGrowthFloat(current, previous float64) float64 {
	if previous == 0 {
		if current > 0 {
			return 100.0
		}
		return 0.0
	}
	return (current - previous) / previous * 100
}

func getGrowthDirection(growthPercent float64) string {
	if growthPercent > 0.5 {
		return "up"
	} else if growthPercent < -0.5 {
		return "down"
	}
	return "stable"
}

// =============================================================================
// Channel Details Methods
// =============================================================================

// GetChannelDetails returns detailed metrics for a specific channel
func (s *contentStaffAnalyticsService) GetChannelDetails(
	ctx context.Context,
	channelID uuid.UUID,
	filter *requests.ChannelDetailsRequest,
) (*responses.ChannelDetailsResponse, error) {
	// Get channel info
	channel, err := s.channelRepo.GetByID(ctx, channelID, nil)
	if err != nil {
		return nil, err
	}

	currentRange, previousRange := filter.GetDateRanges()

	response := &responses.ChannelDetailsResponse{
		Channel: responses.ChannelInfo{
			ID:          channel.ID,
			Name:        channel.Name,
			Code:        channel.Code,
			Description: channel.Description,
			HomePageURL: channel.HomePageURL,
			IsActive:    channel.IsActive,
		},
		Period: responses.DashboardPeriodInfo{
			PresetLabel:   filter.GetPresetLabel(),
			CompareLabel:  filter.GetCompareLabel(),
			CurrentStart:  currentRange.Start,
			CurrentEnd:    currentRange.End,
			PreviousStart: previousRange.Start,
			PreviousEnd:   previousRange.End,
		},
		MappedMetrics:  make(map[string]float64),
		FetchedMetrics: make(map[string]any),
	}

	// Add token info if available
	if channel.HashedAccessToken != nil && *channel.HashedAccessToken != "" {
		response.Channel.TokenInfo = &responses.TokenInfo{
			AccountName:          channel.AccountName,
			ExternalID:           channel.ExternalID,
			AccessTokenExpiresAt: channel.AccessTokenExpiresAt,
			LastSyncedAt:         channel.LastSyncedAt,
		}
	}

	var mu sync.Mutex

	// Run parallel queries for channel details
	tasks := []func(ctx context.Context) error{
		// Mapped Metrics from kpi_metrics
		func(ctx context.Context) error {
			metrics, err := s.getChannelMappedMetrics(ctx, channelID, currentRange.Start, currentRange.End)
			if err != nil {
				zap.L().Warn("Failed to get channel mapped metrics", zap.Error(err))
				return nil // Don't fail entire request
			}
			mu.Lock()
			response.MappedMetrics = metrics
			mu.Unlock()
			return nil
		},
		// Fetched Metrics from channel.Metrics JSONB
		func(ctx context.Context) error {
			if channel.Metrics != nil {
				var metrics model.ContentChannelMetrics
				if err := json.Unmarshal(channel.Metrics, &metrics); err == nil {
					mu.Lock()
					response.FetchedMetrics = metrics.CurrentFetched
					mu.Unlock()
				}
			}
			return nil
		},
		// Content Trend
		func(ctx context.Context) error {
			trend, err := s.getChannelContentTrend(ctx, channelID, currentRange.Start, currentRange.End)
			if err != nil {
				zap.L().Warn("Failed to get channel content trend", zap.Error(err))
				return nil
			}
			mu.Lock()
			response.ContentTrend = trend
			mu.Unlock()
			return nil
		},
		// Engagement Trend
		func(ctx context.Context) error {
			trend, err := s.dashboardRepo.GetTrendData(ctx, currentRange.Start, currentRange.End, constant.TrendGranularityDay, &channelID)
			if err != nil {
				zap.L().Warn("Failed to get channel engagement trend", zap.Error(err))
				return nil
			}
			mu.Lock()
			response.EngagementTrend = s.mapTrendDataToTimeSeries(trend)
			mu.Unlock()
			return nil
		},
		// Top Content
		func(ctx context.Context) error {
			topContent, err := s.dashboardRepo.GetTopContentByPerformance(ctx, 5, currentRange.Start, currentRange.End, &channelID)
			if err != nil {
				zap.L().Warn("Failed to get channel top content", zap.Error(err))
				return nil
			}
			// Map to response format
			items := make([]responses.ContentPerformanceItem, 0, len(topContent))
			for i, c := range topContent {
				items = append(items, responses.ContentPerformanceItem{
					ContentID:        c.ContentID,
					Title:            c.Title,
					Type:             c.ContentType,
					ChannelName:      c.ChannelName,
					Views:            c.Views,
					Engagement:       c.Engagement,
					CTR:              c.CTR,
					PerformanceScore: c.PerformanceScore,
					PublishedAt:      c.PublishedAt,
					Thumbnail:        c.ThumbnailURL,
					Rank:             i + 1,
				})
			}
			mu.Lock()
			response.TopContent = items
			mu.Unlock()
			return nil
		},
		// Recent Content
		func(ctx context.Context) error {
			recentContent, err := s.getChannelRecentContent(ctx, channelID, 10)
			if err != nil {
				zap.L().Warn("Failed to get channel recent content", zap.Error(err))
				return nil
			}
			mu.Lock()
			response.RecentContent = recentContent
			mu.Unlock()
			return nil
		},
		// Affiliate Stats
		func(ctx context.Context) error {
			stats, err := s.getChannelAffiliateStats(ctx, channelID, currentRange.Start, currentRange.End)
			if err != nil {
				zap.L().Warn("Failed to get channel affiliate stats", zap.Error(err))
				return nil
			}
			mu.Lock()
			response.AffiliateStats = stats
			mu.Unlock()
			return nil
		},
		// Followers Count
		func(ctx context.Context) error {
			count, trend := s.getChannelFollowersCount(ctx, channelID, currentRange, previousRange)
			mu.Lock()
			response.FollowersCount = count
			response.FollowersTrend = trend
			mu.Unlock()
			return nil
		},
	}

	// Execute all tasks in parallel
	_ = utils.RunParallel(ctx, 8, tasks...)

	return response, nil
}

// getChannelMappedMetrics aggregates KPI metrics for a channel from content_channels
// Uses the repository method with DISTINCT ON to get LATEST values
func (s *contentStaffAnalyticsService) getChannelMappedMetrics(
	ctx context.Context,
	channelID uuid.UUID,
	startDate, endDate time.Time,
) (map[string]float64, error) {
	return s.dashboardRepo.GetChannelMappedMetrics(ctx, channelID, startDate, endDate)
}

// getChannelContentTrend returns content posts over time for a channel
func (s *contentStaffAnalyticsService) getChannelContentTrend(
	ctx context.Context,
	channelID uuid.UUID,
	startDate, endDate time.Time,
) ([]responses.ContentTrendPoint, error) {
	// Get content channels for this channel in date range
	contentChannels, _, err := s.contentChannelRepo.GetAll(ctx, func(db *gorm.DB) *gorm.DB {
		return db.Where("channel_id = ?", channelID).
			Where("published_at >= ?", startDate).
			Where("published_at < ?", endDate).
			Where("published_at IS NOT NULL")
	}, nil, 1000, 1)
	if err != nil {
		return nil, err
	}

	// Group by date
	dateCountMap := make(map[string]int)
	for _, cc := range contentChannels {
		if cc.PublishedAt != nil {
			dateKey := cc.PublishedAt.Format("2006-01-02")
			dateCountMap[dateKey]++
		}
	}

	// Convert to trend points
	var trend []responses.ContentTrendPoint
	for dateStr, count := range dateCountMap {
		date, _ := time.Parse("2006-01-02", dateStr)
		trend = append(trend, responses.ContentTrendPoint{
			Date:  date,
			Posts: count,
		})
	}

	return trend, nil
}

// getChannelRecentContent returns recent content for a channel
func (s *contentStaffAnalyticsService) getChannelRecentContent(
	ctx context.Context,
	channelID uuid.UUID,
	limit int,
) ([]responses.ChannelRecentContentItem, error) {
	// Get recent content channels with content preloaded
	contentChannels, _, err := s.contentChannelRepo.GetAll(ctx, func(db *gorm.DB) *gorm.DB {
		return db.Where("channel_id = ?", channelID).
			Order("created_at DESC")
	}, []string{"Content"}, limit, 1)
	if err != nil {
		return nil, err
	}

	var items []responses.ChannelRecentContentItem
	for _, cc := range contentChannels {
		if cc.Content == nil {
			continue
		}

		// Get views and engagement from ContentChannelMetrics struct
		var views, engagement int64
		if cc.Metrics != nil {
			// Try CurrentMapped first (more reliable aggregated values)
			if cc.Metrics.CurrentMapped != nil {
				if v, ok := cc.Metrics.CurrentMapped[enum.KPIValueTypeViews]; ok {
					views = int64(v)
				}
				if e, ok := cc.Metrics.CurrentMapped[enum.KPIValueTypeEngagement]; ok {
					engagement = int64(e)
				}
			}
			// Fallback to CurrentFetched if CurrentMapped is empty
			if views == 0 && cc.Metrics.CurrentFetched != nil {
				if v, ok := cc.Metrics.CurrentFetched["views"].(float64); ok {
					views = int64(v)
				}
				if e, ok := cc.Metrics.CurrentFetched["engagement"].(float64); ok {
					engagement = int64(e)
				}
			}
		}

		items = append(items, responses.ChannelRecentContentItem{
			ContentID:   cc.ContentID,
			Title:       cc.Content.Title,
			Type:        string(cc.Content.Type),
			Status:      string(cc.AutoPostStatus),
			Views:       views,
			Engagement:  engagement,
			PublishedAt: cc.PublishedAt,
		})
	}

	return items, nil
}

// getChannelAffiliateStats returns affiliate link statistics for a channel
func (s *contentStaffAnalyticsService) getChannelAffiliateStats(
	ctx context.Context,
	channelID uuid.UUID,
	startDate, endDate time.Time,
) (*responses.AffiliateStatsResponse, error) {
	// Get click events for this channel
	clickEvents, err := s.clickEventRepo.GetClicksByChannel(ctx, channelID, startDate, endDate)
	if err != nil {
		return nil, err
	}

	totalClicks := int64(len(clickEvents))

	// Count unique users (by UserID or IP hash)
	uniqueUsers := make(map[string]struct{})
	for _, ce := range clickEvents {
		var key string
		if ce.UserID != nil {
			key = ce.UserID.String()
		} else if ce.IPAddress != nil {
			key = *ce.IPAddress
		}
		if key != "" {
			uniqueUsers[key] = struct{}{}
		}
	}
	uniqueCount := int64(len(uniqueUsers))

	// hasLinks := totalClicks > 0 || uniqueCount > 0
	uniqueLinks := make(map[uuid.UUID]struct{})
	for _, ce := range clickEvents {
		if ce.UserID != nil {
			uniqueLinks[*ce.UserID] = struct{}{}
		}
	}
	hasLinks := len(uniqueLinks) > 0

	var ctr any = "N/A"
	if hasLinks && uniqueCount > 0 {
		ctr = float64(totalClicks) / float64(uniqueCount) * 100
	}

	return &responses.AffiliateStatsResponse{
		TotalLinks:  len(uniqueLinks),
		TotalClicks: totalClicks,
		UniqueUsers: uniqueCount,
		CTR:         ctr,
		HasLinks:    hasLinks,
	}, nil
}

// getChannelFollowersCount returns followers count and trend for a channel
func (s *contentStaffAnalyticsService) getChannelFollowersCount(
	ctx context.Context,
	channelID uuid.UUID,
	currentRange, previousRange constant.DateRange,
) (int64, *responses.TrendIndicator) {
	// Get current followers from channel.Metrics JSONB or KPI metrics
	channel, err := s.channelRepo.GetByID(ctx, channelID, nil)
	if err != nil {
		return 0, nil
	}

	var currentFollowers int64
	if channel.Metrics != nil {
		var metrics map[string]any
		if err := json.Unmarshal(channel.Metrics, &metrics); err == nil {
			if f, ok := metrics["followers"].(float64); ok {
				currentFollowers = int64(f)
			} else if f, ok := metrics["page_fan_count"].(float64); ok {
				currentFollowers = int64(f)
			}
		}
	}

	// TODO: Get previous period followers for trend calculation
	// For now, return current followers without trend

	return currentFollowers, nil
}

// mapTrendDataToTimeSeries converts DTO trend data to response format
func (s *contentStaffAnalyticsService) mapTrendDataToTimeSeries(data []dtos.TrendDataPointDTO) []responses.DashboardTimeSeriesPoint {
	result := make([]responses.DashboardTimeSeriesPoint, len(data))
	for i, d := range data {
		result[i] = responses.DashboardTimeSeriesPoint{
			Date:        d.Date,
			Views:       d.Views,
			Likes:       d.Likes,
			Comments:    d.Comments,
			Shares:      d.Shares,
			Engagements: d.Engagements,
		}
	}
	return result
}
