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
		Joins("JOIN tasks ON tasks.contract_id = contracts.id").
		Where("tasks.id = ?", taskID).
		Distinct("contracts.id")
	if err = query.Scan(&contractID).Error; err != nil {
		return uuid.Nil, err
	}
	return contractID, nil
}
