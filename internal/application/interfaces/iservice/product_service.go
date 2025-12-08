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
	CreateStandardProduct(dto *requests.CreateStandardProductRequest, createdBy uuid.UUID) (*responses.ProductResponseV2, error)
	CreateLimitedProduct(dto *requests.CreateLimitedProductRequest, createdBy uuid.UUID) (*responses.ProductResponseV2, error)
	GetProductsPagination(limit, offset int, search string, categoryID string, productType string) ([]*responses.ProductResponse, int, error)
	GetProductsPaginationV2(page, limit int, search, categoryID, brandID string, userID string, productType string, productStatuses []string, isPreOrderOnly bool) ([]responses.ProductResponseV2, int, error)
	GetProductsPaginationV2Partial(page, limit int, search string, categoryID string, brandID string, productType string, isPreOrderOnly bool) ([]responses.ProductResponseV2Partial, int, error)
	GetProductDetail(id uuid.UUID) (*responses.ProductDetailResponse, error)
	// Reviews
	AddProductReview(userID uuid.UUID, req requests.AddProductReviewRequest) (*responses.ProductReviewResponse, error)
	GetProductReviewPagination(productID uuid.UUID, limit, offset int) ([]responses.ProductReviewResponse, int, error)

	GetProductsByTask(taskID uuid.UUID, requestingUserID uuid.UUID, userRole string, limit, offset int) ([]*responses.ProductOverviewResponse, int, error)
	GetProductVariants(productID uuid.UUID, limit, offset int) ([]*responses.ProductVariantResponse, int, error)
	GetTop5NewestProducts() (*responses.ProductResponseTop5Newest, error)

	PublishProduct(productID uuid.UUID, isActive bool) (*responses.ProductResponseV2, error)

	// Update Product (standard)
	UpdateProduct(ctx context.Context, productID uuid.UUID, update requests.UpdateProductRequest) (*model.Product, error)

	// Update Limited Product
	UpdateLimitedProduct(ctx context.Context, productID uuid.UUID, update requests.UpdateLimitedProductRequest) (*model.Product, error)

	// Variants
	CreateProductVariance(ctx context.Context, userID uuid.UUID, productID uuid.UUID, variant requests.CreateProductVariantRequest, unitOfWork irepository.UnitOfWork) (*model.ProductVariant, error)
	CreateProductStory(ctx context.Context, variantID uuid.UUID, story requests.CreateProductStoryRequest, uow irepository.UnitOfWork) (*model.ProductStory, error)
	CreateVarianceImage(ctx context.Context, variantID uuid.UUID, image requests.CreateVariantImagesRequest, uow irepository.UnitOfWork) (*model.VariantImage, error)
	UpdateVariantImage(ctx context.Context, variantImageID uuid.UUID, image requests.UpdateVariantImagesRequest, uow irepository.UnitOfWork) (*model.VariantImage, error)
	UpdateVariantImageAsync(ctx context.Context, userID, variantImageID uuid.UUID, filePath *string, image requests.UpdateVariantImagesRequest, uow irepository.UnitOfWork) (*model.VariantImage, error)
	// Attributes
	AddVariantAttributeValue(ctx context.Context, variantID uuid.UUID, attributeID uuid.UUID, attributeValue requests.CreateVariantAttributeValueRequest, uow irepository.UnitOfWork) (*model.VariantAttributeValue, error)
	CreateVariantAttribute(ctx context.Context, createdByID uuid.UUID, attribute requests.CreateVariantAttributeRequest, uow irepository.UnitOfWork) (*model.VariantAttribute, error)
	GetVariantAttributePaginationPartial(limit, offset int, search string) ([]responses.VariantAttributeResponse, int, error)
	GetVariantAttributePagination(limit, offset int, search string) ([]model.VariantAttribute, int, error)

	// Update Variant
	UpdateVariant(ctx context.Context, variantID uuid.UUID, update requests.UpdateProductVariantRequest) (*model.ProductVariant, error)

	UpdateLimitedVariant(ctx context.Context, variantID uuid.UUID, update requests.UpdateLimitedProductVariantRequest) (*model.ProductVariant, error)

	// Concepts
	AddConceptToLimitedProduct(ctx context.Context, limitedProductID uuid.UUID, conceptID uuid.UUID, uow irepository.UnitOfWork) (*model.LimitedProduct, error)

	// Helpers
	BuildFileURL(fileName string) string
}
