package iservice

import (
	"context"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/irepository"

	"github.com/google/uuid"
)

type TagService interface {
	// Create create a new tag
	Create(ctx context.Context, uow irepository.UnitOfWork, request *requests.CreateTagRequest) (*responses.TagResponse, error)
	// GetByID get a tag by its ID
	GetByID(ctx context.Context, id uuid.UUID) (*responses.TagResponse, error)
	// GetByName get a tag by its name.
	// If the tag does not exist, create a new one with the given name.
	GetByName(ctx context.Context, uow irepository.UnitOfWork, name string, userID uuid.UUID) (*responses.TagResponse, error)
	// GetByFilter get a list of tags by filtering and pagination
	GetByFilter(ctx context.Context, filterRequest *requests.TagFilterRequest) ([]responses.TagResponse, int64, error)
	// UpdateByID update a tag by its ID
	UpdateByID(ctx context.Context, uow irepository.UnitOfWork, request *requests.UpdateTagRequest) (*responses.TagResponse, error)
	// DeleteByID delete a tag by its ID
	DeleteByID(ctx context.Context, uow irepository.UnitOfWork, id uuid.UUID) error
}
