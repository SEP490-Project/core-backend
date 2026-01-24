package gormrepository

import (
	"context"
	"fmt"
	"slices"
	"time"

	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/domain/constant"
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
	"core-backend/pkg/utils"

	"github.com/google/uuid"
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

	// We can use a CTE to gather base stats
	// Note: Growth rate needs previous period data, which is better handled in service or by querying previous period here.
	// For simplicity and performance, let's calculate current period metrics here.

	// =========================================================================
	// QUERY 1: Financials & Basic Counts (Revenue, AOV, Limited Counts)
	// =========================================================================
	type BaseStats struct {
		TotalRevenue         float64
		TotalStandardRevenue float64
		TotalLimitedRevenue  float64
		StandardNetRevenue   float64 // Standard orders without shipping fee
		LimitedGrossRevenue  float64 // Limited orders + PreOrders (including shipping)
		LimitedNetRevenue    float64 // Limited orders + PreOrders * KOL percentage (without shipping)
		OrderRevenue         float64
		PreOrderRevenue      float64
		TotalCount           int64
		OrderCount           int64
		PreOrderCount        int64
		LimitedCount         int64 // PreOrders + Limited Orders
	}

	var stats BaseStats
	var returningCount, newCount int64
	var totalRefund float64

	err := utils.RunParallel(ctx, 4,
		func(ctx context.Context) error {
			query := `
		WITH valid_orders AS (
			SELECT user_id, total_amount, order_type, shipping_fee, 'ORDER' as source
			FROM orders
			WHERE status IN ? 
				AND (?::timestamp IS NULL OR ? = TIMESTAMP '0001-01-01 00:00:00' OR created_at >= ?)
				AND (?::timestamp IS NULL OR ? = TIMESTAMP '0001-01-01 00:00:00' OR created_at <= ?)
				AND deleted_at IS NULL
		),
		valid_pre_orders AS (
			SELECT user_id, total_amount, 'LIMITED' as order_type, shipping_fee, 'PRE_ORDER' as source
			FROM pre_orders
			WHERE status IN ? 
				AND (?::timestamp IS NULL OR ? = TIMESTAMP '0001-01-01 00:00:00' OR created_at >= ?)
				AND (?::timestamp IS NULL OR ? = TIMESTAMP '0001-01-01 00:00:00' OR created_at <= ?)
				AND deleted_at IS NULL
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
			COALESCE(SUM(CASE WHEN order_type = 'STANDARD' THEN total_amount ELSE 0 END), 0) as total_standard_revenue,
			COALESCE(SUM(CASE WHEN order_type = 'LIMITED' OR source = 'PRE_ORDER' THEN total_amount ELSE 0 END), 0) as total_limited_revenue,
			COALESCE(SUM(CASE WHEN order_type = 'STANDARD' THEN total_amount - shipping_fee ELSE 0 END), 0) as standard_net_revenue,
			COALESCE(SUM(CASE WHEN order_type = 'LIMITED' OR source = 'PRE_ORDER' THEN total_amount ELSE 0 END), 0) as limited_gross_revenue,
			COALESCE(SUM(CASE WHEN source = 'ORDER' THEN total_amount ELSE 0 END), 0) as order_revenue,
			COALESCE(SUM(CASE WHEN source = 'PRE_ORDER' THEN total_amount ELSE 0 END), 0) as pre_order_revenue,
			COUNT(*) as total_count,
			COUNT(CASE WHEN source = 'ORDER' THEN 1 END) as order_count,
			COUNT(CASE WHEN source = 'PRE_ORDER' THEN 1 END) as pre_order_count,
			COUNT(CASE WHEN order_type = 'LIMITED' OR source = 'PRE_ORDER' THEN 1 END) as limited_count
		FROM combined;
	`
			return r.db.WithContext(ctx).Raw(query,
				completedOrderStatuses, from, from, from, to, to, to,
				completedPreOrderStatuses, from, from, from, to, to, to,
			).Scan(&stats).Error
		},
		func(ctx context.Context) error {
			// Calculate Limited Net Revenue (KOL share) for Orders with LIMITED type and PreOrders
			// Join through: order_items -> product_variants -> products -> tasks -> milestones -> campaigns -> contracts
			// KOL Net Revenue = (item_total) * kol_percent / 100 - shipping_fee
			// Note: Shipping fee is covered by company policy, so it's subtracted from KOL's revenue
			limitedOrderNetQuery := `
		WITH limited_order_items AS (
			SELECT 
				o.id as order_id,
				SUM(oi.unit_price * oi.quantity) as item_total,
				o.shipping_fee as shipping_fee,
				COALESCE((c.financial_terms->>'profit_split_kol_percent')::float, 0) as kol_percent
			FROM orders o
			JOIN order_items oi ON o.id = oi.order_id
			JOIN product_variants pv ON oi.variant_id = pv.id
			JOIN products p ON pv.product_id = p.id
			LEFT JOIN tasks t ON p.task_id = t.id
			LEFT JOIN milestones m ON t.milestone_id = m.id
			LEFT JOIN campaigns camp ON m.campaign_id = camp.id
			LEFT JOIN contracts c ON camp.contract_id = c.id
			WHERE o.status IN ?
				AND o.order_type = 'LIMITED'
				AND (?::timestamp IS NULL OR ? = TIMESTAMP '0001-01-01 00:00:00' OR o.created_at >= ?)
				AND (?::timestamp IS NULL OR ? = TIMESTAMP '0001-01-01 00:00:00' OR o.created_at <= ?)
				AND o.deleted_at IS NULL
			GROUP BY o.id, o.shipping_fee, c.financial_terms
		),
		pre_order_items AS (
			SELECT 
				po.id as order_id,
				po.total_amount as item_total,
				po.shipping_fee as shipping_fee,
				COALESCE((c.financial_terms->>'profit_split_kol_percent')::float, 0) as kol_percent
			FROM pre_orders po
			JOIN product_variants pv ON po.variant_id = pv.id
			JOIN products p ON pv.product_id = p.id
			LEFT JOIN tasks t ON p.task_id = t.id
			LEFT JOIN milestones m ON t.milestone_id = m.id
			LEFT JOIN campaigns camp ON m.campaign_id = camp.id
			LEFT JOIN contracts c ON camp.contract_id = c.id
			WHERE po.status IN ?
				AND (?::timestamp IS NULL OR ? = TIMESTAMP '0001-01-01 00:00:00' OR po.created_at >= ?)
				AND (?::timestamp IS NULL OR ? = TIMESTAMP '0001-01-01 00:00:00' OR po.created_at <= ?)
				AND po.deleted_at IS NULL
		),
		combined AS (
			SELECT item_total, shipping_fee, kol_percent FROM limited_order_items
			UNION ALL
			SELECT item_total, shipping_fee, kol_percent FROM pre_order_items
		)
		SELECT COALESCE(SUM((item_total * kol_percent / 100.0) - COALESCE(shipping_fee, 0)), 0)
		FROM combined
	`
			return r.db.WithContext(ctx).Raw(limitedOrderNetQuery,
				completedOrderStatuses, from, from, from, to, to, to,
				completedPreOrderStatuses, from, from, from, to, to, to,
			).Scan(&stats.LimitedNetRevenue).Error
		},
		func(ctx context.Context) error {
			returningQuery := `
    WITH valid_orders AS (
        SELECT user_id, created_at
        FROM orders
        WHERE status IN ? 
            AND (?::timestamp IS NULL OR ? = TIMESTAMP '0001-01-01 00:00:00' OR created_at <= ?)
            AND deleted_at IS NULL
    ),
    valid_pre_orders AS (
        SELECT user_id, created_at
        FROM pre_orders
        WHERE status IN ? 
            AND (?::timestamp IS NULL OR ? = TIMESTAMP '0001-01-01 00:00:00' OR created_at <= ?)
            AND deleted_at IS NULL
    ),
    combined AS (
        SELECT user_id, created_at FROM valid_orders
        UNION ALL
        SELECT user_id, created_at FROM valid_pre_orders
    ),
    user_stats AS (
        SELECT 
            user_id,
            MIN(created_at) as first_seen_date,
            COUNT(CASE 
                WHEN (?::timestamp IS NULL OR ? = TIMESTAMP '0001-01-01 00:00:00' OR created_at >= ?) THEN 1 
            END) as tx_in_period
        FROM combined
        GROUP BY user_id
    )
    SELECT 
        COUNT(CASE 
            WHEN tx_in_period > 0 AND first_seen_date < ? THEN 1 
        END) as returning_count,
        COUNT(CASE 
            WHEN tx_in_period > 0 AND first_seen_date >= ? THEN 1 
        END) as new_count
    FROM user_stats
`
			return r.db.WithContext(ctx).Raw(returningQuery,
				// valid_orders
				completedOrderStatuses, to, to, to,
				// valid_pre_orders
				completedPreOrderStatuses, to, to, to,
				// tx_in_period
				from, from, from,
				// final selection
				from, from,
			).Row().Scan(&returningCount, &newCount)
		},
		func(ctx context.Context) error {
			refundQuery := `
		WITH refunded_orders AS (
			SELECT total_amount
			FROM orders
			WHERE status IN ?
				AND (?::timestamp IS NULL OR ? = TIMESTAMP '0001-01-01 00:00:00' OR created_at >= ?)
				AND (?::timestamp IS NULL OR ? = TIMESTAMP '0001-01-01 00:00:00' OR created_at <= ?)
				AND deleted_at IS NULL
		),
		refunded_pre_orders AS (
			SELECT total_amount
			FROM pre_orders
			WHERE status IN ?
				AND (?::timestamp IS NULL OR ? = TIMESTAMP '0001-01-01 00:00:00' OR created_at >= ?)
				AND (?::timestamp IS NULL OR ? = TIMESTAMP '0001-01-01 00:00:00' OR created_at <= ?)
				AND deleted_at IS NULL
		),
		combined_refunds AS (
			SELECT total_amount FROM refunded_orders
			UNION ALL
			SELECT total_amount FROM refunded_pre_orders
		)
		SELECT COALESCE(SUM(total_amount), 0) FROM combined_refunds
	`
			return r.db.WithContext(ctx).Raw(refundQuery,
				constant.ValidRefundedOrderStatus, from, from, from, to, to, to,
				constant.ValidRefundedPreOrderStatus, from, from, from, to, to, to,
			).Scan(&totalRefund).Error
		},
	)

	if err != nil {
		return nil, err
	}

	result.TotalSoldRevenue = stats.TotalRevenue
	result.TotalStandardRevenue = stats.TotalStandardRevenue
	result.TotalLimitedRevenue = stats.TotalLimitedRevenue
	result.StandardNetRevenue = stats.StandardNetRevenue
	result.LimitedGrossRevenue = stats.LimitedGrossRevenue
	result.LimitedNetRevenue = stats.LimitedNetRevenue
	result.TotalRefund = totalRefund
	result.ReturningCustomerCount = returningCount
	result.NewCustomerCount = newCount

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

	return &result, nil
}

