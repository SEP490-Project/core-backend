package gormrepository

import (
	"context"
	"core-backend/internal/application/dto/dtos"
	"core-backend/internal/application/interfaces/irepository"
	"time"

	"gorm.io/gorm"
)

type adminAnalyticsRepository struct {
	db *gorm.DB
}

// NewAdminAnalyticsRepository creates a new admin analytics repository
func NewAdminAnalyticsRepository(db *gorm.DB) irepository.AdminAnalyticsRepository {
	return &adminAnalyticsRepository{db: db}
}

// =============================================================================
// USERS
// =============================================================================

// GetTotalUsersCount returns the total number of users
func (r *adminAnalyticsRepository) GetTotalUsersCount(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Table("users").
		Where("deleted_at IS NULL").
		Count(&count).Error
	return count, err
}

// GetActiveUsersCount returns the count of users active within the specified days
func (r *adminAnalyticsRepository) GetActiveUsersCount(ctx context.Context, activeDays int) (int64, error) {
	var count int64
	cutoff := time.Now().AddDate(0, 0, -activeDays)
	err := r.db.WithContext(ctx).
		Table("users").
		Where("deleted_at IS NULL").
		Where("last_login_at >= ?", cutoff).
		Count(&count).Error
	return count, err
}

// GetUserCountByRole returns the count of users by role
func (r *adminAnalyticsRepository) GetUserCountByRole(ctx context.Context, role string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Table("users").
		Where("deleted_at IS NULL").
		Where("role = ?", role).
		Count(&count).Error
	return count, err
}

// GetNewUsersCount returns the count of users registered within the date range
func (r *adminAnalyticsRepository) GetNewUsersCount(ctx context.Context, startDate, endDate time.Time) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Table("users").
		Where("deleted_at IS NULL").
		Where("created_at >= ? AND created_at < ?", startDate, endDate).
		Count(&count).Error
	return count, err
}

// GetUserGrowthTrend returns user growth trend over time
func (r *adminAnalyticsRepository) GetUserGrowthTrend(ctx context.Context, granularity string, startDate, endDate *time.Time) ([]dtos.UserGrowthResult, error) {
	var results []dtos.UserGrowthResult

	dateFunc := getDateTruncFunc(granularity)
	query := r.db.WithContext(ctx).
		Table("users").
		Select(dateFunc + " AS date, COUNT(*) AS new_users").
		Where("deleted_at IS NULL")

	if startDate != nil {
		query = query.Where("created_at >= ?", *startDate)
	}
	if endDate != nil {
		query = query.Where("created_at < ?", *endDate)
	}

	query = query.Group(dateFunc).Order("date")

	err := query.Scan(&results).Error
	return results, err
}

// =============================================================================
// BRANDS
// =============================================================================

// GetTotalBrandsCount returns the total number of brands
func (r *adminAnalyticsRepository) GetTotalBrandsCount(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Table("brands").
		Where("deleted_at IS NULL").
		Count(&count).Error
	return count, err
}

// GetActiveBrandsCount returns the count of brands with active contracts
func (r *adminAnalyticsRepository) GetActiveBrandsCount(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Table("brands b").
		Joins("INNER JOIN contracts c ON c.brand_id = b.id AND c.deleted_at IS NULL AND c.status = 'ACTIVE'").
		Where("b.deleted_at IS NULL").
		Distinct("b.id").
		Count(&count).Error
	return count, err
}

// =============================================================================
// CONTRACTS
// =============================================================================

// GetTotalContractsCount returns the total number of contracts
func (r *adminAnalyticsRepository) GetTotalContractsCount(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Table("contracts").
		Where("deleted_at IS NULL").
		Count(&count).Error
	return count, err
}

// GetContractCountByStatus returns the count of contracts by status
func (r *adminAnalyticsRepository) GetContractCountByStatus(ctx context.Context, status string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Table("contracts").
		Where("deleted_at IS NULL").
		Where("status = ?", status).
		Count(&count).Error
	return count, err
}

// GetTotalContractValue returns the total value of all contracts
func (r *adminAnalyticsRepository) GetTotalContractValue(ctx context.Context) (float64, error) {
	var total float64
	err := r.db.WithContext(ctx).
		Table("contracts").
		Where("deleted_at IS NULL").
		Select("COALESCE(SUM(total_value), 0)").
		Scan(&total).Error
	return total, err
}

