package requests

import "core-backend/internal/application/dto/responses"

// Internal functions and types for PayOS integration
type PaymentSignatureRequest struct {
	Amount      float64 `json:"amount"`
	CancelUrl   string  `json:"cancelUrl"`
	Description string  `json:"description"`
	OrderCode   int     `json:"orderCode"`
	ReturnUrl   string  `json:"returnUrl"`
}

type PayOSRequest struct {
	PaymentSignatureRequest
	BuyerName        string                  `json:"buyerName"`
	BuyerCompanyName string                  `json:"buyerCompanyName"`
	BuyerTaxCode     string                  `json:"buyerTaxCode"`
	BuyerAddress     string                  `json:"buyerAddress"`
	BuyerEmail       string                  `json:"buyerEmail"`
	BuyerPhone       string                  `json:"buyerPhone"`
	Items            []responses.PaymentItem `json:"items"`
	Invoice          responses.Invoice       `json:"invoice"`
	ExpiredAt        int64                   `json:"expiredAt"`
	Signature        string                  `json:"signature"`
}

// ***** API Request *****
type PaymentRequest struct {
	PaymentSignatureRequest
	CancelUrl        string `json:"cancelUrl"`
	BuyerName        string `json:"buyerName"`
	BuyerCompanyName string `json:"buyerCompanyName"`
	BuyerTaxCode     string `json:"buyerTaxCode"`
	BuyerAddress     string `json:"buyerAddress"`
	BuyerEmail       string `json:"buyerEmail"`
	BuyerPhone       string `json:"buyerPhone"`
}
