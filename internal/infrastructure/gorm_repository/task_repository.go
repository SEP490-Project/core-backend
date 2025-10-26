package gormrepository

import (
	"context"
	"core-backend/internal/application/dto/dtos"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/domain/model"
	"core-backend/pkg/utils"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type TaskRepository struct {
	*genericRepository[model.Task]
}

// GetListTasks implements irepository.TaskRepository.
func (r *TaskRepository) GetListTasks(ctx context.Context, filter *requests.TaskFilterRequest) (data []dtos.TaskListDTO, total int64, err error) {
	filterQuery := func(db *gorm.DB) *gorm.DB {
		if filter.CreatedByID != nil {
			db = db.Where("created_by_id = ?", *filter.CreatedByID)
		}
		if filter.AssignedToID != nil {
			db = db.Where("a.id = ?", *filter.AssignedToID)
		}
		if filter.MilestoneID != nil {
			db = db.Where("m.id = ?", *filter.MilestoneID)
		}
		if filter.CampaignID != nil {
			db = db.Where("c.id = ?", *filter.CampaignID)
		}
		if filter.ContractID != nil {
			db = db.Where("c.contract_id = ?", *filter.ContractID)
		}
		if filter.DeadlineFromDate != nil {
			db = db.Where("tasks.deadline >= ?", *filter.DeadlineFromDate)
		}
		if filter.DeadlineToDate != nil {
			db = db.Where("tasks.deadline <= ?", *filter.DeadlineToDate)
		}
		if filter.UpdatedFromDate != nil {
			db = db.Where("tasks.updated_at >= ? or tasks.created_at >= ?", *filter.UpdatedFromDate, *filter.UpdatedFromDate)
		}
		if filter.UpdatedToDate != nil {
			db = db.Where("tasks.updated_at <= ? or tasks.created_at <= ?", *filter.UpdatedToDate, *filter.UpdatedToDate)
		}
		if filter.Status != nil {
			db = db.Where("tasks.status = ?", *filter.Status)
		}
		if filter.Type != nil {
			db = db.Where("tasks.type = ?", *filter.Type)
		}

		sortBy := filter.SortBy
		if sortBy == "" {
			sortBy = "created_at"
		}
		sortOrder := strings.ToLower(filter.SortOrder)
		if sortOrder != "asc" && sortOrder != "desc" {
			sortOrder = "desc"
		}
		db = db.Order(fmt.Sprintf("%s %s", sortBy, sortOrder))

		return db
	}

	findQuery := r.db.WithContext(ctx).Model(new(model.Task))
	findQuery = filterQuery(findQuery)

	if filter.Limit > 0 && filter.Page > 0 {
		pageSize := min(filter.Limit, 100)
		findQuery = findQuery.Offset((filter.Page - 1) * pageSize).Limit(pageSize)
	}

	findQuery = findQuery.
		Select(
			"tasks.id",
			"tasks.name",
			"tasks.deadline",
			"tasks.type",
			"tasks.status",
			"a.id as assigned_to_id",
			"a.full_name as assigned_to_name",
			"a.role as assigned_to_role",
			"tasks.created_at",
			"tasks.updated_at",
			"m.id as milestone_id",
			"c.id as campaign_id",
			"c.contract_id as contract_id").
		Joins("LEFT JOIN users AS a ON tasks.assigned_to = a.id").
		Joins("LEFT JOIN milestones AS m ON tasks.milestone_id = m.id").
		Joins("LEFT JOIN campaigns AS c ON m.campaign_id = c.id").
		Find(&data)

	if findQuery.Error != nil {
		return []dtos.TaskListDTO{}, 0, findQuery.Error
	}

	countQuery := r.db.WithContext(ctx).Model(new(model.Task)).
		Joins("LEFT JOIN users AS a ON tasks.assigned_to = a.id").
		Joins("LEFT JOIN milestones AS m ON tasks.milestone_id = m.id").
		Joins("LEFT JOIN campaigns AS c ON m.campaign_id = c.id")
	countQuery = filterQuery(countQuery)
	if err := countQuery.Count(&total).Error; err != nil {
		return []dtos.TaskListDTO{}, 0, err
	}

	return data, total, nil
}

// GetDetailTask implements irepository.TaskRepository.
func (r *TaskRepository) GetDetailTask(ctx context.Context, taskID uuid.UUID) (*dtos.TaskDetailDTO, error) {
	var (
		data       dtos.TaskDetailDTO
		contentIDs []uuid.UUID
		productIDs []uuid.UUID
	)

	findQuery := func(ctx context.Context) error {
		return r.db.
			WithContext(ctx).
			Model(new(model.Task)).
			Joins("LEFT JOIN users AS a ON tasks.assigned_to = a.id").
			Joins("LEFT JOIN users AS c ON tasks.created_by = c.id").
			Joins("LEFT JOIN users AS u ON tasks.updated_by = u.id").
			Joins("LEFT JOIN milestones AS m ON tasks.milestone_id = m.id").
			Joins("LEFT JOIN campaigns AS ca ON m.campaign_id = ca.id").
			Where("tasks.id = ?", taskID).
			Select("tasks.id",
				"tasks.name",
				"tasks.description",
				"tasks.deadline",
				"tasks.type",
				"tasks.status",
				"a.id as assigned_to_id",
				"a.full_name as assigned_to_name",
				"a.role as assigned_to_role",
				"c.id as created_by_id",
				"c.full_name as created_by_name",
				"c.role as created_by_role",
				"u.id as updated_by_id",
				"u.full_name as updated_by_name",
				"u.role as updated_by_role",
				"tasks.created_at",
				"tasks.updated_at",
				"m.id as milestone_id",
				"ca.id as campaign_id",
				"ca.contract_id as contract_id").
			First(&data).
			Error
	}
	contenIDsQuery := func(ctx context.Context) error {
		return r.db.WithContext(ctx).
			Model(&model.Content{}).
			Where("task_id = ?", taskID).
			Pluck("id", &contentIDs).
			Error
	}
	productIDsQuery := func(ctx context.Context) error {
		return r.db.WithContext(ctx).
			Model(&model.Product{}).
			Where("task_id = ?", taskID).
			Pluck("id", &productIDs).
			Error
	}

	if err := utils.RunParallel(ctx, 0, findQuery, contenIDsQuery, productIDsQuery); err != nil {
		return nil, err
	}

	data.ContentIDs = contentIDs
	data.ProductIDs = productIDs
	return &data, nil
}

func NewTaskRepository(db *gorm.DB) irepository.TaskRepository {
	return &TaskRepository{
		genericRepository: &genericRepository[model.Task]{db: db},
	}
}
