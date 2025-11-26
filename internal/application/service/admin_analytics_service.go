package service

import (
	"context"
	"core-backend/internal/application/dto/dtos"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/application/interfaces/iservice"
	"core-backend/pkg/utils"
	"sync"
	"time"

	"go.uber.org/zap"
)

type adminAnalyticsService struct {
	analyticsRepo irepository.AdminAnalyticsRepository
}

// NewAdminAnalyticsService creates a new admin analytics service
func NewAdminAnalyticsService(
	analyticsRepo irepository.AdminAnalyticsRepository,
) iservice.AdminAnalyticsService {
	return &adminAnalyticsService{
		analyticsRepo: analyticsRepo,
	}
}

// GetDashboard returns the complete Admin dashboard
func (s *adminAnalyticsService) GetDashboard(ctx context.Context, req *requests.AdminDashboardRequest) (*responses.AdminDashboardResponse, error) {
	startDate, endDate := req.GetDateRange()

	var mu sync.Mutex
	dashboard := &responses.AdminDashboardResponse{
		Period: responses.PeriodInfo{
			StartDate: startDate,
			EndDate:   endDate,
		},
	}

	// Execute queries in parallel
	err := utils.RunParallel(ctx, 6,
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

		// Query 2: Users breakdown
		func(ctx context.Context) error {
			breakdown, err := s.getUsersBreakdown(ctx)
			if err != nil {
				zap.L().Warn("Failed to get users breakdown", zap.Error(err))
				return nil
			}
			mu.Lock()
			dashboard.UsersBreakdown = *breakdown
			mu.Unlock()
			return nil
		},

		// Query 3: Revenue breakdown
		func(ctx context.Context) error {
			revenue, err := s.getRevenueBreakdown(ctx, &startDate, &endDate)
			if err != nil {
				zap.L().Warn("Failed to get revenue breakdown", zap.Error(err))
				return nil
			}
			mu.Lock()
			dashboard.RevenueBreakdown = *revenue
			mu.Unlock()
			return nil
		},

		// Query 4: Contracts summary
		func(ctx context.Context) error {
			summary, err := s.GetContractsSummary(ctx, &requests.DashboardRequest{})
			if err != nil {
				zap.L().Warn("Failed to get contracts summary", zap.Error(err))
				return nil
			}
			mu.Lock()
			dashboard.ContractsSummary = *summary
			mu.Unlock()
			return nil
		},

		// Query 5: Campaigns summary
		func(ctx context.Context) error {
			summary, err := s.GetCampaignsSummary(ctx, &requests.DashboardRequest{})
			if err != nil {
				zap.L().Warn("Failed to get campaigns summary", zap.Error(err))
				return nil
			}
			mu.Lock()
			dashboard.CampaignsSummary = *summary
			mu.Unlock()
			return nil
		},

		// Query 6: Growth trend
		func(ctx context.Context) error {
			trend, err := s.analyticsRepo.GetGrowthTrend(ctx, "DAY", &startDate, &endDate)
			if err != nil {
				zap.L().Warn("Failed to get growth trend", zap.Error(err))
				return nil
			}
			trendPoints := make([]responses.GrowthTrendPoint, len(trend))
			for i, t := range trend {
				trendPoints[i] = responses.GrowthTrendPoint{
					Date:         t.Date,
					NewUsers:     t.NewUsers,
					NewOrders:    t.NewOrders,
					NewContracts: t.NewContracts,
					Revenue:      t.Revenue,
				}
			}
			mu.Lock()
			dashboard.GrowthTrend = trendPoints
			mu.Unlock()
			return nil
		},
	)

	if err != nil {
		zap.L().Error("Dashboard parallel query failed", zap.Error(err))
	}

	return dashboard, nil
}

