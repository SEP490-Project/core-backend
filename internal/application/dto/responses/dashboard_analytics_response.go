package responses

import (
	"core-backend/internal/application/dto/requests"
	"time"
)

// ContractStatusDistributionResponse represents contract counts grouped by status categories
type ContractStatusDistributionResponse struct {
	Draft           int64               `json:"draft" gorm:"column:draft"`
	Active          int64               `json:"active" gorm:"column:active"`
	Completed       int64               `json:"completed" gorm:"column:completed"`
	Terminated      int64               `json:"terminated" gorm:"column:terminated"`
	BrandViolations int64               `json:"brand_violations" gorm:"column:brand_violations"`
	KOLViolations   int64               `json:"kol_violations" gorm:"column:kol_violations"`
	Total           int64               `json:"total" gorm:"column:total"`
	Period          requests.PeriodInfo `json:"period" gorm:"-"`
}

// TaskStatusDistributionResponse represents task counts grouped by status
type TaskStatusDistributionResponse struct {
	ToDo       int64               `json:"todo" gorm:"column:todo"`
	InProgress int64               `json:"in_progress" gorm:"column:in_progress"`
	Done       int64               `json:"done" gorm:"column:done"`
	Cancelled  int64               `json:"cancelled" gorm:"column:cancelled"`
	Total      int64               `json:"total" gorm:"column:total"`
	Period     requests.PeriodInfo `json:"period" gorm:"-"`
}

// RevenueOverTimePoint represents a single data point in the revenue time series
type RevenueOverTimePoint struct {
	Date                  time.Time `json:"date"`
	ContractBaseRevenue   float64   `json:"contract_base_revenue"`
	AffiliateRevenue      float64   `json:"affiliate_revenue"`
	LimitedProductRevenue float64   `json:"limited_product_revenue"`
	TotalRevenue          float64   `json:"total_revenue"`
}

// RevenueOverTimeSummary provides totals for the entire period
type RevenueOverTimeSummary struct {
	TotalContractBaseRevenue   float64 `json:"total_contract_base_revenue"`
	TotalAffiliateRevenue      float64 `json:"total_affiliate_revenue"`
	TotalLimitedProductRevenue float64 `json:"total_limited_product_revenue"`
	GrandTotalRevenue          float64 `json:"grand_total_revenue"`
}

// RevenueOverTimeResponse represents the full revenue over time chart data
type RevenueOverTimeResponse struct {
	Data        []RevenueOverTimePoint `json:"data"`
	Granularity string                 `json:"granularity"`
	Period      requests.PeriodInfo    `json:"period" gorm:"-"`
	Summary     RevenueOverTimeSummary `json:"summary"`
}

// ContractRevenueBreakdownPoint represents a single data point for the combo chart
type ContractRevenueBreakdownPoint struct {
	Date                      time.Time `json:"date"`
	ContractBaseCost          float64   `json:"contract_base_cost"`           // Line 1: Base cost from contract payments
	AffiliateRevenue          float64   `json:"affiliate_revenue"`            // Line 2: Tiered click revenue
	LimitedProductBrandShare  float64   `json:"limited_product_brand_share"`  // Line 3: Brand's share (company_percent)
	LimitedProductSystemShare float64   `json:"limited_product_system_share"` // Line 4: System's share (kol_percent)
	TotalContractRevenue      float64   `json:"total_contract_revenue"`       // Bar: Total paid to system
}

// ContractRevenueBreakdownSummary provides totals for the entire period
type ContractRevenueBreakdownSummary struct {
	TotalContractBaseCost          float64 `json:"total_contract_base_cost"`
	TotalAffiliateRevenue          float64 `json:"total_affiliate_revenue"`
	TotalLimitedProductBrandShare  float64 `json:"total_limited_product_brand_share"`  // Brand's total share
	TotalLimitedProductSystemShare float64 `json:"total_limited_product_system_share"` // System's total share
	GrandTotalRevenue              float64 `json:"grand_total_revenue"`
	RefundsPaid                    float64 `json:"refunds_paid"` // CO_PRODUCING refunds (subtracted)
}

// ContractRevenueBreakdownResponse represents the full breakdown for combo chart
type ContractRevenueBreakdownResponse struct {
	Data        []ContractRevenueBreakdownPoint `json:"data"`
	Granularity string                          `json:"granularity"`
	Period      requests.PeriodInfo             `json:"period" gorm:"-"`
	Summary     ContractRevenueBreakdownSummary `json:"summary"`
}

// AffiliateClicksPeriod represents click data for a period (used for tiered calculation)
type AffiliateClicksPeriod struct {
	Date           time.Time `json:"date"`
	ContractID     string    `json:"contract_id"`
	ClickCount     int64     `json:"click_count"`
	FinancialTerms string    `json:"financial_terms"` // JSON string for tiered calculation
}

// LimitedProductSharePeriod represents limited product revenue with brand/system shares
type LimitedProductSharePeriod struct {
	Date         time.Time `json:"date"`
	GrossRevenue float64   `json:"gross_revenue"`
	BrandShare   float64   `json:"brand_share"`  // company_percent share
	SystemShare  float64   `json:"system_share"` // kol_percent share
}

