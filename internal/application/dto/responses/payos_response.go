package responses

import "time"

// PayOSLinkResponse represents the response from PayOS when creating a payment link
type PayOSLinkResponse struct {
	Bin           string  `json:"bin"`
	AccountNumber string  `json:"accountNumber"`
	AccountName   string  `json:"accountName"`
	Currency      string  `json:"currency"`
	PaymentLinkID string  `json:"paymentLinkId"`
	Amount        float64 `json:"amount"`
	Description   string  `json:"description"`
	OrderCode     int     `json:"orderCode"`
	ExpiredAt     int64   `json:"expiredAt"`
	Status        string  `json:"status"`
	CheckoutURL   string  `json:"checkoutUrl"`
	QRCode        string  `json:"qrCode"`
}

// PayOSOrderInfoResponse represents detailed payment information from PayOS
type PayOSOrderInfoResponse struct {
	ID              string    `json:"id"`
	OrderCode       int       `json:"orderCode"`
	Amount          int       `json:"amount"`
	AmountPaid      int       `json:"amountPaid"`
	AmountRemaining int       `json:"amountRemaining"`
	Status          string    `json:"status"`
	CreatedAt       time.Time `json:"createdAt"`
	Transactions    []struct {
		Amount                 int       `json:"amount"`
		Description            string    `json:"description"`
		AccountNumber          string    `json:"accountNumber"`
		Reference              string    `json:"reference"`
		TransactionDateTime    time.Time `json:"transactionDateTime"`
		CounterAccountBankID   string    `json:"counterAccountBankId"`
		CounterAccountBankName string    `json:"counterAccountBankName"`
		CounterAccountName     string    `json:"counterAccountName"`
		CounterAccountNumber   string    `json:"counterAccountNumber"`
		VirtualAccountName     string    `json:"virtualAccountName"`
		VirtualAccountNumber   string    `json:"virtualAccountNumber"`
	} `json:"transactions"`
	CanceledAt         time.Time `json:"canceledAt"`
	CancellationReason string    `json:"cancellationReason"`
}
