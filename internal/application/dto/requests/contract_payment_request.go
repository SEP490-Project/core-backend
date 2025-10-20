package requests

type UpdateContractPaymentStatusRequest struct {
	ContractPaymentID string `json:"-"`
	Status            string `json:"status" validate:"required,oneof=PENDING PAID OVERDUE" example:"paid"`
	PaymentMethod     string `json:"payment_method" validate:"required,oneof=BANK_TRANSFER CASH CHECK" example:"BANK_TRANSFER"`
}
