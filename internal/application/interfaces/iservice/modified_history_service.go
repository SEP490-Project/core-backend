package iservice

import (
	"context"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/irepository"

	"github.com/google/uuid"
)

type ModifiedHistoryService interface {
	Add(ctx context.Context, request *requests.CreateModifiedHistoryRequest) (*responses.ModifiedHistoryResponse, error)
	AddWithUOW(ctx context.Context, request *requests.CreateModifiedHistoryRequest, uow irepository.UnitOfWork) (*responses.ModifiedHistoryResponse, error)
	Update(ctx context.Context, id uuid.UUID, request *requests.UpdateModifiedHistoryRequest) (*responses.ModifiedHistoryResponse, error)
	UpdateWithUOW(ctx context.Context, id uuid.UUID, request *requests.UpdateModifiedHistoryRequest, uow irepository.UnitOfWork) (*responses.ModifiedHistoryResponse, error)
	GetByFilter(ctx context.Context, request *requests.ModifiedHistoryFilterRequest) ([]responses.ModifiedHistoryResponse, error)
}
