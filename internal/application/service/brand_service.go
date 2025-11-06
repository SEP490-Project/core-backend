package service

import (
	"context"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/application/interfaces/iservice"
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
	"core-backend/pkg/utils"
	"fmt"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type brandService struct {
	BrandRepository irepository.GenericRepository[model.Brand]
}

// CreateBrand implements iservice.BrandService.
func (b *brandService) CreateBrand(ctx context.Context, request *requests.CreateBrandRequest) (*responses.BrandResponse, error) {
	zap.L().Info("Creating new brand", zap.Any("request", request))

	conditions := func(db *gorm.DB) *gorm.DB {
		return db.Where("name = ?", request.Name)
	}
	if brand, err := b.BrandRepository.GetByCondition(ctx, conditions, nil); err != nil {
		zap.L().Error("Failed to check existing brand", zap.Error(err))
		return nil, err
	} else if brand != nil {
		zap.L().Warn("Brand with the same name already exists", zap.String("name", request.Name))
		return nil, fmt.Errorf("brand with name %s already exists", request.Name)
	}

	brandModel := &model.Brand{
		ID:           uuid.New(),
		Name:         request.Name,
		Description:  request.Description,
		ContactEmail: request.ContactEmail,
		ContactPhone: request.ContactPhone,
		Address:      request.Address,
		Website:      request.Website,
		LogoURL:      request.LogoURL,
		Status:       enum.BrandStatusActive,
	}
	if err := b.BrandRepository.Add(ctx, brandModel); err != nil {
		zap.L().Error("Failed to create brand", zap.Error(err))
		return nil, err
	}

	return responses.BrandResponse{}.ToBrandResponse(brandModel), nil
}

// GetByFilter implements iservice.BrandService.
func (b *brandService) GetByFilter(ctx context.Context, request *requests.ListBrandsRequest) ([]responses.BrandResponse, int64, error) {
	zap.L().Info("Fetching brands with filter", zap.Any("request", request))

	filter := func(db *gorm.DB) *gorm.DB {
		if request.Keywords != nil && *request.Keywords != "" {
			likePattern := fmt.Sprintf("%%%s%%", *request.Keywords)
			db = db.Where("name ILIKE ?", likePattern)
		}
		if request.Status != nil && *request.Status != "" {
			db = db.Where("status = ?", enum.BrandStatus(*request.Status))
		}

		sortBy := "created_at"
		sortOrder := "desc"

		if request.SortBy != "" {
			sortBy = request.SortBy
		}
		if request.SortOrder != "" {
			sortOrder = request.SortOrder
		}

		switch sortBy {
		case "number_of_contracts":
			db = db.
				Select("brands.*, COUNT(contracts.id) AS number_of_contracts").
				Joins("LEFT JOIN contracts ON contracts.brand_id = brands.id").
				Group("brands.id").
				Order(fmt.Sprintf("COUNT(contracts.id) %s", sortOrder))
		case "number_of_active_contracts":
			db = db.
				Select("brands.*, SUM(CASE WHEN contracts.status = ? THEN 1 ELSE 0 END) AS number_of_active_contracts", enum.ContractStatusActive).
				Joins("LEFT JOIN contracts ON contracts.brand_id = brands.id").
				Group("brands.id").
				Order(fmt.Sprintf("SUM(CASE WHEN contracts.status = '%s' THEN 1 ELSE 0 END) %s", enum.ContractStatusActive, sortOrder))
		default:
			db = db.Order(fmt.Sprintf("%s %s", sortBy, sortOrder))
		}

		return db
	}

	var brands []model.Brand
	var err error
	var totalCount int64
	brandResponses := make([]responses.BrandResponse, 0)
	if brands, totalCount, err = b.BrandRepository.GetAll(ctx, filter, nil, request.Limit, request.Page); err != nil {
		zap.L().Error("Failed to fetch brands from repository", zap.Error(err))
		return nil, 0, err
	} else if len(brands) == 0 {
		zap.L().Debug("No brands found matching the filter criteria")
		return brandResponses, 0, nil
	}

	brandIDs := utils.MapSlice(brands, func(b model.Brand) uuid.UUID { return b.ID })

	type BrandCount struct {
		ID                      uuid.UUID
		NumberOfContracts       int
		NumberOfActiveContracts int
	}

	brandCounts := make([]BrandCount, 0, len(brandIDs))
	b.BrandRepository.DB().
		Model(&model.Brand{}).
		InnerJoins("INNER JOIN contracts ON contracts.brand_id = brands.id").
		Where("brands.id IN ?", brandIDs).
		Select("brands.id, COUNT(contracts.id) AS number_of_contracts, SUM(CASE WHEN contracts.status = ? THEN 1 ELSE 0 END) AS number_of_active_contracts", enum.ContractStatusActive).
		Group("brands.id").
		Scan(&brandCounts)

	brandCountsMap := utils.MapKeyFromSlice(brandCounts, func(bc BrandCount) (uuid.UUID, BrandCount) {
		return bc.ID, bc
	})

	for _, brand := range brands {
		response := responses.BrandResponse{}.ToBrandResponse(&brand)
		count, ok := brandCountsMap[brand.ID]
		if ok {
			response.NumberOfContracts = count.NumberOfContracts
			response.NumberOfActiveContracts = count.NumberOfActiveContracts
		}
		brandResponses = append(brandResponses, *response)
	}

	return brandResponses, totalCount, nil
}

// GetByID implements iservice.BrandService.
func (b *brandService) GetByID(ctx context.Context, brandID uuid.UUID) (*responses.BrandDetailResponse, error) {
	zap.L().Info("Fetching brand by ID", zap.String("brand_id", brandID.String()))

	conditions := func(db *gorm.DB) *gorm.DB {
		return db.Where("id = ?", brandID)
	}
	var brand *model.Brand
	var err error
	if brand, err = b.BrandRepository.GetByCondition(ctx, conditions, nil); err != nil {
		zap.L().Error("Failed to fetch brand from repository", zap.Error(err))
		return nil, err
	} else if brand == nil {
		zap.L().Warn("Brand not found", zap.String("brand_id", brandID.String()))
		return nil, fmt.Errorf("brand with ID %s not found", brandID.String())
	}

	return responses.BrandDetailResponse{}.ToBrandDetailResponse(brand), nil
}

// UpdateBrand implements iservice.BrandService.
func (b *brandService) UpdateBrand(ctx context.Context, brandID uuid.UUID, request *requests.UpdateBrandRequest) (*responses.BrandResponse, error) {
	zap.L().Info("Updating brand", zap.String("brand_id", brandID.String()), zap.Any("request", request))

	conditions := func(db *gorm.DB) *gorm.DB {
		return db.Where("id = ?", brandID)
	}
	var brand *model.Brand
	var err error
	if brand, err = b.BrandRepository.GetByCondition(ctx, conditions, nil); err != nil {
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
func (b *brandService) UpdateBrandStatus(ctx context.Context, brandID uuid.UUID, status enum.BrandStatus) (*responses.BrandResponse, error) {
	zap.L().Info("Updating brand status", zap.String("brand_id", brandID.String()), zap.String("status", string(status)))

	conditions := func(db *gorm.DB) *gorm.DB {
		return db.Where("id = ?", brandID)
	}
	var brand *model.Brand
	var err error
	if brand, err = b.BrandRepository.GetByCondition(ctx, conditions, nil); err != nil {
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

// CreateBrandWithInActiveUsers implements iservice.BrandService.
func (b *brandService) CreateBrandWithInActiveUsers(
	ctx context.Context,
	uow *irepository.UnitOfWork,
	request *requests.CreateBrandWithUserRequest,
) (*responses.BrandResponse, error) {
	zap.L().Info("Creating new brand with inactive useer", zap.Any("request.CreateBrandRequest", request))
	brandRepo := (*uow).Brands()
	usersRepo := (*uow).Users()

	conditions := func(db *gorm.DB) *gorm.DB {
		return db.Where("name = ? OR contact_email = ?", request.Name, request.ContactEmail)
	}
	if exists, err := brandRepo.Exists(ctx, conditions); err != nil {
		zap.L().Error("Failed to check existing brand", zap.Error(err))
		return nil, err
	} else if exists {
		zap.L().Warn("Brand with the same name already exists", zap.String("name", request.Name))
		return nil, fmt.Errorf("brand with name %s already exists", request.Name)
	}

	// Create an new inactive user with placeholder password
	// The real password will be auto-generated after the admin verifies the creation of the brand and users
	usersModel := &model.User{
		ID:              uuid.New(),
		Username:        utils.ToUsernameString(request.ContactEmail),
		Email:           request.ContactEmail,
		Phone:           request.ContactPhone,
		PasswordHash:    "<placeholder>",
		FullName:        "",
		Role:            enum.UserRoleBrandPartner,
		IsActive:        false,
		ShippingAddress: []model.ShippingAddress{},
		Sessions:        []model.LoggedSession{},
		DateOfBirth:     nil,
	}
	if err := usersRepo.Add(ctx, usersModel); err != nil {
		zap.L().Error("Failed to create inactive user for brand", zap.Error(err))
		return nil, err
	}

	brandModel := &model.Brand{
		ID:                      uuid.New(),
		UserID:                  &usersModel.ID,
		Name:                    request.Name,
		Description:             request.Description,
		ContactEmail:            request.ContactEmail,
		ContactPhone:            request.ContactPhone,
		Address:                 request.Address,
		Website:                 request.Website,
		LogoURL:                 request.LogoURL,
		Status:                  enum.BrandStatusInactive,
		TaxNumber:               request.TaxNumber,
		RepresentativeName:      request.RepresentativeName,
		RepresentativeRole:      request.RepresentativeRole,
		RepresentativeEmail:     request.RepresentativeEmail,
		RepresentativePhone:     request.RepresentativePhone,
		RepresentativeCitizenID: request.RepresentativeCitizenID,
	}
	if err := brandRepo.Add(ctx, brandModel); err != nil {
		zap.L().Error("Failed to create brand", zap.Error(err))
		return nil, err
	}

	return responses.BrandResponse{}.ToBrandResponse(brandModel), nil

}

func NewBrandService(brandRepository irepository.GenericRepository[model.Brand]) iservice.BrandService {
	return &brandService{BrandRepository: brandRepository}
}