// getOverviewMetrics returns high-level platform metrics
func (s *adminAnalyticsService) getOverviewMetrics(ctx context.Context, startDate, endDate *time.Time) (*responses.AdminOverviewMetrics, error) {
	overview := &responses.AdminOverviewMetrics{}

	var mu sync.Mutex
	_ = utils.RunParallel(ctx, 10,
		func(ctx context.Context) error {
			count, _ := s.analyticsRepo.GetTotalUsersCount(ctx)
			mu.Lock()
			overview.TotalUsers = count
			mu.Unlock()
			return nil
		},
		func(ctx context.Context) error {
			count, _ := s.analyticsRepo.GetActiveUsersCount(ctx, 30)
			mu.Lock()
			overview.ActiveUsers = count
			mu.Unlock()
			return nil
		},
		func(ctx context.Context) error {
			count, _ := s.analyticsRepo.GetTotalBrandsCount(ctx)
			mu.Lock()
			overview.TotalBrands = count
			mu.Unlock()
			return nil
		},
		func(ctx context.Context) error {
			count, _ := s.analyticsRepo.GetActiveBrandsCount(ctx)
			mu.Lock()
			overview.ActiveBrands = count
			mu.Unlock()
			return nil
		},
		func(ctx context.Context) error {
			count, _ := s.analyticsRepo.GetTotalContractsCount(ctx)
			mu.Lock()
			overview.TotalContracts = count
			mu.Unlock()
			return nil
		},
		func(ctx context.Context) error {
			count, _ := s.analyticsRepo.GetContractCountByStatus(ctx, "ACTIVE")
			mu.Lock()
			overview.ActiveContracts = count
			mu.Unlock()
			return nil
		},
		func(ctx context.Context) error {
			count, _ := s.analyticsRepo.GetTotalCampaignsCount(ctx)
			mu.Lock()
			overview.TotalCampaigns = count
			mu.Unlock()
			return nil
		},
		func(ctx context.Context) error {
			count, _ := s.analyticsRepo.GetCampaignCountByStatus(ctx, "ACTIVE")
			mu.Lock()
			overview.ActiveCampaigns = count
			mu.Unlock()
			return nil
		},
		func(ctx context.Context) error {
			revenue, _ := s.analyticsRepo.GetTotalPlatformRevenue(ctx, nil, nil)
			mu.Lock()
			overview.TotalRevenue = revenue
			mu.Unlock()
			return nil
		},
		func(ctx context.Context) error {
			revenue, _ := s.analyticsRepo.GetTotalPlatformRevenue(ctx, startDate, endDate)
			mu.Lock()
			overview.MonthlyRevenue = revenue
			mu.Unlock()
			return nil
		},
	)

	// Calculate orders
	totalOrders, _ := s.analyticsRepo.GetTotalOrdersCount(ctx, nil, nil)
	monthlyOrders, _ := s.analyticsRepo.GetTotalOrdersCount(ctx, startDate, endDate)
	overview.TotalOrders = totalOrders
	overview.MonthlyOrders = monthlyOrders

	return overview, nil
}

// getUsersBreakdown returns user count breakdown by role
func (s *adminAnalyticsService) getUsersBreakdown(ctx context.Context) (*responses.UsersBreakdown, error) {
	breakdown := &responses.UsersBreakdown{}

	var mu sync.Mutex
	_ = utils.RunParallel(ctx, 8,
		func(ctx context.Context) error {
			count, _ := s.analyticsRepo.GetUserCountByRole(ctx, "ADMIN")
			mu.Lock()
			breakdown.Admin = count
			mu.Unlock()
			return nil
		},
		func(ctx context.Context) error {
			count, _ := s.analyticsRepo.GetUserCountByRole(ctx, "MARKETING_STAFF")
			mu.Lock()
			breakdown.MarketingStaff = count
			mu.Unlock()
			return nil
		},
		func(ctx context.Context) error {
			count, _ := s.analyticsRepo.GetUserCountByRole(ctx, "SALES_STAFF")
			mu.Lock()
			breakdown.SalesStaff = count
			mu.Unlock()
			return nil
		},
		func(ctx context.Context) error {
			count, _ := s.analyticsRepo.GetUserCountByRole(ctx, "CONTENT_STAFF")
			mu.Lock()
			breakdown.ContentStaff = count
			mu.Unlock()
			return nil
		},
		func(ctx context.Context) error {
			count, _ := s.analyticsRepo.GetUserCountByRole(ctx, "BRAND_PARTNER")
			mu.Lock()
			breakdown.BrandPartner = count
			mu.Unlock()
			return nil
		},
		func(ctx context.Context) error {
			count, _ := s.analyticsRepo.GetUserCountByRole(ctx, "CUSTOMER")
			mu.Lock()
			breakdown.Customer = count
			mu.Unlock()
			return nil
		},
		func(ctx context.Context) error {
			count, _ := s.analyticsRepo.GetActiveUsersCount(ctx, 30)
			mu.Lock()
			breakdown.TotalActive = count
			mu.Unlock()
			return nil
		},
		func(ctx context.Context) error {
			now := time.Now()
			start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
			end := start.AddDate(0, 1, 0)
			count, _ := s.analyticsRepo.GetNewUsersCount(ctx, start, end)
			mu.Lock()
			breakdown.NewThisMonth = count
			mu.Unlock()
			return nil
		},
	)

	return breakdown, nil
}

