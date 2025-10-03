package service

import (
	"context"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/application/interfaces/iservice"
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
	"core-backend/internal/domain/state/productsm"
	"core-backend/internal/domain/state/tasksm"
	"errors"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type stateTransferService struct {
	taskRepository    irepository.GenericRepository[model.Task]
	productRepository irepository.GenericRepository[model.Product]
}

func (t stateTransferService) MoveProductToState(productID uuid.UUID, targetState enum.ProductStatus) error {
	ctx := context.Background()

	product, err := t.productRepository.GetByID(ctx, productID, []string{})

	if err != nil {
		zap.L().Error("Failed to load Product from DB",
			zap.String("product ID", productID.String()),
			zap.Error(err))
		return errors.New("Failed to load task from DB" + err.Error())
	}

	// load current product context
	productCtx := &productsm.ProductContext{
		State: productsm.NewProductState(product.Status),
	}

	// init next product State
	nextProductState := productsm.NewProductState(targetState)
	if nextProductState == nil {
		zap.L().Error("Invalid target state",
			zap.String("target_state", targetState.String()))
		return errors.New("Invalid target state")
	}

	//4. Forward state
	if err := productCtx.State.Next(productCtx, nextProductState); err != nil {
		zap.L().Error("State transition failed",
			zap.String("user_id", productID.String()),
			zap.String("from", productCtx.State.Name().String()),
			zap.String("to", targetState.String()),
			zap.Error(err))
		return errors.New("State transition failed: " + err.Error())
	}

	//5. Save to DB
	product.Status = targetState
	if err := t.productRepository.Update(ctx, product); err != nil {
		zap.L().Error("Failed to update product state in DB",
			zap.String("user_id", productID.String()),
			zap.String("new_state", targetState.String()),
			zap.Error(err))
		return errors.New("Failed to update product state in DB: " + err.Error())
	}

	return nil
}

func (t stateTransferService) MoveMileStoneToState(mileStoneID uuid.UUID, targetState enum.MilestoneStatus) error {
	//TODO implement me
	panic("implement me")
}

func (t stateTransferService) MoveCampaignToState(campaignID uuid.UUID, targetState enum.CampaignStatus) error {
	//TODO implement me
	panic("implement me")
}

func (t stateTransferService) MoveContractToState(contractID uuid.UUID, targetState enum.ContractStatus) error {
	//TODO implement me
	panic("implement me")
}

func (t stateTransferService) MoveTaskToState(taskID uuid.UUID, targetState enum.TaskStatus) error {
	//1. Load current task from DB
	ctx := context.Background()
	task, err := t.taskRepository.GetByID(ctx, taskID, []string{"Products", "Contents"})
	if err != nil {
		zap.L().Error("Failed to load task from DB",
			zap.String("task ID", taskID.String()),
			zap.Error(err))
		return errors.New("Failed to load task from DB" + err.Error())
	}

	//2. Load task context
	taskCtx := &tasksm.TaskContext{
		State:    tasksm.NewTaskState(task.Status),
		Products: task.Products,
	}

	//3. Init target State
	nextState := tasksm.NewTaskState(targetState)
	if nextState == nil {
		zap.L().Error("Invalid target state",
			zap.String("user_id", taskID.String()),
			zap.String("target_state", targetState.String()))
		return errors.New("Invalid target state")
	}

	//4. Forward state
	if err := taskCtx.State.Next(taskCtx, nextState); err != nil {
		zap.L().Error("State transition failed",
			zap.String("user_id", taskID.String()),
			zap.String("from", taskCtx.State.Name().String()),
			zap.String("to", targetState.String()),
			zap.Error(err))
		return errors.New("State transition failed: " + err.Error())
	}

	//5. Save to DB
	task.Status = targetState
	if err := t.taskRepository.Update(ctx, task); err != nil {
		zap.L().Error("Failed to update task state in DB",
			zap.String("user_id", taskID.String()),
			zap.String("new_state", targetState.String()),
			zap.Error(err))
		return errors.New("Failed to update task state in DB: " + err.Error())
	}

	return nil
}

func NewStateTransferService(taskRepo irepository.GenericRepository[model.Task], productRepo irepository.GenericRepository[model.Product]) iservice.StateTransferService {
	return &stateTransferService{taskRepository: taskRepo, productRepository: productRepo}
}
