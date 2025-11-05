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
	ReferenceID   uuid.UUID                            `json:"referenceId" validate:"required,uuid"`
	ReferenceType enum.PaymentTransactionReferenceType `json:"referenceType" validate:"required,oneof=ORDER CONTRACT"`

	// Payment details
	Amount      int64                `json:"amount" validate:"required,gt=0"`
	Description string               `json:"description"`
	Items       []PaymentItemRequest `json:"items,omitempty"`

	// Buyer information (for PayOS fields)
	BuyerName  string `json:"buyerName"`
	BuyerEmail string `json:"buyerEmail"`
	BuyerPhone string `json:"buyerPhone"`
}

// ConfirmWebhookRequest represents a request to confirm the webhook URL with PayOS
type ConfirmWebhookRequest struct {
	WebhookURL string `json:"webhookUrl" validate:"required,url"`
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
