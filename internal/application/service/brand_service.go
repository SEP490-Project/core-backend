package service

import (
	"context"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/application/interfaces/iservice"
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type BrandService struct {
	BrandRepository irepository.GenericRepository[model.Brand]
}

// CreateBrand implements iservice.BrandService.
func (b *BrandService) CreateBrand(ctx context.Context, request *requests.CreateBrandRequest) (*responses.BrandResponse, error) {
	zap.L().Info("Creating new brand", zap.Any("request", request))

	conditions := func(db *gorm.DB) *gorm.DB {
		return db.Where("name = ?", request.Name)
	}
	if brand, err := b.BrandRepository.GetByCondition(ctx, conditions, nil, true); err != nil {
		zap.L().Error("Failed to check existing brand", zap.Error(err))
		return nil, err
	} else if brand != nil {
		zap.L().Warn("Brand with the same name already exists", zap.String("name", request.Name))
		return nil, fmt.Errorf("brand with name %s already exists", request.Name)
	}

	brandModel := &model.Brand{
		ID:     uuid.New(),
		Name:   request.Name,
		Status: enum.BrandStatusActive,
	}
	if err := b.BrandRepository.Add(ctx, brandModel); err != nil {
		zap.L().Error("Failed to create brand", zap.Error(err))
		return nil, err
	}

	return responses.BrandResponse{}.ToBrandResponse(brandModel), nil
}

// GetByFilter implements iservice.BrandService.
func (b *BrandService) GetByFilter(ctx context.Context, request *requests.ListBrandsRequest) (*responses.PaginationResponse[responses.BrandResponse], error) {
	zap.L().Info("Fetching brands with filter", zap.Any("request", request))

	filter := func(db *gorm.DB) *gorm.DB {
		return db
	}

	var brands []model.Brand
	var err error
	var totalCount int64
	brandResponses := make([]responses.BrandResponse, 0)
	if brands, totalCount, err = b.BrandRepository.GetAll(ctx, filter, nil, request.Limit, request.Page); err != nil {
		zap.L().Error("Failed to fetch brands from repository", zap.Error(err))
		return nil, err
	} else if len(brands) == 0 {
		zap.L().Debug("No brands found matching the filter criteria")
		return responses.EmptyPaginationResponse[responses.BrandResponse]("No brands found matching the filter criteria", nil, request.Page, request.Limit), nil
	}

	totalPages := int((totalCount + int64(request.Limit) - 1) / int64(request.Limit))
	return responses.PaginatedResponse("Successfully fetched brands", http.StatusOK, brandResponses, responses.Pagination{
		Page:       request.Page,
		Limit:      request.Limit,
		Total:      totalCount,
		TotalPages: totalPages,
		HasNext:    request.Page < totalPages,
		HasPrev:    request.Page > 1,
	}), nil
}

// GetByID implements iservice.BrandService.
func (b *BrandService) GetByID(ctx context.Context, brandID uuid.UUID) (*responses.BrandResponse, error) {
	zap.L().Info("Fetching brand by ID", zap.String("brand_id", brandID.String()))

	conditions := func(db *gorm.DB) *gorm.DB {
		return db.Where("id = ?", brandID)
	}
	var brand *model.Brand
	var err error
	if brand, err = b.BrandRepository.GetByCondition(ctx, conditions, nil, true); err != nil {
		zap.L().Error("Failed to fetch brand from repository", zap.Error(err))
		return nil, err
	} else if brand == nil {
		zap.L().Warn("Brand not found", zap.String("brand_id", brandID.String()))
		return nil, fmt.Errorf("brand with ID %s not found", brandID.String())
	}

	return responses.BrandResponse{}.ToBrandResponse(brand), nil
}

// UpdateBrand implements iservice.BrandService.
func (b *BrandService) UpdateBrand(ctx context.Context, brandID uuid.UUID, request *requests.UpdateBrandRequest) (*responses.BrandResponse, error) {
	zap.L().Info("Updating brand", zap.String("brand_id", brandID.String()), zap.Any("request", request))

	conditions := func(db *gorm.DB) *gorm.DB {
		return db.Where("id = ?", brandID)
	}
	var brand *model.Brand
	var err error
	if brand, err = b.BrandRepository.GetByCondition(ctx, conditions, nil, true); err != nil {
		zap.L().Error("Failed to fetch brand from repository", zap.Error(err))
		return nil, err
	} else if brand == nil {
		zap.L().Warn("Brand not found", zap.String("brand_id", brandID.String()))
		return nil, fmt.Errorf("brand with ID %s not found", brandID.String())
	}

	updatedBrand := request.ToExistingBrand(brand)
	if err := b.BrandRepository.Update(ctx, updatedBrand); err != nil {
		zap.L().Error("Failed to update brand", zap.Error(err))
		return nil, err
	}

	return responses.BrandResponse{}.ToBrandResponse(updatedBrand), nil
}

// UpdateBrandStatus implements iservice.BrandService.
func (b *BrandService) UpdateBrandStatus(ctx context.Context, brandID uuid.UUID, status enum.BrandStatus) (*responses.BrandResponse, error) {
	zap.L().Info("Updating brand status", zap.String("brand_id", brandID.String()), zap.String("status", string(status)))

	conditions := func(db *gorm.DB) *gorm.DB {
		return db.Where("id = ?", brandID)
	}
	var brand *model.Brand
	var err error
	if brand, err = b.BrandRepository.GetByCondition(ctx, conditions, nil, true); err != nil {
		zap.L().Error("Failed to fetch brand from repository", zap.Error(err))
		return nil, err
	} else if brand == nil {
		zap.L().Warn("Brand not found", zap.String("brand_id", brandID.String()))
		return nil, fmt.Errorf("brand with ID %s not found", brandID.String())
	}

	brand.Status = status
	if err := b.BrandRepository.Update(ctx, brand); err != nil {
		zap.L().Error("Failed to update brand status", zap.Error(err))
		return nil, err
	}

	return responses.BrandResponse{}.ToBrandResponse(brand), nil
}

func NewBrandService(brandRepository irepository.GenericRepository[model.Brand]) iservice.BrandService {
	return &BrandService{BrandRepository: brandRepository}
}
