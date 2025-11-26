package service

import (
	"context"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/application/interfaces/iservice"
	"core-backend/internal/domain/enum"
	"core-backend/pkg/utils"
	"sync"
	"time"

	"go.uber.org/zap"
)

type contentStaffAnalyticsService struct {
	analyticsRepo irepository.ContentStaffAnalyticsRepository
}

// NewContentStaffAnalyticsService creates a new content staff analytics service
func NewContentStaffAnalyticsService(
	analyticsRepo irepository.ContentStaffAnalyticsRepository,
) iservice.ContentStaffAnalyticsService {
	return &contentStaffAnalyticsService{
		analyticsRepo: analyticsRepo,
	}
}

// GetDashboard returns the complete Content Staff dashboard
func (s *contentStaffAnalyticsService) GetDashboard(ctx context.Context, req *requests.ContentStaffDashboardRequest) (*responses.ContentStaffDashboardResponse, error) {
	startDate, endDate := req.GetDateRange()

	var mu sync.Mutex
	dashboard := &responses.ContentStaffDashboardResponse{
		Period: responses.PeriodInfo{
			StartDate: startDate,
			EndDate:   endDate,
		},
	}

	// Execute queries in parallel
	err := utils.RunParallel(ctx, 7,
		// Query 1: Overview metrics
		func(ctx context.Context) error {
			overview, err := s.getOverviewMetrics(ctx, &startDate, &endDate)
			if err != nil {
				zap.L().Warn("Failed to get overview metrics", zap.Error(err))
				return nil
			}
			mu.Lock()
			dashboard.Overview = *overview
			mu.Unlock()
			return nil
		},

		// Query 2: Content status breakdown
		func(ctx context.Context) error {
			breakdown, err := s.GetContentStatusBreakdown(ctx, &requests.ContentStatusRequest{
				StartDate: &startDate,
				EndDate:   &endDate,
			})
			if err != nil {
				zap.L().Warn("Failed to get content status breakdown", zap.Error(err))
				return nil
			}
			mu.Lock()
			dashboard.ContentStatusBreakdown = *breakdown
			mu.Unlock()
			return nil
		},

		// Query 3: Platform metrics
		func(ctx context.Context) error {
			metrics, err := s.GetMetricsByPlatform(ctx, &requests.PlatformMetricsRequest{
				StartDate: &startDate,
				EndDate:   &endDate,
			})
			if err != nil {
				zap.L().Warn("Failed to get platform metrics", zap.Error(err))
				return nil
			}
			mu.Lock()
			dashboard.PlatformMetrics = metrics
			mu.Unlock()
			return nil
		},

		// Query 4: Top content
		func(ctx context.Context) error {
			content, err := s.GetTopContent(ctx, &requests.TopContentRequest{
				StartDate: &startDate,
				EndDate:   &endDate,
				Limit:     5,
			})
			if err != nil {
				zap.L().Warn("Failed to get top content", zap.Error(err))
				return nil
			}
			mu.Lock()
			dashboard.TopContent = content
			mu.Unlock()
			return nil
		},

		// Query 5: Top channels
		func(ctx context.Context) error {
			channels, err := s.GetTopChannels(ctx, &requests.TopChannelsRequest{
				StartDate: &startDate,
				EndDate:   &endDate,
				Limit:     5,
			})
			if err != nil {
				zap.L().Warn("Failed to get top channels", zap.Error(err))
				return nil
			}
			mu.Lock()
			dashboard.TopChannels = channels
			mu.Unlock()
			return nil
		},

		// Query 6: Recent content
		func(ctx context.Context) error {
			recent, err := s.analyticsRepo.GetRecentContent(ctx, 10)
			if err != nil {
				zap.L().Warn("Failed to get recent content", zap.Error(err))
				return nil
			}
			recentItems := make([]responses.RecentContentItem, len(recent))
			for i, r := range recent {
				recentItems[i] = responses.RecentContentItem{
					ContentID:    r.ContentID,
					Title:        r.Title,
					Status:       r.Status,
					CampaignName: r.CampaignName,
					CreatorName:  r.CreatorName,
					CreatedAt:    r.CreatedAt,
					UpdatedAt:    r.UpdatedAt,
				}
			}
			mu.Lock()
			dashboard.RecentContent = recentItems
			mu.Unlock()
			return nil
		},

		// Query 7: Engagement trend
		func(ctx context.Context) error {
			trend, err := s.GetEngagementTrend(ctx, &requests.EngagementTrendRequest{
				StartDate:   &startDate,
				EndDate:     &endDate,
				Granularity: "DAY",
			})
			if err != nil {
				zap.L().Warn("Failed to get engagement trend", zap.Error(err))
				return nil
			}
			mu.Lock()
			dashboard.EngagementTrend = trend
			mu.Unlock()
			return nil
		},
	)

	if err != nil {
		zap.L().Error("Dashboard parallel query failed", zap.Error(err))
	}

	return dashboard, nil
}

