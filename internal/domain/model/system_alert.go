package model

import (
	"core-backend/internal/domain/enum"
	"database/sql/driver"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// SystemAlert represents a centralized alert for all staff roles
type SystemAlert struct {
	ID              uuid.UUID            `json:"id" gorm:"type:uuid;primaryKey;column:id"`
	Type            enum.AlertType       `json:"type" gorm:"type:varchar(30);not null;column:type"`
	Category        enum.AlertCategory   `json:"category" gorm:"type:varchar(50);not null;column:category"`
	Severity        enum.AlertSeverity   `json:"severity" gorm:"type:varchar(20);not null;default:'MEDIUM';column:severity"`
	Title           string               `json:"title" gorm:"type:varchar(255);not null;column:title"`
	Description     string               `json:"description" gorm:"type:text;not null;column:description"`
	Metadata        datatypes.JSON       `json:"metadata,omitempty" gorm:"type:jsonb;column:metadata"`
	ReferenceID     *uuid.UUID           `json:"reference_id,omitempty" gorm:"type:uuid;column:reference_id"`
	ReferenceType   *string              `json:"reference_type,omitempty" gorm:"type:varchar(50);column:reference_type"`
	ActionURL       *string              `json:"action_url,omitempty" gorm:"type:text;column:action_url"`
	Status          enum.AlertStatus     `json:"status" gorm:"type:varchar(20);not null;default:'ACTIVE';column:status"`
	TargetRoles     AlertTargetRoles     `json:"target_roles" gorm:"type:jsonb;not null;column:target_roles"`
	Acknowledgement *AlertAcknowledgment `json:"acknowledgement,omitempty" gorm:"type:jsonb;column:acknowledgement"`

	// Resolved fields
	ResolvedBy *uuid.UUID `json:"resolved_by,omitempty" gorm:"type:uuid;column:resolved_by"`
	ResolvedAt *time.Time `json:"resolved_at,omitempty" gorm:"column:resolved_at"`

	// Metadata fields
	ExpiresAt *time.Time `json:"expires_at,omitempty" gorm:"index;column:expires_at"`
	CreatedAt *time.Time `json:"created_at" gorm:"autoCreateTime;column:created_at"`
	UpdatedAt *time.Time `json:"updated_at" gorm:"autoUpdateTime;column:updated_at"`
}

func (SystemAlert) TableName() string { return "system_alerts" }

func (sa *SystemAlert) BeforeCreate(_ *gorm.DB) error {
	if sa.ID == uuid.Nil {
		sa.ID = uuid.New()
	}
	return nil
}

func (sa *SystemAlert) IsAcknowledged() bool {
	return sa.Acknowledgement != nil
}

// IsActive checks if the alert is active
func (sa *SystemAlert) IsActive() bool {
	return sa.Status == enum.AlertStatusActive
}

// IsExpired checks if the alert has expired
func (sa *SystemAlert) IsExpired() bool {
	if sa.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*sa.ExpiresAt)
}

// Resolve marks the alert as resolved
func (sa *SystemAlert) Resolve() {
	sa.Status = enum.AlertStatusResolved
}

// Expire marks the alert as expired
func (sa *SystemAlert) Expire() {
	sa.Status = enum.AlertStatusExpired
}

// region: ======== AlertAcknowledgment ==========

// AlertAcknowledgment tracks user interactions with alerts
type AlertAcknowledgment struct {
	UserID    uuid.UUID  `json:"user_id" gorm:"type:uuid;not null;index;column:user_id"`
	Action    string     `json:"action" gorm:"type:varchar(20);not null;column:action"` // "read", "dismissed", "snoozed"
	CreatedAt *time.Time `json:"created_at" gorm:"autoCreateTime;column:created_at"`
}

func (aa *AlertAcknowledgment) Value() (driver.Value, error) {
	return json.Marshal(aa)
}

func (aa *AlertAcknowledgment) Scan(value any) error {
	if value == nil {
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, aa)
}

// endregion

// region: ======== AlertTargetRoles ==========

type AlertTargetRoles struct {
	Roles []enum.UserRole `json:"roles" gorm:"type:jsonb;not null;column:roles"`
}

func (atr *AlertTargetRoles) Scan(value any) error {
	if value == nil {
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, atr)
}

func (atr AlertTargetRoles) Value() (driver.Value, error) {
	return json.Marshal(atr)
}

// endregion
