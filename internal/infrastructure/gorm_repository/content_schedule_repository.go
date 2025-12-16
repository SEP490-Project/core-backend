package gormrepository

import (
	"context"
	"core-backend/internal/application/dto/dtos"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type contentScheduleRepository struct {
	db *gorm.DB
}

// NewContentScheduleRepository creates a new content schedule repository
func NewContentScheduleRepository(db *gorm.DB) irepository.ContentScheduleRepository {
	return &contentScheduleRepository{db: db}
}

// Create creates a new content schedule
func (r *contentScheduleRepository) Create(ctx context.Context, schedule *model.ContentSchedule) error {
	if err := r.db.WithContext(ctx).Create(schedule).Error; err != nil {
		zap.L().Error("Failed to create content schedule", zap.Error(err))
		return err
	}
	return nil
}

// GetByID returns a schedule by its ID
func (r *contentScheduleRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.ContentSchedule, error) {
	var schedule model.ContentSchedule
	if err := r.db.WithContext(ctx).
		Where("id = ?", id).
		Where("deleted_at IS NULL").
		First(&schedule).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		zap.L().Error("Failed to get content schedule by ID", zap.Error(err))
		return nil, err
	}
	return &schedule, nil
}

// GetByContentChannelID returns a schedule by content channel ID
func (r *contentScheduleRepository) GetByContentChannelID(ctx context.Context, contentChannelID uuid.UUID) (*model.ContentSchedule, error) {
	var schedule model.ContentSchedule
	if err := r.db.WithContext(ctx).
		Where("content_channel_id = ?", contentChannelID).
		Where("deleted_at IS NULL").
		First(&schedule).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		zap.L().Error("Failed to get content schedule by content channel ID", zap.Error(err))
		return nil, err
	}
	return &schedule, nil
}

// Update updates an existing schedule
func (r *contentScheduleRepository) Update(ctx context.Context, schedule *model.ContentSchedule) error {
	if err := r.db.WithContext(ctx).Save(schedule).Error; err != nil {
		zap.L().Error("Failed to update content schedule", zap.Error(err))
		return err
	}
	return nil
}

// Delete soft deletes a schedule
func (r *contentScheduleRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if err := r.db.WithContext(ctx).
		Model(&model.ContentSchedule{}).
		Where("id = ?", id).
		Update("deleted_at", time.Now()).Error; err != nil {
		zap.L().Error("Failed to delete content schedule", zap.Error(err))
		return err
	}
	return nil
}

// GetPendingSchedules returns all pending schedules that should be processed
func (r *contentScheduleRepository) GetPendingSchedules(ctx context.Context, before time.Time) ([]*model.ContentSchedule, error) {
	var schedules []*model.ContentSchedule
	if err := r.db.WithContext(ctx).
		Where("status = ?", enum.ScheduleStatusPending.String()).
		Where("scheduled_at <= ?", before).
		Where("deleted_at IS NULL").
		Order("scheduled_at ASC").
		Find(&schedules).Error; err != nil {
		zap.L().Error("Failed to get pending schedules", zap.Error(err))
		return nil, err
	}
	return schedules, nil
}

