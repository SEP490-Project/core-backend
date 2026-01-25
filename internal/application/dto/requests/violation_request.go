package requests

import (
	"core-backend/internal/domain/enum"

	"github.com/google/uuid"
)

// InitiateViolationRequest is used to manually initiate a violation
type InitiateViolationRequest struct {
	// ContractID is now taken from URL param
	Reason string `json:"reason" validate:"required,min=10,max=2000"`
}

// SubmitRefundProofRequest is used by Marketing Staff to submit proof of refund
type SubmitRefundProofRequest struct {
	// ViolationID is inferred from ContractID in URL
	ProofURL string  `json:"proof_url" validate:"required,url"`
	Message  *string `json:"message" validate:"omitempty,max=1000"`
}

// ReviewRefundProofRequest is used by Brand to approve/reject refund proof
type ReviewRefundProofRequest struct {
	// ViolationID is inferred from ContractID in URL
	Action       string  `json:"action" validate:"required,oneof=APPROVE REJECT"`
	RejectReason *string `json:"reject_reason" validate:"omitempty,min=10,max=1000"`
}

// IsApprove returns true if the action is APPROVE
func (r *ReviewRefundProofRequest) IsApprove() bool {
	return r.Action == "APPROVE"
}

// ViolationFilterRequest is used to filter violation list
type ViolationFilterRequest struct {
	ContractID  *uuid.UUID                 `json:"contract_id" form:"contract_id"`
	CampaignID  *uuid.UUID                 `json:"campaign_id" form:"campaign_id"`
	BrandID     *uuid.UUID                 `json:"brand_id" form:"brand_id"`
	Type        *enum.ViolationType        `json:"type" form:"type" validate:"omitempty,oneof=BRAND KOL"`
	IsResolved  *bool                      `json:"is_resolved" form:"is_resolved"`
	ProofStatus *enum.ViolationProofStatus `json:"proof_status" form:"proof_status" validate:"omitempty,oneof=PENDING APPROVED REJECTED"`

	// Pagination
	Page     int `json:"page" form:"page" validate:"min=1"`
	PageSize int `json:"page_size" form:"page_size" validate:"min=1,max=100"`
}

// GetPage returns page with default value
func (r *ViolationFilterRequest) GetPage() int {
	if r.Page <= 0 {
		return 1
	}
	return r.Page
}

// GetPageSize returns page size with default value
func (r *ViolationFilterRequest) GetPageSize() int {
	if r.PageSize <= 0 {
		return 20
	}
	if r.PageSize > 100 {
		return 100
	}
	return r.PageSize
}

// CreatePenaltyPaymentRequest is used to create a penalty payment link
type CreatePenaltyPaymentRequest struct {
	ViolationID *uuid.UUID `json:"violation_id,omitempty" validate:"uuid" example:"a1b2c3d4-e5f6-7a8b-9c0d-e1f2a3b4c5d6"`
	ReturnURL   *string    `json:"return_url,omitempty" form:"return_url" validate:"url" example:"https://example.com/return"`
	CancelURL   *string    `json:"cancel_url,omitempty" form:"cancel_url" validate:"url" example:"https://example.com/cancel"`
}
