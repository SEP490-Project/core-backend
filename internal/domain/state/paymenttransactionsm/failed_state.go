package paymenttransactionsm

import (
	"core-backend/internal/domain/enum"
	"fmt"
)

// FailedState represents FAILED payment transaction status (terminal state)
type FailedState struct{}

func (f *FailedState) Name() enum.PaymentTransactionStatus {
	return enum.PaymentTransactionStatusFailed
}

func (f *FailedState) AllowedTransitions() map[enum.PaymentTransactionStatus]bool {
	return map[enum.PaymentTransactionStatus]bool{
		// Terminal state - no transitions allowed
	}
}

func (f *FailedState) Next(ctx *PaymentTransactionContext, next PaymentTransactionState) error {
	return fmt.Errorf("cannot transition from FAILED state (terminal state)")
}
