package gormrepository

import (
	"context"
	"core-backend/internal/application/dto/dtos"
	"core-backend/internal/application/interfaces/irepository"
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
func (r *brandPartnerAnalyticsRepository) GetBrandContractCount(ctx context.Context, brandID uuid.UUID, status *string) (int64, error) {
	var count int64
	query := r.db.WithContext(ctx).Table("contracts").
		Where("brand_id = ?", brandID).
		Where("deleted_at IS NULL")

	if status != nil && *status != "" {
		query = query.Where("status = ?", *status)
	}

	if err := query.Count(&count).Error; err != nil {
		zap.L().Error("Failed to get brand contract count", zap.Error(err))
		return 0, err
	}
	return count, nil
}

// GetBrandCampaignCount returns count of campaigns for a brand
func (r *brandPartnerAnalyticsRepository) GetBrandCampaignCount(ctx context.Context, brandID uuid.UUID, status *string) (int64, error) {
	var count int64
	query := r.db.WithContext(ctx).Table("campaigns cmp").
		Joins("JOIN contracts c ON c.id = cmp.contract_id").
		Where("c.brand_id = ?", brandID).
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
func (r *brandPartnerAnalyticsRepository) GetBrandTotalRevenue(ctx context.Context, brandID uuid.UUID, startDate, endDate *time.Time) (float64, error) {
	var revenue float64
	query := r.db.WithContext(ctx).Table("orders o").
		Select("COALESCE(SUM(oi.quantity * oi.unit_price), 0)").
		Joins("JOIN order_items oi ON oi.order_id = o.id").
		Joins("JOIN products p ON p.id = oi.product_id").
		Where("p.brand_id = ?", brandID).
		Where("o.status = ?", "PAID").
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
func (r *brandPartnerAnalyticsRepository) GetBrandTotalPayments(ctx context.Context, brandID uuid.UUID, startDate, endDate *time.Time) (float64, error) {
	var payments float64
	query := r.db.WithContext(ctx).Table("contract_payments cp").
		Select("COALESCE(SUM(cp.amount), 0)").
		Joins("JOIN contracts c ON c.id = cp.contract_id").
		Where("c.brand_id = ?", brandID).
		Where("cp.status = ?", "PAID").
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
func (r *brandPartnerAnalyticsRepository) GetBrandPendingPayments(ctx context.Context, brandID uuid.UUID) (float64, error) {
	var pending float64
	query := r.db.WithContext(ctx).Table("contract_payments cp").
		Select("COALESCE(SUM(cp.amount), 0)").
		Joins("JOIN contracts c ON c.id = cp.contract_id").
		Where("c.brand_id = ?", brandID).
		Where("cp.status = ?", "PENDING").
		Where("cp.deleted_at IS NULL")

	if err := query.Scan(&pending).Error; err != nil {
		zap.L().Error("Failed to get brand pending payments", zap.Error(err))
		return 0, err
	}
	return pending, nil
}

// GetBrandProductCount returns count of products for a brand
func (r *brandPartnerAnalyticsRepository) GetBrandProductCount(ctx context.Context, brandID uuid.UUID, status *string) (int64, error) {
	var count int64
	query := r.db.WithContext(ctx).Table("products").
		Where("brand_id = ?", brandID).
		Where("deleted_at IS NULL")

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
func (r *brandPartnerAnalyticsRepository) GetBrandOrderCount(ctx context.Context, brandID uuid.UUID, status *string, startDate, endDate *time.Time) (int64, error) {
	var count int64
	query := r.db.WithContext(ctx).Table("orders o").
		Select("COUNT(DISTINCT o.id)").
		Joins("JOIN order_items oi ON oi.order_id = o.id").
		Joins("JOIN products p ON p.id = oi.product_id").
		Where("p.brand_id = ?", brandID).
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
func (r *brandPartnerAnalyticsRepository) GetBrandTopProducts(ctx context.Context, brandID uuid.UUID, limit int, startDate, endDate *time.Time) ([]dtos.BrandProductMetrics, error) {
	var results []dtos.BrandProductMetrics

	query := r.db.WithContext(ctx).Table("products p").
		Select(`
			p.id as product_id,
			p.name as product_name,
			p.type as product_type,
			p.status,
			COUNT(DISTINCT o.id) as order_count,
			COALESCE(SUM(oi.quantity), 0) as units_sold,
			COALESCE(SUM(oi.quantity * oi.unit_price), 0) as revenue
		`).
		Joins("LEFT JOIN order_items oi ON oi.product_id = p.id").
		Joins("LEFT JOIN orders o ON o.id = oi.order_id AND o.status = 'PAID'").
		Where("p.brand_id = ?", brandID).
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
func (r *brandPartnerAnalyticsRepository) GetBrandCampaignMetrics(ctx context.Context, brandID uuid.UUID, limit int, startDate, endDate *time.Time) ([]dtos.BrandCampaignMetrics, error) {
	var results []dtos.BrandCampaignMetrics

	query := r.db.WithContext(ctx).Table("campaigns cmp").
		Select(`
			cmp.id as campaign_id,
			cmp.name as campaign_name,
			cmp.status,
			cmp.start_date,
			cmp.end_date,
			COUNT(DISTINCT m.id) as milestone_count,
			COUNT(DISTINCT t.id) as task_count,
			SUM(CASE WHEN t.status = 'DONE' THEN 1 ELSE 0 END) as completed_tasks,
			COUNT(DISTINCT ct.id) as content_count,
			COALESCE(SUM(cc.views), 0) as total_views,
			COALESCE(SUM(cc.likes) + SUM(cc.comments) + SUM(cc.shares), 0) as total_engagements
		`).
		Joins("JOIN contracts c ON c.id = cmp.contract_id").
		Joins("LEFT JOIN milestones m ON m.campaign_id = cmp.id").
		Joins("LEFT JOIN tasks t ON t.milestone_id = m.id").
		Joins("LEFT JOIN contents ct ON ct.milestone_id = m.id").
		Joins("LEFT JOIN content_channels cc ON cc.content_id = ct.id").
		Where("c.brand_id = ?", brandID).
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
func (r *brandPartnerAnalyticsRepository) GetBrandContentMetrics(ctx context.Context, brandID uuid.UUID, startDate, endDate *time.Time) (*dtos.BrandContentMetrics, error) {
	var result dtos.BrandContentMetrics

	query := r.db.WithContext(ctx).Table("contents ct").
		Select(`
			COUNT(DISTINCT ct.id) as total_content,
			SUM(CASE WHEN ct.status = 'POSTED' THEN 1 ELSE 0 END) as posted_content,
			COALESCE(SUM(cc.views), 0) as total_views,
			COALESCE(SUM(cc.likes), 0) as total_likes,
			COALESCE(SUM(cc.comments), 0) as total_comments,
			COALESCE(SUM(cc.shares), 0) as total_shares
		`).
		Joins("JOIN milestones m ON m.id = ct.milestone_id").
		Joins("JOIN campaigns cmp ON cmp.id = m.campaign_id").
		Joins("JOIN contracts c ON c.id = cmp.contract_id").
		Joins("LEFT JOIN content_channels cc ON cc.content_id = ct.id").
		Where("c.brand_id = ?", brandID).
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
func (r *brandPartnerAnalyticsRepository) GetBrandRevenueTrend(ctx context.Context, brandID uuid.UUID, granularity string, startDate, endDate *time.Time) ([]dtos.BrandRevenueTrendResult, error) {
	var results []dtos.BrandRevenueTrendResult

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
			COUNT(DISTINCT o.id) as order_count,
			COALESCE(SUM(oi.quantity), 0) as units_sold,
			COALESCE(SUM(oi.quantity * oi.unit_price), 0) as revenue
		`).
		Joins("JOIN order_items oi ON oi.order_id = o.id").
		Joins("JOIN products p ON p.id = oi.product_id").
		Where("p.brand_id = ?", brandID).
		Where("o.status = ?", "PAID").
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
func (r *brandPartnerAnalyticsRepository) GetBrandAffiliateMetrics(ctx context.Context, brandID uuid.UUID, startDate, endDate *time.Time) (*dtos.BrandAffiliateMetrics, error) {
	var result dtos.BrandAffiliateMetrics

	// Get affiliate link stats
	query := r.db.WithContext(ctx).Table("affiliate_links al").
		Select(`
			COUNT(DISTINCT al.id) as total_links,
			COUNT(DISTINCT CASE WHEN al.is_active = true THEN al.id END) as active_links,
			COALESCE(SUM(ce.click_count), 0) as total_clicks
		`).
		Joins("JOIN contracts c ON c.id = al.contract_id").
		Joins(`LEFT JOIN (
			SELECT affiliate_link_id, COUNT(*) as click_count 
			FROM click_events 
			GROUP BY affiliate_link_id
		) ce ON ce.affiliate_link_id = al.id`).
		Where("c.brand_id = ?", brandID).
		Where("al.deleted_at IS NULL")

	if err := query.Scan(&result).Error; err != nil {
		zap.L().Error("Failed to get brand affiliate metrics", zap.Error(err))
		return nil, err
	}

	return &result, nil
}

// GetBrandContractDetails returns contract details for a brand
func (r *brandPartnerAnalyticsRepository) GetBrandContractDetails(ctx context.Context, brandID uuid.UUID, limit int) ([]dtos.BrandContractDetails, error) {
	var results []dtos.BrandContractDetails

	query := r.db.WithContext(ctx).Table("contracts c").
		Select(`
			c.id as contract_id,
			c.contract_number,
			c.type,
			c.status,
			c.total_value,
			c.start_date,
			c.end_date,
			COALESCE(paid.amount, 0) as paid_amount,
			COALESCE(pending.amount, 0) as pending_amount,
			COUNT(DISTINCT cmp.id) as campaign_count
		`).
		Joins(`LEFT JOIN (
			SELECT contract_id, SUM(amount) as amount 
			FROM contract_payments 
			WHERE status = 'PAID' AND deleted_at IS NULL
			GROUP BY contract_id
		) paid ON paid.contract_id = c.id`).
		Joins(`LEFT JOIN (
			SELECT contract_id, SUM(amount) as amount 
			FROM contract_payments 
			WHERE status = 'PENDING' AND deleted_at IS NULL
			GROUP BY contract_id
		) pending ON pending.contract_id = c.id`).
		Joins("LEFT JOIN campaigns cmp ON cmp.contract_id = c.id AND cmp.deleted_at IS NULL").
		Where("c.brand_id = ?", brandID).
		Where("c.deleted_at IS NULL").
		Group("c.id, c.contract_number, c.type, c.status, c.total_value, c.start_date, c.end_date, paid.amount, pending.amount").
		Order("c.start_date DESC").
		Limit(limit)

	if err := query.Scan(&results).Error; err != nil {
		zap.L().Error("Failed to get brand contract details", zap.Error(err))
		return nil, err
	}
	return results, nil
}
