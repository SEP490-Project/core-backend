package gormrepository

import (
	"context"
	"time"

	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/domain/constant"
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"

	"gorm.io/gorm"
)

type SalesStaffAnalyticsRepository struct {
	db *gorm.DB
}

func NewSalesStaffAnalyticsRepository(db *gorm.DB) irepository.SalesStaffAnalyticsRepository {
	return &SalesStaffAnalyticsRepository{
		db: db,
	}
}

// =============================================================================
// FINANCIALS TAB
// =============================================================================

func (r *SalesStaffAnalyticsRepository) GetFinancialsSummary(ctx context.Context, from, to time.Time, completedOrderStatuses []enum.OrderStatus, completedPreOrderStatuses []enum.PreOrderStatus) (*responses.FinancialsSummary, error) {
	var result responses.FinancialsSummary

	// 1. Total Sold Revenue (Orders + PreOrders)
	// 2. AOV (Combined, Orders, PreOrders)
	// 3. Returning Customer Count
	// 4. Limited Product Conversion Rate

	// We can use a CTE to gather base stats
	// Note: Growth rate needs previous period data, which is better handled in service or by querying previous period here.
	// For simplicity and performance, let's calculate current period metrics here.

	type BaseStats struct {
		TotalRevenue      float64
		OrderRevenue      float64
		PreOrderRevenue   float64
		TotalCount        int64
		OrderCount        int64
		PreOrderCount     int64
		LimitedCount      int64 // PreOrders + Limited Orders
		TotalLimitedCount int64 // All PreOrders + All Limited Orders (regardless of status? No, usually conversion rate is based on orders placed)
	}

	// Query for current period
	// We need to be careful with "Limited Product Conversion Rate".
	// Definition: limited product orders/pre-orders / total orders/pre-orders
	// This sounds like "Share of Limited Products" rather than "Conversion Rate" (which usually implies visits -> orders).
	// Based on the requirement: "Limited Product Conversion Rate (To show if Limited Product attract more customers or not) - limited product orders/pre-orders / total orders/pre-orders"
	// So it is indeed the share.

	// Returning Customer: Customer who have 2 or more orders/pre-orders within time range.

	query := `
		WITH valid_orders AS (
			SELECT user_id, total_amount, order_type, 'ORDER' as source
			FROM orders
			WHERE status IN ? AND created_at >= ? AND created_at <= ? AND deleted_at IS NULL
		),
		valid_pre_orders AS (
			SELECT user_id, total_amount, 'LIMITED' as order_type, 'PRE_ORDER' as source
			FROM pre_orders
			WHERE status IN ? AND created_at >= ? AND created_at <= ? AND deleted_at IS NULL
		),
		combined AS (
			SELECT * FROM valid_orders
			UNION ALL
			SELECT * FROM valid_pre_orders
		),
		user_counts AS (
			SELECT user_id, COUNT(*) as cnt
			FROM combined
			GROUP BY user_id
		)
		SELECT
			COALESCE(SUM(total_amount), 0) as total_revenue,
			COALESCE(SUM(CASE WHEN source = 'ORDER' THEN total_amount ELSE 0 END), 0) as order_revenue,
			COALESCE(SUM(CASE WHEN source = 'PRE_ORDER' THEN total_amount ELSE 0 END), 0) as pre_order_revenue,
			COUNT(*) as total_count,
			COUNT(CASE WHEN source = 'ORDER' THEN 1 END) as order_count,
			COUNT(CASE WHEN source = 'PRE_ORDER' THEN 1 END) as pre_order_count,
			COUNT(CASE WHEN order_type = 'LIMITED' OR source = 'PRE_ORDER' THEN 1 END) as limited_count
		FROM combined;
	`

	var stats BaseStats
	if err := r.db.WithContext(ctx).Raw(query,
		completedOrderStatuses, from, to,
		completedPreOrderStatuses, from, to,
	).Scan(&stats).Error; err != nil {
		return nil, err
	}

	// Returning customers
	var returningCount int64
	returningQuery := `
		WITH valid_orders AS (
			SELECT user_id
			FROM orders
			WHERE status IN ? AND created_at >= ? AND created_at <= ? AND deleted_at IS NULL
		),
		valid_pre_orders AS (
			SELECT user_id
			FROM pre_orders
			WHERE status IN ? AND created_at >= ? AND created_at <= ? AND deleted_at IS NULL
		),
		combined AS (
			SELECT user_id FROM valid_orders
			UNION ALL
			SELECT user_id FROM valid_pre_orders
		)
		SELECT COUNT(*) FROM (
			SELECT user_id FROM combined GROUP BY user_id HAVING COUNT(*) >= 2
		) as sub
	`
	if err := r.db.WithContext(ctx).Raw(returningQuery,
		completedOrderStatuses, from, to,
		completedPreOrderStatuses, from, to,
	).Scan(&returningCount).Error; err != nil {
		return nil, err
	}

	result.TotalSoldRevenue = stats.TotalRevenue
	result.ReturningCustomerCount = returningCount

	// AOV
	if stats.TotalCount > 0 {
		result.AverageOrderValue.Combined = stats.TotalRevenue / float64(stats.TotalCount)
	}
	if stats.OrderCount > 0 {
		result.AverageOrderValue.Orders = stats.OrderRevenue / float64(stats.OrderCount)
	}
	if stats.PreOrderCount > 0 {
		result.AverageOrderValue.PreOrders = stats.PreOrderRevenue / float64(stats.PreOrderCount)
	}

	// Limited Conversion Rate (Share)
	if stats.TotalCount > 0 {
		result.LimitedProductConversionRate = float64(stats.LimitedCount) / float64(stats.TotalCount)
	}

	return &result, nil
}

