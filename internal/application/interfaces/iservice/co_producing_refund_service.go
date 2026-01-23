package iservice

import (
	"context"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/domain/model"
	"time"

	"github.com/google/uuid"
)

// CoProducingRefundService handles refund proof workflow for CO_PRODUCING contracts
// when the net amount (base_cost - performance_cost) is negative
type CoProducingRefundService interface {
	// SubmitRefundProof allows Marketing Staff to submit proof of refund to brand
	// Status transition: KOL_PENDING or KOL_PROOF_REJECTED -> KOL_PROOF_SUBMITTED
	SubmitRefundProof(ctx context.Context, req *requests.SubmitCoProducingRefundProofRequest, submittedBy uuid.UUID) (*model.ContractPayment, error)

	// ReviewRefundProof allows Brand to approve or reject the submitted refund proof
	// Status transition:
	//   - Approve: KOL_PROOF_SUBMITTED -> KOL_REFUND_APPROVED
	//   - Reject: KOL_PROOF_SUBMITTED -> KOL_PROOF_REJECTED (if attempts remaining)
	ReviewRefundProof(ctx context.Context, req *requests.ReviewCoProducingRefundProofRequest, reviewedBy uuid.UUID) (*model.ContractPayment, error)

	// AutoApproveRefundProof auto-approves refund proof after review deadline
	// Called by daily job when brand hasn't reviewed within configured days
	AutoApproveRefundProof(ctx context.Context, paymentID uuid.UUID) error

	// GetRefundPayments returns all contract payments in refund workflow for a brand
	GetRefundPayments(ctx context.Context, brandUserID uuid.UUID) ([]responses.ContractPaymentResponse, error)

	// GetPendingRefundProofs returns payments awaiting brand review
	// Used by daily job for auto-approval check and reminder notifications
	GetPendingRefundProofs(ctx context.Context, submittedBefore *time.Time) ([]*model.ContractPayment, error)
}
