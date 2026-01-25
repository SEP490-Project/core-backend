package service

import (
	"context"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/application/interfaces/iservice"
	"core-backend/internal/domain/constant"
	"core-backend/internal/domain/enum"
	"core-backend/pkg/utils"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type brandPartnerAnalyticsService struct {
	analyticsRepo irepository.BrandPartnerAnalyticsRepository
}

// NewBrandPartnerAnalyticsService creates a new brand partner analytics service
func NewBrandPartnerAnalyticsService(
	analyticsRepo irepository.BrandPartnerAnalyticsRepository,
) iservice.BrandPartnerAnalyticsService {
	return &brandPartnerAnalyticsService{
		analyticsRepo: analyticsRepo,
	}
}

// GetDashboard returns the complete Brand Partner dashboard
func (s *brandPartnerAnalyticsService) GetDashboard(ctx context.Context, brandUserID uuid.UUID, req *requests.BrandPartnerDashboardRequest) (*responses.BrandPartnerDashboardResponse, error) {
	startDate, endDate := req.GetDateRange()

	var mu sync.Mutex
	dashboard := &responses.BrandPartnerDashboardResponse{
		Period: responses.PeriodInfo{
			StartDate: startDate,
			EndDate:   endDate,
		},
	}

	// Execute queries in parallel
	err := utils.RunParallel(ctx, 9,
		// Query 1: Overview metrics
		func(ctx context.Context) error {
			overview, err := s.getOverviewMetrics(ctx, brandUserID, &startDate, &endDate)
			if err != nil {
				zap.L().Warn("Failed to get overview metrics", zap.Error(err))
				return nil
			}
			mu.Lock()
			dashboard.Overview = *overview
			mu.Unlock()
			return nil
		},

		// Query 2: Top products
		func(ctx context.Context) error {
			products, err := s.GetTopProducts(ctx, brandUserID, &requests.BrandTopProductsRequest{
				StartDate: &startDate,
				EndDate:   &endDate,
				Limit:     5,
			})
			if err != nil {
				zap.L().Warn("Failed to get top products", zap.Error(err))
				return nil
			}
			mu.Lock()
			dashboard.TopProducts = products
			mu.Unlock()
			return nil
		},

		// Query 3: Campaign metrics
		func(ctx context.Context) error {
			campaigns, err := s.GetCampaignMetrics(ctx, brandUserID, &requests.BrandCampaignsRequest{
				StartDate: &startDate,
				EndDate:   &endDate,
				Limit:     5,
			})
			if err != nil {
				zap.L().Warn("Failed to get campaign metrics", zap.Error(err))
				return nil
			}
			mu.Lock()
			dashboard.Campaigns = campaigns
			mu.Unlock()
			return nil
		},

		// Query 4: Content metrics
		func(ctx context.Context) error {
			content, err := s.GetContentMetrics(ctx, brandUserID, &requests.BrandContentMetricsRequest{
				StartDate: &startDate,
				EndDate:   &endDate,
			})
			if err != nil {
				zap.L().Warn("Failed to get content metrics", zap.Error(err))
				return nil
			}
			mu.Lock()
			dashboard.ContentMetrics = *content
			mu.Unlock()
			return nil
		},

		// Query 5: Revenue trend
		func(ctx context.Context) error {
			trend, err := s.GetRevenueTrend(ctx, brandUserID, &requests.BrandRevenueTrendRequest{
				StartDate:   &startDate,
				EndDate:     &endDate,
				Granularity: "DAY",
			})
			if err != nil {
				zap.L().Warn("Failed to get revenue trend", zap.Error(err))
				return nil
			}
			mu.Lock()
			dashboard.RevenueTrend = trend
			mu.Unlock()
			return nil
		},

		// Query 6: Affiliate metrics
		func(ctx context.Context) error {
			affiliate, err := s.GetAffiliateMetrics(ctx, brandUserID, &requests.BrandAffiliateMetricsRequest{
				StartDate: &startDate,
				EndDate:   &endDate,
			})
			if err != nil {
				zap.L().Warn("Failed to get affiliate metrics", zap.Error(err))
				return nil
			}
			mu.Lock()
			dashboard.AffiliateMetrics = *affiliate
			mu.Unlock()
			return nil
		},

		// Query 7: Contract details
		func(ctx context.Context) error {
			contracts, err := s.GetContractDetails(ctx, brandUserID, &requests.BrandContractsRequest{
				Limit: 5,
			})
			if err != nil {
				zap.L().Warn("Failed to get contract details", zap.Error(err))
				return nil
			}
			mu.Lock()
			dashboard.Contracts = contracts
			mu.Unlock()
			return nil
		},

		// Query 8: Top Rating Products
		func(ctx context.Context) error {
			topRating, err := s.GetBrandTopRatingProduct(ctx, brandUserID, &requests.BrandTopRatingProductRequest{
				StartDate: &startDate,
				EndDate:   &endDate,
				Limit:     5,
			})
			if err != nil {
				zap.L().Warn("Failed to get top rating products", zap.Error(err))
				return nil
			}
			mu.Lock()
			dashboard.TopRatingProducts = topRating
			mu.Unlock()
			return nil
		},

		// Query 9: Top Sold Products
		func(ctx context.Context) error {
			topSold, err := s.GetBrandTopSoldProduct(ctx, brandUserID, &requests.BrandTopSoldProductRequest{
				StartDate: &startDate,
				EndDate:   &endDate,
				Limit:     5,
			})
			if err != nil {
				zap.L().Warn("Failed to get top sold products", zap.Error(err))
				return nil
			}
			mu.Lock()
			dashboard.TopSoldProducts = topSold
			mu.Unlock()
			return nil
		},
	)

	if err != nil {
		zap.L().Error("Dashboard parallel query failed", zap.Error(err))
	}

	return dashboard, nil
}

// GetTopProducts returns top products by revenue for a brand
func (s *brandPartnerAnalyticsService) GetTopProducts(ctx context.Context, brandUserID uuid.UUID, req *requests.BrandTopProductsRequest) ([]responses.BrandProductMetric, error) {
	results, err := s.analyticsRepo.GetBrandTopProducts(ctx, brandUserID, req.GetLimit(), req.StartDate, req.EndDate)
	if err != nil {
		return nil, err
	}

	products := make([]responses.BrandProductMetric, len(results))
	for i, r := range results {
		products[i] = responses.BrandProductMetric{
			ProductID:   r.ProductID,
			ProductName: r.ProductName,
			ProductType: r.ProductType,
			Status:      r.Status,
			OrderCount:  r.OrderCount,
			UnitsSold:   r.UnitsSold,
			Revenue:     r.Revenue,
			Rank:        i + 1,
		}
	}
	return products, nil
}

// GetCampaignMetrics returns campaign performance metrics for a brand
func (s *brandPartnerAnalyticsService) GetCampaignMetrics(ctx context.Context, brandUserID uuid.UUID, req *requests.BrandCampaignsRequest) ([]responses.BrandCampaignMetric, error) {
	results, err := s.analyticsRepo.GetBrandCampaignMetrics(ctx, brandUserID, req.GetLimit(), req.StartDate, req.EndDate)
	if err != nil {
		return nil, err
	}

	campaigns := make([]responses.BrandCampaignMetric, len(results))
	for i, r := range results {
		var completionRate float64
		if r.TaskCount > 0 {
			completionRate = float64(r.CompletedTasks) / float64(r.TaskCount) * 100
		}
		campaigns[i] = responses.BrandCampaignMetric{
			CampaignID:       r.CampaignID,
			CampaignName:     r.CampaignName,
			Status:           r.Status,
			StartDate:        r.StartDate,
			EndDate:          r.EndDate,
			MilestoneCount:   r.MilestoneCount,
			TaskCount:        r.TaskCount,
			CompletedTasks:   r.CompletedTasks,
			CompletionRate:   completionRate,
			ContentCount:     r.ContentCount,
			TotalViews:       r.TotalViews,
			TotalEngagements: r.TotalEngagements,
		}
	}
	return campaigns, nil
}

// GetContentMetrics returns content performance metrics for a brand
func (s *brandPartnerAnalyticsService) GetContentMetrics(ctx context.Context, brandUserID uuid.UUID, req *requests.BrandContentMetricsRequest) (*responses.BrandContentMetric, error) {
	result, err := s.analyticsRepo.GetBrandContentMetrics(ctx, brandUserID, req.StartDate, req.EndDate)
	if err != nil {
		return nil, err
	}

	return &responses.BrandContentMetric{
		TotalContent:   result.TotalContent,
		PostedContent:  result.PostedContent,
		TotalViews:     result.TotalViews,
		TotalLikes:     result.TotalLikes,
		TotalComments:  result.TotalComments,
		TotalShares:    result.TotalShares,
		EngagementRate: result.EngagementRate,
	}, nil
}

// GetRevenueTrend returns revenue time-series for a brand
func (s *brandPartnerAnalyticsService) GetRevenueTrend(ctx context.Context, brandUserID uuid.UUID, req *requests.BrandRevenueTrendRequest) ([]responses.BrandRevenueTrendPoint, error) {
	results, err := s.analyticsRepo.GetBrandRevenueTrend(ctx, brandUserID, req.GetGranularity(), req.StartDate, req.EndDate)
	if err != nil {
		return nil, err
	}

	trend := make([]responses.BrandRevenueTrendPoint, len(results))
	for i, r := range results {
		trend[i] = responses.BrandRevenueTrendPoint{
			Date:       r.Date,
			OrderCount: r.OrderCount,
			UnitsSold:  r.UnitsSold,
			Revenue:    r.Revenue,
		}
	}
	return trend, nil
}

// GetAffiliateMetrics returns affiliate link performance for a brand
func (s *brandPartnerAnalyticsService) GetAffiliateMetrics(ctx context.Context, brandUserID uuid.UUID, req *requests.BrandAffiliateMetricsRequest) (*responses.BrandAffiliateMetric, error) {
	result, err := s.analyticsRepo.GetBrandAffiliateMetrics(ctx, brandUserID, req.StartDate, req.EndDate)
	if err != nil {
		return nil, err
	}

	return &responses.BrandAffiliateMetric{
		TotalLinks:  result.TotalLinks,
		ActiveLinks: result.ActiveLinks,
		TotalClicks: result.TotalClicks,
	}, nil
}

// GetContractDetails returns contract details for a brand
func (s *brandPartnerAnalyticsService) GetContractDetails(ctx context.Context, brandUserID uuid.UUID, req *requests.BrandContractsRequest) ([]responses.BrandContractDetail, error) {
	results, err := s.analyticsRepo.GetBrandContractDetails(ctx, brandUserID, req.GetLimit())
	if err != nil {
		return nil, err
	}

	contracts := make([]responses.BrandContractDetail, len(results))
	for i, r := range results {
		contracts[i] = responses.BrandContractDetail{
			ContractID:     r.ContractID,
			ContractNumber: r.ContractNumber,
			Type:           r.Type,
			Status:         r.Status,
			TotalValue:     r.TotalValue,
			StartDate:      r.StartDate,
			EndDate:        r.EndDate,
			PaidAmount:     r.PaidAmount,
			PendingAmount:  r.PendingAmount,
			CampaignCount:  r.CampaignCount,
		}
	}
	return contracts, nil
}

// region: ============== Private Helper Methods ==============

// getOverviewMetrics returns high-level overview metrics for a brand
func (s *brandPartnerAnalyticsService) getOverviewMetrics(ctx context.Context, brandUserID uuid.UUID, startDate, endDate *time.Time) (*responses.BrandOverviewMetrics, error) {
	overview := &responses.BrandOverviewMetrics{}

	var mu sync.Mutex
	_ = utils.RunParallel(ctx, 8,
		func(ctx context.Context) error {
			count, _ := s.analyticsRepo.GetBrandContractCount(ctx, brandUserID, nil)
			mu.Lock()
			overview.TotalContracts = count
			mu.Unlock()
			return nil
		},
		func(ctx context.Context) error {
			active := enum.ContractStatusActive.String()
			count, _ := s.analyticsRepo.GetBrandContractCount(ctx, brandUserID, &active)
			mu.Lock()
			overview.ActiveContracts = count
			mu.Unlock()
			return nil
		},
		func(ctx context.Context) error {
			count, _ := s.analyticsRepo.GetBrandCampaignCount(ctx, brandUserID, nil)
			mu.Lock()
			overview.TotalCampaigns = count
			mu.Unlock()
			return nil
		},
		func(ctx context.Context) error {
			active := enum.CampaignRunning.String()
			count, _ := s.analyticsRepo.GetBrandCampaignCount(ctx, brandUserID, &active)
			mu.Lock()
			overview.ActiveCampaigns = count
			mu.Unlock()
			return nil
		},
		func(ctx context.Context) error {
			count, _ := s.analyticsRepo.GetBrandProductCount(ctx, brandUserID, nil)
			mu.Lock()
			overview.TotalProducts = count
			mu.Unlock()
			return nil
		},
		func(ctx context.Context) error {
			count, _ := s.analyticsRepo.GetBrandOrderCount(ctx, brandUserID, nil, startDate, endDate)
			mu.Lock()
			overview.TotalOrders = count
			mu.Unlock()
			return nil
		},
		func(ctx context.Context) error {
			revenue, _ := s.analyticsRepo.GetBrandTotalRevenue(ctx, brandUserID, startDate, endDate)
			mu.Lock()
			overview.TotalRevenue = revenue
			mu.Unlock()
			return nil
		},
		func(ctx context.Context) error {
			pending, _ := s.analyticsRepo.GetBrandPendingPayments(ctx, brandUserID)
			mu.Lock()
			overview.PendingPayments = pending
			mu.Unlock()
			return nil
		},
	)

	return overview, nil
}

// endregion

func (s *brandPartnerAnalyticsService) GetBrandTopRatingProduct(ctx context.Context, brandUserID uuid.UUID, req *requests.BrandTopRatingProductRequest) ([]responses.BrandProductRating, error) {
	results, err := s.analyticsRepo.GetBrandTopRatingProduct(ctx, brandUserID, req.Limit, req.StartDate, req.EndDate)
	if err != nil {
		return nil, err
	}
	topRating := make([]responses.BrandProductRating, len(results))
	for i, r := range results {
		topRating[i] = responses.BrandProductRating{
			ProductID:     r.ProductID,
			ProductName:   r.ProductName,
			AverageRating: r.AverageRating,
			Type:          r.Type,
			Rank:          i + 1,
		}
	}
	return topRating, nil
}

func (s *brandPartnerAnalyticsService) GetBrandTopSoldProduct(ctx context.Context, brandUserID uuid.UUID, req *requests.BrandTopSoldProductRequest) ([]responses.BrandTopSoldProducts, error) {
	results, err := s.analyticsRepo.GetBrandTopSoldProduct(ctx, brandUserID, req.Limit, req.StartDate, req.EndDate)
	if err != nil {

		zap.L().Error("failed to get top sold products from repository: ", zap.Error(err))
		return nil, err
	}
	topSold := make([]responses.BrandTopSoldProducts, len(results))
	for i, r := range results {
		topSold[i] = responses.BrandTopSoldProducts{
			ProductID:    r.ProductID,
			ProductName:  r.ProductName,
			UnitsSold:    r.UnitsSold,
			TotalRevenue: r.TotalRevenue,
			Rank:         i + 1,
		}
	}
	return topSold, nil
}

// GetBrandContractStatusDistribution returns contract status distribution for a brand
func (s *brandPartnerAnalyticsService) GetBrandContractStatusDistribution(ctx context.Context, brandUserID uuid.UUID, filter *requests.DashboardFilterRequest) (*responses.ContractStatusDistributionResponse, error) {
	return s.analyticsRepo.GetBrandContractStatusDistribution(ctx, brandUserID, filter)
}

// GetBrandTaskStatusDistribution returns task status distribution for a brand
func (s *brandPartnerAnalyticsService) GetBrandTaskStatusDistribution(ctx context.Context, brandUserID uuid.UUID, filter *requests.DashboardFilterRequest) (*responses.TaskStatusDistributionResponse, error) {
	return s.analyticsRepo.GetBrandTaskStatusDistribution(ctx, brandUserID, filter)
}

// GetBrandRevenueOverTime returns revenue trend over time for a brand (Limited Products only)
func (s *brandPartnerAnalyticsService) GetBrandRevenueOverTime(ctx context.Context, brandUserID uuid.UUID, filter *requests.DashboardFilterRequest, granularity constant.TrendGranularity) (*responses.BrandRevenueOverTimeResponse, error) {
	data, err := s.analyticsRepo.GetBrandLimitedProductRevenueOverTime(ctx, brandUserID, filter, granularity)
	if err != nil {
		return nil, err
	}

	// Calculate summary totals
	var summary responses.BrandRevenueOverTimeSummary
	for _, d := range data {
		d.TotalRevenue = d.BrandLimitedRevenue + d.BrandAffiliateEarnings
		summary.TotalBrandLimitedRevenue += d.BrandLimitedRevenue
		summary.TotalBrandAffiliateEarnings += d.BrandAffiliateEarnings
	}
	summary.GrandTotalRevenue = summary.TotalBrandLimitedRevenue + summary.TotalBrandAffiliateEarnings

	return &responses.BrandRevenueOverTimeResponse{
		Data:        data,
		Granularity: string(granularity),
		Period:      filter.GetPeriodInfo(),
		Summary:     summary,
	}, nil
}

// GetBrandRefundViolationStats returns refund and contract violation stats for a brand
func (s *brandPartnerAnalyticsService) GetBrandRefundViolationStats(ctx context.Context, brandUserID uuid.UUID, filter *requests.DashboardFilterRequest) (*responses.RefundViolationStatsResponse, error) {
	return s.analyticsRepo.GetBrandRefundViolationStats(ctx, brandUserID, filter)
}

// GetBrandGrossIncome returns brand's gross income (limited product revenue × brand share percentage)
func (s *brandPartnerAnalyticsService) GetBrandGrossIncome(ctx context.Context, brandUserID uuid.UUID, filter *requests.DashboardFilterRequest) (*responses.BrandIncomeResponse, error) {
	return s.analyticsRepo.GetBrandGrossIncome(ctx, brandUserID, filter)
}

// GetBrandNetIncome returns brand's net income (gross income - paid contract payments)
func (s *brandPartnerAnalyticsService) GetBrandNetIncome(ctx context.Context, brandUserID uuid.UUID, filter *requests.DashboardFilterRequest) (*responses.BrandNetIncomeResponse, error) {
	return s.analyticsRepo.GetBrandNetIncome(ctx, brandUserID, filter)
}