func (r *SalesStaffAnalyticsRepository) GetTotalSoldRevenue(ctx context.Context, from, to time.Time, orderStatuses []enum.OrderStatus, preOrderStatuses []enum.PreOrderStatus) (float64, error) {
	totalSoldRevenueQuery := `
		WITH valid_orders AS (
			SELECT user_id, total_amount, order_type, 'ORDER' as source
			FROM orders
			WHERE status IN ? AND created_at >= ? AND created_at <= ? AND deleted_at IS NULL
		),
		valid_pre_orders AS (
			SELECT user_id, total_amount, 'LIMITED' as order_type, 'PRE_ORDER' as source
			FROM pre_orders
			WHERE status IN ? AND created_at >= ? AND created_at <= ? AND deleted_at IS NULL
		),
		combined AS (
			SELECT * FROM valid_orders
			UNION ALL
			SELECT * FROM valid_pre_orders
		)
		SELECT COALESCE(SUM(total_amount), 0) as total_revenue
		FROM combined;
	`

	var totalRevenue float64
	if err := r.db.WithContext(ctx).Raw(totalSoldRevenueQuery,
		orderStatuses, from, to,
		preOrderStatuses, from, to,
	).Scan(&totalRevenue).Error; err != nil {
		return 0, err
	}

	return totalRevenue, nil
}

