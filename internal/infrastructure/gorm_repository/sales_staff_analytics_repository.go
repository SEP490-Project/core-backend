package gormrepository

import (
	"context"
	"core-backend/internal/application/dto/dtos"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/domain/enum"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type salesStaffAnalyticsRepository struct {
	db *gorm.DB
}

// NewSalesStaffAnalyticsRepository creates a new sales staff analytics repository
func NewSalesStaffAnalyticsRepository(db *gorm.DB) irepository.SalesStaffAnalyticsRepository {
	return &salesStaffAnalyticsRepository{db: db}
}

// GetOrdersCountByType returns the count of orders by type (STANDARD or LIMITED)
func (r *salesStaffAnalyticsRepository) GetOrdersCountByType(ctx context.Context, orderType string, startDate, endDate *time.Time) (int64, error) {
	var count int64
	query := r.db.WithContext(ctx).Table("orders").Where("deleted_at IS NULL")

	if orderType != "" {
		query = query.Where("order_type = ?", orderType)
	}
	if startDate != nil {
		query = query.Where("created_at >= ?", *startDate)
	}
	if endDate != nil {
		query = query.Where("created_at <= ?", *endDate)
	}

	if err := query.Count(&count).Error; err != nil {
		zap.L().Error("Failed to get orders count by type", zap.Error(err))
		return 0, err
	}
	return count, nil
}

// GetOrdersRevenueByType returns the total revenue from orders by type
func (r *salesStaffAnalyticsRepository) GetOrdersRevenueByType(ctx context.Context, orderType string, startDate, endDate *time.Time) (float64, error) {
	var revenue float64
	query := r.db.WithContext(ctx).Table("orders").
		Select("COALESCE(SUM(total_amount), 0)").
		Where("deleted_at IS NULL").
		Where("status = ?", enum.OrderStatusReceived.String()) // Only completed orders

	if orderType != "" {
		query = query.Where("order_type = ?", orderType)
	}
	if startDate != nil {
		query = query.Where("created_at >= ?", *startDate)
	}
	if endDate != nil {
		query = query.Where("created_at <= ?", *endDate)
	}

	if err := query.Scan(&revenue).Error; err != nil {
		zap.L().Error("Failed to get orders revenue by type", zap.Error(err))
		return 0, err
	}
	return revenue, nil
}

