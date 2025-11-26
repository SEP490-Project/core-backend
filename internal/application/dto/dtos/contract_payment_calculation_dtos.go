package dtos

import (
	"time"

	"github.com/google/uuid"
)

// =============================================================================
// AFFILIATE Contract Payment Calculation DTOs
// =============================================================================

// AffiliatePaymentCalculation represents the calculated payment for an AFFILIATE contract.
// This is used internally during payment calculation and stored in CalculationBreakdown JSONB.
type AffiliatePaymentCalculation struct {
	ContractID   uuid.UUID               `json:"contract_id"`
	PeriodStart  time.Time               `json:"period_start"`
	PeriodEnd    time.Time               `json:"period_end"`
	TotalClicks  int64                   `json:"total_clicks"`
	GrossPayment int64                   `json:"gross_payment"` // Before tax deduction
	TaxAmount    int64                   `json:"tax_amount"`    // Tax withholding amount
	NetPayment   int64                   `json:"net_payment"`   // After tax deduction (final payment)
	Breakdown    []LevelPaymentBreakdown `json:"breakdown"`
	CalculatedAt time.Time               `json:"calculated_at"`
}

// LevelPaymentBreakdown shows the payment calculation for a single tier/level.
// This implements electricity-style tiered billing.
type LevelPaymentBreakdown struct {
	Level        int     `json:"level"`          // Level number (1, 2, 3...)
	ClicksInTier int64   `json:"clicks_in_tier"` // Number of clicks in this tier
	Multiplier   float32 `json:"multiplier"`     // Rate multiplier for this tier
	RatePerClick int     `json:"rate_per_click"` // base_per_click × multiplier
	TierPayment  int64   `json:"tier_payment"`   // clicks_in_tier × rate_per_click
}

// =============================================================================
// CO_PRODUCING Contract Payment Calculation DTOs
// =============================================================================

// CoProducingPaymentCalculation represents the calculated revenue distribution for a CO_PRODUCING contract.
// This is used internally during payment calculation and stored in CalculationBreakdown JSONB.
type CoProducingPaymentCalculation struct {
	ContractID       uuid.UUID                    `json:"contract_id"`
	PeriodStart      time.Time                    `json:"period_start"`
	PeriodEnd        time.Time                    `json:"period_end"`
	TotalRevenue     float64                      `json:"total_revenue"`     // Total revenue from limited products
	CompanyPercent   int                          `json:"company_percent"`   // Company's share percentage
	BrandPercent     int                          `json:"brand_percent"`     // Brand/KOL's share percentage
	CompanyShare     float64                      `json:"company_share"`     // Calculated company share amount
	BrandShare       float64                      `json:"brand_share"`       // Calculated brand share amount (this is the payment)
	RevenueBreakdown *LimitedProductRevenueBreakdown `json:"revenue_breakdown"`
	CalculatedAt     time.Time                    `json:"calculated_at"`
}

// LimitedProductRevenueBreakdown shows the revenue sources for CO_PRODUCING contracts.
type LimitedProductRevenueBreakdown struct {
	PreOrderRevenue float64 `json:"preorder_revenue"` // Revenue from pre_orders (status = RECEIVED)
	OrderRevenue    float64 `json:"order_revenue"`    // Revenue from orders (order_type = LIMITED, status = RECEIVED)
	TotalRevenue    float64 `json:"total_revenue"`    // Sum of both
}

// =============================================================================
// Payment Period Helpers
// =============================================================================

// PaymentPeriod represents a payment period with start and end times.
type PaymentPeriod struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// ContainsTime checks if a given time falls within the payment period.
// Start is inclusive, End is exclusive: [Start, End)
func (p *PaymentPeriod) ContainsTime(t time.Time) bool {
	return !t.Before(p.Start) && t.Before(p.End)
}

// IsCurrentPeriod checks if the payment period contains the current time.
func (p *PaymentPeriod) IsCurrentPeriod() bool {
	return p.ContainsTime(time.Now())
}
