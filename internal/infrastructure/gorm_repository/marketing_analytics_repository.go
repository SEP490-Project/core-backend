package gormrepository

import (
	"context"
	"core-backend/internal/application/dto/requests"
	dtoResponses "core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/domain/enum"
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

// GetActiveBrandsCount returns the count of brands with status = 'ACTIVE'
func (r *MarketingAnalyticsRepository) GetActiveBrandsCount(ctx context.Context) (int64, error) {
	var count int64

	err := r.db.WithContext(ctx).
		Model(&struct{}{}).
		Table("brands").
		Where("status = ?", "ACTIVE").
		Where("deleted_at IS NULL").
		Count(&count).Error

	if err != nil {
		zap.L().Error("Failed to get active brands count", zap.Error(err))
		return 0, err
	}

	return count, nil
}

// GetActiveCampaignsCount returns the count of campaigns with status = 'RUNNING'
func (r *MarketingAnalyticsRepository) GetActiveCampaignsCount(ctx context.Context) (int64, error) {
	var count int64

	err := r.db.WithContext(ctx).
		Model(&struct{}{}).
		Table("campaigns").
		Where("status = ?", "RUNNING").
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

// GetMonthlyContractRevenue returns sum of PAID contract payments for specified month
func (r *MarketingAnalyticsRepository) GetMonthlyContractRevenue(ctx context.Context, year, month int) (float64, error) {
	var revenue float64

	query := `
		SELECT COALESCE(SUM(amount), 0) as revenue
		FROM contract_payments
		WHERE status = 'PAID'
		  AND deleted_at IS NULL
		  AND EXTRACT(YEAR FROM due_date) = ?
		  AND EXTRACT(MONTH FROM due_date) = ?
	`

	err := r.db.WithContext(ctx).Raw(query, year, month).Scan(&revenue).Error
	if err != nil {
		zap.L().Error("Failed to get monthly contract revenue",
			zap.Int("year", year),
			zap.Int("month", month),
			zap.Error(err))
		return 0, err
	}

	return revenue, nil
}

// GetTopBrandsByRevenue returns top 4 brands by total revenue (contract + product sales)
func (r *MarketingAnalyticsRepository) GetTopBrandsByRevenue(ctx context.Context, filter *requests.TimeFilter) ([]dtoResponses.BrandRevenueResponse, error) {
	startDate, endDate, err := filter.GetDateRange()
	if err != nil {
		return nil, err
	}

	query := `
		WITH contract_revenue AS (
			SELECT 
				c.brand_id,
				COALESCE(SUM(cp.amount), 0) as revenue
			FROM contracts c
			JOIN contract_payments cp ON c.id = cp.contract_id
			WHERE cp.status = 'PAID'
			  AND cp.deleted_at IS NULL
			  AND c.deleted_at IS NULL
			  AND cp.due_date >= ?
			  AND cp.due_date <= ?
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
			WHERE p.type = 'STANDARD'
			  AND o.status = 'PAID'
			  AND o.created_at >= ?
			  AND o.created_at <= ?
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
			WHERE b.status = 'ACTIVE' AND b.deleted_at IS NULL
		)
		SELECT 
			brand_id,
			brand_name,
			total_revenue as revenue,
			ROW_NUMBER() OVER (ORDER BY total_revenue DESC) as rank
		FROM total_revenue
		WHERE total_revenue > 0
		ORDER BY total_revenue DESC
		LIMIT 4
	`

	var results []dtoResponses.BrandRevenueResponse
	err = r.db.WithContext(ctx).Raw(query, startDate, endDate, startDate, endDate).Scan(&results).Error
	if err != nil {
		zap.L().Error("Failed to get top brands by revenue",
			zap.String("filter_type", filter.FilterType),
			zap.Int("year", filter.Year),
			zap.Error(err))
		return nil, err
	}

	return results, nil
}

// GetRevenueByContractType returns revenue breakdown: 4 contract types + standard products
func (r *MarketingAnalyticsRepository) GetRevenueByContractType(ctx context.Context, filter *requests.TimeFilter) (*dtoResponses.RevenueByTypeResponse, error) {
	startDate, endDate, err := filter.GetDateRange()
	if err != nil {
		return nil, err
	}

	query := `
		WITH contract_revenue AS (
			SELECT
				c.type,
				COALESCE(SUM(cp.amount), 0) as revenue
			FROM contracts c
					 JOIN contract_payments cp ON c.id = cp.contract_id
			WHERE cp.status = 'PAID'
			  AND cp.deleted_at IS NULL
			  AND c.deleted_at IS NULL
			  AND cp.due_date >= ?
			  AND cp.due_date <= ?
			GROUP BY c.type
		),
			 standard_product_revenue AS (
				 SELECT COALESCE(SUM(o.total_amount), 0) as revenue
				 FROM product_variants pv 
					 INNER JOIN products p on pv.product_id = p.id
						  JOIN order_items oi ON p.id = oi.variant_id
						  JOIN orders o ON oi.order_id = o.id
				 WHERE p.type = 'STANDARD'
				   AND o.status = 'PAID'
				   AND o.created_at >= ?
				   AND o.created_at <= ?
			 )
		SELECT
			COALESCE(MAX(CASE WHEN type = 'ADVERTISING' THEN revenue END), 0) as advertising,
			COALESCE(MAX(CASE WHEN type = 'AFFILIATE' THEN revenue END), 0) as affiliate,
			COALESCE(MAX(CASE WHEN type = 'BRAND_AMBASSADOR' THEN revenue END), 0) as brand_ambassador,
			COALESCE(MAX(CASE WHEN type = 'CO_PRODUCING' THEN revenue END), 0) as co_produce,
			COALESCE((SELECT revenue FROM standard_product_revenue), 0) as standard_product,
			COALESCE(SUM(revenue), 0) + COALESCE((SELECT revenue FROM standard_product_revenue), 0) as total_revenue
		FROM contract_revenue
	`

	var result dtoResponses.RevenueByTypeResponse
	err = r.db.WithContext(ctx).Raw(query, startDate, endDate, startDate, endDate).Scan(&result).Error
	if err != nil {
		zap.L().Error("Failed to get revenue by contract type",
			zap.String("filter_type", filter.FilterType),
			zap.Int("year", filter.Year),
			zap.Error(err))
		return nil, err
	}

	return &result, nil
}

// GetUpcomingDeadlineCampaigns returns campaigns with end_date within X days and status = 'RUNNING'
func (r *MarketingAnalyticsRepository) GetUpcomingDeadlineCampaigns(ctx context.Context, daysBeforeDeadline int) ([]dtoResponses.UpcomingCampaignResponse, error) {
	if daysBeforeDeadline <= 0 {
		return nil, errors.New("daysBeforeDeadline must be greater than 0")
	}

	now := time.Now()
	futureDate := now.AddDate(0, 0, daysBeforeDeadline)

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
		WHERE c.status = 'RUNNING'
		  AND c.deleted_at IS NULL
		  AND ct.deleted_at IS NULL
		  AND b.deleted_at IS NULL
		  AND c.end_date >= ?
		  AND c.end_date <= ?
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

	err := r.db.WithContext(ctx).Raw(query, now, futureDate).Scan(&results).Error
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
