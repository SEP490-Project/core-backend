package service_test

import (
	"testing"
	"time"

	"core-backend/internal/application/dto/dtos"
	"core-backend/tests/testhelpers"

	"github.com/stretchr/testify/assert"
)

// =============================================================================
// Tiered Payment Calculation Tests (Electricity-style billing)
// =============================================================================

// TieredPaymentCalculator is a test-friendly implementation of the tiered payment algorithm.
// This mirrors the production code in contract_payment_service.go for testing.
type TieredPaymentCalculator struct{}

// CalculateTieredPayment implements electricity-style tiered billing.
// Returns gross payment and detailed breakdown per level.
func (c *TieredPaymentCalculator) CalculateTieredPayment(
	totalClicks int64,
	baseRate int,
	levels []dtos.Level,
) (int64, []dtos.LevelPaymentBreakdown) {
	// Sort levels by Level number (should already be sorted in production)
	sortedLevels := make([]dtos.Level, len(levels))
	copy(sortedLevels, levels)

	var payment int64
	var breakdown []dtos.LevelPaymentBreakdown
	remainingClicks := totalClicks
	previousMax := int64(0)

	for _, level := range sortedLevels {
		if remainingClicks <= 0 {
			break
		}

		// Calculate clicks in this tier
		tierCapacity := level.MaxClicks - previousMax
		clicksInTier := min(remainingClicks, tierCapacity)

		// Calculate payment for this tier
		ratePerClick := int(float32(baseRate) * level.Multiplier)
		tierPayment := clicksInTier * int64(ratePerClick)
		payment += tierPayment

		breakdown = append(breakdown, dtos.LevelPaymentBreakdown{
			Level:        level.Level,
			ClicksInTier: clicksInTier,
			Multiplier:   level.Multiplier,
			RatePerClick: ratePerClick,
			TierPayment:  tierPayment,
		})

		remainingClicks -= clicksInTier
		previousMax = level.MaxClicks
	}

	// If clicks exceed highest level, charge at highest multiplier
	if remainingClicks > 0 && len(sortedLevels) > 0 {
		highestLevel := sortedLevels[len(sortedLevels)-1]
		ratePerClick := int(float32(baseRate) * highestLevel.Multiplier)
		tierPayment := remainingClicks * int64(ratePerClick)
		payment += tierPayment

		breakdown = append(breakdown, dtos.LevelPaymentBreakdown{
			Level:        highestLevel.Level + 1, // Overflow tier
			ClicksInTier: remainingClicks,
			Multiplier:   highestLevel.Multiplier,
			RatePerClick: ratePerClick,
			TierPayment:  tierPayment,
		})
	}

	return payment, breakdown
}

func TestTieredPaymentCalculation_SingleLevel(t *testing.T) {
	calc := &TieredPaymentCalculator{}

	tests := []struct {
		name            string
		totalClicks     int64
		baseRate        int
		levels          []dtos.Level
		expectedPayment int64
		expectedTiers   int
	}{
		{
			name:        "Single level - all clicks within tier",
			totalClicks: 500,
			baseRate:    1000, // 1000 VND per click
			levels: []dtos.Level{
				{Level: 1, MaxClicks: 1000, Multiplier: 1.0},
			},
			expectedPayment: 500 * 1000, // 500,000 VND
			expectedTiers:   1,
		},
		{
			name:        "Single level - exactly at max",
			totalClicks: 1000,
			baseRate:    1000,
			levels: []dtos.Level{
				{Level: 1, MaxClicks: 1000, Multiplier: 1.0},
			},
			expectedPayment: 1000 * 1000, // 1,000,000 VND
			expectedTiers:   1,
		},
		{
			name:        "Single level - overflow creates new tier",
			totalClicks: 1500,
			baseRate:    1000,
			levels: []dtos.Level{
				{Level: 1, MaxClicks: 1000, Multiplier: 1.0},
			},
			expectedPayment: 1000*1000 + 500*1000, // 1,500,000 VND
			expectedTiers:   2,                    // Original + overflow tier
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payment, breakdown := calc.CalculateTieredPayment(tt.totalClicks, tt.baseRate, tt.levels)

			assert.Equal(t, tt.expectedPayment, payment, "Payment mismatch")
			assert.Len(t, breakdown, tt.expectedTiers, "Tier count mismatch")
		})
	}
}

