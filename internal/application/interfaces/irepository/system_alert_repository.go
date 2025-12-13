package irepository

import (
	"context"
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
	"time"

	"github.com/google/uuid"
)

// SystemAlertRepository defines the interface for system alert data access
type SystemAlertRepository interface {
	// Create creates a new system alert
	Create(ctx context.Context, alert *model.SystemAlert) error

	// GetByID returns an alert by its ID
	GetByID(ctx context.Context, id uuid.UUID) (*model.SystemAlert, error)

	// Update updates an existing alert
	Update(ctx context.Context, alert *model.SystemAlert) error

	// GetActiveAlerts returns all active (non-resolved, non-expired) alerts
	GetActiveAlerts(ctx context.Context, category *enum.AlertCategory, severity *enum.AlertSeverity) ([]*model.SystemAlert, error)

	// GetAlertsByReferenceID returns alerts for a specific reference
	GetAlertsByReferenceID(ctx context.Context, referenceID uuid.UUID) ([]*model.SystemAlert, error)

	// GetAlertsByDateRange returns alerts within a date range
	GetAlertsByDateRange(ctx context.Context, startDate, endDate time.Time, category *enum.AlertCategory, pageSize, pageNumber int) ([]*model.SystemAlert, int64, error)

	// ResolveAlert marks an alert as resolved
	ResolveAlert(ctx context.Context, id uuid.UUID, resolvedBy uuid.UUID) error

	// ExpireOldAlerts marks old alerts as expired
	ExpireOldAlerts(ctx context.Context, before time.Time) (int64, error)

	// CreateAcknowledgment creates an acknowledgment for an alert
	CreateAcknowledgment(ctx context.Context, alertID uuid.UUID, ack *model.AlertAcknowledgment) error

	// // GetAcknowledgmentsByAlertID returns all acknowledgments for an alert
	// GetAcknowledgmentsByAlertID(ctx context.Context, alertID uuid.UUID) ([]*model.AlertAcknowledgment, error)

	// IsAlertAcknowledgedByUser checks if a user has acknowledged an alert
	IsAlertAcknowledgedByUser(ctx context.Context, alertID, userID uuid.UUID) (bool, error)

	// // GetUnacknowledgedAlertCountForUser returns count of unacknowledged alerts for a user
	// GetUnacknowledgedAlertCountForUser(ctx context.Context, userID uuid.UUID) (int64, error)

	// GetActiveAlertCount returns the count of active alerts
	GetActiveAlertCount(ctx context.Context) (int64, error)

	// GetAlertCountBySeverity returns the count of active alerts by severity
	GetAlertCountBySeverity(ctx context.Context, severity enum.AlertSeverity) (int64, error)

	// GetAlertCountByCategory returns the count of active alerts by category
	GetAlertCountByCategory(ctx context.Context, category enum.AlertCategory) (int64, error)
}
