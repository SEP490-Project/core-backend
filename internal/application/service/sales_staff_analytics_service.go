package service

import (
	"context"
	"sort"
	"strings"
	"time"

	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/application/interfaces/iservice"
	"core-backend/internal/domain/constant"
	"core-backend/pkg/utils"

	"go.uber.org/zap"
)

type SalesStaffAnalyticsService struct {
	repo irepository.SalesStaffAnalyticsRepository
}

func NewSalesStaffAnalyticsService(
	repo irepository.SalesStaffAnalyticsRepository,
) iservice.SalesStaffAnalyticsService {
	return &SalesStaffAnalyticsService{
		repo: repo,
	}
}

func (s *SalesStaffAnalyticsService) GetFinancialsDashboard(ctx context.Context, req *requests.SalesDashboardFilter) (*responses.FinancialsDashboardResponse, error) {
	var response responses.FinancialsDashboardResponse

	// Default dates
	from, to, err := s.getDateRange(ctx, req)
	if err != nil {
		return nil, err
	}
	limit := req.Limit
	if limit <= 0 {
		limit = 5
	}

	// Statuses
	completedOrders := constant.ValidCompletedOrderStatus
	completedPreOrders := constant.ValidCompletedPreOrderStatus

	// Parallel execution
	err = utils.RunParallel(ctx, 5,
		func(ctx context.Context) error {
			var summary *responses.FinancialsSummary
			summary, err = s.repo.GetFinancialsSummary(ctx, from, to, completedOrders, completedPreOrders)
			if err != nil {
				return err
			}
			// Calculate Growth Rate (requires previous period)
			prevFrom, prevTo := s.getPreviousPeriod(from, to)
			var previousSoldRevenue float64
			previousSoldRevenue, err = s.repo.GetTotalSoldRevenue(ctx, prevFrom, prevTo, completedOrders, completedPreOrders)
			if err == nil && previousSoldRevenue > 0 {
				summary.RevenueGrowth = ((summary.TotalSoldRevenue - previousSoldRevenue) / previousSoldRevenue) * 100
			}

			response.Summary = *summary
			return nil
		},
		func(ctx context.Context) error {
			var byProd []responses.RevenueByProductType
			var byCat []responses.RevenueByCategory
			byProd, byCat, err = s.repo.GetRevenueBreakdown(ctx, from, to, completedOrders, completedPreOrders)
			if err != nil {
				return err
			}
			response.RevenueByProduct = byProd
			response.RevenueByCategory = byCat
			return nil
		},
		func(ctx context.Context) error {
			var revenueTrend map[string][]responses.SalesTimeSeriesPoint
			revenueTrend, err = s.repo.GetRevenueTrend(ctx, from, to, req.PeriodGap, completedOrders, completedPreOrders)
			if err != nil {
				return err
			}
			response.RevenueTrend = revenueTrend
			return nil
		},
		func(ctx context.Context) error {
			var prods, cats, brands []responses.TopEntity
			prods, cats, brands, err = s.repo.GetTopSellingByRevenue(ctx, from, to, completedOrders, completedPreOrders, limit, req.SortBy, req.SortOrder)
			if err != nil {
				return err
			}
			response.TopLists = responses.FinancialsTopLists{
				TopProducts:   prods,
				TopCategories: cats,
				TopBrands:     brands,
			}
			return nil
		},
	)

	if err != nil {
		return nil, err
	}

	return &response, nil
}

