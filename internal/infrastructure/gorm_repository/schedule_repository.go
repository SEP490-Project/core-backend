package gormrepository

import (
	"context"
	"core-backend/internal/application/dto/dtos"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
	"core-backend/pkg/utils"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type scheduleRepository struct {
	*genericRepository[model.Schedule]
}

// NewScheduleRepository creates a new schedule repository
func NewScheduleRepository(db *gorm.DB) irepository.ScheduleRepository {
	return &scheduleRepository{genericRepository: &genericRepository[model.Schedule]{db: db}}
}

// GetByReferenceID returns a schedule by reference ID
func (r *scheduleRepository) GetByReferenceID(ctx context.Context, referenceID uuid.UUID) (*model.Schedule, error) {
	var schedule model.Schedule
	if err := r.db.WithContext(ctx).
		Where("reference_id = ?", referenceID).
		Where("deleted_at IS NULL").
		First(&schedule).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		zap.L().Error("Failed to get schedule by reference ID", zap.Error(err))
		return nil, err
	}
	return &schedule, nil
}

// GetPendingByReferenceID returns all pending schedules for a reference ID
func (r *scheduleRepository) GetPendingByReferenceID(ctx context.Context, referenceID uuid.UUID) ([]*model.Schedule, error) {
	var schedules []*model.Schedule
	if err := r.db.WithContext(ctx).
		Where("reference_id = ?", referenceID).
		Where("status = ?", enum.ScheduleStatusPending.String()).
		Where("deleted_at IS NULL").
		Find(&schedules).Error; err != nil {
		zap.L().Error("Failed to get pending schedules by reference ID", zap.Error(err))
		return nil, err
	}
	return schedules, nil
}

