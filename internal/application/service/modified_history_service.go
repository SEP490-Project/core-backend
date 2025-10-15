package service

import (
	"context"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/application/interfaces/iservice"
	"core-backend/internal/domain/model"
	"fmt"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type ModifiedHistoryService struct {
	repo       irepository.GenericRepository[model.ModifiedHistory]
	retryCount int
}

// Add implements iservice.ModifiedHistoryService.
func (m *ModifiedHistoryService) Add(ctx context.Context, request *requests.CreateModifiedHistoryRequest) (*responses.ModifiedHistoryResponse, error) {
	zap.L().Debug("Adding new ModifiedHistory", zap.Any("request", request))

	model, err := request.ToModel()
	if err != nil {
		zap.L().Error("Failed to convert request to model", zap.Error(err))
		return nil, err
	}

	addFunc := func(attempt int) error {
		if err = m.repo.Add(ctx, model); err != nil {
			zap.L().Error(fmt.Sprintf("Failed to add ModifiedHistory (attempt %d)", attempt), zap.Error(err))
			return err
		}
		return nil
	}
	for i := 0; i < m.retryCount; i++ {
		err = addFunc(i)
		if err == nil {
			return responses.ModifiedHistoryResponse{}.ToModifiedHistoryResponse(model), nil
		}
	}

	zap.L().Debug("Successfully added ModifiedHistory", zap.Any("model", model))
	return nil, err
}

func (m *ModifiedHistoryService) AddWithUOW(ctx context.Context, request *requests.CreateModifiedHistoryRequest, uow irepository.UnitOfWork) (*responses.ModifiedHistoryResponse, error) {
	zap.L().Debug("Adding new ModifiedHistory through UnitOfWork", zap.Any("request", request))

	repo := uow.ModifiedHistories()
	model, err := request.ToModel()
	if err != nil {
		zap.L().Error("Failed to convert request to model", zap.Error(err))
		return nil, err
	}
	addFunc := func(attempt int) error {
		if err = repo.Add(ctx, model); err != nil {
			zap.L().Error(fmt.Sprintf("Failed to add ModifiedHistory (attempt %d)", attempt), zap.Error(err))
			return err
		}
		return nil
	}
	for i := 0; i < m.retryCount; i++ {
		err = addFunc(i)
		if err == nil {
			return responses.ModifiedHistoryResponse{}.ToModifiedHistoryResponse(model), nil
		}
	}

	return nil, err
}

// Update implements iservice.ModifiedHistoryService.
func (m *ModifiedHistoryService) Update(ctx context.Context, id uuid.UUID, request *requests.UpdateModifiedHistoryRequest) (*responses.ModifiedHistoryResponse, error) {
	zap.L().Debug("Updating ModifiedHistory status",
		zap.String("id", id.String()),
		zap.Any("request", request))

	history, err := m.repo.GetByID(ctx, id, nil)
	if err != nil {
		zap.L().Error("Failed to fetch ModifiedHistory", zap.Error(err))
		return nil, err
	} else if history == nil {
		zap.L().Warn("ModifiedHistory not found", zap.String("id", id.String()))
		return nil, fmt.Errorf("modified history not found")
	}

	history, err = request.ToExistingModel(history)
	if err != nil {
		zap.L().Error("Failed to update ModifiedHistory", zap.Error(err))
		return nil, err
	}

	for i := 0; i < m.retryCount; i++ {
		err = m.repo.Update(ctx, history)
		if err == nil {
			return responses.ModifiedHistoryResponse{}.ToModifiedHistoryResponse(history), nil
		}
	}

	return nil, err
}

// UpdateWithUOW implements iservice.ModifiedHistoryService.
func (m *ModifiedHistoryService) UpdateWithUOW(ctx context.Context, id uuid.UUID, request *requests.UpdateModifiedHistoryRequest, uow irepository.UnitOfWork) (*responses.ModifiedHistoryResponse, error) {
	zap.L().Debug("Updating ModifiedHistory status",
		zap.String("id", id.String()),
		zap.Any("request", request))

	repo := uow.ModifiedHistories()

	history, err := repo.GetByID(ctx, id, nil)
	if err != nil {
		zap.L().Error("Failed to fetch ModifiedHistory", zap.Error(err))
		return nil, err
	} else if history == nil {
		zap.L().Warn("ModifiedHistory not found", zap.String("id", id.String()))
		return nil, fmt.Errorf("modified history not found")
	}

	history, err = request.ToExistingModel(history)
	if err != nil {
		zap.L().Error("Failed to update ModifiedHistory", zap.Error(err))
		return nil, err
	}

	for i := 0; i < m.retryCount; i++ {
		err = repo.Update(ctx, history)
		if err == nil {
			return responses.ModifiedHistoryResponse{}.ToModifiedHistoryResponse(history), nil
		}
	}

	return nil, err
}

// GetByFilter implements iservice.ModifiedHistoryService.
func (m *ModifiedHistoryService) GetByFilter(
	ctx context.Context,
	filterRequest *requests.ModifiedHistoryFilterRequest,
) ([]responses.ModifiedHistoryResponse, error) {
	zap.L().Debug("Retrieving ModifiedHistory by filter", zap.Any("filter_request", filterRequest))

	filterQuery := func(db *gorm.DB) *gorm.DB {
		if filterRequest.ReferenceID != nil {
			db = db.Where("reference_id = ?", *filterRequest.ReferenceID)
		}
		if filterRequest.ReferenceType != nil {
			db = db.Where("reference_type = ?", *filterRequest.ReferenceType)
		}
		if filterRequest.ChangedByID != nil {
			db = db.Where("changed_by_id = ?", *filterRequest.ChangedByID)
		}
		if filterRequest.StartChangedAt != nil {
			db = db.Where("changed_at >= ?", *filterRequest.StartChangedAt)
		}
		if filterRequest.EndChangedAt != nil {
			db = db.Where("changed_at <= ?", *filterRequest.EndChangedAt)
		}

		sortBy := filterRequest.SortBy
		sortOrder := filterRequest.SortOrder
		if sortBy == "" {
			sortBy = "changed_at"
			if sortOrder == "" {
				sortOrder = "desc"
			}
		}
		db.Order(fmt.Sprintf("%s %s", sortBy, sortOrder))
		return db
	}
	histories, _, err := m.repo.GetAll(ctx, filterQuery, []string{}, filterRequest.Limit, filterRequest.Page)
	if err != nil {
		zap.L().Error("Failed to retrieve ModifiedHistory", zap.Error(err))
		return nil, err
	}

	return responses.ModifiedHistoryResponse{}.ToModifiedHistoryResponseList(histories), nil
}

func NewModifiedHistoryService(repo irepository.GenericRepository[model.ModifiedHistory]) iservice.ModifiedHistoryService {
	return &ModifiedHistoryService{repo: repo, retryCount: 3}
}