// RefundViolationStatsResponse represents system-wide refund and violation statistics
type RefundViolationStatsResponse struct {
	// Brand Violations (brand pays penalty)
	BrandViolationsPending       int64   `json:"brand_violations_pending" gorm:"column:brand_violations_pending"`
	BrandViolationsPendingAmount float64 `json:"brand_violations_pending_amount" gorm:"column:brand_violations_pending_amount"`
	BrandViolationsPaid          int64   `json:"brand_violations_paid" gorm:"column:brand_violations_paid"`
	BrandViolationsPaidAmount    float64 `json:"brand_violations_paid_amount" gorm:"column:brand_violations_paid_amount"`
	// BrandPenaltyAmountOwed       float64 `json:"brand_penalty_amount_owed" gorm:"column:brand_penalty_amount_owed"`
	// BrandPenaltyAmountPaid       float64 `json:"brand_penalty_amount_paid" gorm:"column:brand_penalty_amount_paid"`

	// KOL Violations (KOL pays refund to brand)
	KOLViolationsPending        int64   `json:"kol_violations_pending" gorm:"column:kol_violations_pending"`
	KOLViolationsPendingAmount  float64 `json:"kol_violations_pending_amount" gorm:"column:kol_violations_pending_amount"`
	KOLViolationsResolved       int64   `json:"kol_violations_resolved" gorm:"column:kol_violations_resolved"`
	KOLViolationsResolvedAmount float64 `json:"kol_violations_resolved_amount" gorm:"column:kol_violations_resolved_amount"`
	CompensationPending         float64 `json:"compensation_pending"`
	CompensationPaid            float64 `json:"compensation_paid"`

	// CO_PRODUCING Refunds (system owes brand when NetAmount < 0)
	CoProducingRefundsPending  int64   `json:"co_producing_refunds_pending"`
	CoProducingRefundsApproved int64   `json:"co_producing_refunds_approved"`
	CoProducingAmountPending   float64 `json:"co_producing_amount_pending"`
	CoProducingAmountPaid      float64 `json:"co_producing_amount_paid"`

	// Totals
	TotalViolationCount int64               `json:"total_violation_count"`
	TotalRefundAmount   float64             `json:"total_refund_amount"`
	Period              requests.PeriodInfo `json:"period" gorm:"-"`
}

// BrandRevenueOverTimePoint represents a single data point in brand's revenue time series
type BrandRevenueOverTimePoint struct {
	Date                   time.Time `json:"date"`
	BrandLimitedRevenue    float64   `json:"brand_limited_revenue"`
	BrandAffiliateEarnings float64   `json:"brand_affiliate_earnings"`
	TotalRevenue           float64   `json:"total_revenue"`
}

// BrandRevenueOverTimeSummary provides totals for brand's revenue in the period
type BrandRevenueOverTimeSummary struct {
	TotalBrandLimitedRevenue    float64 `json:"total_brand_limited_revenue"`
	TotalBrandAffiliateEarnings float64 `json:"total_brand_affiliate_earnings"`
	GrandTotalRevenue           float64 `json:"grand_total_revenue"`
}

// BrandRevenueOverTimeResponse represents brand partner's revenue over time chart data
type BrandRevenueOverTimeResponse struct {
	Data        []BrandRevenueOverTimePoint `json:"data"`
	Granularity string                      `json:"granularity"`
	Period      requests.PeriodInfo         `json:"period" gorm:"-"`
	Summary     BrandRevenueOverTimeSummary `json:"summary"`
}

// BrandIncomeResponse represents the brand's income (gross or net)
type BrandIncomeResponse struct {
	GrossIncome         float64             `json:"gross_income"`
	OrderRevenue        float64             `json:"order_revenue"`     // Revenue from completed orders (LIMITED products)
	PreorderRevenue     float64             `json:"preorder_revenue"`  // Revenue from completed pre-orders
	PaymentRefunds      float64             `json:"payment_refunds"`   // Refunds from KOL_REFUND_APPROVED payments
	ViolationRefunds    float64             `json:"violation_refunds"` // Refunds from resolved KOL violations
	Period              requests.PeriodInfo `json:"period" gorm:"-"`
	PreviousGrossIncome float64             `json:"previous_gross_income,omitempty"`
	PercentageChange    float64             `json:"percentage_change,omitempty"`
	ChangeDirection     string              `json:"change_direction,omitempty"` // "up", "down", "unchanged"
}

// BrandNetIncomeResponse provides detailed breakdown of net income
type BrandNetIncomeResponse struct {
	GrossIncome           float64             `json:"gross_income"`
	OrderRevenue          float64             `json:"order_revenue"`           // Revenue from completed orders (LIMITED products)
	PreorderRevenue       float64             `json:"preorder_revenue"`        // Revenue from completed pre-orders
	PaymentRefunds        float64             `json:"payment_refunds"`         // Refunds from KOL_REFUND_APPROVED payments
	ViolationRefunds      float64             `json:"violation_refunds"`       // Refunds from resolved KOL violations
	TotalContractPayments float64             `json:"total_contract_payments"` // Total paid contract payments (deducted)
	NetIncome             float64             `json:"net_income"`
	Period                requests.PeriodInfo `json:"period" gorm:"-"`
	PreviousNetIncome     float64             `json:"previous_net_income,omitempty"`
	PreviousGrossIncome   float64             `json:"previous_gross_income,omitempty"`
	PercentageChange      float64             `json:"percentage_change,omitempty"`
	ChangeDirection       string              `json:"change_direction,omitempty"` // "up", "down", "unchanged"
}
