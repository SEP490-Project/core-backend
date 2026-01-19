package irepository

import (
	"context"
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
	"time"

	"github.com/google/uuid"
)

// ContractViolationRepository defines the repository interface for contract violations
type ContractViolationRepository interface {
	GenericRepository[model.ContractViolation]

	// FindByContractID returns all violations for a contract
	FindByContractID(ctx context.Context, contractID uuid.UUID) ([]*model.ContractViolation, error)

	// FindActiveByContractID returns the active (unresolved) violation for a contract
	FindActiveByContractID(ctx context.Context, contractID uuid.UUID) (*model.ContractViolation, error)

	// FindByProofStatus returns all violations with a specific proof status
	FindByProofStatus(ctx context.Context, status enum.ViolationProofStatus) ([]*model.ContractViolation, error)

	// FindPendingProofReview returns all KOL violations with submitted proof awaiting review
	FindPendingProofReview(ctx context.Context) ([]*model.ContractViolation, error)

	// FindProofsOverdueForAutoApproval returns violations with proof_status=PENDING
	// and proof_submitted_at is older than the cutoff date
	FindProofsOverdueForAutoApproval(ctx context.Context, cutoffDate time.Time) ([]*model.ContractViolation, error)

	// FindByCampaignID returns the violation for a specific campaign
	FindByCampaignID(ctx context.Context, campaignID uuid.UUID) (*model.ContractViolation, error)

	// GetUnresolvedCount returns count of unresolved violations
	GetUnresolvedCount(ctx context.Context) (int64, error)

	// GetUnresolvedByType returns count of unresolved violations by type
	GetUnresolvedByType(ctx context.Context, violationType enum.ViolationType) (int64, error)
}
