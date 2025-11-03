// Package paymenttransactionsm provides payment transaction state implementations
package paymenttransactionsm

import (
	"core-backend/internal/domain/enum"
	"errors"
)

// PaymentTransactionState interface defines behavior for payment transaction states
type PaymentTransactionState interface {
	Name() enum.PaymentTransactionStatus
	Next(ctx *PaymentTransactionContext, next PaymentTransactionState) error
	AllowedTransitions() map[enum.PaymentTransactionStatus]bool
}

// NewPaymentTransactionState factory method to create state instances
func NewPaymentTransactionState(status enum.PaymentTransactionStatus) (PaymentTransactionState, error) {
	switch status {
	case enum.PaymentTransactionStatusPending:
		return &PendingState{}, nil
	case enum.PaymentTransactionStatusCompleted:
		return &CompletedState{}, nil
	case enum.PaymentTransactionStatusFailed:
		return &FailedState{}, nil
	case enum.PaymentTransactionStatusCancelled:
		return &CancelledState{}, nil
	case enum.PaymentTransactionStatusExpired:
		return &ExpiredState{}, nil
	default:
		return nil, errors.New("invalid payment transaction status: " + string(status))
	}
}