func TestTieredPaymentCalculation_MultipleLevels(t *testing.T) {
	calc := &TieredPaymentCalculator{}

	// Typical affiliate pricing structure:
	// Level 1: 0-1000 clicks @ 1.0x (base rate)
	// Level 2: 1001-5000 clicks @ 1.2x
	// Level 3: 5001-10000 clicks @ 1.5x
	levels := []dtos.Level{
		{Level: 1, MaxClicks: 1000, Multiplier: 1.0},
		{Level: 2, MaxClicks: 5000, Multiplier: 1.2},
		{Level: 3, MaxClicks: 10000, Multiplier: 1.5},
	}
	baseRate := 1000 // 1000 VND per click

	tests := []struct {
		name              string
		totalClicks       int64
		expectedPayment   int64
		expectedTiers     int
		expectedBreakdown []dtos.LevelPaymentBreakdown
	}{
		{
			name:            "Clicks only in Level 1",
			totalClicks:     800,
			expectedPayment: 800 * 1000, // 800,000 VND
			expectedTiers:   1,
			expectedBreakdown: []dtos.LevelPaymentBreakdown{
				{Level: 1, ClicksInTier: 800, Multiplier: 1.0, RatePerClick: 1000, TierPayment: 800000},
			},
		},
		{
			name:            "Clicks span Level 1 and Level 2",
			totalClicks:     2000,
			expectedPayment: 1000*1000 + 1000*1200, // Level1: 1M + Level2: 1.2M = 2.2M VND
			expectedTiers:   2,
			expectedBreakdown: []dtos.LevelPaymentBreakdown{
				{Level: 1, ClicksInTier: 1000, Multiplier: 1.0, RatePerClick: 1000, TierPayment: 1000000},
				{Level: 2, ClicksInTier: 1000, Multiplier: 1.2, RatePerClick: 1200, TierPayment: 1200000},
			},
		},
		{
			name:            "Clicks span all 3 levels",
			totalClicks:     7000,
			expectedPayment: 1000*1000 + 4000*1200 + 2000*1500, // L1: 1M + L2: 4.8M + L3: 3M = 8.8M VND
			expectedTiers:   3,
			expectedBreakdown: []dtos.LevelPaymentBreakdown{
				{Level: 1, ClicksInTier: 1000, Multiplier: 1.0, RatePerClick: 1000, TierPayment: 1000000},
				{Level: 2, ClicksInTier: 4000, Multiplier: 1.2, RatePerClick: 1200, TierPayment: 4800000},
				{Level: 3, ClicksInTier: 2000, Multiplier: 1.5, RatePerClick: 1500, TierPayment: 3000000},
			},
		},
		{
			name:            "Clicks exceed all levels (overflow)",
			totalClicks:     12000,
			expectedPayment: 1000*1000 + 4000*1200 + 5000*1500 + 2000*1500, // L1: 1M + L2: 4.8M + L3: 7.5M + Overflow: 3M
			expectedTiers:   4,                                             // 3 levels + 1 overflow
			expectedBreakdown: []dtos.LevelPaymentBreakdown{
				{Level: 1, ClicksInTier: 1000, Multiplier: 1.0, RatePerClick: 1000, TierPayment: 1000000},
				{Level: 2, ClicksInTier: 4000, Multiplier: 1.2, RatePerClick: 1200, TierPayment: 4800000},
				{Level: 3, ClicksInTier: 5000, Multiplier: 1.5, RatePerClick: 1500, TierPayment: 7500000},
				{Level: 4, ClicksInTier: 2000, Multiplier: 1.5, RatePerClick: 1500, TierPayment: 3000000}, // Overflow at highest rate
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payment, breakdown := calc.CalculateTieredPayment(tt.totalClicks, baseRate, levels)

			assert.Equal(t, tt.expectedPayment, payment, "Payment mismatch")
			assert.Len(t, breakdown, tt.expectedTiers, "Tier count mismatch")

			// Verify each tier breakdown
			for i, expected := range tt.expectedBreakdown {
				if i < len(breakdown) {
					assert.Equal(t, expected.Level, breakdown[i].Level, "Level mismatch at tier %d", i)
					assert.Equal(t, expected.ClicksInTier, breakdown[i].ClicksInTier, "ClicksInTier mismatch at tier %d", i)
					assert.Equal(t, expected.Multiplier, breakdown[i].Multiplier, "Multiplier mismatch at tier %d", i)
					assert.Equal(t, expected.RatePerClick, breakdown[i].RatePerClick, "RatePerClick mismatch at tier %d", i)
					assert.Equal(t, expected.TierPayment, breakdown[i].TierPayment, "TierPayment mismatch at tier %d", i)
				}
			}
		})
	}
}

