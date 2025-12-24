package irepository

import (
	"context"
	"core-backend/internal/application/dto/dtos"
	"time"

	"github.com/google/uuid"
)

// ContractPaymentCalculationRepository provides specialized queries for
// AFFILIATE and CO_PRODUCING contract payment calculations.
// These queries involve complex JOINs across multiple tables and are
// optimized for payment period calculations.
type ContractPaymentCalculationRepository interface {
	// GetTotalClicksForContract counts all clicks for affiliate links
	// associated with a contract within a payment period.
	// Used for AFFILIATE contract payment calculation.
	GetTotalClicksForContract(ctx context.Context, contractID uuid.UUID, periodStart, periodEnd time.Time) (int64, error)

	// GetLimitedProductRevenue calculates total revenue from limited products
	// associated with a contract within a payment period.
	// Returns pre-order revenue, order revenue, and total.
	// Used for CO_PRODUCING contract payment calculation.
	GetLimitedProductRevenue(ctx context.Context, contractID uuid.UUID, periodStart, periodEnd time.Time) (*dtos.LimitedProductRevenueResult, error)
}
