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

type brandPartnerAnalyticsRepository struct {
	db *gorm.DB
}

// NewBrandPartnerAnalyticsRepository creates a new brand partner analytics repository
func NewBrandPartnerAnalyticsRepository(db *gorm.DB) irepository.BrandPartnerAnalyticsRepository {
	return &brandPartnerAnalyticsRepository{db: db}
}

// GetBrandContractCount returns count of contracts for a brand
func (r *brandPartnerAnalyticsRepository) GetBrandContractCount(ctx context.Context, brandUserID uuid.UUID, status *string) (int64, error) {
	var count int64
	query := r.db.WithContext(ctx).Table("contracts").
		Joins("inner join brands on brands.id = contracts.brand_id").
		Where("brands.user_id = ?", brandUserID).
		Where("contracts.deleted_at IS NULL")

	if status != nil && *status != "" {
		query = query.Where("contracts.status = ?", *status)
	}

	if err := query.Count(&count).Error; err != nil {
		zap.L().Error("Failed to get brand contract count", zap.Error(err))
		return 0, err
	}
	return count, nil
}

// GetBrandCampaignCount returns count of campaigns for a brand
func (r *brandPartnerAnalyticsRepository) GetBrandCampaignCount(ctx context.Context, brandUserID uuid.UUID, status *string) (int64, error) {
	var count int64
	query := r.db.WithContext(ctx).Table("campaigns cmp").
		Joins("JOIN contracts c ON c.id = cmp.contract_id").
		Joins("JOIN brands b ON b.id = c.brand_id").
		Where("b.user_id = ?", brandUserID).
		Where("cmp.deleted_at IS NULL")

	if status != nil && *status != "" {
		query = query.Where("cmp.status = ?", *status)
	}

	if err := query.Count(&count).Error; err != nil {
		zap.L().Error("Failed to get brand campaign count", zap.Error(err))
		return 0, err
	}
	return count, nil
}

// GetBrandTotalRevenue returns total revenue from product sales for a brand
func (r *brandPartnerAnalyticsRepository) GetBrandTotalRevenue(ctx context.Context, brandUserID uuid.UUID, startDate, endDate *time.Time) (float64, error) {
	var revenue float64

	receivedStatus := enum.OrderStatusReceived.String()

	query := r.db.WithContext(ctx).Table("orders o").
		Select("COALESCE(SUM(oi.subtotal), 0)").
		Joins("JOIN order_items oi ON oi.order_id = o.id").
		Joins("JOIN product_variants pv ON pv.id = oi.variant_id").
		Joins("JOIN products p ON p.id = pv.product_id").
		Joins("JOIN brands b ON b.id = p.brand_id").
		Where("b.user_id = ?", brandUserID).
		Where("o.status = ?", receivedStatus).
		Where("o.deleted_at IS NULL")

	if startDate != nil {
		query = query.Where("o.created_at >= ?", *startDate)
	}
	if endDate != nil {
		query = query.Where("o.created_at <= ?", *endDate)
	}

	if err := query.Scan(&revenue).Error; err != nil {
		zap.L().Error("Failed to get brand total revenue", zap.Error(err))
		return 0, err
	}
	return revenue, nil
}

// GetBrandTotalPayments returns total payments received from contracts
func (r *brandPartnerAnalyticsRepository) GetBrandTotalPayments(ctx context.Context, brandUserID uuid.UUID, startDate, endDate *time.Time) (float64, error) {
	var payments float64

	paidStatus := enum.ContractPaymentStatusPaid.String()

	query := r.db.WithContext(ctx).Table("contract_payments cp").
		Select("COALESCE(SUM(cp.amount), 0)").
		Joins("JOIN contracts c ON c.id = cp.contract_id").
		Joins("JOIN brands b ON b.id = c.brand_id").
		Where("b.user_id = ?", brandUserID).
		Where("cp.status = ?", paidStatus).
		Where("cp.deleted_at IS NULL")

	if startDate != nil {
		query = query.Where("cp.due_date >= ?", *startDate)
	}
	if endDate != nil {
		query = query.Where("cp.due_date <= ?", *endDate)
	}

	if err := query.Scan(&payments).Error; err != nil {
		zap.L().Error("Failed to get brand total payments", zap.Error(err))
		return 0, err
	}
	return payments, nil
}