func (s *SalesStaffAnalyticsService) GetOrdersDashboard(ctx context.Context, req *requests.SalesDashboardFilter) (*responses.OrdersDashboardResponse, error) {
	var response responses.OrdersDashboardResponse

	from, to, err := s.getDateRange(ctx, req)
	if err != nil {
		return nil, err
	}
	limit := req.Limit
	if limit <= 0 {
		limit = 5
	}

	// Statuses for Top Selling by Volume (usually completed orders)
	completedOrders := constant.ValidCompletedOrderStatus
	completedPreOrders := constant.ValidCompletedPreOrderStatus

	err = utils.RunParallel(ctx, 5,
		func(ctx context.Context) error {
			var summary *responses.OrdersSummary
			summary, err = s.repo.GetOrdersSummary(ctx, from, to)
			if err != nil {
				return err
			}
			response.Summary = *summary
			return nil
		},
		func(ctx context.Context) error {
			var ordersDist, preOrdersDist responses.OrderStatusDistribution
			ordersDist, preOrdersDist, err = s.repo.GetOrderStatusDistribution(ctx, from, to)
			if err != nil {
				return err
			}
			response.OrdersPieChart = ordersDist
			response.PreOrdersPieChart = preOrdersDist
			return nil
		},
		func(ctx context.Context) error {
			var orders, preOrders, standard, limited []responses.SalesTimeSeriesPoint
			orders, preOrders, standard, limited, err = s.repo.GetOrdersTrend(ctx, from, to, req.PeriodGap)
			if err != nil {
				return err
			}
			// Sort limited trend
			sort.Slice(limited, func(i, j int) bool {
				return limited[i].Time.Before(limited[j].Time)
			})

			response.OrdersTrend = responses.OrdersTrendCharts{
				OrdersVsPreOrders: map[string][]responses.SalesTimeSeriesPoint{
					"ORDER":     orders,
					"PRE_ORDER": preOrders,
				},
				StandardVsLimited: map[string][]responses.SalesTimeSeriesPoint{
					"STANDARD": standard,
					"LIMITED":  limited,
				},
			}
			return nil
		},
		func(ctx context.Context) error {
			var prods, cats, brands []responses.TopEntity
			prods, cats, brands, err = s.repo.GetTopSellingByVolume(ctx, from, to, completedOrders, completedPreOrders, limit, req.SortBy, req.SortOrder)
			if err != nil {
				return err
			}
			response.TopLists = responses.OrdersTopLists{
				TopProducts:   prods,
				TopCategories: cats,
				TopBrands:     brands,
			}
			return nil
		},
		func(ctx context.Context) error {
			var latest []responses.LatestOrder
			latest, err = s.repo.GetLatestOrders(ctx, from, to, limit)
			if err != nil {
				return err
			}
			response.LatestOrders = latest
			return nil
		},
	)

	if err != nil {
		return nil, err
	}

	return &response, nil
}

// Specific Card APIs

func (s *SalesStaffAnalyticsService) GetRevenueTrend(ctx context.Context, req *requests.SalesDashboardFilter) (map[string][]responses.SalesTimeSeriesPoint, error) {
	from, to, err := s.getDateRange(ctx, req)
	if err != nil {
		return nil, err
	}

	completedOrders := constant.ValidCompletedOrderStatus
	completedPreOrders := constant.ValidCompletedPreOrderStatus

	return s.repo.GetRevenueTrend(ctx, from, to, req.PeriodGap, completedOrders, completedPreOrders)
}

func (s *SalesStaffAnalyticsService) GetOrdersTrend(ctx context.Context, req *requests.SalesDashboardFilter) (*responses.OrdersTrendCharts, error) {
	from, to, err := s.getDateRange(ctx, req)
	if err != nil {
		return nil, err
	}

	orders, preOrders, standard, limited, err := s.repo.GetOrdersTrend(ctx, from, to, req.PeriodGap)
	if err != nil {
		return nil, err
	}

	// Sort limited trend
	sort.Slice(limited, func(i, j int) bool {
		return limited[i].Time.Before(limited[j].Time)
	})

	return &responses.OrdersTrendCharts{
		OrdersVsPreOrders: map[string][]responses.SalesTimeSeriesPoint{
			"ORDER":     orders,
			"PRE_ORDER": preOrders,
		},
		StandardVsLimited: map[string][]responses.SalesTimeSeriesPoint{
			"STANDARD": standard,
			"LIMITED":  limited,
		},
	}, nil
}

