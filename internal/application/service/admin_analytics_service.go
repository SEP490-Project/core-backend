package service

import (
	"context"
	"core-backend/internal/application/dto/dtos"
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

// GetDashboard returns the complete Admin dashboard using optimized batch queries
func (s *adminAnalyticsService) GetDashboard(ctx context.Context, req *requests.AdminDashboardRequest) (*responses.AdminDashboardResponse, error) {
	startDate, endDate := req.GetDateRange()

	// Calculate month boundaries for user metrics
	now := time.Now()
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	monthEnd := monthStart.AddDate(0, 1, 0)

	var mu sync.Mutex
	dashboard := &responses.AdminDashboardResponse{
		Period: responses.PeriodInfo{
			StartDate: startDate,
			EndDate:   endDate,
		},
	}

	// Execute 7 optimized batch queries in parallel (instead of 42+ individual queries)
	err := utils.RunParallel(ctx, 7,
		// Query 1: All user metrics in ONE query
		func(ctx context.Context) error {
			users, err := s.analyticsRepo.GetDashboardUsersMetrics(ctx, 30, monthStart, monthEnd)
			if err != nil {
				zap.L().Warn("Failed to get users metrics", zap.Error(err))
				return nil
			}
			mu.Lock()
			dashboard.Overview.TotalUsers = users.TotalUsers
			dashboard.Overview.ActiveUsers = users.ActiveUsers
			dashboard.UsersBreakdown = responses.UsersBreakdown{
				Admin:          users.Admin,
				MarketingStaff: users.MarketingStaff,
				SalesStaff:     users.SalesStaff,
				ContentStaff:   users.ContentStaff,
				BrandPartner:   users.BrandPartner,
				Customer:       users.Customer,
				TotalActive:    users.ActiveUsers,
				NewThisMonth:   users.NewThisMonth,
			}
			mu.Unlock()
			return nil
		},

		// Query 2: All contract metrics in ONE query
		func(ctx context.Context) error {
			contracts, err := s.analyticsRepo.GetDashboardContractsMetrics(ctx)
			if err != nil {
				zap.L().Warn("Failed to get contracts metrics", zap.Error(err))
				return nil
			}
			mu.Lock()
			dashboard.Overview.TotalContracts = contracts.TotalContracts
			dashboard.Overview.ActiveContracts = contracts.Active
			dashboard.ContractsSummary = responses.ContractsSummary{
				TotalContracts:  contracts.TotalContracts,
				Draft:           contracts.Draft,
				Approved:        contracts.Approved,
				Active:          contracts.Active,
				Completed:       contracts.Completed,
				Terminated:      contracts.Terminated,
				TotalValue:      contracts.TotalValue,
				CollectedAmount: contracts.CollectedAmount,
				PendingAmount:   contracts.PendingAmount,
			}
			mu.Unlock()
			return nil
		},

		// Query 3: All campaign metrics in ONE query
		func(ctx context.Context) error {
			campaigns, err := s.analyticsRepo.GetDashboardCampaignsMetrics(ctx)
			if err != nil {
				zap.L().Warn("Failed to get campaigns metrics", zap.Error(err))
				return nil
			}
			mu.Lock()
			dashboard.Overview.TotalCampaigns = campaigns.TotalCampaigns
			dashboard.Overview.ActiveCampaigns = campaigns.Running
			dashboard.CampaignsSummary = responses.AdminCampaignsSummary{
				TotalCampaigns: campaigns.TotalCampaigns,
				Draft:          campaigns.Draft,
				Running:        campaigns.Running,
				Completed:      campaigns.Completed,
				Cancelled:      campaigns.Cancelled,
				ContentCreated: campaigns.ContentCreated,
				ContentPosted:  campaigns.ContentPosted,
			}
			mu.Unlock()
			return nil
		},

		// Query 4: All brand metrics in ONE query
		func(ctx context.Context) error {
			brands, err := s.analyticsRepo.GetDashboardBrandsMetrics(ctx)
			if err != nil {
				zap.L().Warn("Failed to get brands metrics", zap.Error(err))
				return nil
			}
			mu.Lock()
			dashboard.Overview.TotalBrands = brands.TotalBrands
			dashboard.Overview.ActiveBrands = brands.ActiveBrands
			mu.Unlock()
			return nil
		},

		// Query 5: All order metrics in ONE query
		func(ctx context.Context) error {
			orders, err := s.analyticsRepo.GetDashboardOrdersMetrics(ctx, &startDate, &endDate)
			if err != nil {
				zap.L().Warn("Failed to get orders metrics", zap.Error(err))
				return nil
			}
			mu.Lock()
			dashboard.Overview.TotalOrders = orders.TotalOrders
			dashboard.Overview.MonthlyOrders = orders.MonthlyOrders
			mu.Unlock()
			return nil
		},

		// Query 6: All revenue metrics in ONE query
		func(ctx context.Context) error {
			revenue, err := s.analyticsRepo.GetDashboardRevenueMetrics(ctx, &startDate, &endDate)
			if err != nil {
				zap.L().Warn("Failed to get revenue metrics", zap.Error(err))
				return nil
			}
			mu.Lock()
			dashboard.Overview.TotalRevenue = revenue.TotalRevenue
			dashboard.Overview.MonthlyRevenue = revenue.MonthlyRevenue
			dashboard.RevenueBreakdown = responses.AdminRevenueBreakdown{
				AdvertisingRevenue:     revenue.AdvertisingRevenue,
				AffiliateRevenue:       revenue.AffiliateRevenue,
				AmbassadorRevenue:      revenue.AmbassadorRevenue,
				CoProducingRevenue:     revenue.CoProducingRevenue,
				StandardProductRevenue: revenue.StandardProductRevenue,
				LimitedProductRevenue:  revenue.LimitedProductRevenue,
				TotalContractRevenue:   revenue.AdvertisingRevenue + revenue.AffiliateRevenue + revenue.AmbassadorRevenue + revenue.CoProducingRevenue,
				TotalProductRevenue:    revenue.StandardProductRevenue + revenue.LimitedProductRevenue,
				TotalRevenue:           revenue.MonthlyRevenue,
			}
			mu.Unlock()
			return nil
		},

		// Query 7: Growth trend (already optimized as single CTE query)
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
			count, _ := s.analyticsRepo.GetContractCountByStatus(ctx, enum.ContractStatusDraft.String())
			mu.Lock()
			summary.Draft = count
			mu.Unlock()
			return nil
		},
		func(ctx context.Context) error {
			count, _ := s.analyticsRepo.GetContractCountByStatus(ctx, enum.ContractStatusApproved.String())
			mu.Lock()
			summary.Approved = count
			mu.Unlock()
			return nil
		},
		func(ctx context.Context) error {
			count, _ := s.analyticsRepo.GetContractCountByStatus(ctx, enum.ContractStatusActive.String())
			mu.Lock()
			summary.Active = count
			mu.Unlock()
			return nil
		},
		func(ctx context.Context) error {
			count, _ := s.analyticsRepo.GetContractCountByStatus(ctx, enum.ContractStatusCompleted.String())
			mu.Lock()
			summary.Completed = count
			mu.Unlock()
			return nil
		},
		func(ctx context.Context) error {
			count, _ := s.analyticsRepo.GetContractCountByStatus(ctx, enum.ContractStatusTerminated.String())
			mu.Lock()
			summary.Terminated = count
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
	_ = utils.RunParallel(ctx, 5,
		func(ctx context.Context) error {
			count, _ := s.analyticsRepo.GetTotalCampaignsCount(ctx)
			mu.Lock()
			summary.TotalCampaigns = count
			mu.Unlock()
			return nil
		},
		func(ctx context.Context) error {
			count, _ := s.analyticsRepo.GetCampaignCountByStatus(ctx, enum.CampaignDraft.String())
			mu.Lock()
			summary.Draft = count
			mu.Unlock()
			return nil
		},
		func(ctx context.Context) error {
			count, _ := s.analyticsRepo.GetCampaignCountByStatus(ctx, enum.CampaignRunning.String())
			mu.Lock()
			summary.Running = count
			mu.Unlock()
			return nil
		},
		func(ctx context.Context) error {
			count, _ := s.analyticsRepo.GetCampaignCountByStatus(ctx, enum.CampaignCompleted.String())
			mu.Lock()
			summary.Completed = count
			mu.Unlock()
			return nil
		},
		func(ctx context.Context) error {
			count, _ := s.analyticsRepo.GetCampaignCountByStatus(ctx, enum.CampaignCancelled.String())
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

// region: ============== Private Helper Methods ==============

// getUsersBreakdown returns user count breakdown by role
func (s *adminAnalyticsService) getUsersBreakdown(ctx context.Context) (*responses.UsersBreakdown, error) {
	breakdown := &responses.UsersBreakdown{}

	var mu sync.Mutex
	_ = utils.RunParallel(ctx, 8,
		func(ctx context.Context) error {
			count, _ := s.analyticsRepo.GetUserCountByRole(ctx, enum.UserRoleAdmin.String())
			mu.Lock()
			breakdown.Admin = count
			mu.Unlock()
			return nil
		},
		func(ctx context.Context) error {
			count, _ := s.analyticsRepo.GetUserCountByRole(ctx, enum.UserRoleMarketingStaff.String())
			mu.Lock()
			breakdown.MarketingStaff = count
			mu.Unlock()
			return nil
		},
		func(ctx context.Context) error {
			count, _ := s.analyticsRepo.GetUserCountByRole(ctx, enum.UserRoleSalesStaff.String())
			mu.Lock()
			breakdown.SalesStaff = count
			mu.Unlock()
			return nil
		},
		func(ctx context.Context) error {
			count, _ := s.analyticsRepo.GetUserCountByRole(ctx, enum.UserRoleContentStaff.String())
			mu.Lock()
			breakdown.ContentStaff = count
			mu.Unlock()
			return nil
		},
		func(ctx context.Context) error {
			count, _ := s.analyticsRepo.GetUserCountByRole(ctx, enum.UserRoleBrandPartner.String())
			mu.Lock()
			breakdown.BrandPartner = count
			mu.Unlock()
			return nil
		},
		func(ctx context.Context) error {
			count, _ := s.analyticsRepo.GetUserCountByRole(ctx, enum.UserRoleCustomer.String())
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
			revenue, _ := s.analyticsRepo.GetPlatformRevenueByContractType(ctx, enum.ContractTypeAdvertising.String(), startDate, endDate)
			mu.Lock()
			breakdown.AdvertisingRevenue = revenue
			mu.Unlock()
			return nil
		},
		func(ctx context.Context) error {
			revenue, _ := s.analyticsRepo.GetPlatformRevenueByContractType(ctx, enum.ContractTypeAffiliate.String(), startDate, endDate)
			mu.Lock()
			breakdown.AffiliateRevenue = revenue
			mu.Unlock()
			return nil
		},
		func(ctx context.Context) error {
			revenue, _ := s.analyticsRepo.GetPlatformRevenueByContractType(ctx, enum.ContractTypeAmbassador.String(), startDate, endDate)
			mu.Lock()
			breakdown.AmbassadorRevenue = revenue
			mu.Unlock()
			return nil
		},
		func(ctx context.Context) error {
			revenue, _ := s.analyticsRepo.GetPlatformRevenueByContractType(ctx, enum.ContractTypeCoProduce.String(), startDate, endDate)
			mu.Lock()
			breakdown.CoProducingRevenue = revenue
			mu.Unlock()
			return nil
		},
		func(ctx context.Context) error {
			revenue, _ := s.analyticsRepo.GetPlatformProductRevenue(ctx, enum.ProductTypeStandard.String(), startDate, endDate)
			mu.Lock()
			breakdown.StandardProductRevenue = revenue
			mu.Unlock()
			return nil
		},
		func(ctx context.Context) error {
			revenue, _ := s.analyticsRepo.GetPlatformProductRevenue(ctx, enum.ProductTypeLimited.String(), startDate, endDate)
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

// endregion
