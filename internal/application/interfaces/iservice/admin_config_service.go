package iservice

import (
	"context"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/irepository"
)

type AdminConfigService interface {
	// GetConfigByKey retrieves the admin configuration identified by its key.
	GetConfigByKey(ctx context.Context, key string) (*responses.AdminConfigResponse, error)

	// GetConfigValueByKey retrieves the value of a specific admin configuration identified by its key.
	GetConfigValueByKey(ctx context.Context, key string) (string, error)

	// GetConfigValuesByKeys retrieves the values of multiple admin configurations identified by their keys.
	GetConfigValuesByKeys(ctx context.Context, keys []string) (map[string]string, error)

	// GetConfigByKey retrieves all of the admin configurations stored in the system.
	GetAllConfig(ctx context.Context) ([]responses.AdminConfigResponse, error)

	// GetConfigByKey retrieves the value of a specific admin configuration identified by its key.
	UpdateConfigByKey(ctx context.Context, key string, value string, uow irepository.UnitOfWork) error

	// UpdateConfigs updates multiple admin configurations at once.
	UpdateConfigs(ctx context.Context, configs map[string]string, uow irepository.UnitOfWork) error
}