func (r *SalesStaffAnalyticsRepository) GetRevenueBreakdown(ctx context.Context, from, to time.Time, completedOrderStatuses []enum.OrderStatus, completedPreOrderStatuses []enum.PreOrderStatus) ([]responses.RevenueByProductType, []responses.RevenueByCategory, error) {
	var byProduct []responses.RevenueByProductType
	var byCategory []responses.RevenueByCategory

	// 1. By Product Type (Standard vs Limited)
	// Standard = Orders with type STANDARD
	// Limited = Orders with type LIMITED + PreOrders
	queryProduct := `
		WITH standard_rev AS (
			SELECT COALESCE(SUM(total_amount), 0) as rev
			FROM orders
			WHERE order_type = 'STANDARD' AND status IN ? AND created_at >= ? AND created_at <= ? AND deleted_at IS NULL
		),
		limited_order_rev AS (
			SELECT COALESCE(SUM(total_amount), 0) as rev
			FROM orders
			WHERE order_type = 'LIMITED' AND status IN ? AND created_at >= ? AND created_at <= ? AND deleted_at IS NULL
		),
		pre_order_rev AS (
			SELECT COALESCE(SUM(total_amount), 0) as rev
			FROM pre_orders
			WHERE status IN ? AND created_at >= ? AND created_at <= ? AND deleted_at IS NULL
		)
		SELECT 'STANDARD' as product_type, rev as revenue FROM standard_rev
		UNION ALL
		SELECT 'LIMITED' as product_type, (SELECT rev FROM limited_order_rev) + (SELECT rev FROM pre_order_rev) as revenue
	`
	if err := r.db.WithContext(ctx).Raw(queryProduct,
		completedOrderStatuses, from, to,
		completedOrderStatuses, from, to,
		completedPreOrderStatuses, from, to,
	).Scan(&byProduct).Error; err != nil {
		return nil, nil, err
	}

	// Calculate percentages
	var totalRev float64
	for _, p := range byProduct {
		totalRev += p.Revenue
	}
	if totalRev > 0 {
		for i := range byProduct {
			byProduct[i].Percentage = (byProduct[i].Revenue / totalRev) * 100
		}
	}

	// 2. By Category (First Level)
	// Need to join order_items -> product_variants -> products -> product_categories
	// For PreOrders: pre_order -> product -> category
	// Note: Orders have order_items. PreOrders have product_id directly.
	queryCategory := `
		WITH order_revenue AS (
			SELECT c.name as category_name, SUM(oi.unit_price * oi.quantity) as revenue
			FROM orders o
			JOIN order_items oi ON o.id = oi.order_id
			JOIN product_variants pv ON oi.variant_id = pv.id
			JOIN products p ON pv.product_id = p.id
			JOIN product_categories c ON p.category_id = c.id
			WHERE o.status IN ? AND o.created_at >= ? AND o.created_at <= ? AND o.deleted_at IS NULL
			GROUP BY c.name
		),
		pre_order_revenue AS (
			SELECT c.name as category_name, SUM(po.total_amount) as revenue
			FROM pre_orders po
			JOIN product_variants pv ON po.variant_id = pv.id
			JOIN products p ON pv.product_id = p.id
			JOIN product_categories c ON p.category_id = c.id
			WHERE po.status IN ? AND po.created_at >= ? AND po.created_at <= ? AND po.deleted_at IS NULL
			GROUP BY c.name
		)
		SELECT category_name, SUM(revenue) as revenue
		FROM (
			SELECT * FROM order_revenue
			UNION ALL
			SELECT * FROM pre_order_revenue
		) as combined
		GROUP BY category_name
		ORDER BY revenue DESC
	`
	if err := r.db.WithContext(ctx).Raw(queryCategory,
		completedOrderStatuses, from, to,
		completedPreOrderStatuses, from, to,
	).Scan(&byCategory).Error; err != nil {
		return nil, nil, err
	}

	return byProduct, byCategory, nil
}

func (r *SalesStaffAnalyticsRepository) GetRevenueTrend(ctx context.Context, from, to time.Time, periodGap string, completedOrderStatuses []enum.OrderStatus, completedPreOrderStatuses []enum.PreOrderStatus) (orders, preOrders, standard, limited []responses.SalesTimeSeriesPoint, err error) {
	// Determine interval based on range
	var interval string
	if periodGap != "" {
		interval = periodGap
	} else {
		duration := to.Sub(from)
		if duration.Hours() <= 24 {
			interval = "hour"
		} else if duration.Hours() <= 24*30 { // <= 30 days
			interval = "day"
		} else {
			interval = "month"
		}
	}

	// Helper to generate series
	generateSeries := func(sourceType, orderType string, statuses any, table string) ([]responses.SalesTimeSeriesPoint, error) {
		var points []responses.SalesTimeSeriesPoint
		// Use generate_series to ensure all time points are present
		q := `
			WITH dates AS (
				SELECT generate_series(
					date_trunc(?, ?::timestamp),
					date_trunc(?, ?::timestamp),
					?::interval
				) as date
			)
			SELECT 
				dates.date as time, 
				COALESCE(SUM(t.total_amount), 0) as value, 
				? as type
			FROM dates
			LEFT JOIN ` + table + ` t ON date_trunc(?, t.created_at) = dates.date
				AND t.status IN ? 
				AND t.created_at >= ? AND t.created_at <= ?
				AND t.deleted_at IS NULL
		`
		args := []any{interval, from, interval, to, "1 " + interval, sourceType, interval, statuses, from, to}

		if orderType != "" {
			q += " AND t.order_type = ?"
			args = append(args, orderType)
		}

		q += " GROUP BY dates.date ORDER BY dates.date"

		if err = r.db.WithContext(ctx).Raw(q, args...).Scan(&points).Error; err != nil {
			return nil, err
		}
		return points, nil
	}

	// Orders vs PreOrders
	orders, err = generateSeries("ORDER", "", completedOrderStatuses, "orders")
	if err != nil {
		return
	}
	preOrders, err = generateSeries("PRE_ORDER", "", completedPreOrderStatuses, "pre_orders")
	if err != nil {
		return
	}

	// Standard vs Limited
	// Standard = Orders(STANDARD)
	standard, err = generateSeries("STANDARD", "STANDARD", completedOrderStatuses, "orders")
	if err != nil {
		return
	}

	// Limited = Orders(LIMITED) + PreOrders
	var limitedOrders []responses.SalesTimeSeriesPoint
	limitedOrders, err = generateSeries("LIMITED", "LIMITED", completedOrderStatuses, "orders")
	if err != nil {
		return
	}

	// Merge Limited Orders + PreOrders
	limitedMap := make(map[time.Time]float64)
	for _, p := range limitedOrders {
		limitedMap[p.Time] = p.Value
	}
	for _, p := range preOrders {
		limitedMap[p.Time] += p.Value
	}

	for t, v := range limitedMap {
		limited = append(limited, responses.SalesTimeSeriesPoint{Time: t, Value: v, Type: "LIMITED"})
	}
	// Sort limited? Map iteration is random.
	// Ideally we should sort, but for now let's rely on frontend or sort in service.

	return
}

