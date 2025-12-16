package responses

import (
	"time"

	"github.com/google/uuid"

	"core-backend/internal/domain/enum"
)

// AlertResponse represents a system alert
type AlertResponse struct {
	ID              uuid.UUID           `json:"id"`
	Category        enum.AlertCategory  `json:"category"`
	Severity        enum.AlertSeverity  `json:"severity"`
	Status          enum.AlertStatus    `json:"status"`
	Title           string              `json:"title"`
	Description     string              `json:"description"`
	ReferenceType   *enum.ReferenceType `json:"reference_type,omitempty"`
	ReferenceID     *uuid.UUID          `json:"reference_id,omitempty"`
	Metadata        map[string]any      `json:"metadata,omitempty"`
	AcknowledgedAt  *time.Time          `json:"acknowledged_at,omitempty"`
	ResolvedAt      *time.Time          `json:"resolved_at,omitempty"`
	AutoResolveAt   *time.Time          `json:"auto_resolve_at,omitempty"`
	CreatedAt       *time.Time          `json:"created_at"`
	Acknowledgments []AcknowledgmentDTO `json:"acknowledgments,omitempty"`
}

// AcknowledgmentDTO represents an acknowledgment record
type AcknowledgmentDTO struct {
	ID             uuid.UUID  `json:"id"`
	UserID         uuid.UUID  `json:"user_id"`
	Username       string     `json:"username,omitempty"`
	Notes          *string    `json:"notes,omitempty"`
	AcknowledgedAt *time.Time `json:"acknowledged_at"`
}

// AlertStatsResponse represents alert statistics
type AlertStatsResponse struct {
	TotalActive       int64                  `json:"total_active"`
	TotalAcknowledged int64                  `json:"total_acknowledged"`
	TotalResolved     int64                  `json:"total_resolved"`
	BySeverity        map[string]int64       `json:"by_severity"`
	ByCategory        map[string]int64       `json:"by_category"`
	RecentAlerts      []AlertSummaryResponse `json:"recent_alerts,omitempty"`
}

// AlertSummaryResponse represents a brief alert summary
type AlertSummaryResponse struct {
	ID        uuid.UUID          `json:"id"`
	Category  enum.AlertCategory `json:"category"`
	Severity  enum.AlertSeverity `json:"severity"`
	Title     string             `json:"title"`
	CreatedAt *time.Time         `json:"created_at"`
}
