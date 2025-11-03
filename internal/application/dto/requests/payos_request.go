package requests

import (
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"

	"github.com/google/uuid"
)

type PaymentItemRequest struct {
	Name     string `json:"name"`
	Quantity int    `json:"quantity"`
	Price    int64  `json:"price"`
}

// PaymentRequest represents a request to create a payment link for an order or contract
type PaymentRequest struct {
	// Reference information
	ReferenceID   uuid.UUID                            `json:"referenceId"`   // Order ID or Contract Payment ID
	ReferenceType enum.PaymentTransactionReferenceType `json:"referenceType"` // "ORDER" or "CONTRACT_PAYMENT"

	// Payment details
	Amount      int64                `json:"amount"`
	Description string               `json:"description"`
	Items       []PaymentItemRequest `json:"items,omitempty"`

	// Buyer information (for PayOS fields)
	BuyerName  string `json:"buyerName"`
	BuyerEmail string `json:"buyerEmail"`
	BuyerPhone string `json:"buyerPhone"`
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
