package service

import (
	"context"
	"core-backend/config"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/application/interfaces/iservice"
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
	"core-backend/pkg/tiptap"
	"core-backend/pkg/utils"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type AdminConfigService struct {
	adminConfig *config.AdminConfig
	configRepo  irepository.GenericRepository[model.Config]
	listeners   []func()
}

// RegisterListener implements iservice.AdminConfigService.
func (a *AdminConfigService) RegisterListener(listener func()) {
	a.listeners = append(a.listeners, listener)
}

// notifyListeners calls all registered listeners.
func (a *AdminConfigService) notifyListeners() {
	for _, listener := range a.listeners {
		go listener() // Run in goroutine to avoid blocking
	}
}

// GetAllConfig implements iservice.AdminConfigService.
func (a *AdminConfigService) GetAllConfig(ctx context.Context) ([]responses.AdminConfigResponse, error) {
	zap.L().Info("Fetching all admin configurations")

	// 1. Fetch all configs from database
	dbConfigs, _, err := a.configRepo.GetAll(ctx, nil, nil, 1000, 1)
	if err != nil {
		zap.L().Error("Failed to fetch configs from database", zap.Error(err))
		// Continue with in-memory config if DB fails? Or return error?
		// Let's return error to be safe, or log and continue.
		// Given the requirement, we probably want to mix them.
		return nil, err
	}

	// Map for quick lookup
	dbConfigMap := make(map[string]model.Config)
	for _, cfg := range dbConfigs {
		dbConfigMap[cfg.Key] = cfg
	}

	var res []responses.AdminConfigResponse
	val := reflect.ValueOf(a.adminConfig).Elem()

	for i := 0; i < val.NumField(); i++ {
		typeField := val.Type().Field(i)
		key := utils.ToSnakeCase(typeField.Name)

		// Default from struct
		value := fmt.Sprintf("%v", val.Field(i).Interface())
		valueType := determineValueType(typeField, value)

		response := responses.AdminConfigResponse{
			Key:       key,
			Value:     value,
			ValueType: valueType,
		}

		// Override/Enrich with DB data if exists
		if dbConfig, exists := dbConfigMap[key]; exists {
			response.ID = dbConfig.ID.String()
			response.Value = dbConfig.Value // DB value takes precedence
			value = dbConfig.Value
			response.ValueType = dbConfig.ValueType
			response.Description = dbConfig.Description
			response.CreatedAt = utils.FormatLocalTime(&dbConfig.CreatedAt, utils.TimeFormat)
			response.UpdatedAt = utils.FormatLocalTime(&dbConfig.UpdatedAt, utils.TimeFormat)
			if dbConfig.UpdatedByID != uuid.Nil {
				response.UpdatedByID = dbConfig.UpdatedByID.String()
			}
		}
		if response.ValueType == enum.ConfigValueTypeTipTapJSON ||
			response.ValueType == enum.ConfigValueTypeJSON {
			temp := strings.TrimSpace(value)
			response.Value, err = utils.JSONStrToMap(value)
			if err != nil {
				response.Value = temp // Fallback to raw string if JSON parsing fails
			}
		}

		res = append(res, response)
	}

	return res, nil
}

// GetConfigByKey implements iservice.AdminConfigService.
func (a *AdminConfigService) GetConfigByKey(ctx context.Context, key string) (*responses.AdminConfigResponse, error) {
	zap.L().Info("Fetching configuration by key", zap.String("key", key))

	structKey := utils.ToStructFieldName(key)
	reflectVal := reflect.ValueOf(a.adminConfig).Elem().FieldByName(structKey)

	if !reflectVal.IsValid() {
		zap.L().Warn("Configuration not found in struct", zap.String("key", structKey))
		return nil, fmt.Errorf("configuration with key %s not found", key)
	}

	// Default from struct
	value := fmt.Sprintf("%v", reflectVal.Interface())
	typeField, _ := reflect.TypeOf(a.adminConfig).Elem().FieldByName(structKey)
	valueType := determineValueType(typeField, value)

	response := &responses.AdminConfigResponse{
		Key:       key,
		Value:     value,
		ValueType: valueType,
	}

	// Try to fetch from DB
	dbConfig, err := a.configRepo.GetByCondition(ctx, func(db *gorm.DB) *gorm.DB {
		return db.Where("key = ?", key)
	}, nil)

	if err == nil && dbConfig != nil {
		response.ID = dbConfig.ID.String()
		response.Value = dbConfig.Value
		value = dbConfig.Value
		response.ValueType = dbConfig.ValueType
		response.Description = dbConfig.Description
		response.CreatedAt = dbConfig.CreatedAt.Format(utils.TimeFormat)
		response.UpdatedAt = dbConfig.UpdatedAt.Format(utils.TimeFormat)
		if dbConfig.UpdatedByID != uuid.Nil {
			response.UpdatedByID = dbConfig.UpdatedByID.String()
		}
	}

	if response.ValueType == enum.ConfigValueTypeTipTapJSON ||
		response.ValueType == enum.ConfigValueTypeJSON {
		temp := strings.TrimSpace(value)
		response.Value, err = utils.JSONStrToMap(value)
		if err != nil {
			response.Value = temp // Fallback to raw string if JSON parsing fails
		}
	}

	return response, nil
}

// GetConfigValueByKey implements iservice.AdminConfigService.
func (a *AdminConfigService) GetConfigValueByKey(ctx context.Context, key string) (string, error) {
	// For simple value retrieval, we can check DB first, then struct
	// But since UpdateConfigByKey updates both, checking struct is faster and should be consistent.
	// However, to be strictly "live" from DB if other instances updated it:

	dbConfig, err := a.configRepo.GetByCondition(ctx, func(db *gorm.DB) *gorm.DB {
		return db.Where("key = ?", key)
	}, nil)

	if err == nil && dbConfig != nil {
		return dbConfig.Value, nil
	}

	// Fallback to struct
	structKey := utils.ToStructFieldName(key)
	reflectVal := reflect.ValueOf(a.adminConfig).Elem().FieldByName(structKey)

	if !reflectVal.IsValid() {
		return "", fmt.Errorf("configuration with key %s not found", key)
	}

	return fmt.Sprintf("%v", reflectVal.Interface()), nil
}

// GetConfigValuesByKeys implements iservice.AdminConfigService.
func (a *AdminConfigService) GetConfigValuesByKeys(ctx context.Context, keys []string) (map[string]string, error) {
	zap.L().Info("Fetching configuration values by keys", zap.Any("keys", keys))

	// Fetch from DB
	dbConfigs, _, err := a.configRepo.GetAll(ctx, func(db *gorm.DB) *gorm.DB {
		return db.Where("key IN ?", keys)
	}, nil, len(keys), 1)

	if err != nil {
		return nil, err
	}

	dbConfigMap := make(map[string]string)
	for _, cfg := range dbConfigs {
		dbConfigMap[cfg.Key] = cfg.Value
	}

	var values = make(map[string]string, len(keys))
	for _, key := range keys {
		// Check DB first
		if val, ok := dbConfigMap[key]; ok {
			values[utils.ToSnakeCase(key)] = val
			continue
		}

		// Fallback to struct
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
func (a *AdminConfigService) UpdateConfigByKey(ctx context.Context, key string, value string, uow irepository.UnitOfWork, updatedBy uuid.UUID) error {
	zap.L().Debug("Updating configuration by key",
		zap.String("key", key),
		zap.String("value", value),
		zap.String("updated_by", updatedBy.String()))

	// 1. Validate against struct
	structKey := utils.ToStructFieldName(key)
	field, ok := reflect.TypeOf(a.adminConfig).Elem().FieldByName(structKey)
	if !ok {
		return fmt.Errorf("invalid config key: %s", key)
	}

	if err := utils.SetStringToReflectValue(a.adminConfig, key, value); err != nil {
		return fmt.Errorf("validation failed for key '%s': %w", key, err)
	}

	// 2. Upsert to DB
	configRepo := uow.AdminConfigs()

	// Check if exists
	existing, err := configRepo.GetByCondition(ctx, func(db *gorm.DB) *gorm.DB {
		return db.Where("key = ?", key)
	}, nil)

	if err != nil && err != gorm.ErrRecordNotFound {
		return err
	}

	if existing != nil {
		// Update
		if err := configRepo.UpdateByCondition(ctx, func(db *gorm.DB) *gorm.DB {
			return db.Where("key = ?", key)
		}, map[string]any{"value": value, "updated_by": updatedBy}); err != nil {
			zap.L().Error("Failed to update configuration in database", zap.Error(err))
			return err
		}
	} else {
		// Create
		newConfig := &model.Config{
			Key:       key,
			Value:     value,
			ValueType: determineValueType(field, value),
		}
		if err := configRepo.Add(ctx, newConfig); err != nil {
			zap.L().Error("Failed to create configuration in database", zap.Error(err))
			return err
		}
	}

	// 3. Update in-memory
	_ = utils.SetStringToReflectValue(a.adminConfig, key, value)

	// 4. Notify listeners
	a.notifyListeners()

	return nil
}

// UpdateConfigs implements iservice.AdminConfigService.
func (a *AdminConfigService) UpdateConfigs(ctx context.Context, configs map[string]string, uow irepository.UnitOfWork, updatedBy uuid.UUID) error {
	zap.L().Debug("Updating multiple configurations",
		zap.Any("configs", configs),
		zap.String("updated_by", updatedBy.String()))

	// 1. Validation Phase
	for key, value := range configs {
		if err := utils.SetStringToReflectValue(a.adminConfig, key, value); err != nil {
			return fmt.Errorf("validation failed for key '%s': %w", key, err)
		}
	}

	// 2. Database Update Phase (Upsert)
	configRepo := uow.AdminConfigs()

	// Get existing configs to know which to update vs create
	keys := make([]string, 0, len(configs))
	for k := range configs {
		keys = append(keys, k)
	}

	existingConfigs, _, err := configRepo.GetAll(ctx, func(db *gorm.DB) *gorm.DB {
		return db.Where("key IN ?", keys)
	}, nil, len(keys), 1)
	if err != nil {
		return err
	}

	existingMap := make(map[string]*model.Config)
	for _, cfg := range existingConfigs {
		// Need to take address of cfg, but range var is reused.
		// But GetAll returns []model.Config (values).
		// So we need to be careful.
		// Actually, let's just store the ID or bool.
		// Wait, GetAll returns []T.
		// Let's just map key -> bool
		existingMap[cfg.Key] = &model.Config{ID: cfg.ID} // Just need to know it exists
	}

	reflectConfig := reflect.TypeOf(a.adminConfig).Elem()
	for key, value := range configs {
		if _, exists := existingMap[key]; exists {
			// Update
			if err := configRepo.UpdateByCondition(ctx, func(db *gorm.DB) *gorm.DB {
				return db.Where("key = ?", key)
			}, map[string]any{"value": value, "updated_by": updatedBy}); err != nil {
				return err
			}
		} else {
			// Create
			structKey := utils.ToStructFieldName(key)
			field, _ := reflectConfig.FieldByName(structKey)

			newConfig := &model.Config{
				Key:       key,
				Value:     value,
				ValueType: determineValueType(field, value),
			}
			if err := configRepo.Add(ctx, newConfig); err != nil {
				return err
			}
		}
	}

	// 3. In-Memory Update Phase
	for key, value := range configs {
		_ = utils.SetStringToReflectValue(a.adminConfig, key, value)
	}

	// 4. Notify listeners
	a.notifyListeners()

	return nil
}

func determineValueType(sf reflect.StructField, value string) enum.ConfigValueType {
	// Check for explicit tag first
	if tag := sf.Tag.Get("type"); tag == "textarea" {
		return enum.ConfigValueTypeTextArea
	}

	switch sf.Type.Kind() {
	case reflect.String:
		value = strings.TrimSpace(value)
		if strings.HasPrefix(value, "{") || strings.HasPrefix(value, "[") {
			if tiptap.IsValidTipTapJSON([]byte(value)) {
				return enum.ConfigValueTypeTipTapJSON
			}
			return enum.ConfigValueTypeJSON
		} else if strings.Contains(value, "\n") || len(value) > 255 {
			return enum.ConfigValueTypeTextArea
		}
		return enum.ConfigValueTypeString
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return enum.ConfigValueTypeNumber
	case reflect.Bool:
		return enum.ConfigValueTypeBoolean
	case reflect.Slice, reflect.Array:
		return enum.ConfigValueTypeArray
	case reflect.Struct:
		if sf.Type == reflect.TypeOf(time.Time{}) {
			return enum.ConfigValueTypeTime
		}
		return enum.ConfigValueTypeJSON
	case reflect.Map:
		return enum.ConfigValueTypeJSON
	default:
		return enum.ConfigValueTypeString
	}
}

func NewAdminConfigService(adminConfig *config.AdminConfig, configRepo irepository.GenericRepository[model.Config]) iservice.AdminConfigService {
	return &AdminConfigService{
		adminConfig: adminConfig,
		configRepo:  configRepo,
	}
}
