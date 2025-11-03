package iproxies

import (
	"context"
	"core-backend/internal/application/dto/dtos"
	"core-backend/internal/application/dto/responses"
)

// PayOSProxy handles HTTP communication with PayOS API
type PayOSProxy interface {
	// CreatePaymentLink creates a new payment link in PayOS
	// POST /v2/payment-requests
	CreatePaymentLink(ctx context.Context, req *dtos.PayOSCreateLinkRequest) (*responses.PayOSLinkResponse, error)
	
	// GetPaymentInfo retrieves payment information by order code or payment link ID
	// GET /v2/payment-requests/{id}
	GetPaymentInfo(ctx context.Context, orderCode string) (*responses.PayOSOrderInfoResponse, error)
	
	// CancelPaymentLink cancels an existing payment link
	// POST /v2/payment-requests/{id}/cancel
	CancelPaymentLink(ctx context.Context, orderCode string, reason string) (*responses.PayOSOrderInfoResponse, error)
	
	// VerifyWebhookSignature verifies the HMAC-SHA256 signature of webhook data
	VerifyWebhookSignature(data []byte, signature string) bool
}