// GetBrandPendingPayments returns pending payments for a brand
func (r *brandPartnerAnalyticsRepository) GetBrandPendingPayments(ctx context.Context, brandUserID uuid.UUID) (float64, error) {
	var pending float64

	pendingStatus := enum.ContractPaymentStatusPending.String()

	query := r.db.WithContext(ctx).Table("contract_payments cp").
		Select("COALESCE(SUM(cp.amount), 0)").
		Joins("JOIN contracts c ON c.id = cp.contract_id").
		Joins("JOIN brands b ON b.id = c.brand_id").
		Where("b.user_id = ?", brandUserID).
		Where("cp.status = ?", pendingStatus).
		Where("cp.deleted_at IS NULL")

	if err := query.Scan(&pending).Error; err != nil {
		zap.L().Error("Failed to get brand pending payments", zap.Error(err))
		return 0, err
	}
	return pending, nil
}

// GetBrandProductCount returns count of products for a brand
func (r *brandPartnerAnalyticsRepository) GetBrandProductCount(ctx context.Context, brandUserID uuid.UUID, status *string) (int64, error) {
	var count int64
	query := r.db.WithContext(ctx).Table("products").
		Joins("JOIN brands ON brands.id = products.brand_id").
		Joins("JOIN users ON users.id = brands.user_id").
		Where("users.id = ?", brandUserID).
		Where("products.deleted_at IS NULL")

	if status != nil && *status != "" {
		query = query.Where("status = ?", *status)
	}

	if err := query.Count(&count).Error; err != nil {
		zap.L().Error("Failed to get brand product count", zap.Error(err))
		return 0, err
	}
	return count, nil
}

// GetBrandOrderCount returns count of orders for a brand's products
func (r *brandPartnerAnalyticsRepository) GetBrandOrderCount(ctx context.Context, brandUserID uuid.UUID, status *string, startDate, endDate *time.Time) (int64, error) {
	var count int64
	query := r.db.WithContext(ctx).Table("orders o").
		Select("COUNT(DISTINCT o.id)").
		Joins("JOIN order_items oi ON oi.order_id = o.id").
		Joins("JOIN product_variants pv ON pv.id = oi.variant_id").
		Joins("JOIN products p ON p.id = pv.product_id").
		Joins("JOIN brands b ON b.id = p.brand_id").
		Joins("JOIN users u ON u.id = b.user_id").
		Where("u.id = ?", brandUserID).
		Where("o.deleted_at IS NULL")

	if status != nil && *status != "" {
		query = query.Where("o.status = ?", *status)
	}
	if startDate != nil {
		query = query.Where("o.created_at >= ?", *startDate)
	}
	if endDate != nil {
		query = query.Where("o.created_at <= ?", *endDate)
	}

	if err := query.Scan(&count).Error; err != nil {
		zap.L().Error("Failed to get brand order count", zap.Error(err))
		return 0, err
	}
	return count, nil
}

// GetBrandTopProducts returns top products by revenue for a brand
func (r *brandPartnerAnalyticsRepository) GetBrandTopProducts(ctx context.Context, brandUserID uuid.UUID, limit int, startDate, endDate *time.Time) ([]dtos.BrandProductMetrics, error) {
	var results []dtos.BrandProductMetrics

	receivedStatus := enum.OrderStatusReceived.String()

	query := r.db.WithContext(ctx).Table("products p").
		Select(`
			p.id as product_id,
			p.name as product_name,
			p.type as product_type,
			p.status,
			COUNT(DISTINCT o.id) as order_count,
			COALESCE(SUM(oi.quantity), 0) as units_sold,
			COALESCE(SUM(oi.subtotal), 0) as revenue
		`).
		Joins("LEFT JOIN product_variants pv ON pv.product_id = p.id").
		Joins("LEFT JOIN order_items oi ON oi.variant_id = pv.id").
		Joins("LEFT JOIN orders o ON o.id = oi.order_id AND o.status = ?", receivedStatus).
		Joins("JOIN brands b ON b.id = p.brand_id").
		Joins("JOIN users u ON u.id = b.user_id").
		Where("u.id = ?", brandUserID).
		Where("p.deleted_at IS NULL")

	if startDate != nil {
		query = query.Where("(o.created_at >= ? OR o.created_at IS NULL)", *startDate)
	}
	if endDate != nil {
		query = query.Where("(o.created_at <= ? OR o.created_at IS NULL)", *endDate)
	}

	query = query.Group("p.id, p.name, p.type, p.status").
		Order("revenue DESC").
		Limit(limit)

	if err := query.Scan(&results).Error; err != nil {
		zap.L().Error("Failed to get brand top products", zap.Error(err))
		return nil, err
	}
	return results, nil
}