func (s *SalesStaffAnalyticsService) GetRevenueGrowth(ctx context.Context, req *requests.SalesDashboardFilter) (float64, error) {
	from, to, err := s.getDateRange(ctx, req)
	if err != nil {
		return 0, err
	}
	completedOrders := constant.ValidCompletedOrderStatus
	completedPreOrders := constant.ValidCompletedPreOrderStatus

	var currentSoldRevenue, previousSoldRevenue float64
	err = utils.RunParallel(ctx, 2,
		func(ctx context.Context) error {
			currentSoldRevenue, err = s.repo.GetTotalSoldRevenue(ctx, from, to, completedOrders, completedPreOrders)
			return nil
		},
		func(ctx context.Context) error {
			prevFrom, prevTo := s.getPreviousPeriod(from, to)
			previousSoldRevenue, err = s.repo.GetTotalSoldRevenue(ctx, prevFrom, prevTo, completedOrders, completedPreOrders)
			return nil
		},
	)
	if err != nil {
		return 0, err
	}

	zap.L().Debug("Revenue Growth Calculation",
		zap.Float64("currentSoldRevenue", currentSoldRevenue),
		zap.Float64("previousSoldRevenue", previousSoldRevenue),
		zap.Float64("growthPercentage", (currentSoldRevenue-previousSoldRevenue)/previousSoldRevenue),
	)
	if previousSoldRevenue == 0 && currentSoldRevenue > 0 {
		return 100, nil
	} else if previousSoldRevenue > 0 {
		return ((currentSoldRevenue - previousSoldRevenue) / previousSoldRevenue) * 100, nil
	}

	return 0, nil
}

// Helper methods

func (s *SalesStaffAnalyticsService) getDateRange(ctx context.Context, req *requests.SalesDashboardFilter) (time.Time, time.Time, error) {
	now := time.Now()

	// 1. If both provided, use them
	if req.FromDate != nil && req.ToDate != nil {
		return startOfDay(*req.FromDate), endOfDay(*req.ToDate), nil
	}

	// 2. If only FromDate provided → ToDate = today end
	if req.FromDate != nil {
		return startOfDay(*req.FromDate), endOfDay(now), nil
	}

	// 3. Base end = today end
	end := endOfDay(now)
	if req.ToDate != nil {
		end = endOfDay(*req.ToDate)
	}

	var start time.Time
	if req.PeriodGap != "" {
		periodGap := strings.ToLower(req.PeriodGap)

		switch periodGap {
		case "day":
			// Within a month
			start = end.AddDate(0, -1, 0)
		case "week":
			// Within 2 months
			start = end.AddDate(0, -2, 0)
		case "month":
			// Within 6 months
			start = end.AddDate(0, -6, 0)
		case "quarter":
			// Within 2 years
			start = end.AddDate(-2, 0, 0)
		case "year":
			// All time (max 3 years)
			firstOrder, err := s.repo.GetFirstOrderDate(ctx)
			if err != nil {
				return time.Time{}, time.Time{}, err
			}
			start = *firstOrder
			// Max 3 years back
			limitStart := end.AddDate(-3, 0, 0)
			if start.Before(limitStart) {
				start = limitStart
			}
		case "all":
			start = time.Time{}
			end = endOfDay(now)
		default:
			start = end.AddDate(0, -1, 0)
		}
	}
	if req.CompareWith != "" {
		switch strings.ToLower(req.CompareWith) {
		case "day":
			start = end.AddDate(0, 0, -1)
		case "week":
			start = end.AddDate(0, 0, -7)
		case "month":
			start = end.AddDate(0, -1, 0)
		case "quarter":
			start = end.AddDate(0, -3, 0)
		case "year":
			start = end.AddDate(-1, 0, 0)
		default: // Default fallback (e.g. 1 month)
			start = end.AddDate(0, -1, 0)
		}
	}

	// 🔒 Final normalization
	if !start.IsZero() {
		start = startOfDay(start)
	}
	end = endOfDay(end)

	return start, end, nil
}

func startOfDay(t time.Time) time.Time {
	return time.Date(
		t.Year(), t.Month(), t.Day(),
		0, 0, 1, 0, t.Location(),
	)
}

func endOfDay(t time.Time) time.Time {
	return time.Date(
		t.Year(), t.Month(), t.Day(),
		23, 59, 59, 0, t.Location(),
	)
}

func (s *SalesStaffAnalyticsService) getPreviousPeriod(from, to time.Time) (time.Time, time.Time) {
	duration := to.Sub(from)
	prevTo := from.Add(-time.Second)
	prevFrom := prevTo.Add(-duration)
	return prevFrom, prevTo
}

// =============================================================================
// REVENUE DETAIL SERVICE IMPLEMENTATIONS
// =============================================================================