// GetPendingSchedules returns all pending schedules that should be processed
func (r *scheduleRepository) GetPendingSchedules(ctx context.Context, before time.Time) ([]*model.Schedule, error) {
	var schedules []*model.Schedule
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
func (r *scheduleRepository) GetSchedulesByStatus(ctx context.Context, status enum.ScheduleStatus, pageSize, pageNumber int) ([]*model.Schedule, int64, error) {
	var schedules []*model.Schedule
	var total int64

	query := r.db.WithContext(ctx).Model(&model.Schedule{}).
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

// GetSchedulesWithDetails returns schedules with basic details (generic, no JOINs)
func (r *scheduleRepository) GetSchedulesWithDetails(ctx context.Context, filter *requests.ScheduleFilterRequest) ([]dtos.ScheduleDTO, int64, error) {
	var results []dtos.ScheduleDTO
	var total int64

	predicateQuery := func(db *gorm.DB) *gorm.DB {
		db = db.
			Joins("LEFT JOIN users u ON u.id = s.created_by").
			Where("s.deleted_at IS NULL")
		if filter.Status != nil {
			db = db.Where("s.status = ?", filter.Status.String())
		}
		if filter.ReferenceID != nil {
			db = db.Where("s.reference_id = ?", *filter.ReferenceID)
		}
		if filter.ReferenceType != nil && filter.ReferenceType.IsValid() {
			db = db.Where("s.type = ?", *filter.ReferenceType)
		}
		if filter.CreatedBy != nil {
			db = db.Where("s.created_by = ?", *filter.CreatedBy)
		}
		if filter.FromDate != nil {
			fromDate := utils.ParseLocalTimeWithFallback(*filter.FromDate, utils.TimeFormat)
			if fromDate != nil {
				db = db.Where("s.scheduled_at >= ?", fromDate)
			}
		}
		if filter.ToDate != nil {
			toDate := utils.ParseLocalTimeWithFallback(*filter.ToDate, utils.TimeFormat)
			if toDate != nil {
				db = db.Where("s.scheduled_at < ?", toDate)
			}
		}
		return db
	}

	// Count total
	if err := predicateQuery(r.db.WithContext(ctx).Table("schedules s")).Count(&total).Error; err != nil {
		zap.L().Error("Failed to count schedules with details", zap.Error(err))
		return nil, 0, err
	}

	query := predicateQuery(r.db.WithContext(ctx).Table("schedules s")).
		Select(`
			s.id as schedule_id,
			s.reference_id,
			s.reference_type,
			s.type,
			s.scheduled_at,
			s.status,
			s.retry_count,
			s.last_error,
			s.executed_at,
			s.created_at,
			s.created_by,
			s.updated_at,
			u.username as created_by_name
		`).
		Offset((filter.Page - 1) * filter.Limit).
		Limit(filter.Limit).
		Order("s.scheduled_at ASC")
	if err := query.Scan(&results).Error; err != nil {
		zap.L().Error("Failed to get schedules with details", zap.Error(err))
		return nil, 0, err
	}

	return results, total, nil
}

// GetContentSchedulesWithDetails returns content-specific schedules with full details
func (r *scheduleRepository) GetContentSchedulesWithDetails(ctx context.Context, filter *requests.ScheduleFilterRequest) ([]dtos.ScheduleDTO, int64, error) {
	var rawResults []dtos.ContentScheduleRawDTO
	var total int64

	predicateQuery := func(db *gorm.DB) *gorm.DB {
		db = db.
			Joins("JOIN content_channels cc ON cc.id = s.reference_id").
			Joins("JOIN contents c ON c.id = cc.content_id").
			Joins("JOIN channels ch ON ch.id = cc.channel_id").
			Joins("LEFT JOIN users u ON u.id = s.created_by").
			Where("s.deleted_at IS NULL").
			Where("s.type = ?", enum.ScheduleTypeContentPublish)
		if filter.Status != nil {
			db = db.Where("s.status = ?", filter.Status.String())
		}
		if filter.ReferenceID != nil {
			db = db.Where("s.reference_id = ?", *filter.ReferenceID)
		}
		if filter.CreatedBy != nil {
			db = db.Where("s.created_by = ?", *filter.CreatedBy)
		}
		if filter.ContentID != nil {
			db = db.Where("c.id = ?", *filter.ContentID)
		}
		if filter.ChannelID != nil {
			db = db.Where("ch.id = ?", *filter.ChannelID)
		}
		if filter.FromDate != nil {
			fromDate := utils.ParseLocalTimeWithFallback(*filter.FromDate, utils.TimeFormat)
			if fromDate != nil {
				db = db.Where("s.scheduled_at >= ?", fromDate)
			}
		}
		if filter.ToDate != nil {
			toDate := utils.ParseLocalTimeWithFallback(*filter.ToDate, utils.TimeFormat)
			if toDate != nil {
				db = db.Where("s.scheduled_at < ?", toDate)
			}
		}
		return db
	}

	// Count total
	if err := predicateQuery(r.db.WithContext(ctx).Table("schedules s")).Count(&total).Error; err != nil {
		zap.L().Error("Failed to count content schedules with details", zap.Error(err))
		return nil, 0, err
	}

	query := predicateQuery(r.db.WithContext(ctx).Table("schedules s")).
		Select(`
			s.id as schedule_id,
			s.reference_id,
			s.reference_type,
			s.type,
			s.reference_id as content_channel_id,
			c.id as content_id,
			c.title as content_title,
			c.type as content_type,
			ch.id as channel_id,
			ch.name as channel_name,
			ch.code as channel_code,
			ch.platform as platform,
			c.thumbnail as thumbnail_url,
			s.scheduled_at,
			s.status,
			s.retry_count,
			s.last_error,
			s.executed_at,
			s.created_at,
			s.created_by,
			s.updated_at,
			u.username as created_by_name
		`).
		Offset((filter.Page - 1) * filter.Limit).
		Limit(filter.Limit).
		Order("s.scheduled_at ASC")
	if err := query.Scan(&rawResults).Error; err != nil {
		zap.L().Error("Failed to get content schedules with details", zap.Error(err))
		return nil, 0, err
	}

	// Convert raw results to structured DTOs
	results := make([]dtos.ScheduleDTO, len(rawResults))
	for i, raw := range rawResults {
		results[i] = *raw.ToScheduleDTO()
	}

	return results, total, nil
}

// GetUpcomingSchedules returns upcoming schedules within a time range
func (r *scheduleRepository) GetUpcomingSchedules(ctx context.Context, from, to time.Time, limit int) ([]*model.Schedule, error) {
	var schedules []*model.Schedule
	if err := r.db.WithContext(ctx).
		Where("status = ?", enum.ScheduleStatusPending.String()).
		Where("scheduled_at >= ? AND scheduled_at < ?", from, to).
		Where("deleted_at IS NULL").
		Order("scheduled_at ASC").
		Limit(limit).
		Find(&schedules).Error; err != nil {
		zap.L().Error("Failed to get upcoming schedules", zap.Error(err))
		return nil, err
	}
	return schedules, nil
}

// GetUpcomingSchedulesByType returns upcoming schedules of a specific type
func (r *scheduleRepository) GetUpcomingSchedulesByType(ctx context.Context, scheduleType enum.ScheduleType, from, to time.Time, limit int) ([]*model.Schedule, error) {
	var schedules []*model.Schedule
	if err := r.db.WithContext(ctx).
		Where("type = ?", scheduleType.String()).
		Where("status = ?", enum.ScheduleStatusPending.String()).
		Where("scheduled_at >= ? AND scheduled_at < ?", from, to).
		Where("deleted_at IS NULL").
		Order("scheduled_at ASC").
		Limit(limit).
		Find(&schedules).Error; err != nil {
		zap.L().Error("Failed to get upcoming schedules by type", zap.Error(err))
		return nil, err
	}
	return schedules, nil
}

// CancelScheduleByReferenceID cancels all pending schedules by reference ID
func (r *scheduleRepository) CancelScheduleByReferenceID(ctx context.Context, referenceID uuid.UUID) error {
	if err := r.db.WithContext(ctx).
		Model(&model.Schedule{}).
		Where("reference_id = ?", referenceID).
		Where("status = ?", enum.ScheduleStatusPending.String()).
		Update("status", enum.ScheduleStatusCancelled).Error; err != nil {
		zap.L().Error("Failed to cancel schedule by reference ID", zap.Error(err))
		return err
	}
	return nil
}

// GetScheduleByIDWithDetails returns a single schedule with basic details
func (r *scheduleRepository) GetScheduleByIDWithDetails(ctx context.Context, id uuid.UUID) (*dtos.ScheduleDTO, error) {
	var result dtos.ScheduleDTO

	if err := r.db.WithContext(ctx).Table("schedules s").
		Select(`
			s.id as schedule_id,
			s.reference_id,
			s.reference_type,
			s.type,
			s.scheduled_at,
			s.status,
			s.retry_count,
			s.last_error,
			s.executed_at,
			s.created_at,
			s.created_by,
			s.updated_at,
			u.username as created_by_name
		`).
		Joins("LEFT JOIN users u ON u.id = s.created_by").
		Where("s.id = ?", id).
		Where("s.deleted_at IS NULL").
		Scan(&result).Error; err != nil {
		zap.L().Error("Failed to get schedule details by ID", zap.Error(err))
		return nil, err
	}

	if result.ScheduleID == uuid.Nil {
		return nil, nil
	}

	return &result, nil
}

// GetContentScheduleByIDWithDetails returns a content schedule with full details
func (r *scheduleRepository) GetContentScheduleByIDWithDetails(ctx context.Context, id uuid.UUID) (*dtos.ScheduleDTO, error) {
	var raw dtos.ContentScheduleRawDTO

	if err := r.db.WithContext(ctx).Table("schedules s").
		Select(`
			s.id as schedule_id,
			s.reference_id,
			s.reference_type,
			s.type,
			s.reference_id as content_channel_id,
			c.id as content_id,
			c.title as content_title,
			c.type as content_type,
			ch.id as channel_id,
			ch.name as channel_name,
			ch.code as channel_code,
			ch.platform as platform,
			c.thumbnail as thumbnail_url,
			s.scheduled_at,
			s.status,
			s.retry_count,
			s.last_error,
			s.executed_at,
			s.created_at,
			s.created_by,
			s.updated_at,
			u.username as created_by_name
		`).
		Joins("JOIN content_channels cc ON cc.id = s.reference_id").
		Joins("JOIN contents c ON c.id = cc.content_id").
		Joins("JOIN channels ch ON ch.id = cc.channel_id").
		Joins("LEFT JOIN users u ON u.id = s.created_by").
		Where("s.id = ?", id).
		Where("s.deleted_at IS NULL").
		Scan(&raw).Error; err != nil {
		zap.L().Error("Failed to get content schedule details by ID", zap.Error(err))
		return nil, err
	}

	if raw.ScheduleID == uuid.Nil {
		return nil, nil
	}

	return raw.ToScheduleDTO(), nil
}

// GetSchedulesByContentID returns all schedules for a content ID
func (r *scheduleRepository) GetSchedulesByContentID(ctx context.Context, contentID uuid.UUID, status *enum.ScheduleStatus) ([]dtos.ScheduleDTO, error) {
	var rawResults []dtos.ContentScheduleRawDTO

	query := r.db.WithContext(ctx).Table("schedules s").
		Select(`
			s.id as schedule_id,
			s.reference_id,
			s.reference_type,
			s.type,
			s.reference_id as content_channel_id,
			c.id as content_id,
			c.title as content_title,
			c.type as content_type,
			ch.id as channel_id,
			ch.name as channel_name,
			ch.code as channel_code,
			ch.platform as platform,
			c.thumbnail as thumbnail_url,
			s.scheduled_at,
			s.status,
			s.retry_count,
			s.last_error,
			s.executed_at,
			s.created_at,
			s.created_by,
			s.updated_at,
			u.username as created_by_name
		`).
		Joins("JOIN content_channels cc ON cc.id = s.reference_id").
		Joins("JOIN contents c ON c.id = cc.content_id").
		Joins("JOIN channels ch ON ch.id = cc.channel_id").
		Joins("LEFT JOIN users u ON u.id = s.created_by").
		Where("c.id = ?", contentID).
		Where("s.type = ?", enum.ScheduleTypeContentPublish).
		Where("s.deleted_at IS NULL")

	if status != nil {
		query = query.Where("s.status = ?", status.String())
	}

	if err := query.Order("s.scheduled_at ASC").Scan(&rawResults).Error; err != nil {
		zap.L().Error("Failed to get schedules by content ID", zap.Error(err))
		return nil, err
	}

	// Convert raw results to structured DTOs
	results := make([]dtos.ScheduleDTO, len(rawResults))
	for i, raw := range rawResults {
		results[i] = *raw.ToScheduleDTO()
	}

	return results, nil
}

func (r *scheduleRepository) UpdateScheduleStatus(ctx context.Context, id uuid.UUID, status enum.ScheduleStatus, lastError *string) error {
	query := r.db.WithContext(ctx).Model(&model.Schedule{}).
		Where("id = ?", id).
		Where("deleted_at IS NULL")
	if lastError != nil {
		query = query.Update("last_error", *lastError)
	}
	if err := query.Update("status", status).Error; err != nil {
		zap.L().Error("Failed to update schedule status", zap.Error(err))
		return err
	}

	return nil
}