func (r *SalesStaffAnalyticsRepository) GetTotalSoldRevenue(ctx context.Context, from, to time.Time, orderStatuses []enum.OrderStatus, preOrderStatuses []enum.PreOrderStatus) (float64, error) {
	totalSoldRevenueQuery := `
		WITH valid_orders AS (
			SELECT user_id, total_amount, order_type, 'ORDER' as source
			FROM orders
			WHERE status IN ? 
				AND (?::timestamp IS NULL OR ? = TIMESTAMP '0001-01-01 00:00:00' OR created_at >= ?)
				AND (?::timestamp IS NULL OR ? = TIMESTAMP '0001-01-01 00:00:00' OR created_at <= ?)
				AND deleted_at IS NULL
		),
		valid_pre_orders AS (
			SELECT user_id, total_amount, 'LIMITED' as order_type, 'PRE_ORDER' as source
			FROM pre_orders
			WHERE status IN ? 
				AND (?::timestamp IS NULL OR ? = TIMESTAMP '0001-01-01 00:00:00' OR created_at >= ?)
				AND (?::timestamp IS NULL OR ? = TIMESTAMP '0001-01-01 00:00:00' OR created_at <= ?)
				AND deleted_at IS NULL
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
		orderStatuses, from, from, from, to, to, to,
		preOrderStatuses, from, from, from, to, to, to,
	).Scan(&totalRevenue).Error; err != nil {
		return 0, err
	}

	return totalRevenue, nil
}

func (r *SalesStaffAnalyticsRepository) GetRevenueBreakdown(ctx context.Context, from, to time.Time, completedOrderStatuses []enum.OrderStatus, completedPreOrderStatuses []enum.PreOrderStatus) ([]responses.RevenueByProductType, []responses.RevenueByCategory, error) {
	var byProduct []responses.RevenueByProductType
	var byCategory []responses.RevenueByCategory

	err := utils.RunParallel(ctx, 2,
		func(ctx context.Context) error {
			// 1. By Product Type (Standard vs Limited)
			// Standard = Orders with type STANDARD
			// Limited = Orders with type LIMITED + PreOrders
			queryProduct := `
		WITH standard_rev AS (
			SELECT COALESCE(SUM(total_amount), 0) as rev
			FROM orders
			WHERE order_type = ? AND status IN ? 
				AND (?::timestamp IS NULL OR ? = TIMESTAMP '0001-01-01 00:00:00' OR created_at >= ?)
				AND (?::timestamp IS NULL OR ? = TIMESTAMP '0001-01-01 00:00:00' OR created_at <= ?)
				AND deleted_at IS NULL
		),
		limited_order_rev AS (
			SELECT COALESCE(SUM(total_amount), 0) as rev
			FROM orders
			WHERE order_type = ? AND status IN ? 
				AND (?::timestamp IS NULL OR ? = TIMESTAMP '0001-01-01 00:00:00' OR created_at >= ?)
				AND (?::timestamp IS NULL OR ? = TIMESTAMP '0001-01-01 00:00:00' OR created_at <= ?)
				AND deleted_at IS NULL
		),
		pre_order_rev AS (
			SELECT COALESCE(SUM(total_amount), 0) as rev
			FROM pre_orders
			WHERE status IN ? 
				AND (?::timestamp IS NULL OR ? = TIMESTAMP '0001-01-01 00:00:00' OR created_at >= ?)
				AND (?::timestamp IS NULL OR ? = TIMESTAMP '0001-01-01 00:00:00' OR created_at <= ?)
				AND deleted_at IS NULL
		)
		SELECT 'STANDARD' as product_type, rev as revenue FROM standard_rev
		UNION ALL
		SELECT 'LIMITED' as product_type, (SELECT rev FROM limited_order_rev) + (SELECT rev FROM pre_order_rev) as revenue
	`
			err := r.db.WithContext(ctx).Raw(queryProduct,
				enum.ProductTypeStandard, completedOrderStatuses, from, from, from, to, to, to,
				enum.ProductTypeLimited, completedOrderStatuses, from, from, from, to, to, to,
				completedPreOrderStatuses, from, from, from, to, to, to,
			).Scan(&byProduct).Error

			if err == nil {
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
			}

			return err
		},
		func(ctx context.Context) error {
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
			WHERE o.status IN ? 
				AND (?::timestamp IS NULL OR ? = TIMESTAMP '0001-01-01 00:00:00' OR o.created_at >= ?)
				AND (?::timestamp IS NULL OR ? = TIMESTAMP '0001-01-01 00:00:00' OR o.created_at <= ?)
				AND o.deleted_at IS NULL
			GROUP BY c.name
		),
		pre_order_revenue AS (
			SELECT c.name as category_name, SUM(po.total_amount) as revenue
			FROM pre_orders po
			JOIN product_variants pv ON po.variant_id = pv.id
			JOIN products p ON pv.product_id = p.id
			JOIN product_categories c ON p.category_id = c.id
			WHERE po.status IN ? 
				AND (?::timestamp IS NULL OR ? = TIMESTAMP '0001-01-01 00:00:00' OR po.created_at >= ?)
				AND (?::timestamp IS NULL OR ? = TIMESTAMP '0001-01-01 00:00:00' OR po.created_at <= ?)
				AND po.deleted_at IS NULL
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
			err := r.db.WithContext(ctx).Raw(queryCategory,
				completedOrderStatuses, from, from, from, to, to, to,
				completedPreOrderStatuses, from, from, from, to, to, to,
			).Scan(&byCategory).Error

			if err == nil {
				// Calculate percentages
				var totalRev float64
				for _, p := range byCategory {
					totalRev += p.Revenue
				}
				if totalRev > 0 {
					for i := range byCategory {
						byCategory[i].Percentage = (byCategory[i].Revenue / totalRev) * 100
					}
				}
			}
			return err
		},
	)

	if err != nil {
		return nil, nil, err
	}

	return byProduct, byCategory, nil
}

func (r *SalesStaffAnalyticsRepository) GetRevenueTrend(ctx context.Context, from, to time.Time, periodGap string, completedOrderStatuses []enum.OrderStatus, completedPreOrderStatuses []enum.PreOrderStatus) (map[string][]responses.SalesTimeSeriesPoint, error) {
	result := make(map[string][]responses.SalesTimeSeriesPoint)

	// Determine interval based on range
	var interval string
	isAll := false
	if periodGap == "all" {
		isAll = true
	} else if periodGap != "" {
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
	// Generate series backwards from 'to' date to ensure alignment with date range boundaries
	generateSeries := func(sourceType, orderType string, statuses any, table string) ([]responses.SalesTimeSeriesPoint, error) {
		var points []responses.SalesTimeSeriesPoint
		var q string
		var args []any

		if isAll {
			q = `
				SELECT 
					? as time, 
					COALESCE(SUM(t.total_amount), 0) as value, 
					? as type
				FROM ` + table + ` t
				WHERE t.status IN ? 
					AND (?::timestamp IS NULL OR ? = TIMESTAMP '0001-01-01 00:00:00' OR t.created_at >= ?)
					AND (?::timestamp IS NULL OR ? = TIMESTAMP '0001-01-01 00:00:00' OR t.created_at <= ?)
					AND t.deleted_at IS NULL
			`
			args = []any{from, sourceType, statuses, from, from, from, to, to, to}
		} else {
			// Generate series backwards from 'to' to 'from' to align with date range boundaries
			// This ensures the last point is at 'to' date and points go backwards by interval
			q = `
				WITH dates AS (
					SELECT generate_series(
						date_trunc('day', ?::timestamp),
						date_trunc('day', ?::timestamp),
						('-1 ' || ?)::interval
					) as date
				)
				SELECT 
					dates.date as time, 
					COALESCE(SUM(t.total_amount), 0) as value, 
					? as type
				FROM dates
				LEFT JOIN ` + table + ` t ON t.created_at >= dates.date
					AND t.created_at < dates.date + ('1 ' || ?)::interval
					AND t.status IN ? 
					AND (?::timestamp IS NULL OR ? = TIMESTAMP '0001-01-01 00:00:00' OR t.created_at >= ?)
					AND (?::timestamp IS NULL OR ? = TIMESTAMP '0001-01-01 00:00:00' OR t.created_at <= ?)
					AND t.deleted_at IS NULL
			`
			args = []any{to, from, interval, sourceType, interval, statuses, from, from, from, to, to, to}
		}

		if orderType != "" {
			q += " AND t.order_type = ?"
			args = append(args, orderType)
		}

		if !isAll {
			q += " GROUP BY dates.date ORDER BY dates.date"
		}

		if err := r.db.WithContext(ctx).Raw(q, args...).Scan(&points).Error; err != nil {
			return nil, err
		}
		return points, nil
	}

	var standardOrders, limitedOrders, preOrders []responses.SalesTimeSeriesPoint

	err := utils.RunParallel(ctx, 3,
		func(ctx context.Context) error {
			var err error
			standardOrders, err = generateSeries("STANDARD", "STANDARD", completedOrderStatuses, "orders")
			return err
		},
		func(ctx context.Context) error {
			var err error
			limitedOrders, err = generateSeries("LIMITED", "LIMITED", completedOrderStatuses, "orders")
			return err
		},
		func(ctx context.Context) error {
			var err error
			preOrders, err = generateSeries("PRE_ORDER", "", completedPreOrderStatuses, "pre_orders")
			return err
		},
	)
	if err != nil {
		return nil, err
	}

	// Helper to merge points
	mergePoints := func(p1, p2 []responses.SalesTimeSeriesPoint, typeName string) []responses.SalesTimeSeriesPoint {
		m := make(map[time.Time]float64)
		for _, p := range p1 {
			m[p.Time] += p.Value
		}
		for _, p := range p2 {
			m[p.Time] += p.Value
		}

		var merged []responses.SalesTimeSeriesPoint
		for t, v := range m {
			merged = append(merged, responses.SalesTimeSeriesPoint{Time: t, Value: v, Type: typeName})
		}
		slices.SortFunc(merged, func(a, b responses.SalesTimeSeriesPoint) int {
			return a.Time.Compare(b.Time)
		})
		return merged
	}

	// 1. STANDARD Series
	result["STANDARD"] = standardOrders

	// 2. LIMITED Series (Limited Orders + PreOrders)
	result["LIMITED"] = mergePoints(limitedOrders, preOrders, "LIMITED")

	// 3. ALL Series (Standard + Limited)
	result["ALL"] = mergePoints(result["STANDARD"], result["LIMITED"], "ALL")

	return result, nil
}

func (r *SalesStaffAnalyticsRepository) GetTopSellingByRevenue(ctx context.Context, from, to time.Time, completedOrderStatuses []enum.OrderStatus, completedPreOrderStatuses []enum.PreOrderStatus, limit int, sortBy string, sortOrder string) ([]responses.TopEntity, []responses.TopEntity, []responses.TopEntity, error) {
	var products, categories, brands []responses.TopEntity

	// Determine Order By
	orderBy := "value DESC"
	if sortBy == "name" {
		orderBy = "name"
	} else {
		orderBy = "value"
	}
	if sortOrder == "asc" {
		orderBy += " ASC"
	} else {
		orderBy += " DESC"
	}

	// Top Products
	queryProducts := `
		WITH order_sales AS (
			SELECT pv.product_id, p.name, SUM(oi.unit_price * oi.quantity) as revenue
			FROM orders o
			JOIN order_items oi ON o.id = oi.order_id
			JOIN product_variants pv ON oi.variant_id = pv.id
			JOIN products p ON pv.product_id = p.id
			WHERE o.status IN ?
				AND (?::timestamp IS NULL OR ? = TIMESTAMP '0001-01-01 00:00:00' OR o.created_at >= ?)
				AND (?::timestamp IS NULL OR ? = TIMESTAMP '0001-01-01 00:00:00' OR o.created_at <= ?)
				AND o.deleted_at IS NULL
			GROUP BY pv.product_id, p.name
		),
		pre_order_sales AS (
			SELECT pv.product_id, p.name, SUM(po.total_amount) as revenue
			FROM pre_orders po
			JOIN product_variants pv ON po.variant_id = pv.id
			JOIN products p ON pv.product_id = p.id
			WHERE po.status IN ?
				AND (?::timestamp IS NULL OR ? = TIMESTAMP '0001-01-01 00:00:00' OR po.created_at >= ?)
				AND (?::timestamp IS NULL OR ? = TIMESTAMP '0001-01-01 00:00:00' OR po.created_at <= ?)
				AND po.deleted_at IS NULL
			GROUP BY pv.product_id, p.name
		)
		SELECT product_id as id, name, SUM(revenue) as value
		FROM (
			SELECT * FROM order_sales
			UNION ALL
			SELECT * FROM pre_order_sales
		) as combined
		GROUP BY product_id, name
		ORDER BY ` + orderBy + `
		LIMIT ?
	`

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
		ORDER BY ` + orderBy + `
		LIMIT ?
	`

	// Top Brands
	queryBrands := `
		WITH order_sales AS (
			SELECT b.id, b.name, SUM(oi.unit_price * oi.quantity) as revenue
			FROM orders o
			JOIN order_items oi ON o.id = oi.order_id
			JOIN product_variants pv ON oi.variant_id = pv.id
			JOIN products p ON pv.product_id = p.id
			JOIN brands b ON p.brand_id = b.id
			WHERE o.status IN ?
				AND (?::timestamp IS NULL OR ? = TIMESTAMP '0001-01-01 00:00:00' OR o.created_at >= ?)
				AND (?::timestamp IS NULL OR ? = TIMESTAMP '0001-01-01 00:00:00' OR o.created_at <= ?)
				AND o.deleted_at IS NULL
			GROUP BY b.id, b.name
		),
		pre_order_sales AS (
			SELECT b.id, b.name, SUM(po.total_amount) as revenue
			FROM pre_orders po
			JOIN product_variants pv ON po.variant_id = pv.id
			JOIN products p ON pv.product_id = p.id
			JOIN brands b ON p.brand_id = b.id
			WHERE po.status IN ?
				AND (?::timestamp IS NULL OR ? = TIMESTAMP '0001-01-01 00:00:00' OR po.created_at >= ?)
				AND (?::timestamp IS NULL OR ? = TIMESTAMP '0001-01-01 00:00:00' OR po.created_at <= ?)
				AND po.deleted_at IS NULL
			GROUP BY b.id, b.name
		)
		SELECT id, name, SUM(revenue) as value
		FROM (
			SELECT * FROM order_sales
			UNION ALL
			SELECT * FROM pre_order_sales
		) as combined
		GROUP BY id, name
		ORDER BY ` + orderBy + `
		LIMIT ?
	`

	err := utils.RunParallel(ctx, 3,
		func(ctx context.Context) error {
			return r.db.WithContext(ctx).Raw(queryProducts,
				completedOrderStatuses, from, from, from, to, to, to,
				completedPreOrderStatuses, from, from, from, to, to, to,
				limit,
			).Scan(&products).Error
		},
		func(ctx context.Context) error {
			return r.db.WithContext(ctx).Raw(queryCategories,
				completedOrderStatuses, from, to,
				completedPreOrderStatuses, from, to,
				limit,
			).Scan(&categories).Error
		},
		func(ctx context.Context) error {
			return r.db.WithContext(ctx).Raw(queryBrands,
				completedOrderStatuses, from, from, from, to, to, to,
				completedPreOrderStatuses, from, from, from, to, to, to,
				limit,
			).Scan(&brands).Error
		},
	)

	if err != nil {
		return nil, nil, nil, err
	}

	return products, categories, brands, nil
}