// GetCollectedContractAmount returns the total paid amount from contract payments
func (r *adminAnalyticsRepository) GetCollectedContractAmount(ctx context.Context) (float64, error) {
	var total float64
	err := r.db.WithContext(ctx).
		Table("contract_payments cp").
		Joins("INNER JOIN contracts c ON c.id = cp.contract_id AND c.deleted_at IS NULL").
		Where("cp.deleted_at IS NULL").
		Where("cp.status = ?", "PAID").
		Select("COALESCE(SUM(cp.amount), 0)").
		Scan(&total).Error
	return total, err
}

// GetPendingContractAmount returns the total pending amount from contract payments
func (r *adminAnalyticsRepository) GetPendingContractAmount(ctx context.Context) (float64, error) {
	var total float64
	err := r.db.WithContext(ctx).
		Table("contract_payments cp").
		Joins("INNER JOIN contracts c ON c.id = cp.contract_id AND c.deleted_at IS NULL").
		Where("cp.deleted_at IS NULL").
		Where("cp.status IN ?", []string{"PENDING", "OVERDUE"}).
		Select("COALESCE(SUM(cp.amount), 0)").
		Scan(&total).Error
	return total, err
}

// =============================================================================
// CAMPAIGNS
// =============================================================================

// GetTotalCampaignsCount returns the total number of campaigns
func (r *adminAnalyticsRepository) GetTotalCampaignsCount(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Table("campaigns").
		Where("deleted_at IS NULL").
		Count(&count).Error
	return count, err
}

// GetCampaignCountByStatus returns the count of campaigns by status
func (r *adminAnalyticsRepository) GetCampaignCountByStatus(ctx context.Context, status string) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Table("campaigns").
		Where("deleted_at IS NULL").
		Where("status = ?", status).
		Count(&count).Error
	return count, err
}

// GetTotalContentCount returns the total number of content items
func (r *adminAnalyticsRepository) GetTotalContentCount(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Table("contents").
		Where("deleted_at IS NULL").
		Count(&count).Error
	return count, err
}

// GetPostedContentCount returns the count of posted content
func (r *adminAnalyticsRepository) GetPostedContentCount(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Table("contents").
		Where("deleted_at IS NULL").
		Where("status = ?", "POSTED").
		Count(&count).Error
	return count, err
}

// =============================================================================
// REVENUE
// =============================================================================

// GetTotalPlatformRevenue returns the total platform revenue
func (r *adminAnalyticsRepository) GetTotalPlatformRevenue(ctx context.Context, startDate, endDate *time.Time) (float64, error) {
	var total float64

	// Revenue from paid orders
	orderQuery := r.db.WithContext(ctx).
		Table("orders").
		Where("deleted_at IS NULL").
		Where("status IN ?", []string{"PAID", "PROCESSING", "SHIPPED", "DELIVERED"})

	if startDate != nil {
		orderQuery = orderQuery.Where("created_at >= ?", *startDate)
	}
	if endDate != nil {
		orderQuery = orderQuery.Where("created_at < ?", *endDate)
	}

	var orderRevenue float64
	if err := orderQuery.Select("COALESCE(SUM(total_amount), 0)").Scan(&orderRevenue).Error; err != nil {
		return 0, err
	}

	// Revenue from contract payments
	paymentQuery := r.db.WithContext(ctx).
		Table("contract_payments cp").
		Joins("INNER JOIN contracts c ON c.id = cp.contract_id AND c.deleted_at IS NULL").
		Where("cp.deleted_at IS NULL").
		Where("cp.status = ?", "PAID")

	if startDate != nil {
		paymentQuery = paymentQuery.Where("cp.paid_at >= ?", *startDate)
	}
	if endDate != nil {
		paymentQuery = paymentQuery.Where("cp.paid_at < ?", *endDate)
	}

	var paymentRevenue float64
	if err := paymentQuery.Select("COALESCE(SUM(cp.amount), 0)").Scan(&paymentRevenue).Error; err != nil {
		return 0, err
	}

	total = orderRevenue + paymentRevenue
	return total, nil
}

