package gormrepository

import (
	"context"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type systemAlertRepository struct {
	db *gorm.DB
}

// NewSystemAlertRepository creates a new system alert repository
func NewSystemAlertRepository(db *gorm.DB) irepository.SystemAlertRepository {
	return &systemAlertRepository{db: db}
}

// Create creates a new system alert
func (r *systemAlertRepository) Create(ctx context.Context, alert *model.SystemAlert) error {
	if err := r.db.WithContext(ctx).Create(alert).Error; err != nil {
		zap.L().Error("Failed to create system alert", zap.Error(err))
		return err
	}
	return nil
}

// GetByID returns an alert by its ID
func (r *systemAlertRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.SystemAlert, error) {
	var alert model.SystemAlert
	if err := r.db.WithContext(ctx).
		Where("id = ?", id).
		First(&alert).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		zap.L().Error("Failed to get system alert by ID", zap.Error(err))
		return nil, err
	}
	return &alert, nil
}

// Update updates an existing alert
func (r *systemAlertRepository) Update(ctx context.Context, alert *model.SystemAlert) error {
	if err := r.db.WithContext(ctx).Save(alert).Error; err != nil {
		zap.L().Error("Failed to update system alert", zap.Error(err))
		return err
	}
	return nil
}

// GetActiveAlerts returns all active (non-resolved, non-expired) alerts
func (r *systemAlertRepository) GetActiveAlerts(
	ctx context.Context, category *enum.AlertCategory, severity *enum.AlertSeverity, targetRoles []enum.UserRole,
	acknowledgedByID *uuid.UUID, isAcknowledged *bool,
) ([]*model.SystemAlert, error) {
	var alerts []*model.SystemAlert

	query := r.db.WithContext(ctx).
		Where("status = ?", enum.AlertStatusActive.String()).
		Where("(expires_at IS NULL OR expires_at > ?)", time.Now())

	if len(targetRoles) > 0 {
		query = query.Where(
			`target_roles IS NULL 
            OR target_roles->'roles' IS NULL 
            OR jsonb_array_length(target_roles->'roles') = 0 
            OR EXISTS (
                SELECT 1 
                FROM jsonb_array_elements_text(target_roles->'roles') AS role 
                WHERE role IN (?)
            )`,
			targetRoles,
		)
	}
	if acknowledgedByID != nil && *acknowledgedByID != uuid.Nil {
		query = query.Where("acknowledgement IS NOT NULL").
			Where("acknowledgement->>'user_id' = ?", acknowledgedByID)
	}
	if isAcknowledged != nil && *isAcknowledged {
		query = query.Where("acknowledgement IS NOT NULL AND acknowledgement::text != 'null'")
	} else if isAcknowledged != nil && !*isAcknowledged {
		query = query.Where("acknowledgement IS NULL OR acknowledgement::text = 'null'")
	}

	if category != nil {
		query = query.Where("category = ?", category.String())
	}
	if severity != nil {
		query = query.Where("severity = ?", severity.String())
	}

	if err := query.Order("created_at DESC").Find(&alerts).Error; err != nil {
		zap.L().Error("Failed to get active alerts", zap.Error(err))
		return nil, err
	}
	return alerts, nil
}

// GetAlertsByReferenceID returns alerts for a specific reference
func (r *systemAlertRepository) GetAlertsByReferenceID(ctx context.Context, referenceID uuid.UUID) ([]*model.SystemAlert, error) {
	var alerts []*model.SystemAlert
	if err := r.db.WithContext(ctx).
		Where("reference_id = ?", referenceID).
		Order("created_at DESC").
		Find(&alerts).Error; err != nil {
		zap.L().Error("Failed to get alerts by reference ID", zap.Error(err))
		return nil, err
	}
	return alerts, nil
}

// GetAlertsByDateRange returns alerts within a date range
func (r *systemAlertRepository) GetAlertsByDateRange(ctx context.Context, startDate, endDate time.Time, category *enum.AlertCategory, pageSize, pageNumber int) ([]*model.SystemAlert, int64, error) {
	var alerts []*model.SystemAlert
	var total int64

	query := r.db.WithContext(ctx).Model(&model.SystemAlert{}).
		Where("created_at >= ?", startDate).
		Where("created_at < ?", endDate)

	if category != nil {
		query = query.Where("category = ?", category.String())
	}

	if err := query.Count(&total).Error; err != nil {
		zap.L().Error("Failed to count alerts by date range", zap.Error(err))
		return nil, 0, err
	}

	offset := (pageNumber - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&alerts).Error; err != nil {
		zap.L().Error("Failed to get alerts by date range", zap.Error(err))
		return nil, 0, err
	}

	return alerts, total, nil
}

// ResolveAlert marks an alert as resolved
func (r *systemAlertRepository) ResolveAlert(ctx context.Context, id uuid.UUID, resolvedBy uuid.UUID) error {
	now := time.Now()
	if err := r.db.WithContext(ctx).
		Model(&model.SystemAlert{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"status":      enum.AlertStatusResolved.String(),
			"resolved_at": now,
			"resolved_by": resolvedBy,
		}).Error; err != nil {
		zap.L().Error("Failed to resolve alert", zap.Error(err))
		return err
	}
	return nil
}

