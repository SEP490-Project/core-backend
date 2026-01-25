package requests

import (
	"core-backend/internal/domain/enum"

	"github.com/google/uuid"
)

// RaiseAlertRequest represents a request to raise a new system alert
type RaiseAlertRequest struct {
	Type           enum.AlertType      `json:"type" validate:"required"`
	Category       enum.AlertCategory  `json:"category" validate:"required"`
	Severity       enum.AlertSeverity  `json:"severity" validate:"required"`
	Title          string              `json:"title" validate:"required,min=1,max=255"`
	Description    string              `json:"description" validate:"required,min=1,max=2000"`
	ReferenceID    *uuid.UUID          `json:"reference_id,omitempty"`
	ReferenceType  *enum.ReferenceType `json:"reference_type,omitempty"`
	ActionURL      *string             `json:"action_url,omitempty" validate:"omitempty,url"`
	ExpiresInHours *int                `json:"expires_in_hours,omitempty" validate:"omitempty,min=1,max=720"`
	TargetRoles    []enum.UserRole     `json:"target_roles" validate:"required,min=1,dive,required"`
}

// AlertFilterRequest represents filter parameters for alert queries
type AlertFilterRequest struct {
	PaginationRequest
	Category *string `form:"category" validate:"omitempty,oneof=LOW_CTR CONTENT_REJECTED SCHEDULE_FAILED MILESTONE_DEADLINE SYSTEM_ERROR PERFORMANCE_DEGRADATION"`
	Severity *string `form:"severity" validate:"omitempty,oneof=LOW MEDIUM HIGH CRITICAL"`
	Status   *string `form:"status" validate:"omitempty,oneof=ACTIVE RESOLVED EXPIRED ACKNOWLEDGED"`
	FromDate *string `form:"from_date" validate:"omitempty,datetime=2006-01-02"`
	ToDate   *string `form:"to_date" validate:"omitempty,datetime=2006-01-02"`
}

// AcknowledgeAlertRequest represents a request to acknowledge an alert
type AcknowledgeAlertRequest struct {
	Notes *string `json:"notes,omitempty" validate:"omitempty,max=1000"`
}

// ResolveAlertRequest represents a request to resolve an alert
type ResolveAlertRequest struct {
	Resolution *string `json:"resolution,omitempty" validate:"omitempty,max=2000"`
}