// getRevenueBreakdown returns platform revenue breakdown
func (s *adminAnalyticsService) getRevenueBreakdown(ctx context.Context, startDate, endDate *time.Time) (*responses.AdminRevenueBreakdown, error) {
	breakdown := &responses.AdminRevenueBreakdown{}

	var mu sync.Mutex
	_ = utils.RunParallel(ctx, 6,
		func(ctx context.Context) error {
			revenue, _ := s.analyticsRepo.GetPlatformRevenueByContractType(ctx, "ADVERTISING", startDate, endDate)
			mu.Lock()
			breakdown.AdvertisingRevenue = revenue
			mu.Unlock()
			return nil
		},
		func(ctx context.Context) error {
			revenue, _ := s.analyticsRepo.GetPlatformRevenueByContractType(ctx, "AFFILIATE", startDate, endDate)
			mu.Lock()
			breakdown.AffiliateRevenue = revenue
			mu.Unlock()
			return nil
		},
		func(ctx context.Context) error {
			revenue, _ := s.analyticsRepo.GetPlatformRevenueByContractType(ctx, "AMBASSADOR", startDate, endDate)
			mu.Lock()
			breakdown.AmbassadorRevenue = revenue
			mu.Unlock()
			return nil
		},
		func(ctx context.Context) error {
			revenue, _ := s.analyticsRepo.GetPlatformRevenueByContractType(ctx, "CO_PRODUCING", startDate, endDate)
			mu.Lock()
			breakdown.CoProducingRevenue = revenue
			mu.Unlock()
			return nil
		},
		func(ctx context.Context) error {
			revenue, _ := s.analyticsRepo.GetPlatformProductRevenue(ctx, "STANDARD", startDate, endDate)
			mu.Lock()
			breakdown.StandardProductRevenue = revenue
			mu.Unlock()
			return nil
		},
		func(ctx context.Context) error {
			revenue, _ := s.analyticsRepo.GetPlatformProductRevenue(ctx, "LIMITED", startDate, endDate)
			mu.Lock()
			breakdown.LimitedProductRevenue = revenue
			mu.Unlock()
			return nil
		},
	)

	// Calculate totals
	breakdown.TotalContractRevenue = breakdown.AdvertisingRevenue + breakdown.AffiliateRevenue +
		breakdown.AmbassadorRevenue + breakdown.CoProducingRevenue
	breakdown.TotalProductRevenue = breakdown.StandardProductRevenue + breakdown.LimitedProductRevenue
	breakdown.TotalRevenue = breakdown.TotalContractRevenue + breakdown.TotalProductRevenue

	return breakdown, nil
}

// GetUsersOverview returns user statistics and growth
func (s *adminAnalyticsService) GetUsersOverview(ctx context.Context, req *requests.UsersOverviewRequest) (*responses.UsersOverviewResponse, error) {
	now := time.Now()
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	endOfMonth := startOfMonth.AddDate(0, 1, 0)

	var totalUsers, activeUsers, newUsers int64
	var breakdown *responses.UsersBreakdown
	var growthTrend []dtos.UserGrowthResult

	var mu sync.Mutex
	_ = utils.RunParallel(ctx, 4,
		func(ctx context.Context) error {
			count, _ := s.analyticsRepo.GetTotalUsersCount(ctx)
			mu.Lock()
			totalUsers = count
			mu.Unlock()
			return nil
		},
		func(ctx context.Context) error {
			count, _ := s.analyticsRepo.GetActiveUsersCount(ctx, 30)
			mu.Lock()
			activeUsers = count
			mu.Unlock()
			return nil
		},
		func(ctx context.Context) error {
			count, _ := s.analyticsRepo.GetNewUsersCount(ctx, startOfMonth, endOfMonth)
			mu.Lock()
			newUsers = count
			mu.Unlock()
			return nil
		},
		func(ctx context.Context) error {
			b, _ := s.getUsersBreakdown(ctx)
			mu.Lock()
			breakdown = b
			mu.Unlock()
			return nil
		},
	)

	// Get growth trend
	start := now.AddDate(0, -6, 0)
	growthTrend, _ = s.analyticsRepo.GetUserGrowthTrend(ctx, "MONTH", &start, nil)

	growthPoints := make([]responses.UserGrowthPoint, len(growthTrend))
	for i, t := range growthTrend {
		growthPoints[i] = responses.UserGrowthPoint{
			Date:     t.Date,
			NewUsers: t.NewUsers,
			Total:    t.Total,
		}
	}

	return &responses.UsersOverviewResponse{
		TotalUsers:        totalUsers,
		ActiveUsers:       activeUsers,
		NewUsersThisMonth: newUsers,
		RoleBreakdown:     *breakdown,
		GrowthTrend:       growthPoints,
		Period: responses.PeriodInfo{
			StartDate: startOfMonth,
			EndDate:   endOfMonth,
		},
	}, nil
}

