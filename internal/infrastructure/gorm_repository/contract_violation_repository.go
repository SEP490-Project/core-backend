package gormrepository

import (
	"context"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ContractViolationRepository struct {
	irepository.GenericRepository[model.ContractViolation]
	db *gorm.DB
}

// NewContractViolationRepository creates a new ContractViolationRepository
func NewContractViolationRepository(db *gorm.DB) irepository.ContractViolationRepository {
	return &ContractViolationRepository{
		GenericRepository: NewGenericRepository[model.ContractViolation](db),
		db:                db,
	}
}

// FindByContractID returns all violations for a contract
func (r *ContractViolationRepository) FindByContractID(ctx context.Context, contractID uuid.UUID) ([]*model.ContractViolation, error) {
	var violations []*model.ContractViolation
	err := r.db.WithContext(ctx).
		Where("contract_id = ?", contractID).
		Where("deleted_at IS NULL").
		Order("created_at DESC").
		Find(&violations).Error
	if err != nil {
		return nil, err
	}
	return violations, nil
}

// FindActiveByContractID returns the active (unresolved) violation for a contract
func (r *ContractViolationRepository) FindActiveByContractID(ctx context.Context, contractID uuid.UUID) (*model.ContractViolation, error) {
	var violation model.ContractViolation
	err := r.db.WithContext(ctx).
		Preload("PaymentTransaction").
		Where("contract_id = ?", contractID).
		// Where("resolved_at IS NULL").
		Where("deleted_at IS NULL").
		First(&violation).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &violation, nil
}

// FindByProofStatus returns all violations with a specific proof status
func (r *ContractViolationRepository) FindByProofStatus(ctx context.Context, status enum.ViolationProofStatus) ([]*model.ContractViolation, error) {
	var violations []*model.ContractViolation
	err := r.db.WithContext(ctx).
		Where("proof_status = ?", status).
		Where("deleted_at IS NULL").
		Order("created_at DESC").
		Find(&violations).Error
	if err != nil {
		return nil, err
	}
	return violations, nil
}

// FindPendingProofReview returns all KOL violations with submitted proof awaiting review
func (r *ContractViolationRepository) FindPendingProofReview(ctx context.Context) ([]*model.ContractViolation, error) {
	var violations []*model.ContractViolation
	err := r.db.WithContext(ctx).
		Where("type = ?", enum.ViolationTypeKOL).
		Where("proof_status = ?", enum.ViolationProofStatusPending).
		Where("proof_url IS NOT NULL").
		Where("proof_url != ''").
		Where("resolved_at IS NULL").
		Where("deleted_at IS NULL").
		Order("proof_submitted_at ASC"). // Oldest first for FIFO processing
		Find(&violations).Error
	if err != nil {
		return nil, err
	}
	return violations, nil
}

// FindProofsOverdueForAutoApproval returns violations with proof_status=PENDING
// and proof_submitted_at is older than the cutoff date
func (r *ContractViolationRepository) FindProofsOverdueForAutoApproval(ctx context.Context, cutoffDate time.Time) ([]*model.ContractViolation, error) {
	var violations []*model.ContractViolation
	err := r.db.WithContext(ctx).
		Where("type = ?", enum.ViolationTypeKOL).
		Where("proof_status = ?", enum.ViolationProofStatusPending).
		Where("proof_url IS NOT NULL").
		Where("proof_url != ''").
		Where("proof_submitted_at IS NOT NULL").
		Where("proof_submitted_at < ?", cutoffDate).
		Where("resolved_at IS NULL").
		Where("deleted_at IS NULL").
		Order("proof_submitted_at ASC"). // Oldest first for FIFO processing
		Find(&violations).Error
	if err != nil {
		return nil, err
	}
	return violations, nil
}

// FindByCampaignID returns the violation for a specific campaign
func (r *ContractViolationRepository) FindByCampaignID(ctx context.Context, campaignID uuid.UUID) (*model.ContractViolation, error) {
	var violation model.ContractViolation
	err := r.db.WithContext(ctx).
		Where("campaign_id = ?", campaignID).
		Where("deleted_at IS NULL").
		First(&violation).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &violation, nil
}

// GetUnresolvedCount returns count of unresolved violations
func (r *ContractViolationRepository) GetUnresolvedCount(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.ContractViolation{}).
		Where("resolved_at IS NULL").
		Where("deleted_at IS NULL").
		Count(&count).Error
	return count, err
}

// GetUnresolvedByType returns count of unresolved violations by type
func (r *ContractViolationRepository) GetUnresolvedByType(ctx context.Context, violationType enum.ViolationType) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.ContractViolation{}).
		Where("type = ?", violationType).
		Where("resolved_at IS NULL").
		Where("deleted_at IS NULL").
		Count(&count).Error
	return count, err
}
