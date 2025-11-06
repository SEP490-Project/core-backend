package service

import (
	"context"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/application/interfaces/iservice"
	"core-backend/pkg/utils"
	"sync"

	"go.uber.org/zap"
)

type marketingAnalyticsService struct {
	analyticsRepo irepository.MarketingAnalyticsRepository
}

// NewMarketingAnalyticsService creates a new marketing analytics service
func NewMarketingAnalyticsService(
	analyticsRepo irepository.MarketingAnalyticsRepository,
) iservice.MarketingAnalyticsService {
	return &marketingAnalyticsService{
		analyticsRepo: analyticsRepo,
	}
}

// GetActiveBrandsCount returns the count of active brands
func (s *marketingAnalyticsService) GetActiveBrandsCount(ctx context.Context) (int64, error) {
	count, err := s.analyticsRepo.GetActiveBrandsCount(ctx)
	if err != nil {
		zap.L().Error("Failed to get active brands count", zap.Error(err))
		return 0, err
	}
	return count, nil
}

// GetActiveCampaignsCount returns the count of active campaigns
func (s *marketingAnalyticsService) GetActiveCampaignsCount(ctx context.Context) (int64, error) {
	count, err := s.analyticsRepo.GetActiveCampaignsCount(ctx)
	if err != nil {
		zap.L().Error("Failed to get active campaigns count", zap.Error(err))
		return 0, err
	}
	return count, nil
}

// GetDraftCampaignsCount returns the count of draft campaigns with contracts
func (s *marketingAnalyticsService) GetDraftCampaignsCount(ctx context.Context) (int64, error) {
	count, err := s.analyticsRepo.GetDraftCampaignsCount(ctx)
	if err != nil {
		zap.L().Error("Failed to get draft campaigns count", zap.Error(err))
		return 0, err
	}
	return count, nil
}

// GetMonthlyContractRevenue returns total revenue from paid contract payments for specified month
func (s *marketingAnalyticsService) GetMonthlyContractRevenue(ctx context.Context, req *requests.MonthlyRevenueRequest) (float64, error) {
	revenue, err := s.analyticsRepo.GetMonthlyContractRevenue(ctx, req.Year, req.Month)
	if err != nil {
		zap.L().Error("Failed to get monthly contract revenue",
			zap.Int("year", req.Year),
			zap.Int("month", req.Month),
			zap.Error(err))
		return 0, err
	}
	return revenue, nil
}

// GetTopBrandsByRevenue returns top 4 brands by total revenue (contracts + products)
func (s *marketingAnalyticsService) GetTopBrandsByRevenue(ctx context.Context, filter *requests.TimeFilter) ([]responses.BrandRevenueResponse, error) {
	brands, err := s.analyticsRepo.GetTopBrandsByRevenue(ctx, filter)
	if err != nil {
		zap.L().Error("Failed to get top brands by revenue",
			zap.String("filter_type", filter.FilterType),
			zap.Error(err))
		return nil, err
	}
	return brands, nil
}

// GetRevenueByContractType returns revenue breakdown by contract type and standard products
func (s *marketingAnalyticsService) GetRevenueByContractType(ctx context.Context, filter *requests.TimeFilter) (*responses.RevenueByTypeResponse, error) {
	revenueBreakdown, err := s.analyticsRepo.GetRevenueByContractType(ctx, filter)
	if err != nil {
		zap.L().Error("Failed to get revenue by contract type",
			zap.String("filter_type", filter.FilterType),
			zap.Error(err))
		return nil, err
	}
	return revenueBreakdown, nil
}

// GetUpcomingDeadlineCampaigns returns campaigns approaching deadline
func (s *marketingAnalyticsService) GetUpcomingDeadlineCampaigns(ctx context.Context, filter *requests.UpcomingDeadlineFilter) ([]responses.UpcomingCampaignResponse, error) {
	daysBeforeDeadline := filter.GetDays()
	campaigns, err := s.analyticsRepo.GetUpcomingDeadlineCampaigns(ctx, daysBeforeDeadline)
	if err != nil {
		zap.L().Error("Failed to get upcoming deadline campaigns",
			zap.Int("days_before_deadline", daysBeforeDeadline),
			zap.Error(err))
		return nil, err
	}
	return campaigns, nil
}

