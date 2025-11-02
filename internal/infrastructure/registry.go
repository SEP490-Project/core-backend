// Package infrastructure provides the InfrastructureRegistry struct that holds various infrastructure services.
package infrastructure

import (
	"context"
	"core-backend/config"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/application/interfaces/iservice_third_party"
	"core-backend/internal/domain/model"
	gormrepository "core-backend/internal/infrastructure/gorm_repository"
	"core-backend/internal/infrastructure/persistence"
	"core-backend/internal/infrastructure/rabbitmq"
	"core-backend/internal/infrastructure/scheduler"
	"core-backend/internal/infrastructure/service"
	"core-backend/internal/infrastructure/third_party_repository"
	"fmt"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

type InfrastructureRegistry struct {
	Config            *config.AppConfig
	DB                *gorm.DB
	ThirdPartyStorage *third_party_repository.ThirdPartyStorageRegistry
	UnitOfWork        irepository.UnitOfWork
	ValkeyCache       *persistence.ValkeyCache
	RabbitMQ          *rabbitmq.RabbitMQ
	PayOsService      iservice_third_party.PayOSService
	EmailService      *service.EmailService
	FCMService        *service.FCMService
	HealthMonitor     *service.HealthMonitor
	GHNService        iservice_third_party.GHNService

	//Automatic Trigger
	schedulers []scheduler.TaskScheduler
	//Manual Trigger Schedulers
	LocationSyncTask scheduler.TaskScheduler
}

func NewInfrastructureRegistry(
	config *config.AppConfig,
	db *gorm.DB,
	s3Bucket *persistence.S3Bucket,
	s3StreamBucket *persistence.S3StreamingBucket,
) *InfrastructureRegistry {
	zap.L().Info("Initializing infrastructure registry")

	registry := &InfrastructureRegistry{
		Config:     config,
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

	//========================EXTERNAL SERVICES========================//
	//Initialize PAYOS Service
	zap.L().Debug("Initializing PayOS...")
	registry.PayOsService = service.NewPayOsService(gormrepository.NewGenericRepository[model.PaymentTransaction](db))

	//Initialize EmailService
	zap.L().Debug("Initializing EmailService...")
	emailService, err := service.NewEmailService(config)
	if err != nil {
		zap.L().Warn("Failed to initialize EmailService, skipping email notifications", zap.Error(err))
	} else {
		registry.EmailService = emailService
		zap.L().Info("EmailService initialized successfully")
	}

	//Initialize FCMService
	zap.L().Debug("Initializing FCMService...")
	fcmService, err := service.NewFCMService(config.FirebaseFCM.ServiceAccountPath, config)
	if err != nil {
		zap.L().Warn("Failed to initialize FCMService, skipping FCM notifications", zap.Error(err))
	} else {
		registry.FCMService = fcmService
		zap.L().Info("FCMService initialized successfully")
	}

	//External Services
	zap.L().Debug("Initializing GHN Service...")
	registry.GHNService = service.NewGHNService(config)

	//==============================================================

	//Initialize Task Schedulers
	registry.schedulers = []scheduler.TaskScheduler{
		//Location Sync Scheduler
		scheduler.NewLocationSyncScheduler(config, db),
		// Add more schedulers here as needed
	}
	//Initialize Manual Trigger Schedulers
	registry.LocationSyncTask = scheduler.NewLocationSyncScheduler(config, db)

	// Initialize Health Monitor
	zap.L().Debug("Initializing Health Monitor...")
	healthMonitor := service.NewHealthMonitor(
		emailService,
		fcmService,
		db,
		registry.ValkeyCache,
		registry.RabbitMQ,
	)
	registry.HealthMonitor = healthMonitor
	zap.L().Info("Health Monitor initialized successfully")

	// Override AdminConfig from Database
	zap.L().Debug("Overriding AdminConfig from database")
	err = registry.OverrideAdminConfig()
	if err != nil {
		zap.L().Error("Failed to override admin config", zap.Error(err))
	}

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

// OverrideAdminConfig overrides the AdminConfig with values from the database
func (r *InfrastructureRegistry) OverrideAdminConfig() error {
	var adminConfig []model.Config
	query := r.DB.Model(&model.Config{}).Find(&adminConfig)
	if err := query.Error; err != nil {
		zap.L().Error("Failed to load admin config from database", zap.Error(err))
		return err
	}
	zap.L().Info("Loaded admin config from database", zap.Int("config_count", len(adminConfig)),
		zap.Any("configs", r.Config.AdminConfig))
	err := r.Config.AdminConfig.Override(adminConfig)
	if err != nil {
		zap.L().Error("Failed to override admin config from database", zap.Error(err))
		return err
	}
	return nil
}

// StartSchedulers launches background schedulers controlled by config.
// It uses the provided context to manage lifecycle and shutdown.
func (r *InfrastructureRegistry) StartSchedulers(ctx context.Context) {
	zap.L().Info("=================Starting schedulers=================")
	for idx, s := range r.schedulers {
		go func(s scheduler.TaskScheduler) {
			zap.L().Info(fmt.Sprintf("Task #%d: ", idx+1), zap.String("type", fmt.Sprintf("%T", s)))
			s.Start(ctx)
			zap.L().Info("-----------------")
		}(s)
	}
}

// StopServices gracefully stops all services
func (r *InfrastructureRegistry) StopServices() {
	zap.L().Info("Stopping infrastructure services...")

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
