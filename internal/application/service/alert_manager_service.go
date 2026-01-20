package service

import (
	"context"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/application/interfaces/iservice"
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type alertManagerService struct {
	alertRepo irepository.SystemAlertRepository
}

// NewAlertManagerService creates a new alert manager service
func NewAlertManagerService(alertRepo irepository.SystemAlertRepository) iservice.AlertManagerService {
	return &alertManagerService{
		alertRepo: alertRepo,
	}
}

// region: ========== Alert Status Methods ==========

// ResolveAlert marks an alert as resolved
func (s *alertManagerService) ResolveAlert(ctx context.Context, alertID uuid.UUID, resolvedBy uuid.UUID, resolution *string) error {
	alert, err := s.alertRepo.GetByID(ctx, alertID)
	if err != nil {
		return fmt.Errorf("alert not found: %w", err)
	}

	if alert.Status != enum.AlertStatusActive {
		return errors.New("alert is not active")
	}

	if err := s.alertRepo.ResolveAlert(ctx, alertID, resolvedBy); err != nil {
		return fmt.Errorf("failed to resolve alert: %w", err)
	}

	zap.L().Info("Alert resolved",
		zap.String("alert_id", alertID.String()),
		zap.String("resolved_by", resolvedBy.String()),
	)

	return nil
}

// AcknowledgeAlert marks an alert as acknowledged by a user
func (s *alertManagerService) AcknowledgeAlert(ctx context.Context, alertID uuid.UUID, userID uuid.UUID, notes *string) error {
	alert, err := s.alertRepo.GetByID(ctx, alertID)
	if err != nil {
		return fmt.Errorf("alert not found: %w", err)
	}

	// Check if already acknowledged
	isAcknowledged, err := s.alertRepo.IsAlertAcknowledgedByUser(ctx, alertID, userID)
	if err != nil {
		return fmt.Errorf("failed to check acknowledgment: %w", err)
	}

	if isAcknowledged {
		return nil // Already acknowledged, no action needed
	}

	// Create acknowledgment
	action := "acknowledged"
	if notes != nil {
		action = "acknowledged_with_notes"
	}
	ack := &model.AlertAcknowledgment{
		UserID: userID,
		Action: action,
	}

	if err := s.alertRepo.CreateAcknowledgment(ctx, alertID, ack); err != nil {
		return fmt.Errorf("failed to create acknowledgment: %w", err)
	}

	zap.L().Info("Alert acknowledged",
		zap.String("alert_id", alertID.String()),
		zap.String("user_id", userID.String()),
		zap.String("category", string(alert.Category)),
	)

	return nil
}

// ExpireOldAlerts marks expired alerts as expired
func (s *alertManagerService) ExpireOldAlerts(ctx context.Context) (int64, error) {
	count, err := s.alertRepo.ExpireOldAlerts(ctx, time.Now())
	if err != nil {
		return 0, fmt.Errorf("failed to expire old alerts: %w", err)
	}

	if count > 0 {
		zap.L().Info("Expired old alerts", zap.Int64("count", count))
	}

	return count, nil
}

// endregion

// region: ========== Alert Retrieval Methods ==========

// GetAlert returns a single alert by ID
func (s *alertManagerService) GetAlert(ctx context.Context, alertID uuid.UUID) (*responses.AlertResponse, error) {
	alert, err := s.alertRepo.GetByID(ctx, alertID)
	if err != nil {
		return nil, fmt.Errorf("alert not found: %w", err)
	}

	resp := &responses.AlertResponse{
		ID:          alert.ID,
		Category:    alert.Category,
		Severity:    alert.Severity,
		Status:      alert.Status,
		Title:       alert.Title,
		Description: alert.Description,
		CreatedAt:   alert.CreatedAt,
	}

	if alert.ReferenceID != nil {
		resp.ReferenceID = alert.ReferenceID
	}
	if alert.ReferenceType != nil {
		refType := enum.ReferenceType(*alert.ReferenceType)
		resp.ReferenceType = &refType
	}
	if alert.ExpiresAt != nil {
		resp.AutoResolveAt = alert.ExpiresAt
	}

	return resp, nil
}

// GetActiveAlerts returns all active alerts
func (s *alertManagerService) GetActiveAlerts(ctx context.Context, category *enum.AlertCategory, severity *enum.AlertSeverity) ([]*model.SystemAlert, error) {
	return s.alertRepo.GetActiveAlerts(ctx, category, severity, nil, nil, nil)
}

// GetAlertsWithPagination returns alerts with pagination
func (s *alertManagerService) GetAlertsWithPagination(ctx context.Context, filter *requests.AlertFilterRequest) (*responses.AlertsResponse, int64, error) {
	// Parse date filters
	var startDate, endDate time.Time
	var category *enum.AlertCategory

	if filter.FromDate != nil {
		parsedDate, err := time.Parse("2006-01-02", *filter.FromDate)
		if err == nil {
			startDate = parsedDate
		}
	} else {
		// Default to last 30 days
		startDate = time.Now().AddDate(0, 0, -30)
	}

	if filter.ToDate != nil {
		parsedDate, err := time.Parse("2006-01-02", *filter.ToDate)
		if err == nil {
			endDate = parsedDate.Add(24*time.Hour - time.Second) // End of day
		}
	} else {
		endDate = time.Now()
	}

	if filter.Category != nil {
		cat := enum.AlertCategory(*filter.Category)
		if cat.IsValid() {
			category = &cat
		}
	}

	// Set defaults for pagination
	pageSize := filter.Limit
	if pageSize <= 0 {
		pageSize = 10
	}
	pageNumber := filter.Page
	if pageNumber <= 0 {
		pageNumber = 1
	}

	// Get alerts
	alerts, total, err := s.alertRepo.GetAlertsByDateRange(
		ctx,
		startDate,
		endDate,
		category,
		pageSize,
		pageNumber,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get alerts: %w", err)
	}

	// Build response
	alertItems := make([]responses.AlertItem, 0, len(alerts))
	for _, alert := range alerts {
		item := responses.AlertItem{
			ID:          alert.ID,
			Type:        string(alert.Type),
			Category:    string(alert.Category),
			Severity:    string(alert.Severity),
			Title:       alert.Title,
			Description: alert.Description,
			CreatedAt:   *alert.CreatedAt,
			IsRead:      false, // Will be updated per user if needed
		}

		if alert.ReferenceID != nil {
			item.ReferenceID = alert.ReferenceID
		}
		if alert.ReferenceType != nil {
			refType := string(*alert.ReferenceType)
			item.ReferenceType = &refType
		}
		if alert.ActionURL != nil {
			item.ActionURL = alert.ActionURL
		}

		alertItems = append(alertItems, item)
	}

	return &responses.AlertsResponse{
		Alerts: alertItems,
		Total:  total,
	}, total, nil
}

// GetUnacknowledgedCount returns the count of unacknowledged alerts
func (s *alertManagerService) GetUnacknowledgedCount(ctx context.Context) (int64, error) {
	return s.alertRepo.GetActiveAlertCount(ctx)
}

// GetAlertStats returns alert statistics
func (s *alertManagerService) GetAlertStats(ctx context.Context) (*responses.AlertStatsResponse, error) {
	// Get active alerts count
	activeCount, err := s.alertRepo.GetActiveAlertCount(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get active alerts count: %w", err)
	}

	// Get count by severity
	severities := []enum.AlertSeverity{
		enum.AlertSeverityLow,
		enum.AlertSeverityMedium,
		enum.AlertSeverityHigh,
		enum.AlertSeverityCritical,
	}
	bySeverity := make(map[string]int64)
	for _, sev := range severities {
		count, err := s.alertRepo.GetAlertCountBySeverity(ctx, sev)
		if err != nil {
			bySeverity[string(sev)] = 0
		} else {
			bySeverity[string(sev)] = count
		}
	}

	// Get count by category
	categories := []enum.AlertCategory{
		enum.AlertCategoryLowCTR,
		enum.AlertCategoryContentRejected,
		enum.AlertCategoryScheduleFailed,
		enum.AlertCategoryMilestoneDeadline,
	}
	byCategory := make(map[string]int64)
	for _, cat := range categories {
		count, err := s.alertRepo.GetAlertCountByCategory(ctx, cat)
		if err != nil {
			byCategory[string(cat)] = 0
		} else {
			byCategory[string(cat)] = count
		}
	}

	return &responses.AlertStatsResponse{
		TotalActive:       activeCount,
		TotalAcknowledged: 0, // Can be calculated if needed
		TotalResolved:     0, // Can be calculated if needed
		BySeverity:        bySeverity,
		ByCategory:        byCategory,
	}, nil
}

// endregion

// region: ========== Alert Raising Methods ==========

// RaiseAlert creates a new system alert
func (s *alertManagerService) RaiseAlert(ctx context.Context, req *requests.RaiseAlertRequest) (*model.SystemAlert, error) {
	// Validate alert type
	if !req.Type.IsValid() {
		return nil, errors.New("invalid alert type")
	}

	// Validate alert category
	if !req.Category.IsValid() {
		return nil, errors.New("invalid alert category")
	}

	// Validate alert severity
	if !req.Severity.IsValid() {
		return nil, errors.New("invalid alert severity")
	}

	// Check for duplicate active alerts with same reference
	if req.ReferenceID != nil {
		existingAlerts, err := s.alertRepo.GetAlertsByReferenceID(ctx, *req.ReferenceID)
		if err == nil && len(existingAlerts) > 0 {
			// Check if there's an active alert with same category
			for _, alert := range existingAlerts {
				if alert.Category == req.Category && alert.Status == enum.AlertStatusActive {
					// Update existing alert instead of creating duplicate
					alert.Severity = req.Severity
					alert.Description = req.Description
					if err := s.alertRepo.Update(ctx, alert); err != nil {
						return nil, fmt.Errorf("failed to update existing alert: %w", err)
					}
					return alert, nil
				}
			}
		}
	}

	// Calculate expiry time based on severity
	var expiresAt *time.Time
	if req.ExpiresInHours != nil {
		expiry := time.Now().Add(time.Duration(*req.ExpiresInHours) * time.Hour)
		expiresAt = &expiry
	} else {
		// Default expiry based on severity
		defaultExpiry := s.getDefaultExpiryForSeverity(req.Severity)
		if defaultExpiry > 0 {
			expiry := time.Now().Add(defaultExpiry)
			expiresAt = &expiry
		}
	}

	// Create new alert
	alert := &model.SystemAlert{
		Type:        req.Type,
		Category:    req.Category,
		Severity:    req.Severity,
		Title:       req.Title,
		Description: req.Description,
		ReferenceID: req.ReferenceID,
		ActionURL:   req.ActionURL,
		Status:      enum.AlertStatusActive,
		ExpiresAt:   expiresAt,
		TargetRoles: model.AlertTargetRoles{Roles: req.TargetRoles},
	}

	// Set reference type if provided
	if req.ReferenceType != nil {
		refTypeStr := string(*req.ReferenceType)
		alert.ReferenceType = &refTypeStr
	}

	if err := s.alertRepo.Create(ctx, alert); err != nil {
		return nil, fmt.Errorf("failed to create alert: %w", err)
	}

	zap.L().Info("Alert raised",
		zap.String("alert_id", alert.ID.String()),
		zap.String("category", string(alert.Category)),
		zap.String("severity", string(alert.Severity)),
	)

	return alert, nil
}

// RaiseLowCTRAlert raises an alert for low CTR on a content
func (s *alertManagerService) RaiseLowCTRAlert(ctx context.Context, contentID uuid.UUID, contentTitle string, ctr float64, threshold float64) error {
	_, err := s.RaiseAlert(ctx, &requests.RaiseAlertRequest{
		Type:        enum.AlertTypeWarning,
		Category:    enum.AlertCategoryLowCTR,
		Severity:    enum.AlertSeverityMedium,
		Title:       fmt.Sprintf("Low CTR on \"%s\"", contentTitle),
		Description: fmt.Sprintf("Content CTR is %.2f%% which is below the threshold of %.2f%%", ctr*100, threshold*100),
		ReferenceID: &contentID,
		ReferenceType: func() *enum.ReferenceType {
			rt := enum.ReferenceTypeContent
			return &rt
		}(),
		ActionURL: func() *string {
			url := fmt.Sprintf("/manage/content/%s", contentID.String())
			return &url
		}(),
	})
	return err
}

// RaiseContentRejectedAlert raises an alert when content is rejected
func (s *alertManagerService) RaiseContentRejectedAlert(ctx context.Context, contentID uuid.UUID, contentTitle string, reason string) error {
	_, err := s.RaiseAlert(ctx, &requests.RaiseAlertRequest{
		Type:        enum.AlertTypeError,
		Category:    enum.AlertCategoryContentRejected,
		Severity:    enum.AlertSeverityHigh,
		Title:       fmt.Sprintf("Content Rejected: \"%s\"", contentTitle),
		Description: fmt.Sprintf("Content was rejected. Reason: %s", reason),
		ReferenceID: &contentID,
		ReferenceType: func() *enum.ReferenceType {
			rt := enum.ReferenceTypeContent
			return &rt
		}(),
		ActionURL: func() *string {
			url := fmt.Sprintf("/manage/content/%s", contentID.String())
			return &url
		}(),
	})
	return err
}

// RaiseScheduleFailedAlert raises an alert when a scheduled publish fails
func (s *alertManagerService) RaiseScheduleFailedAlert(ctx context.Context, scheduleID uuid.UUID, contentTitle string, errorMessage string) error {
	_, err := s.RaiseAlert(ctx, &requests.RaiseAlertRequest{
		Type:        enum.AlertTypeError,
		Category:    enum.AlertCategoryScheduleFailed,
		Severity:    enum.AlertSeverityHigh,
		Title:       fmt.Sprintf("Scheduled Publish Failed: \"%s\"", contentTitle),
		Description: fmt.Sprintf("Failed to publish scheduled content. Error: %s", errorMessage),
		ReferenceID: &scheduleID,
		ReferenceType: func() *enum.ReferenceType {
			rt := enum.ReferenceTypeSchedule
			return &rt
		}(),
		ActionURL: func() *string {
			url := fmt.Sprintf("/manage/schedule/%s", scheduleID.String())
			return &url
		}(),
	})
	return err
}

// RaiseMilestoneDeadlineAlert raises an alert when a milestone deadline is approaching
func (s *alertManagerService) RaiseMilestoneDeadlineAlert(ctx context.Context, milestoneID uuid.UUID, milestoneName string, daysUntilDeadline int) error {
	severity := enum.AlertSeverityLow
	if daysUntilDeadline <= 1 {
		severity = enum.AlertSeverityCritical
	} else if daysUntilDeadline <= 3 {
		severity = enum.AlertSeverityHigh
	} else if daysUntilDeadline <= 7 {
		severity = enum.AlertSeverityMedium
	}

	_, err := s.RaiseAlert(ctx, &requests.RaiseAlertRequest{
		Type:        enum.AlertTypeWarning,
		Category:    enum.AlertCategoryMilestoneDeadline,
		Severity:    severity,
		Title:       fmt.Sprintf("Milestone Deadline Approaching: \"%s\"", milestoneName),
		Description: fmt.Sprintf("Milestone deadline is in %d days", daysUntilDeadline),
		ReferenceID: &milestoneID,
		ReferenceType: func() *enum.ReferenceType {
			rt := enum.ReferenceTypeMilestone
			return &rt
		}(),
		ActionURL: func() *string {
			url := fmt.Sprintf("/manage/milestone/%s", milestoneID.String())
			return &url
		}(),
	})
	return err
}

// endregion

// region: ========== Private Methods ==========

// getDefaultExpiryForSeverity returns the default expiry duration based on severity
func (s *alertManagerService) getDefaultExpiryForSeverity(severity enum.AlertSeverity) time.Duration {
	switch severity {
	case enum.AlertSeverityCritical:
		return 0 // Never expire critical alerts automatically
	case enum.AlertSeverityHigh:
		return 7 * 24 * time.Hour // 7 days
	case enum.AlertSeverityMedium:
		return 3 * 24 * time.Hour // 3 days
	case enum.AlertSeverityLow:
		return 24 * time.Hour // 1 day
	default:
		return 3 * 24 * time.Hour
	}
}

// endregion