// getOverviewMetrics returns high-level overview metrics
func (s *contentStaffAnalyticsService) getOverviewMetrics(ctx context.Context, startDate, endDate *time.Time) (*responses.ContentOverviewMetrics, error) {
	overview := &responses.ContentOverviewMetrics{}

	var mu sync.Mutex
	_ = utils.RunParallel(ctx, 7,
		func(ctx context.Context) error {
			count, _ := s.analyticsRepo.GetTotalContentCount(ctx, startDate, endDate)
			mu.Lock()
			overview.TotalContent = count
			mu.Unlock()
			return nil
		},
		func(ctx context.Context) error {
			count, _ := s.analyticsRepo.GetContentCountByStatus(ctx, enum.ContentStatusPosted.String(), startDate, endDate)
			mu.Lock()
			overview.PostedContent = count
			mu.Unlock()
			return nil
		},
		func(ctx context.Context) error {
			// Note: "PENDING" doesn't map to a ContentStatus enum - may need to combine AWAIT_STAFF and AWAIT_BRAND
			awaitStaffCount, _ := s.analyticsRepo.GetContentCountByStatus(ctx, enum.ContentStatusAwaitStaff.String(), startDate, endDate)
			awaitBrandCount, _ := s.analyticsRepo.GetContentCountByStatus(ctx, enum.ContentStatusAwaitBrand.String(), startDate, endDate)
			mu.Lock()
			overview.PendingContent = awaitStaffCount + awaitBrandCount
			mu.Unlock()
			return nil
		},
		func(ctx context.Context) error {
			count, _ := s.analyticsRepo.GetContentCountByStatus(ctx, enum.ContentStatusDraft.String(), startDate, endDate)
			mu.Lock()
			overview.DraftContent = count
			mu.Unlock()
			return nil
		},
		func(ctx context.Context) error {
			views, _ := s.analyticsRepo.GetTotalViews(ctx, startDate, endDate)
			mu.Lock()
			overview.TotalViews = views
			mu.Unlock()
			return nil
		},
		func(ctx context.Context) error {
			engagements, _ := s.analyticsRepo.GetTotalEngagements(ctx, startDate, endDate)
			mu.Lock()
			overview.TotalEngagements = engagements
			mu.Unlock()
			return nil
		},
		func(ctx context.Context) error {
			clicks, _ := s.analyticsRepo.GetTotalClicks(ctx, startDate, endDate)
			mu.Lock()
			overview.TotalClicks = clicks
			mu.Unlock()
			return nil
		},
	)

	// Calculate engagement rate
	if overview.TotalViews > 0 {
		overview.EngagementRate = float64(overview.TotalEngagements) / float64(overview.TotalViews) * 100
	}

	return overview, nil
}

// GetContentStatusBreakdown returns content counts by status
func (s *contentStaffAnalyticsService) GetContentStatusBreakdown(ctx context.Context, req *requests.ContentStatusRequest) (*responses.ContentStatusBreakdown, error) {
	results, err := s.analyticsRepo.GetContentStatusBreakdown(ctx, req.StartDate, req.EndDate)
	if err != nil {
		return nil, err
	}

	breakdown := &responses.ContentStatusBreakdown{}
	for _, r := range results {
		switch r.Status {
		case enum.ContentStatusDraft.String():
			breakdown.DraftCount = r.Count
		case enum.ContentStatusAwaitStaff.String(), enum.ContentStatusAwaitBrand.String():
			breakdown.PendingCount += r.Count
		case enum.ContentStatusPosted.String():
			breakdown.PostedCount = r.Count
		case enum.ContentStatusApproved.String():
			breakdown.ApprovedCount = r.Count
		case enum.ContentStatusRejected.String():
			breakdown.RejectedCount = r.Count
		}
		breakdown.TotalCount += r.Count
	}

	return breakdown, nil
}

