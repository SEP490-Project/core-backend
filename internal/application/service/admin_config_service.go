package service

import (
	"context"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/application/interfaces/iservice"
	"core-backend/internal/domain/model"
	"fmt"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

type AdminConfigService struct {
	configRepo irepository.GenericRepository[model.Config]
}

// GetAllConfig implements iservice.AdminConfigService.
func (a *AdminConfigService) GetAllConfig(ctx context.Context) ([]responses.AdminConfigResponse, error) {
	zap.L().Info("Fetching all admin configurations")

	configs, _, err := a.configRepo.GetAll(ctx, nil, nil, 0, 0)
	if err != nil {
		zap.L().Error("Failed to fetch configurations", zap.Error(err))
		return nil, err
	}

	return responses.AdminConfigResponse{}.ToResponseList(configs), nil
}

// GetConfigByKey implements iservice.AdminConfigService.
func (a *AdminConfigService) GetConfigByKey(ctx context.Context, key string) (*responses.AdminConfigResponse, error) {
	zap.L().Info("Fetching configuration by key", zap.String("key", key))

	query := func(db *gorm.DB) *gorm.DB {
		return db.Where("key = ?", key)
	}
	config, err := a.configRepo.GetByCondition(ctx, query, nil)
	if err != nil {
		zap.L().Error("Failed to fetch configuration", zap.Error(err))
		return nil, err
	} else if config == nil {
		zap.L().Warn("Configuration not found", zap.String("key", key))
		return nil, fmt.Errorf("configuration with key %s not found", key)
	}

	return responses.AdminConfigResponse{}.ToResponse(config), nil
}

// GetConfigValueByKey implements iservice.AdminConfigService.
func (a *AdminConfigService) GetConfigValueByKey(ctx context.Context, key string) (string, error) {
	zap.L().Info("Fetching configuration value by key", zap.String("key", key))

	query := func(db *gorm.DB) *gorm.DB {
		return db.Where("key = ?", key)
	}
	config, err := a.configRepo.GetByCondition(ctx, query, nil)
	if err != nil {
		zap.L().Error("Failed to fetch configuration", zap.Error(err))
		return "", err
	} else if config == nil {
		zap.L().Warn("Configuration not found", zap.String("key", key))
		return "", fmt.Errorf("configuration with key %s not found", key)
	}

	return config.Value, nil
}

// UpdateConfigByKey implements iservice.AdminConfigService.
func (a *AdminConfigService) UpdateConfigByKey(ctx context.Context, key string, value string, uow irepository.UnitOfWork) error {
	zap.L().Debug("Updating configuration by key", zap.String("key", key), zap.String("value", value))

	configRepo := uow.Configs()

	filter := func(db *gorm.DB) *gorm.DB {
		return db.Where("key = ?", key)
	}
	err := configRepo.UpdateByCondition(ctx, filter, map[string]interface{}{"value": value})
	if err != nil {
		zap.L().Error("Failed to update configuration", zap.Error(err))
		return err
	}

	return nil
}

// UpdateConfigs implements iservice.AdminConfigService.
func (a *AdminConfigService) UpdateConfigs(ctx context.Context, configs map[string]string, uow irepository.UnitOfWork) error {
	zap.L().Debug("Updating multiple configurations", zap.Any("configs", configs))

	for key, value := range configs {
		if err := a.UpdateConfigByKey(ctx, key, value, uow); err != nil {
			zap.L().Error("Failed to update configuration", zap.Error(err))
			return err
		}
	}

	return nil
}

func NewAdminConfigService(configRepo irepository.GenericRepository[model.Config]) iservice.AdminConfigService {
	return &AdminConfigService{
		configRepo: configRepo,
	}
}