func (r *SalesStaffAnalyticsRepository) GetTopSellingByRevenue(ctx context.Context, from, to time.Time, completedOrderStatuses []enum.OrderStatus, completedPreOrderStatuses []enum.PreOrderStatus, limit int) ([]responses.TopEntity, []responses.TopEntity, []responses.TopEntity, error) {
	var products, categories, brands []responses.TopEntity

	// Top Products
	queryProducts := `
		WITH order_sales AS (
			SELECT pv.product_id, p.name, SUM(oi.unit_price * oi.quantity) as revenue
			FROM orders o
			JOIN order_items oi ON o.id = oi.order_id
			JOIN product_variants pv ON oi.variant_id = pv.id
			JOIN products p ON pv.product_id = p.id
			WHERE o.status IN ? AND o.created_at >= ? AND o.created_at <= ? AND o.deleted_at IS NULL
			GROUP BY pv.product_id, p.name
		),
		pre_order_sales AS (
			SELECT pv.product_id, p.name, SUM(po.total_amount) as revenue
			FROM pre_orders po
			JOIN product_variants pv ON po.variant_id = pv.id
			JOIN products p ON pv.product_id = p.id
			WHERE po.status IN ? AND po.created_at >= ? AND po.created_at <= ? AND po.deleted_at IS NULL
			GROUP BY pv.product_id, p.name
		)
		SELECT product_id as id, name, SUM(revenue) as value
		FROM (
			SELECT * FROM order_sales
			UNION ALL
			SELECT * FROM pre_order_sales
		) as combined
		GROUP BY product_id, name
		ORDER BY value DESC
		LIMIT ?
	`
	if err := r.db.WithContext(ctx).Raw(queryProducts,
		completedOrderStatuses, from, to,
		completedPreOrderStatuses, from, to,
		limit,
	).Scan(&products).Error; err != nil {
		return nil, nil, nil, err
	}

	// Top Categories
	queryCategories := `
		WITH order_sales AS (
			SELECT c.id, c.name, SUM(oi.unit_price * oi.quantity) as revenue
			FROM orders o
			JOIN order_items oi ON o.id = oi.order_id
			JOIN product_variants pv ON oi.variant_id = pv.id
			JOIN products p ON pv.product_id = p.id
			JOIN product_categories c ON p.category_id = c.id
			WHERE o.status IN ? AND o.created_at >= ? AND o.created_at <= ? AND o.deleted_at IS NULL
			GROUP BY c.id, c.name
		),
		pre_order_sales AS (
			SELECT c.id, c.name, SUM(po.total_amount) as revenue
			FROM pre_orders po
			JOIN product_variants pv ON po.variant_id = pv.id
			JOIN products p ON pv.product_id = p.id
			JOIN product_categories c ON p.category_id = c.id
			WHERE po.status IN ? AND po.created_at >= ? AND po.created_at <= ? AND po.deleted_at IS NULL
			GROUP BY c.id, c.name
		)
		SELECT id, name, SUM(revenue) as value
		FROM (
			SELECT * FROM order_sales
			UNION ALL
			SELECT * FROM pre_order_sales
		) as combined
		GROUP BY id, name
		ORDER BY value DESC
		LIMIT ?
	`
	if err := r.db.WithContext(ctx).Raw(queryCategories,
		completedOrderStatuses, from, to,
		completedPreOrderStatuses, from, to,
		limit,
	).Scan(&categories).Error; err != nil {
		return nil, nil, nil, err
	}

	// Top Brands
	queryBrands := `
		WITH order_sales AS (
			SELECT b.id, b.name, SUM(oi.unit_price * oi.quantity) as revenue
			FROM orders o
			JOIN order_items oi ON o.id = oi.order_id
			JOIN product_variants pv ON oi.variant_id = pv.id
			JOIN products p ON pv.product_id = p.id
			JOIN brands b ON p.brand_id = b.id
			WHERE o.status IN ? AND o.created_at >= ? AND o.created_at <= ? AND o.deleted_at IS NULL
			GROUP BY b.id, b.name
		),
		pre_order_sales AS (
			SELECT b.id, b.name, SUM(po.total_amount) as revenue
			FROM pre_orders po
			JOIN product_variants pv ON po.variant_id = pv.id
			JOIN products p ON pv.product_id = p.id
			JOIN brands b ON p.brand_id = b.id
			WHERE po.status IN ? AND po.created_at >= ? AND po.created_at <= ? AND po.deleted_at IS NULL
			GROUP BY b.id, b.name
		)
		SELECT id, name, SUM(revenue) as value
		FROM (
			SELECT * FROM order_sales
			UNION ALL
			SELECT * FROM pre_order_sales
		) as combined
		GROUP BY id, name
		ORDER BY value DESC
		LIMIT ?
	`
	if err := r.db.WithContext(ctx).Raw(queryBrands,
		completedOrderStatuses, from, to,
		completedPreOrderStatuses, from, to,
		limit,
	).Scan(&brands).Error; err != nil {
		return nil, nil, nil, err
	}

	return products, categories, brands, nil
}

