package infrastructure

import (
	"context"
	"core-backend/internal/infrastructure/persistence"
	"core-backend/internal/infrastructure/queue"
	"core-backend/internal/infrastructure/rabbitmq"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

type InfrastructureRegistry struct {
	DB           *gorm.DB
	ValkeyCache  *persistence.ValkeyCache
	RabbitMQ     *rabbitmq.RabbitMQ
	AsynqClient  *queue.AsynqClient
	AsynqServer  *queue.AsynqServer
}

func NewInfrastructureRegistry(db *gorm.DB) *InfrastructureRegistry {
	registry := &InfrastructureRegistry{
		DB: db,
	}

	// Initialize Valkey cache
	if valkeyCache := persistence.NewValkeyCache(); valkeyCache != nil {
		registry.ValkeyCache = valkeyCache
		zap.L().Info("Valkey cache initialized successfully")
	} else {
		zap.L().Warn("Failed to initialize Valkey cache, continuing without cache")
	}

	// Initialize RabbitMQ
	if rabbitMQ, err := rabbitmq.NewRabbitMQ(); err != nil {
		zap.L().Warn("Failed to initialize RabbitMQ", zap.Error(err))
	} else {
		registry.RabbitMQ = rabbitMQ
		zap.L().Info("RabbitMQ initialized successfully")
	}

	// Initialize Asynq
	registry.AsynqClient = queue.NewAsynqClient()
	registry.AsynqServer = queue.NewAsynqServer()
	zap.L().Info("Asynq client and server initialized successfully")

	return registry
}

// StartBackgroundServices starts all background services
func (r *InfrastructureRegistry) StartBackgroundServices(ctx context.Context) {
	// Start Asynq server in a goroutine
	go func() {
		if err := r.AsynqServer.Start(); err != nil {
			zap.L().Error("Asynq server failed", zap.Error(err))
		}
	}()

	// Start RabbitMQ consumer if RabbitMQ is available
	if r.RabbitMQ != nil {
		go func() {
			if err := r.RabbitMQ.Consume(ctx, r.handleRabbitMQMessage); err != nil {
				zap.L().Error("Failed to start RabbitMQ consumer", zap.Error(err))
			}
		}()
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
	health := make(map[string]bool)

	// Check Valkey cache
	if r.ValkeyCache != nil {
		health["valkey"] = r.ValkeyCache.Ping() == nil
	} else {
		health["valkey"] = false
	}

	// Check RabbitMQ
	if r.RabbitMQ != nil {
		health["rabbitmq"] = r.RabbitMQ.IsConnected()
	} else {
		health["rabbitmq"] = false
	}

	// Check database
	if r.DB != nil {
		sqlDB, err := r.DB.DB()
		if err == nil {
			health["database"] = sqlDB.Ping() == nil
		} else {
			health["database"] = false
		}
	} else {
		health["database"] = false
	}

	health["asynq_client"] = r.AsynqClient != nil
	health["asynq_server"] = r.AsynqServer != nil

	return health
}