// GetOrdersCountByStatus returns the count of orders by type and status
func (r *salesStaffAnalyticsRepository) GetOrdersCountByStatus(ctx context.Context, orderType, status string, startDate, endDate *time.Time) (int64, error) {
	var count int64
	query := r.db.WithContext(ctx).Table("orders").Where("deleted_at IS NULL")

	if orderType != "" {
		query = query.Where("order_type = ?", orderType)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if startDate != nil {
		query = query.Where("created_at >= ?", *startDate)
	}
	if endDate != nil {
		query = query.Where("created_at <= ?", *endDate)
	}

	if err := query.Count(&count).Error; err != nil {
		zap.L().Error("Failed to get orders count by status", zap.Error(err))
		return 0, err
	}
	return count, nil
}

// GetPreOrdersCount returns the count of pre-orders
func (r *salesStaffAnalyticsRepository) GetPreOrdersCount(ctx context.Context, startDate, endDate *time.Time) (int64, error) {
	var count int64
	query := r.db.WithContext(ctx).Table("pre_orders").Where("deleted_at IS NULL")

	if startDate != nil {
		query = query.Where("created_at >= ?", *startDate)
	}
	if endDate != nil {
		query = query.Where("created_at <= ?", *endDate)
	}

	if err := query.Count(&count).Error; err != nil {
		zap.L().Error("Failed to get pre-orders count", zap.Error(err))
		return 0, err
	}
	return count, nil
}

// GetPreOrdersRevenue returns the total revenue from pre-orders
func (r *salesStaffAnalyticsRepository) GetPreOrdersRevenue(ctx context.Context, startDate, endDate *time.Time) (float64, error) {
	var revenue float64
	query := r.db.WithContext(ctx).Table("pre_orders").
		Select("COALESCE(SUM(total_amount), 0)").
		Where("deleted_at IS NULL").
		Where("status = ?", enum.PreOrderStatusReceived.String())

	if startDate != nil {
		query = query.Where("created_at >= ?", *startDate)
	}
	if endDate != nil {
		query = query.Where("created_at <= ?", *endDate)
	}

	if err := query.Scan(&revenue).Error; err != nil {
		zap.L().Error("Failed to get pre-orders revenue", zap.Error(err))
		return 0, err
	}
	return revenue, nil
}

// GetPreOrdersCountByStatus returns the count of pre-orders by status
func (r *salesStaffAnalyticsRepository) GetPreOrdersCountByStatus(ctx context.Context, status string, startDate, endDate *time.Time) (int64, error) {
	var count int64
	query := r.db.WithContext(ctx).Table("pre_orders").Where("deleted_at IS NULL")

	if status != "" {
		query = query.Where("status = ?", status)
	}
	if startDate != nil {
		query = query.Where("created_at >= ?", *startDate)
	}
	if endDate != nil {
		query = query.Where("created_at <= ?", *endDate)
	}

	if err := query.Count(&count).Error; err != nil {
		zap.L().Error("Failed to get pre-orders count by status", zap.Error(err))
		return 0, err
	}
	return count, nil
}

// GetContractRevenueByType returns the total revenue from contract payments by contract type
func (r *salesStaffAnalyticsRepository) GetContractRevenueByType(ctx context.Context, contractType string, startDate, endDate *time.Time) (float64, error) {
	var revenue float64
	query := r.db.WithContext(ctx).Table("contract_payments cp").
		Select("COALESCE(SUM(cp.amount), 0)").
		Joins("JOIN contracts c ON c.id = cp.contract_id").
		Where("cp.deleted_at IS NULL").
		Where("cp.status = ?", enum.ContractPaymentStatusPaid.String())

	if contractType != "" {
		query = query.Where("c.type = ?", contractType)
	}
	if startDate != nil {
		query = query.Where("cp.due_date >= ?", *startDate)
	}
	if endDate != nil {
		query = query.Where("cp.due_date <= ?", *endDate)
	}

	if err := query.Scan(&revenue).Error; err != nil {
		zap.L().Error("Failed to get contract revenue by type", zap.Error(err))
		return 0, err
	}
	return revenue, nil
}

// GetTopBrandsByRevenue returns top brands by total revenue
func (r *salesStaffAnalyticsRepository) GetTopBrandsByRevenue(ctx context.Context, limit int, startDate, endDate *time.Time) ([]dtos.BrandRevenueResult, error) {
	var results []dtos.BrandRevenueResult

	// Build date filter clause
	dateFilter := ""
	receivedStatus := enum.OrderStatusReceived.String()
	args := []any{receivedStatus}
	if startDate != nil && endDate != nil {
		dateFilter = "AND o.created_at BETWEEN ? AND ?"
		args = append(args, *startDate, *endDate)
	}

	query := `
		WITH order_revenue AS (
			SELECT 
				p.brand_id,
				SUM(oi.subtotal) as revenue,
				COUNT(DISTINCT o.id) as order_count,
				COUNT(DISTINCT p.id) as product_count
			FROM orders o
			JOIN order_items oi ON oi.order_id = o.id
			JOIN product_variants pv ON pv.id = oi.variant_id
			JOIN products p ON p.id = pv.product_id
			WHERE o.deleted_at IS NULL 
			AND o.status = ?
			` + dateFilter + `
			GROUP BY p.brand_id
		)
		SELECT 
			b.id as brand_id,
			b.name as brand_name,
			COALESCE(r.revenue, 0) as total_revenue,
			COALESCE(r.order_count, 0) as order_count,
			COALESCE(r.product_count, 0) as product_count
		FROM brands b
		LEFT JOIN order_revenue r ON r.brand_id = b.id
		WHERE b.deleted_at IS NULL
		ORDER BY total_revenue DESC
		LIMIT ?
	`

	args = append(args, limit)
	if err := r.db.WithContext(ctx).Raw(query, args...).Scan(&results).Error; err != nil {
		zap.L().Error("Failed to get top brands by revenue", zap.Error(err))
		return nil, err
	}
	return results, nil
}

// GetTopProductsByRevenue returns top products by revenue
func (r *salesStaffAnalyticsRepository) GetTopProductsByRevenue(ctx context.Context, productType string, limit int, startDate, endDate *time.Time) ([]dtos.ProductRevenueResult, error) {
	var results []dtos.ProductRevenueResult

	query := r.db.WithContext(ctx).Table("order_items oi").
		Select(`
			p.id as product_id,
			p.name as product_name,
			b.name as brand_name,
			p.type as product_type,
			SUM(oi.subtotal) as total_revenue,
			SUM(oi.quantity) as units_sold
		`).
		Joins("JOIN orders o ON o.id = oi.order_id").
		Joins("JOIN product_variants pv ON pv.id = oi.variant_id").
		Joins("JOIN products p ON p.id = pv.product_id").
		Joins("JOIN brands b ON b.id = p.brand_id").
		Where("o.deleted_at IS NULL").
		Where("o.status = ?", enum.OrderStatusReceived.String())

	if productType != "" {
		query = query.Where("p.type = ?", productType)
	}
	if startDate != nil {
		query = query.Where("o.created_at >= ?", *startDate)
	}
	if endDate != nil {
		query = query.Where("o.created_at <= ?", *endDate)
	}

	query = query.Group("p.id, p.name, b.name, p.type").
		Order("total_revenue DESC").
		Limit(limit)

	if err := query.Scan(&results).Error; err != nil {
		zap.L().Error("Failed to get top products by revenue", zap.Error(err))
		return nil, err
	}
	return results, nil
}

// GetRevenueTrend returns revenue trend over time
func (r *salesStaffAnalyticsRepository) GetRevenueTrend(ctx context.Context, granularity string, startDate, endDate *time.Time) ([]dtos.RevenueTrendResult, error) {
	var results []dtos.RevenueTrendResult

	timeBucket := "date_trunc('day', o.created_at)"
	switch granularity {
	case "WEEK":
		timeBucket = "date_trunc('week', o.created_at)"
	case "MONTH":
		timeBucket = "date_trunc('month', o.created_at)"
	}

	query := r.db.WithContext(ctx).Table("orders o").
		Select(`
			`+timeBucket+` as date,
			SUM(o.total_amount) as revenue,
			COUNT(*) as order_count,
			AVG(o.total_amount) as average_order_value
		`).
		Where("o.deleted_at IS NULL").
		Where("o.status = ?", enum.OrderStatusReceived.String())

	if startDate != nil {
		query = query.Where("o.created_at >= ?", *startDate)
	}
	if endDate != nil {
		query = query.Where("o.created_at <= ?", *endDate)
	}

	query = query.Group(timeBucket).Order("date ASC")

	if err := query.Scan(&results).Error; err != nil {
		zap.L().Error("Failed to get revenue trend", zap.Error(err))
		return nil, err
	}
	return results, nil
}

// GetRecentOrders returns recent orders
func (r *salesStaffAnalyticsRepository) GetRecentOrders(ctx context.Context, limit int) ([]dtos.RecentOrderResult, error) {
	var results []dtos.RecentOrderResult

	query := r.db.WithContext(ctx).Table("orders o").
		Select(`
			o.id as order_id,
			u.full_name as customer_name,
			o.total_amount,
			o.status,
			o.order_type,
			(SELECT COUNT(*) FROM order_items WHERE order_id = o.id) as item_count,
			o.created_at
		`).
		Joins("LEFT JOIN users u ON u.id = o.user_id").
		Where("o.deleted_at IS NULL").
		Order("o.created_at DESC").
		Limit(limit)

	if err := query.Scan(&results).Error; err != nil {
		zap.L().Error("Failed to get recent orders", zap.Error(err))
		return nil, err
	}
	return results, nil
}

// GetPaymentStatusCounts returns payment status counts and amounts
func (r *salesStaffAnalyticsRepository) GetPaymentStatusCounts(ctx context.Context, contractID *uuid.UUID, startDate, endDate *time.Time) (*dtos.PaymentStatusResult, error) {
	result := &dtos.PaymentStatusResult{}

	paidStatus := enum.ContractPaymentStatusPaid.String()
	pendingStatus := enum.ContractPaymentStatusPending.String()
	now := time.Now()

	baseQuery := r.db.WithContext(ctx).Table("contract_payments cp").
		Where("cp.deleted_at IS NULL")

	if contractID != nil {
		baseQuery = baseQuery.Where("cp.contract_id = ?", *contractID)
	}
	if startDate != nil {
		baseQuery = baseQuery.Where("cp.due_date >= ?", *startDate)
	}
	if endDate != nil {
		baseQuery = baseQuery.Where("cp.due_date <= ?", *endDate)
	}

	// Total counts
	if err := baseQuery.Count(&result.TotalPayments).Error; err != nil {
		return nil, err
	}

	// Paid
	if err := r.db.WithContext(ctx).Table("contract_payments cp").
		Where("cp.deleted_at IS NULL").
		Where("cp.status = ?", paidStatus).
		Count(&result.PaidPayments).Error; err != nil {
		return nil, err
	}

	// Pending (due date in future)
	if err := r.db.WithContext(ctx).Table("contract_payments cp").
		Where("cp.deleted_at IS NULL").
		Where("cp.status = ?", pendingStatus).
		Where("cp.due_date >= ?", now).
		Count(&result.PendingPayments).Error; err != nil {
		return nil, err
	}

	// Overdue (pending but due date passed)
	if err := r.db.WithContext(ctx).Table("contract_payments cp").
		Where("cp.deleted_at IS NULL").
		Where("cp.status = ?", pendingStatus).
		Where("cp.due_date < ?", now).
		Count(&result.OverduePayments).Error; err != nil {
		return nil, err
	}

	// Amounts
	r.db.WithContext(ctx).Table("contract_payments cp").
		Select("COALESCE(SUM(amount), 0)").
		Where("cp.deleted_at IS NULL").
		Scan(&result.TotalAmount)

	r.db.WithContext(ctx).Table("contract_payments cp").
		Select("COALESCE(SUM(amount), 0)").
		Where("cp.deleted_at IS NULL").
		Where("cp.status = ?", paidStatus).
		Scan(&result.PaidAmount)

	r.db.WithContext(ctx).Table("contract_payments cp").
		Select("COALESCE(SUM(amount), 0)").
		Where("cp.deleted_at IS NULL").
		Where("cp.status = ?", pendingStatus).
		Where("cp.due_date >= ?", now).
		Scan(&result.PendingAmount)

	r.db.WithContext(ctx).Table("contract_payments cp").
		Select("COALESCE(SUM(amount), 0)").
		Where("cp.deleted_at IS NULL").
		Where("cp.status = ?", pendingStatus).
		Where("cp.due_date < ?", now).
		Scan(&result.OverdueAmount)

	return result, nil
}