// GetDashboard returns aggregated analytics data for marketing dashboard
// Uses parallel execution with mutex protection for safe concurrent access
func (s *marketingAnalyticsService) GetDashboard(ctx context.Context, filter *requests.DashboardFilter) (*responses.MarketingDashboardResponse, error) {
	year, month := filter.GetYearMonth()

	// Prepare result structure with mutex for concurrent writes
	var mu sync.Mutex
	dashboard := &responses.MarketingDashboardResponse{
		RevenueYear:  year,
		RevenueMonth: month,
	}

	// Create time filter for revenue queries
	timeFilter := &requests.TimeFilter{
		FilterType: "MONTH",
		Year:       year,
		Month:      &month,
	}

	// Create upcoming deadline filter
	upcomingDeadlineFilter := &requests.UpcomingDeadlineFilter{
		Days: 0, // Will default to 10 in GetDays()
	}

	// Execute 7 queries in parallel with concurrency limit
	err := utils.RunParallel(ctx, 7, // Allow 7 concurrent queries
		// Query 1: Active Brands Count
		func(ctx context.Context) error {
			count, err := s.analyticsRepo.GetActiveBrandsCount(ctx)
			if err != nil {
				zap.L().Error("Dashboard: Failed to get active brands count", zap.Error(err))
				return err
			}
			mu.Lock()
			dashboard.ActiveBrands = count
			mu.Unlock()
			return nil
		},

		// Query 2: Active Campaigns Count
		func(ctx context.Context) error {
			count, err := s.analyticsRepo.GetActiveCampaignsCount(ctx)
			if err != nil {
				zap.L().Error("Dashboard: Failed to get active campaigns count", zap.Error(err))
				return err
			}
			mu.Lock()
			dashboard.ActiveCampaigns = count
			mu.Unlock()
			return nil
		},

		// Query 3: Draft Campaigns Count
		func(ctx context.Context) error {
			count, err := s.analyticsRepo.GetDraftCampaignsCount(ctx)
			if err != nil {
				zap.L().Error("Dashboard: Failed to get draft campaigns count", zap.Error(err))
				return err
			}
			mu.Lock()
			dashboard.DraftCampaigns = count
			mu.Unlock()
			return nil
		},

		// Query 4: Monthly Contract Revenue
		func(ctx context.Context) error {
			revenue, err := s.analyticsRepo.GetMonthlyContractRevenue(ctx, year, month)
			if err != nil {
				zap.L().Error("Dashboard: Failed to get monthly contract revenue",
					zap.Int("year", year),
					zap.Int("month", month),
					zap.Error(err))
				return err
			}
			mu.Lock()
			dashboard.MonthlyRevenue = revenue
			mu.Unlock()
			return nil
		},

		// Query 5: Top Brands By Revenue
		func(ctx context.Context) error {
			brands, err := s.analyticsRepo.GetTopBrandsByRevenue(ctx, timeFilter)
			if err != nil {
				zap.L().Error("Dashboard: Failed to get top brands by revenue", zap.Error(err))
				return err
			}
			mu.Lock()
			dashboard.TopBrands = brands
			mu.Unlock()
			return nil
		},

		// Query 6: Revenue By Contract Type
		func(ctx context.Context) error {
			revenueBreakdown, err := s.analyticsRepo.GetRevenueByContractType(ctx, timeFilter)
			if err != nil {
				zap.L().Error("Dashboard: Failed to get revenue by contract type", zap.Error(err))
				return err
			}
			mu.Lock()
			dashboard.RevenueByType = *revenueBreakdown
			mu.Unlock()
			return nil
		},

		// Query 7: Upcoming Deadline Campaigns
		func(ctx context.Context) error {
			campaigns, err := s.analyticsRepo.GetUpcomingDeadlineCampaigns(ctx, upcomingDeadlineFilter.GetDays())
			if err != nil {
				zap.L().Error("Dashboard: Failed to get upcoming deadline campaigns", zap.Error(err))
				return err
			}
			mu.Lock()
			dashboard.UpcomingDeadlines = campaigns
			mu.Unlock()
			return nil
		},
	)

	if err != nil {
		zap.L().Error("Dashboard: Parallel query execution failed",
			zap.Int("year", year),
			zap.Int("month", month),
			zap.Error(err))
		return nil, err
	}

	zap.L().Info("Dashboard data retrieved successfully",
		zap.Int("year", year),
		zap.Int("month", month),
		zap.Int64("active_brands", dashboard.ActiveBrands),
		zap.Int64("active_campaigns", dashboard.ActiveCampaigns),
		zap.Float64("monthly_revenue", dashboard.MonthlyRevenue))

	return dashboard, nil
}
