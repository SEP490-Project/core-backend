package iservice

import (
	"context"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/domain/model"

	"github.com/google/uuid"
)

// ViolationService defines the interface for contract violation handling
type ViolationService interface {
	// InitiateBrandViolation creates a brand violation record with calculated penalty amounts
	// Returns ContractViolation - caller is responsible for state transition
	InitiateBrandViolation(ctx context.Context, contractID uuid.UUID, reportedBy uuid.UUID, reason string) (*model.ContractViolation, error)

	// InitiateKOLViolation creates a KOL violation record with calculated refund amounts
	// Returns ContractViolation - caller is responsible for state transition
	InitiateKOLViolation(ctx context.Context, contractID uuid.UUID, reportedBy uuid.UUID, reason string) (*model.ContractViolation, error)

	// CreatePenaltyPayment creates a PayOS payment link for brand penalty
	CreatePenaltyPayment(ctx context.Context, userID uuid.UUID, request *requests.CreatePenaltyPaymentRequest) (*responses.PayOSLinkResponse, error)

	// SubmitRefundProof allows KOL to submit proof of refund
	SubmitRefundProof(ctx context.Context, violationID uuid.UUID, proofURL string, message *string, submittedBy uuid.UUID) (*model.ContractViolation, error)

	// ReviewRefundProof allows admin to approve/reject KOL refund proof
	ReviewRefundProof(ctx context.Context, violationID uuid.UUID, req *requests.ReviewRefundProofRequest, reviewedBy uuid.UUID) (*model.ContractViolation, error)

	// AutoApproveProof auto-approves proof after admin review deadline
	AutoApproveProof(ctx context.Context, violationID uuid.UUID) error

	// ResolveViolation marks a violation as resolved
	ResolveViolation(ctx context.Context, violationID uuid.UUID, resolvedBy uuid.UUID) error

	// GetByID retrieves a violation by ID
	GetByID(ctx context.Context, violationID uuid.UUID) (*model.ContractViolation, error)

	// GetByContractID retrieves active violation for a contract
	GetByContractID(ctx context.Context, contractID uuid.UUID) (*model.ContractViolation, error)

	// List retrieves violations with filtering
	List(ctx context.Context, filter *requests.ViolationFilterRequest) ([]*responses.ViolationListResponse, int64, error)

	// CalculateBrandPenalty calculates penalty amounts for brand violation
	CalculateBrandPenalty(ctx context.Context, contractID uuid.UUID) (*responses.ViolationCalculationResponse, error)

	// CalculateKOLRefund calculates refund amounts for KOL violation
	CalculateKOLRefund(ctx context.Context, contractID uuid.UUID) (*responses.ViolationCalculationResponse, error)
}