// parseRevenueOrdersFilter parses and validates the RevenueOrdersFilter
func (s *SalesStaffAnalyticsService) parseRevenueOrdersFilter(ctx context.Context, req *requests.RevenueOrdersFilter) (from, to time.Time, page, limit int, search, sortBy, sortOrder string, err error) {
	// Parse dates
	now := time.Now()
	if req.FromDateStr != nil && *req.FromDateStr != "" {
		parsedFrom, parseErr := time.Parse("2006-01-02", *req.FromDateStr)
		if parseErr != nil {
			err = parseErr
			return
		}
		from = startOfDay(parsedFrom)
	} else {
		// Default: from beginning of time
		from = time.Date(1, 1, 1, 0, 0, 0, 0, time.UTC)
	}

	if req.ToDateStr != nil && *req.ToDateStr != "" {
		parsedTo, parseErr := time.Parse("2006-01-02", *req.ToDateStr)
		if parseErr != nil {
			err = parseErr
			return
		}
		to = endOfDay(parsedTo)
	} else {
		// Default: end of today
		to = endOfDay(now)
	}

	// Pagination defaults
	page = req.Page
	if page < 1 {
		page = 1
	}
	limit = req.Limit
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	// Search and sorting
	search = req.Search
	sortBy = req.SortBy
	if sortBy == "" {
		sortBy = "created_at"
	}
	sortOrder = req.SortOrder
	if sortOrder == "" {
		sortOrder = "desc"
	}

	return
}

// buildPagination creates a Pagination response from query results
func (s *SalesStaffAnalyticsService) buildPagination(page, limit int, total int64) *responses.Pagination {
	totalPages := int(total) / limit
	if int(total)%limit != 0 {
		totalPages++
	}

	return &responses.Pagination{
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: totalPages,
		HasNext:    page < totalPages,
		HasPrev:    page > 1,
	}
}

// GetTotalRevenueOrders returns all orders contributing to total revenue
func (s *SalesStaffAnalyticsService) GetTotalRevenueOrders(ctx context.Context, req *requests.RevenueOrdersFilter) (*responses.RevenueOrdersWithPaymentResponse, *responses.Pagination, error) {
	from, to, page, limit, search, sortBy, sortOrder, err := s.parseRevenueOrdersFilter(ctx, req)
	if err != nil {
		return nil, nil, err
	}

	completedOrders := constant.ValidCompletedOrderStatus
	completedPreOrders := constant.ValidCompletedPreOrderStatus

	items, total, totalRevenue, err := s.repo.GetTotalRevenueOrders(ctx, from, to, completedOrders, completedPreOrders, page, limit, search, sortBy, sortOrder)
	if err != nil {
		return nil, nil, err
	}

	response := &responses.RevenueOrdersWithPaymentResponse{
		RevenueType:  "TOTAL",
		TotalRevenue: totalRevenue,
		Orders:       items,
	}

	pagination := s.buildPagination(page, limit, total)

	return response, pagination, nil
}

// GetStandardRevenueOrders returns STANDARD type orders
func (s *SalesStaffAnalyticsService) GetStandardRevenueOrders(ctx context.Context, req *requests.RevenueOrdersFilter) (*responses.RevenueOrdersWithPaymentResponse, *responses.Pagination, error) {
	from, to, page, limit, search, sortBy, sortOrder, err := s.parseRevenueOrdersFilter(ctx, req)
	if err != nil {
		return nil, nil, err
	}

	completedOrders := constant.ValidCompletedOrderStatus

	items, total, totalRevenue, err := s.repo.GetStandardRevenueOrders(ctx, from, to, completedOrders, page, limit, search, sortBy, sortOrder)
	if err != nil {
		return nil, nil, err
	}

	response := &responses.RevenueOrdersWithPaymentResponse{
		RevenueType:  "STANDARD",
		TotalRevenue: totalRevenue,
		Orders:       items,
	}

	pagination := s.buildPagination(page, limit, total)

	return response, pagination, nil
}

