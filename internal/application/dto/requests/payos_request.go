package requests

import (
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
	"core-backend/pkg/utils"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

type PaymentTransactionFilterRequest struct {
	PaginationRequest
	OrderCode           *int                                  `json:"order_code,omitempty" form:"order_code" validate:"omitempty,gt=0" example:"12345"`
	ReferenceID         *string                               `json:"reference_id,omitempty" form:"reference_id" validate:"omitempty,uuid" example:"a1b2c3d4-e5f6-7g8h-9i0j-k1l2m3n4o5p6"`
	ReferenceType       *enum.PaymentTransactionReferenceType `json:"reference_type,omitempty" form:"reference_type" validate:"omitempty,oneof=ORDER CONTRACT_PAYMENT" example:"ORDER"`
	PayerID             *string                               `json:"payer_id,omitempty" form:"payer_id" validate:"omitempty,uuid" example:"a1b2c3d4-e5f6-7g8h-9i0j-k1l2m3n4o5p6"`
	Status              *enum.PaymentTransactionStatus        `json:"status,omitempty" form:"status" validate:"omitempty,oneof=PENDING COMPLETED FAILED CANCELLED REFUNDED" example:"COMPLETED"`
	TransactionFromDate *string                               `json:"transaction_from_date,omitempty" form:"transaction_from_date" validate:"omitempty,datetime=2006-01-02" example:"2024-01-01"`
	TransactionToDate   *string                               `json:"transaction_to_date,omitempty" form:"transaction_to_date" validate:"omitempty,datetime=2006-01-02" example:"2024-01-31"`
}

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
	PayerID       *uuid.UUID                           `json:"payer_id,omitempty" validate:"omitempty,uuid"`

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
	ReturnURL string           `json:"returnUrl,omitempty" form:"returnUrl" validate:"required,url"`
	Code      string           `json:"code" form:"code" validate:"required"`
	ID        string           `json:"id" form:"id" validate:"required,uuid"` // payment transaction id
	Cancel    bool             `json:"cancel" form:"cancel" validate:"required"`
	Status    enum.PayOSStatus `json:"status" form:"status" validate:"required"`
	OrderCode string           `json:"orderCode" form:"orderCode" validate:"required"`
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

func ValidatePaymentTransactionFilterRequest(sl validator.StructLevel) {
	filterRequest := sl.Current().Interface().(PaymentTransactionFilterRequest)
	var err error

	if filterRequest.TransactionFromDate != nil {
		_, err = utils.ParseLocalTime(*filterRequest.TransactionFromDate, utils.DateFormat)
		if err != nil {
			sl.ReportError(filterRequest.TransactionFromDate, "transaction_from_date", "TransactionFromDate", "Filter.TransactionFromDate", "Invalid transaction from date format, should be YYYY-MM-DD")
		}
	}
	if filterRequest.TransactionToDate != nil {
		_, err = utils.ParseLocalTime(*filterRequest.TransactionToDate, utils.DateFormat)
		if err != nil {
			sl.ReportError(filterRequest.TransactionToDate, "transaction_to_date", "TransactionToDate", "Filter.TransactionToDate", "Invalid transaction to date format, should be YYYY-MM-DD")
		}
	}
}
