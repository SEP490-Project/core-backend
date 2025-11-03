package paymenttransactionsm

import (
	"core-backend/internal/domain/enum"
	"fmt"
)

// CancelledState represents CANCELLED payment transaction status (terminal state)
type CancelledState struct{}

func (c *CancelledState) Name() enum.PaymentTransactionStatus {
	return enum.PaymentTransactionStatusCancelled
}

func (c *CancelledState) AllowedTransitions() map[enum.PaymentTransactionStatus]bool {
	return map[enum.PaymentTransactionStatus]bool{
		// Terminal state - no transitions allowed
	}
}

func (c *CancelledState) Next(ctx *PaymentTransactionContext, next PaymentTransactionState) error {
	return fmt.Errorf("cannot transition from CANCELLED state (terminal state)")
}
