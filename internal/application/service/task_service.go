package service

import (
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/application/interfaces/iservice"
	"core-backend/internal/domain/model"
)

type taskService struct {
	repository irepository.GenericRepository[model.Task]
}

func NewTaskService(repo irepository.GenericRepository[model.Task]) iservice.TaskService {
	return &taskService{
		repository: repo,
	}
}

func (t taskService) MarkInProgress(taskID string) error {
	//TODO implement me
	//ctx := context.Background()
	//t.repository.GetByCondition(ctx, func(dbCondition any) any {) return dbCondition }, []string{}, "")
	panic("implement me")
}

func (t taskService) SubmitDraft(taskID string, draftContent string) error {
	//TODO implement me
	panic("implement me")
}

func (t taskService) RequestRevision(taskID string, comments string) error {
	//TODO implement me
	panic("implement me")
}

func (t taskService) ApproveDraft(taskID string) error {
	//TODO implement me
	panic("implement me")
}

func (t taskService) ReleaseDraft(taskID string) error {
	//TODO implement me
	panic("implement me")
}

func (t taskService) MarkCompleted(taskID string) error {
	//TODO implement me
	panic("implement me")
}
