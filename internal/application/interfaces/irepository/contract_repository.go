package irepository

import (
	"context"
	"core-backend/internal/domain/model"

	"github.com/google/uuid"
)

type ContractRepository interface {
	GenericRepository[model.Contract]
	GetContractIDByTaskID(ctx context.Context, taskID uuid.UUID) (contractID uuid.UUID, err error)
	GetAllContractIDs(ctx context.Context) (contractIDs []uuid.UUID, err error)
}
