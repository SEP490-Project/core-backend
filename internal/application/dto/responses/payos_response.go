package responses

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

// Response structs
type PaymentResponse struct {
	Code      string              `json:"code"`
	Desc      string              `json:"desc"`
	Data      PaymentResponseData `json:"data"`
	Signature string              `json:"signature"`
}

type PaymentResponseData struct {
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
