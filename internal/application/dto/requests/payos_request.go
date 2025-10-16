package requests

import (
	"core-backend/internal/application/dto/responses"
	"time"
)

type PaymentItemRequest struct {
	Name     string `json:"name"`
	Quantity int    `json:"quantity"`
	Price    int64  `json:"price"`
}
type InvoiceRequest struct {
	InvoiceCode string    `json:"invoiceCode"`
	InvoiceDate time.Time `json:"invoiceDate" swaggertype:"string" format:"date-time"`
}
type PaymentRequest struct {
	Amount           int64                `json:"amount"`
	Description      string               `json:"description"`
	BuyerName        string               `json:"buyerName"`
	BuyerCompanyName string               `json:"buyerCompanyName"`
	BuyerTaxCode     string               `json:"buyerTaxCode,omitempty"`
	BuyerAddress     string               `json:"buyerAddress"`
	BuyerEmail       string               `json:"buyerEmail"`
	BuyerPhone       string               `json:"buyerPhone"`
	Items            []PaymentItemRequest `json:"items,omitempty"`
	Invoice          *InvoiceRequest      `json:"invoice,omitempty"`
}

// ===== Backend Internal =====

// generateSignature
type PaymentSignatureRequest struct {
	Amount      int64  `json:"amount"`
	CancelUrl   string `json:"cancelUrl"` // backend gán từ config
	Description string `json:"description"`
	OrderCode   int64  `json:"orderCode"` // backend sinh
	ReturnUrl   string `json:"returnUrl"` // backend gán từ config
}

type PayOSRequest struct {
	PaymentSignatureRequest
	BuyerName        *string `json:"buyerName,omitempty"`
	BuyerCompanyName *string `json:"buyerCompanyName,omitempty"`
	BuyerTaxCode     *string `json:"buyerTaxCode,omitempty"`
	BuyerAddress     *string `json:"buyerAddress,omitempty"`
	BuyerEmail       *string `json:"buyerEmail,omitempty"`
	BuyerPhone       *string `json:"buyerPhone,omitempty"`

	Items   []responses.PaymentItem `json:"items"`
	Invoice responses.Invoice       `json:"invoice"`

	ExpiredAt int64  `json:"expiredAt"`
	Signature string `json:"signature"`
}
