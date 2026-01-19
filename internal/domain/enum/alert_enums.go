package enum

import (
	"database/sql/driver"
	"fmt"
)

// AlertType represents the type of alert
type AlertType string

const (
	AlertTypeWarning AlertType = "WARNING"
	AlertTypeError   AlertType = "ERROR"
	AlertTypeInfo    AlertType = "INFO"
)

func (t AlertType) IsValid() bool {
	switch t {
	case AlertTypeWarning, AlertTypeError, AlertTypeInfo:
		return true
	}
	return false
}

func (t *AlertType) Scan(value any) error {
	switch v := value.(type) {
	case []byte:
		*t = AlertType(v)
	case string:
		*t = AlertType(v)
	default:
		return fmt.Errorf("failed to scan AlertType: invalid type %T", value)
	}
	return nil
}

func (t AlertType) Value() (driver.Value, error) {
	return string(t), nil
}

func (t AlertType) String() string { return string(t) }

// AlertCategory represents the category of alert for filtering and grouping
type AlertCategory string

const (
	// Content Staff Alerts
	AlertCategoryContentRejected      AlertCategory = "CONTENT_REJECTED"
	AlertCategoryContentPublishFailed AlertCategory = "CONTENT_PUBLISH_FAILED"
	AlertCategoryLowCTR               AlertCategory = "LOW_CTR"
	AlertCategoryLowEngagement        AlertCategory = "LOW_ENGAGEMENT"
	AlertCategoryScheduleFailed       AlertCategory = "SCHEDULE_FAILED"
	AlertCategoryPendingApproval      AlertCategory = "PENDING_APPROVAL"
	AlertCategoryDeadlineApproaching  AlertCategory = "DEADLINE_APPROACHING"
	AlertCategoryMilestoneDeadline    AlertCategory = "MILESTONE_DEADLINE"

	// Marketing Staff Alerts
	AlertCategoryContractTerminateFailed AlertCategory = "CONTRACT_TERMINATE_FAILED"
	AlertCategoryCampaignDeadline        AlertCategory = "CAMPAIGN_DEADLINE"
	AlertCategoryBudgetExceeded          AlertCategory = "BUDGET_EXCEEDED"

	// Sales Staff Alerts
	AlertCategoryOrderIssue     AlertCategory = "ORDER_ISSUE"
	AlertCategoryPaymentOverdue AlertCategory = "PAYMENT_OVERDUE"

	// Admin Alerts
	AlertCategorySystemHealth  AlertCategory = "SYSTEM_HEALTH"
	AlertCategorySecurityIssue AlertCategory = "SECURITY_ISSUE"

	// Contract Violation Alerts
	AlertCategoryViolationDetected   AlertCategory = "VIOLATION_DETECTED"
	AlertCategoryPenaltyPaymentDue   AlertCategory = "PENALTY_PAYMENT_DUE"
	AlertCategoryRefundRequired      AlertCategory = "REFUND_REQUIRED"
	AlertCategoryProofSubmitted      AlertCategory = "PROOF_SUBMITTED"
	AlertCategoryProofReviewRequired AlertCategory = "PROOF_REVIEW_REQUIRED"
	AlertCategoryViolationResolved   AlertCategory = "VIOLATION_RESOLVED"
	AlertCategoryViolationEscalated  AlertCategory = "VIOLATION_ESCALATED"
)

func (c AlertCategory) IsValid() bool {
	switch c {
	case AlertCategoryContentRejected, AlertCategoryContentPublishFailed, AlertCategoryLowCTR, AlertCategoryLowEngagement,
		AlertCategoryScheduleFailed, AlertCategoryPendingApproval, AlertCategoryDeadlineApproaching,
		AlertCategoryMilestoneDeadline, AlertCategoryCampaignDeadline, AlertCategoryBudgetExceeded,
		AlertCategoryOrderIssue, AlertCategoryPaymentOverdue,
		AlertCategorySystemHealth, AlertCategorySecurityIssue,
		AlertCategoryViolationDetected, AlertCategoryPenaltyPaymentDue, AlertCategoryRefundRequired,
		AlertCategoryProofSubmitted, AlertCategoryProofReviewRequired, AlertCategoryViolationResolved,
		AlertCategoryViolationEscalated:
		return true
	}
	return false
}

func (c *AlertCategory) Scan(value any) error {
	switch v := value.(type) {
	case []byte:
		*c = AlertCategory(v)
	case string:
		*c = AlertCategory(v)
	default:
		return fmt.Errorf("failed to scan AlertCategory: invalid type %T", value)
	}
	return nil
}

func (c AlertCategory) Value() (driver.Value, error) {
	return string(c), nil
}