// ExpireOldAlerts marks old alerts as expired
func (r *systemAlertRepository) ExpireOldAlerts(ctx context.Context, before time.Time) (int64, error) {
	result := r.db.WithContext(ctx).
		Model(&model.SystemAlert{}).
		Where("status = ?", enum.AlertStatusActive.String()).
		Where("expires_at IS NOT NULL").
		Where("expires_at <= ?", before).
		Update("status", enum.AlertStatusExpired.String())

	if result.Error != nil {
		zap.L().Error("Failed to expire old alerts", zap.Error(result.Error))
		return 0, result.Error
	}
	return result.RowsAffected, nil
}

// CreateAcknowledgment creates an acknowledgment for an alert
func (r *systemAlertRepository) CreateAcknowledgment(ctx context.Context, alertID uuid.UUID, ack *model.AlertAcknowledgment) error {
	if err := r.db.WithContext(ctx).Model(&model.SystemAlert{}).
		Where("id = ?", alertID).
		Update("acknowledgement", ack).Error; err != nil {
		zap.L().Error("Failed to update alert acknowledgment time", zap.Error(err))
		return err
	}
	return nil
}

/*
// GetAcknowledgmentsByAlertID returns all acknowledgments for an alert
func (r *systemAlertRepository) GetAcknowledgmentsByAlertID(ctx context.Context, alertID uuid.UUID) ([]*model.AlertAcknowledgment, error) {
	var acks []*model.AlertAcknowledgment
	if err := r.db.WithContext(ctx).
		Where("alert_id = ?", alertID).
		Order("acknowledged_at DESC").
		Find(&acks).Error; err != nil {
		zap.L().Error("Failed to get acknowledgments by alert ID", zap.Error(err))
		return nil, err
	}
	return acks, nil
}
*/

// IsAlertAcknowledgedByUser checks if a user has acknowledged an alert
func (r *systemAlertRepository) IsAlertAcknowledgedByUser(ctx context.Context, alertID, userID uuid.UUID) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&model.SystemAlert{}).
		Where("id = ?", alertID).
		Where("acknowledgement IS NOT NULL").
		Where("acknowledgement->>'user_id' = ?", userID).
		Count(&count).Error; err != nil {
		zap.L().Error("Failed to check alert acknowledgment", zap.Error(err))
		return false, err
	}
	return count > 0, nil
}

/*
// GetUnacknowledgedAlertCountForUser returns count of unacknowledged alerts for a user
func (r *systemAlertRepository) GetUnacknowledgedAlertCountForUser(ctx context.Context, userID uuid.UUID) (int64, error) {
	var count int64

	// Get active alerts that the user hasn't acknowledged
	subQuery := r.db.WithContext(ctx).
		Table("alert_acknowledgments").
		Select("alert_id").
		Where("user_id = ?", userID)

	if err := r.db.WithContext(ctx).
		Model(&model.SystemAlert{}).
		Where("status = ?", enum.AlertStatusActive.String()).
		Where("(expires_at IS NULL OR expires_at > ?)", time.Now()).
		Where("id NOT IN (?)", subQuery).
		Count(&count).Error; err != nil {
		zap.L().Error("Failed to get unacknowledged alert count", zap.Error(err))
		return 0, err
	}

	return count, nil
} */

// GetActiveAlertCount returns the count of active alerts
func (r *systemAlertRepository) GetActiveAlertCount(ctx context.Context) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&model.SystemAlert{}).
		Where("status = ?", enum.AlertStatusActive.String()).
		Where("(expires_at IS NULL OR expires_at > ?)", time.Now()).
		Count(&count).Error; err != nil {
		zap.L().Error("Failed to get active alert count", zap.Error(err))
		return 0, err
	}
	return count, nil
}

// GetAlertCountBySeverity returns the count of active alerts by severity
func (r *systemAlertRepository) GetAlertCountBySeverity(ctx context.Context, severity enum.AlertSeverity) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&model.SystemAlert{}).
		Where("status = ?", enum.AlertStatusActive.String()).
		Where("severity = ?", severity.String()).
		Where("(expires_at IS NULL OR expires_at > ?)", time.Now()).
		Count(&count).Error; err != nil {
		zap.L().Error("Failed to get alert count by severity", zap.Error(err))
		return 0, err
	}
	return count, nil
}

// GetAlertCountByCategory returns the count of active alerts by category
func (r *systemAlertRepository) GetAlertCountByCategory(ctx context.Context, category enum.AlertCategory) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&model.SystemAlert{}).
		Where("status = ?", enum.AlertStatusActive.String()).
		Where("category = ?", category.String()).
		Where("(expires_at IS NULL OR expires_at > ?)", time.Now()).
		Count(&count).Error; err != nil {
		zap.L().Error("Failed to get alert count by category", zap.Error(err))
		return 0, err
	}
	return count, nil
}
