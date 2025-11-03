package requests

// ContractPaymentFilterRequest represents the filtering criteria for retrieving contract payments.
type ContractPaymentFilterRequest struct {
	PaginationRequest
	ContractKeyword *string `json:"contract_id,omitempty" validate:"omitempty,uuid4" example:"a1b2c3d4-e5f6-7a8b-9c0d-e1f2a3b4c5d6"`
	Status          *string `json:"status,omitempty" validate:"omitempty,oneof=PENDING PAID OVERDUE" example:"paid"`
	DueDateFrom     *string `json:"due_date_from,omitempty" validate:"omitempty,datetime=2006-01-02" example:"2023-01-01"`
	DueDateTo       *string `json:"due_date_to,omitempty" validate:"omitempty,datetime=2006-01-02" example:"2023-12-31"`
	PaymentMethod   *string `json:"payment_method,omitempty" validate:"omitempty,oneof=BANK_TRANSFER CASH CHECK" example:"BANK_TRANSFER"`
}