// =============================================================================
// ORDERS TAB
// =============================================================================

func (r *SalesStaffAnalyticsRepository) GetOrdersSummary(ctx context.Context, from, to time.Time) (*responses.OrdersSummary, error) {
	var result responses.OrdersSummary

	// 1. Orders Stats
	var orderStats struct {
		TotalOrders     int64
		PendingOrders   int64
		CompletedOrders int64
		CancelledOrders int64
		RefundedOrders  int64
	}

	// 2. PreOrders Stats
	var preOrderStats struct {
		TotalOrders     int64
		PendingOrders   int64
		CompletedOrders int64
		CancelledOrders int64
		RefundedOrders  int64
	}

	err := utils.RunParallel(ctx, 2,
		func(ctx context.Context) error {
			queryOrders := `
				SELECT
					COUNT(*) as total_orders,
					COUNT(CASE WHEN status IN ? THEN 1 END) as completed_orders,
					COUNT(CASE WHEN status IN ? THEN 1 END) as pending_orders,
					COUNT(CASE WHEN status IN ? THEN 1 END) as cancelled_orders,
					COUNT(CASE WHEN status IN ? THEN 1 END) as refunded_orders
				FROM orders
				WHERE (?::timestamp IS NULL OR ? = TIMESTAMP '0001-01-01 00:00:00' OR created_at >= ?)
					AND (?::timestamp IS NULL OR ? = TIMESTAMP '0001-01-01 00:00:00' OR created_at <= ?)
					AND deleted_at IS NULL
			`
			err := r.db.WithContext(ctx).Raw(queryOrders,
				constant.ValidCompletedOrderStatus, constant.ValidPendingOrderStatus,
				constant.ValidCancelledOrderStatus, constant.ValidRefundedOrderStatus,
				from, from, from, to, to, to,
			).Scan(&orderStats).Error

			if err == nil {
				result.Order.Total = orderStats.TotalOrders
				result.Order.Pending = orderStats.PendingOrders
				result.Order.Completed = orderStats.CompletedOrders
				result.Order.Cancelled = orderStats.CancelledOrders
				result.Order.Refunded = orderStats.RefundedOrders
				if orderStats.TotalOrders > 0 {
					result.Order.CancellationRate = (float64(orderStats.CancelledOrders) / float64(orderStats.TotalOrders)) * 100
					result.Order.RefundRate = (float64(orderStats.RefundedOrders) / float64(orderStats.TotalOrders)) * 100
				}
			}
			return err
		},
		func(ctx context.Context) error {
			queryPreOrders := `
				SELECT
					COUNT(*) as total_orders,
					COUNT(CASE WHEN status IN ? THEN 1 END) as completed_orders,
					COUNT(CASE WHEN status IN ? THEN 1 END) as pending_orders,
					COUNT(CASE WHEN status IN ? THEN 1 END) as cancelled_orders,
					COUNT(CASE WHEN status IN ? THEN 1 END) as refunded_orders
				FROM pre_orders
				WHERE (?::timestamp IS NULL OR ? = TIMESTAMP '0001-01-01 00:00:00' OR created_at >= ?)
					AND (?::timestamp IS NULL OR ? = TIMESTAMP '0001-01-01 00:00:00' OR created_at <= ?)
					AND deleted_at IS NULL
			`
			err := r.db.WithContext(ctx).Raw(queryPreOrders,
				constant.ValidCompletedPreOrderStatus, constant.ValidPendingPreOrderStatus,
				constant.ValidCancelledPreOrderStatus, constant.ValidRefundedPreOrderStatus,
				from, from, from, to, to, to,
			).Scan(&preOrderStats).Error

			if err == nil {
				result.PreOrder.Total = preOrderStats.TotalOrders
				result.PreOrder.Pending = preOrderStats.PendingOrders
				result.PreOrder.Completed = preOrderStats.CompletedOrders
				result.PreOrder.Cancelled = preOrderStats.CancelledOrders
				result.PreOrder.Refunded = preOrderStats.RefundedOrders
				if preOrderStats.TotalOrders > 0 {
					result.PreOrder.CancellationRate = (float64(preOrderStats.CancelledOrders) / float64(preOrderStats.TotalOrders)) * 100
					result.PreOrder.RefundRate = (float64(preOrderStats.RefundedOrders) / float64(preOrderStats.TotalOrders)) * 100
				}
			}
			return err
		},
	)
	if err != nil {
		return nil, err
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

	err := utils.RunParallel(ctx, 2,
		func(ctx context.Context) error {
			return r.db.WithContext(ctx).Raw(`
		SELECT 
			SUM(CASE WHEN status IN ? THEN 1 ELSE 0 END) as pending,
			SUM(CASE WHEN status IN ? THEN 1 ELSE 0 END) as completed,
			SUM(CASE WHEN status IN ? THEN 1 ELSE 0 END) as cancelled,
			SUM(CASE WHEN status IN ? THEN 1 ELSE 0 END) as refunded
		FROM orders 
		WHERE deleted_at IS NULL
			AND (?::timestamp IS NULL OR ? = TIMESTAMP '0001-01-01 00:00:00' OR created_at >= ?)
			AND (?::timestamp IS NULL OR ? = TIMESTAMP '0001-01-01 00:00:00' OR created_at <= ?)
		`,
				constant.ValidPendingOrderStatus, constant.ValidCompletedOrderStatus, constant.ValidCancelledOrderStatus,
				constant.ValidRefundedOrderStatus, from, from, from, to, to, to,
			).Scan(&ordersDist).Error
		},
		func(ctx context.Context) error {
			return r.db.WithContext(ctx).Raw(`
		SELECT 
			SUM(CASE WHEN status IN ? THEN 1 ELSE 0 END) as pending,
			SUM(CASE WHEN status IN ? THEN 1 ELSE 0 END) as completed,
			SUM(CASE WHEN status IN ? THEN 1 ELSE 0 END) as cancelled,
			SUM(CASE WHEN status IN ? THEN 1 ELSE 0 END) as refunded
		FROM pre_orders 
		WHERE deleted_at IS NULL
			AND (?::timestamp IS NULL OR ? = TIMESTAMP '0001-01-01 00:00:00' OR created_at >= ?)
			AND (?::timestamp IS NULL OR ? = TIMESTAMP '0001-01-01 00:00:00' OR created_at <= ?)
		`,
				constant.ValidPendingPreOrderStatus, constant.ValidCompletedPreOrderStatus, constant.ValidCancelledPreOrderStatus,
				constant.ValidRefundedPreOrderStatus, from, from, from, to, to, to,
			).Scan(&preOrdersDist).Error
		},
	)

	if err != nil {
		return ordersDist, preOrdersDist, err
	}

	return ordersDist, preOrdersDist, nil
}

func (r *SalesStaffAnalyticsRepository) GetOrdersTrend(ctx context.Context, from, to time.Time, periodGap string) (orders, preOrders, standard, limited []responses.SalesTimeSeriesPoint, err error) {
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
				AND (?::timestamp IS NULL OR ? = TIMESTAMP '0001-01-01 00:00:00' OR t.created_at >= ?)
				AND (?::timestamp IS NULL OR ? = TIMESTAMP '0001-01-01 00:00:00' OR t.created_at <= ?)
				AND t.deleted_at IS NULL
		`
		args := []any{interval, from, interval, to, "1 " + interval, sourceType, interval, from, from, from, to, to, to}

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

	var limitedOrders []responses.SalesTimeSeriesPoint

	err = utils.RunParallel(ctx, 4,
		func(ctx context.Context) error {
			var e error
			orders, e = generateSeries("ORDER", "", "orders")
			return e
		},
		func(ctx context.Context) error {
			var e error
			preOrders, e = generateSeries("PRE_ORDER", "", "pre_orders")
			return e
		},
		func(ctx context.Context) error {
			var e error
			standard, e = generateSeries("STANDARD", "STANDARD", "orders")
			return e
		},
		func(ctx context.Context) error {
			var e error
			limitedOrders, e = generateSeries("LIMITED", "LIMITED", "orders")
			return e
		},
	)
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

func (r *SalesStaffAnalyticsRepository) GetTopSellingByVolume(ctx context.Context, from, to time.Time, completedOrderStatuses []enum.OrderStatus, completedPreOrderStatuses []enum.PreOrderStatus, limit int, sortBy string, sortOrder string) ([]responses.TopEntity, []responses.TopEntity, []responses.TopEntity, error) {
	var products, categories, brands []responses.TopEntity

	// Determine Order By
	orderBy := "value DESC"
	if sortBy == "name" {
		orderBy = "name"
	} else {
		orderBy = "value"
	}
	if sortOrder == "asc" {
		orderBy += " ASC"
	} else {
		orderBy += " DESC"
	}

	// Top Products
	queryProducts := `
		WITH order_sales AS (
			SELECT pv.product_id, p.name, SUM(oi.quantity) as volume
			FROM orders o
			JOIN order_items oi ON o.id = oi.order_id
			JOIN product_variants pv ON oi.variant_id = pv.id
			JOIN products p ON pv.product_id = p.id
			WHERE o.status IN ?
				AND (?::timestamp IS NULL OR ? = TIMESTAMP '0001-01-01 00:00:00' OR o.created_at >= ?)
				AND (?::timestamp IS NULL OR ? = TIMESTAMP '0001-01-01 00:00:00' OR o.created_at <= ?)
				AND o.deleted_at IS NULL
			GROUP BY pv.product_id, p.name
		),
		pre_order_sales AS (
			SELECT pv.product_id, p.name, SUM(po.quantity) as volume
			FROM pre_orders po
			JOIN product_variants pv ON po.variant_id = pv.id
			JOIN products p ON pv.product_id = p.id
			WHERE po.status IN ?
				AND (?::timestamp IS NULL OR ? = TIMESTAMP '0001-01-01 00:00:00' OR po.created_at >= ?)
				AND (?::timestamp IS NULL OR ? = TIMESTAMP '0001-01-01 00:00:00' OR po.created_at <= ?)
				AND po.deleted_at IS NULL
			GROUP BY pv.product_id, p.name
		)
		SELECT product_id as id, name, SUM(volume) as value
		FROM (
			SELECT * FROM order_sales
			UNION ALL
			SELECT * FROM pre_order_sales
		) as combined
		GROUP BY product_id, name
		ORDER BY ` + orderBy + `
		LIMIT ?
	`

	// Top Categories
	queryCategories := `
		WITH order_sales AS (
			SELECT c.id, c.name, SUM(oi.quantity) as volume
			FROM orders o
			JOIN order_items oi ON o.id = oi.order_id
			JOIN product_variants pv ON oi.variant_id = pv.id
			JOIN products p ON pv.product_id = p.id
			JOIN product_categories c ON p.category_id = c.id
			WHERE o.status in ?
				AND (?::timestamp IS NULL OR ? = TIMESTAMP '0001-01-01 00:00:00' OR o.created_at >= ?)
				AND (?::timestamp IS NULL OR ? = TIMESTAMP '0001-01-01 00:00:00' OR o.created_at <= ?)
				AND o.deleted_at IS NULL
			GROUP BY c.id, c.name
		),
		pre_order_sales AS (
			SELECT c.id, c.name, SUM(po.quantity) as volume
			FROM pre_orders po
			JOIN product_variants pv ON po.variant_id = pv.id
			JOIN products p ON pv.product_id = p.id
			JOIN product_categories c ON p.category_id = c.id
			WHERE po.status IN ?
				AND (?::timestamp IS NULL OR ? = TIMESTAMP '0001-01-01 00:00:00' OR po.created_at >= ?)
				AND (?::timestamp IS NULL OR ? = TIMESTAMP '0001-01-01 00:00:00' OR po.created_at <= ?)
				AND po.deleted_at IS NULL
			GROUP BY c.id, c.name
		)
		SELECT id, name, SUM(volume) as value
		FROM (
			SELECT * FROM order_sales
			UNION ALL
			SELECT * FROM pre_order_sales
		) as combined
		GROUP BY id, name
		ORDER BY ` + orderBy + `
		LIMIT ?
	`

	// Top Brands
	queryBrands := `
		WITH order_sales AS (
			SELECT b.id, b.name, SUM(oi.quantity) as volume
			FROM orders o
			JOIN order_items oi ON o.id = oi.order_id
			JOIN product_variants pv ON oi.variant_id = pv.id
			JOIN products p ON pv.product_id = p.id
			JOIN brands b ON p.brand_id = b.id
			WHERE o.status IN ?
				AND (?::timestamp IS NULL OR ? = TIMESTAMP '0001-01-01 00:00:00' OR o.created_at >= ?)
				AND (?::timestamp IS NULL OR ? = TIMESTAMP '0001-01-01 00:00:00' OR o.created_at <= ?)
				AND o.deleted_at IS NULL
			GROUP BY b.id, b.name
		),
		pre_order_sales AS (
			SELECT b.id, b.name, SUM(po.quantity) as volume
			FROM pre_orders po
			JOIN product_variants pv ON po.variant_id = pv.id
			JOIN products p ON pv.product_id = p.id
			JOIN brands b ON p.brand_id = b.id
			WHERE po.status IN ?
				AND (?::timestamp IS NULL OR ? = TIMESTAMP '0001-01-01 00:00:00' OR po.created_at >= ?)
				AND (?::timestamp IS NULL OR ? = TIMESTAMP '0001-01-01 00:00:00' OR po.created_at <= ?)
				AND po.deleted_at IS NULL
			GROUP BY b.id, b.name
		)
		SELECT id, name, SUM(volume) as value
		FROM (
			SELECT * FROM order_sales
			UNION ALL
			SELECT * FROM pre_order_sales
		) as combined
		GROUP BY id, name
		ORDER BY ` + orderBy + `
		LIMIT ?
	`

	err := utils.RunParallel(ctx, 3,
		func(ctx context.Context) error {
			return r.db.WithContext(ctx).Raw(queryProducts,
				completedOrderStatuses, from, from, from, to, to, to,
				completedPreOrderStatuses, from, from, from, to, to, to,
				limit,
			).Scan(&products).Error
		},
		func(ctx context.Context) error {
			return r.db.WithContext(ctx).Raw(queryCategories,
				completedOrderStatuses, from, from, from, to, to, to,
				completedPreOrderStatuses, from, from, from, to, to, to,
				limit,
			).Scan(&categories).Error
		},
		func(ctx context.Context) error {
			return r.db.WithContext(ctx).Raw(queryBrands,
				completedOrderStatuses, from, from, from, to, to, to,
				completedPreOrderStatuses, from, from, from, to, to, to,
				limit,
			).Scan(&brands).Error
		},
	)

	if err != nil {
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
		WHERE status in ?
			AND (?::timestamp IS NULL OR ? = TIMESTAMP '0001-01-01 00:00:00' OR created_at >= ?)
			AND (?::timestamp IS NULL OR ? = TIMESTAMP '0001-01-01 00:00:00' OR created_at <= ?)
			AND deleted_at IS NULL
		UNION ALL
		SELECT id, full_name as customer_name, total_amount as price, status::varchar(50), 'PRE_ORDER' as type, created_at
		FROM pre_orders
		WHERE status in ?
			AND (?::timestamp IS NULL OR ? = TIMESTAMP '0001-01-01 00:00:00' OR created_at >= ?)
			AND (?::timestamp IS NULL OR ? = TIMESTAMP '0001-01-01 00:00:00' OR created_at <= ?)
			AND deleted_at IS NULL
		ORDER BY created_at DESC
		LIMIT ?
	`
	if err := r.db.WithContext(ctx).Raw(query,
		constant.ValidPendingOrderStatus, from, from, from, to, to, to,
		constant.ValidPendingPreOrderStatus, from, from, from, to, to, to,
		limit,
	).Scan(&orders).Error; err != nil {
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

// =============================================================================
// REVENUE DETAIL QUERIES IMPLEMENTATIONS
// =============================================================================

// GetTotalRevenueOrders returns all orders (both standard and limited) and preorders contributing to total revenue
// Includes payment transaction information
func (r *SalesStaffAnalyticsRepository) GetTotalRevenueOrders(ctx context.Context, from, to time.Time, completedOrderStatuses []enum.OrderStatus, completedPreOrderStatuses []enum.PreOrderStatus, page, limit int, search, sortBy, sortOrder string) ([]responses.RevenueOrderItemWithPayment, int64, float64, error) {
	offset := (page - 1) * limit

	// Determine sort column and order
	orderClause := "created_at DESC"
	if sortBy == "total_amount" {
		orderClause = "total_amount"
	} else if sortBy == "created_at" {
		orderClause = "created_at"
	}
	if sortOrder == "asc" {
		orderClause += " ASC"
	} else {
		orderClause += " DESC"
	}

	// Build search condition
	searchCondition := ""
	searchArgs := []interface{}{}
	if search != "" {
		searchCondition = " AND (full_name ILIKE ? OR id::text ILIKE ?)"
		searchPattern := "%" + search + "%"
		searchArgs = append(searchArgs, searchPattern, searchPattern)
	}

	query := `
		WITH orders_data AS (
			SELECT 
				o.id,
				'ORDER' as source,
				o.order_type,
				o.user_id as customer_id,
				o.full_name as customer_name,
				o.email as customer_email,
				o.status::varchar(50) as status,
				o.total_amount,
				COALESCE(o.shipping_fee, 0)::float as shipping_fee,
				o.total_amount as net_amount,
				NULL::float as kol_percent,
				o.created_at
			FROM orders o
			WHERE o.status IN ?
				AND (?::timestamp IS NULL OR ? = TIMESTAMP '0001-01-01 00:00:00' OR o.created_at >= ?)
				AND (?::timestamp IS NULL OR ? = TIMESTAMP '0001-01-01 00:00:00' OR o.created_at <= ?)
				AND o.deleted_at IS NULL
		),
		pre_orders_data AS (
			SELECT 
				po.id,
				'PRE_ORDER' as source,
				'LIMITED' as order_type,
				po.user_id as customer_id,
				po.full_name as customer_name,
				po.email as customer_email,
				po.status::varchar(50) as status,
				po.total_amount,
				COALESCE(po.shipping_fee, 0)::float as shipping_fee,
				po.total_amount as net_amount,
				NULL::float as kol_percent,
				po.created_at
			FROM pre_orders po
			WHERE po.status IN ?
				AND (?::timestamp IS NULL OR ? = TIMESTAMP '0001-01-01 00:00:00' OR po.created_at >= ?)
				AND (?::timestamp IS NULL OR ? = TIMESTAMP '0001-01-01 00:00:00' OR po.created_at <= ?)
				AND po.deleted_at IS NULL
		),
		combined AS (
			SELECT * FROM orders_data
			UNION ALL
			SELECT * FROM pre_orders_data
		)
		SELECT * FROM combined
		WHERE 1=1 ` + searchCondition + `
		ORDER BY ` + orderClause + `
		LIMIT ? OFFSET ?
	`

	args := []interface{}{
		completedOrderStatuses, from, from, from, to, to, to,
		completedPreOrderStatuses, from, from, from, to, to, to,
	}
	args = append(args, searchArgs...)
	args = append(args, limit, offset)

	// Intermediate struct for scanning
	type rawItem struct {
		ID            uuid.UUID `gorm:"column:id"`
		Source        string    `gorm:"column:source"`
		OrderType     string    `gorm:"column:order_type"`
		CustomerID    uuid.UUID `gorm:"column:customer_id"`
		CustomerName  string    `gorm:"column:customer_name"`
		CustomerEmail string    `gorm:"column:customer_email"`
		Status        string    `gorm:"column:status"`
		TotalAmount   float64   `gorm:"column:total_amount"`
		ShippingFee   float64   `gorm:"column:shipping_fee"`
		NetAmount     float64   `gorm:"column:net_amount"`
		KOLPercent    *float64  `gorm:"column:kol_percent"`
		CreatedAt     time.Time `gorm:"column:created_at"`
	}

	var rawItems []rawItem
	if err := r.db.WithContext(ctx).Raw(query, args...).Scan(&rawItems).Error; err != nil {
		return nil, 0, 0, err
	}

	// Get payment transactions for each item
	var items []responses.RevenueOrderItemWithPayment
	for _, raw := range rawItems {
		item := responses.RevenueOrderItemWithPayment{
			ID:            raw.ID,
			Source:        raw.Source,
			OrderType:     raw.OrderType,
			CustomerID:    raw.CustomerID,
			CustomerName:  raw.CustomerName,
			CustomerEmail: raw.CustomerEmail,
			Status:        raw.Status,
			TotalAmount:   raw.TotalAmount,
			ShippingFee:   raw.ShippingFee,
			NetAmount:     raw.NetAmount,
			KOLPercent:    raw.KOLPercent,
			CreatedAt:     raw.CreatedAt,
		}

		// Fetch payment transaction
		var paymentTx model.PaymentTransaction
		referenceType := enum.PaymentTransactionReferenceTypeOrder
		if raw.Source == "PRE_ORDER" {
			referenceType = enum.PaymentTransactionReferenceTypePreOrder
		}
		if err := r.db.WithContext(ctx).
			Where("reference_id = ? AND reference_type = ? AND method = 'PAYOS' ", raw.ID, referenceType).
			Order("transaction_date DESC").
			First(&paymentTx).Error; err == nil {
			amountStr := ""
			if paymentTx.Amount != nil {
				amountStr = fmt.Sprintf("%.2f", *paymentTx.Amount)
			}
			item.PaymentTransaction = &responses.PaymentTransactionResponse{
				ID:              paymentTx.ID,
				ReferenceID:     paymentTx.ReferenceID.String(),
				ReferenceType:   string(paymentTx.ReferenceType),
				Amount:          amountStr,
				Method:          paymentTx.Method,
				Status:          string(paymentTx.Status),
				TransactionDate: paymentTx.TransactionDate.Format(time.RFC3339),
				GatewayRef:      paymentTx.GatewayRef,
				GatewayID:       paymentTx.GatewayID,
				UpdatedAt:       paymentTx.UpdatedAt.Format(time.RFC3339),
				PayerID:         paymentTx.PayerID,
				ReceivedByID:    paymentTx.ReceivedByID,
			}
		}

		items = append(items, item)
	}

	// Get total count and sum
	countQuery := `
		WITH orders_data AS (
			SELECT o.id, o.total_amount, o.full_name
			FROM orders o
			WHERE o.status IN ?
				AND (?::timestamp IS NULL OR ? = TIMESTAMP '0001-01-01 00:00:00' OR o.created_at >= ?)
				AND (?::timestamp IS NULL OR ? = TIMESTAMP '0001-01-01 00:00:00' OR o.created_at <= ?)
				AND o.deleted_at IS NULL
		),
		pre_orders_data AS (
			SELECT po.id, po.total_amount, po.full_name
			FROM pre_orders po
			WHERE po.status IN ?
				AND (?::timestamp IS NULL OR ? = TIMESTAMP '0001-01-01 00:00:00' OR po.created_at >= ?)
				AND (?::timestamp IS NULL OR ? = TIMESTAMP '0001-01-01 00:00:00' OR po.created_at <= ?)
				AND po.deleted_at IS NULL
		),
		combined AS (
			SELECT * FROM orders_data
			UNION ALL
			SELECT * FROM pre_orders_data
		)
		SELECT COUNT(*) as total, COALESCE(SUM(total_amount), 0) as total_revenue
		FROM combined
		WHERE 1=1 ` + searchCondition

	countArgs := []interface{}{
		completedOrderStatuses, from, from, from, to, to, to,
		completedPreOrderStatuses, from, from, from, to, to, to,
	}
	countArgs = append(countArgs, searchArgs...)

	var result struct {
		Total        int64
		TotalRevenue float64
	}
	if err := r.db.WithContext(ctx).Raw(countQuery, countArgs...).Scan(&result).Error; err != nil {
		return nil, 0, 0, err
	}

	return items, result.Total, result.TotalRevenue, nil
}

// GetStandardRevenueOrders returns only STANDARD type orders
// Includes payment transaction information
func (r *SalesStaffAnalyticsRepository) GetStandardRevenueOrders(ctx context.Context, from, to time.Time, completedOrderStatuses []enum.OrderStatus, page, limit int, search, sortBy, sortOrder string) ([]responses.RevenueOrderItemWithPayment, int64, float64, error) {
	offset := (page - 1) * limit

	// Determine sort column and order
	orderClause := "created_at DESC"
	if sortBy == "total_amount" {
		orderClause = "total_amount"
	} else if sortBy == "created_at" {
		orderClause = "created_at"
	}
	if sortOrder == "asc" {
		orderClause += " ASC"
	} else {
		orderClause += " DESC"
	}

	// Build search condition
	searchCondition := ""
	searchArgs := []interface{}{}
	if search != "" {
		searchCondition = " AND (o.full_name ILIKE ? OR o.id::text ILIKE ?)"
		searchPattern := "%" + search + "%"
		searchArgs = append(searchArgs, searchPattern, searchPattern)
	}

	query := `
		SELECT 
			o.id,
			'ORDER' as source,
			o.order_type,
			o.user_id as customer_id,
			o.full_name as customer_name,
			o.email as customer_email,
			o.status::varchar(50) as status,
			o.total_amount,
			COALESCE(o.shipping_fee, 0)::float as shipping_fee,
			o.total_amount as net_amount,
			NULL::float as kol_percent,
			o.created_at
		FROM orders o
		WHERE o.status IN ?
			AND o.order_type = 'STANDARD'
			AND (?::timestamp IS NULL OR ? = TIMESTAMP '0001-01-01 00:00:00' OR o.created_at >= ?)
			AND (?::timestamp IS NULL OR ? = TIMESTAMP '0001-01-01 00:00:00' OR o.created_at <= ?)
			AND o.deleted_at IS NULL
			` + searchCondition + `
		ORDER BY ` + orderClause + `
		LIMIT ? OFFSET ?
	`

	args := []interface{}{
		completedOrderStatuses, from, from, from, to, to, to,
	}
	args = append(args, searchArgs...)
	args = append(args, limit, offset)

	// Intermediate struct for scanning
	type rawItem struct {
		ID            uuid.UUID `gorm:"column:id"`
		Source        string    `gorm:"column:source"`
		OrderType     string    `gorm:"column:order_type"`
		CustomerID    uuid.UUID `gorm:"column:customer_id"`
		CustomerName  string    `gorm:"column:customer_name"`
		CustomerEmail string    `gorm:"column:customer_email"`
		Status        string    `gorm:"column:status"`
		TotalAmount   float64   `gorm:"column:total_amount"`
		ShippingFee   float64   `gorm:"column:shipping_fee"`
		NetAmount     float64   `gorm:"column:net_amount"`
		KOLPercent    *float64  `gorm:"column:kol_percent"`
		CreatedAt     time.Time `gorm:"column:created_at"`
	}

	var rawItems []rawItem
	if err := r.db.WithContext(ctx).Raw(query, args...).Scan(&rawItems).Error; err != nil {
		return nil, 0, 0, err
	}

	// Get payment transactions for each item
	var items []responses.RevenueOrderItemWithPayment
	for _, raw := range rawItems {
		item := responses.RevenueOrderItemWithPayment{
			ID:            raw.ID,
			Source:        raw.Source,
			OrderType:     raw.OrderType,
			CustomerID:    raw.CustomerID,
			CustomerName:  raw.CustomerName,
			CustomerEmail: raw.CustomerEmail,
			Status:        raw.Status,
			TotalAmount:   raw.TotalAmount,
			ShippingFee:   raw.ShippingFee,
			NetAmount:     raw.NetAmount,
			KOLPercent:    raw.KOLPercent,
			CreatedAt:     raw.CreatedAt,
		}

		// Fetch payment transaction (STANDARD orders are always ORDER type)
		var paymentTx model.PaymentTransaction
		if err := r.db.WithContext(ctx).
			Where("reference_id = ? AND reference_type = ? AND method = 'PAYOS' ", raw.ID, enum.PaymentTransactionReferenceTypeOrder).
			Order("transaction_date DESC").
			First(&paymentTx).Error; err == nil {
			amountStr := ""
			if paymentTx.Amount != nil {
				amountStr = fmt.Sprintf("%.2f", *paymentTx.Amount)
			}
			item.PaymentTransaction = &responses.PaymentTransactionResponse{
				ID:              paymentTx.ID,
				ReferenceID:     paymentTx.ReferenceID.String(),
				ReferenceType:   string(paymentTx.ReferenceType),
				Amount:          amountStr,
				Method:          paymentTx.Method,
				Status:          string(paymentTx.Status),
				TransactionDate: paymentTx.TransactionDate.Format(time.RFC3339),
				GatewayRef:      paymentTx.GatewayRef,
				GatewayID:       paymentTx.GatewayID,
				UpdatedAt:       paymentTx.UpdatedAt.Format(time.RFC3339),
				PayerID:         paymentTx.PayerID,
				ReceivedByID:    paymentTx.ReceivedByID,
			}
		}

		items = append(items, item)
	}

	// Get total count and sum
	countQuery := `
		SELECT COUNT(*) as total, COALESCE(SUM(o.total_amount), 0) as total_revenue
		FROM orders o
		WHERE o.status IN ?
			AND o.order_type = 'STANDARD'
			AND (?::timestamp IS NULL OR ? = TIMESTAMP '0001-01-01 00:00:00' OR o.created_at >= ?)
			AND (?::timestamp IS NULL OR ? = TIMESTAMP '0001-01-01 00:00:00' OR o.created_at <= ?)
			AND o.deleted_at IS NULL
			` + searchCondition

	countArgs := []interface{}{
		completedOrderStatuses, from, from, from, to, to, to,
	}
	countArgs = append(countArgs, searchArgs...)

	var result struct {
		Total        int64
		TotalRevenue float64
	}
	if err := r.db.WithContext(ctx).Raw(countQuery, countArgs...).Scan(&result).Error; err != nil {
		return nil, 0, 0, err
	}

	return items, result.Total, result.TotalRevenue, nil
}

// GetLimitedRevenueOrders returns LIMITED type orders and all PreOrders
// Includes payment transaction information
func (r *SalesStaffAnalyticsRepository) GetLimitedRevenueOrders(ctx context.Context, from, to time.Time, completedOrderStatuses []enum.OrderStatus, completedPreOrderStatuses []enum.PreOrderStatus, page, limit int, search, sortBy, sortOrder string) ([]responses.RevenueOrderItemWithPayment, int64, float64, error) {
	offset := (page - 1) * limit

	// Determine sort column and order
	orderClause := "created_at DESC"
	if sortBy == "total_amount" {
		orderClause = "total_amount"
	} else if sortBy == "created_at" {
		orderClause = "created_at"
	}
	if sortOrder == "asc" {
		orderClause += " ASC"
	} else {
		orderClause += " DESC"
	}

	// Build search condition
	searchCondition := ""
	searchArgs := []interface{}{}
	if search != "" {
		searchCondition = " AND (full_name ILIKE ? OR id::text ILIKE ?)"
		searchPattern := "%" + search + "%"
		searchArgs = append(searchArgs, searchPattern, searchPattern)
	}

	query := `
		WITH limited_orders AS (
			SELECT 
				o.id,
				'ORDER' as source,
				o.order_type,
				o.user_id as customer_id,
				o.full_name as customer_name,
				o.email as customer_email,
				o.status::varchar(50) as status,
				o.total_amount,
				COALESCE(o.shipping_fee, 0)::float as shipping_fee,
				o.total_amount as net_amount,
				NULL::float as kol_percent,
				o.created_at
			FROM orders o
			WHERE o.status IN ?
				AND o.order_type = 'LIMITED'
				AND (?::timestamp IS NULL OR ? = TIMESTAMP '0001-01-01 00:00:00' OR o.created_at >= ?)
				AND (?::timestamp IS NULL OR ? = TIMESTAMP '0001-01-01 00:00:00' OR o.created_at <= ?)
				AND o.deleted_at IS NULL
		),
		pre_orders AS (
			SELECT 
				po.id,
				'PRE_ORDER' as source,
				'LIMITED' as order_type,
				po.user_id as customer_id,
				po.full_name as customer_name,
				po.email as customer_email,
				po.status::varchar(50) as status,
				po.total_amount,
				COALESCE(po.shipping_fee, 0)::float as shipping_fee,
				po.total_amount as net_amount,
				NULL::float as kol_percent,
				po.created_at
			FROM pre_orders po
			WHERE po.status IN ?
				AND (?::timestamp IS NULL OR ? = TIMESTAMP '0001-01-01 00:00:00' OR po.created_at >= ?)
				AND (?::timestamp IS NULL OR ? = TIMESTAMP '0001-01-01 00:00:00' OR po.created_at <= ?)
				AND po.deleted_at IS NULL
		),
		combined AS (
			SELECT * FROM limited_orders
			UNION ALL
			SELECT * FROM pre_orders
		)
		SELECT * FROM combined
		WHERE 1=1 ` + searchCondition + `
		ORDER BY ` + orderClause + `
		LIMIT ? OFFSET ?
	`

	args := []interface{}{
		completedOrderStatuses, from, from, from, to, to, to,
		completedPreOrderStatuses, from, from, from, to, to, to,
	}
	args = append(args, searchArgs...)
	args = append(args, limit, offset)

	// Intermediate struct for scanning
	type rawItem struct {
		ID            uuid.UUID `gorm:"column:id"`
		Source        string    `gorm:"column:source"`
		OrderType     string    `gorm:"column:order_type"`
		CustomerID    uuid.UUID `gorm:"column:customer_id"`
		CustomerName  string    `gorm:"column:customer_name"`
		CustomerEmail string    `gorm:"column:customer_email"`
		Status        string    `gorm:"column:status"`
		TotalAmount   float64   `gorm:"column:total_amount"`
		ShippingFee   float64   `gorm:"column:shipping_fee"`
		NetAmount     float64   `gorm:"column:net_amount"`
		KOLPercent    *float64  `gorm:"column:kol_percent"`
		CreatedAt     time.Time `gorm:"column:created_at"`
	}

	var rawItems []rawItem
	if err := r.db.WithContext(ctx).Raw(query, args...).Scan(&rawItems).Error; err != nil {
		return nil, 0, 0, err
	}

	// Get payment transactions for each item
	var items []responses.RevenueOrderItemWithPayment
	for _, raw := range rawItems {
		item := responses.RevenueOrderItemWithPayment{
			ID:            raw.ID,
			Source:        raw.Source,
			OrderType:     raw.OrderType,
			CustomerID:    raw.CustomerID,
			CustomerName:  raw.CustomerName,
			CustomerEmail: raw.CustomerEmail,
			Status:        raw.Status,
			TotalAmount:   raw.TotalAmount,
			ShippingFee:   raw.ShippingFee,
			NetAmount:     raw.NetAmount,
			KOLPercent:    raw.KOLPercent,
			CreatedAt:     raw.CreatedAt,
		}

		// Fetch payment transaction
		var paymentTx model.PaymentTransaction
		referenceType := enum.PaymentTransactionReferenceTypeOrder
		if raw.Source == "PRE_ORDER" {
			referenceType = enum.PaymentTransactionReferenceTypePreOrder
		}
		if err := r.db.WithContext(ctx).
			Where("reference_id = ? AND reference_type = ? AND method = 'PAYOS' ", raw.ID, referenceType).
			Order("transaction_date DESC").
			First(&paymentTx).Error; err == nil {
			amountStr := ""
			if paymentTx.Amount != nil {
				amountStr = fmt.Sprintf("%.2f", *paymentTx.Amount)
			}
			item.PaymentTransaction = &responses.PaymentTransactionResponse{
				ID:              paymentTx.ID,
				ReferenceID:     paymentTx.ReferenceID.String(),
				ReferenceType:   string(paymentTx.ReferenceType),
				Amount:          amountStr,
				Method:          paymentTx.Method,
				Status:          string(paymentTx.Status),
				TransactionDate: paymentTx.TransactionDate.Format(time.RFC3339),
				GatewayRef:      paymentTx.GatewayRef,
				GatewayID:       paymentTx.GatewayID,
				UpdatedAt:       paymentTx.UpdatedAt.Format(time.RFC3339),
				PayerID:         paymentTx.PayerID,
				ReceivedByID:    paymentTx.ReceivedByID,
			}
		}

		items = append(items, item)
	}

	// Get total count and sum
	countQuery := `
		WITH limited_orders AS (
			SELECT o.id, o.total_amount, o.full_name
			FROM orders o
			WHERE o.status IN ?
				AND o.order_type = 'LIMITED'
				AND (?::timestamp IS NULL OR ? = TIMESTAMP '0001-01-01 00:00:00' OR o.created_at >= ?)
				AND (?::timestamp IS NULL OR ? = TIMESTAMP '0001-01-01 00:00:00' OR o.created_at <= ?)
				AND o.deleted_at IS NULL
		),
		pre_orders AS (
			SELECT po.id, po.total_amount, po.full_name
			FROM pre_orders po
			WHERE po.status IN ?
				AND (?::timestamp IS NULL OR ? = TIMESTAMP '0001-01-01 00:00:00' OR po.created_at >= ?)
				AND (?::timestamp IS NULL OR ? = TIMESTAMP '0001-01-01 00:00:00' OR po.created_at <= ?)
				AND po.deleted_at IS NULL
		),
		combined AS (
			SELECT * FROM limited_orders
			UNION ALL
			SELECT * FROM pre_orders
		)
		SELECT COUNT(*) as total, COALESCE(SUM(total_amount), 0) as total_revenue
		FROM combined
		WHERE 1=1 ` + searchCondition

	countArgs := []interface{}{
		completedOrderStatuses, from, from, from, to, to, to,
		completedPreOrderStatuses, from, from, from, to, to, to,
	}
	countArgs = append(countArgs, searchArgs...)

	var result struct {
		Total        int64
		TotalRevenue float64
	}
	if err := r.db.WithContext(ctx).Raw(countQuery, countArgs...).Scan(&result).Error; err != nil {
		return nil, 0, 0, err
	}

	return items, result.Total, result.TotalRevenue, nil
}

// GetStandardNetRevenueOrders returns STANDARD orders with net revenue (total_amount - shipping_fee)
func (r *SalesStaffAnalyticsRepository) GetStandardNetRevenueOrders(ctx context.Context, from, to time.Time, completedOrderStatuses []enum.OrderStatus, page, limit int, search, sortBy, sortOrder string) ([]responses.RevenueOrderItem, int64, float64, error) {
	var items []responses.RevenueOrderItem

	offset := (page - 1) * limit

	// Determine sort column and order
	orderClause := "created_at DESC"
	if sortBy == "total_amount" {
		orderClause = "total_amount"
	} else if sortBy == "net_amount" {
		orderClause = "net_amount"
	} else if sortBy == "created_at" {
		orderClause = "created_at"
	}
	if sortOrder == "asc" {
		orderClause += " ASC"
	} else {
		orderClause += " DESC"
	}

	// Build search condition
	searchCondition := ""
	searchArgs := []interface{}{}
	if search != "" {
		searchCondition = " AND (o.full_name ILIKE ? OR o.id::text ILIKE ?)"
		searchPattern := "%" + search + "%"
		searchArgs = append(searchArgs, searchPattern, searchPattern)
	}

	query := `
		SELECT 
			o.id,
			'ORDER' as source,
			o.order_type,
			o.user_id as customer_id,
			o.full_name as customer_name,
			o.email as customer_email,
			o.status::varchar(50) as status,
			o.total_amount,
			COALESCE(o.shipping_fee, 0)::float as shipping_fee,
			(o.total_amount - COALESCE(o.shipping_fee, 0))::float as net_amount,
			NULL::float as kol_percent,
			o.created_at
		FROM orders o
		WHERE o.status IN ?
			AND o.order_type = 'STANDARD'
			AND (?::timestamp IS NULL OR ? = TIMESTAMP '0001-01-01 00:00:00' OR o.created_at >= ?)
			AND (?::timestamp IS NULL OR ? = TIMESTAMP '0001-01-01 00:00:00' OR o.created_at <= ?)
			AND o.deleted_at IS NULL
			` + searchCondition + `
		ORDER BY ` + orderClause + `
		LIMIT ? OFFSET ?
	`

	args := []interface{}{
		completedOrderStatuses, from, from, from, to, to, to,
	}
	args = append(args, searchArgs...)
	args = append(args, limit, offset)

	if err := r.db.WithContext(ctx).Raw(query, args...).Scan(&items).Error; err != nil {
		return nil, 0, 0, err
	}

	// Get total count and sum (net revenue = total_amount - shipping_fee)
	countQuery := `
		SELECT 
			COUNT(*) as total, 
			COALESCE(SUM(o.total_amount - COALESCE(o.shipping_fee, 0)), 0) as total_revenue
		FROM orders o
		WHERE o.status IN ?
			AND o.order_type = 'STANDARD'
			AND (?::timestamp IS NULL OR ? = TIMESTAMP '0001-01-01 00:00:00' OR o.created_at >= ?)
			AND (?::timestamp IS NULL OR ? = TIMESTAMP '0001-01-01 00:00:00' OR o.created_at <= ?)
			AND o.deleted_at IS NULL
			` + searchCondition

	countArgs := []interface{}{
		completedOrderStatuses, from, from, from, to, to, to,
	}
	countArgs = append(countArgs, searchArgs...)

	var result struct {
		Total        int64
		TotalRevenue float64
	}
	if err := r.db.WithContext(ctx).Raw(countQuery, countArgs...).Scan(&result).Error; err != nil {
		return nil, 0, 0, err
	}

	return items, result.Total, result.TotalRevenue, nil
}

// GetLimitedNetRevenueOrders returns LIMITED orders and PreOrders with KOL net revenue calculation
// KOL Net Revenue = (item_total) * kol_percent / 100 - shipping_fee
// Includes payment transaction information
func (r *SalesStaffAnalyticsRepository) GetLimitedNetRevenueOrders(ctx context.Context, from, to time.Time, completedOrderStatuses []enum.OrderStatus, completedPreOrderStatuses []enum.PreOrderStatus, page, limit int, search, sortBy, sortOrder string) ([]responses.RevenueOrderItemWithPayment, int64, float64, error) {
	offset := (page - 1) * limit

	// Determine sort column and order
	orderClause := "created_at DESC"
	if sortBy == "total_amount" {
		orderClause = "total_amount"
	} else if sortBy == "net_amount" {
		orderClause = "net_amount"
	} else if sortBy == "created_at" {
		orderClause = "created_at"
	}
	if sortOrder == "asc" {
		orderClause += " ASC"
	} else {
		orderClause += " DESC"
	}

	// Build search condition
	searchCondition := ""
	searchArgs := []interface{}{}
	if search != "" {
		searchCondition = " AND (full_name ILIKE ? OR id::text ILIKE ?)"
		searchPattern := "%" + search + "%"
		searchArgs = append(searchArgs, searchPattern, searchPattern)
	}

	// Query for LIMITED orders with KOL net revenue calculation
	query := `
		WITH limited_orders AS (
			SELECT 
				o.id,
				'ORDER' as source,
				o.order_type,
				o.user_id as customer_id,
				o.full_name as customer_name,
				o.email as customer_email,
				o.status::varchar(50) as status,
				o.total_amount,
				COALESCE(o.shipping_fee, 0)::float as shipping_fee,
				COALESCE((c.financial_terms->>'profit_split_kol_percent')::float, 0) as kol_percent,
				((SUM(oi.unit_price * oi.quantity) * COALESCE((c.financial_terms->>'profit_split_kol_percent')::float, 0) / 100.0) - COALESCE(o.shipping_fee, 0))::float as net_amount,
				o.created_at
			FROM orders o
			JOIN order_items oi ON o.id = oi.order_id
			JOIN product_variants pv ON oi.variant_id = pv.id
			JOIN products p ON pv.product_id = p.id
			LEFT JOIN tasks t ON p.task_id = t.id
			LEFT JOIN milestones m ON t.milestone_id = m.id
			LEFT JOIN campaigns camp ON m.campaign_id = camp.id
			LEFT JOIN contracts c ON camp.contract_id = c.id
			WHERE o.status IN ?
				AND o.order_type = 'LIMITED'
				AND (?::timestamp IS NULL OR ? = TIMESTAMP '0001-01-01 00:00:00' OR o.created_at >= ?)
				AND (?::timestamp IS NULL OR ? = TIMESTAMP '0001-01-01 00:00:00' OR o.created_at <= ?)
				AND o.deleted_at IS NULL
			GROUP BY o.id, o.order_type, o.user_id, o.full_name, o.email, o.status, o.total_amount, o.shipping_fee, c.financial_terms, o.created_at
		),
		pre_orders AS (
			SELECT 
				po.id,
				'PRE_ORDER' as source,
				'LIMITED' as order_type,
				po.user_id as customer_id,
				po.full_name as customer_name,
				po.email as customer_email,
				po.status::varchar(50) as status,
				po.total_amount,
				COALESCE(po.shipping_fee, 0)::float as shipping_fee,
				COALESCE((c.financial_terms->>'profit_split_kol_percent')::float, 0) as kol_percent,
				((po.total_amount * COALESCE((c.financial_terms->>'profit_split_kol_percent')::float, 0) / 100.0) - COALESCE(po.shipping_fee, 0))::float as net_amount,
				po.created_at
			FROM pre_orders po
			JOIN product_variants pv ON po.variant_id = pv.id
			JOIN products p ON pv.product_id = p.id
			LEFT JOIN tasks t ON p.task_id = t.id
			LEFT JOIN milestones m ON t.milestone_id = m.id
			LEFT JOIN campaigns camp ON m.campaign_id = camp.id
			LEFT JOIN contracts c ON camp.contract_id = c.id
			WHERE po.status IN ?
				AND (?::timestamp IS NULL OR ? = TIMESTAMP '0001-01-01 00:00:00' OR po.created_at >= ?)
				AND (?::timestamp IS NULL OR ? = TIMESTAMP '0001-01-01 00:00:00' OR po.created_at <= ?)
				AND po.deleted_at IS NULL
		),
		combined AS (
			SELECT * FROM limited_orders
			UNION ALL
			SELECT * FROM pre_orders
		)
		SELECT 
			id,
			source,
			order_type,
			customer_id,
			customer_name,
			customer_email,
			status,
			total_amount,
			shipping_fee,
			net_amount,
			kol_percent,
			created_at
		FROM combined
		WHERE 1=1 ` + searchCondition + `
		ORDER BY ` + orderClause + `
		LIMIT ? OFFSET ?
	`

	args := []interface{}{
		completedOrderStatuses, from, from, from, to, to, to,
		completedPreOrderStatuses, from, from, from, to, to, to,
	}
	args = append(args, searchArgs...)
	args = append(args, limit, offset)

	// Intermediate struct for scanning
	type rawItem struct {
		ID            uuid.UUID `gorm:"column:id"`
		Source        string    `gorm:"column:source"`
		OrderType     string    `gorm:"column:order_type"`
		CustomerID    uuid.UUID `gorm:"column:customer_id"`
		CustomerName  string    `gorm:"column:customer_name"`
		CustomerEmail string    `gorm:"column:customer_email"`
		Status        string    `gorm:"column:status"`
		TotalAmount   float64   `gorm:"column:total_amount"`
		ShippingFee   float64   `gorm:"column:shipping_fee"`
		NetAmount     float64   `gorm:"column:net_amount"`
		KOLPercent    float64   `gorm:"column:kol_percent"`
		CreatedAt     time.Time `gorm:"column:created_at"`
	}

	var rawItems []rawItem
	if err := r.db.WithContext(ctx).Raw(query, args...).Scan(&rawItems).Error; err != nil {
		return nil, 0, 0, err
	}

	// Get payment transactions for each item
	var items []responses.RevenueOrderItemWithPayment
	for _, raw := range rawItems {
		item := responses.RevenueOrderItemWithPayment{
			ID:            raw.ID,
			Source:        raw.Source,
			OrderType:     raw.OrderType,
			CustomerID:    raw.CustomerID,
			CustomerName:  raw.CustomerName,
			CustomerEmail: raw.CustomerEmail,
			Status:        raw.Status,
			TotalAmount:   raw.TotalAmount,
			ShippingFee:   raw.ShippingFee,
			NetAmount:     raw.NetAmount,
			KOLPercent:    &raw.KOLPercent,
			CreatedAt:     raw.CreatedAt,
		}

		// Fetch payment transaction
		var paymentTx model.PaymentTransaction
		referenceType := enum.PaymentTransactionReferenceTypeOrder
		if raw.Source == "PRE_ORDER" {
			referenceType = enum.PaymentTransactionReferenceTypePreOrder
		}
		if err := r.db.WithContext(ctx).
			Where("reference_id = ? AND reference_type = ? AND method = 'PAYOS' ", raw.ID, referenceType).
			Order("transaction_date DESC").
			First(&paymentTx).Error; err == nil {
			amountStr := ""
			if paymentTx.Amount != nil {
				amountStr = fmt.Sprintf("%.2f", *paymentTx.Amount)
			}
			item.PaymentTransaction = &responses.PaymentTransactionResponse{
				ID:              paymentTx.ID,
				ReferenceID:     paymentTx.ReferenceID.String(),
				ReferenceType:   string(paymentTx.ReferenceType),
				Amount:          amountStr,
				Method:          paymentTx.Method,
				Status:          string(paymentTx.Status),
				TransactionDate: paymentTx.TransactionDate.Format(time.RFC3339),
				GatewayRef:      paymentTx.GatewayRef,
				GatewayID:       paymentTx.GatewayID,
				UpdatedAt:       paymentTx.UpdatedAt.Format(time.RFC3339),
				PayerID:         paymentTx.PayerID,
				ReceivedByID:    paymentTx.ReceivedByID,
			}
		}

		items = append(items, item)
	}

	// Get total count and total net revenue
	countQuery := `
		WITH limited_orders AS (
			SELECT 
				o.id,
				o.full_name,
				((SUM(oi.unit_price * oi.quantity) * COALESCE((c.financial_terms->>'profit_split_kol_percent')::float, 0) / 100.0) - COALESCE(o.shipping_fee, 0))::float as net_amount
			FROM orders o
			JOIN order_items oi ON o.id = oi.order_id
			JOIN product_variants pv ON oi.variant_id = pv.id
			JOIN products p ON pv.product_id = p.id
			LEFT JOIN tasks t ON p.task_id = t.id
			LEFT JOIN milestones m ON t.milestone_id = m.id
			LEFT JOIN campaigns camp ON m.campaign_id = camp.id
			LEFT JOIN contracts c ON camp.contract_id = c.id
			WHERE o.status IN ?
				AND o.order_type = 'LIMITED'
				AND (?::timestamp IS NULL OR ? = TIMESTAMP '0001-01-01 00:00:00' OR o.created_at >= ?)
				AND (?::timestamp IS NULL OR ? = TIMESTAMP '0001-01-01 00:00:00' OR o.created_at <= ?)
				AND o.deleted_at IS NULL
			GROUP BY o.id, o.full_name, o.shipping_fee, c.financial_terms
		),
		pre_orders AS (
			SELECT 
				po.id,
				po.full_name,
				((po.total_amount * COALESCE((c.financial_terms->>'profit_split_kol_percent')::float, 0) / 100.0) - COALESCE(po.shipping_fee, 0))::float as net_amount
			FROM pre_orders po
			JOIN product_variants pv ON po.variant_id = pv.id
			JOIN products p ON pv.product_id = p.id
			LEFT JOIN tasks t ON p.task_id = t.id
			LEFT JOIN milestones m ON t.milestone_id = m.id
			LEFT JOIN campaigns camp ON m.campaign_id = camp.id
			LEFT JOIN contracts c ON camp.contract_id = c.id
			WHERE po.status IN ?
				AND (?::timestamp IS NULL OR ? = TIMESTAMP '0001-01-01 00:00:00' OR po.created_at >= ?)
				AND (?::timestamp IS NULL OR ? = TIMESTAMP '0001-01-01 00:00:00' OR po.created_at <= ?)
				AND po.deleted_at IS NULL
		),
		combined AS (
			SELECT id, full_name, net_amount FROM limited_orders
			UNION ALL
			SELECT id, full_name, net_amount FROM pre_orders
		)
		SELECT COUNT(*) as total, COALESCE(SUM(net_amount), 0) as total_revenue
		FROM combined
		WHERE 1=1 ` + searchCondition

	countArgs := []interface{}{
		completedOrderStatuses, from, from, from, to, to, to,
		completedPreOrderStatuses, from, from, from, to, to, to,
	}
	countArgs = append(countArgs, searchArgs...)

	var result struct {
		Total        int64
		TotalRevenue float64
	}
	if err := r.db.WithContext(ctx).Raw(countQuery, countArgs...).Scan(&result).Error; err != nil {
		return nil, 0, 0, err
	}

	return items, result.Total, result.TotalRevenue, nil
}

// GetRefundedOrders returns all refunded orders and preorders with payment transaction information
func (r *SalesStaffAnalyticsRepository) GetRefundedOrders(ctx context.Context, from, to time.Time, refundedOrderStatuses []enum.OrderStatus, refundedPreOrderStatuses []enum.PreOrderStatus, page, limit int, search, sortBy, sortOrder string) ([]responses.RevenueOrderItemWithPayment, int64, float64, error) {
	offset := (page - 1) * limit

	// Determine sort column and order
	orderClause := "created_at DESC"
	if sortBy == "total_amount" {
		orderClause = "total_amount"
	} else if sortBy == "created_at" {
		orderClause = "created_at"
	}
	if sortOrder == "asc" {
		orderClause += " ASC"
	} else {
		orderClause += " DESC"
	}

	// Build search condition
	searchCondition := ""
	searchArgs := []interface{}{}
	if search != "" {
		searchCondition = " AND (full_name ILIKE ? OR id::text ILIKE ?)"
		searchPattern := "%" + search + "%"
		searchArgs = append(searchArgs, searchPattern, searchPattern)
	}

	query := `
		WITH refunded_orders AS (
			SELECT 
				o.id,
				'ORDER' as source,
				o.order_type,
				o.user_id as customer_id,
				o.full_name as customer_name,
				o.email as customer_email,
				o.status::varchar(50) as status,
				o.total_amount,
				COALESCE(o.shipping_fee, 0)::float as shipping_fee,
				o.total_amount as net_amount,
				o.created_at
			FROM orders o
			WHERE o.status IN ?
				AND (?::timestamp IS NULL OR ? = TIMESTAMP '0001-01-01 00:00:00' OR o.created_at >= ?)
				AND (?::timestamp IS NULL OR ? = TIMESTAMP '0001-01-01 00:00:00' OR o.created_at <= ?)
				AND o.deleted_at IS NULL
		),
		refunded_pre_orders AS (
			SELECT 
				po.id,
				'PRE_ORDER' as source,
				'LIMITED' as order_type,
				po.user_id as customer_id,
				po.full_name as customer_name,
				po.email as customer_email,
				po.status::varchar(50) as status,
				po.total_amount,
				COALESCE(po.shipping_fee, 0)::float as shipping_fee,
				po.total_amount as net_amount,
				po.created_at
			FROM pre_orders po
			WHERE po.status IN ?
				AND (?::timestamp IS NULL OR ? = TIMESTAMP '0001-01-01 00:00:00' OR po.created_at >= ?)
				AND (?::timestamp IS NULL OR ? = TIMESTAMP '0001-01-01 00:00:00' OR po.created_at <= ?)
				AND po.deleted_at IS NULL
		),
		combined AS (
			SELECT * FROM refunded_orders
			UNION ALL
			SELECT * FROM refunded_pre_orders
		)
		SELECT * FROM combined
		WHERE 1=1 ` + searchCondition + `
		ORDER BY ` + orderClause + `
		LIMIT ? OFFSET ?
	`

	args := []interface{}{
		refundedOrderStatuses, from, from, from, to, to, to,
		refundedPreOrderStatuses, from, from, from, to, to, to,
	}
	args = append(args, searchArgs...)
	args = append(args, limit, offset)

	// Intermediate struct for scanning
	type rawItem struct {
		ID            uuid.UUID `gorm:"column:id"`
		Source        string    `gorm:"column:source"`
		OrderType     string    `gorm:"column:order_type"`
		CustomerID    uuid.UUID `gorm:"column:customer_id"`
		CustomerName  string    `gorm:"column:customer_name"`
		CustomerEmail string    `gorm:"column:customer_email"`
		Status        string    `gorm:"column:status"`
		TotalAmount   float64   `gorm:"column:total_amount"`
		ShippingFee   float64   `gorm:"column:shipping_fee"`
		NetAmount     float64   `gorm:"column:net_amount"`
		CreatedAt     time.Time `gorm:"column:created_at"`
	}

	var rawItems []rawItem
	if err := r.db.WithContext(ctx).Raw(query, args...).Scan(&rawItems).Error; err != nil {
		return nil, 0, 0, err
	}

	// Get payment transactions for each item
	var items []responses.RevenueOrderItemWithPayment
	for _, raw := range rawItems {
		item := responses.RevenueOrderItemWithPayment{
			ID:            raw.ID,
			Source:        raw.Source,
			OrderType:     raw.OrderType,
			CustomerID:    raw.CustomerID,
			CustomerName:  raw.CustomerName,
			CustomerEmail: raw.CustomerEmail,
			Status:        raw.Status,
			TotalAmount:   raw.TotalAmount,
			ShippingFee:   raw.ShippingFee,
			NetAmount:     raw.NetAmount,
			CreatedAt:     raw.CreatedAt,
		}

		// Fetch payment transaction (look for refund transaction or latest transaction)
		var paymentTx model.PaymentTransaction
		referenceType := enum.PaymentTransactionReferenceTypeOrder
		if raw.Source == "PRE_ORDER" {
			referenceType = enum.PaymentTransactionReferenceTypePreOrder
		}
		if err := r.db.WithContext(ctx).
			Where("reference_id = ? AND reference_type = ? AND method = 'PAYOS'", raw.ID, referenceType).
			Order("transaction_date DESC").
			First(&paymentTx).Error; err == nil {
			amountStr := ""
			if paymentTx.Amount != nil {
				amountStr = fmt.Sprintf("%.2f", *paymentTx.Amount)
			}
			item.PaymentTransaction = &responses.PaymentTransactionResponse{
				ID:              paymentTx.ID,
				ReferenceID:     paymentTx.ReferenceID.String(),
				ReferenceType:   string(paymentTx.ReferenceType),
				Amount:          amountStr,
				Method:          paymentTx.Method,
				Status:          string(paymentTx.Status),
				TransactionDate: paymentTx.TransactionDate.Format(time.RFC3339),
				GatewayRef:      paymentTx.GatewayRef,
				GatewayID:       paymentTx.GatewayID,
				UpdatedAt:       paymentTx.UpdatedAt.Format(time.RFC3339),
				PayerID:         paymentTx.PayerID,
				ReceivedByID:    paymentTx.ReceivedByID,
			}
		}

		items = append(items, item)
	}

	// Get total count and sum
	countQuery := `
		WITH refunded_orders AS (
			SELECT o.id, o.total_amount, o.full_name
			FROM orders o
			WHERE o.status IN ?
				AND (?::timestamp IS NULL OR ? = TIMESTAMP '0001-01-01 00:00:00' OR o.created_at >= ?)
				AND (?::timestamp IS NULL OR ? = TIMESTAMP '0001-01-01 00:00:00' OR o.created_at <= ?)
				AND o.deleted_at IS NULL
		),
		refunded_pre_orders AS (
			SELECT po.id, po.total_amount, po.full_name
			FROM pre_orders po
			WHERE po.status IN ?
				AND (?::timestamp IS NULL OR ? = TIMESTAMP '0001-01-01 00:00:00' OR po.created_at >= ?)
				AND (?::timestamp IS NULL OR ? = TIMESTAMP '0001-01-01 00:00:00' OR po.created_at <= ?)
				AND po.deleted_at IS NULL
		),
		combined AS (
			SELECT * FROM refunded_orders
			UNION ALL
			SELECT * FROM refunded_pre_orders
		)
		SELECT COUNT(*) as total, COALESCE(SUM(total_amount), 0) as total_revenue
		FROM combined
		WHERE 1=1 ` + searchCondition

	countArgs := []interface{}{
		refundedOrderStatuses, from, from, from, to, to, to,
		refundedPreOrderStatuses, from, from, from, to, to, to,
	}
	countArgs = append(countArgs, searchArgs...)

	var result struct {
		Total        int64
		TotalRevenue float64
	}
	if err := r.db.WithContext(ctx).Raw(countQuery, countArgs...).Scan(&result).Error; err != nil {
		return nil, 0, 0, err
	}

	return items, result.Total, result.TotalRevenue, nil
}
