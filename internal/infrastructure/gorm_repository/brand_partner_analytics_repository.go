package gormrepository

import (
	"context"
	"core-backend/internal/application/dto/dtos"
	"core-backend/internal/application/dto/requests"
	dtoResponses "core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/domain/constant"
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
		query = query.Where("o.created_at < ?", *endDate)
	}

	if err := query.Scan(&revenue).Error; err != nil {
		zap.L().Error("Failed to get brand total revenue", zap.Error(err))
		return 0, err
	}
	return revenue, nil
}

// GetBrandTotalPayments returns total payments received from contracts
// Includes PAID + KOL_REFUND_APPROVED payments (net of refunds), filtered by paid_at
func (r *brandPartnerAnalyticsRepository) GetBrandTotalPayments(ctx context.Context, brandUserID uuid.UUID, startDate, endDate *time.Time) (float64, error) {
	var payments float64

	paidStatus := enum.ContractPaymentStatusPaid.String()
	kolRefundApprovedStatus := enum.ContractPaymentStatusKOLRefundApproved.String()

	// Net payment = PAID.amount + (KOL_REFUND_APPROVED.amount - refund_amount)
	query := r.db.WithContext(ctx).Table("contract_payments cp").
		Select(`COALESCE(SUM(
			CASE 
				WHEN cp.status = ? THEN cp.amount
				WHEN cp.status = ? THEN cp.amount - COALESCE(cp.refund_amount, 0)
				ELSE 0
			END
		), 0)`, paidStatus, kolRefundApprovedStatus).
		Joins("JOIN contracts c ON c.id = cp.contract_id").
		Joins("JOIN brands b ON b.id = c.brand_id").
		Where("b.user_id = ?", brandUserID).
		Where("cp.status IN (?, ?)", paidStatus, kolRefundApprovedStatus).
		Where("cp.paid_at IS NOT NULL").
		Where("cp.deleted_at IS NULL")

	if startDate != nil {
		query = query.Where("cp.paid_at >= ?", *startDate)
	}
	if endDate != nil {
		query = query.Where("cp.paid_at < ?", *endDate)
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
		query = query.Where("o.created_at < ?", *endDate)
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
		query = query.Where("(o.created_at < ? OR o.created_at IS NULL)", *endDate)
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
		query = query.Where("cmp.start_date < ?", *endDate)
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
		query = query.Where("ct.created_at < ?", *endDate)
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
		query = query.Where("o.created_at < ?", *endDate)
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

	// Build click events subquery with optional date filtering
	clickEventSubquery := `
		SELECT affiliate_link_id, COUNT(*) as click_count 
		FROM click_events 
		WHERE 1=1
	`
	var clickEventArgs []any
	if startDate != nil {
		clickEventSubquery += " AND clicked_at >= ?"
		clickEventArgs = append(clickEventArgs, *startDate)
	}
	if endDate != nil {
		clickEventSubquery += " AND clicked_at < ?"
		clickEventArgs = append(clickEventArgs, *endDate)
	}
	clickEventSubquery += " GROUP BY affiliate_link_id"

	// Get affiliate link stats
	query := r.db.WithContext(ctx).Table("affiliate_links al").
		Select(`
			COUNT(DISTINCT al.id) as total_links,
			COUNT(DISTINCT CASE WHEN al.status = ? THEN al.id END) as active_links,
			COALESCE(SUM(ce.click_count), 0) as total_clicks
		`, enum.AffiliateLinkStatusActive.String()).
		Joins("JOIN contracts c ON c.id = al.contract_id").
		Joins("LEFT JOIN ("+clickEventSubquery+") ce ON ce.affiliate_link_id = al.id", clickEventArgs...).
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
// paid_amount includes PAID + KOL_REFUND_APPROVED (net of refunds)
func (r *brandPartnerAnalyticsRepository) GetBrandContractDetails(ctx context.Context, brandUserID uuid.UUID, limit int) ([]dtos.BrandContractDetails, error) {
	var results []dtos.BrandContractDetails

	paidStatus := enum.ContractPaymentStatusPaid.String()
	kolRefundApprovedStatus := enum.ContractPaymentStatusKOLRefundApproved.String()
	pendingStatus := enum.ContractPaymentStatusPending.String()

	query := `
		WITH target_contracts AS (
			-- 1. First, find ONLY the contracts for this user.
			SELECT c.id, c.contract_number, c.type, c.status, c.financial_terms, c.start_date, c.end_date
			FROM contracts c
			JOIN brands b ON b.id = c.brand_id
			WHERE b.user_id = $1 
			  AND c.deleted_at IS NULL
		),
		paid_payments AS (
			-- 2. Sum payments (PAID + KOL_REFUND_APPROVED net of refunds) for target contracts
			SELECT cp.contract_id, 
				   SUM(
					   CASE 
						   WHEN cp.status = $2 THEN cp.amount
						   WHEN cp.status = $3 THEN cp.amount - COALESCE(cp.refund_amount, 0)
						   ELSE 0
					   END
				   ) as amount
			FROM contract_payments cp
			JOIN target_contracts tc ON tc.id = cp.contract_id
			WHERE cp.status IN ($2, $3) 
			  AND cp.paid_at IS NOT NULL
			  AND cp.deleted_at IS NULL
			GROUP BY cp.contract_id
		),
		pending_payments AS (
			-- 3. Only sum pending for the contracts found above
			SELECT cp.contract_id, SUM(cp.amount) as amount
			FROM contract_payments cp
			JOIN target_contracts tc ON tc.id = cp.contract_id
			WHERE cp.status = $4 AND cp.deleted_at IS NULL
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
		SELECT tc.id as contract_id,
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
		LIMIT $5;
	`

	// Execute the raw query using GORM
	// Parameter order: brandUserID, paidStatus, kolRefundApprovedStatus, pendingStatus, limit
	if err := r.db.WithContext(ctx).Raw(query, brandUserID, paidStatus, kolRefundApprovedStatus, pendingStatus, limit).Scan(&results).Error; err != nil {
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
		query = query.Where("p.created_at < ?", *endDate)
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
		Where("p.deleted_at IS NULL").
		Where("o.deleted_at IS NULL")

	// Optional startDate - filter by order date, not product creation date
	if startDate != nil {
		query = query.Where("o.created_at >= ?", *startDate)
	}
	// Optional endDate - filter by order date, not product creation date
	if endDate != nil {
		query = query.Where("o.created_at < ?", *endDate)
	}

	// Group
	query = query.Group("p.id, p.name").
		Order("units_sold DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	if err := query.Scan(&results).Error; err != nil {
		return nil, err
	}

	return results, nil
}

// GetBrandContractStatusDistribution returns contract status counts for a brand within the filter period
func (r *brandPartnerAnalyticsRepository) GetBrandContractStatusDistribution(ctx context.Context, brandUserID uuid.UUID, filter *requests.DashboardFilterRequest) (*dtoResponses.ContractStatusDistributionResponse, error) {
	current, _ := filter.GetDateRanges()

	query := `
		SELECT 
			COUNT(CASE WHEN c.status = 'DRAFT' THEN 1 END) as draft,
			COUNT(CASE WHEN c.status = 'ACTIVE' THEN 1 END) as active,
			COUNT(CASE WHEN c.status = 'COMPLETED' THEN 1 END) as completed,
			COUNT(CASE WHEN c.status = 'TERMINATED' THEN 1 END) as terminated,
			COUNT(CASE WHEN c.status IN ('BRAND_VIOLATED', 'BRAND_PENALTY_PENDING', 'BRAND_PENALTY_PAID') THEN 1 END) as brand_violations,
			COUNT(CASE WHEN c.status IN ('KOL_VIOLATED', 'KOL_REFUND_PENDING', 'KOL_PROOF_SUBMITTED', 'KOL_PROOF_REJECTED', 'KOL_REFUND_APPROVED') THEN 1 END) as kol_violations,
			COUNT(*) as total
		FROM contracts c
		JOIN brands b ON b.id = c.brand_id
		WHERE b.user_id = $1
		  AND c.deleted_at IS NULL
		  AND c.created_at >= $2 
		  AND c.created_at < $3
	`

	var result dtoResponses.ContractStatusDistributionResponse
	err := r.db.WithContext(ctx).Raw(query, brandUserID, current.Start, current.End).Scan(&result).Error
	if err != nil {
		zap.L().Error("Failed to get brand contract status distribution", zap.Error(err))
		return nil, err
	}

	result.Period = filter.GetPeriodInfo()
	return &result, nil
}

// GetBrandTaskStatusDistribution returns task status counts for a brand within the filter period
func (r *brandPartnerAnalyticsRepository) GetBrandTaskStatusDistribution(ctx context.Context, brandUserID uuid.UUID, filter *requests.DashboardFilterRequest) (*dtoResponses.TaskStatusDistributionResponse, error) {
	current, _ := filter.GetDateRanges()

	query := `
		SELECT 
			COUNT(CASE WHEN t.status = 'TO_DO' THEN 1 END) as todo,
			COUNT(CASE WHEN t.status = 'IN_PROGRESS' THEN 1 END) as in_progress,
			COUNT(CASE WHEN t.status = 'DONE' THEN 1 END) as done,
			COUNT(CASE WHEN t.status = 'CANCELLED' THEN 1 END) as cancelled
		FROM tasks t
		JOIN milestones m ON m.id = t.milestone_id
		JOIN campaigns cmp ON cmp.id = m.campaign_id
		JOIN contracts c ON c.id = cmp.contract_id
		JOIN brands b ON b.id = c.brand_id
		WHERE b.user_id = $1
		  AND t.deleted_at IS NULL
		  AND t.created_at >= $2 
		  AND t.created_at < $3
	`

	var result dtoResponses.TaskStatusDistributionResponse
	err := r.db.WithContext(ctx).Raw(query, brandUserID, current.Start, current.End).Scan(&result).Error
	if err != nil {
		zap.L().Error("Failed to get brand task status distribution", zap.Error(err))
		return nil, err
	}

	result.Period = filter.GetPeriodInfo()
	return &result, nil
}

// GetBrandLimitedProductRevenueOverTime returns limited product revenue grouped by time periods for a brand
func (r *brandPartnerAnalyticsRepository) GetBrandLimitedProductRevenueOverTime(ctx context.Context, brandUserID uuid.UUID, filter *requests.DashboardFilterRequest, granularity constant.TrendGranularity) ([]dtoResponses.BrandRevenueOverTimePoint, error) {
	current, _ := filter.GetDateRanges()
	interval := granularity.GetPostgreSQLInterval()

	limitedProductType := string(enum.ProductTypeLimited)
	receivedOrderStatus := string(enum.OrderStatusReceived)

	query := `
		SELECT 
			date_trunc($1, o.created_at) as date,
			COALESCE(SUM(oi.subtotal), 0) as brand_limited_revenue
		FROM orders o
		JOIN order_items oi ON oi.order_id = o.id
		JOIN product_variants pv ON pv.id = oi.variant_id
		JOIN products p ON p.id = pv.product_id
		JOIN brands b ON b.id = p.brand_id
		WHERE b.user_id = $2
		  AND p.type = $3
		  AND o.status = $4
		  AND o.deleted_at IS NULL
		  AND o.created_at >= $5 
		  AND o.created_at < $6
		GROUP BY date_trunc($1, o.created_at)
		ORDER BY date_trunc($1, o.created_at)
	`

	var results []dtoResponses.BrandRevenueOverTimePoint
	err := r.db.WithContext(ctx).Raw(query, interval, brandUserID, limitedProductType, receivedOrderStatus, current.Start, current.End).Scan(&results).Error
	if err != nil {
		zap.L().Error("Failed to get brand limited product revenue over time", zap.Error(err))
		return nil, err
	}

	return results, nil
}

// GetBrandRefundViolationStats returns refund and contract violation stats for a brand
func (r *brandPartnerAnalyticsRepository) GetBrandRefundViolationStats(ctx context.Context, brandUserID uuid.UUID, filter *requests.DashboardFilterRequest) (*dtoResponses.RefundViolationStatsResponse, error) {
	current, _ := filter.GetDateRanges()

	coProducingType := string(enum.ContractTypeCoProduce)

	query := `
		WITH brand_violations AS (
			SELECT 
				COUNT(CASE WHEN c.status = 'BRAND_PENALTY_PENDING' THEN 1 END) as pending,
				COUNT(CASE WHEN c.status = 'BRAND_PENALTY_PAID' THEN 1 END) as paid
			FROM contracts c
			JOIN brands b ON b.id = c.brand_id
			WHERE b.user_id = $1
			  AND c.deleted_at IS NULL
			  AND c.updated_at >= $2 AND c.updated_at < $3
		),
		kol_violations AS (
			SELECT 
				COUNT(CASE WHEN c.status IN ('KOL_VIOLATED', 'KOL_REFUND_PENDING', 'KOL_PROOF_SUBMITTED') THEN 1 END) as pending,
				COUNT(CASE WHEN c.status = 'KOL_REFUND_APPROVED' THEN 1 END) as resolved
			FROM contracts c
			JOIN brands b ON b.id = c.brand_id
			WHERE b.user_id = $1
			  AND c.deleted_at IS NULL
			  AND c.updated_at >= $2 AND c.updated_at < $3
		),
		co_producing_refunds AS (
			SELECT 
				COUNT(CASE WHEN cp.status IN ('KOL_PENDING', 'KOL_PROOF_SUBMITTED') AND COALESCE(cp.refund_amount, 0) > 0 THEN 1 END) as pending,
				COUNT(CASE WHEN cp.status = 'KOL_REFUND_APPROVED' AND COALESCE(cp.refund_amount, 0) > 0 THEN 1 END) as approved,
				COALESCE(SUM(CASE WHEN cp.status IN ('KOL_PENDING', 'KOL_PROOF_SUBMITTED') THEN cp.refund_amount END), 0) as pending_amount,
				COALESCE(SUM(CASE WHEN cp.status = 'KOL_REFUND_APPROVED' THEN cp.refund_amount END), 0) as approved_amount
			FROM contract_payments cp
			JOIN contracts c ON c.id = cp.contract_id
			JOIN brands b ON b.id = c.brand_id
			WHERE b.user_id = $1
			  AND c.type = $4
			  AND cp.deleted_at IS NULL
			  AND cp.updated_at >= $2 AND cp.updated_at < $3
		)
		SELECT 
			bv.pending as brand_violations_pending,
			bv.paid as brand_violations_paid,
			kv.pending as kol_violations_pending,
			kv.resolved as kol_violations_resolved,
			cpr.pending as co_producing_refunds_pending,
			cpr.approved as co_producing_refunds_approved,
			cpr.pending_amount as co_producing_amount_pending,
			cpr.approved_amount as co_producing_amount_paid,
			(bv.pending + bv.paid + kv.pending + kv.resolved) as total_violation_count,
			(cpr.pending_amount + cpr.approved_amount) as total_refund_amount
		FROM brand_violations bv, kol_violations kv, co_producing_refunds cpr
	`

	var result dtoResponses.RefundViolationStatsResponse
	err := r.db.WithContext(ctx).Raw(query, brandUserID, current.Start, current.End, coProducingType).Scan(&result).Error
	if err != nil {
		zap.L().Error("Failed to get brand refund violation stats", zap.Error(err))
		return nil, err
	}

	result.Period = filter.GetPeriodInfo()
	return &result, nil
}

// GetBrandGrossIncome returns brand's gross income from direct revenue sources
// Gross Income = Order Revenue (LIMITED products) + Pre-order Revenue + KOL Refunds (payments + violations)
// This approach uses actual revenue amounts without percentage-based calculations
func (r *brandPartnerAnalyticsRepository) GetBrandGrossIncome(ctx context.Context, brandUserID uuid.UUID, filter *requests.DashboardFilterRequest) (*dtoResponses.BrandIncomeResponse, error) {
	current, previous := filter.GetDateRanges()

	kolRefundApprovedStatus := string(enum.ContractPaymentStatusKOLRefundApproved)
	kolViolationType := string(enum.ViolationTypeKOL)
	limitedProductType := string(enum.ProductTypeLimited)

	query := `
		WITH brand_data AS (
			SELECT id FROM brands WHERE user_id = $1
		),
		-- Current period: Order revenue (LIMITED products only)
		order_revenue_current AS (
			SELECT COALESCE(SUM(oi.subtotal), 0) as total
			FROM orders o
			JOIN order_items oi ON oi.order_id = o.id
			JOIN product_variants pv ON pv.id = oi.variant_id
			JOIN products p ON p.id = pv.product_id
			JOIN brand_data b ON b.id = p.brand_id
			WHERE p.type = $6
			  AND o.status = ANY($7)
			  AND o.deleted_at IS NULL
			  AND o.created_at >= $2 AND o.created_at < $3
		),
		-- Current period: Pre-order revenue
		preorder_revenue_current AS (
			SELECT COALESCE(SUM(po.total_amount), 0) as total
			FROM pre_orders po
			JOIN product_variants pv ON pv.id = po.variant_id
			JOIN products p ON p.id = pv.product_id
			JOIN brand_data b ON b.id = p.brand_id
			WHERE po.status = ANY($8)
			  AND po.deleted_at IS NULL
			  AND po.created_at >= $2 AND po.created_at < $3
		),
		-- Current period: Refunds from KOL_REFUND_APPROVED contract payments
		payment_refund_current AS (
			SELECT COALESCE(SUM(cp.refund_amount), 0) as total
			FROM contract_payments cp
			JOIN contracts c ON c.id = cp.contract_id
			JOIN brand_data b ON b.id = c.brand_id
			WHERE cp.status = $9
			  AND cp.paid_at IS NOT NULL
			  AND cp.deleted_at IS NULL
			  AND cp.paid_at >= $2 AND cp.paid_at < $3
		),
		-- Current period: Refunds from resolved KOL contract violations
		violation_refund_current AS (
			SELECT COALESCE(SUM(cv.refund_amount), 0) as total
			FROM contract_violations cv
			JOIN contracts c ON c.id = cv.contract_id
			JOIN brand_data b ON b.id = c.brand_id
			WHERE cv.type = $10
			  AND cv.resolved_at IS NOT NULL
			  AND cv.deleted_at IS NULL
			  AND cv.resolved_at >= $2 AND cv.resolved_at < $3
		),
		-- Previous period: Order revenue
		order_revenue_previous AS (
			SELECT COALESCE(SUM(oi.subtotal), 0) as total
			FROM orders o
			JOIN order_items oi ON oi.order_id = o.id
			JOIN product_variants pv ON pv.id = oi.variant_id
			JOIN products p ON p.id = pv.product_id
			JOIN brand_data b ON b.id = p.brand_id
			WHERE p.type = $6
			  AND o.status = ANY($7)
			  AND o.deleted_at IS NULL
			  AND o.created_at >= $4 AND o.created_at < $5
		),
		-- Previous period: Pre-order revenue
		preorder_revenue_previous AS (
			SELECT COALESCE(SUM(po.total_amount), 0) as total
			FROM pre_orders po
			JOIN product_variants pv ON pv.id = po.variant_id
			JOIN products p ON p.id = pv.product_id
			JOIN brand_data b ON b.id = p.brand_id
			WHERE po.status = ANY($8)
			  AND po.deleted_at IS NULL
			  AND po.created_at >= $4 AND po.created_at < $5
		),
		-- Previous period: Payment refunds
		payment_refund_previous AS (
			SELECT COALESCE(SUM(cp.refund_amount), 0) as total
			FROM contract_payments cp
			JOIN contracts c ON c.id = cp.contract_id
			JOIN brand_data b ON b.id = c.brand_id
			WHERE cp.status = $9
			  AND cp.paid_at IS NOT NULL
			  AND cp.deleted_at IS NULL
			  AND cp.paid_at >= $4 AND cp.paid_at < $5
		),
		-- Previous period: Violation refunds
		violation_refund_previous AS (
			SELECT COALESCE(SUM(cv.refund_amount), 0) as total
			FROM contract_violations cv
			JOIN contracts c ON c.id = cv.contract_id
			JOIN brand_data b ON b.id = c.brand_id
			WHERE cv.type = $10
			  AND cv.resolved_at IS NOT NULL
			  AND cv.deleted_at IS NULL
			  AND cv.resolved_at >= $4 AND cv.resolved_at < $5
		)
		SELECT 
			orc.total as order_revenue,
			prc.total as preorder_revenue,
			pfrc.total as payment_refunds,
			vfrc.total as violation_refunds,
			orc.total + prc.total + pfrc.total + vfrc.total as gross_income,
			orp.total + prp.total + pfrp.total + vfrp.total as previous_gross_income
		FROM order_revenue_current orc,
			 preorder_revenue_current prc,
			 payment_refund_current pfrc,
			 violation_refund_current vfrc,
			 order_revenue_previous orp,
			 preorder_revenue_previous prp,
			 payment_refund_previous pfrp,
			 violation_refund_previous vfrp
	`

	type result struct {
		OrderRevenue        float64 `json:"order_revenue"`
		PreorderRevenue     float64 `json:"preorder_revenue"`
		PaymentRefunds      float64 `json:"payment_refunds"`
		ViolationRefunds    float64 `json:"violation_refunds"`
		GrossIncome         float64 `json:"gross_income"`
		PreviousGrossIncome float64 `json:"previous_gross_income"`
	}

	var res result
	validCompletedOrderStatuses := constant.ValidCompletedOrderStatus
	validCompletedPreOrderStatuses := constant.ValidCompletedPreOrderStatus
	err := r.db.WithContext(ctx).Raw(query,
		brandUserID,                // $1
		current.Start, current.End, // $2, $3
		previous.Start, previous.End, // $4, $5
		limitedProductType,                             // $6
		validCompletedOrderStatuses.ToStringSlice(),    // $7
		validCompletedPreOrderStatuses.ToStringSlice(), // $8
		kolRefundApprovedStatus,                        // $9
		kolViolationType,                               // $10
	).Scan(&res).Error
	if err != nil {
		zap.L().Error("Failed to get brand gross income", zap.Error(err))
		return nil, err
	}

	// Calculate percentage change
	var percentageChange float64
	var changeDirection string
	if res.PreviousGrossIncome > 0 {
		percentageChange = ((res.GrossIncome - res.PreviousGrossIncome) / res.PreviousGrossIncome) * 100
	} else if res.GrossIncome > 0 {
		percentageChange = 100
	}

	if res.GrossIncome > res.PreviousGrossIncome {
		changeDirection = "up"
	} else if res.GrossIncome < res.PreviousGrossIncome {
		changeDirection = "down"
	} else {
		changeDirection = "unchanged"
	}

	return &dtoResponses.BrandIncomeResponse{
		GrossIncome:         res.GrossIncome,
		OrderRevenue:        res.OrderRevenue,
		PreorderRevenue:     res.PreorderRevenue,
		PaymentRefunds:      res.PaymentRefunds,
		ViolationRefunds:    res.ViolationRefunds,
		Period:              filter.GetPeriodInfo(),
		PreviousGrossIncome: res.PreviousGrossIncome,
		PercentageChange:    percentageChange,
		ChangeDirection:     changeDirection,
	}, nil
}

// GetBrandNetIncome returns brand's net income (gross income - paid contract payments)
// Net Income = Gross Income - Total Paid Contract Payments
// Gross Income = Order Revenue (LIMITED) + Pre-order Revenue + KOL Payment Refunds + KOL Violation Refunds
// This approach uses actual revenue amounts without percentage-based calculations
func (r *brandPartnerAnalyticsRepository) GetBrandNetIncome(ctx context.Context, brandUserID uuid.UUID, filter *requests.DashboardFilterRequest) (*dtoResponses.BrandNetIncomeResponse, error) {
	current, previous := filter.GetDateRanges()

	paidPaymentStatus := string(enum.ContractPaymentStatusPaid)
	kolRefundApprovedStatus := string(enum.ContractPaymentStatusKOLRefundApproved)
	kolViolationType := string(enum.ViolationTypeKOL)
	limitedProductType := string(enum.ProductTypeLimited)

	query := `
		WITH brand_data AS (
			SELECT id FROM brands WHERE user_id = $1
		),
		-- Current period: Order revenue (LIMITED products only)
		order_revenue_current AS (
			SELECT COALESCE(SUM(oi.subtotal), 0) as total
			FROM orders o
			JOIN order_items oi ON oi.order_id = o.id
			JOIN product_variants pv ON pv.id = oi.variant_id
			JOIN products p ON p.id = pv.product_id
			JOIN brand_data b ON b.id = p.brand_id
			WHERE p.type = $6
			  AND o.status = ANY($7)
			  AND o.deleted_at IS NULL
			  AND o.created_at >= $2 AND o.created_at < $3
		),
		-- Current period: Pre-order revenue
		preorder_revenue_current AS (
			SELECT COALESCE(SUM(po.total_amount), 0) as total
			FROM pre_orders po
			JOIN product_variants pv ON pv.id = po.variant_id
			JOIN products p ON p.id = pv.product_id
			JOIN brand_data b ON b.id = p.brand_id
			WHERE po.status = ANY($8)
			  AND po.deleted_at IS NULL
			  AND po.created_at >= $2 AND po.created_at < $3
		),
		-- Current period: Payment refunds (from KOL_REFUND_APPROVED)
		payment_refund_current AS (
			SELECT COALESCE(SUM(cp.refund_amount), 0) as total
			FROM contract_payments cp
			JOIN contracts c ON c.id = cp.contract_id
			JOIN brand_data b ON b.id = c.brand_id
			WHERE cp.status = $5
			  AND cp.paid_at IS NOT NULL
			  AND cp.deleted_at IS NULL
			  AND cp.paid_at >= $2 AND cp.paid_at < $3
		),
		-- Current period: Violation refunds (from resolved KOL violations)
		violation_refund_current AS (
			SELECT COALESCE(SUM(cv.refund_amount), 0) as total
			FROM contract_violations cv
			JOIN contracts c ON c.id = cv.contract_id
			JOIN brand_data b ON b.id = c.brand_id
			WHERE cv.type = $9
			  AND cv.resolved_at IS NOT NULL
			  AND cv.deleted_at IS NULL
			  AND cv.resolved_at >= $2 AND cv.resolved_at < $3
		),
		-- Current period: Paid contract payments (net of refunds for KOL_REFUND_APPROVED)
		paid_payments_current AS (
			SELECT COALESCE(SUM(
				CASE 
					WHEN cp.status = $4 THEN cp.amount
					WHEN cp.status = $5 THEN cp.amount - COALESCE(cp.refund_amount, 0)
					ELSE 0
				END
			), 0) as total
			FROM contract_payments cp
			JOIN contracts c ON c.id = cp.contract_id
			JOIN brand_data b ON b.id = c.brand_id
			WHERE cp.status IN ($4, $5)
			  AND cp.paid_at IS NOT NULL
			  AND cp.deleted_at IS NULL
			  AND cp.paid_at >= $2 AND cp.paid_at < $3
		),
		-- Previous period: Order revenue
		order_revenue_previous AS (
			SELECT COALESCE(SUM(oi.subtotal), 0) as total
			FROM orders o
			JOIN order_items oi ON oi.order_id = o.id
			JOIN product_variants pv ON pv.id = oi.variant_id
			JOIN products p ON p.id = pv.product_id
			JOIN brand_data b ON b.id = p.brand_id
			WHERE p.type = $6
			  AND o.status = ANY($7)
			  AND o.deleted_at IS NULL
			  AND o.created_at >= $10 AND o.created_at < $11
		),
		-- Previous period: Pre-order revenue
		preorder_revenue_previous AS (
			SELECT COALESCE(SUM(po.total_amount), 0) as total
			FROM pre_orders po
			JOIN product_variants pv ON pv.id = po.variant_id
			JOIN products p ON p.id = pv.product_id
			JOIN brand_data b ON b.id = p.brand_id
			WHERE po.status = ANY($8)
			  AND po.deleted_at IS NULL
			  AND po.created_at >= $10 AND po.created_at < $11
		),
		-- Previous period: Payment refunds
		payment_refund_previous AS (
			SELECT COALESCE(SUM(cp.refund_amount), 0) as total
			FROM contract_payments cp
			JOIN contracts c ON c.id = cp.contract_id
			JOIN brand_data b ON b.id = c.brand_id
			WHERE cp.status = $5
			  AND cp.paid_at IS NOT NULL
			  AND cp.deleted_at IS NULL
			  AND cp.paid_at >= $10 AND cp.paid_at < $11
		),
		-- Previous period: Violation refunds
		violation_refund_previous AS (
			SELECT COALESCE(SUM(cv.refund_amount), 0) as total
			FROM contract_violations cv
			JOIN contracts c ON c.id = cv.contract_id
			JOIN brand_data b ON b.id = c.brand_id
			WHERE cv.type = $9
			  AND cv.resolved_at IS NOT NULL
			  AND cv.deleted_at IS NULL
			  AND cv.resolved_at >= $10 AND cv.resolved_at < $11
		),
		-- Previous period: Paid payments
		paid_payments_previous AS (
			SELECT COALESCE(SUM(
				CASE 
					WHEN cp.status = $4 THEN cp.amount
					WHEN cp.status = $5 THEN cp.amount - COALESCE(cp.refund_amount, 0)
					ELSE 0
				END
			), 0) as total
			FROM contract_payments cp
			JOIN contracts c ON c.id = cp.contract_id
			JOIN brand_data b ON b.id = c.brand_id
			WHERE cp.status IN ($4, $5)
			  AND cp.paid_at IS NOT NULL
			  AND cp.deleted_at IS NULL
			  AND cp.paid_at >= $10 AND cp.paid_at < $11
		)
		SELECT 
			orc.total as order_revenue,
			prc.total as preorder_revenue,
			pfrc.total as payment_refunds,
			vfrc.total as violation_refunds,
			orc.total + prc.total + pfrc.total + vfrc.total as gross_income,
			ppc.total as total_contract_payments,
			(orc.total + prc.total + pfrc.total + vfrc.total) - ppc.total as net_income,
			orp.total + prp.total + pfrp.total + vfrp.total as previous_gross_income,
			(orp.total + prp.total + pfrp.total + vfrp.total) - ppp.total as previous_net_income
		FROM order_revenue_current orc,
			 preorder_revenue_current prc,
			 payment_refund_current pfrc,
			 violation_refund_current vfrc,
			 paid_payments_current ppc,
			 order_revenue_previous orp,
			 preorder_revenue_previous prp,
			 payment_refund_previous pfrp,
			 violation_refund_previous vfrp,
			 paid_payments_previous ppp
	`

	type result struct {
		OrderRevenue          float64 `json:"order_revenue"`
		PreorderRevenue       float64 `json:"preorder_revenue"`
		PaymentRefunds        float64 `json:"payment_refunds"`
		ViolationRefunds      float64 `json:"violation_refunds"`
		GrossIncome           float64 `json:"gross_income"`
		TotalContractPayments float64 `json:"total_contract_payments"`
		NetIncome             float64 `json:"net_income"`
		PreviousGrossIncome   float64 `json:"previous_gross_income"`
		PreviousNetIncome     float64 `json:"previous_net_income"`
	}

	var res result
	validCompletedOrderStatuses := constant.ValidCompletedOrderStatus
	validCompletedPreOrderStatuses := constant.ValidCompletedPreOrderStatus
	err := r.db.WithContext(ctx).Raw(query,
		brandUserID,                // $1
		current.Start, current.End, // $2, $3
		paidPaymentStatus, kolRefundApprovedStatus, // $4, $5
		limitedProductType,                             // $6
		validCompletedOrderStatuses.ToStringSlice(),    // $7
		validCompletedPreOrderStatuses.ToStringSlice(), // $8
		kolViolationType,                               // $9
		previous.Start, previous.End,                   // $10, $11
	).Scan(&res).Error
	if err != nil {
		zap.L().Error("Failed to get brand net income", zap.Error(err))
		return nil, err
	}

	// Calculate percentage change
	var percentageChange float64
	var changeDirection string
	if res.PreviousNetIncome != 0 {
		percentageChange = ((res.NetIncome - res.PreviousNetIncome) / res.PreviousNetIncome) * 100
		if percentageChange < 0 {
			percentageChange = -percentageChange
		}
	} else if res.NetIncome != 0 {
		percentageChange = 100
	}

	if res.NetIncome > res.PreviousNetIncome {
		changeDirection = "up"
	} else if res.NetIncome < res.PreviousNetIncome {
		changeDirection = "down"
	} else {
		changeDirection = "unchanged"
	}

	return &dtoResponses.BrandNetIncomeResponse{
		GrossIncome:           res.GrossIncome,
		OrderRevenue:          res.OrderRevenue,
		PreorderRevenue:       res.PreorderRevenue,
		PaymentRefunds:        res.PaymentRefunds,
		ViolationRefunds:      res.ViolationRefunds,
		TotalContractPayments: res.TotalContractPayments,
		NetIncome:             res.NetIncome,
		Period:                filter.GetPeriodInfo(),
		PreviousGrossIncome:   res.PreviousGrossIncome,
		PreviousNetIncome:     res.PreviousNetIncome,
		PercentageChange:      percentageChange,
		ChangeDirection:       changeDirection,
	}, nil
}