// GetBrandCampaignMetrics returns campaign performance metrics for a brand
func (r *brandPartnerAnalyticsRepository) GetBrandCampaignMetrics(ctx context.Context, brandUserID uuid.UUID, limit int, startDate, endDate *time.Time) ([]dtos.BrandCampaignMetrics, error) {
	var results []dtos.BrandCampaignMetrics

	doneStatus := enum.TaskStatusDone.String()

	query := r.db.WithContext(ctx).Table("campaigns cmp").
		Select(`
			cmp.id as campaign_id,
			cmp.name as campaign_name,
			cmp.status,
			cmp.start_date,
			cmp.end_date,
			COUNT(DISTINCT m.id) as milestone_count,
			COUNT(DISTINCT t.id) as task_count,
			SUM(CASE WHEN t.status = ? THEN 1 ELSE 0 END) as completed_tasks,
			COUNT(DISTINCT ct.id) as content_count,
			COALESCE(SUM((cc.metrics->>'views')::int), 0) as total_views,
			COALESCE(SUM((cc.metrics->>'likes')::int) + SUM((cc.metrics->>'comments')::int) + SUM((cc.metrics->>'shares')::int), 0) as total_engagements
		`, doneStatus).
		Joins("JOIN contracts c ON c.id = cmp.contract_id").
		Joins("LEFT JOIN milestones m ON m.campaign_id = cmp.id").
		Joins("LEFT JOIN tasks t ON t.milestone_id = m.id").
		Joins("LEFT JOIN contents ct ON ct.task_id = t.id").
		Joins("LEFT JOIN content_channels cc ON cc.content_id = ct.id").
		Joins("JOIN brands b ON b.id = c.brand_id").
		Where("b.user_id = ?", brandUserID).
		Where("cmp.deleted_at IS NULL")

	if startDate != nil {
		query = query.Where("cmp.start_date >= ?", *startDate)
	}
	if endDate != nil {
		query = query.Where("cmp.start_date <= ?", *endDate)
	}

	query = query.Group("cmp.id, cmp.name, cmp.status, cmp.start_date, cmp.end_date").
		Order("cmp.start_date DESC").
		Limit(limit)

	if err := query.Scan(&results).Error; err != nil {
		zap.L().Error("Failed to get brand campaign metrics", zap.Error(err))
		return nil, err
	}
	return results, nil
}

// GetBrandContentMetrics returns content performance metrics for a brand
func (r *brandPartnerAnalyticsRepository) GetBrandContentMetrics(ctx context.Context, brandUserID uuid.UUID, startDate, endDate *time.Time) (*dtos.BrandContentMetrics, error) {
	var result dtos.BrandContentMetrics

	postedStatus := enum.ContentStatusPosted.String()

	query := r.db.WithContext(ctx).Table("contents ct").
		Select(`
			COUNT(DISTINCT ct.id) as total_content,
			SUM(CASE WHEN ct.status = ? THEN 1 ELSE 0 END) as posted_content,
			COALESCE(SUM((cc.metrics->>'views')::int), 0) as total_views,
			COALESCE(SUM((cc.metrics->>'likes')::int), 0) as total_likes,
			COALESCE(SUM((cc.metrics->>'comments')::int), 0) as total_comments,
			COALESCE(SUM((cc.metrics->>'shares')::int), 0) as total_shares
		`, postedStatus).
		Joins("JOIN tasks t ON t.id = ct.task_id").
		Joins("JOIN milestones m ON m.id = t.milestone_id").
		Joins("JOIN campaigns cmp ON cmp.id = m.campaign_id").
		Joins("JOIN contracts c ON c.id = cmp.contract_id").
		Joins("LEFT JOIN content_channels cc ON cc.content_id = ct.id").
		Joins("JOIN brands b ON b.id = c.brand_id").
		Where("b.user_id = ?", brandUserID).
		Where("ct.deleted_at IS NULL")

	if startDate != nil {
		query = query.Where("ct.created_at >= ?", *startDate)
	}
	if endDate != nil {
		query = query.Where("ct.created_at <= ?", *endDate)
	}

	if err := query.Scan(&result).Error; err != nil {
		zap.L().Error("Failed to get brand content metrics", zap.Error(err))
		return nil, err
	}

	// Calculate engagement rate
	if result.TotalViews > 0 {
		result.EngagementRate = float64(result.TotalLikes+result.TotalComments+result.TotalShares) / float64(result.TotalViews) * 100
	}

	return &result, nil
}