func (c AlertCategory) String() string { return string(c) }

// AlertSeverity represents the severity level of an alert
type AlertSeverity string

const (
	AlertSeverityLow      AlertSeverity = "LOW"
	AlertSeverityMedium   AlertSeverity = "MEDIUM"
	AlertSeverityHigh     AlertSeverity = "HIGH"
	AlertSeverityCritical AlertSeverity = "CRITICAL"
)

func (s AlertSeverity) IsValid() bool {
	switch s {
	case AlertSeverityLow, AlertSeverityMedium, AlertSeverityHigh, AlertSeverityCritical:
		return true
	}
	return false
}

func (s *AlertSeverity) Scan(value any) error {
	switch v := value.(type) {
	case []byte:
		*s = AlertSeverity(v)
	case string:
		*s = AlertSeverity(v)
	default:
		return fmt.Errorf("failed to scan AlertSeverity: invalid type %T", value)
	}
	return nil
}

func (s AlertSeverity) Value() (driver.Value, error) {
	return string(s), nil
}

func (s AlertSeverity) String() string { return string(s) }

// Priority returns a numeric priority for sorting (higher is more severe)
func (s AlertSeverity) Priority() int {
	priorities := map[AlertSeverity]int{
		AlertSeverityLow:      1,
		AlertSeverityMedium:   2,
		AlertSeverityHigh:     3,
		AlertSeverityCritical: 4,
	}
	return priorities[s]
}

// AlertStatus represents the current status of an alert
type AlertStatus string

const (
	AlertStatusActive   AlertStatus = "ACTIVE"
	AlertStatusResolved AlertStatus = "RESOLVED"
	AlertStatusExpired  AlertStatus = "EXPIRED"
)

func (s AlertStatus) IsValid() bool {
	switch s {
	case AlertStatusActive, AlertStatusResolved, AlertStatusExpired:
		return true
	}
	return false
}

func (s *AlertStatus) Scan(value any) error {
	switch v := value.(type) {
	case []byte:
		*s = AlertStatus(v)
	case string:
		*s = AlertStatus(v)
	default:
		return fmt.Errorf("failed to scan AlertStatus: invalid type %T", value)
	}
	return nil
}

func (s AlertStatus) Value() (driver.Value, error) {
	return string(s), nil
}

func (s AlertStatus) String() string { return string(s) }

// ReferenceType represents the type of entity an alert references
type ReferenceType string

const (
	ReferenceTypeContentChannel      ReferenceType = "CONTENT_CHANNEL"
	ReferenceTypeNotification        ReferenceType = "NOTIFICATION"
	ReferenceTypeContent             ReferenceType = "CONTENT"
	ReferenceTypeSchedule            ReferenceType = "SCHEDULE"
	ReferenceTypeMilestone           ReferenceType = "MILESTONE"
	ReferenceTypeCampaign            ReferenceType = "CAMPAIGN"
	ReferenceTypeContract            ReferenceType = "CONTRACT"
	ReferenceTypeContractViolation   ReferenceType = "CONTRACT_VIOLATION"
	ReferenceTypeOrder               ReferenceType = "ORDER"
	ReferenceTypeUser                ReferenceType = "USER"
	ReferenceTypeBrand               ReferenceType = "BRAND"
	ReferenceTypePaymentTransaction  ReferenceType = "PAYMENT_TRANSACTION"
	ReferenceTypePreOrderOpening     ReferenceType = "PRE_ORDER_OPENING"
	ReferenceTypePreOrderAutoReceive ReferenceType = "PRE_ORDER_AUTO_RECEIVE"
	ReferenceTypeOrderAutoReceive    ReferenceType = "ORDER_AUTO_RECEIVE"
	ReferenceTypeOther               ReferenceType = "OTHER"
)

func (r ReferenceType) IsValid() bool {
	switch r {
	case ReferenceTypeContent, ReferenceTypeSchedule, ReferenceTypeMilestone,
		ReferenceTypeCampaign, ReferenceTypeContract, ReferenceTypeOrder,
		ReferenceTypeUser, ReferenceTypeBrand, ReferenceTypePaymentTransaction,
		ReferenceTypePreOrderOpening, ReferenceTypeContractViolation:
		return true
	}
	return false
}

func (r *ReferenceType) Scan(value any) error {
	switch v := value.(type) {
	case []byte:
		*r = ReferenceType(v)
	case string:
		*r = ReferenceType(v)
	default:
		return fmt.Errorf("failed to scan ReferenceType: invalid type %T", value)
	}
	return nil
}

func (r ReferenceType) Value() (driver.Value, error) {
	return string(r), nil
}

func (r ReferenceType) String() string { return string(r) }
