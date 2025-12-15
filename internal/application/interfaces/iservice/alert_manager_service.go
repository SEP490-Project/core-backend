package iservice

import (
	"context"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"

	"github.com/google/uuid"
)

// AlertManagerService defines the interface for centralized alert management
type AlertManagerService interface {
	// RaiseAlert creates a new system alert
	RaiseAlert(ctx context.Context, req *requests.RaiseAlertRequest) (*model.SystemAlert, error)

	// ResolveAlert marks an alert as resolved
	ResolveAlert(ctx context.Context, alertID uuid.UUID, resolvedBy uuid.UUID, resolution *string) error

	// AcknowledgeAlert marks an alert as acknowledged by a user
	AcknowledgeAlert(ctx context.Context, alertID uuid.UUID, userID uuid.UUID, notes *string) error

	// GetAlert returns a single alert by ID
	GetAlert(ctx context.Context, alertID uuid.UUID) (*responses.AlertResponse, error)

	// GetActiveAlerts returns all active alerts
	GetActiveAlerts(ctx context.Context, category *enum.AlertCategory, severity *enum.AlertSeverity) ([]*model.SystemAlert, error)

	// GetAlertsWithPagination returns alerts with pagination
	GetAlertsWithPagination(ctx context.Context, filter *requests.AlertFilterRequest) (*responses.AlertsResponse, int64, error)

	// GetUnacknowledgedCount returns the count of unacknowledged alerts
	GetUnacknowledgedCount(ctx context.Context) (int64, error)

	// GetAlertStats returns alert statistics
	GetAlertStats(ctx context.Context) (*responses.AlertStatsResponse, error)

	// ExpireOldAlerts marks expired alerts as expired
	ExpireOldAlerts(ctx context.Context) (int64, error)

	// Convenience methods for raising specific alert types

	// RaiseLowCTRAlert raises an alert for low CTR on a content
	RaiseLowCTRAlert(ctx context.Context, contentID uuid.UUID, contentTitle string, ctr float64, threshold float64) error

	// RaiseContentRejectedAlert raises an alert when content is rejected
	RaiseContentRejectedAlert(ctx context.Context, contentID uuid.UUID, contentTitle string, reason string) error

	// RaiseScheduleFailedAlert raises an alert when a scheduled publish fails
	RaiseScheduleFailedAlert(ctx context.Context, scheduleID uuid.UUID, contentTitle string, errorMessage string) error

	// RaiseMilestoneDeadlineAlert raises an alert when a milestone deadline is approaching
	RaiseMilestoneDeadlineAlert(ctx context.Context, milestoneID uuid.UUID, milestoneName string, daysUntilDeadline int) error
}
