package iservice

import (
	"context"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/domain/model"

	"github.com/google/uuid"
)

// ProductOptionService defines the interface for product option business logic
type ProductOptionService interface {
	// GetByType retrieves all product options of a specific type (with caching)
	GetByType(ctx context.Context, optionType string) ([]responses.ProductOptionResponse, error)
	// GetAll retrieves all product options with optional filtering and pagination
	GetAll(ctx context.Context, req *requests.ProductOptionFilterRequest) ([]responses.ProductOptionResponse, int64, error)
	// GetByID retrieves a product option by ID
	GetByID(ctx context.Context, id uuid.UUID) (*responses.ProductOptionResponse, error)
	// ValidateOption validates if a code is valid for the given option type
	ValidateOption(ctx context.Context, optionType model.ProductOptionType, code string) error
	// Create creates a new product option (Admin only)
	Create(ctx context.Context, uow irepository.UnitOfWork, req *requests.CreateProductOptionRequest) (*responses.ProductOptionResponse, error)
	// Update updates an existing product option (Admin only)
	Update(ctx context.Context, uow irepository.UnitOfWork, id uuid.UUID, req *requests.UpdateProductOptionRequest) (*responses.ProductOptionResponse, error)
	// Delete soft deletes a product option (Admin only)
	Delete(ctx context.Context, uow irepository.UnitOfWork, id uuid.UUID) error
	// InvalidateCache invalidates the cache for a specific option type
	InvalidateCache(optionType string)
}
