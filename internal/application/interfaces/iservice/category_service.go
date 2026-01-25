package iservice

import (
	"context"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/irepository"

	"github.com/google/uuid"
)

type ProductCategoryService interface {
	GetAllCategories(limit, offset int, search string, deleted string) ([]*responses.ProductCategoryResponse, int64, error)
	CreateCategory(requests.CreateProductCategoryRequest) (*responses.ProductCategoryResponse, error)
	AddParentCategory(currentID uuid.UUID, parentID uuid.UUID) (*responses.ProductCategoryResponse, error)
	DeleteCategory(ctx context.Context, categoryID uuid.UUID, uow irepository.UnitOfWork) error
	UpdateCategory(categoryID uuid.UUID, req requests.UpdateProductCategoryRequest) (*responses.ProductCategoryResponse, error)
}
