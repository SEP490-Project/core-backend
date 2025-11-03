package paymenttransactionsm

import (
	"core-backend/internal/domain/enum"
	"fmt"
)

// ExpiredState represents EXPIRED payment transaction status (terminal state)
type ExpiredState struct{}

func (e *ExpiredState) Name() enum.PaymentTransactionStatus {
	return enum.PaymentTransactionStatusExpired
}

func (e *ExpiredState) AllowedTransitions() map[enum.PaymentTransactionStatus]bool {
	return map[enum.PaymentTransactionStatus]bool{
		// Terminal state - no transitions allowed
	}
}

func (e *ExpiredState) Next(ctx *PaymentTransactionContext, next PaymentTransactionState) error {
	return fmt.Errorf("cannot transition from EXPIRED state (terminal state)")
}
