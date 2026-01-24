package service

import (
	"context"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/application/interfaces/iservice"
	"core-backend/internal/domain/constant"
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

// GetGrossContractRevenue returns total revenue from paid contract payments (before refund deductions)
// Includes PAID + KOL_REFUND_APPROVED payments (full amounts)
func (s *marketingAnalyticsService) GetGrossContractRevenue(ctx context.Context, filter *requests.DashboardFilterRequest) (float64, error) {
	revenue, err := s.analyticsRepo.GetGrossContractRevenue(ctx, filter)
	if err != nil {
		zap.L().Error("Failed to get gross contract revenue",
			zap.String("period", filter.GetPresetLabel()),
			zap.Error(err))
		return 0, err
	}
	return revenue, nil
}

// GetNetContractRevenue returns net revenue after refund deductions
// Returns: grossRevenue (PAID + KOL_REFUND_APPROVED full amounts), netRevenue (after refund), totalRefunds
func (s *marketingAnalyticsService) GetNetContractRevenue(ctx context.Context, filter *requests.DashboardFilterRequest) (grossRevenue, netRevenue, totalRefunds float64, err error) {
	grossRevenue, netRevenue, totalRefunds, err = s.analyticsRepo.GetNetContractRevenue(ctx, filter)
	if err != nil {
		zap.L().Error("Failed to get net contract revenue",
			zap.String("period", filter.GetPresetLabel()),
			zap.Error(err))
		return 0, 0, 0, err
	}
	return grossRevenue, netRevenue, totalRefunds, nil
}

// GetTopBrandsByRevenue returns top brands by total revenue (contracts + products)
func (s *marketingAnalyticsService) GetTopBrandsByRevenue(ctx context.Context, filter *requests.DashboardFilterRequest) ([]responses.BrandRevenueResponse, error) {
	brands, err := s.analyticsRepo.GetTopBrandsByRevenue(ctx, filter)
	if err != nil {
		zap.L().Error("Failed to get top brands by revenue",
			zap.String("period", filter.GetPresetLabel()),
			zap.Error(err))
		return nil, err
	}
	return brands, nil
}

// GetRevenueByContractType returns revenue breakdown by contract type and standard products
func (s *marketingAnalyticsService) GetRevenueByContractType(ctx context.Context, filter *requests.DashboardFilterRequest) (*responses.RevenueByTypeResponse, error) {
	revenueBreakdown, err := s.analyticsRepo.GetRevenueByContractType(ctx, filter)
	if err != nil {
		zap.L().Error("Failed to get revenue by contract type",
			zap.String("period", filter.GetPresetLabel()),
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
func (s *marketingAnalyticsService) GetDashboard(ctx context.Context, filter *requests.DashboardFilterRequest) (*responses.MarketingDashboardResponse, error) {
	current, _ := filter.GetDateRanges()

	// Prepare result structure with mutex for concurrent writes
	var mu sync.Mutex
	dashboard := &responses.MarketingDashboardResponse{
		RevenueYear:  current.Start.Year(),
		RevenueMonth: int(current.Start.Month()),
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

		// Query 4: Net Contract Revenue (includes gross, net, and total refunds)
		func(ctx context.Context) error {
			gross, net, refunds, err := s.analyticsRepo.GetNetContractRevenue(ctx, filter)
			if err != nil {
				zap.L().Error("Dashboard: Failed to get net contract revenue",
					zap.String("period", filter.GetPresetLabel()),
					zap.Error(err))
				return err
			}
			mu.Lock()
			dashboard.GrossRevenue = gross
			dashboard.NetRevenue = net
			dashboard.TotalRefunds = refunds
			mu.Unlock()
			return nil
		},

		// Query 5: Top Brands By Revenue
		func(ctx context.Context) error {
			brands, err := s.analyticsRepo.GetTopBrandsByRevenue(ctx, filter)
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
			revenueBreakdown, err := s.analyticsRepo.GetRevenueByContractType(ctx, filter)
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
			zap.String("period", filter.GetPresetLabel()),
			zap.Error(err))
		return nil, err
	}

	zap.L().Info("Dashboard data retrieved successfully",
		zap.String("period", filter.GetPresetLabel()),
		zap.Int64("active_brands", dashboard.ActiveBrands),
		zap.Int64("active_campaigns", dashboard.ActiveCampaigns),
		zap.Float64("gross_revenue", dashboard.GrossRevenue),
		zap.Float64("net_revenue", dashboard.NetRevenue),
		zap.Float64("total_refunds", dashboard.TotalRefunds))

	return dashboard, nil
}

// GetContractStatusDistribution returns contract counts grouped by status categories
func (s *marketingAnalyticsService) GetContractStatusDistribution(ctx context.Context, filter *requests.DashboardFilterRequest) (*responses.ContractStatusDistributionResponse, error) {
	result, err := s.analyticsRepo.GetContractStatusDistribution(ctx, filter)
	if err != nil {
		zap.L().Error("Failed to get contract status distribution", zap.Error(err))
		return nil, err
	}
	return result, nil
}

// GetTaskStatusDistribution returns task counts grouped by status
func (s *marketingAnalyticsService) GetTaskStatusDistribution(ctx context.Context, filter *requests.DashboardFilterRequest) (*responses.TaskStatusDistributionResponse, error) {
	result, err := s.analyticsRepo.GetTaskStatusDistribution(ctx, filter)
	if err != nil {
		zap.L().Error("Failed to get task status distribution", zap.Error(err))
		return nil, err
	}
	return result, nil
}

// GetRevenueOverTime returns revenue breakdown by source over time for combo chart
// TODO: Implement affiliate revenue calculation based on click_events and tiered pricing
// See api-specification.md section 3.3 and contract_payment_service.go -> calculateAffiliatePayment()
// For now, only contract base revenue and limited product revenue are included
func (s *marketingAnalyticsService) GetRevenueOverTime(ctx context.Context, filter *requests.DashboardFilterRequest) (*responses.RevenueOverTimeResponse, error) {
	granularity := filter.GetTrendGranularity()

	// Fetch contract base revenue and limited product revenue in parallel
	// TODO: Add third query for affiliate revenue from click_events (tiered calculation)
	var contractRevenue []responses.RevenueOverTimePoint
	var productRevenue []responses.RevenueOverTimePoint
	var mu sync.Mutex

	err := utils.RunParallel(ctx, 2,
		func(ctx context.Context) error {
			result, err := s.analyticsRepo.GetContractBaseRevenueOverTime(ctx, filter, granularity)
			if err != nil {
				return err
			}
			mu.Lock()
			contractRevenue = result
			mu.Unlock()
			return nil
		},
		func(ctx context.Context) error {
			result, err := s.analyticsRepo.GetLimitedProductRevenueOverTime(ctx, filter, granularity)
			if err != nil {
				return err
			}
			mu.Lock()
			productRevenue = result
			mu.Unlock()
			return nil
		},
	)

	if err != nil {
		zap.L().Error("Failed to get revenue over time", zap.Error(err))
		return nil, err
	}

	// Merge the two revenue streams into a single time series
	dataMap := make(map[string]*responses.RevenueOverTimePoint)

	for _, point := range contractRevenue {
		key := point.Date.Format("2006-01-02T15:04:05")
		if existing, ok := dataMap[key]; ok {
			existing.ContractBaseRevenue = point.ContractBaseRevenue
		} else {
			dataMap[key] = &responses.RevenueOverTimePoint{
				Date:                point.Date,
				ContractBaseRevenue: point.ContractBaseRevenue,
			}
		}
	}

	for _, point := range productRevenue {
		key := point.Date.Format("2006-01-02T15:04:05")
		if existing, ok := dataMap[key]; ok {
			existing.LimitedProductRevenue = point.LimitedProductRevenue
		} else {
			dataMap[key] = &responses.RevenueOverTimePoint{
				Date:                  point.Date,
				LimitedProductRevenue: point.LimitedProductRevenue,
			}
		}
	}

	// Convert map to sorted slice and calculate totals
	data := make([]responses.RevenueOverTimePoint, 0, len(dataMap))
	var summary responses.RevenueOverTimeSummary

	for _, point := range dataMap {
		point.TotalRevenue = point.ContractBaseRevenue + point.AffiliateRevenue + point.LimitedProductRevenue
		data = append(data, *point)
		summary.TotalContractBaseRevenue += point.ContractBaseRevenue
		summary.TotalAffiliateRevenue += point.AffiliateRevenue
		summary.TotalLimitedProductRevenue += point.LimitedProductRevenue
	}

	// Sort by date
	sortRevenueData(data)

	summary.GrandTotalRevenue = summary.TotalContractBaseRevenue + summary.TotalAffiliateRevenue + summary.TotalLimitedProductRevenue

	return &responses.RevenueOverTimeResponse{
		Data:        data,
		Granularity: string(granularity),
		Period:      filter.GetPeriodInfo(),
		Summary:     summary,
	}, nil
}

// sortRevenueData sorts revenue data points by date ascending
func sortRevenueData(data []responses.RevenueOverTimePoint) {
	for i := 0; i < len(data)-1; i++ {
		for j := i + 1; j < len(data); j++ {
			if data[i].Date.After(data[j].Date) {
				data[i], data[j] = data[j], data[i]
			}
		}
	}
}

// GetRefundViolationStats returns system-wide refund and violation statistics
func (s *marketingAnalyticsService) GetRefundViolationStats(ctx context.Context, filter *requests.DashboardFilterRequest) (*responses.RefundViolationStatsResponse, error) {
	result, err := s.analyticsRepo.GetRefundViolationStats(ctx, filter)
	if err != nil {
		zap.L().Error("Failed to get refund violation stats", zap.Error(err))
		return nil, err
	}
	return result, nil
}

// GetContractRevenueBreakdown returns revenue breakdown over time for ComposedChart
// Combines: Base Cost, Affiliate Revenue (tiered), Limited Product Brand/System Shares
// Refactored to use pre-calculated values from contract_payments
func (s *marketingAnalyticsService) GetContractRevenueBreakdown(ctx context.Context, filter *requests.DashboardFilterRequest, granularity constant.TrendGranularity) (*responses.ContractRevenueBreakdownResponse, error) {
	// Fetch detailed breakdown directly from repository
	dataPoints, totalRefunds, err := s.analyticsRepo.GetDetailedContractRevenueBreakdown(ctx, filter, granularity)
	if err != nil {
		zap.L().Error("Failed to get detail contract revenue breakdown", zap.Error(err))
		return nil, err
	}

	// Calculate summary
	var summary responses.ContractRevenueBreakdownSummary
	for _, point := range dataPoints {
		summary.TotalContractBaseCost += point.ContractBaseCost
		summary.TotalAffiliateRevenue += point.AffiliateRevenue
		summary.TotalLimitedProductBrandShare += point.LimitedProductBrandShare
		summary.TotalLimitedProductSystemShare += point.LimitedProductSystemShare
		summary.GrandTotalRevenue += point.TotalContractRevenue
	}
	summary.RefundsPaid = totalRefunds

	return &responses.ContractRevenueBreakdownResponse{
		Data:        dataPoints,
		Summary:     summary,
		Granularity: string(granularity),
		Period:      filter.GetPeriodInfo(),
	}, nil
}