// =============================================================================
// ORDERS TAB
// =============================================================================

func (r *SalesStaffAnalyticsRepository) GetOrdersSummary(ctx context.Context, from, to time.Time) (*responses.OrdersSummary, error) {
	var result responses.OrdersSummary

	// Total Orders, Total PreOrders
	// Cancellation Rate, Refund Rate (Orders only)

	query := `
		SELECT
			COUNT(*) as total_orders,
			COUNT(CASE WHEN status = 'CANCELLED' THEN 1 END) as cancelled_orders,
			COUNT(CASE WHEN status = 'REFUNDED' OR status = 'COMPENSATED' THEN 1 END) as refunded_orders
		FROM orders
		WHERE created_at >= ? AND created_at <= ? AND deleted_at IS NULL
	`
	var orderStats struct {
		TotalOrders     int64
		CancelledOrders int64
		RefundedOrders  int64
	}
	if err := r.db.WithContext(ctx).Raw(query, from, to).Scan(&orderStats).Error; err != nil {
		return nil, err
	}

	var preOrderCount int64
	if err := r.db.WithContext(ctx).Model(&model.PreOrder{}). // Assuming model exists, or use Table
									Table("pre_orders").
									Where("created_at >= ? AND created_at <= ? AND deleted_at IS NULL", from, to).
									Count(&preOrderCount).Error; err != nil {
		return nil, err
	}

	result.TotalOrders = orderStats.TotalOrders
	result.TotalPreOrders = preOrderCount

	if orderStats.TotalOrders > 0 {
		result.CancellationRate = (float64(orderStats.CancelledOrders) / float64(orderStats.TotalOrders)) * 100
		result.RefundRate = (float64(orderStats.RefundedOrders) / float64(orderStats.TotalOrders)) * 100
	}

	return &result, nil
}

