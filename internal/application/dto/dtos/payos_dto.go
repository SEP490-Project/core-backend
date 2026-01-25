package dtos

import (
	"core-backend/internal/domain/enum"
)

// PayOSWebhookPayload represents the webhook payload sent by PayOS
type PayOSWebhookPayload struct {
	Code      string           `json:"code"`
	Desc      string           `json:"desc"`
	Success   bool             `json:"success"`
	Data      PayOSWebhookData `json:"data"`
	Signature string           `json:"signature"`
}

// PayOSWebhookData represents the data field in PayOS webhook payload
type PayOSWebhookData struct {
	OrderCode              int64  `json:"orderCode"`
	Amount                 int64  `json:"amount"`
	Description            string `json:"description"`
	AccountNumber          string `json:"accountNumber"`
	Reference              string `json:"reference"`
	TransactionDateTime    string `json:"transactionDateTime"`
	Currency               string `json:"currency"`
	PaymentLinkID          string `json:"paymentLinkId"`
	Code                   string `json:"code"`
	Desc                   string `json:"desc"`
	CounterAccountBankID   string `json:"counterAccountBankId"`
	CounterAccountBankName string `json:"counterAccountBankName"`
	CounterAccountName     string `json:"counterAccountName"`
	CounterAccountNumber   string `json:"counterAccountNumber"`
	VirtualAccountName     string `json:"virtualAccountName"`
	VirtualAccountNumber   string `json:"virtualAccountNumber"`
}

// PayOSCreateLinkRequest represents the internal request structure for creating a PayOS payment link
type PayOSCreateLinkRequest struct {
	OrderCode    int64       `json:"orderCode"`
	Amount       int64       `json:"amount"`
	Description  string      `json:"description"`
	BuyerName    *string     `json:"buyerName,omitempty"`
	BuyerEmail   *string     `json:"buyerEmail,omitempty"`
	BuyerPhone   *string     `json:"buyerPhone,omitempty"`
	BuyerAddress *string     `json:"buyerAddress,omitempty"`
	Items        []PayOSItem `json:"items,omitempty"`
	CancelURL    string      `json:"cancelUrl"`
	ReturnURL    string      `json:"returnUrl"`
	ExpiredAt    int64       `json:"expiredAt,omitempty"`
	Signature    string      `json:"signature"`
}

// PayOSItem represents an item in the payment
type PayOSItem struct {
	Name     string  `json:"name"`
	Quantity int     `json:"quantity"`
	Price    float64 `json:"price"`
}

// PayOSCancelRequest represents the request to cancel a payment link
type PayOSCancelRequest struct {
	CancellationReason string `json:"cancellationReason,omitempty"`
}

// PayOSConfirmWebhookRequest represents the request to confirm webhook setup
type PayOSConfirmWebhookRequest struct {
	WebhookURL string `json:"webhookUrl"`
}

// PayOSWrapperResponse is used internally by the proxy to parse PayOS API responses
// Generic wrapper that PayOS uses for all API responses
type PayOSWrapperResponse[T any] struct {
	Code      string `json:"code"`
	Desc      string `json:"desc"`
	Data      T      `json:"data"`
	Signature string `json:"signature"`
}

// PayOSConfirmWebhookResponse represents the response from PayOS when confirming webhook setup
type PayOSConfirmWebhookResponse struct {
	WebhookURL    string `json:"webhookUrl"`
	AccountNumber string `json:"accountNumber"`
	AccountName   string `json:"accountName"`
	Name          string `json:"name"`
	ShortName     string `json:"shortName"`
}

// MapPayOSStatusString converts a PayOS status string to internal PaymentTransactionStatus
func MapPayOSStatusString(payosStatus string) enum.PaymentTransactionStatus {
	switch payosStatus {
	case "PENDING":
		return enum.PaymentTransactionStatusPending
	case "PAID":
		return enum.PaymentTransactionStatusCompleted
	case "CANCELLED":
		return enum.PaymentTransactionStatusCancelled
	case "EXPIRED":
		return enum.PaymentTransactionStatusExpired
	default:
		return enum.PaymentTransactionStatusPending
	}
}
