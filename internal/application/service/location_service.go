package service

import (
	"context"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/application/interfaces/iservice"
	"core-backend/internal/domain/model"
	gormrepository "core-backend/internal/infrastructure/gorm_repository"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type locationService struct {
	shippingAddressRepo irepository.GenericRepository[model.ShippingAddress]
	userRepo            irepository.GenericRepository[model.User]
	provinceRepo        irepository.GenericRepository[model.Province]
	districtRepo        irepository.GenericRepository[model.District]
	wardRepo            irepository.GenericRepository[model.Ward]
}

func (l locationService) GetProvinces() ([]responses.ProvinceResponse, error) {
	//TODO: Switch to local
	ctx := context.Background()
	filter := func(db *gorm.DB) *gorm.DB { return db.Order("name ASC") }
	provinces, _, err := l.provinceRepo.GetAll(ctx, filter, nil, 0, 0)
	if err != nil {
		zap.L().Error(fmt.Sprintf("Failed to get provinces: %s", err))
		return nil, err
	}
	out := make([]responses.ProvinceResponse, 0, len(provinces))
	for _, p := range provinces {
		out = append(out, mapProvinceToResponse(p))
	}
	return out, nil
}

func (l locationService) GetDistrictsByProvinceID(provinceID int) ([]responses.DistrictResponse, error) {
	ctx := context.Background()
	filter := func(db *gorm.DB) *gorm.DB { return db.Where("province_id = ?", provinceID).Order("name ASC") }
	districts, _, err := l.districtRepo.GetAll(ctx, filter, nil, 0, 0)
	if err != nil {
		zap.L().Error(fmt.Sprintf("Failed to get districts: %s", err))
		return nil, err
	}
	out := make([]responses.DistrictResponse, 0, len(districts))
	for _, d := range districts {
		out = append(out, mapDistrictToResponse(d))
	}
	return out, nil
}

func (l locationService) GetWardsByDistrictID(districtID int) ([]responses.WardResponse, error) {
	//TODO: Switch to local
	ctx := context.Background()
	filter := func(db *gorm.DB) *gorm.DB { return db.Where("district_id = ?", districtID).Order("name ASC") }
	wards, _, err := l.wardRepo.GetAll(ctx, filter, nil, 0, 0)
	if err != nil {
		zap.L().Error(fmt.Sprintf("Failed to get wards: %s", err))
		return nil, err
	}
	out := make([]responses.WardResponse, 0, len(wards))
	for _, w := range wards {
		out = append(out, mapWardToResponse(w))
	}
	return out, nil
}

func (l locationService) InputUserAddress(userID uuid.UUID, addressReq requests.InputAddressRequest) (*model.ShippingAddress, error) {
	ctx := context.Background()
	// check user exists
	isExisted, _ := l.userRepo.ExistsByID(ctx, userID)
	if !isExisted {
		return nil, fmt.Errorf("user not found")
	}

	// Prepare zero-value containers to pass to ToModel
	var ward model.Ward
	var dist model.District
	var province model.Province

	// Validate GHN location fields when provided
	if addressReq.GhnWardCode != "" {
		// Fetch ward with its district
		wardInclude := []string{"District"}
		w, err := l.wardRepo.GetByID(ctx, addressReq.GhnWardCode, wardInclude)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, fmt.Errorf("ward with code '%s' not found", addressReq.GhnWardCode)
			}
			return nil, fmt.Errorf("failed to fetch ward %s: %w", addressReq.GhnWardCode, err)
		}
		if w == nil {
			return nil, fmt.Errorf("ward with code '%s' not found", addressReq.GhnWardCode)
		}
		ward = *w

		// If user supplied district, ensure it matches the ward's parent
		if addressReq.GhnDistrictID != 0 && ward.DistrictID != addressReq.GhnDistrictID {
			return nil, fmt.Errorf("ward '%s' (code=%s) does not belong to district %d", ward.Name, ward.Code, addressReq.GhnDistrictID)
		}

		// Fetch district (including province if requested)
		if addressReq.GhnProvinceID != 0 {
			dInclude := []string{"Province"}
			d, err := l.districtRepo.GetByID(ctx, ward.DistrictID, dInclude)
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return nil, fmt.Errorf("district %d not found", ward.DistrictID)
				}
				return nil, fmt.Errorf("failed to fetch district %d: %w", ward.DistrictID, err)
			}
			if d == nil {
				return nil, fmt.Errorf("district %d not found", ward.DistrictID)
			}
			dist = *d
			// ensure province matches
			if dist.ProvinceID != addressReq.GhnProvinceID {
				return nil, fmt.Errorf("ward '%s' (code=%s) belongs to district %d which does not belong to province %d", ward.Name, ward.Code, dist.ID, addressReq.GhnProvinceID)
			}
			province = dist.Province
		} else {
			// fetch district without province to populate district name
			d, err := l.districtRepo.GetByID(ctx, ward.DistrictID, nil)
			if err != nil {
				return nil, fmt.Errorf("failed to fetch district %d: %w", ward.DistrictID, err)
			}
			if d == nil {
				return nil, fmt.Errorf("district %d not found", ward.DistrictID)
			}
			dist = *d
		}

	} else {
		// Ward not provided. If district provided, validate it and optional province
		if addressReq.GhnDistrictID != 0 {
			d, err := l.districtRepo.GetByID(ctx, addressReq.GhnDistrictID, []string{"Province"})
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return nil, fmt.Errorf("district %d not found", addressReq.GhnDistrictID)
				}
				return nil, fmt.Errorf("failed to fetch district %d: %w", addressReq.GhnDistrictID, err)
			}
			if d == nil {
				return nil, fmt.Errorf("district %d not found", addressReq.GhnDistrictID)
			}
			dist = *d
			if addressReq.GhnProvinceID != 0 && dist.ProvinceID != addressReq.GhnProvinceID {
				return nil, fmt.Errorf("district %d does not belong to province %d", dist.ID, addressReq.GhnProvinceID)
			}
			if addressReq.GhnProvinceID != 0 {
				province = dist.Province
			}
		} else if addressReq.GhnProvinceID != 0 {
			p, err := l.provinceRepo.GetByID(ctx, addressReq.GhnProvinceID, nil)
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return nil, fmt.Errorf("province %d not found", addressReq.GhnProvinceID)
				}
				return nil, fmt.Errorf("failed to fetch province %d: %w", addressReq.GhnProvinceID, err)
			}
			if p == nil {
				return nil, fmt.Errorf("province %d not found", addressReq.GhnProvinceID)
			}
			province = *p
		}
	}

	// Persist address inside a DB transaction
	var persistedModel *model.ShippingAddress
	err := l.shippingAddressRepo.DB().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Unset other default addresses if needed
		if addressReq.IsDefault {
			if err := tx.Model(&model.ShippingAddress{}).
				Where("user_id = ? AND is_default = ?", userID, true).
				Update("is_default", false).Error; err != nil {
				return err
			}
		}

		// Add new address
		persistedModel = addressReq.ToModel(userID, ward, dist, province)
		if err := tx.Create(persistedModel).Error; err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		zap.L().Error(fmt.Sprintf("Failed to persist new address: %s", err))
		return nil, err
	}

	return persistedModel, nil
}