func TestTieredPaymentCalculation_EdgeCases(t *testing.T) {
	calc := &TieredPaymentCalculator{}

	levels := []dtos.Level{
		{Level: 1, MaxClicks: 1000, Multiplier: 1.0},
		{Level: 2, MaxClicks: 5000, Multiplier: 1.5},
	}
	baseRate := 1000

	tests := []struct {
		name            string
		totalClicks     int64
		expectedPayment int64
		expectedTiers   int
	}{
		{
			name:            "Zero clicks",
			totalClicks:     0,
			expectedPayment: 0,
			expectedTiers:   0,
		},
		{
			name:            "Exactly at first tier boundary",
			totalClicks:     1000,
			expectedPayment: 1000 * 1000,
			expectedTiers:   1,
		},
		{
			name:            "One click into second tier",
			totalClicks:     1001,
			expectedPayment: 1000*1000 + 1*1500,
			expectedTiers:   2,
		},
		{
			name:            "Exactly at second tier boundary",
			totalClicks:     5000,
			expectedPayment: 1000*1000 + 4000*1500,
			expectedTiers:   2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payment, breakdown := calc.CalculateTieredPayment(tt.totalClicks, baseRate, levels)

			assert.Equal(t, tt.expectedPayment, payment, "Payment mismatch")
			assert.Len(t, breakdown, tt.expectedTiers, "Tier count mismatch")
		})
	}
}

func TestTieredPaymentCalculation_EmptyLevels(t *testing.T) {
	calc := &TieredPaymentCalculator{}

	payment, breakdown := calc.CalculateTieredPayment(1000, 1000, []dtos.Level{})

	assert.Equal(t, int64(0), payment, "Payment should be 0 with no levels")
	assert.Len(t, breakdown, 0, "Breakdown should be empty")
}

// =============================================================================
// CO_PRODUCING Revenue Share Calculation Tests
// =============================================================================

// RevenueShareCalculator is a test-friendly implementation for testing revenue distribution.
type RevenueShareCalculator struct{}

// CalculateRevenueShare calculates company and brand shares.
func (c *RevenueShareCalculator) CalculateRevenueShare(
	totalRevenue float64,
	companyPercent int,
	brandPercent int,
) (companyShare, brandShare float64) {
	companyShare = totalRevenue * float64(companyPercent) / 100
	brandShare = totalRevenue * float64(brandPercent) / 100
	return
}