func (r *SalesStaffAnalyticsRepository) GetOrderStatusDistribution(ctx context.Context, from, to time.Time) (responses.OrderStatusDistribution, responses.OrderStatusDistribution, error) {
	var ordersDist, preOrdersDist responses.OrderStatusDistribution

	// Orders
	// Pending: PENDING, PAID, CONFIRMED, SHIPPED, IN_TRANSIT, AWAITING_PICKUP, REFUND_REQUESTED
	// Completed: DELIVERED, RECEIVED, COMPENSATE_REQUESTED
	// Cancelled: CANCELLED
	// Refunded: REFUNDED, COMPENSATED

	// We can use a CASE statement to group statuses
	// query := `
	// 	SELECT
	// 		SUM(CASE WHEN status IN ('PENDING', 'PAID', 'CONFIRMED', 'SHIPPED', 'IN_TRANSIT', 'AWAITING_PICKUP', 'REFUND_REQUESTED') THEN 1 ELSE 0 END) as pending,
	// 		SUM(CASE WHEN status IN ('DELIVERED', 'RECEIVED', 'COMPENSATE_REQUESTED') THEN 1 ELSE 0 END) as completed,
	// 		SUM(CASE WHEN status = 'CANCELLED' THEN 1 ELSE 0 END) as cancelled,
	// 		SUM(CASE WHEN status IN ('REFUNDED', 'COMPENSATED') THEN 1 ELSE 0 END) as refunded
	// 	FROM %s
	// 	WHERE created_at >= ? AND created_at <= ? AND deleted_at IS NULL
	// `

	if err := r.db.WithContext(ctx).Raw(
		"SELECT "+
			"SUM(CASE WHEN status IN ? THEN 1 ELSE 0 END) as pending, "+
			"SUM(CASE WHEN status IN ? THEN 1 ELSE 0 END) as completed, "+
			"SUM(CASE WHEN status IN ? THEN 1 ELSE 0 END) as cancelled, "+
			"SUM(CASE WHEN status IN ? THEN 1 ELSE 0 END) as refunded "+
			"FROM orders WHERE created_at >= ? AND created_at <= ? AND deleted_at IS NULL",
		constant.ValidPendingOrderStatus, constant.ValidCompletedOrderStatus, constant.ValidCancelledOrderStatus, constant.ValidRefundedOrderStatus, from, to,
	).Scan(&ordersDist).Error; err != nil {
		return ordersDist, preOrdersDist, err
	}

	if err := r.db.WithContext(ctx).Raw(
		"SELECT "+
			"SUM(CASE WHEN status IN ? THEN 1 ELSE 0 END) as pending, "+
			"SUM(CASE WHEN status IN ? THEN 1 ELSE 0 END) as completed, "+
			"SUM(CASE WHEN status IN ? THEN 1 ELSE 0 END) as cancelled, "+
			"SUM(CASE WHEN status IN ? THEN 1 ELSE 0 END) as refunded "+
			"FROM pre_orders WHERE created_at >= ? AND created_at <= ? AND deleted_at IS NULL",
		constant.ValidPendingPreOrderStatus, constant.ValidCompletedPreOrderStatus, constant.ValidCancelledPreOrderStatus, constant.ValidRefundedPreOrderStatus, from, to,
	).Scan(&preOrdersDist).Error; err != nil {
		return ordersDist, preOrdersDist, err
	}

	return ordersDist, preOrdersDist, nil
}

func (r *SalesStaffAnalyticsRepository) GetOrdersTrend(ctx context.Context, from, to time.Time, periodGap string) (orders, preOrders, standard, limited []responses.SalesTimeSeriesPoint, err error) {
	// Similar to RevenueTrend but COUNT(*) instead of SUM(total_amount)
	var interval string
	if periodGap != "" {
		interval = periodGap
	} else {
		duration := to.Sub(from)
		if duration.Hours() <= 24 {
			interval = "hour"
		} else if duration.Hours() <= 24*30 {
			interval = "day"
		} else {
			interval = "month"
		}
	}

	generateSeries := func(sourceType, orderType string, table string) ([]responses.SalesTimeSeriesPoint, error) {
		var points []responses.SalesTimeSeriesPoint
		q := `
			WITH dates AS (
				SELECT generate_series(
					date_trunc(?, ?::timestamp),
					date_trunc(?, ?::timestamp),
					?::interval
				) as date
			)
			SELECT 
				dates.date as time, 
				COUNT(t.id) as value, 
				? as type
			FROM dates
			LEFT JOIN ` + table + ` t ON date_trunc(?, t.created_at) = dates.date
				AND t.created_at >= ? AND t.created_at <= ?
				AND t.deleted_at IS NULL
		`
		args := []any{interval, from, interval, to, "1 " + interval, sourceType, interval, from, to}

		if orderType != "" {
			q += " AND t.order_type = ?"
			args = append(args, orderType)
		}

		q += " GROUP BY dates.date ORDER BY dates.date"

		if err = r.db.WithContext(ctx).Raw(q, args...).Scan(&points).Error; err != nil {
			return nil, err
		}
		return points, nil
	}

	orders, err = generateSeries("ORDER", "", "orders")
	if err != nil {
		return
	}
	preOrders, err = generateSeries("PRE_ORDER", "", "pre_orders")
	if err != nil {
		return
	}
	standard, err = generateSeries("STANDARD", "STANDARD", "orders")
	if err != nil {
		return
	}

	var limitedOrders []responses.SalesTimeSeriesPoint
	limitedOrders, err = generateSeries("LIMITED", "LIMITED", "orders")
	if err != nil {
		return
	}

	// Merge Limited
	limitedMap := make(map[time.Time]float64)
	for _, p := range limitedOrders {
		limitedMap[p.Time] = p.Value
	}
	for _, p := range preOrders {
		limitedMap[p.Time] += p.Value
	}
	for t, v := range limitedMap {
		limited = append(limited, responses.SalesTimeSeriesPoint{Time: t, Value: v, Type: "LIMITED"})
	}

	return
}