// GetBrandRevenueTrend returns revenue time-series for a brand
func (r *brandPartnerAnalyticsRepository) GetBrandRevenueTrend(ctx context.Context, brandUserID uuid.UUID, granularity string, startDate, endDate *time.Time) ([]dtos.BrandRevenueTrendResult, error) {
	var results []dtos.BrandRevenueTrendResult

	timeBucket := "date_trunc('day', o.created_at)"
	switch granularity {
	case "WEEK":
		timeBucket = "date_trunc('week', o.created_at)"
	case "MONTH":
		timeBucket = "date_trunc('month', o.created_at)"
	}

	receivedStatus := enum.OrderStatusReceived.String()

	query := r.db.WithContext(ctx).Table("orders o").
		Select(`
			`+timeBucket+` as date,
			COUNT(DISTINCT o.id) as order_count,
			COALESCE(SUM(oi.quantity), 0) as units_sold,
			COALESCE(SUM(oi.subtotal), 0) as revenue
		`).
		Joins("JOIN order_items oi ON oi.order_id = o.id").
		Joins("JOIN product_variants pv ON pv.id = oi.variant_id").
		Joins("JOIN products p ON p.id = pv.product_id").
		Joins("JOIN brands b ON b.id = p.brand_id").
		Joins("JOIN users u ON u.id = b.user_id").
		Where("u.id = ?", brandUserID).
		Where("o.status = ?", receivedStatus).
		Where("o.deleted_at IS NULL")

	if startDate != nil {
		query = query.Where("o.created_at >= ?", *startDate)
	}
	if endDate != nil {
		query = query.Where("o.created_at <= ?", *endDate)
	}

	query = query.Group(timeBucket).Order("date ASC")

	if err := query.Scan(&results).Error; err != nil {
		zap.L().Error("Failed to get brand revenue trend", zap.Error(err))
		return nil, err
	}
	return results, nil
}

// GetBrandAffiliateMetrics returns affiliate link performance for a brand
func (r *brandPartnerAnalyticsRepository) GetBrandAffiliateMetrics(ctx context.Context, brandUserID uuid.UUID, startDate, endDate *time.Time) (*dtos.BrandAffiliateMetrics, error) {
	var result dtos.BrandAffiliateMetrics

	// Get affiliate link stats
	query := r.db.WithContext(ctx).Table("affiliate_links al").
		Select(`
			COUNT(DISTINCT al.id) as total_links,
			COUNT(DISTINCT CASE WHEN al.status = ? THEN al.id END) as active_links,
			COALESCE(SUM(ce.click_count), 0) as total_clicks
		`, enum.AffiliateLinkStatusActive.String()).
		Joins("JOIN contracts c ON c.id = al.contract_id").
		Joins(`LEFT JOIN (
			SELECT affiliate_link_id, COUNT(*) as click_count 
			FROM click_events 
			GROUP BY affiliate_link_id
		) ce ON ce.affiliate_link_id = al.id`).
		Joins("JOIN brands b ON b.id = c.brand_id").
		Where("b.user_id = ?", brandUserID).
		Where("al.deleted_at IS NULL")

	if err := query.Scan(&result).Error; err != nil {
		zap.L().Error("Failed to get brand affiliate metrics", zap.Error(err))
		return nil, err
	}

	return &result, nil
}

