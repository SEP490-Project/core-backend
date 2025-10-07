package responses

import "time"

type PaymentItem struct {
	Name          string  `json:"name"`
	Quantity      int     `json:"quantity"`
	Price         float64 `json:"price"`
	Unit          string  `json:"unit"`
	TaxPercentage int     `json:"taxPercentage"`
}

type Invoice struct {
	BuyerNotGetInvoice bool `json:"buyerNotGetInvoice"`
	TaxPercentage      int  `json:"taxPercentage"`
}

// PaymentResponse represents the response structure for a payment operation.
type PaymentResponse struct {
	Code      string            `json:"code"`
	Desc      string            `json:"desc"`
	Data      PayOSLinkResponse `json:"data"`
	Signature string            `json:"signature"`
}

type PayOSLinkResponse struct {
	Bin           string  `json:"bin"`
	AccountNumber string  `json:"accountNumber"`
	AccountName   string  `json:"accountName"`
	Currency      string  `json:"currency"`
	PaymentLinkId string  `json:"paymentLinkId"`
	Amount        float64 `json:"amount"`
	Description   string  `json:"description"`
	OrderCode     int     `json:"orderCode"`
	ExpiredAt     int64   `json:"expiredAt"`
	Status        string  `json:"status"`
	CheckoutUrl   string  `json:"checkoutUrl"`
	QRCode        string  `json:"qrCode"`
}

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

type PayOSWrapperResponse[T any] struct {
	Code      string `json:"code"`
	Desc      string `json:"desc"`
	Data      T      `json:"data"`
	Signature string `json:"signature"`
}
