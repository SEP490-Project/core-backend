package gormrepository

import (
	"context"
	"core-backend/internal/application/dto/requests"
	dtoResponses "core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/domain/constant"
	"core-backend/internal/domain/enum"
	"core-backend/pkg/utils"
	"errors"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

type MarketingAnalyticsRepository struct {
	db *gorm.DB
}

// NewMarketingAnalyticsRepository creates a new marketing analytics repository
func NewMarketingAnalyticsRepository(db *gorm.DB) irepository.MarketingAnalyticsRepository {
	return &MarketingAnalyticsRepository{db: db}
}

// GetActiveBrandsCount returns the count of brands with status = ACTIVE
func (r *MarketingAnalyticsRepository) GetActiveBrandsCount(ctx context.Context) (int64, error) {
	var count int64

	err := r.db.WithContext(ctx).
		Model(&struct{}{}).
		Table("brands").
		Where("status = ?", string(enum.BrandStatusActive)).
		Where("deleted_at IS NULL").
		Count(&count).Error

	if err != nil {
		zap.L().Error("Failed to get active brands count", zap.Error(err))
		return 0, err
	}

	return count, nil
}

// GetActiveCampaignsCount returns the count of campaigns with status = RUNNING
func (r *MarketingAnalyticsRepository) GetActiveCampaignsCount(ctx context.Context) (int64, error) {
	var count int64

	err := r.db.WithContext(ctx).
		Model(&struct{}{}).
		Table("campaigns").
		Where("status = ?", enum.CampaignRunning.String()).
		Where("deleted_at IS NULL").
		Count(&count).Error

	if err != nil {
		zap.L().Error("Failed to get active campaigns count", zap.Error(err))
		return 0, err
	}

	return count, nil
}

// GetDraftCampaignsCount returns the count of campaigns with status = 'DRAFT' AND contract_id IS NOT NULL
func (r *MarketingAnalyticsRepository) GetDraftCampaignsCount(ctx context.Context) (int64, error) {
	var count int64

	err := r.db.WithContext(ctx).
		Model(&struct{}{}).
		Table("campaigns").
		Where("status = ?", enum.CampaignDraft).
		Where("contract_id IS NOT NULL").
		Where("deleted_at IS NULL").
		Count(&count).Error

	if err != nil {
		zap.L().Error("Failed to get draft campaigns count", zap.Error(err))
		return 0, err
	}

	return count, nil
}

// GetGrossContractRevenue returns sum of contract payment amounts (before refunds) for specified period
// Includes PAID and KOL_REFUND_APPROVED statuses, uses paid_at for accurate timing, BRAND_VIOLATIION penalty are included
func (r *MarketingAnalyticsRepository) GetGrossContractRevenue(ctx context.Context, filter *requests.DashboardFilterRequest) (float64, error) {
	var revenue float64

	current, _ := filter.GetDateRanges()

	query := `
		WITH brand_violation_penalty AS (
				SELECT COALESCE(SUM(penalty_amount), 0) as total_penalty_amount
				FROM contract_violations
				WHERE type = ?
				  AND deleted_at IS NULL
				  AND resolved_at >= ?
				  AND resolved_at < ?
		), gross_contract_payments_revenue AS (
			SELECT COALESCE(SUM(amount), 0) as revenue
			FROM contract_payments
			WHERE status IN ?
			  AND deleted_at IS NULL
			  AND paid_at IS NOT NULL
			  AND paid_at >= ?
			  AND paid_at < ?
		)
		SELECT COALESCE(gross.revenue, 0) + COALESCE(penalty.total_penalty_amount, 0) as revenue
		FROM gross_contract_payments_revenue gross, brand_violation_penalty penalty
	`
	args := []any{
		enum.ViolationTypeBrand,
		current.Start, current.End,
		[]enum.ContractPaymentStatus{enum.ContractPaymentStatusPaid, enum.ContractPaymentStatusKOLRefundApproved},
		current.Start, current.End,
	}

	err := r.db.WithContext(ctx).Raw(query, args...).Scan(&revenue).Error
	if err != nil {
		zap.L().Error("Failed to get gross contract revenue",
			zap.String("period", filter.GetPresetLabel()),
			zap.Error(err))
		return 0, err
	}

	return revenue, nil
}

// GetNetContractRevenue returns net contract revenue (gross - refunds) for specified period
// Includes PAID and KOL_REFUND_APPROVED statuses, subtracts refund_amount for KOL_REFUND_APPROVED
func (r *MarketingAnalyticsRepository) GetNetContractRevenue(ctx context.Context, filter *requests.DashboardFilterRequest) (grossRevenue float64, netRevenue float64, totalRefunds float64, err error) {
	var (
		refundAmount float64
		current, _   = filter.GetDateRanges()
	)

	grossFunc := func(ctx context.Context) error {
		var tempErr error
		grossRevenue, tempErr = r.GetGrossContractRevenue(ctx, filter)
		return tempErr
	}
	refundedFunc := func(ctx context.Context) error {
		query := `
			SELECT COALESCE(SUM(refund_amount), 0) as total_refund_amount
			FROM contract_violations
			WHERE type = ?
			  AND deleted_at IS NULL
			  AND resolved_at >= ?
			  AND resolved_at < ?
		`
		args := []any{
			enum.ViolationTypeKOL,
			current.Start, current.End,
		}
		return r.db.WithContext(ctx).Raw(query, args...).Scan(&refundAmount).Error
	}
	utils.RunParallel(ctx, 2, grossFunc, refundedFunc)

	return grossRevenue, grossRevenue - refundAmount, refundAmount, nil
}

// GetTopBrandsByRevenue returns top brands by total revenue (contract + product sales)
// Includes KOL_REFUND_APPROVED status, subtracts refund_amount, uses paid_at for contract payments
func (r *MarketingAnalyticsRepository) GetTopBrandsByRevenue(ctx context.Context, filter *requests.DashboardFilterRequest) ([]dtoResponses.BrandRevenueResponse, error) {
	current, _ := filter.GetDateRanges()
	limit := filter.GetLimit()

	// Use enum values for status filtering
	paidPaymentStatus := enum.ContractPaymentStatusPaid.String()
	kolRefundApprovedStatus := enum.ContractPaymentStatusKOLRefundApproved.String()
	paidOrderStatus := enum.OrderStatusPaid.String()
	standardProductType := string(enum.ProductTypeStandard)
	activeBrandStatus := string(enum.BrandStatusActive)

	query := `
		WITH contract_revenue AS (
			SELECT 
				c.brand_id,
				COALESCE(SUM(
					CASE 
						WHEN cp.status = $1 THEN cp.amount
						WHEN cp.status = $2 THEN cp.amount - COALESCE(cp.refund_amount, 0)
						ELSE 0
					END
				), 0) as revenue
			FROM contracts c
			JOIN contract_payments cp ON c.id = cp.contract_id
			WHERE cp.status IN ($1, $2)
			  AND cp.deleted_at IS NULL
			  AND c.deleted_at IS NULL
			  AND cp.paid_at IS NOT NULL
			  AND cp.paid_at >= $3
			  AND cp.paid_at < $4
			GROUP BY c.brand_id
		),
		product_revenue AS (
			SELECT 
				p.brand_id,
				COALESCE(SUM(o.total_amount), 0) as revenue
			from product_variants pv
				inner join products p on pv.product_id = p.id
				JOIN order_items oi ON p.id = oi.variant_id
				JOIN orders o ON oi.order_id = o.id
			WHERE p.type = $5
			  AND o.status = $6
			  AND o.created_at >= $3
			  AND o.created_at < $4
			GROUP BY p.brand_id
		),
		total_revenue AS (
			SELECT 
				b.id as brand_id,
				b.name as brand_name,
				COALESCE(cr.revenue, 0) + COALESCE(pr.revenue, 0) as total_revenue
			FROM brands b
			LEFT JOIN contract_revenue cr ON b.id = cr.brand_id
			LEFT JOIN product_revenue pr ON b.id = pr.brand_id
			WHERE b.status = $7 AND b.deleted_at IS NULL
		)
		SELECT 
			brand_id,
			brand_name,
			total_revenue as revenue,
			ROW_NUMBER() OVER (ORDER BY total_revenue DESC) as rank
		FROM total_revenue
		WHERE total_revenue > 0
		ORDER BY total_revenue DESC
		LIMIT $8
	`

	var results []dtoResponses.BrandRevenueResponse
	err := r.db.WithContext(ctx).Raw(query, paidPaymentStatus, kolRefundApprovedStatus, current.Start, current.End, standardProductType, paidOrderStatus, activeBrandStatus, limit).Scan(&results).Error
	if err != nil {
		zap.L().Error("Failed to get top brands by revenue",
			zap.String("period", filter.GetPresetLabel()),
			zap.Error(err))
		return nil, err
	}

	return results, nil
}

// GetRevenueByContractType returns revenue breakdown: 4 contract types + standard products
// Includes KOL_REFUND_APPROVED status, subtracts refund_amount, uses paid_at for contract payments
func (r *MarketingAnalyticsRepository) GetRevenueByContractType(ctx context.Context, filter *requests.DashboardFilterRequest) (*dtoResponses.RevenueByTypeResponse, error) {
	current, _ := filter.GetDateRanges()

	// Use enum values for status and type filtering
	paidPaymentStatus := enum.ContractPaymentStatusPaid.String()
	kolRefundApprovedStatus := enum.ContractPaymentStatusKOLRefundApproved.String()
	paidOrderStatus := enum.OrderStatusPaid.String()
	standardProductType := string(enum.ProductTypeStandard)
	advertisingType := string(enum.ContractTypeAdvertising)
	affiliateType := string(enum.ContractTypeAffiliate)
	brandAmbassadorType := string(enum.ContractTypeAmbassador)
	coProducingType := string(enum.ContractTypeCoProduce)

	query := `
		WITH contract_revenue AS (
			SELECT
				c.type,
				COALESCE(SUM(
					CASE 
						WHEN cp.status = $1 THEN cp.amount
						WHEN cp.status = $2 THEN cp.amount - COALESCE(cp.refund_amount, 0)
						ELSE 0
					END
				), 0) as revenue
			FROM contracts c
					 JOIN contract_payments cp ON c.id = cp.contract_id
			WHERE cp.status IN ($1, $2)
			  AND cp.deleted_at IS NULL
			  AND c.deleted_at IS NULL
			  AND cp.paid_at IS NOT NULL
			  AND cp.paid_at >= $3
			  AND cp.paid_at < $4
			GROUP BY c.type
		),
			 standard_product_revenue AS (
				 SELECT COALESCE(SUM(o.total_amount), 0) as revenue
				 FROM product_variants pv 
					 INNER JOIN products p on pv.product_id = p.id
						  JOIN order_items oi ON p.id = oi.variant_id
						  JOIN orders o ON oi.order_id = o.id
				 WHERE p.type = $5
				   AND o.status = $6
				   AND o.created_at >= $3
				   AND o.created_at < $4
			 )
		SELECT
			COALESCE(MAX(CASE WHEN type = $7 THEN revenue END), 0) as advertising,
			COALESCE(MAX(CASE WHEN type = $8 THEN revenue END), 0) as affiliate,
			COALESCE(MAX(CASE WHEN type = $9 THEN revenue END), 0) as brand_ambassador,
			COALESCE(MAX(CASE WHEN type = $10 THEN revenue END), 0) as co_produce,
			COALESCE((SELECT revenue FROM standard_product_revenue), 0) as standard_product,
			COALESCE(SUM(revenue), 0) + COALESCE((SELECT revenue FROM standard_product_revenue), 0) as total_revenue
		FROM contract_revenue
	`

	var result dtoResponses.RevenueByTypeResponse
	err := r.db.WithContext(ctx).Raw(query, paidPaymentStatus, kolRefundApprovedStatus, current.Start, current.End, standardProductType, paidOrderStatus, advertisingType, affiliateType, brandAmbassadorType, coProducingType).Scan(&result).Error
	if err != nil {
		zap.L().Error("Failed to get revenue by contract type",
			zap.String("period", filter.GetPresetLabel()),
			zap.Error(err))
		return nil, err
	}

	return &result, nil
}

// GetUpcomingDeadlineCampaigns returns campaigns with end_date within X days and status = RUNNING
func (r *MarketingAnalyticsRepository) GetUpcomingDeadlineCampaigns(ctx context.Context, daysBeforeDeadline int) ([]dtoResponses.UpcomingCampaignResponse, error) {
	if daysBeforeDeadline <= 0 {
		return nil, errors.New("daysBeforeDeadline must be greater than 0")
	}

	now := time.Now()
	futureDate := now.AddDate(0, 0, daysBeforeDeadline)
	runningStatus := enum.CampaignRunning.String()

	query := `
		SELECT 
			c.id as campaign_id,
			c.name,
			c.end_date,
			EXTRACT(DAY FROM (c.end_date - NOW())) as days_remaining,
			c.contract_id,
			b.name as brand_name
		FROM campaigns c
		JOIN contracts ct ON c.contract_id = ct.id
		JOIN brands b ON ct.brand_id = b.id
		WHERE c.status = $1
		  AND c.deleted_at IS NULL
		  AND ct.deleted_at IS NULL
		  AND b.deleted_at IS NULL
		  AND c.end_date >= $2
		  AND c.end_date <= $3
		ORDER BY c.end_date ASC
	`

	var results []struct {
		CampaignID    string    `gorm:"column:campaign_id"`
		Name          string    `gorm:"column:name"`
		EndDate       time.Time `gorm:"column:end_date"`
		DaysRemaining int       `gorm:"column:days_remaining"`
		ContractID    string    `gorm:"column:contract_id"`
		BrandName     string    `gorm:"column:brand_name"`
	}

	err := r.db.WithContext(ctx).Raw(query, runningStatus, now, futureDate).Scan(&results).Error
	if err != nil {
		zap.L().Error("Failed to get upcoming deadline campaigns",
			zap.Int("days_before_deadline", daysBeforeDeadline),
			zap.Error(err))
		return nil, err
	}

	// Convert to response format
	campaignResponses := make([]dtoResponses.UpcomingCampaignResponse, len(results))
	for i, result := range results {
		campaignResponses[i] = dtoResponses.UpcomingCampaignResponse{
			CampaignID:    result.CampaignID,
			Name:          result.Name,
			EndDate:       result.EndDate.Format("2006-01-02 15:04:05"),
			DaysRemaining: result.DaysRemaining,
			ContractID:    result.ContractID,
			BrandName:     result.BrandName,
		}
	}

	return campaignResponses, nil
}

// GetContractStatusDistribution returns contract counts grouped by status categories
func (r *MarketingAnalyticsRepository) GetContractStatusDistribution(ctx context.Context, filter *requests.DashboardFilterRequest) (*dtoResponses.ContractStatusDistributionResponse, error) {
	current, _ := filter.GetDateRanges()

	query := `
		SELECT 
			COUNT(CASE WHEN c.status = ? THEN 1 END) as draft,
			COUNT(CASE WHEN c.status = ? THEN 1 END) as active,
			COUNT(CASE WHEN c.status = ? THEN 1 END) as completed,
			COUNT(CASE WHEN c.status = ? THEN 1 END) as terminated,
			COUNT(CASE WHEN c.status IN ? AND cv.type = ? THEN 1 END) as brand_violations,
			COUNT(CASE WHEN c.status IN ? AND cv.type = ? THEN 1 END) as kol_violations,
			COUNT(*) as total
		FROM contracts c
			LEFT JOIN contract_violations cv ON c.id = cv.contract_id AND cv.deleted_at IS NULL
		WHERE c.deleted_at IS NULL
		  AND c.created_at >= ?
		  AND c.created_at < ?
	`

	var result dtoResponses.ContractStatusDistributionResponse
	err := r.db.WithContext(ctx).Raw(query,
		enum.ContractStatusDraft,
		enum.ContractStatusActive,
		enum.ContractStatusCompleted,
		enum.ContractStatusTerminated,
		[]enum.ContractStatus{enum.ContractStatusBrandViolated, enum.ContractStatusBrandPenaltyPending, enum.ContractStatusBrandPenaltyPaid, enum.ContractStatusTerminated},
		enum.ViolationTypeBrand,
		[]enum.ContractStatus{enum.ContractStatusKOLViolated, enum.ContractStatusKOLRefundPending, enum.ContractStatusKOLProofSubmitted, enum.ContractStatusKOLProofRejected, enum.ContractStatusKOLRefundApproved, enum.ContractStatusTerminated},
		enum.ViolationTypeKOL,
		current.Start, current.End,
	).Scan(&result).Error
	if err != nil {
		zap.L().Error("Failed to get contract status distribution", zap.Error(err))
		return nil, err
	}

	result.Period = filter.GetPeriodInfo()
	return &result, nil
}

// GetTaskStatusDistribution returns task counts grouped by status
func (r *MarketingAnalyticsRepository) GetTaskStatusDistribution(ctx context.Context, filter *requests.DashboardFilterRequest) (*dtoResponses.TaskStatusDistributionResponse, error) {
	current, _ := filter.GetDateRanges()

	query := `
		SELECT 
			COUNT(CASE WHEN t.status = $1 THEN 1 END) as todo,
			COUNT(CASE WHEN t.status = $2 THEN 1 END) as in_progress,
			COUNT(CASE WHEN t.status = $3 THEN 1 END) as done,
			COUNT(CASE WHEN t.status = $4 THEN 1 END) as cancelled,
			COUNT(*) as total
		FROM tasks t
		JOIN milestones m ON m.id = t.milestone_id
		JOIN campaigns c ON c.id = m.campaign_id
		WHERE t.deleted_at IS NULL
		  AND t.created_at >= $5 
		  AND t.created_at < $6
	`

	var result dtoResponses.TaskStatusDistributionResponse
	err := r.db.WithContext(ctx).Raw(query,
		enum.TaskStatusToDo, enum.TaskStatusInProgress, enum.TaskStatusDone, enum.TaskStatusCancelled,
		current.Start, current.End,
	).Scan(&result).Error
	if err != nil {
		zap.L().Error("Failed to get task status distribution", zap.Error(err))
		return nil, err
	}

	result.Period = filter.GetPeriodInfo()
	return &result, nil
}

// GetContractBaseRevenueOverTime returns contract base revenue grouped by time periods
// Includes KOL_REFUND_APPROVED status, subtracts refund_amount, uses paid_at for accurate timing
func (r *MarketingAnalyticsRepository) GetContractBaseRevenueOverTime(ctx context.Context, filter *requests.DashboardFilterRequest, granularity constant.TrendGranularity) ([]dtoResponses.RevenueOverTimePoint, error) {
	current, _ := filter.GetDateRanges()
	interval := granularity.GetPostgreSQLInterval()

	paidStatus := enum.ContractPaymentStatusPaid.String()
	kolRefundApprovedStatus := enum.ContractPaymentStatusKOLRefundApproved.String()

	query := `
		SELECT 
			date_trunc($1, cp.paid_at) as date,
			COALESCE(SUM(cp.base_amount), 0) as contract_base_revenue,
			COALESCE(SUM(
				CASE 
					WHEN cp.status = $2 THEN cp.amount
					WHEN cp.status = $3 THEN cp.amount - COALESCE(cp.refund_amount, 0)
					ELSE 0
				END
			), 0) as net_revenue
		FROM contract_payments cp
		WHERE cp.status IN ($2, $3)
		  AND cp.deleted_at IS NULL
		  AND cp.paid_at IS NOT NULL
		  AND cp.paid_at >= $4 
		  AND cp.paid_at < $5
		GROUP BY date_trunc($1, cp.paid_at)
		ORDER BY date_trunc($1, cp.paid_at)
	`

	var results []dtoResponses.RevenueOverTimePoint
	err := r.db.WithContext(ctx).Raw(query, interval, paidStatus, kolRefundApprovedStatus, current.Start, current.End).Scan(&results).Error
	if err != nil {
		zap.L().Error("Failed to get contract base revenue over time", zap.Error(err))
		return nil, err
	}

	return results, nil
}

// GetLimitedProductRevenueOverTime returns limited product revenue grouped by time periods
func (r *MarketingAnalyticsRepository) GetLimitedProductRevenueOverTime(ctx context.Context, filter *requests.DashboardFilterRequest, granularity constant.TrendGranularity) ([]dtoResponses.RevenueOverTimePoint, error) {
	current, _ := filter.GetDateRanges()
	interval := granularity.GetPostgreSQLInterval()

	limitedProductType := string(enum.ProductTypeLimited)
	receivedOrderStatus := string(enum.OrderStatusReceived)

	query := `
		SELECT 
			date_trunc($1, o.created_at) as date,
			COALESCE(SUM(oi.subtotal), 0) as limited_product_revenue
		FROM orders o
		JOIN order_items oi ON oi.order_id = o.id
		JOIN product_variants pv ON pv.id = oi.variant_id
		JOIN products p ON p.id = pv.product_id
		WHERE p.type = $2
		  AND o.status = $3
		  AND o.deleted_at IS NULL
		  AND o.created_at >= $4 
		  AND o.created_at < $5
		GROUP BY date_trunc($1, o.created_at)
		ORDER BY date_trunc($1, o.created_at)
	`

	var results []dtoResponses.RevenueOverTimePoint
	err := r.db.WithContext(ctx).Raw(query, interval, limitedProductType, receivedOrderStatus, current.Start, current.End).Scan(&results).Error
	if err != nil {
		zap.L().Error("Failed to get limited product revenue over time", zap.Error(err))
		return nil, err
	}

	return results, nil
}

// GetRefundViolationStats returns system-wide refund and violation statistics
func (r *MarketingAnalyticsRepository) GetRefundViolationStats(ctx context.Context, filter *requests.DashboardFilterRequest) (*dtoResponses.RefundViolationStatsResponse, error) {
	current, _ := filter.GetDateRanges()

	coProducingType := string(enum.ContractTypeCoProduce)

	query := `
		WITH brand_violations AS (
			SELECT 
				COUNT(CASE WHEN c.status = $4 THEN 1 END) as pending,
        		-- SUM(CASE WHEN c.status = $4 THEN cv.penalty_amount END) as pending_amount,
				COUNT(CASE WHEN (c.status = $5) OR (cv.type = $10) THEN 1 END) as paid
        		-- SUM(CASE WHEN (c.status = $5) OR (cv.type = $10 AND cv.proof_status = $11) THEN cv.penalty_amount END) as paid_amount
			FROM contracts c
			LEFT JOIN contract_violations cv on c.id = cv.contract_id
			WHERE c.deleted_at IS NULL
			AND (c.created_at >= $1 AND c.created_at < $2 
				OR c.updated_at >= $1 AND c.updated_at < $2)
		),
		kol_violations AS (
			SELECT 
				COUNT(CASE WHEN c.status = ANY($6) THEN 1 END) as pending,
        		-- SUM(CASE WHEN c.status = ANY($6) THEN cv.refund_amount END) as pending_amount,
				COUNT(CASE WHEN (c.status = $7) OR (cv.type = $12 AND cv.proof_status = $11) THEN 1 END) as resolved
				-- SUM(CASE WHEN (c.status = $7) OR (cv.type = $12 AND cv.proof_status = $11) THEN cv.refund_amount END) as resolved_amount
			FROM contracts c
			LEFT JOIN contract_violations cv on c.id = cv.contract_id
			WHERE c.deleted_at IS NULL
			AND (c.created_at >= $1 AND c.created_at < $2 
				OR c.updated_at >= $1 AND c.updated_at < $2)
		),
		co_producing_refunds AS (
			SELECT 
				COUNT(CASE WHEN cp.status = ANY($8) AND COALESCE(cp.refund_amount, 0) > 0 THEN 1 END) as pending,
				COUNT(CASE WHEN cp.status = $9 AND COALESCE(cp.refund_amount, 0) > 0 THEN 1 END) as approved,
				COALESCE(SUM(CASE WHEN cp.status = ANY($8) THEN cp.refund_amount END), 0) as pending_amount,
				COALESCE(SUM(CASE WHEN cp.status = $9 THEN cp.refund_amount END), 0) as approved_amount
			FROM contract_payments cp
			JOIN contracts c ON c.id = cp.contract_id
			WHERE c.type = $3
			  AND cp.deleted_at IS NULL
			  AND (c.created_at >= $1 AND c.created_at < $2 
				OR c.updated_at >= $1 AND c.updated_at < $2)
		)
		SELECT 
			bv.pending as brand_violations_pending,
			-- bv.pending_amount as brand_violations_pending_amount,
			bv.paid as brand_violations_paid,
			-- bv.paid_amount as brand_violations_paid_amount,
			kv.pending as kol_violations_pending,
			-- kv.pending_amount as kol_violations_pending_amount,
			kv.resolved as kol_violations_resolved,
			-- kv.resolved_amount as kol_violations_resolved_amount,
			cpr.pending as co_producing_refunds_pending,
			cpr.approved as co_producing_refunds_approved,
			cpr.pending_amount as co_producing_amount_pending,
			cpr.approved_amount as co_producing_amount_paid,
			(bv.pending + bv.paid + kv.pending + kv.resolved) as total_violation_count,
			(cpr.pending_amount + cpr.approved_amount) as total_refund_amount
		FROM brand_violations bv, kol_violations kv, co_producing_refunds cpr
	`

	var result dtoResponses.RefundViolationStatsResponse
	err := r.db.WithContext(ctx).Raw(query,
		current.Start, current.End, // 1 - 2
		coProducingType,                        // 3
		enum.ContractStatusBrandPenaltyPending, // 4
		enum.ContractStatusBrandPenaltyPaid,    // 5
		[]string{enum.ContractStatusKOLViolated.String(), enum.ContractStatusKOLRefundPending.String(), enum.ContractStatusKOLProofSubmitted.String(), enum.ContractStatusKOLProofRejected.String()}, // 6
		enum.ContractStatusKOLRefundApproved, // 7
		[]string{enum.ContractPaymentStatusKOLPending.String(), enum.ContractPaymentStatusKOLProofSubmitted.String()}, // 8
		enum.ContractPaymentStatusKOLRefundApproved,                                                                  // 9
		enum.ViolationTypeBrand.String(), enum.ViolationProofStatusApproved.String(), enum.ViolationTypeKOL.String(), // 10 - 12
	).Scan(&result).Error
	if err != nil {
		zap.L().Error("Failed to get refund violation stats", zap.Error(err))
		return nil, err
	}

	result.Period = filter.GetPeriodInfo()
	return &result, nil
}

// GetAffiliateClicksOverTime returns click counts per contract per period for tiered calculation
func (r *MarketingAnalyticsRepository) GetAffiliateClicksOverTime(ctx context.Context, filter *requests.DashboardFilterRequest, granularity constant.TrendGranularity) ([]dtoResponses.AffiliateClicksPeriod, error) {
	current, _ := filter.GetDateRanges()
	interval := granularity.GetPostgreSQLInterval()

	affiliateType := string(enum.ContractTypeAffiliate)

	query := `
		WITH contract_clicks AS (
			SELECT 
				date_trunc($1, ce.clicked_at) as period,
				al.contract_id,
				COUNT(*) as click_count
			FROM click_events ce
			JOIN affiliate_links al ON al.id = ce.affiliate_link_id
			WHERE ce.clicked_at >= $2
			  AND ce.clicked_at < $3
			GROUP BY date_trunc($1, ce.clicked_at), al.contract_id
		)
		SELECT 
			cc.period as date,
			cc.contract_id::text as contract_id,
			cc.click_count,
			c.financial_terms::text as financial_terms
		FROM contract_clicks cc
		JOIN contracts c ON c.id = cc.contract_id
		WHERE c.type = $4
		  AND c.deleted_at IS NULL
		ORDER BY cc.period, cc.contract_id
	`

	var results []dtoResponses.AffiliateClicksPeriod
	err := r.db.WithContext(ctx).Raw(query, interval, current.Start, current.End, affiliateType).Scan(&results).Error
	if err != nil {
		zap.L().Error("Failed to get affiliate clicks over time", zap.Error(err))
		return nil, err
	}

	return results, nil
}

// GetLimitedProductRevenueWithSharesOverTime returns limited product revenue with brand/system shares
func (r *MarketingAnalyticsRepository) GetLimitedProductRevenueWithSharesOverTime(ctx context.Context, filter *requests.DashboardFilterRequest, granularity constant.TrendGranularity) ([]dtoResponses.LimitedProductSharePeriod, error) {
	current, _ := filter.GetDateRanges()
	interval := granularity.GetPostgreSQLInterval()

	limitedProductType := string(enum.ProductTypeLimited)

	// Query calculates brand share (company_percent) and system share (kol_percent)
	// For products without CO_PRODUCING contracts, assume 100% to brand (company_percent = 100)
	query := `
		WITH limited_sales AS (
			SELECT 
				date_trunc($1, o.created_at) as period,
				c.id as contract_id,
				c.financial_terms,
				SUM(oi.subtotal) as gross_revenue
			FROM orders o
			JOIN order_items oi ON oi.order_id = o.id
			JOIN product_variants pv ON pv.id = oi.variant_id
			JOIN products p ON p.id = pv.product_id
			LEFT JOIN tasks t ON t.id = p.task_id
			LEFT JOIN milestones m ON m.id = t.milestone_id
			LEFT JOIN campaigns cmp ON cmp.id = m.campaign_id
			LEFT JOIN contracts c ON c.id = cmp.contract_id
			WHERE p.type = $2
			  AND o.status IN ('RECEIVED', 'PAID')
			  AND o.deleted_at IS NULL
			  AND o.created_at >= $3 
			  AND o.created_at < $4
			GROUP BY date_trunc($1, o.created_at), c.id, c.financial_terms
		)
		SELECT 
			period as date,
			SUM(gross_revenue) as gross_revenue,
			SUM(
				CASE 
					WHEN contract_id IS NOT NULL AND financial_terms->>'profit_split_company_percent' IS NOT NULL
					THEN gross_revenue * (financial_terms->>'profit_split_company_percent')::float / 100
					ELSE gross_revenue  -- Default: 100% to brand if no contract
				END
			) as brand_share,
			SUM(
				CASE 
					WHEN contract_id IS NOT NULL AND financial_terms->>'profit_split_kol_percent' IS NOT NULL
					THEN gross_revenue * (financial_terms->>'profit_split_kol_percent')::float / 100
					ELSE 0  -- Default: 0% to system if no contract
				END
			) as system_share
		FROM limited_sales
		GROUP BY period
		ORDER BY period
	`

	var results []dtoResponses.LimitedProductSharePeriod
	err := r.db.WithContext(ctx).Raw(query, interval, limitedProductType, current.Start, current.End).Scan(&results).Error
	if err != nil {
		zap.L().Error("Failed to get limited product revenue with shares over time", zap.Error(err))
		return nil, err
	}

	return results, nil
}

// GetTotalRefundsPaid returns total refunds paid during the period
func (r *MarketingAnalyticsRepository) GetTotalRefundsPaid(ctx context.Context, filter *requests.DashboardFilterRequest) (float64, error) {
	current, _ := filter.GetDateRanges()
	kolRefundApprovedStatus := enum.ContractPaymentStatusKOLRefundApproved.String()

	query := `
		SELECT COALESCE(SUM(cp.refund_amount), 0) as total_refunds
		FROM contract_payments cp
		WHERE cp.status = $1
		  AND cp.paid_at IS NOT NULL
		  AND cp.paid_at >= $2
		  AND cp.paid_at < $3
		  AND cp.deleted_at IS NULL
	`

	var totalRefunds float64
	err := r.db.WithContext(ctx).Raw(query, kolRefundApprovedStatus, current.Start, current.End).Scan(&totalRefunds).Error
	if err != nil {
		zap.L().Error("Failed to get total refunds paid", zap.Error(err))
		return 0, err
	}

	return totalRefunds, nil
}

// GetDetailedContractRevenueBreakdown returns aggregated revenue breakdown from contract payments
func (r *MarketingAnalyticsRepository) GetDetailedContractRevenueBreakdown(ctx context.Context, filter *requests.DashboardFilterRequest, granularity constant.TrendGranularity) ([]dtoResponses.ContractRevenueBreakdownPoint, float64, error) {
	current, _ := filter.GetDateRanges()
	interval := granularity.GetPostgreSQLInterval()

	// Statuses
	paidStatus := enum.ContractPaymentStatusPaid.String()
	refundApprovedStatus := enum.ContractPaymentStatusKOLRefundApproved.String()
	affiliateType := string(enum.ContractTypeAffiliate)
	coProducingType := string(enum.ContractTypeCoProduce)

	query := `
		SELECT
			date_trunc($1, cp.paid_at) as date,
			COALESCE(SUM(cp.base_amount), 0) as contract_base_cost,
			COALESCE(SUM(
				CASE WHEN c.type = $2 THEN (cp.calculation_breakdown->>'gross_payment')::numeric ELSE 0 END
			), 0) as affiliate_revenue,
			COALESCE(SUM(
				CASE WHEN c.type = $3 THEN (cp.calculation_breakdown->>'company_share')::numeric ELSE 0 END
			), 0) as limited_product_brand_share,
			COALESCE(SUM(
				CASE WHEN c.type = $3 THEN (cp.calculation_breakdown->>'brand_share')::numeric ELSE 0 END
			), 0) as limited_product_system_share
		FROM contract_payments cp
		JOIN contracts c ON c.id = cp.contract_id
		WHERE cp.status IN ($4, $5)
		  AND cp.deleted_at IS NULL
		  AND cp.paid_at >= $6
		  AND cp.paid_at < $7
		GROUP BY date_trunc($1, cp.paid_at)
		ORDER BY date_trunc($1, cp.paid_at)
	`

	var results []dtoResponses.ContractRevenueBreakdownPoint
	err := r.db.WithContext(ctx).Raw(query,
		interval,
		affiliateType,
		coProducingType,
		paidStatus,
		refundApprovedStatus,
		current.Start,
		current.End,
	).Scan(&results).Error

	if err != nil {
		zap.L().Error("Failed to get detailed contract revenue breakdown", zap.Error(err))
		return nil, 0, err
	}

	// Calculate TotalContractRevenue for each point in Go to ensure consistency
	for i := range results {
		results[i].TotalContractRevenue = results[i].ContractBaseCost +
			results[i].AffiliateRevenue +
			results[i].LimitedProductBrandShare +
			results[i].LimitedProductSystemShare
	}

	// Calculate total refunds
	refundQuery := `
		SELECT COALESCE(SUM(cp.refund_amount), 0)
		FROM contract_payments cp
		WHERE cp.status = $1
		  AND cp.paid_at >= $2
		  AND cp.paid_at < $3
		  AND cp.deleted_at IS NULL
	`
	var totalRefunds float64
	if err := r.db.WithContext(ctx).Raw(refundQuery, refundApprovedStatus, current.Start, current.End).Scan(&totalRefunds).Error; err != nil {
		zap.L().Error("Failed to get total refunds", zap.Error(err))
		return nil, 0, err
	}

	return results, totalRefunds, nil
}
