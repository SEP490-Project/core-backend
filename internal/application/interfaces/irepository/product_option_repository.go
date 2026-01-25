package irepository

import (
	"context"
	"core-backend/internal/domain/model"

	"github.com/google/uuid"
)

// ProductOptionRepository defines the interface for product option data access
type ProductOptionRepository interface {
	// GetByType retrieves all active product options of a specific type
	GetByType(ctx context.Context, optionType model.ProductOptionType, activeOnly bool) ([]model.ProductOption, error)
	// GetByTypeAndCode retrieves a specific product option by type and code
	GetByTypeAndCode(ctx context.Context, optionType model.ProductOptionType, code string) (*model.ProductOption, error)
	// IsValidOption checks if a code is valid for the given option type
	IsValidOption(ctx context.Context, optionType model.ProductOptionType, code string) (bool, error)
	// GetAll retrieves all product options with optional filtering
	GetAll(ctx context.Context, optionType *model.ProductOptionType, activeOnly bool, pageSize, pageNumber int) ([]model.ProductOption, int64, error)
	// GetByID retrieves a product option by ID
	GetByID(ctx context.Context, id uuid.UUID) (*model.ProductOption, error)
	// Create adds a new product option
	Create(ctx context.Context, option *model.ProductOption) error
	// Update modifies an existing product option
	Update(ctx context.Context, option *model.ProductOption) error
	// Delete soft deletes a product option
	Delete(ctx context.Context, id uuid.UUID) error
}