func (l locationService) SetAddressAsDefault(userID string, addressID string) error {
	ctx := context.Background()
	// check user exists
	isExisted, _ := l.userRepo.ExistsByID(ctx, userID)
	if !isExisted {
		return fmt.Errorf("user not found")
	}

	err := l.shippingAddressRepo.DB().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// set other addresses to non-default
		if err := tx.Model(&model.ShippingAddress{}).
			Where("user_id = ? AND is_default = ?", userID, true).
			Update("is_default", false).Error; err != nil {
			return err
		}
		// set the specified address to default
		if err := tx.Model(&model.ShippingAddress{}).
			Where("id = ? AND user_id = ?", addressID, userID).
			Update("is_default", true).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		zap.L().Error(fmt.Sprintf("Failed to set address as default: %s", err))
		return err
	}

	return nil
}

func (l locationService) GetUserAddresses(userID uuid.UUID) ([]model.ShippingAddress, error) {
	ctx := context.Background()

	filter := func(db *gorm.DB) *gorm.DB {
		return db.Where("user_id = ?", userID).
			Order("is_default DESC").
			Order("updated_at DESC")
	}
	addresses, _, err := l.shippingAddressRepo.GetAll(ctx, filter, nil, 0, 0)

	if err != nil {
		zap.L().Error(fmt.Sprintf("Failed to get user addresses: %s", err))
		return nil, err
	}

	return addresses, nil
}

func NewLocationService(dbRegistry *gormrepository.DatabaseRegistry) iservice.LocationService {
	return &locationService{
		shippingAddressRepo: dbRegistry.ShippingAddressRepository,
		userRepo:            dbRegistry.UserRepository,
		provinceRepo:        dbRegistry.ProvinceRepository,
		districtRepo:        dbRegistry.DistrictRepository,
		wardRepo:            dbRegistry.WardRepository,
	}
}

// Mapping helpers
func mapProvinceToResponse(p model.Province) responses.ProvinceResponse {
	return responses.ProvinceResponse{
		ProvinceID:   p.ID,
		ProvinceName: p.Name,
		CountryID:    p.CountryID,
		Code:         p.Code,
		RegionID:     p.RegionID,
		RegionCPN:    p.RegionCPN,
		GeneralLocationResponse: responses.GeneralLocationResponse{
			IsEnable:     p.IsEnable,
			CanUpdateCOD: p.CanUpdateCOD,
			Status:       p.Status,
		},
	}
}

func mapDistrictToResponse(d model.District) responses.DistrictResponse {
	return responses.DistrictResponse{
		DistrictID:     d.ID,
		ProvinceID:     d.ProvinceID,
		DistrictName:   d.Name,
		Code:           d.Code,
		Type:           d.Type,
		SupportType:    d.SupportType,
		PickType:       d.PickType,
		DeliverType:    d.DeliverType,
		GovernmentCode: d.GovernmentCode,
		GeneralLocationResponse: responses.GeneralLocationResponse{
			IsEnable:     d.IsEnable,
			CanUpdateCOD: d.CanUpdateCOD,
			Status:       d.Status,
		},
	}
}

func mapWardToResponse(w model.Ward) responses.WardResponse {
	return responses.WardResponse{
		WardCode:       w.Code,
		DistrictID:     w.DistrictID,
		WardName:       w.Name,
		SupportType:    w.SupportType,
		PickType:       w.PickType,
		DeliverType:    w.DeliverType,
		GovernmentCode: w.GovernmentCode,
		GeneralLocationResponse: responses.GeneralLocationResponse{
			IsEnable:     w.IsEnable,
			CanUpdateCOD: w.CanUpdateCOD,
			Status:       w.Status,
		},
	}
}