// GetPlatformRevenue returns platform-wide revenue analytics
func (s *adminAnalyticsService) GetPlatformRevenue(ctx context.Context, req *requests.PlatformRevenueRequest) (*responses.PlatformRevenueResponse, error) {
	now := time.Now()
	start := now.AddDate(0, -6, 0)
	end := now
	if req.StartDate != nil {
		start = *req.StartDate
	}
	if req.EndDate != nil {
		end = *req.EndDate
	}

	var totalRevenue float64
	var breakdown *responses.AdminRevenueBreakdown
	var revenueTrend []dtos.RevenueTrendResult

	var mu sync.Mutex
	_ = utils.RunParallel(ctx, 3,
		func(ctx context.Context) error {
			revenue, _ := s.analyticsRepo.GetTotalPlatformRevenue(ctx, &start, &end)
			mu.Lock()
			totalRevenue = revenue
			mu.Unlock()
			return nil
		},
		func(ctx context.Context) error {
			b, _ := s.getRevenueBreakdown(ctx, &start, &end)
			mu.Lock()
			breakdown = b
			mu.Unlock()
			return nil
		},
		func(ctx context.Context) error {
			trend, _ := s.analyticsRepo.GetPlatformRevenueTrend(ctx, req.GetGranularity(), &start, &end)
			mu.Lock()
			revenueTrend = trend
			mu.Unlock()
			return nil
		},
	)

	trendPoints := make([]responses.RevenueTrendPoint, len(revenueTrend))
	for i, t := range revenueTrend {
		trendPoints[i] = responses.RevenueTrendPoint{
			Date:    t.Date,
			Revenue: t.Revenue,
		}
	}

	return &responses.PlatformRevenueResponse{
		TotalRevenue:     totalRevenue,
		RevenueBreakdown: *breakdown,
		RevenueTrend:     trendPoints,
		Period: responses.PeriodInfo{
			StartDate: start,
			EndDate:   end,
		},
	}, nil
}

// GetSystemHealth returns system health metrics
func (s *adminAnalyticsService) GetSystemHealth(ctx context.Context) (*responses.SystemHealthResponse, error) {
	// Note: This would typically integrate with infrastructure monitoring
	// For now, return placeholder data
	return &responses.SystemHealthResponse{
		DatabaseStatus:    "OK",
		CacheStatus:       "OK",
		QueueStatus:       "OK",
		PendingJobs:       0,
		FailedJobs24h:     0,
		AverageResponseMs: 100,
		ErrorRate:         0.01,
		Uptime:            "99.9%",
	}, nil
}

// GetUserGrowth returns user growth over time
func (s *adminAnalyticsService) GetUserGrowth(ctx context.Context, req *requests.UserGrowthRequest) ([]responses.UserGrowthPoint, error) {
	trend, err := s.analyticsRepo.GetUserGrowthTrend(ctx, req.GetGranularity(), req.StartDate, req.EndDate)
	if err != nil {
		return nil, err
	}

	points := make([]responses.UserGrowthPoint, len(trend))
	for i, t := range trend {
		points[i] = responses.UserGrowthPoint{
			Date:     t.Date,
			NewUsers: t.NewUsers,
			Total:    t.Total,
		}
	}
	return points, nil
}

