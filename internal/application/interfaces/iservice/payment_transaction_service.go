package iservice

import (
	"context"
	"core-backend/internal/application/dto/dtos"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/irepository"

	"github.com/google/uuid"
)

// PaymentTransactionService handles PayOS-related payment operations at the application layer
// The name reflects that these operations create and manage payment transactions in the system.
type PaymentTransactionService interface {
	// GeneratePaymentLink creates a new payment link in PayOS and persists a PaymentTransaction record
	GeneratePaymentLink(ctx context.Context, uow irepository.UnitOfWork, req *requests.PaymentRequest) (*responses.PayOSLinkResponse, error)

	// GetPaymentStatus retrieves the current payment status from PayOS (read-only, no UnitOfWork needed)
	GetPaymentStatus(ctx context.Context, orderCode string) (*responses.PayOSOrderInfoResponse, error)

	// CancelPaymentLink cancels an existing payment link and updates database
	CancelPaymentLink(ctx context.Context, uow irepository.UnitOfWork, orderCode string, reason string) error

	// ProcessWebhook processes incoming webhook from PayOS and updates payment status
	ProcessWebhook(ctx context.Context, uow irepository.UnitOfWork, webhookPayload *dtos.PayOSWebhookPayload) error

	// CancelExpiredLinks finds and cancels all expired payment links
	// Returns the count of cancelled links and any error
	// Note: This method creates its own transactions internally for each cancellation
	CancelExpiredLinks(ctx context.Context) (int, error)

	// SyncPaymentStatus fetches latest status from PayOS and updates local record
	SyncPaymentStatus(ctx context.Context, uow irepository.UnitOfWork, paymentTransactionID uuid.UUID) error

	// ConfirmWebhookURL confirms the webhook URL with PayOS
	ConfirmWebhookURL(ctx context.Context, webhookURL string) (*dtos.PayOSConfirmWebhookResponse, error)
}
