package service

import (
	"context"
	"core-backend/internal/application/interfaces/iservice_third_party"
	"core-backend/internal/infrastructure/asynq"
	"errors"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// healthMonitor provides on-demand health checks for all infrastructure services
type healthMonitor struct {
	emailService iservice_third_party.EmailService
	fcmService   iservice_third_party.FCMService
	db           *gorm.DB
	valkeyCache  ValkeyHealthChecker
	rabbitMQ     RabbitMQHealthChecker
	asynqClient  *asynq.AsynqClient

	mu sync.RWMutex
}

// ValkeyHealthChecker defines the interface for checking Valkey health
type ValkeyHealthChecker interface {
	Ping() error
}

// RabbitMQHealthChecker defines the interface for checking RabbitMQ health
type RabbitMQHealthChecker interface {
	IsConnected() bool
	GetConnection() *amqp.Connection
}

// NewHealthMonitor creates a new on-demand health monitor
func NewHealthMonitor(
	emailService iservice_third_party.EmailService,
	fcmService iservice_third_party.FCMService,
	db *gorm.DB,
	valkeyCache ValkeyHealthChecker,
	rabbitMQ RabbitMQHealthChecker,
	asynqClient *asynq.AsynqClient,
) iservice_third_party.HealthMonitor {
	return &healthMonitor{
		emailService: emailService,
		fcmService:   fcmService,
		db:           db,
		valkeyCache:  valkeyCache,
		rabbitMQ:     rabbitMQ,
		asynqClient:  asynqClient,
	}
}

// CheckAllServices performs health checks on all registered services
func (h *healthMonitor) CheckAllServices(ctx context.Context) map[string]iservice_third_party.ServiceHealth {
	h.mu.Lock()
	defer h.mu.Unlock()

	results := make(map[string]iservice_third_party.ServiceHealth)

	// Check Email Service
	if h.emailService != nil {
		results["email"] = h.checkEmailService(ctx)
	}

	// Check FCM Service
	if h.fcmService != nil {
		results["fcm"] = h.checkFCMService(ctx)
	}

	// Check Database
	if h.db != nil {
		results["database"] = h.checkDatabase(ctx)
		results["timescaledb"] = h.CheckTimescaleDB(ctx)
	}

	// Check Valkey Cache
	if h.valkeyCache != nil {
		results["valkey"] = h.checkValkey(ctx)
	}

	// Check RabbitMQ
	if h.rabbitMQ != nil {
		results["rabbitmq"] = h.checkRabbitMQ(ctx)
	}

	if h.asynqClient != nil {
		results["asynq"] = h.CheckAsynq(ctx)
	}

	return results
}

// GetEmailHealth performs an on-demand check and returns email service health status
func (h *healthMonitor) GetEmailHealth() iservice_third_party.ServiceHealth {
	if h.emailService == nil {
		return iservice_third_party.ServiceHealth{
			Name:      "EmailService",
			IsHealthy: false,
			LastError: errors.New("email service not initialized"),
		}
	}
	return h.checkEmailService(context.Background())
}

// GetFCMHealth performs an on-demand check and returns FCM service health status
func (h *healthMonitor) GetFCMHealth() iservice_third_party.ServiceHealth {
	if h.fcmService == nil {
		return iservice_third_party.ServiceHealth{
			Name:      "FCMService",
			IsHealthy: false,
			LastError: errors.New("FCM service not initialized"),
		}
	}
	return h.checkFCMService(context.Background())
}

// CheckTimescaleDB checks TimescaleDB extension and hypertable status
func (h *healthMonitor) CheckTimescaleDB(ctx context.Context) iservice_third_party.ServiceHealth {
	health := iservice_third_party.ServiceHealth{
		Name:          "TimescaleDB",
		LastCheckTime: time.Now(),
		Details:       make(map[string]any),
	}

	// Check if TimescaleDB extension is installed
	var extensionVersion string
	err := h.db.WithContext(ctx).Raw(`
		SELECT extversion 
		FROM pg_extension 
		WHERE extname = 'timescaledb'
	`).Scan(&extensionVersion).Error

	if err != nil {
		health.IsHealthy = false
		health.LastError = errors.New("TimescaleDB extension not found or inaccessible")
		health.Details["error"] = err.Error()
		zap.L().Debug("TimescaleDB health check failed - extension not found", zap.Error(err))
		return health
	}

	health.Details["extension_version"] = extensionVersion

	// Check hypertables status
	var hypertableCount int64
	err = h.db.WithContext(ctx).Raw(`
		SELECT COUNT(*) 
		FROM timescaledb_information.hypertables
	`).Scan(&hypertableCount).Error

	if err != nil {
		health.IsHealthy = false
		health.LastError = err
		health.Details["error"] = "Failed to query hypertables"
		zap.L().Debug("TimescaleDB health check failed - hypertables query failed", zap.Error(err))
		return health
	}

	health.Details["hypertable_count"] = hypertableCount

	// Check specific hypertables we expect (click_events, kpi_metrics)
	type HypertableInfo struct {
		HypertableName string `gorm:"column:hypertable_name"`
		NumChunks      int64  `gorm:"column:num_chunks"`
	}

	var hypertables []HypertableInfo
	err = h.db.WithContext(ctx).Raw(`
		SELECT 
			hypertable_name,
			num_chunks
		FROM timescaledb_information.hypertables
		WHERE hypertable_name IN ('click_events', 'kpi_metrics')
	`).Scan(&hypertables).Error

	if err != nil {
		health.IsHealthy = false
		health.LastError = err
		health.Details["error"] = "Failed to query hypertable details"
		zap.L().Debug("TimescaleDB health check failed - hypertable details query failed", zap.Error(err))
		return health
	}

	hypertableDetails := make(map[string]any)
	for _, ht := range hypertables {
		hypertableDetails[ht.HypertableName] = map[string]any{
			"num_chunks": ht.NumChunks,
		}
	}
	health.Details["hypertables"] = hypertableDetails

	// If we got here, everything is healthy
	health.IsHealthy = true
	zap.L().Debug("TimescaleDB health check passed",
		zap.String("version", extensionVersion),
		zap.Int64("hypertable_count", hypertableCount))

	return health
}

func (h *healthMonitor) CheckAsynq(ctx context.Context) iservice_third_party.ServiceHealth {
	health := iservice_third_party.ServiceHealth{
		Name:          "Asynq",
		LastCheckTime: time.Now(),
		Details:       make(map[string]any),
	}

	if h.asynqClient == nil {
		health.IsHealthy = false
		health.LastError = errors.New("asynq client not initialized")
		zap.L().Debug("Asynq health check failed - client not initialized")
		return health
	}
	err := h.asynqClient.HealthCheck()
	if err != nil {
		health.IsHealthy = false
		health.LastError = err
		health.Details["error"] = err.Error()
		zap.L().Debug("Asynq health check failed", zap.Error(err))
	} else {
		health.IsHealthy = true
		zap.L().Debug("Asynq health check passed")
	}
	return health
}

// IsEmailHealthy performs an on-demand check and returns whether email service is healthy
func (h *healthMonitor) IsEmailHealthy() bool {
	if h.emailService == nil {
		return false
	}
	health := h.checkEmailService(context.Background())
	return health.IsHealthy
}

// IsFCMHealthy performs an on-demand check and returns whether FCM service is healthy
func (h *healthMonitor) IsFCMHealthy() bool {
	if h.fcmService == nil {
		return false
	}
	health := h.checkFCMService(context.Background())
	return health.IsHealthy
}

// region: ============== Private Methods ==============

// checkEmailService checks SMTP connectivity
func (h *healthMonitor) checkEmailService(ctx context.Context) iservice_third_party.ServiceHealth {
	return h.emailService.Health(ctx)
}

// checkFCMService checks FCM connectivity
func (h *healthMonitor) checkFCMService(ctx context.Context) iservice_third_party.ServiceHealth {
	return h.fcmService.Health(ctx)
}

// checkDatabase checks database connectivity
func (h *healthMonitor) checkDatabase(ctx context.Context) iservice_third_party.ServiceHealth {
	health := iservice_third_party.ServiceHealth{
		Name:          "Database",
		LastCheckTime: time.Now(),
		Details:       make(map[string]any),
	}

	sqlDB, err := h.db.DB()
	if err != nil {
		health.IsHealthy = false
		health.LastError = err
		zap.L().Debug("Database health check failed - cannot get SQL DB", zap.Error(err))
		return health
	}

	if err := sqlDB.PingContext(ctx); err != nil {
		health.IsHealthy = false
		health.LastError = err
		zap.L().Debug("Database health check failed - ping failed", zap.Error(err))
	} else {
		health.IsHealthy = true

		// Get database stats
		stats := sqlDB.Stats()
		health.Details["open_connections"] = stats.OpenConnections
		health.Details["in_use"] = stats.InUse
		health.Details["idle"] = stats.Idle

		zap.L().Debug("Database health check passed",
			zap.Int("open_connections", stats.OpenConnections))
	}

	return health
}

// checkValkey checks Valkey/Redis connectivity
func (h *healthMonitor) checkValkey(_ context.Context) iservice_third_party.ServiceHealth {
	health := iservice_third_party.ServiceHealth{
		Name:          "Valkey",
		LastCheckTime: time.Now(),
		Details:       make(map[string]any),
	}

	if err := h.valkeyCache.Ping(); err != nil {
		health.IsHealthy = false
		health.LastError = err
		zap.L().Debug("Valkey health check failed", zap.Error(err))
	} else {
		health.IsHealthy = true
		zap.L().Debug("Valkey health check passed")
	}

	return health
}

// checkRabbitMQ checks RabbitMQ connectivity
func (h *healthMonitor) checkRabbitMQ(_ context.Context) iservice_third_party.ServiceHealth {
	health := iservice_third_party.ServiceHealth{
		Name:          "RabbitMQ",
		LastCheckTime: time.Now(),
		Details:       make(map[string]any),
	}

	if !h.rabbitMQ.IsConnected() {
		health.IsHealthy = false
		health.LastError = errors.New("not connected")
		zap.L().Debug("RabbitMQ health check failed - not connected")
	} else {
		conn := h.rabbitMQ.GetConnection()
		if conn != nil && !conn.IsClosed() {
			health.IsHealthy = true
			zap.L().Debug("RabbitMQ health check passed")
		} else {
			health.IsHealthy = false
			health.LastError = errors.New("connection closed")
			zap.L().Debug("RabbitMQ health check failed - connection closed")
		}
	}

	return health
}

// endregion
