package requests

import "github.com/google/uuid"

// region: ============== CO_PRODUCING Payment Refund Proof DTOs ==============

// SubmitCoProducingRefundProofRequest is used by Marketing Staff to submit refund proof
// when the CO_PRODUCING contract payment has a negative net amount (system owes brand)
type SubmitCoProducingRefundProofRequest struct {
	// ContractPaymentID is the ID of the contract payment in KOL_PENDING or KOL_PROOF_REJECTED status
	ContractPaymentID uuid.UUID `json:"contract_payment_id" validate:"required" example:"b3e1f9d2-8c4e-4f5a-9f1e-2d3c4b5a6e7f"`

	// RefundProofURL is the URL to the proof of refund (e.g., bank transfer screenshot)
	RefundProofURL string `json:"refund_proof_url" validate:"required,url" example:"https://s3.example.com/refund-proof.pdf"`

	// RefundProofNote is an optional note explaining the refund proof
	RefundProofNote *string `json:"refund_proof_note,omitempty" validate:"omitempty,max=1000" example:"Bank transfer completed on 2024-07-15"`
}

// ReviewCoProducingRefundProofRequest is used by Brand to review submitted refund proof
type ReviewCoProducingRefundProofRequest struct {
	// ContractPaymentID is the ID of the contract payment in KOL_PROOF_SUBMITTED status
	ContractPaymentID uuid.UUID `json:"contract_payment_id" validate:"required" example:"b3e1f9d2-8c4e-4f5a-9f1e-2d3c4b5a6e7f"`

	// Approved indicates whether the brand approves or rejects the refund proof
	Approved bool `json:"approved" example:"true"`

	// RejectReason is required when Approved is false
	// This field is required if Approved is false
	RejectReason *string `json:"reject_reason,omitempty" validate:"required_if=Approved false,omitempty,max=1000" example:"The bank transfer proof is unclear"`
}

// endregion
