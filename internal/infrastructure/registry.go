// Package infrastructure provides the InfrastructureRegistry struct that holds various infrastructure services.
package infrastructure

import (
	"context"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/application/interfaces/irepository_third_party"
	"core-backend/internal/application/interfaces/iservice_third_party"
	"core-backend/internal/domain/model"
	gormrepository "core-backend/internal/infrastructure/gorm_repository"
	"core-backend/internal/infrastructure/persistence"
	"core-backend/internal/infrastructure/queue"
	"core-backend/internal/infrastructure/rabbitmq"
	"core-backend/internal/infrastructure/service"
	"core-backend/internal/infrastructure/third_party_repository"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

type InfrastructureRegistry struct {
	DB           *gorm.DB
	UnitOfWork   irepository.UnitOfWork
	ValkeyCache  *persistence.ValkeyCache
	RabbitMQ     *rabbitmq.RabbitMQ
	AsynqClient  *queue.AsynqClient
	AsynqServer  *queue.AsynqServer
	S3Repository irepository_third_party.S3Repository
	PayOsService iservice_third_party.PayOSService
}

func NewInfrastructureRegistry(db *gorm.DB, s3Bucket *persistence.S3Bucket) *InfrastructureRegistry {
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

	// Initialize Asynq
	zap.L().Debug("Initializing Asynq client and server")
	registry.AsynqClient = queue.NewAsynqClient()
	registry.AsynqServer = queue.NewAsynqServer()
	zap.L().Info("Asynq client and server initialized successfully")

	// Initialize S3 repository
	zap.L().Debug("Attempting to initialize S3 repository")
	if s3Repo := third_party_repository.NewS3Repository(s3Bucket); s3Repo != nil {
		registry.S3Repository = s3Repo
		zap.L().Info("S3 repository initialized successfully")
	} else {
		zap.L().Warn("Failed to initialize S3 repository, continuing without S3 support")
	}

	//Initialize PAYOS Service
	zap.L().Debug("Initializing PayOS...")
	registry.PayOsService = service.NewPayOsService(gormrepository.NewGenericRepository[model.PaymentTransaction](db))

	zap.L().Info("Infrastructure registry initialization completed")
	return registry
}

// StartBackgroundServices starts all background services
func (r *InfrastructureRegistry) StartBackgroundServices(ctx context.Context) {
	zap.L().Info("Starting background services")

	// Start Asynq server in a goroutine
	zap.L().Debug("Starting Asynq server in background")
	go func() {
		if err := r.AsynqServer.Start(); err != nil {
			zap.L().Error("Asynq server failed", zap.Error(err))
		} else {
			zap.L().Info("Asynq server started successfully")
		}
	}()

	// Start RabbitMQ consumer if RabbitMQ is available
	if r.RabbitMQ != nil {
		zap.L().Debug("Starting RabbitMQ consumer in background")
		go func() {
			if err := r.RabbitMQ.Consume(ctx, r.handleRabbitMQMessage); err != nil {
				zap.L().Error("Failed to start RabbitMQ consumer", zap.Error(err))
			} else {
				zap.L().Info("RabbitMQ consumer started successfully")
			}
		}()
	} else {
		zap.L().Debug("RabbitMQ not available, skipping consumer startup")
	}

	zap.L().Info("Background services started successfully")
}

// StopServices gracefully stops all services
func (r *InfrastructureRegistry) StopServices() {
	zap.L().Info("Stopping infrastructure services...")

	// Stop Asynq server
	if r.AsynqServer != nil {
		r.AsynqServer.Stop()
	}

	// Close Asynq client
	if r.AsynqClient != nil {
		r.AsynqClient.Close()
	}

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

// handleRabbitMQMessage handles incoming RabbitMQ messages
func (r *InfrastructureRegistry) handleRabbitMQMessage(message []byte) error {
	zap.L().Info("Received RabbitMQ message", zap.ByteString("message", message))

	// TODO: Implement message processing logic based on your needs
	// This could involve:
	// - Parsing the message to determine the type
	// - Routing to appropriate handlers
	// - Enqueuing tasks in Asynq for processing

	return nil
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

	health["asynq_client"] = r.AsynqClient != nil
	health["asynq_server"] = r.AsynqServer != nil

	zap.L().Debug("Asynq services health check",
		zap.Bool("client_available", r.AsynqClient != nil),
		zap.Bool("server_available", r.AsynqServer != nil))

	zap.L().Info("Infrastructure health check completed", zap.Any("health_status", health))
	return health
}
