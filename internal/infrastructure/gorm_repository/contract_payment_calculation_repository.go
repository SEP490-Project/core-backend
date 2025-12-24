package gormrepository

import (
	"context"
	"core-backend/internal/application/dto/dtos"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/domain/constant"
	"core-backend/internal/domain/enum"
	"core-backend/pkg/utils"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// contractPaymentCalculationRepository implements ContractPaymentCalculationRepository
// with optimized queries for AFFILIATE and CO_PRODUCING payment calculations.
type contractPaymentCalculationRepository struct {
	db *gorm.DB
}

// NewContractPaymentCalculationRepository creates a new instance
func NewContractPaymentCalculationRepository(db *gorm.DB) irepository.ContractPaymentCalculationRepository {
	return &contractPaymentCalculationRepository{db: db}
}

// GetTotalClicksForContract counts all clicks for affiliate links
// associated with a contract within a payment period.
//
// Query path: contracts -> affiliate_links -> click_events
//
// This query is optimized for TimescaleDB by filtering on clicked_at
// which allows chunk exclusion for efficient time-range queries.
func (r *contractPaymentCalculationRepository) GetTotalClicksForContract(
	ctx context.Context,
	contractID uuid.UUID,
	periodStart, periodEnd time.Time,
) (int64, error) {
	var totalClicks int64

	err := r.db.WithContext(ctx).Raw(`
		SELECT COUNT(ce.id)
		FROM affiliate_links al
		JOIN click_events ce ON ce.affiliate_link_id = al.id
		WHERE (al.contract_id = ? OR al.metadata ->> 'contract_id' = ?)
			AND al.deleted_at IS NULL
			AND ce.clicked_at >= ?
			AND ce.clicked_at < ?
	`, contractID, contractID, periodStart, periodEnd).Scan(&totalClicks).Error

	if err != nil {
		return 0, err
	}

	return totalClicks, nil
}

// GetLimitedProductRevenue calculates total revenue from limited products
// associated with a contract within a payment period.
//
// Query path for CO_PRODUCING contracts:
//
//	contracts -> campaigns -> milestones -> tasks -> products (type=LIMITED) -> limited_products
//	Then: products -> product_variants -> order_items/pre_orders
//
// Revenue sources:
//  1. pre_orders (status = 'RECEIVED') - Pre-order purchases
//  2. orders (order_type = 'LIMITED', status = 'RECEIVED') - Regular limited product orders
func (r *contractPaymentCalculationRepository) GetLimitedProductRevenue(
	ctx context.Context,
	contractID uuid.UUID,
	periodStart, periodEnd time.Time,
) (*dtos.LimitedProductRevenueResult, error) {
	var preOrderRevenue, limitedOrderRevenue float64

	tasks := make([]func(context.Context) error, 0)

	// Get PreOrder revenue for limited products under this contract
	// Path: contracts -> campaigns -> milestones -> tasks -> products -> product_variants -> pre_orders
	tasks = append(tasks, func(ctx context.Context) error {
		query := `
		SELECT COALESCE(SUM(po.total_amount), 0) as preorder_revenue
		FROM pre_orders po
		JOIN product_variants pv ON pv.id = po.variant_id
		JOIN products p ON p.id = pv.product_id
		JOIN tasks t ON t.id = p.task_id
		JOIN milestones m ON m.id = t.milestone_id
		JOIN campaigns c ON c.id = m.campaign_id
		WHERE c.contract_id = ?
			AND p.type = ?
			AND po.status in ?
			AND po.deleted_at IS NULL
			AND p.deleted_at IS NULL
			AND t.deleted_at IS NULL
			AND m.deleted_at IS NULL
			AND c.deleted_at IS NULL
			AND po.created_at >= ?
			AND po.created_at < ?
		`
		return r.db.WithContext(ctx).Raw(
			query,
			contractID,
			enum.ProductTypeLimited,
			constant.ValidCompletedPreOrderStatus.ToStringSlice(),
			periodStart, periodEnd,
		).Scan(&preOrderRevenue).Error
	})

	// Get Order revenue for limited products under this contract
	// Path: contracts -> campaigns -> milestones -> tasks -> products -> product_variants -> order_items -> orders
	tasks = append(tasks, func(ctx context.Context) error {
		query := `
		SELECT COALESCE(SUM(oi.subtotal), 0) as order_revenue
		FROM orders o
		JOIN order_items oi ON oi.order_id = o.id
		JOIN product_variants pv ON pv.id = oi.variant_id
		JOIN products p ON p.id = pv.product_id
		JOIN tasks t ON t.id = p.task_id
		JOIN milestones m ON m.id = t.milestone_id
		JOIN campaigns c ON c.id = m.campaign_id
		WHERE c.contract_id = ?
			AND p.type = ?
			AND o.status IN ?
			AND o.order_type = ?
			AND o.deleted_at IS NULL
			AND oi.deleted_at IS NULL
			AND p.deleted_at IS NULL
			AND t.deleted_at IS NULL
			AND m.deleted_at IS NULL
			AND c.deleted_at IS NULL
			AND o.created_at >= ?
			AND o.created_at < ?
		`
		return r.db.WithContext(ctx).Raw(
			query,
			contractID,
			enum.ProductTypeLimited,
			constant.ValidCompletedOrderStatus.ToStringSlice(),
			enum.ProductTypeLimited,
			periodStart, periodEnd,
		).Scan(&limitedOrderRevenue).Error
	})
	if err := utils.RunParallel(ctx, 2, tasks...); err != nil {
		return nil, err
	}

	return &dtos.LimitedProductRevenueResult{
		PreOrderRevenue: preOrderRevenue,
		OrderRevenue:    limitedOrderRevenue,
		TotalRevenue:    preOrderRevenue + limitedOrderRevenue,
	}, nil
}
