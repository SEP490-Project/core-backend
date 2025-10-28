package service

import (
	"bytes"
	"context"
	"core-backend/config"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/application/interfaces/iservice"
	"core-backend/internal/domain/model"
	gormrepository "core-backend/internal/infrastructure/gorm_repository"
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"io"
	"net/http"

	"go.uber.org/zap"
)

type locationService struct {
	shippingAddressRepo irepository.GenericRepository[model.ShippingAddress]
	userRepo            irepository.GenericRepository[model.User]
}

func (l locationService) GetProvinces() ([]responses.ProvinceResponse, error) {
	cfg := config.GetAppConfig()
	url := cfg.GHN.BaseURL + "/province"
	token := cfg.GHN.Token
	return doRequest[responses.ProvinceResponse]("GET", url, token, nil)
}

func (l locationService) GetDistrictsByProvinceID(provinceID int) ([]responses.DistrictResponse, error) {
	cfg := config.GetAppConfig()
	url := fmt.Sprintf("%s/district?province_id=%d", cfg.GHN.BaseURL, provinceID)
	token := cfg.GHN.Token
	return doRequest[responses.DistrictResponse]("GET", url, token, nil)
}

func (l locationService) GetWardsByDistrictID(districtID int) ([]responses.WardResponse, error) {
	loadedCfg := config.GetAppConfig()
	url := fmt.Sprintf("%s/ward?district_id=%d", loadedCfg.GHN.BaseURL, districtID)
	token := loadedCfg.GHN.Token
	return doRequest[responses.WardResponse]("GET", url, token, nil)
}

func doRequest[T any](method, url, token string, body any) ([]T, error) {
	var buf io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			zap.L().Error(fmt.Sprintf("Failed to marshal body: %s", err))
			return nil, err
		}
		buf = bytes.NewBuffer(jsonBody)
	}

	req, err := http.NewRequest(method, url, buf)
	if err != nil {
		zap.L().Error(fmt.Sprintf("Failed to create request: %s", err))
		return nil, err
	}
	req.Header.Add("Token", token)
	if body != nil {
		req.Header.Add("Content-Type", "application/json")
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		zap.L().Error(fmt.Sprintf("Failed to send request: %s", err))
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		zap.L().Error(fmt.Sprintf("Failed to read response: %s", err))
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		zap.L().Warn(fmt.Sprintf("GHN API returned non-200: %s\nBody: %s", resp.Status, string(respBody)))
	}

	var result responses.GHNAPIResponse[T]
	if err := json.Unmarshal(respBody, &result); err != nil {
		zap.L().Error(fmt.Sprintf("Failed to unmarshal response: %s", err))
		return nil, err
	}
	return result.Data, nil
}

func (l locationService) InputUserAddress(userID uuid.UUID, addressReq requests.InputAddressRequest) (*model.ShippingAddress, error) {
	ctx := context.Background()
	// check user exists
	isExisted, _ := l.userRepo.ExistsByID(ctx, userID)
	if !isExisted {
		return nil, fmt.Errorf("user not found")
	}

	//Other address validation
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
		persistedModel = addressReq.ToModel(userID)
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
	}
}
