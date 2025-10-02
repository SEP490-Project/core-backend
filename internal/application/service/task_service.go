package service

import (
	"context"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/application/interfaces/iservice"
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
	"core-backend/internal/domain/state/tasksm"
	"errors"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type taskService struct {
	repository irepository.GenericRepository[model.Task]
}

func (t taskService) MoveTaskToState(taskID uuid.UUID, targetState enum.TaskStatus) error {
	//1. Load current task from DB
	ctx := context.Background()
	task, err := t.repository.GetByID(ctx, taskID, []string{"Products"})
	if err != nil {
		zap.L().Error("Failed to load task from DB",
			zap.String("user_id", taskID.String()),
			zap.Error(err))
		return errors.New("Failed to load task from DB" + err.Error())
	}

	//2. Load task context
	taskCtx := tasksm.NewTaskContext(tasksm.NewTaskState(task.Status), task.Products)

	//3. Init target State
	nextState := tasksm.NewTaskState(targetState)
	if nextState == nil {
		zap.L().Error("Invalid target state",
			zap.String("user_id", taskID.String()),
			zap.String("target_state", targetState.String()))
		return errors.New("Invalid target state")
	}

	//4. Forward state
	if err := taskCtx.State().Next(taskCtx, nextState); err != nil {
		zap.L().Error("State transition failed",
			zap.String("user_id", taskID.String()),
			zap.String("from", taskCtx.State().Name().String()),
			zap.String("to", targetState.String()),
			zap.Error(err))
		return errors.New("State transition failed: " + err.Error())
	}

	//5. Save to DB
	task.Status = targetState
	if err := t.repository.Update(ctx, task); err != nil {
		zap.L().Error("Failed to update task state in DB",
			zap.String("user_id", taskID.String()),
			zap.String("new_state", targetState.String()),
			zap.Error(err))
		return errors.New("Failed to update task state in DB: " + err.Error())
	}

	return nil
}

func NewTaskService(repo irepository.GenericRepository[model.Task]) iservice.TaskService {
	return &taskService{repository: repo}
}