// GetSchedulesByStatus returns schedules by status
func (r *contentScheduleRepository) GetSchedulesByStatus(ctx context.Context, status enum.ScheduleStatus, pageSize, pageNumber int) ([]*model.ContentSchedule, int64, error) {
	var schedules []*model.ContentSchedule
	var total int64

	query := r.db.WithContext(ctx).Model(&model.ContentSchedule{}).
		Where("status = ?", status.String()).
		Where("deleted_at IS NULL")

	if err := query.Count(&total).Error; err != nil {
		zap.L().Error("Failed to count schedules by status", zap.Error(err))
		return nil, 0, err
	}

	offset := (pageNumber - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("scheduled_at ASC").Find(&schedules).Error; err != nil {
		zap.L().Error("Failed to get schedules by status", zap.Error(err))
		return nil, 0, err
	}

	return schedules, total, nil
}

// GetSchedulesWithDetails returns schedules with content and channel details
func (r *contentScheduleRepository) GetSchedulesWithDetails(ctx context.Context, filter *irepository.ScheduleFilter) ([]dtos.ScheduleDTO, int64, error) {
	var results []dtos.ScheduleDTO
	var total int64

	query := r.db.WithContext(ctx).Table("content_schedules cs").
		Select(`
			cs.id as schedule_id,
			cs.content_channel_id,
			c.id as content_id,
			c.title as content_title,
			c.type as content_type,
			ch.id as channel_id,
			ch.name as channel_name,
			ch.code as channel_code,
			cs.scheduled_at,
			cs.status,
			cs.retry_count,
			cs.last_error,
			cs.executed_at,
			cs.created_at,
			cs.created_by,
			u.username as created_by_name
		`).
		Joins("JOIN content_channels cc ON cc.id = cs.content_channel_id").
		Joins("JOIN contents c ON c.id = cc.content_id").
		Joins("JOIN channels ch ON ch.id = cc.channel_id").
		Joins("LEFT JOIN users u ON u.id = cs.created_by").
		Where("cs.deleted_at IS NULL")

	if filter.Status != nil {
		query = query.Where("cs.status = ?", filter.Status.String())
	}
	if filter.ChannelID != nil {
		query = query.Where("ch.id = ?", *filter.ChannelID)
	}
	if filter.From != nil {
		query = query.Where("cs.scheduled_at >= ?", *filter.From)
	}
	if filter.To != nil {
		query = query.Where("cs.scheduled_at < ?", *filter.To)
	}

	// Count total
	countQuery := r.db.WithContext(ctx).Table("content_schedules cs").
		Joins("JOIN content_channels cc ON cc.id = cs.content_channel_id").
		Joins("JOIN contents c ON c.id = cc.content_id").
		Joins("JOIN channels ch ON ch.id = cc.channel_id").
		Where("cs.deleted_at IS NULL")

	if filter.Status != nil {
		countQuery = countQuery.Where("cs.status = ?", filter.Status.String())
	}
	if filter.ChannelID != nil {
		countQuery = countQuery.Where("ch.id = ?", *filter.ChannelID)
	}
	if filter.From != nil {
		countQuery = countQuery.Where("cs.scheduled_at >= ?", *filter.From)
	}
	if filter.To != nil {
		countQuery = countQuery.Where("cs.scheduled_at < ?", *filter.To)
	}

	if err := countQuery.Count(&total).Error; err != nil {
		zap.L().Error("Failed to count schedules with details", zap.Error(err))
		return nil, 0, err
	}

	offset := (filter.PageNumber - 1) * filter.PageSize
	if err := query.Offset(offset).Limit(filter.PageSize).Order("cs.scheduled_at ASC").Scan(&results).Error; err != nil {
		zap.L().Error("Failed to get schedules with details", zap.Error(err))
		return nil, 0, err
	}

	return results, total, nil
}

// GetUpcomingSchedules returns upcoming schedules within a time range
func (r *contentScheduleRepository) GetUpcomingSchedules(ctx context.Context, from, to time.Time, limit int) ([]*model.ContentSchedule, error) {
	var schedules []*model.ContentSchedule
	if err := r.db.WithContext(ctx).
		Where("status = ?", enum.ScheduleStatusPending.String()).
		Where("scheduled_at >= ?", from).
		Where("scheduled_at < ?", to).
		Where("deleted_at IS NULL").
		Order("scheduled_at ASC").
		Limit(limit).
		Find(&schedules).Error; err != nil {
		zap.L().Error("Failed to get upcoming schedules", zap.Error(err))
		return nil, err
	}
	return schedules, nil
}

// CancelScheduleByContentChannelID cancels a schedule by content channel ID
func (r *contentScheduleRepository) CancelScheduleByContentChannelID(ctx context.Context, contentChannelID uuid.UUID) error {
	now := time.Now()
	if err := r.db.WithContext(ctx).
		Model(&model.ContentSchedule{}).
		Where("content_channel_id = ?", contentChannelID).
		Where("status = ?", enum.ScheduleStatusPending.String()).
		Where("deleted_at IS NULL").
		Updates(map[string]interface{}{
			"status":     enum.ScheduleStatusCancelled.String(),
			"updated_at": now,
		}).Error; err != nil {
		zap.L().Error("Failed to cancel schedule by content channel ID", zap.Error(err))
		return err
	}
	return nil
}

// GetScheduleByIDWithDetails returns a single schedule with full details
func (r *contentScheduleRepository) GetScheduleByIDWithDetails(ctx context.Context, id uuid.UUID) (*dtos.ScheduleDTO, error) {
	var result dtos.ScheduleDTO

	err := r.db.WithContext(ctx).Table("content_schedules cs").
		Select(`
			cs.id as schedule_id,
			cs.content_channel_id,
			c.id as content_id,
			c.title as content_title,
			c.type as content_type,
			ch.id as channel_id,
			ch.name as channel_name,
			ch.code as channel_code,
			cs.scheduled_at,
			cs.status,
			cs.retry_count,
			cs.last_error,
			cs.executed_at,
			cs.created_at,
			cs.created_by,
			u.username as created_by_name
		`).
		Joins("JOIN content_channels cc ON cc.id = cs.content_channel_id").
		Joins("JOIN contents c ON c.id = cc.content_id").
		Joins("JOIN channels ch ON ch.id = cc.channel_id").
		Joins("LEFT JOIN users u ON u.id = cs.created_by").
		Where("cs.id = ?", id).
		Where("cs.deleted_at IS NULL").
		Scan(&result).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		zap.L().Error("Failed to get schedule by ID with details", zap.Error(err))
		return nil, err
	}

	if result.ScheduleID == uuid.Nil {
		return nil, nil
	}

	return &result, nil
}
