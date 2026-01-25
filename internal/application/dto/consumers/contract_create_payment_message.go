package consumers

// ContractCreatePaymentMessage represents the message structure for contract payment creation
type ContractCreatePaymentMessage struct {
	UserID     string `json:"user_id"`
	ContractID string `json:"contract_id"`
}