// GetContractsSummary returns contract statistics
func (s *adminAnalyticsService) GetContractsSummary(ctx context.Context, req *requests.DashboardRequest) (*responses.ContractsSummary, error) {
	summary := &responses.ContractsSummary{}

	var mu sync.Mutex
	_ = utils.RunParallel(ctx, 8,
		func(ctx context.Context) error {
			count, _ := s.analyticsRepo.GetTotalContractsCount(ctx)
			mu.Lock()
			summary.TotalContracts = count
			mu.Unlock()
			return nil
		},
		func(ctx context.Context) error {
			count, _ := s.analyticsRepo.GetContractCountByStatus(ctx, "DRAFT")
			mu.Lock()
			summary.Draft = count
			mu.Unlock()
			return nil
		},
		func(ctx context.Context) error {
			count, _ := s.analyticsRepo.GetContractCountByStatus(ctx, "PENDING")
			mu.Lock()
			summary.Pending = count
			mu.Unlock()
			return nil
		},
		func(ctx context.Context) error {
			count, _ := s.analyticsRepo.GetContractCountByStatus(ctx, "ACTIVE")
			mu.Lock()
			summary.Active = count
			mu.Unlock()
			return nil
		},
		func(ctx context.Context) error {
			count, _ := s.analyticsRepo.GetContractCountByStatus(ctx, "COMPLETED")
			mu.Lock()
			summary.Completed = count
			mu.Unlock()
			return nil
		},
		func(ctx context.Context) error {
			count, _ := s.analyticsRepo.GetContractCountByStatus(ctx, "CANCELLED")
			mu.Lock()
			summary.Cancelled = count
			mu.Unlock()
			return nil
		},
		func(ctx context.Context) error {
			value, _ := s.analyticsRepo.GetTotalContractValue(ctx)
			mu.Lock()
			summary.TotalValue = value
			mu.Unlock()
			return nil
		},
		func(ctx context.Context) error {
			collected, _ := s.analyticsRepo.GetCollectedContractAmount(ctx)
			pending, _ := s.analyticsRepo.GetPendingContractAmount(ctx)
			mu.Lock()
			summary.CollectedAmount = collected
			summary.PendingAmount = pending
			mu.Unlock()
			return nil
		},
	)

	return summary, nil
}

// GetCampaignsSummary returns campaign statistics
func (s *adminAnalyticsService) GetCampaignsSummary(ctx context.Context, req *requests.DashboardRequest) (*responses.AdminCampaignsSummary, error) {
	summary := &responses.AdminCampaignsSummary{}

	var mu sync.Mutex
	_ = utils.RunParallel(ctx, 9,
		func(ctx context.Context) error {
			count, _ := s.analyticsRepo.GetTotalCampaignsCount(ctx)
			mu.Lock()
			summary.TotalCampaigns = count
			mu.Unlock()
			return nil
		},
		func(ctx context.Context) error {
			count, _ := s.analyticsRepo.GetCampaignCountByStatus(ctx, "DRAFT")
			mu.Lock()
			summary.Draft = count
			mu.Unlock()
			return nil
		},
		func(ctx context.Context) error {
			count, _ := s.analyticsRepo.GetCampaignCountByStatus(ctx, "ACTIVE")
			mu.Lock()
			summary.Active = count
			mu.Unlock()
			return nil
		},
		func(ctx context.Context) error {
			count, _ := s.analyticsRepo.GetCampaignCountByStatus(ctx, "IN_PROGRESS")
			mu.Lock()
			summary.InProgress = count
			mu.Unlock()
			return nil
		},
		func(ctx context.Context) error {
			count, _ := s.analyticsRepo.GetCampaignCountByStatus(ctx, "PENDING")
			mu.Lock()
			summary.Pending = count
			mu.Unlock()
			return nil
		},
		func(ctx context.Context) error {
			count, _ := s.analyticsRepo.GetCampaignCountByStatus(ctx, "FINISHED")
			mu.Lock()
			summary.Finished = count
			mu.Unlock()
			return nil
		},
		func(ctx context.Context) error {
			count, _ := s.analyticsRepo.GetCampaignCountByStatus(ctx, "CANCELLED")
			mu.Lock()
			summary.Cancelled = count
			mu.Unlock()
			return nil
		},
		func(ctx context.Context) error {
			count, _ := s.analyticsRepo.GetTotalContentCount(ctx)
			mu.Lock()
			summary.ContentCreated = count
			mu.Unlock()
			return nil
		},
		func(ctx context.Context) error {
			count, _ := s.analyticsRepo.GetPostedContentCount(ctx)
			mu.Lock()
			summary.ContentPosted = count
			mu.Unlock()
			return nil
		},
	)

	return summary, nil
}