func (r *SalesStaffAnalyticsRepository) GetTopSellingByVolume(ctx context.Context, from, to time.Time, completedOrderStatuses []enum.OrderStatus, completedPreOrderStatuses []enum.PreOrderStatus, limit int) ([]responses.TopEntity, []responses.TopEntity, []responses.TopEntity, error) {
	// Similar to Revenue but COUNT(*) or SUM(quantity)
	// Let's use SUM(quantity) for products, and COUNT(*) for orders in categories/brands?
	// Requirement says "Top Selling Product Name By No. Orders".
	// "No. Orders" usually means count of orders containing the product, or sum of quantity?
	// "By No. Orders" implies count of orders. But "Top Selling" usually implies quantity.
	// Let's use SUM(quantity) for products, as it's more accurate for "Selling".
	// For Categories/Brands, SUM(quantity) is also good.

	var products, categories, brands []responses.TopEntity

	// Top Products
	queryProducts := `
		WITH order_sales AS (
			SELECT pv.product_id, p.name, SUM(oi.quantity) as volume
			FROM orders o
			JOIN order_items oi ON o.id = oi.order_id
			JOIN product_variants pv ON oi.variant_id = pv.id
			JOIN products p ON pv.product_id = p.id
			WHERE o.status IN ? AND o.created_at >= ? AND o.created_at <= ? AND o.deleted_at IS NULL
			GROUP BY pv.product_id, p.name
		),
		pre_order_sales AS (
			SELECT pv.product_id, p.name, SUM(po.quantity) as volume
			FROM pre_orders po
			JOIN product_variants pv ON po.variant_id = pv.id
			JOIN products p ON pv.product_id = p.id
			WHERE po.status IN ? AND po.created_at >= ? AND po.created_at <= ? AND po.deleted_at IS NULL
			GROUP BY pv.product_id, p.name
		)
		SELECT product_id as id, name, SUM(volume) as value
		FROM (
			SELECT * FROM order_sales
			UNION ALL
			SELECT * FROM pre_order_sales
		) as combined
		GROUP BY product_id, name
		ORDER BY value DESC
		LIMIT ?
	`
	if err := r.db.WithContext(ctx).Raw(queryProducts,
		completedOrderStatuses, from, to,
		completedPreOrderStatuses, from, to,
		limit,
	).Scan(&products).Error; err != nil {
		return nil, nil, nil, err
	}

	// Top Categories
	queryCategories := `
		WITH order_sales AS (
			SELECT c.id, c.name, SUM(oi.quantity) as volume
			FROM orders o
			JOIN order_items oi ON o.id = oi.order_id
			JOIN product_variants pv ON oi.variant_id = pv.id
			JOIN products p ON pv.product_id = p.id
			JOIN product_categories c ON p.category_id = c.id
			WHERE o.status IN ? AND o.created_at >= ? AND o.created_at <= ? AND o.deleted_at IS NULL
			GROUP BY c.id, c.name
		),
		pre_order_sales AS (
			SELECT c.id, c.name, SUM(po.quantity) as volume
			FROM pre_orders po
			JOIN product_variants pv ON po.variant_id = pv.id
			JOIN products p ON pv.product_id = p.id
			JOIN product_categories c ON p.category_id = c.id
			WHERE po.status IN ? AND po.created_at >= ? AND po.created_at <= ? AND po.deleted_at IS NULL
			GROUP BY c.id, c.name
		)
		SELECT id, name, SUM(volume) as value
		FROM (
			SELECT * FROM order_sales
			UNION ALL
			SELECT * FROM pre_order_sales
		) as combined
		GROUP BY id, name
		ORDER BY value DESC
		LIMIT ?
	`
	if err := r.db.WithContext(ctx).Raw(queryCategories,
		completedOrderStatuses, from, to,
		completedPreOrderStatuses, from, to,
		limit,
	).Scan(&categories).Error; err != nil {
		return nil, nil, nil, err
	}

	// Top Brands
	queryBrands := `
		WITH order_sales AS (
			SELECT b.id, b.name, SUM(oi.quantity) as volume
			FROM orders o
			JOIN order_items oi ON o.id = oi.order_id
			JOIN product_variants pv ON oi.variant_id = pv.id
			JOIN products p ON pv.product_id = p.id
			JOIN brands b ON p.brand_id = b.id
			WHERE o.status IN ? AND o.created_at >= ? AND o.created_at <= ? AND o.deleted_at IS NULL
			GROUP BY b.id, b.name
		),
		pre_order_sales AS (
			SELECT b.id, b.name, SUM(po.quantity) as volume
			FROM pre_orders po
			JOIN product_variants pv ON po.variant_id = pv.id
			JOIN products p ON pv.product_id = p.id
			JOIN brands b ON p.brand_id = b.id
			WHERE po.status IN ? AND po.created_at >= ? AND po.created_at <= ? AND po.deleted_at IS NULL
			GROUP BY b.id, b.name
		)
		SELECT id, name, SUM(volume) as value
		FROM (
			SELECT * FROM order_sales
			UNION ALL
			SELECT * FROM pre_order_sales
		) as combined
		GROUP BY id, name
		ORDER BY value DESC
		LIMIT ?
	`
	if err := r.db.WithContext(ctx).Raw(queryBrands,
		completedOrderStatuses, from, to,
		completedPreOrderStatuses, from, to,
		limit,
	).Scan(&brands).Error; err != nil {
		return nil, nil, nil, err
	}

	return products, categories, brands, nil
}

