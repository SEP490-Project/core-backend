// Package infrastructure provides the InfrastructureRegistry struct that holds various infrastructure services.
package infrastructure

import (
	"context"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/application/interfaces/iservice_third_party"
	"core-backend/internal/domain/model"
	gormrepository "core-backend/internal/infrastructure/gorm_repository"
	"core-backend/internal/infrastructure/persistence"
	"core-backend/internal/infrastructure/rabbitmq"
	"core-backend/internal/infrastructure/service"
	"core-backend/internal/infrastructure/third_party_repository"
	"fmt"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

type InfrastructureRegistry struct {
	DB                *gorm.DB
	ThirdPartyStorage *third_party_repository.ThirdPartyStorageRegistry
	UnitOfWork        irepository.UnitOfWork
	ValkeyCache       *persistence.ValkeyCache
	RabbitMQ          *rabbitmq.RabbitMQ
	PayOsService      iservice_third_party.PayOSService
}

func NewInfrastructureRegistry(db *gorm.DB, s3Bucket *persistence.S3Bucket, s3StreamBucket *persistence.S3StreamingBucket) *InfrastructureRegistry {
	zap.L().Info("Initializing infrastructure registry")

	registry := &InfrastructureRegistry{
		DB:         db,
		UnitOfWork: persistence.NewUnitOfWork(db),
	}

	// Initialize Valkey cache
	zap.L().Debug("Attempting to initialize Valkey cache")
	if valkeyCache := persistence.NewValkeyCache(); valkeyCache != nil {
		registry.ValkeyCache = valkeyCache
		zap.L().Info("Valkey cache initialized successfully")
	} else {
		zap.L().Warn("Failed to initialize Valkey cache, continuing without cache")
	}

	// Initialize RabbitMQ
	zap.L().Debug("Attempting to initialize RabbitMQ")
	if rabbitMQ, err := rabbitmq.NewRabbitMQ(); err != nil {
		zap.L().Warn("Failed to initialize RabbitMQ", zap.Error(err))
	} else {
		registry.RabbitMQ = rabbitMQ
		zap.L().Info("RabbitMQ initialized successfully")
	}

	//Initialize Third Party Storage Registry
	zap.L().Debug("Initializing Third Party Storage Registry...")
	registry.ThirdPartyStorage = third_party_repository.NewThirdPartyStorageRegistry(
		s3Bucket,
		s3StreamBucket,
	)

	//Initialize PAYOS Service
	zap.L().Debug("Initializing PayOS...")
	registry.PayOsService = service.NewPayOsService(gormrepository.NewGenericRepository[model.PaymentTransaction](db))

	zap.L().Info("Infrastructure registry initialization completed")
	return registry
}

// RegisterRabbitMQConsumers registers consumer handlers with RabbitMQ
// This method should be called after creating the consumer registry in the presentation layer
func (r *InfrastructureRegistry) RegisterRabbitMQConsumers(
	ctx context.Context,
	handlers map[string]func(context.Context, []byte) error,
) error {
	if r.RabbitMQ == nil {
		zap.L().Warn("RabbitMQ not available, skipping consumer registration")
		return nil
	}

	zap.L().Info("Registering RabbitMQ consumer handlers", zap.Int("handler_count", len(handlers)))

	// Register each handler
	for consumerName, handler := range handlers {
		if err := r.RabbitMQ.RegisterConsumerHandlerFunc(consumerName, handler); err != nil {
			zap.L().Error("Failed to register consumer handler",
				zap.String("consumer", consumerName),
				zap.Error(err))
			return fmt.Errorf("failed to register handler for %s: %w", consumerName, err)
		}
		zap.L().Info("Registered consumer handler", zap.String("consumer", consumerName))
	}

	// Start all configured consumers
	zap.L().Info("Starting RabbitMQ consumers")
	go func() {
		if err := r.RabbitMQ.StartConsumers(ctx); err != nil {
			zap.L().Error("Failed to start RabbitMQ consumers", zap.Error(err))
		} else {
			zap.L().Info("RabbitMQ consumers started successfully")
		}
	}()

	return nil
}

// StopServices gracefully stops all services
func (r *InfrastructureRegistry) StopServices() {
	zap.L().Info("Stopping infrastructure services...")

	// Stop Asynq server
	// Close RabbitMQ connection
	if r.RabbitMQ != nil {
		r.RabbitMQ.Close()
	}

	// Close Valkey cache connection
	if r.ValkeyCache != nil {
		r.ValkeyCache.Close()
	}

	zap.L().Info("Infrastructure services stopped successfully")
}

// IsHealthy checks if all critical infrastructure services are healthy
func (r *InfrastructureRegistry) IsHealthy() map[string]bool {
	zap.L().Debug("Performing infrastructure health check")
	health := make(map[string]bool)

	// Check Valkey cache
	if r.ValkeyCache != nil {
		valkeyHealthy := r.ValkeyCache.Ping() == nil
		health["valkey"] = valkeyHealthy
		zap.L().Debug("Valkey health check completed", zap.Bool("healthy", valkeyHealthy))
	} else {
		health["valkey"] = false
		zap.L().Debug("Valkey not available for health check")
	}

	// Check RabbitMQ
	if r.RabbitMQ != nil {
		rabbitHealthy := r.RabbitMQ.IsConnected()
		health["rabbitmq"] = rabbitHealthy
		zap.L().Debug("RabbitMQ health check completed", zap.Bool("healthy", rabbitHealthy))
	} else {
		health["rabbitmq"] = false
		zap.L().Debug("RabbitMQ not available for health check")
	}

	// Check database
	if r.DB != nil {
		sqlDB, err := r.DB.DB()
		if err == nil {
			dbHealthy := sqlDB.Ping() == nil
			health["database"] = dbHealthy
			zap.L().Debug("Database health check completed", zap.Bool("healthy", dbHealthy))
		} else {
			health["database"] = false
			zap.L().Debug("Database health check failed", zap.Error(err))
		}
	} else {
		health["database"] = false
		zap.L().Debug("Database not available for health check")
	}

	zap.L().Info("Infrastructure health check completed", zap.Any("health_status", health))
	return health
}