// GetBrandContractDetails returns contract details for a brand
func (r *brandPartnerAnalyticsRepository) GetBrandContractDetails(ctx context.Context, brandUserID uuid.UUID, limit int) ([]dtos.BrandContractDetails, error) {
	var results []dtos.BrandContractDetails

	paidStatus := enum.ContractPaymentStatusPaid.String()
	pendingStatus := enum.ContractPaymentStatusPending.String()

	query := `
		WITH target_contracts AS (
			-- 1. First, find ONLY the contracts for this user.
			SELECT c.id, c.contract_number, c.type, c.status, c.financial_terms, c.start_date, c.end_date
			FROM contracts c
			JOIN brands b ON b.id = c.brand_id
			WHERE b.user_id = ? 
			  AND c.deleted_at IS NULL
		),
		paid_payments AS (
			-- 2. Only sum payments for the contracts found above
			SELECT cp.contract_id, SUM(cp.amount) as amount
			FROM contract_payments cp
			JOIN target_contracts tc ON tc.id = cp.contract_id
			WHERE cp.status = ? AND cp.deleted_at IS NULL
			GROUP BY cp.contract_id
		),
		pending_payments AS (
			-- 3. Only sum pending for the contracts found above
			SELECT cp.contract_id, SUM(cp.amount) as amount
			FROM contract_payments cp
			JOIN target_contracts tc ON tc.id = cp.contract_id
			WHERE cp.status = ? AND cp.deleted_at IS NULL
			GROUP BY cp.contract_id
		),
		campaign_counts AS (
			-- 4. Only count campaigns for the contracts found above
			SELECT cmp.contract_id, COUNT(cmp.id) as cnt
			FROM campaigns cmp
			JOIN target_contracts tc ON tc.id = cmp.contract_id
			WHERE cmp.deleted_at IS NULL
			GROUP BY cmp.contract_id
		)
		SELECT tc.id as contract_id	paidStatus := enum.ContractPaymentStatusPaid.String()
	pendingStatus := enum.ContractPaymentStatusPending.String()
,
			   tc.contract_number,
			   tc.type,
			   tc.status,
			   (tc.financial_terms ->> 'total_cost')::numeric as total_value,
			   tc.start_date,
			   tc.end_date,
			   COALESCE(paid.amount, 0)    as paid_amount,
			   COALESCE(pending.amount, 0) as pending_amount,
			   COALESCE(cmp.cnt, 0)        as campaign_count
		FROM target_contracts tc
				 LEFT JOIN paid_payments paid ON paid.contract_id = tc.id
				 LEFT JOIN pending_payments pending ON pending.contract_id = tc.id
				 LEFT JOIN campaign_counts cmp ON cmp.contract_id = tc.id
		ORDER BY tc.start_date DESC
		LIMIT ?;
	`

	// Execute the raw query using GORM
	// Parameter order must match the ? in the query:
	// 1. brandUserID
	// 2. paidStatus
	// 3. pendingStatus
	// 4. limit
	if err := r.db.WithContext(ctx).Raw(query, brandUserID, paidStatus, pendingStatus, limit).Scan(&results).Error; err != nil {
		zap.L().Error("Failed to get brand contract details", zap.Error(err))
		return nil, err
	}

	return results, nil
}

func (r *brandPartnerAnalyticsRepository) GetBrandTopRatingProduct(ctx context.Context, brandUserID uuid.UUID, limit int, startDate, endDate *time.Time) ([]dtos.BrandProductRating, error) {
	var results []dtos.BrandProductRating

	query := r.db.
		WithContext(ctx).
		Table("products p").
		Select(`
			p.id AS product_id,
			p.name AS product_name,
			p.type,
			p.average_rating
		`).
		Joins("JOIN brands b ON p.brand_id = b.id").
		Where("b.user_id = ?", brandUserID).
		Where("p.deleted_at IS NULL")

	// Optional startDate
	if startDate != nil {
		query = query.Where("p.created_at >= ?", *startDate)
	}

	// Optional endDate
	if endDate != nil {
		query = query.Where("p.created_at <= ?", *endDate)
	}

	// Optional limit (recommended)
	if limit > 0 {
		query = query.Limit(limit)
	}

	err := query.
		Order("p.average_rating DESC").
		Scan(&results).Error

	if err != nil {
		return nil, err
	}

	return results, nil
}

func (r *brandPartnerAnalyticsRepository) GetBrandTopSoldProduct(ctx context.Context, brandUserID uuid.UUID, limit int, startDate, endDate *time.Time) ([]dtos.BrandTopSoldProducts, error) {
	var results []dtos.BrandTopSoldProducts
	orderStatus := enum.OrderStatusReceived.String()

	query := r.db.
		WithContext(ctx).
		Table("order_items oi").
		Select(`
			p.id AS product_id,
			p.name AS product_name,
			COALESCE(SUM(oi.quantity), 0) AS units_sold,
			COALESCE(SUM(oi.subtotal), 0) AS total_revenue
		`).
		Joins("JOIN product_variants pv ON pv.id = oi.variant_id").
		Joins("JOIN products p ON p.id = pv.product_id").
		Joins("JOIN brands b ON b.id = p.brand_id").
		Joins("JOIN orders o ON o.id = oi.order_id").
		Where("b.user_id = ?", brandUserID).
		Where("o.status = ?", orderStatus).
		Where("p.deleted_at IS NULL")

	// Optional startDate
	if startDate != nil {
		query = query.Where("p.created_at >= ?", *startDate)
	}
	// Optional endDate
	if endDate != nil {
		query = query.Where("p.created_at <= ?", *endDate)
	}

	// Group
	query = query.Group("p.id, p.name").
		Order("total_sold DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	if err := query.Scan(&results).Error; err != nil {
		return nil, err
	}

	return results, nil
}