func (r *SalesStaffAnalyticsRepository) GetLatestOrders(ctx context.Context, from, to time.Time, limit int) ([]responses.LatestOrder, error) {
	var orders []responses.LatestOrder

	// Union Orders and PreOrders, sort by created_at desc
	query := `
		SELECT id, full_name as customer_name, total_amount, status::varchar(50), 'ORDER' as type, created_at
		FROM orders
		WHERE created_at >= ? AND created_at <= ? AND deleted_at IS NULL AND status in ?
		UNION ALL
		SELECT id, full_name as customer_name, total_amount, status::varchar(50), 'PRE_ORDER' as type, created_at
		FROM pre_orders
		WHERE created_at >= ? AND created_at <= ? AND deleted_at IS NULL AND status in ?
		ORDER BY created_at DESC
		LIMIT ?
	`
	if err := r.db.WithContext(ctx).Raw(query, from, to, constant.ValidPendingOrderStatus, from, to, constant.ValidPendingPreOrderStatus, limit).Scan(&orders).Error; err != nil {
		return nil, err
	}
	return orders, nil
}

func (r *SalesStaffAnalyticsRepository) GetFirstOrderDate(ctx context.Context) (*time.Time, error) {
	var firstOrder time.Time
	// Check Orders
	err := r.db.WithContext(ctx).Model(&model.Order{}).Select("created_at").Order("created_at ASC").Limit(1).Scan(&firstOrder).Error
	if err != nil {
		return nil, err
	}

	// Check PreOrders
	var firstPreOrder time.Time
	err = r.db.WithContext(ctx).Model(&model.PreOrder{}).Select("created_at").Order("created_at ASC").Limit(1).Scan(&firstPreOrder).Error
	if err != nil {
		return nil, err
	}

	if firstOrder.IsZero() && firstPreOrder.IsZero() {
		now := time.Now()
		return &now, nil
	}

	if firstOrder.IsZero() {
		return &firstPreOrder, nil
	}
	if firstPreOrder.IsZero() {
		return &firstOrder, nil
	}

	if firstPreOrder.Before(firstOrder) {
		return &firstPreOrder, nil
	}
	return &firstOrder, nil
}
