package iservice

import (
	"context"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/domain/enum"

	"github.com/google/uuid"
)

type BrandService interface {
	CreateBrand(ctx context.Context, request *requests.CreateBrandRequest) (*responses.BrandResponse, error)
	CreateBrandWithInActiveUsers(ctx context.Context, uow *irepository.UnitOfWork, request *requests.CreateBrandWithUserRequest) (*responses.BrandResponse, error)
	UpdateBrand(ctx context.Context, brandID uuid.UUID, request *requests.UpdateBrandRequest) (*responses.BrandResponse, error)
	GetByID(ctx context.Context, brandID uuid.UUID) (*responses.BrandDetailResponse, error)
	GetByFilter(ctx context.Context, request *requests.ListBrandsRequest) ([]responses.BrandResponse, int64, error)
	UpdateBrandStatus(ctx context.Context, brandID uuid.UUID, status enum.BrandStatus) (*responses.BrandResponse, error)

	// Products under brand Pagination (added page & limit)
	MyProducts(ctx context.Context, userID uuid.UUID, page int, limit int) ([]responses.ProductResponseV2, int64, error)
}