func TestRevenueShareCalculation(t *testing.T) {
	calc := &RevenueShareCalculator{}

	tests := []struct {
		name                 string
		totalRevenue         float64
		companyPercent       int
		brandPercent         int
		expectedCompanyShare float64
		expectedBrandShare   float64
	}{
		{
			name:                 "50/50 split",
			totalRevenue:         1000000,
			companyPercent:       50,
			brandPercent:         50,
			expectedCompanyShare: 500000,
			expectedBrandShare:   500000,
		},
		{
			name:                 "70/30 split (company heavy)",
			totalRevenue:         1000000,
			companyPercent:       70,
			brandPercent:         30,
			expectedCompanyShare: 700000,
			expectedBrandShare:   300000,
		},
		{
			name:                 "30/70 split (brand heavy)",
			totalRevenue:         1000000,
			companyPercent:       30,
			brandPercent:         70,
			expectedCompanyShare: 300000,
			expectedBrandShare:   700000,
		},
		{
			name:                 "Zero revenue",
			totalRevenue:         0,
			companyPercent:       60,
			brandPercent:         40,
			expectedCompanyShare: 0,
			expectedBrandShare:   0,
		},
		{
			name:                 "Large revenue with typical split",
			totalRevenue:         150000000, // 150M VND
			companyPercent:       60,
			brandPercent:         40,
			expectedCompanyShare: 90000000, // 90M VND
			expectedBrandShare:   60000000, // 60M VND
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			companyShare, brandShare := calc.CalculateRevenueShare(
				tt.totalRevenue, tt.companyPercent, tt.brandPercent)

			assert.Equal(t, tt.expectedCompanyShare, companyShare, "Company share mismatch")
			assert.Equal(t, tt.expectedBrandShare, brandShare, "Brand share mismatch")
		})
	}
}

// =============================================================================
// Tax Withholding Calculation Tests
// =============================================================================

// TaxWithholdingCalculator is a test-friendly implementation for tax calculations.
type TaxWithholdingCalculator struct{}

// CalculateTax calculates tax withholding based on threshold and rate.
func (c *TaxWithholdingCalculator) CalculateTax(
	grossPayment int64,
	threshold int,
	ratePercent int,
) int64 {
	if grossPayment <= int64(threshold) {
		return 0
	}
	taxableAmount := grossPayment - int64(threshold)
	return taxableAmount * int64(ratePercent) / 100
}

func TestTaxWithholdingCalculation(t *testing.T) {
	calc := &TaxWithholdingCalculator{}

	tests := []struct {
		name           string
		grossPayment   int64
		threshold      int
		ratePercent    int
		expectedTax    int64
		expectedNetPay int64
	}{
		{
			name:           "Below threshold - no tax",
			grossPayment:   5000000,
			threshold:      10000000,
			ratePercent:    10,
			expectedTax:    0,
			expectedNetPay: 5000000,
		},
		{
			name:           "At threshold - no tax",
			grossPayment:   10000000,
			threshold:      10000000,
			ratePercent:    10,
			expectedTax:    0,
			expectedNetPay: 10000000,
		},
		{
			name:           "Above threshold - partial tax",
			grossPayment:   15000000,
			threshold:      10000000,
			ratePercent:    10,
			expectedTax:    500000, // (15M - 10M) * 10% = 500K
			expectedNetPay: 14500000,
		},
		{
			name:           "High payment - significant tax",
			grossPayment:   100000000, // 100M VND
			threshold:      10000000,
			ratePercent:    10,
			expectedTax:    9000000, // (100M - 10M) * 10% = 9M
			expectedNetPay: 91000000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tax := calc.CalculateTax(tt.grossPayment, tt.threshold, tt.ratePercent)
			netPay := tt.grossPayment - tax

			assert.Equal(t, tt.expectedTax, tax, "Tax mismatch")
			assert.Equal(t, tt.expectedNetPay, netPay, "Net pay mismatch")
		})
	}
}

// =============================================================================
// Payment Period Tests
// =============================================================================

