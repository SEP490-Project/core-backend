package gormrepository

import (
	"context"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/domain/model"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type contractRepository struct {
	*genericRepository[model.Contract]
}

func NewContractRepository(db *gorm.DB) irepository.ContractRepository {
	return &contractRepository{genericRepository: &genericRepository[model.Contract]{db: db}}
}

func (r *contractRepository) GetContractIDByTaskID(ctx context.Context, taskID uuid.UUID) (contractID uuid.UUID, err error) {
	if taskID == uuid.Nil {
		return uuid.Nil, errors.New("taskID cannot be nil")
	}

	query := r.db.WithContext(ctx).Model(new(model.Contract)).
		Select("contracts.id").
		Joins("JOIN campaigns ON campaigns.contract_id = contracts.id").
		Joins("JOIN milestones ON milestones.campaign_id = campaigns.id").
		Joins("JOIN tasks ON tasks.milestone_id = milestones.id").
		Where("tasks.id = ?", taskID).
		Distinct("contracts.id")
	if err = query.Scan(&contractID).Error; err != nil {
		return uuid.Nil, err
	}
	return contractID, nil
}

func (r *contractRepository) GetAllContractIDs(ctx context.Context) (contractIDs []uuid.UUID, err error) {
	if err = r.db.WithContext(ctx).Model(new(model.Contract)).Pluck("id", &contractIDs).Error; err != nil {
		return nil, err
	}
	return contractIDs, nil
}
