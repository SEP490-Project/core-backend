package paymenttransactionsm

import (
	"core-backend/internal/domain/enum"
	"fmt"
)

// PendingState represents PENDING payment transaction status
type PendingState struct{}

func (p *PendingState) Name() enum.PaymentTransactionStatus {
	return enum.PaymentTransactionStatusPending
}

func (p *PendingState) AllowedTransitions() map[enum.PaymentTransactionStatus]bool {
	return map[enum.PaymentTransactionStatus]bool{
		enum.PaymentTransactionStatusCompleted: true,
		enum.PaymentTransactionStatusFailed:    true,
		enum.PaymentTransactionStatusCancelled: true,
		enum.PaymentTransactionStatusExpired:   true,
	}
}

func (p *PendingState) Next(ctx *PaymentTransactionContext, next PaymentTransactionState) error {
	if _, ok := p.AllowedTransitions()[next.Name()]; ok {
		ctx.State = next
		return nil
	}
	return fmt.Errorf("invalid transition from PENDING to %s", next.Name())
}
