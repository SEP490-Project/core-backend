package paymenttransactionsm

import (
	"core-backend/internal/domain/enum"
	"fmt"
)

// CompletedState represents COMPLETED payment transaction status (terminal state)
type CompletedState struct{}

func (c *CompletedState) Name() enum.PaymentTransactionStatus {
	return enum.PaymentTransactionStatusCompleted
}

func (c *CompletedState) AllowedTransitions() map[enum.PaymentTransactionStatus]bool {
	return map[enum.PaymentTransactionStatus]bool{
		// Terminal state - no transitions allowed
	}
}

func (c *CompletedState) Next(ctx *PaymentTransactionContext, next PaymentTransactionState) error {
	return fmt.Errorf("cannot transition from COMPLETED state (terminal state)")
}
