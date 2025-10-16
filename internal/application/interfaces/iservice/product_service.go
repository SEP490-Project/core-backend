package iservice

import (
	"context"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/domain/model"

	"github.com/google/uuid"
)

// ProductService defines product related operations.
type ProductService interface {
	CreateStandardProduct(dto *requests.CreateStandardProductRequest, createdBy uuid.UUID) (*responses.ProductResponse, error)
	CreateLimitedProduct(dto *requests.CreateLimitedProductRequest, createdBy uuid.UUID) (*responses.ProductResponse, error)
	GetProductsPagination(limit, offset int, search string) ([]*responses.ProductResponse, int, error)
	GetProductByID(id string) (*responses.ProductResponse, error)
	// GetProductsByTask returns overview list of products belonging to a task (with pagination) after authorization.
	GetProductsByTask(taskID uuid.UUID, requestingUserID uuid.UUID, userRole string, limit, offset int) ([]*responses.ProductOverviewResponse, int, error)
	// GetProductVariants returns variants of a product with pagination.
	GetProductVariants(productID uuid.UUID, limit, offset int) ([]*responses.ProductVariantResponse, int, error)

	CreateProductVariance(ctx context.Context, userID uuid.UUID, productID uuid.UUID, variant requests.CreateProductVariantRequest, unitOfWork irepository.UnitOfWork) (*model.ProductVariant, error)
	CreateProductStory(ctx context.Context, variantID uuid.UUID, story requests.CreateProductStoryRequest, uow irepository.UnitOfWork) (*model.ProductStory, error)
	CreateVarianceImage(ctx context.Context, variantID uuid.UUID, image requests.CreateVariantImagesRequest, uow irepository.UnitOfWork) (*model.VariantImage, error)
	AddVariantAttributeValue(ctx context.Context, variantID uuid.UUID, attributeID uuid.UUID, attributeValue requests.CreateVariantAttributeValueRequest, uow irepository.UnitOfWork) (*model.VariantAttributeValue, error)
	CreateVariantAttribute(ctx context.Context, createdByID uuid.UUID, attribute requests.CreateVariantAttributeRequest, uow irepository.UnitOfWork) (*model.VariantAttribute, error)
}
