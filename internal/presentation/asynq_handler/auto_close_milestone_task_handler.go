package asynqhandler

import (
	"context"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/application/interfaces/iservice"

	"github.com/hibiken/asynq"
)

type AutoCloseMilestoneTaskHandler struct {
	contractService iservice.ContractService
	unitOfWork      irepository.UnitOfWork
}

func NewAutoCloseMilestoneTaskHandler(
	contractService iservice.ContractService,
	unitOfWork irepository.UnitOfWork,
) *AutoCloseMilestoneTaskHandler {
	return &AutoCloseMilestoneTaskHandler{
		contractService: contractService,
		unitOfWork:      unitOfWork,
	}
}

func (h *AutoCloseMilestoneTaskHandler) ProcessTask(ctx context.Context, task *asynq.Task) error {
	return nil
}
