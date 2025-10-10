package iservice

import (
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"

	"github.com/google/uuid"
)

// ProductService defines product related operations.
type ProductService interface {
	CreateProduct(dto *requests.CreateProductDTO, createdBy uuid.UUID) (*responses.ProductResponse, error)
	GetProductsPagination(limit, offset int, search string) ([]*responses.ProductResponse, int, error)
	GetProductByID(id string) (*responses.ProductResponse, error)
	// GetProductsByTask returns overview list of products belonging to a task (with pagination) after authorization.
	GetProductsByTask(taskID uuid.UUID, requestingUserID uuid.UUID, userRole string, limit, offset int) ([]*responses.ProductOverviewResponse, int, error)
}