// GetMetricsByPlatform returns metrics aggregated by platform
func (s *contentStaffAnalyticsService) GetMetricsByPlatform(ctx context.Context, req *requests.PlatformMetricsRequest) ([]responses.PlatformMetric, error) {
	results, err := s.analyticsRepo.GetMetricsByPlatform(ctx, req.StartDate, req.EndDate)
	if err != nil {
		return nil, err
	}

	metrics := make([]responses.PlatformMetric, len(results))
	for i, r := range results {
		metrics[i] = responses.PlatformMetric{
			Platform:       r.Platform,
			ContentCount:   r.ContentCount,
			TotalViews:     r.TotalViews,
			TotalLikes:     r.TotalLikes,
			TotalComments:  r.TotalComments,
			TotalShares:    r.TotalShares,
			TotalClicks:    r.TotalClicks,
			EngagementRate: r.EngagementRate,
		}
	}
	return metrics, nil
}

// GetTopContent returns top content by views
func (s *contentStaffAnalyticsService) GetTopContent(ctx context.Context, req *requests.TopContentRequest) ([]responses.ContentMetric, error) {
	results, err := s.analyticsRepo.GetTopContentByViews(ctx, req.Platform, req.GetLimit(), req.StartDate, req.EndDate)
	if err != nil {
		return nil, err
	}

	content := make([]responses.ContentMetric, len(results))
	for i, r := range results {
		content[i] = responses.ContentMetric{
			ContentID:      r.ContentID,
			Title:          r.Title,
			Platform:       r.Platform,
			ChannelName:    r.ChannelName,
			CampaignName:   r.CampaignName,
			Views:          r.Views,
			Likes:          r.Likes,
			Comments:       r.Comments,
			Shares:         r.Shares,
			Clicks:         r.Clicks,
			EngagementRate: r.EngagementRate,
			PostedAt:       r.PostedAt,
			Rank:           i + 1,
		}
	}
	return content, nil
}

// GetTopChannels returns top channels by engagement
func (s *contentStaffAnalyticsService) GetTopChannels(ctx context.Context, req *requests.TopChannelsRequest) ([]responses.ChannelMetric, error) {
	results, err := s.analyticsRepo.GetTopChannelsByEngagement(ctx, req.GetLimit(), req.StartDate, req.EndDate)
	if err != nil {
		return nil, err
	}

	channels := make([]responses.ChannelMetric, len(results))
	for i, r := range results {
		channels[i] = responses.ChannelMetric{
			ChannelID:        r.ChannelID,
			ChannelName:      r.ChannelName,
			Platform:         r.Platform,
			OwnerName:        r.OwnerName,
			ContentCount:     r.ContentCount,
			TotalViews:       r.TotalViews,
			TotalLikes:       r.TotalLikes,
			TotalComments:    r.TotalComments,
			TotalShares:      r.TotalShares,
			TotalEngagements: r.TotalEngagements,
			EngagementRate:   r.EngagementRate,
			Rank:             i + 1,
		}
	}
	return channels, nil
}

// GetEngagementTrend returns engagement time-series data
func (s *contentStaffAnalyticsService) GetEngagementTrend(ctx context.Context, req *requests.EngagementTrendRequest) ([]responses.EngagementTrendPoint, error) {
	results, err := s.analyticsRepo.GetEngagementTrend(ctx, req.GetGranularity(), req.StartDate, req.EndDate)
	if err != nil {
		return nil, err
	}

	trend := make([]responses.EngagementTrendPoint, len(results))
	for i, r := range results {
		trend[i] = responses.EngagementTrendPoint{
			Date:        r.Date,
			Views:       r.Views,
			Likes:       r.Likes,
			Comments:    r.Comments,
			Shares:      r.Shares,
			Engagements: r.Engagements,
		}
	}
	return trend, nil
}

// GetCampaignContentMetrics returns content metrics by campaign
func (s *contentStaffAnalyticsService) GetCampaignContentMetrics(ctx context.Context, req *requests.CampaignContentRequest) ([]responses.CampaignContentMetric, error) {
	results, err := s.analyticsRepo.GetCampaignContentMetrics(ctx, req.CampaignID, req.GetLimit(), req.StartDate, req.EndDate)
	if err != nil {
		return nil, err
	}

	metrics := make([]responses.CampaignContentMetric, len(results))
	for i, r := range results {
		metrics[i] = responses.CampaignContentMetric{
			CampaignID:       r.CampaignID,
			CampaignName:     r.CampaignName,
			ContentCount:     r.ContentCount,
			PostedCount:      r.PostedCount,
			PendingCount:     r.PendingCount,
			DraftCount:       r.DraftCount,
			TotalViews:       r.TotalViews,
			TotalEngagements: r.TotalEngagements,
		}
	}
	return metrics, nil
}