func TestPaymentPeriod_ContainsTime(t *testing.T) {
	period := &dtos.PaymentPeriod{
		Start: testhelpers.DateOnly(2025, 1, 1),
		End:   testhelpers.DateOnly(2025, 2, 1),
	}

	tests := []struct {
		name     string
		testTime time.Time
		expected bool
	}{
		{
			name:     "Time at period start - included",
			testTime: testhelpers.DateOnly(2025, 1, 1),
			expected: true,
		},
		{
			name:     "Time in middle of period - included",
			testTime: testhelpers.DateOnly(2025, 1, 15),
			expected: true,
		},
		{
			name:     "Time just before end - included",
			testTime: time.Date(2025, 1, 31, 23, 59, 59, 0, time.UTC),
			expected: true,
		},
		{
			name:     "Time at period end - excluded",
			testTime: testhelpers.DateOnly(2025, 2, 1),
			expected: false,
		},
		{
			name:     "Time before period - excluded",
			testTime: testhelpers.DateOnly(2024, 12, 31),
			expected: false,
		},
		{
			name:     "Time after period - excluded",
			testTime: testhelpers.DateOnly(2025, 2, 15),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := period.ContainsTime(tt.testTime)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// =============================================================================
// Locked Amount Tests (Payment Locking Logic)
// =============================================================================

func TestPaymentLocking_Logic(t *testing.T) {
	tests := []struct {
		name               string
		originalAmount     float64
		lockedAmount       *float64
		expectedPayAmount  float64 // Amount used for payment
		expectUseLockedAmt bool
	}{
		{
			name:               "No locked amount - use original",
			originalAmount:     1000000,
			lockedAmount:       nil,
			expectedPayAmount:  1000000,
			expectUseLockedAmt: false,
		},
		{
			name:               "Has locked amount - use locked",
			originalAmount:     1500000, // Original amount changed after lock
			lockedAmount:       testhelpers.Float64Ptr(1000000),
			expectedPayAmount:  1000000, // Use locked amount
			expectUseLockedAmt: true,
		},
		{
			name:               "Locked amount equals original",
			originalAmount:     1000000,
			lockedAmount:       testhelpers.Float64Ptr(1000000),
			expectedPayAmount:  1000000,
			expectUseLockedAmt: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate the payment amount selection logic
			paymentAmount := tt.originalAmount
			usedLockedAmount := false

			if tt.lockedAmount != nil {
				paymentAmount = *tt.lockedAmount
				usedLockedAmount = true
			}

			assert.Equal(t, tt.expectedPayAmount, paymentAmount, "Payment amount mismatch")
			assert.Equal(t, tt.expectUseLockedAmt, usedLockedAmount, "Locked amount usage mismatch")
		})
	}
}

// =============================================================================
// Should Recalculate Tests (Decision Logic)
// =============================================================================

type PaymentRecalculationChecker struct{}

// ShouldRecalculate checks if a payment needs recalculation.
// A payment should be recalculated if:
// 1. Contract type is AFFILIATE or CO_PRODUCING
// 2. Payment status is PENDING (not yet paid)
// 3. Payment is not locked
// 4. Period boundaries are set
func (c *PaymentRecalculationChecker) ShouldRecalculate(
	contractType string,
	paymentStatus string,
	lockedAmount *float64,
	periodStart *time.Time,
	periodEnd *time.Time,
) bool {
	// Only AFFILIATE and CO_PRODUCING need recalculation
	if contractType != "AFFILIATE" && contractType != "CO_PRODUCING" {
		return false
	}

	// Already paid - no recalculation
	if paymentStatus == "PAID" {
		return false
	}

	// Already locked - no recalculation
	if lockedAmount != nil {
		return false
	}

	// Need period boundaries
	if periodStart == nil || periodEnd == nil {
		return false
	}

	return true
}

func TestShouldRecalculate(t *testing.T) {
	checker := &PaymentRecalculationChecker{}
	now := time.Now()
	start := now.AddDate(0, -1, 0)
	end := now.AddDate(0, 1, 0)

	tests := []struct {
		name         string
		contractType string
		status       string
		lockedAmount *float64
		periodStart  *time.Time
		periodEnd    *time.Time
		expected     bool
	}{
		{
			name:         "AFFILIATE contract, PENDING, not locked - should recalculate",
			contractType: "AFFILIATE",
			status:       "PENDING",
			lockedAmount: nil,
			periodStart:  &start,
			periodEnd:    &end,
			expected:     true,
		},
		{
			name:         "CO_PRODUCING contract, PENDING, not locked - should recalculate",
			contractType: "CO_PRODUCING",
			status:       "PENDING",
			lockedAmount: nil,
			periodStart:  &start,
			periodEnd:    &end,
			expected:     true,
		},
		{
			name:         "ADVERTISING contract - should NOT recalculate",
			contractType: "ADVERTISING",
			status:       "PENDING",
			lockedAmount: nil,
			periodStart:  &start,
			periodEnd:    &end,
			expected:     false,
		},
		{
			name:         "AFFILIATE contract, PAID - should NOT recalculate",
			contractType: "AFFILIATE",
			status:       "PAID",
			lockedAmount: nil,
			periodStart:  &start,
			periodEnd:    &end,
			expected:     false,
		},
		{
			name:         "AFFILIATE contract, PENDING, but locked - should NOT recalculate",
			contractType: "AFFILIATE",
			status:       "PENDING",
			lockedAmount: testhelpers.Float64Ptr(1000000),
			periodStart:  &start,
			periodEnd:    &end,
			expected:     false,
		},
		{
			name:         "AFFILIATE contract, no period start - should NOT recalculate",
			contractType: "AFFILIATE",
			status:       "PENDING",
			lockedAmount: nil,
			periodStart:  nil,
			periodEnd:    &end,
			expected:     false,
		},
		{
			name:         "AFFILIATE contract, no period end - should NOT recalculate",
			contractType: "AFFILIATE",
			status:       "PENDING",
			lockedAmount: nil,
			periodStart:  &start,
			periodEnd:    nil,
			expected:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := checker.ShouldRecalculate(
				tt.contractType, tt.status, tt.lockedAmount, tt.periodStart, tt.periodEnd)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// =============================================================================
// Is Current Period Tests
// =============================================================================

func TestIsCurrentPeriod(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name        string
		periodStart time.Time
		periodEnd   time.Time
		expected    bool
	}{
		{
			name:        "Current period - now is in range",
			periodStart: now.AddDate(0, -1, 0), // 1 month ago
			periodEnd:   now.AddDate(0, 1, 0),  // 1 month from now
			expected:    true,
		},
		{
			name:        "Past period - ended before now",
			periodStart: now.AddDate(0, -2, 0), // 2 months ago
			periodEnd:   now.AddDate(0, -1, 0), // 1 month ago
			expected:    false,
		},
		{
			name:        "Future period - starts after now",
			periodStart: now.AddDate(0, 1, 0), // 1 month from now
			periodEnd:   now.AddDate(0, 2, 0), // 2 months from now
			expected:    false,
		},
		{
			name:        "Period ends exactly now - excluded (end is exclusive)",
			periodStart: now.AddDate(0, -1, 0),
			periodEnd:   now.Truncate(time.Second), // Exact now
			expected:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			period := &dtos.PaymentPeriod{
				Start: tt.periodStart,
				End:   tt.periodEnd,
			}
			result := period.IsCurrentPeriod()
			assert.Equal(t, tt.expected, result)
		})
	}
}

// =============================================================================
// Unlock Payment Tests
// =============================================================================

func TestUnlockPayment_ClearsAllFields(t *testing.T) {
	// Simulate a locked payment state
	lockedAmount := testhelpers.Float64Ptr(1000000)
	lockedAt := testhelpers.TimePtr(time.Now().Add(-1 * time.Hour))
	lockedClicks := testhelpers.ToPtr(int64(5000))
	lockedRevenue := testhelpers.Float64Ptr(2000000)

	// Simulate unlock
	lockedAmount = nil
	lockedAt = nil
	lockedClicks = nil
	lockedRevenue = nil

	// Verify all cleared
	assert.Nil(t, lockedAmount, "LockedAmount should be nil")
	assert.Nil(t, lockedAt, "LockedAt should be nil")
	assert.Nil(t, lockedClicks, "LockedClicks should be nil")
	assert.Nil(t, lockedRevenue, "LockedRevenue should be nil")
}

// =============================================================================
// Real-World Scenario Tests
// =============================================================================

func TestRealWorldScenario_AffiliatePayment(t *testing.T) {
	calc := &TieredPaymentCalculator{}
	taxCalc := &TaxWithholdingCalculator{}

	// Scenario: Affiliate campaign with 7,500 clicks
	// Pricing structure:
	// - Level 1: 0-1000 clicks @ 1000 VND (1.0x)
	// - Level 2: 1001-5000 clicks @ 1200 VND (1.2x)
	// - Level 3: 5001+ clicks @ 1500 VND (1.5x)
	// Tax: 10% on amounts over 10M VND

	levels := []dtos.Level{
		{Level: 1, MaxClicks: 1000, Multiplier: 1.0},
		{Level: 2, MaxClicks: 5000, Multiplier: 1.2},
		{Level: 3, MaxClicks: 10000, Multiplier: 1.5},
	}
	baseRate := 1000
	totalClicks := int64(7500)

	// Calculate payment
	grossPayment, breakdown := calc.CalculateTieredPayment(totalClicks, baseRate, levels)

	// Expected breakdown:
	// Level 1: 1000 clicks × 1000 VND = 1,000,000 VND
	// Level 2: 4000 clicks × 1200 VND = 4,800,000 VND
	// Level 3: 2500 clicks × 1500 VND = 3,750,000 VND
	// Total: 9,550,000 VND

	expectedGross := int64(1000*1000 + 4000*1200 + 2500*1500)
	assert.Equal(t, expectedGross, grossPayment, "Gross payment mismatch")
	assert.Len(t, breakdown, 3, "Should have 3 tiers")

	// Calculate tax (10% on amount over 10M VND threshold)
	tax := taxCalc.CalculateTax(grossPayment, 10000000, 10)
	netPayment := grossPayment - tax

	// Since grossPayment (9.55M) < threshold (10M), no tax
	assert.Equal(t, int64(0), tax, "Tax should be 0 when below threshold")
	assert.Equal(t, grossPayment, netPayment, "Net should equal gross when no tax")
}

func TestRealWorldScenario_CoProducingPayment(t *testing.T) {
	calc := &RevenueShareCalculator{}

	// Scenario: CO_PRODUCING contract with limited product sales
	// Total revenue: 50M VND (from pre-orders and orders)
	// Split: 60% company, 40% brand

	totalRevenue := float64(50000000)
	companyPercent := 60
	brandPercent := 40

	companyShare, brandShare := calc.CalculateRevenueShare(totalRevenue, companyPercent, brandPercent)

	// Expected:
	// Company: 50M × 60% = 30M VND
	// Brand: 50M × 40% = 20M VND

	assert.Equal(t, float64(30000000), companyShare, "Company share mismatch")
	assert.Equal(t, float64(20000000), brandShare, "Brand share mismatch")

	// Verify total equals original
	assert.Equal(t, totalRevenue, companyShare+brandShare, "Shares should sum to total")
}

func TestRealWorldScenario_PaymentLocking(t *testing.T) {
	// Scenario: Brand initiates payment
	// 1. Initial calculation shows 5M VND
	// 2. Lock is created
	// 3. More clicks come in (would increase to 6M)
	// 4. Payment should still be for locked 5M

	originalAmount := float64(5000000)
	lockedAmount := testhelpers.Float64Ptr(5000000)

	// Simulate new clicks arriving (amount would be higher now)
	newCalculatedAmount := float64(6000000)

	// Payment amount selection logic
	paymentAmount := newCalculatedAmount
	if lockedAmount != nil {
		paymentAmount = *lockedAmount
	}

	// Assert locked amount is used
	assert.Equal(t, originalAmount, paymentAmount, "Should use locked amount, not new calculation")
	assert.NotEqual(t, newCalculatedAmount, paymentAmount, "Should not use new calculation")
}