// GetLimitedRevenueOrders returns LIMITED type orders and PreOrders
func (s *SalesStaffAnalyticsService) GetLimitedRevenueOrders(ctx context.Context, req *requests.RevenueOrdersFilter) (*responses.RevenueOrdersWithPaymentResponse, *responses.Pagination, error) {
	from, to, page, limit, search, sortBy, sortOrder, err := s.parseRevenueOrdersFilter(ctx, req)
	if err != nil {
		return nil, nil, err
	}

	completedOrders := constant.ValidCompletedOrderStatus
	completedPreOrders := constant.ValidCompletedPreOrderStatus

	items, total, totalRevenue, err := s.repo.GetLimitedRevenueOrders(ctx, from, to, completedOrders, completedPreOrders, page, limit, search, sortBy, sortOrder)
	if err != nil {
		return nil, nil, err
	}

	response := &responses.RevenueOrdersWithPaymentResponse{
		RevenueType:  "LIMITED",
		TotalRevenue: totalRevenue,
		Orders:       items,
	}

	pagination := s.buildPagination(page, limit, total)

	return response, pagination, nil
}

// GetStandardNetRevenueOrders returns STANDARD orders with net revenue (total_amount - shipping_fee)
//func (s *SalesStaffAnalyticsService) GetStandardNetRevenueOrders(ctx context.Context, req *requests.RevenueOrdersFilter) (*responses.RevenueOrdersResponse, *responses.Pagination, error) {
//	from, to, page, limit, search, sortBy, sortOrder, err := s.parseRevenueOrdersFilter(ctx, req)
//	if err != nil {
//		return nil, nil, err
//	}
//
//	completedOrders := constant.ValidCompletedOrderStatus
//
//	items, total, totalRevenue, err := s.repo.GetStandardNetRevenueOrders(ctx, from, to, completedOrders, page, limit, search, sortBy, sortOrder)
//	if err != nil {
//		return nil, nil, err
//	}
//
//	response := &responses.RevenueOrdersResponse{
//		RevenueType:  "STANDARD_NET",
//		TotalRevenue: totalRevenue,
//		Orders:       items,
//	}
//
//	pagination := s.buildPagination(page, limit, total)
//
//	return response, pagination, nil
//}

// GetLimitedNetRevenueOrders returns LIMITED orders and PreOrders with KOL net revenue calculation
// KOL Net Revenue = (item_total) * kol_percent / 100 - shipping_fee
func (s *SalesStaffAnalyticsService) GetLimitedNetRevenueOrders(ctx context.Context, req *requests.RevenueOrdersFilter) (*responses.RevenueOrdersWithPaymentResponse, *responses.Pagination, error) {
	from, to, page, limit, search, sortBy, sortOrder, err := s.parseRevenueOrdersFilter(ctx, req)
	if err != nil {
		return nil, nil, err
	}

	completedOrders := constant.ValidCompletedOrderStatus
	completedPreOrders := constant.ValidCompletedPreOrderStatus

	items, total, totalRevenue, err := s.repo.GetLimitedNetRevenueOrders(ctx, from, to, completedOrders, completedPreOrders, page, limit, search, sortBy, sortOrder)
	if err != nil {
		return nil, nil, err
	}

	response := &responses.RevenueOrdersWithPaymentResponse{
		RevenueType:  "LIMITED_NET",
		TotalRevenue: totalRevenue,
		Orders:       items,
	}

	pagination := s.buildPagination(page, limit, total)

	return response, pagination, nil
}

// GetRefundedOrders returns all refunded orders and preorders with payment transaction information
func (s *SalesStaffAnalyticsService) GetRefundedOrders(ctx context.Context, req *requests.RevenueOrdersFilter) (*responses.RevenueOrdersWithPaymentResponse, *responses.Pagination, error) {
	from, to, page, limit, search, sortBy, sortOrder, err := s.parseRevenueOrdersFilter(ctx, req)
	if err != nil {
		return nil, nil, err
	}

	refundedOrders := constant.ValidRefundedOrderStatus
	refundedPreOrders := constant.ValidRefundedPreOrderStatus

	items, total, totalRevenue, err := s.repo.GetRefundedOrders(ctx, from, to, refundedOrders, refundedPreOrders, page, limit, search, sortBy, sortOrder)
	if err != nil {
		return nil, nil, err
	}

	response := &responses.RevenueOrdersWithPaymentResponse{
		RevenueType:  "REFUNDED",
		TotalRevenue: totalRevenue,
		Orders:       items,
	}

	pagination := s.buildPagination(page, limit, total)

	return response, pagination, nil
}
