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

	"go.uber.org/zap"
)

type salesStaffAnalyticsService struct {
	analyticsRepo irepository.SalesStaffAnalyticsRepository
}

// NewSalesStaffAnalyticsService creates a new sales staff analytics service
func NewSalesStaffAnalyticsService(
	analyticsRepo irepository.SalesStaffAnalyticsRepository,
) iservice.SalesStaffAnalyticsService {
	return &salesStaffAnalyticsService{
		analyticsRepo: analyticsRepo,
	}
}

// GetDashboard returns the complete Sales Staff dashboard
func (s *salesStaffAnalyticsService) GetDashboard(ctx context.Context, req *requests.SalesStaffDashboardRequest) (*responses.SalesStaffDashboardResponse, error) {
	startDate, endDate := req.GetDateRange()

	var mu sync.Mutex
	dashboard := &responses.SalesStaffDashboardResponse{
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
				return nil // Don't fail entire dashboard
			}
			mu.Lock()
			dashboard.Overview = *overview
			mu.Unlock()
			return nil
		},

		// Query 2: Orders breakdown
		func(ctx context.Context) error {
			breakdown, err := s.GetOrdersOverview(ctx, &requests.OrdersOverviewRequest{
				StartDate: &startDate,
				EndDate:   &endDate,
			})
			if err != nil {
				zap.L().Warn("Failed to get orders breakdown", zap.Error(err))
				return nil
			}
			mu.Lock()
			dashboard.OrdersBreakdown = *breakdown
			mu.Unlock()
			return nil
		},

		// Query 3: Revenue by source
		func(ctx context.Context) error {
			revenue, err := s.GetRevenueBySource(ctx, &requests.RevenueBySourceRequest{
				StartDate: &startDate,
				EndDate:   &endDate,
			})
			if err != nil {
				zap.L().Warn("Failed to get revenue by source", zap.Error(err))
				return nil
			}
			mu.Lock()
			dashboard.RevenueBySource = *revenue
			mu.Unlock()
			return nil
		},

		// Query 4: Top brands
		func(ctx context.Context) error {
			brands, err := s.GetTopBrands(ctx, &requests.TopBrandsRequest{
				StartDate: &startDate,
				EndDate:   &endDate,
				Limit:     5,
			})
			if err != nil {
				zap.L().Warn("Failed to get top brands", zap.Error(err))
				return nil
			}
			mu.Lock()
			dashboard.TopBrands = brands
			mu.Unlock()
			return nil
		},

		// Query 5: Top products
		func(ctx context.Context) error {
			products, err := s.GetTopProducts(ctx, &requests.TopProductsRequest{
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

		// Query 6: Recent orders
		func(ctx context.Context) error {
			recent, err := s.analyticsRepo.GetRecentOrders(ctx, 10)
			if err != nil {
				zap.L().Warn("Failed to get recent orders", zap.Error(err))
				return nil
			}
			recentItems := make([]responses.RecentOrderItem, len(recent))
			for i, r := range recent {
				recentItems[i] = responses.RecentOrderItem{
					OrderID:      r.OrderID,
					OrderNumber:  r.OrderNumber,
					CustomerName: r.CustomerName,
					TotalAmount:  r.TotalAmount,
					Status:       r.Status,
					OrderType:    r.OrderType,
					ItemCount:    r.ItemCount,
					CreatedAt:    r.CreatedAt,
				}
			}
			mu.Lock()
			dashboard.RecentOrders = recentItems
			mu.Unlock()
			return nil
		},

		// Query 7: Revenue trend
		func(ctx context.Context) error {
			trend, err := s.GetRevenueTrend(ctx, &requests.RevenueGrowthRequest{
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
	)

	if err != nil {
		zap.L().Error("Dashboard parallel query failed", zap.Error(err))
	}

	return dashboard, nil
}

// getOverviewMetrics returns high-level overview metrics
func (s *salesStaffAnalyticsService) getOverviewMetrics(ctx context.Context, startDate, endDate *time.Time) (*responses.SalesOverviewMetrics, error) {
	overview := &responses.SalesOverviewMetrics{}

	var mu sync.Mutex
	_ = utils.RunParallel(ctx, 10,
		func(ctx context.Context) error {
			revenue, _ := s.analyticsRepo.GetOrdersRevenueByType(ctx, "", startDate, endDate)
			mu.Lock()
			overview.OrderRevenue = revenue
			mu.Unlock()
			return nil
		},
		func(ctx context.Context) error {
			var contractRevenue float64
			for _, ct := range []string{"ADVERTISING", "AFFILIATE", "AMBASSADOR", "CO_PRODUCING"} {
				r, _ := s.analyticsRepo.GetContractRevenueByType(ctx, ct, startDate, endDate)
				contractRevenue += r
			}
			mu.Lock()
			overview.ContractRevenue = contractRevenue
			mu.Unlock()
			return nil
		},
		func(ctx context.Context) error {
			count, _ := s.analyticsRepo.GetOrdersCountByType(ctx, "", startDate, endDate)
			mu.Lock()
			overview.TotalOrders = count
			mu.Unlock()
			return nil
		},
		func(ctx context.Context) error {
			count, _ := s.analyticsRepo.GetPreOrdersCount(ctx, startDate, endDate)
			mu.Lock()
			overview.TotalPreOrders = count
			mu.Unlock()
			return nil
		},
		func(ctx context.Context) error {
			count, _ := s.analyticsRepo.GetOrdersCountByStatus(ctx, "", "RECEIVED", startDate, endDate)
			mu.Lock()
			overview.CompletedOrders = count
			mu.Unlock()
			return nil
		},
		func(ctx context.Context) error {
			count, _ := s.analyticsRepo.GetOrdersCountByStatus(ctx, "", "PENDING", startDate, endDate)
			mu.Lock()
			overview.PendingOrders = count
			mu.Unlock()
			return nil
		},
	)

	overview.TotalRevenue = overview.OrderRevenue + overview.ContractRevenue
	if overview.TotalOrders > 0 {
		overview.AverageOrderValue = overview.OrderRevenue / float64(overview.TotalOrders)
	}

	return overview, nil
}

// GetOrdersOverview returns orders statistics by type and status
func (s *salesStaffAnalyticsService) GetOrdersOverview(ctx context.Context, req *requests.OrdersOverviewRequest) (*responses.OrdersBreakdown, error) {
	breakdown := &responses.OrdersBreakdown{}

	var mu sync.Mutex
	_ = utils.RunParallel(ctx, 6,
		// Standard orders
		func(ctx context.Context) error {
			count, _ := s.analyticsRepo.GetOrdersCountByType(ctx, "STANDARD", req.StartDate, req.EndDate)
			revenue, _ := s.analyticsRepo.GetOrdersRevenueByType(ctx, "STANDARD", req.StartDate, req.EndDate)
			completed, _ := s.analyticsRepo.GetOrdersCountByStatus(ctx, "STANDARD", "RECEIVED", req.StartDate, req.EndDate)
			pending, _ := s.analyticsRepo.GetOrdersCountByStatus(ctx, "STANDARD", "PENDING", req.StartDate, req.EndDate)
			cancelled, _ := s.analyticsRepo.GetOrdersCountByStatus(ctx, "STANDARD", "CANCELLED", req.StartDate, req.EndDate)

			mu.Lock()
			breakdown.StandardOrders = responses.OrderTypeStats{
				TotalCount:     count,
				TotalRevenue:   revenue,
				CompletedCount: completed,
				PendingCount:   pending,
				CancelledCount: cancelled,
			}
			mu.Unlock()
			return nil
		},

		// Limited orders
		func(ctx context.Context) error {
			count, _ := s.analyticsRepo.GetOrdersCountByType(ctx, "LIMITED", req.StartDate, req.EndDate)
			revenue, _ := s.analyticsRepo.GetOrdersRevenueByType(ctx, "LIMITED", req.StartDate, req.EndDate)
			completed, _ := s.analyticsRepo.GetOrdersCountByStatus(ctx, "LIMITED", "RECEIVED", req.StartDate, req.EndDate)
			pending, _ := s.analyticsRepo.GetOrdersCountByStatus(ctx, "LIMITED", "PENDING", req.StartDate, req.EndDate)
			cancelled, _ := s.analyticsRepo.GetOrdersCountByStatus(ctx, "LIMITED", "CANCELLED", req.StartDate, req.EndDate)

			mu.Lock()
			breakdown.LimitedOrders = responses.OrderTypeStats{
				TotalCount:     count,
				TotalRevenue:   revenue,
				CompletedCount: completed,
				PendingCount:   pending,
				CancelledCount: cancelled,
			}
			mu.Unlock()
			return nil
		},

		// Pre-orders
		func(ctx context.Context) error {
			count, _ := s.analyticsRepo.GetPreOrdersCount(ctx, req.StartDate, req.EndDate)
			revenue, _ := s.analyticsRepo.GetPreOrdersRevenue(ctx, req.StartDate, req.EndDate)
			received, _ := s.analyticsRepo.GetPreOrdersCountByStatus(ctx, "RECEIVED", req.StartDate, req.EndDate)
			pending, _ := s.analyticsRepo.GetPreOrdersCountByStatus(ctx, "PENDING", req.StartDate, req.EndDate)
			confirmed, _ := s.analyticsRepo.GetPreOrdersCountByStatus(ctx, "CONFIRMED", req.StartDate, req.EndDate)
			cancelled, _ := s.analyticsRepo.GetPreOrdersCountByStatus(ctx, "CANCELLED", req.StartDate, req.EndDate)

			mu.Lock()
			breakdown.PreOrders = responses.PreOrderStats{
				TotalCount:     count,
				TotalRevenue:   revenue,
				ReceivedCount:  received,
				PendingCount:   pending,
				ConfirmedCount: confirmed,
				CancelledCount: cancelled,
			}
			mu.Unlock()
			return nil
		},
	)

	return breakdown, nil
}

// GetPreOrdersOverview returns pre-orders statistics
func (s *salesStaffAnalyticsService) GetPreOrdersOverview(ctx context.Context, req *requests.PreOrdersOverviewRequest) (*responses.PreOrderStats, error) {
	count, _ := s.analyticsRepo.GetPreOrdersCount(ctx, req.StartDate, req.EndDate)
	revenue, _ := s.analyticsRepo.GetPreOrdersRevenue(ctx, req.StartDate, req.EndDate)
	received, _ := s.analyticsRepo.GetPreOrdersCountByStatus(ctx, "RECEIVED", req.StartDate, req.EndDate)
	pending, _ := s.analyticsRepo.GetPreOrdersCountByStatus(ctx, "PENDING", req.StartDate, req.EndDate)
	confirmed, _ := s.analyticsRepo.GetPreOrdersCountByStatus(ctx, "CONFIRMED", req.StartDate, req.EndDate)
	cancelled, _ := s.analyticsRepo.GetPreOrdersCountByStatus(ctx, "CANCELLED", req.StartDate, req.EndDate)

	return &responses.PreOrderStats{
		TotalCount:     count,
		TotalRevenue:   revenue,
		ReceivedCount:  received,
		PendingCount:   pending,
		ConfirmedCount: confirmed,
		CancelledCount: cancelled,
	}, nil
}

// GetRevenueBySource returns revenue breakdown by source
func (s *salesStaffAnalyticsService) GetRevenueBySource(ctx context.Context, req *requests.RevenueBySourceRequest) (*responses.RevenueBySource, error) {
	revenue := &responses.RevenueBySource{}

	var mu sync.Mutex
	_ = utils.RunParallel(ctx, 6,
		func(ctx context.Context) error {
			r, _ := s.analyticsRepo.GetOrdersRevenueByType(ctx, "STANDARD", req.StartDate, req.EndDate)
			mu.Lock()
			revenue.StandardProductRevenue = r
			mu.Unlock()
			return nil
		},
		func(ctx context.Context) error {
			limitedOrders, _ := s.analyticsRepo.GetOrdersRevenueByType(ctx, "LIMITED", req.StartDate, req.EndDate)
			preOrders, _ := s.analyticsRepo.GetPreOrdersRevenue(ctx, req.StartDate, req.EndDate)
			mu.Lock()
			revenue.LimitedProductRevenue = limitedOrders + preOrders
			mu.Unlock()
			return nil
		},
		func(ctx context.Context) error {
			r, _ := s.analyticsRepo.GetContractRevenueByType(ctx, "ADVERTISING", req.StartDate, req.EndDate)
			mu.Lock()
			revenue.AdvertisingRevenue = r
			mu.Unlock()
			return nil
		},
		func(ctx context.Context) error {
			r, _ := s.analyticsRepo.GetContractRevenueByType(ctx, "AFFILIATE", req.StartDate, req.EndDate)
			mu.Lock()
			revenue.AffiliateRevenue = r
			mu.Unlock()
			return nil
		},
		func(ctx context.Context) error {
			r, _ := s.analyticsRepo.GetContractRevenueByType(ctx, "AMBASSADOR", req.StartDate, req.EndDate)
			mu.Lock()
			revenue.AmbassadorRevenue = r
			mu.Unlock()
			return nil
		},
		func(ctx context.Context) error {
			r, _ := s.analyticsRepo.GetContractRevenueByType(ctx, "CO_PRODUCING", req.StartDate, req.EndDate)
			mu.Lock()
			revenue.CoProducingRevenue = r
			mu.Unlock()
			return nil
		},
	)

	revenue.TotalRevenue = revenue.StandardProductRevenue + revenue.LimitedProductRevenue +
		revenue.AdvertisingRevenue + revenue.AffiliateRevenue +
		revenue.AmbassadorRevenue + revenue.CoProducingRevenue

	return revenue, nil
}

// GetTopBrands returns top brands by revenue
func (s *salesStaffAnalyticsService) GetTopBrands(ctx context.Context, req *requests.TopBrandsRequest) ([]responses.BrandSalesMetric, error) {
	results, err := s.analyticsRepo.GetTopBrandsByRevenue(ctx, req.GetLimit(), req.StartDate, req.EndDate)
	if err != nil {
		return nil, err
	}

	brands := make([]responses.BrandSalesMetric, len(results))
	for i, r := range results {
		brands[i] = responses.BrandSalesMetric{
			BrandID:      r.BrandID,
			BrandName:    r.BrandName,
			TotalRevenue: r.TotalRevenue,
			OrderCount:   r.OrderCount,
			ProductCount: r.ProductCount,
			Rank:         i + 1,
		}
	}
	return brands, nil
}

// GetTopProducts returns top products by revenue
func (s *salesStaffAnalyticsService) GetTopProducts(ctx context.Context, req *requests.TopProductsRequest) ([]responses.ProductSalesMetric, error) {
	productType := ""
	if req.ProductType != nil {
		productType = *req.ProductType
	}

	results, err := s.analyticsRepo.GetTopProductsByRevenue(ctx, productType, req.GetLimit(), req.StartDate, req.EndDate)
	if err != nil {
		return nil, err
	}

	products := make([]responses.ProductSalesMetric, len(results))
	for i, r := range results {
		products[i] = responses.ProductSalesMetric{
			ProductID:    r.ProductID,
			ProductName:  r.ProductName,
			BrandName:    r.BrandName,
			ProductType:  r.ProductType,
			TotalRevenue: r.TotalRevenue,
			UnitsSold:    r.UnitsSold,
			Rank:         i + 1,
		}
	}
	return products, nil
}

// GetRevenueTrend returns revenue time-series data
func (s *salesStaffAnalyticsService) GetRevenueTrend(ctx context.Context, req *requests.RevenueGrowthRequest) ([]responses.RevenueTrendPoint, error) {
	results, err := s.analyticsRepo.GetRevenueTrend(ctx, req.GetGranularity(), req.StartDate, req.EndDate)
	if err != nil {
		return nil, err
	}

	trend := make([]responses.RevenueTrendPoint, len(results))
	for i, r := range results {
		trend[i] = responses.RevenueTrendPoint{
			Date:              r.Date,
			Revenue:           r.Revenue,
			OrderCount:        r.OrderCount,
			AverageOrderValue: r.AverageOrderValue,
		}
	}
	return trend, nil
}

// GetPaymentStatus returns contract payment status overview
func (s *salesStaffAnalyticsService) GetPaymentStatus(ctx context.Context, req *requests.PaymentStatusRequest) (*responses.PaymentStatusOverview, error) {
	result, err := s.analyticsRepo.GetPaymentStatusCounts(ctx, req.ContractID, req.StartDate, req.EndDate)
	if err != nil {
		return nil, err
	}

	return &responses.PaymentStatusOverview{
		TotalPayments:   result.TotalPayments,
		PaidPayments:    result.PaidPayments,
		PendingPayments: result.PendingPayments,
		OverduePayments: result.OverduePayments,
		TotalAmount:     result.TotalAmount,
		PaidAmount:      result.PaidAmount,
		PendingAmount:   result.PendingAmount,
		OverdueAmount:   result.OverdueAmount,
	}, nil
}
