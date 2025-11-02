package service

import (
	"context"
	"errors"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// ServiceHealth represents health status of a service
type ServiceHealth struct {
	Name          string
	IsHealthy     bool
	LastCheckTime time.Time
	LastError     error
	Details       map[string]any
}

// HealthMonitor provides on-demand health checks for all infrastructure services
type HealthMonitor struct {
	emailService *EmailService
	fcmService   *FCMService
	db           *gorm.DB
	valkeyCache  ValkeyHealthChecker
	rabbitMQ     RabbitMQHealthChecker

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
	emailService *EmailService,
	fcmService *FCMService,
	db *gorm.DB,
	valkeyCache ValkeyHealthChecker,
	rabbitMQ RabbitMQHealthChecker,
) *HealthMonitor {
	return &HealthMonitor{
		emailService: emailService,
		fcmService:   fcmService,
		db:           db,
		valkeyCache:  valkeyCache,
		rabbitMQ:     rabbitMQ,
	}
}

// CheckAllServices performs health checks on all registered services
func (h *HealthMonitor) CheckAllServices(ctx context.Context) map[string]ServiceHealth {
	h.mu.Lock()
	defer h.mu.Unlock()

	results := make(map[string]ServiceHealth)

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

	return results
}

// checkEmailService checks SMTP connectivity
func (h *HealthMonitor) checkEmailService(ctx context.Context) ServiceHealth {
	health := ServiceHealth{
		Name:          "EmailService",
		LastCheckTime: time.Now(),
		Details:       make(map[string]any),
	}

	if h.emailService.connPool == nil {
		health.IsHealthy = false
		health.LastError = errors.New("connection pool not initialized")
		return health
	}

	// Try to get a connection from the pool (light check)
	conn, err := h.emailService.connPool.getConnection(ctx)
	if err != nil {
		health.IsHealthy = false
		health.LastError = err
		zap.L().Debug("Email service health check failed", zap.Error(err))
	} else {
		// Return connection to pool
		h.emailService.connPool.returnConnection(conn)
		health.IsHealthy = true
		health.Details["pool_size"] = h.emailService.connPool.maxSize
		zap.L().Debug("Email service health check passed")
	}

	return health
}

// checkFCMService checks FCM connectivity
func (h *HealthMonitor) checkFCMService(_ context.Context) ServiceHealth {
	health := ServiceHealth{
		Name:          "FCMService",
		LastCheckTime: time.Now(),
		Details:       make(map[string]any),
	}

	if h.fcmService.client == nil {
		health.IsHealthy = false
		health.LastError = ErrFCMNotInitialized
		zap.L().Debug("FCM service health check failed - client not initialized")
	} else {
		health.IsHealthy = true
		zap.L().Debug("FCM service health check passed")
	}

	return health
}

// checkDatabase checks database connectivity
func (h *HealthMonitor) checkDatabase(ctx context.Context) ServiceHealth {
	health := ServiceHealth{
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

// CheckTimescaleDB checks TimescaleDB extension and hypertable status
func (h *HealthMonitor) CheckTimescaleDB(ctx context.Context) ServiceHealth {
	health := ServiceHealth{
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

// checkValkey checks Valkey/Redis connectivity
func (h *HealthMonitor) checkValkey(_ context.Context) ServiceHealth {
	health := ServiceHealth{
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
func (h *HealthMonitor) checkRabbitMQ(_ context.Context) ServiceHealth {
	health := ServiceHealth{
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

// IsEmailHealthy performs an on-demand check and returns whether email service is healthy
func (h *HealthMonitor) IsEmailHealthy() bool {
	if h.emailService == nil {
		return false
	}
	health := h.checkEmailService(context.Background())
	return health.IsHealthy
}

// IsFCMHealthy performs an on-demand check and returns whether FCM service is healthy
func (h *HealthMonitor) IsFCMHealthy() bool {
	if h.fcmService == nil {
		return false
	}
	health := h.checkFCMService(context.Background())
	return health.IsHealthy
}

// GetEmailHealth performs an on-demand check and returns email service health status
func (h *HealthMonitor) GetEmailHealth() ServiceHealth {
	if h.emailService == nil {
		return ServiceHealth{
			Name:      "EmailService",
			IsHealthy: false,
			LastError: errors.New("email service not initialized"),
		}
	}
	return h.checkEmailService(context.Background())
}

// GetFCMHealth performs an on-demand check and returns FCM service health status
func (h *HealthMonitor) GetFCMHealth() ServiceHealth {
	if h.fcmService == nil {
		return ServiceHealth{
			Name:      "FCMService",
			IsHealthy: false,
			LastError: errors.New("FCM service not initialized"),
		}
	}
	return h.checkFCMService(context.Background())
}