// GetPlatformRevenueByContractType returns revenue by contract type
func (r *adminAnalyticsRepository) GetPlatformRevenueByContractType(ctx context.Context, contractType string, startDate, endDate *time.Time) (float64, error) {
	var total float64

	query := r.db.WithContext(ctx).
		Table("contract_payments cp").
		Joins("INNER JOIN contracts c ON c.id = cp.contract_id AND c.deleted_at IS NULL").
		Where("cp.deleted_at IS NULL").
		Where("cp.status = ?", "PAID").
		Where("c.type = ?", contractType)

	if startDate != nil {
		query = query.Where("cp.paid_at >= ?", *startDate)
	}
	if endDate != nil {
		query = query.Where("cp.paid_at < ?", *endDate)
	}

	err := query.Select("COALESCE(SUM(cp.amount), 0)").Scan(&total).Error
	return total, err
}

// GetPlatformProductRevenue returns revenue from product sales by type
func (r *adminAnalyticsRepository) GetPlatformProductRevenue(ctx context.Context, productType string, startDate, endDate *time.Time) (float64, error) {
	var total float64

	query := r.db.WithContext(ctx).
		Table("orders o").
		Joins("INNER JOIN order_items oi ON oi.order_id = o.id").
		Joins("INNER JOIN products p ON p.id = oi.product_id AND p.deleted_at IS NULL").
		Where("o.deleted_at IS NULL").
		Where("o.status IN ?", []string{"PAID", "PROCESSING", "SHIPPED", "DELIVERED"}).
		Where("p.type = ?", productType)

	if startDate != nil {
		query = query.Where("o.created_at >= ?", *startDate)
	}
	if endDate != nil {
		query = query.Where("o.created_at < ?", *endDate)
	}

	err := query.Select("COALESCE(SUM(oi.subtotal), 0)").Scan(&total).Error
	return total, err
}

// GetPlatformRevenueTrend returns platform revenue trend over time (combines order + contract payment revenue)
func (r *adminAnalyticsRepository) GetPlatformRevenueTrend(ctx context.Context, granularity string, startDate, endDate *time.Time) ([]dtos.RevenueTrendResult, error) {
	var results []dtos.RevenueTrendResult

	// Determine date_trunc interval based on granularity
	var truncInterval string
	switch granularity {
	case "DAY":
		truncInterval = "day"
	case "WEEK":
		truncInterval = "week"
	default:
		truncInterval = "month"
	}

	// Build date filter conditions
	var dateConditions string
	var params []any

	if startDate != nil && endDate != nil {
		dateConditions = "AND created_at >= ? AND created_at < ?"
		params = append(params, *startDate, *endDate)
	} else if startDate != nil {
		dateConditions = "AND created_at >= ?"
		params = append(params, *startDate)
	} else if endDate != nil {
		dateConditions = "AND created_at < ?"
		params = append(params, *endDate)
	}

	// Build payment date filter (uses paid_at instead of created_at)
	var paymentDateConditions string
	if startDate != nil && endDate != nil {
		paymentDateConditions = "AND cp.paid_at >= ? AND cp.paid_at < ?"
		params = append(params, *startDate, *endDate)
	} else if startDate != nil {
		paymentDateConditions = "AND cp.paid_at >= ?"
		params = append(params, *startDate)
	} else if endDate != nil {
		paymentDateConditions = "AND cp.paid_at < ?"
		params = append(params, *endDate)
	}

	// CTE query combining order revenue and contract payment revenue
	query := `
		WITH order_revenue AS (
			SELECT date_trunc('` + truncInterval + `', created_at) AS date, 
			       COALESCE(SUM(total_amount), 0) AS revenue
			FROM orders
			WHERE deleted_at IS NULL
				AND status IN ('PAID', 'PROCESSING', 'SHIPPED', 'DELIVERED')
				` + dateConditions + `
			GROUP BY date_trunc('` + truncInterval + `', created_at)
		),
		payment_revenue AS (
			SELECT date_trunc('` + truncInterval + `', cp.paid_at) AS date, 
			       COALESCE(SUM(cp.amount), 0) AS revenue
			FROM contract_payments cp
			INNER JOIN contracts c ON c.id = cp.contract_id AND c.deleted_at IS NULL
			WHERE cp.deleted_at IS NULL
				AND cp.status = 'PAID'
				` + paymentDateConditions + `
			GROUP BY date_trunc('` + truncInterval + `', cp.paid_at)
		)
		SELECT 
			COALESCE(o.date, p.date) AS date,
			COALESCE(o.revenue, 0) + COALESCE(p.revenue, 0) AS revenue
		FROM order_revenue o
		FULL OUTER JOIN payment_revenue p ON o.date = p.date
		ORDER BY date
	`

	err := r.db.WithContext(ctx).Raw(query, params...).Scan(&results).Error
	return results, err
}

