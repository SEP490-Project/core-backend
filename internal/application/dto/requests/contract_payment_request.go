package requests

import "github.com/google/uuid"

// ContractPaymentFilterRequest represents the filtering criteria for retrieving contract payments.
type ContractPaymentFilterRequest struct {
	PaginationRequest
	BrandID       *string `json:"brand_id,omitempty" form:"brand_id" validate:"omitempty,uuid" example:"a1b2c3d4-e5f6-7a8b-9c0d-e1f2a3b4c5d6"`
	BrandUserID   *string `json:"brand_user_id,omitempty" form:"brand_user_id" validate:"omitempty,uuid" example:"a1b2c3d4-e5f6-7a8b-9c0d-e1f2a3b4c5d6"`
	ContractID    *string `json:"contract_id,omitempty" form:"contract_id" validate:"omitempty,uuid" example:"a1b2c3d4-e5f6-7a8b-9c0d-e1f2a3b4c5d6"`
	Status        *string `json:"status,omitempty" form:"status" validate:"omitempty,oneof=PENDING PAID OVERDUE" example:"paid"`
	DueDateFrom   *string `json:"due_date_from,omitempty" form:"due_date_from" validate:"omitempty,datetime=2006-01-02" example:"2023-01-01"`
	DueDateTo     *string `json:"due_date_to,omitempty" form:"due_date_to" validate:"omitempty,datetime=2006-01-02" example:"2023-12-31"`
	PaymentMethod *string `json:"payment_method,omitempty" form:"payment_method" validate:"omitempty,oneof=BANK_TRANSFER CASH CHECK" example:"BANK_TRANSFER"`
}

type GenerateContractPaymentLinkRequest struct {
	ContractPaymentID uuid.UUID `json:"-" validate:"required,uuid" example:"a1b2c3d4-e5f6-7a8b-9c0d-e1f2a3b4c5d6"`
	ReturnURL         string    `json:"return_url,omitempty" form:"returnUrl" validate:"required,url" example:"https://example.com/return"`
	CancelURL         string    `json:"cancel_url,omitempty" form:"cancelUrl" validate:"required,url" example:"https://example.com/cancel"`
}
