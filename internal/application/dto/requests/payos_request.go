package requests

import (
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"

	"github.com/google/uuid"
)

type PaymentItemRequest struct {
	Name     string `json:"name" validate:"required"`
	Quantity int    `json:"quantity" validate:"required,gt=0"`
	Price    int64  `json:"price" validate:"required,gt=0"`
}

// PaymentRequest represents a request to create a payment link for an order or contract
type PaymentRequest struct {
	// Reference information
	ReferenceID   uuid.UUID                            `json:"reference_id" validate:"required,uuid"`
	ReferenceType enum.PaymentTransactionReferenceType `json:"reference_type" validate:"required,oneof=ORDER CONTRACT_PAYMENT"`

	// Payment details
	Amount      int64                `json:"amount" validate:"required,gt=0"`
	Description string               `json:"description"`
	Items       []PaymentItemRequest `json:"items,omitempty"`

	// Buyer information (for PayOS fields)
	BuyerName  string `json:"buyer_name"`
	BuyerEmail string `json:"buyer_email"`
	BuyerPhone string `json:"buyer_phone"`

	// URLs
	ReturnURL *string `json:"return_url,omitempty" validate:"omitempty,url"`
	CancelURL *string `json:"cancel_url,omitempty" validate:"omitempty,url"`
}

// ConfirmWebhookRequest represents a request to confirm the webhook URL with PayOS
type ConfirmWebhookRequest struct {
	WebhookURL string `json:"webhook_url" validate:"required,url"`
}

// CancelPaymentRequest represents a request to cancel a payment transaction
// The json format is adhere to the PayOS return format for cancelling payment instead of internal snake_case format
type CancelPaymentRequest struct {
	ReturnURL string           `json:"returnUrl,omitempty" validate:"omitempty,url"`
	Code      string           `json:"code" validate:"required"`
	ID        string           `json:"id" validate:"required,uuid"` // payment transaction id
	Cancel    bool             `json:"cancel" validate:"required"`
	Status    enum.PayOSStatus `json:"status" validate:"required"`
	OrderCode string           `json:"orderCode" validate:"required"`
}

// MapPaymentItemsFromOrderItems converts OrderItems to PaymentItemRequest
func MapPaymentItemsFromOrderItems(orderItems []model.OrderItem) []PaymentItemRequest {
	paymentItems := make([]PaymentItemRequest, 0, len(orderItems))
	for _, item := range orderItems {
		paymentItems = append(paymentItems, PaymentItemRequest{
			Name:     item.Variant.Product.Name,
			Quantity: item.Quantity,
			Price:    int64(item.UnitPrice),
		})
	}
	return paymentItems
}

// Mappers
