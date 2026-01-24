package gormrepository

import (
	"context"
	"core-backend/internal/application/dto/dtos"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/domain/constant"
	"core-backend/internal/domain/enum"
	"strings"
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
		Where("last_login >= ?", cutoff).
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
func (r *adminAnalyticsRepository) GetUserGrowthTrend(ctx context.Context, granularity string, startDate, endDate *time.Time, role *string) ([]dtos.UserGrowthResult, error) {
	var results []dtos.UserGrowthResult

	dateFunc := getDateTruncFunc(granularity)
	query := r.db.WithContext(ctx).
		Table("users").
		Select(dateFunc + " AS date, COUNT(*) AS new_users").
		Where("deleted_at IS NULL")

	if role != nil {
		query = query.Where("role = ?", *role)
	}

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
		Joins("INNER JOIN contracts c ON c.brand_id = b.id AND c.deleted_at IS NULL AND c.status = ?", enum.ContractStatusActive.String()).
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
		Select("COALESCE(SUM((financial_terms ->> 'total_cost')::integer), 0)").
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
		Where("cp.status = ?", enum.ContractPaymentStatusPaid.String()).
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
		Where("cp.status IN ?", []string{enum.ContractPaymentStatusPending.String(), enum.ContractPaymentStatusOverdue.String()}).
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
		Where("status = ?", enum.ContentStatusPosted.String()).
		Count(&count).Error
	return count, err
}

// =============================================================================
// REVENUE
// =============================================================================

// GetTotalPlatformRevenue returns the total platform revenue
func (r *adminAnalyticsRepository) GetTotalPlatformRevenue(ctx context.Context, startDate, endDate *time.Time) (float64, error) {
	var total float64

	// Valid order statuses for revenue calculation
	validOrderStatuses := []string{
		enum.OrderStatusPaid.String(),
		enum.OrderStatusConfirmed.String(),
		enum.OrderStatusShipped.String(),
		enum.OrderStatusInTransit.String(),
		enum.OrderStatusDelivered.String(),
		enum.OrderStatusReceived.String(),
		enum.OrderStatusAwaitingPickUp.String(),
	}

	// Revenue from paid orders
	orderQuery := r.db.WithContext(ctx).
		Table("orders").
		Where("deleted_at IS NULL").
		Where("status IN ?", validOrderStatuses)

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
		Where("cp.status = ?", enum.ContractPaymentStatusPaid.String())

	if startDate != nil {
		paymentQuery = paymentQuery.Where("cp.created_at >= ? OR cp.updated_at >= ?", *startDate, *startDate)
	}
	if endDate != nil {
		paymentQuery = paymentQuery.Where("cp.created_at < ? OR cp.updated_at < ?", *endDate, *endDate)
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
		Where("cp.status = ?", enum.ContractPaymentStatusPaid.String()).
		Where("c.type = ?", contractType)

	if startDate != nil {
		query = query.Where("cp.created_at >= ? OR cp.updated_at >= ?", *startDate, *startDate)
	}
	if endDate != nil {
		query = query.Where("cp.created_at < ? OR cp.updated_at < ?", *endDate, *endDate)
	}

	err := query.Select("COALESCE(SUM(cp.amount), 0)").Scan(&total).Error
	return total, err
}

// GetPlatformProductRevenue returns revenue from product sales by type
func (r *adminAnalyticsRepository) GetPlatformProductRevenue(ctx context.Context, productType string, startDate, endDate *time.Time) (float64, error) {
	var total float64

	// Valid order statuses for revenue calculation
	validOrderStatuses := []string{
		enum.OrderStatusPaid.String(),
		enum.OrderStatusConfirmed.String(),
		enum.OrderStatusShipped.String(),
		enum.OrderStatusInTransit.String(),
		enum.OrderStatusDelivered.String(),
		enum.OrderStatusReceived.String(),
		enum.OrderStatusAwaitingPickUp.String(),
	}

	query := r.db.WithContext(ctx).
		Table("orders o").
		Joins("INNER JOIN order_items oi ON oi.order_id = o.id").
		Joins("INNER JOIN product_variants pv ON pv.id = oi.variant_id AND pv.deleted_at IS NULL").
		Joins("INNER JOIN products p on p.id = pv.product_id and p.deleted_at IS NULL").
		Where("o.deleted_at IS NULL").
		Where("o.status IN ?", validOrderStatuses).
		Where("p.type = ?", productType)

	if startDate != nil {
		query = query.Where("o.created_at >= ?", *startDate)
	}
	if endDate != nil {
		query = query.Where("o.created_at < ?", *endDate)
	}

	err := query.Select("COALESCE(SUM(oi.subtotal), 0)").Scan(&total).Error
	if productType == enum.ProductTypeLimited.String() {
		var revenue float64

		// Valid pre-order statuses
		validPreOrderStatuses := []string{
			enum.PreOrderStatusPaid.String(),
			enum.PreOrderStatusAwaitingPickup.String(),
			enum.PreOrderStatusInTransit.String(),
			enum.PreOrderStatusDelivered.String(),
			enum.PreOrderStatusReceived.String(),
		}

		query := r.db.WithContext(ctx).Table("pre_orders").
			Select("COALESCE(SUM(total_amount), 0)").
			Where("deleted_at IS NULL").
			Where("status IN ?", validPreOrderStatuses)

		if startDate != nil {
			query = query.Where("created_at >= ?", *startDate)
		}
		if endDate != nil {
			query = query.Where("created_at <= ?", *endDate)
		}

		if err = query.Scan(&revenue).Error; err != nil {
			return 0, err
		}
		total += revenue
	}
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

	// Build valid order status list for SQL
	validOrderStatuses := getValidOrderStatusesSQL()
	paidStatus := enum.ContractPaymentStatusPaid.String()

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

	// Build payment date filter (uses updated_at instead of created_at)
	var paymentDateConditions string
	if startDate != nil && endDate != nil {
		paymentDateConditions = "AND cp.updated_at >= ? AND cp.updated_at < ?"
		params = append(params, *startDate, *endDate)
	} else if startDate != nil {
		paymentDateConditions = "AND cp.updated_at >= ?"
		params = append(params, *startDate)
	} else if endDate != nil {
		paymentDateConditions = "AND cp.updated_at < ?"
		params = append(params, *endDate)
	}

	// CTE query combining order revenue and contract payment revenue
	query := `
		WITH order_revenue AS (
			SELECT date_trunc('` + truncInterval + `', created_at) AS date, 
			       COALESCE(SUM(total_amount), 0) AS revenue
			FROM orders
			WHERE deleted_at IS NULL
				AND status IN (` + validOrderStatuses + `)
				` + dateConditions + `
			GROUP BY date_trunc('` + truncInterval + `', created_at)
		),
		payment_revenue AS (
			SELECT date_trunc('` + truncInterval + `', cp.updated_at) AS date, 
			       COALESCE(SUM(cp.amount), 0) AS revenue
			FROM contract_payments cp
			INNER JOIN contracts c ON c.id = cp.contract_id AND c.deleted_at IS NULL
			WHERE cp.deleted_at IS NULL
				AND cp.status = '` + paidStatus + `'
				` + paymentDateConditions + `
			GROUP BY date_trunc('` + truncInterval + `', cp.updated_at)
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

	// Get date_trunc functions for different contexts
	dateFuncCreatedAt := getDateTruncFunc(granularity)               // For tables with created_at column
	dateFuncSeries := getDateTruncFuncForGenerateSeries(granularity) // For generate_series output

	// Get valid order statuses for revenue calculations
	validOrderStatuses := getValidOrderStatusesSQL()

	// Build the query using CTE
	query := `
		WITH dates AS (
			SELECT DISTINCT ` + dateFuncSeries + ` AS date
			FROM generate_series(
				$1::timestamp,
				$2::timestamp,
				$3::interval
			) AS t(date)
		),
		user_counts AS (
			SELECT ` + dateFuncCreatedAt + ` AS date, COUNT(*) AS new_users
			FROM users
			WHERE deleted_at IS NULL
				AND created_at >= $1 AND created_at < $2
			GROUP BY ` + dateFuncCreatedAt + `
		),
		order_counts AS (
			SELECT ` + dateFuncCreatedAt + ` AS date, COUNT(*) AS new_orders, COALESCE(SUM(total_amount), 0) AS revenue
			FROM orders
			WHERE deleted_at IS NULL
				AND status IN (` + validOrderStatuses + `) 
				AND created_at >= $1 AND created_at < $2
			GROUP BY ` + dateFuncCreatedAt + `
		),
		contract_counts AS (
			SELECT ` + dateFuncCreatedAt + ` AS date, COUNT(*) AS new_contracts
			FROM contracts
			WHERE deleted_at IS NULL
				AND created_at >= $1 AND created_at < $2
			GROUP BY ` + dateFuncCreatedAt + `
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

// =============================================================================
// CONSOLIDATED DASHBOARD QUERIES (optimized batch methods)
// =============================================================================

// GetDashboardUsersMetrics returns all user metrics in a single query
func (r *adminAnalyticsRepository) GetDashboardUsersMetrics(ctx context.Context, activeDays int, monthStart, monthEnd time.Time) (*dtos.DashboardUsersResult, error) {
	cutoff := time.Now().AddDate(0, 0, -activeDays)

	query := `
		SELECT
			COUNT(*) AS total_users,
			COUNT(*) FILTER (WHERE last_login >= $1) AS active_users,
			COUNT(*) FILTER (WHERE role = $4) AS admin,
			COUNT(*) FILTER (WHERE role = $5) AS marketing_staff,
			COUNT(*) FILTER (WHERE role = $6) AS sales_staff,
			COUNT(*) FILTER (WHERE role = $7) AS content_staff,
			COUNT(*) FILTER (WHERE role = $8) AS brand_partner,
			COUNT(*) FILTER (WHERE role = $9) AS customer,
			COUNT(*) FILTER (WHERE created_at >= $2 AND created_at < $3) AS new_this_month
		FROM users
		WHERE deleted_at IS NULL
	`

	var result dtos.DashboardUsersResult
	err := r.db.WithContext(ctx).Raw(query,
		cutoff, monthStart, monthEnd,
		enum.UserRoleAdmin.String(),
		enum.UserRoleMarketingStaff.String(),
		enum.UserRoleSalesStaff.String(),
		enum.UserRoleContentStaff.String(),
		enum.UserRoleBrandPartner.String(),
		enum.UserRoleCustomer.String(),
	).Scan(&result).Error
	return &result, err
}

// GetDashboardContractsMetrics returns all contract metrics in a single query
func (r *adminAnalyticsRepository) GetDashboardContractsMetrics(ctx context.Context) (*dtos.DashboardContractsResult, error) {
	query := `
		WITH contract_stats AS (
			SELECT
				COUNT(*) AS total_contracts,
				COUNT(*) FILTER (WHERE status = $1) AS draft,
				COUNT(*) FILTER (WHERE status = $2) AS approved,
				COUNT(*) FILTER (WHERE status = $3) AS active,
				COUNT(*) FILTER (WHERE status = $4) AS completed,
				COUNT(*) FILTER (WHERE status = $5) AS terminated,
				COALESCE(SUM((financial_terms ->> 'total_cost')::numeric), 0) AS total_value
			FROM contracts
			WHERE deleted_at IS NULL
		),
		payment_stats AS (
			SELECT
				COALESCE(SUM(cp.amount) FILTER (WHERE cp.status = $6), 0) AS collected_amount,
				COALESCE(SUM(cp.amount) FILTER (WHERE cp.status IN ($7, $8)), 0) AS pending_amount
			FROM contract_payments cp
			INNER JOIN contracts c ON c.id = cp.contract_id AND c.deleted_at IS NULL
			WHERE cp.deleted_at IS NULL
		)
		SELECT
			cs.total_contracts,
			cs.draft,
			cs.approved,
			cs.active,
			cs.completed,
			cs.terminated,
			cs.total_value,
			ps.collected_amount,
			ps.pending_amount
		FROM contract_stats cs, payment_stats ps
	`

	var result dtos.DashboardContractsResult
	err := r.db.WithContext(ctx).Raw(query,
		enum.ContractStatusDraft.String(),
		enum.ContractStatusApproved.String(),
		enum.ContractStatusActive.String(),
		enum.ContractStatusCompleted.String(),
		enum.ContractStatusTerminated.String(),
		enum.ContractPaymentStatusPaid.String(),
		enum.ContractPaymentStatusPending.String(),
		enum.ContractPaymentStatusOverdue.String(),
	).Scan(&result).Error
	return &result, err
}

// GetDashboardCampaignsMetrics returns all campaign metrics in a single query
func (r *adminAnalyticsRepository) GetDashboardCampaignsMetrics(ctx context.Context) (*dtos.DashboardCampaignsResult, error) {
	query := `
		WITH campaign_stats AS (
			SELECT
				COUNT(*) AS total_campaigns,
				COUNT(*) FILTER (WHERE status = $1) AS draft,
				COUNT(*) FILTER (WHERE status = $2) AS running,
				COUNT(*) FILTER (WHERE status = $3) AS completed,
				COUNT(*) FILTER (WHERE status = $4) AS cancelled
			FROM campaigns
			WHERE deleted_at IS NULL
		),
		content_stats AS (
			SELECT
				COUNT(*) AS content_created,
				COUNT(*) FILTER (WHERE status = $5) AS content_posted
			FROM contents
			WHERE deleted_at IS NULL
		)
		SELECT
			cs.total_campaigns,
			cs.draft,
			cs.running,
			cs.completed,
			cs.cancelled,
			cts.content_created,
			cts.content_posted
		FROM campaign_stats cs, content_stats cts
	`

	var result dtos.DashboardCampaignsResult
	err := r.db.WithContext(ctx).Raw(query,
		enum.CampaignDraft.String(),
		enum.CampaignRunning.String(),
		enum.CampaignCompleted.String(),
		enum.CampaignCancelled.String(),
		enum.ContentStatusPosted.String(),
	).Scan(&result).Error
	return &result, err
}

// GetDashboardBrandsMetrics returns all brand metrics in a single query
func (r *adminAnalyticsRepository) GetDashboardBrandsMetrics(ctx context.Context) (*dtos.DashboardBrandsResult, error) {
	query := `
		SELECT
			COUNT(*) AS total_brands,
			COUNT(DISTINCT b.id) FILTER (
				WHERE EXISTS (
					SELECT 1 FROM contracts c
					WHERE c.brand_id = b.id AND c.deleted_at IS NULL AND c.status = $1
				)
			) AS active_brands
		FROM brands b
		WHERE b.deleted_at IS NULL
	`

	var result dtos.DashboardBrandsResult
	err := r.db.WithContext(ctx).Raw(query, enum.ContractStatusActive.String()).Scan(&result).Error
	return &result, err
}

// GetDashboardOrdersMetrics returns all order metrics in a single query
func (r *adminAnalyticsRepository) GetDashboardOrdersMetrics(ctx context.Context, startDate, endDate *time.Time) (*dtos.DashboardOrdersResult, error) {
	query := `
		SELECT
			COUNT(*) AS total_orders,
			COUNT(*) FILTER (WHERE created_at >= $1 AND created_at < $2) AS monthly_orders
		FROM orders
		WHERE deleted_at IS NULL
	`

	var result dtos.DashboardOrdersResult
	err := r.db.WithContext(ctx).Raw(query, startDate, endDate).Scan(&result).Error
	return &result, err
}

// GetDashboardRevenueMetrics returns all revenue metrics in a single query
func (r *adminAnalyticsRepository) GetDashboardRevenueMetrics(ctx context.Context, startDate, endDate *time.Time) (*dtos.DashboardRevenueResult, error) {
	query := `
		WITH 
		-- 1. Orders within time range
		-- We grab total_amount (for global sum) and shipping_fee (for breakdown)
		range_orders AS (
			SELECT id, total_amount, shipping_fee
			FROM orders
			WHERE deleted_at IS NULL
				AND status = ANY($3)
				AND created_at >= $1 AND created_at < $2
		),

		-- 2. Pre-Orders within time range
		-- LOGIC: All Pre-Orders are considered 'Limited' products per requirement
		range_pre_orders AS (
			SELECT total_amount
			FROM pre_orders
			WHERE deleted_at IS NULL
				AND status = ANY($11)
				AND created_at >= $1 AND created_at < $2
		),

		-- 3. Contracts (B2B) within time range
		range_contracts AS (
			SELECT cp.amount, c.type
			FROM contract_payments cp
			INNER JOIN contracts c ON c.id = cp.contract_id
         	INNER JOIN payment_transactions pt ON pt.reference_id = cp.id 
				AND pt.method = 'PAYOS'
				AND pt.reference_type = 'CONTRACT_PAYMENT'
			WHERE cp.deleted_at IS NULL
				AND c.deleted_at IS NULL
				AND cp.status = $8
				AND pt.transaction_date >= $1 AND pt.transaction_date < $2
		),

		-- 4. Order Item Breakdown
		-- Isolates subtotal of items to distinguish Standard vs Limited inside Orders
		range_order_items AS (
			SELECT 
				p.type as product_type, 
				SUM(oi.subtotal) as subtotal
			FROM order_items oi
			INNER JOIN range_orders ro ON ro.id = oi.order_id
			INNER JOIN product_variants pv ON pv.id = oi.variant_id
			INNER JOIN products p ON p.id = pv.product_id
			WHERE pv.deleted_at IS NULL AND p.deleted_at IS NULL
			GROUP BY p.type
		),

		-- 5. Aggregation
		calculations AS (
			SELECT
				-- Base Totals
				COALESCE((SELECT SUM(total_amount) FROM range_orders), 0) as orders_total,
				COALESCE((SELECT SUM(total_amount) FROM range_pre_orders), 0) as pre_orders_total,
				COALESCE((SELECT SUM(amount) FROM range_contracts), 0) as contracts_total,
				COALESCE((SELECT SUM(shipping_fee) FROM range_orders), 0) as shipping_total,

				-- Contract Breakdowns
				COALESCE((SELECT SUM(amount) FROM range_contracts WHERE type = $4), 0) as adv_rev,
				COALESCE((SELECT SUM(amount) FROM range_contracts WHERE type = $5), 0) as aff_rev,
				COALESCE((SELECT SUM(amount) FROM range_contracts WHERE type = $6), 0) as amb_rev,
				COALESCE((SELECT SUM(amount) FROM range_contracts WHERE type = $7), 0) as co_rev,

				-- Product Logic
				-- Standard = Sum of Standard Items from Orders
				COALESCE((SELECT SUM(subtotal) FROM range_order_items WHERE product_type = $9), 0) as standard_rev,

				-- Limited = Sum of Limited Items from Orders + ALL Pre-Order Revenue
				COALESCE((SELECT SUM(subtotal) FROM range_order_items WHERE product_type = $10), 0) +
				COALESCE((SELECT SUM(total_amount) FROM range_pre_orders), 0) as limited_rev
		)
		SELECT 
			-- Total System Revenue
			(orders_total + pre_orders_total + contracts_total) AS total_revenue,
			(orders_total + pre_orders_total + contracts_total) AS monthly_revenue,
			
			-- Breakdowns
			adv_rev AS advertising_revenue,
			aff_rev AS affiliate_revenue,
			amb_rev AS ambassador_revenue,
			co_rev AS co_producing_revenue,
			
			standard_rev AS standard_product_revenue,
			limited_rev AS limited_product_revenue,
			shipping_total AS shipping_revenue
		FROM calculations
	`

	var result dtos.DashboardRevenueResult
	err := r.db.WithContext(ctx).Raw(query,
		startDate, // $1
		endDate,   // $2
		constant.ValidCompletedOrderStatus.ToStringSlice(),    // $3
		enum.ContractTypeAdvertising.String(),                 // $4
		enum.ContractTypeAffiliate.String(),                   // $5
		enum.ContractTypeAmbassador.String(),                  // $6
		enum.ContractTypeCoProduce.String(),                   // $7
		enum.ContractPaymentStatusPaid.String(),               // $8
		enum.ProductTypeStandard.String(),                     // $9
		enum.ProductTypeLimited.String(),                      // $10
		constant.ValidCompletedPreOrderStatus.ToStringSlice(), // $11
	).Scan(&result).Error

	return &result, err
}

// getDateTruncFunc returns the appropriate date_trunc function based on granularity
// This version uses 'created_at' column for tables
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

// getDateTruncFuncForGenerateSeries returns the appropriate date_trunc function for generate_series output
// This version uses 'date' column from generate_series alias
func getDateTruncFuncForGenerateSeries(granularity string) string {
	switch granularity {
	case "DAY":
		return "date_trunc('day', date)"
	case "WEEK":
		return "date_trunc('week', date)"
	default:
		return "date_trunc('month', date)"
	}
}

// getValidOrderStatusesSQL returns a SQL-safe string of valid order statuses for revenue calculations
// These are statuses that represent completed/valid orders for revenue reporting
func getValidOrderStatusesSQL() string {
	var quoted []string
	for _, s := range constant.ValidCompletedOrderStatus {
		quoted = append(quoted, "'"+s.String()+"'")
	}
	return strings.Join(quoted, ", ")
}

func getValidPreOrderStatusesSQL() string {
	var quoted []string
	for _, s := range constant.ValidCompletedPreOrderStatus {
		quoted = append(quoted, "'"+s.String()+"'")
	}
	return strings.Join(quoted, ", ")
}