// =============================================================================
// ORDERS
// =============================================================================

// GetTotalOrdersCount returns the total number of orders
func (r *adminAnalyticsRepository) GetTotalOrdersCount(ctx context.Context, startDate, endDate *time.Time) (int64, error) {
	var count int64
	query := r.db.WithContext(ctx).
		Table("orders").
		Where("deleted_at IS NULL")

	if startDate != nil {
		query = query.Where("created_at >= ?", *startDate)
	}
	if endDate != nil {
		query = query.Where("created_at < ?", *endDate)
	}

	err := query.Count(&count).Error
	return count, err
}

// =============================================================================
// GROWTH TREND
// =============================================================================

// GetGrowthTrend returns overall growth trend
func (r *adminAnalyticsRepository) GetGrowthTrend(ctx context.Context, granularity string, startDate, endDate *time.Time) ([]dtos.GrowthTrendResult, error) {
	var results []dtos.GrowthTrendResult

	dateFunc := getDateTruncFunc(granularity)

	// Build the query using CTE
	query := `
		WITH dates AS (
			SELECT DISTINCT ` + dateFunc + ` AS date
			FROM generate_series(
				$1::timestamp,
				$2::timestamp,
				$3::interval
			) AS t(date)
		),
		user_counts AS (
			SELECT ` + dateFunc + ` AS date, COUNT(*) AS new_users
			FROM users
			WHERE deleted_at IS NULL
				AND created_at >= $1 AND created_at < $2
			GROUP BY ` + dateFunc + `
		),
		order_counts AS (
			SELECT ` + dateFunc + ` AS date, COUNT(*) AS new_orders, COALESCE(SUM(total_amount), 0) AS revenue
			FROM orders
			WHERE deleted_at IS NULL
				AND status IN ('PAID', 'PROCESSING', 'SHIPPED', 'DELIVERED')
				AND created_at >= $1 AND created_at < $2
			GROUP BY ` + dateFunc + `
		),
		contract_counts AS (
			SELECT ` + dateFunc + ` AS date, COUNT(*) AS new_contracts
			FROM contracts
			WHERE deleted_at IS NULL
				AND created_at >= $1 AND created_at < $2
			GROUP BY ` + dateFunc + `
		)
		SELECT 
			d.date,
			COALESCE(u.new_users, 0) AS new_users,
			COALESCE(o.new_orders, 0) AS new_orders,
			COALESCE(c.new_contracts, 0) AS new_contracts,
			COALESCE(o.revenue, 0) AS revenue
		FROM dates d
		LEFT JOIN user_counts u ON u.date = d.date
		LEFT JOIN order_counts o ON o.date = d.date
		LEFT JOIN contract_counts c ON c.date = d.date
		ORDER BY d.date
	`

	// Determine interval based on granularity
	var interval string
	switch granularity {
	case "DAY":
		interval = "1 day"
	case "WEEK":
		interval = "1 week"
	default:
		interval = "1 month"
	}

	// Default date range if not provided
	now := time.Now()
	start := now.AddDate(0, -6, 0) // 6 months ago
	end := now
	if startDate != nil {
		start = *startDate
	}
	if endDate != nil {
		end = *endDate
	}

	err := r.db.WithContext(ctx).Raw(query, start, end, interval).Scan(&results).Error
	return results, err
}

// getDateTruncFunc returns the appropriate date_trunc function based on granularity
func getDateTruncFunc(granularity string) string {
	switch granularity {
	case "DAY":
		return "date_trunc('day', created_at)"
	case "WEEK":
		return "date_trunc('week', created_at)"
	default:
		return "date_trunc('month', created_at)"
	}
}
