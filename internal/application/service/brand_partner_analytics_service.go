package service

import (
	"context"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/application/interfaces/iservice"
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
func (s *brandPartnerAnalyticsService) GetDashboard(ctx context.Context, brandID uuid.UUID, req *requests.BrandPartnerDashboardRequest) (*responses.BrandPartnerDashboardResponse, error) {
	startDate, endDate := req.GetDateRange()

	var mu sync.Mutex
	dashboard := &responses.BrandPartnerDashboardResponse{
		Period: responses.PeriodInfo{
			StartDate: startDate,
			EndDate:   endDate,
		},
	}

	// Execute queries in parallel
	err := utils.RunParallel(ctx, 7,
		// Query 1: Overview metrics
		func(ctx context.Context) error {
			overview, err := s.getOverviewMetrics(ctx, brandID, &startDate, &endDate)
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
			products, err := s.GetTopProducts(ctx, brandID, &requests.BrandTopProductsRequest{
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
			campaigns, err := s.GetCampaignMetrics(ctx, brandID, &requests.BrandCampaignsRequest{
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
			content, err := s.GetContentMetrics(ctx, brandID, &requests.BrandContentMetricsRequest{
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
			trend, err := s.GetRevenueTrend(ctx, brandID, &requests.BrandRevenueTrendRequest{
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
			affiliate, err := s.GetAffiliateMetrics(ctx, brandID, &requests.BrandAffiliateMetricsRequest{
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
			contracts, err := s.GetContractDetails(ctx, brandID, &requests.BrandContractsRequest{
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
	)

	if err != nil {
		zap.L().Error("Dashboard parallel query failed", zap.Error(err))
	}

	return dashboard, nil
}

// getOverviewMetrics returns high-level overview metrics for a brand
func (s *brandPartnerAnalyticsService) getOverviewMetrics(ctx context.Context, brandID uuid.UUID, startDate, endDate *time.Time) (*responses.BrandOverviewMetrics, error) {
	overview := &responses.BrandOverviewMetrics{}

	var mu sync.Mutex
	_ = utils.RunParallel(ctx, 8,
		func(ctx context.Context) error {
			count, _ := s.analyticsRepo.GetBrandContractCount(ctx, brandID, nil)
			mu.Lock()
			overview.TotalContracts = count
			mu.Unlock()
			return nil
		},
		func(ctx context.Context) error {
			active := "ACTIVE"
			count, _ := s.analyticsRepo.GetBrandContractCount(ctx, brandID, &active)
			mu.Lock()
			overview.ActiveContracts = count
			mu.Unlock()
			return nil
		},
		func(ctx context.Context) error {
			count, _ := s.analyticsRepo.GetBrandCampaignCount(ctx, brandID, nil)
			mu.Lock()
			overview.TotalCampaigns = count
			mu.Unlock()
			return nil
		},
		func(ctx context.Context) error {
			active := "ACTIVE"
			count, _ := s.analyticsRepo.GetBrandCampaignCount(ctx, brandID, &active)
			mu.Lock()
			overview.ActiveCampaigns = count
			mu.Unlock()
			return nil
		},
		func(ctx context.Context) error {
			count, _ := s.analyticsRepo.GetBrandProductCount(ctx, brandID, nil)
			mu.Lock()
			overview.TotalProducts = count
			mu.Unlock()
			return nil
		},
		func(ctx context.Context) error {
			count, _ := s.analyticsRepo.GetBrandOrderCount(ctx, brandID, nil, startDate, endDate)
			mu.Lock()
			overview.TotalOrders = count
			mu.Unlock()
			return nil
		},
		func(ctx context.Context) error {
			revenue, _ := s.analyticsRepo.GetBrandTotalRevenue(ctx, brandID, startDate, endDate)
			mu.Lock()
			overview.TotalRevenue = revenue
			mu.Unlock()
			return nil
		},
		func(ctx context.Context) error {
			pending, _ := s.analyticsRepo.GetBrandPendingPayments(ctx, brandID)
			mu.Lock()
			overview.PendingPayments = pending
			mu.Unlock()
			return nil
		},
	)

	return overview, nil
}

// GetTopProducts returns top products by revenue for a brand
func (s *brandPartnerAnalyticsService) GetTopProducts(ctx context.Context, brandID uuid.UUID, req *requests.BrandTopProductsRequest) ([]responses.BrandProductMetric, error) {
	results, err := s.analyticsRepo.GetBrandTopProducts(ctx, brandID, req.GetLimit(), req.StartDate, req.EndDate)
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
func (s *brandPartnerAnalyticsService) GetCampaignMetrics(ctx context.Context, brandID uuid.UUID, req *requests.BrandCampaignsRequest) ([]responses.BrandCampaignMetric, error) {
	results, err := s.analyticsRepo.GetBrandCampaignMetrics(ctx, brandID, req.GetLimit(), req.StartDate, req.EndDate)
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
func (s *brandPartnerAnalyticsService) GetContentMetrics(ctx context.Context, brandID uuid.UUID, req *requests.BrandContentMetricsRequest) (*responses.BrandContentMetric, error) {
	result, err := s.analyticsRepo.GetBrandContentMetrics(ctx, brandID, req.StartDate, req.EndDate)
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
func (s *brandPartnerAnalyticsService) GetRevenueTrend(ctx context.Context, brandID uuid.UUID, req *requests.BrandRevenueTrendRequest) ([]responses.BrandRevenueTrendPoint, error) {
	results, err := s.analyticsRepo.GetBrandRevenueTrend(ctx, brandID, req.GetGranularity(), req.StartDate, req.EndDate)
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
func (s *brandPartnerAnalyticsService) GetAffiliateMetrics(ctx context.Context, brandID uuid.UUID, req *requests.BrandAffiliateMetricsRequest) (*responses.BrandAffiliateMetric, error) {
	result, err := s.analyticsRepo.GetBrandAffiliateMetrics(ctx, brandID, req.StartDate, req.EndDate)
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
func (s *brandPartnerAnalyticsService) GetContractDetails(ctx context.Context, brandID uuid.UUID, req *requests.BrandContractsRequest) ([]responses.BrandContractDetail, error) {
	results, err := s.analyticsRepo.GetBrandContractDetails(ctx, brandID, req.GetLimit())
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
