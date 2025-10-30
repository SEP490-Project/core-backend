package service

import (
	"context"
	"core-backend/config"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/application/interfaces/iservice"
	"core-backend/internal/domain/model"
	"core-backend/pkg/utils"
	"fmt"
	"reflect"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

type AdminConfigService struct {
	adminConfig *config.AdminConfig
	configRepo  irepository.GenericRepository[model.Config]
}

// GetAllConfig implements iservice.AdminConfigService.
func (a *AdminConfigService) GetAllConfig(ctx context.Context) ([]responses.AdminConfigResponse, error) {
	zap.L().Info("Fetching all admin configurations")

	var res []responses.AdminConfigResponse
	val := reflect.ValueOf(a.adminConfig).Elem()

	for i := 0; i < val.NumField(); i++ {
		typeField := val.Type().Field(i)
		key := utils.ToSnakeCase(typeField.Name)
		value := fmt.Sprintf("%v", val.Field(i).Interface())
		res = append(res, responses.AdminConfigResponse{
			Key:   key,
			Value: value,
		})
	}

	return res, nil
}

// GetConfigByKey implements iservice.AdminConfigService.
func (a *AdminConfigService) GetConfigByKey(ctx context.Context, key string) (*responses.AdminConfigResponse, error) {
	zap.L().Info("Fetching configuration by key", zap.String("key", key))

	structKey := utils.ToStructFieldName(key)
	reflectVal := reflect.ValueOf(a.adminConfig).Elem().FieldByName(structKey)

	if !reflectVal.IsValid() {
		zap.L().Warn("Configuration not found", zap.String("key", structKey))
		return nil, fmt.Errorf("configuration with key %s not found", key)
	}

	return &responses.AdminConfigResponse{
		Key:   key,
		Value: fmt.Sprintf("%v", reflectVal.Interface()),
	}, nil
}

// GetConfigValueByKey implements iservice.AdminConfigService.
func (a *AdminConfigService) GetConfigValueByKey(ctx context.Context, key string) (string, error) {
	zap.L().Info("Fetching configuration value by key", zap.String("key", key))

	structKey := utils.ToStructFieldName(key)
	reflectVal := reflect.ValueOf(a.adminConfig).Elem().FieldByName(structKey)

	if !reflectVal.IsValid() {
		zap.L().Warn("Configuration not found", zap.String("key", structKey))
		return "", fmt.Errorf("configuration with key %s not found", key)
	}

	return fmt.Sprintf("%v", reflectVal.Interface()), nil
}

// GetConfigValuesByKeys implements iservice.AdminConfigService.
func (a *AdminConfigService) GetConfigValuesByKeys(ctx context.Context, keys []string) (map[string]string, error) {
	zap.L().Info("Fetching configuration values by keys", zap.Any("keys", keys))

	var values = make(map[string]string, len(keys))
	for _, key := range keys {
		structKey := utils.ToStructFieldName(key)
		reflectVal := reflect.ValueOf(a.adminConfig).Elem().FieldByName(structKey)
		if !reflectVal.IsValid() {
			zap.L().Warn("Configuration not found", zap.String("key", structKey))
			continue
		}
		values[utils.ToSnakeCase(key)] = fmt.Sprintf("%v", reflectVal.Interface())
	}

	return values, nil
}

// UpdateConfigByKey implements iservice.AdminConfigService.
func (a *AdminConfigService) UpdateConfigByKey(ctx context.Context, key string, value string, uow irepository.UnitOfWork) error {
	zap.L().Debug("Updating configuration by key", zap.String("key", key), zap.String("value", value))

	if err := utils.SetStringToReflectValue(a.adminConfig, key, value); err != nil {
		return fmt.Errorf("validation failed for key '%s': %w", key, err)
	}

	configRepo := uow.AdminConfigs()
	filter := func(db *gorm.DB) *gorm.DB {
		return db.Where("key = ?", key)
	}
	if err := configRepo.UpdateByCondition(ctx, filter, map[string]any{"value": value}); err != nil {
		zap.L().Error("Failed to update configuration in database", zap.Error(err))
		// Note: At this point, the in-memory config is updated but the DB failed.
		// A more robust solution might revert the in-memory change or use a two-phase commit.
		// For now, we return the DB error.
		return err
	}

	// Re-apply the change to the in-memory struct after successful DB operation
	// This ensures consistency. The value from the line above is now stale.
	_ = utils.SetStringToReflectValue(a.adminConfig, key, value)

	return nil
}

// UpdateConfigs implements iservice.AdminConfigService.
func (a *AdminConfigService) UpdateConfigs(ctx context.Context, configs map[string]string, uow irepository.UnitOfWork) error {
	zap.L().Debug("Updating multiple configurations", zap.Any("configs", configs))

	// 1. Validation Phase: Validate all inputs before making any changes.
	for key, value := range configs {
		if err := utils.SetStringToReflectValue(a.adminConfig, key, value); err != nil {
			return fmt.Errorf("validation failed for key '%s': %w", key, err)
		}
	}

	// 2. Database Update Phase: Update all values in the database within a transaction.
	configRepo := uow.AdminConfigs()
	for key, value := range configs {
		filter := func(db *gorm.DB) *gorm.DB {
			return db.Where("key = ?", key)
		}
		if err := configRepo.UpdateByCondition(ctx, filter, map[string]any{"value": value}); err != nil {
			zap.L().Error("Failed to update configuration in database", zap.Error(err), zap.String("key", key))
			return err // The Unit of Work should handle the transaction rollback.
		}
	}

	// 3. In-Memory Update Phase: After successful database updates, update the live config struct.
	for key, value := range configs {
		_ = utils.SetStringToReflectValue(a.adminConfig, key, value)
	}

	return nil
}

func NewAdminConfigService(adminConfig *config.AdminConfig, configRepo irepository.GenericRepository[model.Config]) iservice.AdminConfigService {
	return &AdminConfigService{
		adminConfig: adminConfig,
		configRepo:  configRepo,
	}
}
