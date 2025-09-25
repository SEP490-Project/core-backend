package iservice

import (
	"context"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/domain/enum"

	"github.com/google/uuid"
)

type BrandService interface {
	CreateBrand(ctx context.Context, request *requests.CreateBrandRequest) (*responses.BrandResponse, error)
	UpdateBrand(ctx context.Context, brandID uuid.UUID, request *requests.UpdateBrandRequest) (*responses.BrandResponse, error)
	GetByID(ctx context.Context, brandID uuid.UUID) (*responses.BrandResponse, error)
	GetByFilter(ctx context.Context, request *requests.ListBrandsRequest) (*responses.PaginationResponse[responses.BrandResponse], error)
	UpdateBrandStatus(ctx context.Context, brandID uuid.UUID, status enum.BrandStatus) (*responses.BrandResponse, error)
}
