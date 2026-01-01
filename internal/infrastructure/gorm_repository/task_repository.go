package gormrepository

import (
	"context"
	"core-backend/internal/application/dto/dtos"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/domain/constant"
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
	"core-backend/pkg/utils"
	"errors"
	"fmt"
	"strconv"
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
			db = db.Where("tasks.created_by = ?", *filter.CreatedByID)
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
			deadlineFromDate := utils.ParseLocalTimeWithFallback(*filter.DeadlineFromDate, utils.DateFormat)
			if deadlineFromDate != nil {
				db = db.Where("tasks.deadline >= ?", deadlineFromDate)
			}
		}
		if filter.DeadlineToDate != nil {
			deadlineToDate := utils.ParseLocalTimeWithFallback(*filter.DeadlineToDate, utils.DateFormat)
			if deadlineToDate != nil {
				db = db.Where("tasks.deadline <= ?", deadlineToDate)
			}
		}
		if filter.UpdatedFromDate != nil {
			updatedFromDate := utils.ParseLocalTimeWithFallback(*filter.UpdatedFromDate, utils.DateFormat)
			if updatedFromDate != nil {
				db = db.Where("tasks.updated_at >= ? or tasks.created_at >= ?", updatedFromDate, updatedFromDate)
			}
		}
		if filter.UpdatedToDate != nil {
			updatedToDate := utils.ParseLocalTimeWithFallback(*filter.UpdatedToDate, utils.DateFormat)
			if updatedToDate != nil {
				db = db.Where("tasks.updated_at <= ? or tasks.created_at <= ?", updatedToDate, updatedToDate)
			}
		}
		if filter.Status != nil {
			db = db.Where("tasks.status = ?", *filter.Status)
		}
		if filter.Type != nil {
			db = db.Where("tasks.type = ?", *filter.Type)
		}
		if filter.HasContent != nil {
			db = db.Where("(EXISTS (SELECT 1 FROM contents WHERE contents.task_id = tasks.id and contents.deleted_at is null) = ?)", *filter.HasContent)
		}
		if filter.HasProduct != nil {
			db = db.Where("(EXISTS (SELECT 1 FROM products WHERE products.task_id = tasks.id and products.deleted_at is null) = ?)", *filter.HasProduct)
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
			"c.name as campaign_name",
			"c.id as campaign_id",
			"c.contract_id as contract_id",
			"p.id as product_id",
			"p.status as child_status"). // <-- đây là product status
		Joins("LEFT JOIN users AS a ON tasks.assigned_to = a.id").
		Joins("LEFT JOIN milestones AS m ON tasks.milestone_id = m.id").
		Joins("LEFT JOIN campaigns AS c ON m.campaign_id = c.id").
		Joins("LEFT JOIN products AS p ON p.task_id = tasks.id"). // 1-1 nên chỉ có 1 product
		Find(&data)

	if findQuery.Error != nil {
		return []dtos.TaskListDTO{}, 0, findQuery.Error
	}
	// Count distinct tasks to avoid duplicates in case joins produce multiple rows
	countQuery := r.db.WithContext(ctx).Model(new(model.Task)).
		Joins("LEFT JOIN users AS a ON tasks.assigned_to = a.id").
		Joins("LEFT JOIN milestones AS m ON tasks.milestone_id = m.id").
		Joins("LEFT JOIN campaigns AS c ON m.campaign_id = c.id").
		Joins("LEFT JOIN products AS p ON p.task_id = tasks.id")
	countQuery = filterQuery(countQuery)

	if err := countQuery.Distinct("tasks.id").Count(&total).Error; err != nil {
		return nil, 0, err
	}

	return data, total, nil
}

// GetDetailTask implements irepository.TaskRepository.
func (r *TaskRepository) GetDetailTask(ctx context.Context, taskID uuid.UUID) (*dtos.TaskDetailDTO, error) {
	var (
		data         dtos.TaskDetailDTO
		contentInfos []dtos.ContentInfo
		productInfos []dtos.ProductInfo
		brandInfo    dtos.BrandInfo
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
				"ca.contract_id as contract_id",
				// JSON build for MilestoneDTO
				`CASE WHEN m.id IS NOT NULL THEN jsonb_build_object(
					'id', m.id,
					'description', m.description,
					'due_date', m.due_date,
					'completed_at', m.completed_at,
					'completion_percentage', m.completion_percentage,
					'status', m.status,
					'behind_schedule', m.behind_schedule
				) ELSE NULL END AS milestone_info`,
				// JSON build for CampaignDTO
				`CASE WHEN ca.id IS NOT NULL THEN jsonb_build_object(
					'id', ca.id,
					'name', ca.name,
					'description', ca.description,
					'start_date', ca.start_date,
					'end_date', ca.end_date,
					'status', ca.status,
					'type', ca.type
				) ELSE NULL END AS campaign_info`).
			First(&data).
			Error
	}
	contenIDsQuery := func(ctx context.Context) error {
		return r.db.WithContext(ctx).
			Model(&model.Content{}).
			Where("task_id = ?", taskID).
			Where("contents.deleted_at is null").
			Select("id", "title", "description", "type").
			Find(&contentInfos).
			Error
	}
	productIDsQuery := func(ctx context.Context) error {
		return r.db.WithContext(ctx).
			Model(&model.Product{}).
			Where("task_id = ?", taskID).
			Where("products.deleted_at is null").
			Select("id", "name", "type").
			Find(&productInfos).
			Error
	}
	brandQuery := func(ctx context.Context) error {
		return r.db.WithContext(ctx).
			Model(&model.Task{}).
			Joins("inner join milestones m on m.id = tasks.milestone_id").
			Joins("inner join campaigns c on c.id = m.campaign_id").
			Joins("inner join contracts con on con.id = c.contract_id").
			Joins("inner join brands b on b.id = con.brand_id").
			Where("tasks.id = ?", taskID).
			Where("b.deleted_at is null").
			Select("b.id", "b.name", "b.logo_url", "b.status").
			First(&brandInfo).
			Error
	}

	if err := utils.RunParallel(ctx, 0, findQuery, contenIDsQuery, productIDsQuery, brandQuery); err != nil {
		return nil, err
	}

	data.ContentInfos = contentInfos
	data.ProductInfos = productInfos
	data.BrandInfo = &brandInfo
	return &data, nil
}

func (r *TaskRepository) GetContractTrackingLinkByTaskID(ctx context.Context, taskID uuid.UUID) (string, uuid.UUID, error) {
	query := r.db.WithContext(ctx).Model(new(model.Task)).
		Joins("INNER JOIN milestones AS m ON tasks.milestone_id = m.id").
		Joins("INNER JOIN campaigns AS c ON m.campaign_id = c.id").
		Joins("INNER JOIN contracts AS ct ON c.contract_id = ct.id").
		Where("tasks.id = ?", taskID).
		Where("ct.type = ?", enum.ContractTypeAffiliate).
		Select(
			"ct.scope_of_work -> 'deliverables' ->> 'tracking_link' AS tracking_link",
			"ct.id AS contract_id",
		)
	var result struct {
		TrackingLink string    `json:"tracking_link"`
		ContractID   uuid.UUID `json:"contract_id"`
	}
	if err := query.Scan(&result).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", uuid.Nil, nil
		}
		return "", uuid.Nil, err
	}
	return result.TrackingLink, result.ContractID, nil
}

func (r *TaskRepository) GetTaskIDsByContractID(ctx context.Context, contractID uuid.UUID) (taskIDs []uuid.UUID, err error) {
	err = r.db.WithContext(ctx).Model(new(model.Task)).
		Joins("INNER JOIN milestones AS m ON tasks.milestone_id = m.id").
		Joins("INNER JOIN campaigns AS c ON m.campaign_id = c.id").
		Where("c.contract_id = ?", contractID).
		Select("tasks.id").
		Pluck("tasks.id", &taskIDs).Error
	return
}

func (r *TaskRepository) GetListTasksByIDs(ctx context.Context, taskIDs []uuid.UUID) (tasks []dtos.TaskWithScopeOfWorkID, err error) {
	var (
		taskList []dtos.TaskWithScopeOfWorkID
		// Slices to handle edge cases where a task has multiple products or contents
		productInfoMap = make(map[uuid.UUID][]dtos.ProductInfo) // map[TaskID] slice of ProductInfo
		contentInfoMap = make(map[uuid.UUID][]dtos.ContentInfo) // map[TaskID] slice of ContentInfo
	)
	taskFunc := func(ctx context.Context) error {
		err := r.db.WithContext(ctx).Model(new(model.Task)).
			Where("tasks.id IN ?", taskIDs).
			Where("tasks.deleted_at IS NULL").
			Select("tasks.id", "tasks.type", "tasks.status", "tasks.scope_of_work_item_id").
			Find(&taskList).Error
		if err != nil {
			return err
		}
		for i := range taskList {
			if taskList[i].ScopeOfWorkItemID == nil {
				continue
			}
			splittedID := strings.Split(*taskList[i].ScopeOfWorkItemID, "|")
			if len(splittedID) != 3 {
				continue
			}
			if contractID, idErr := uuid.Parse(splittedID[0]); idErr == nil {
				taskList[i].ContractID = &contractID
			}
			if itemType := constant.ScopeOfWorkIDType(splittedID[1]); itemType.IsValid() {
				taskList[i].ScopeOfWorkItemType = &itemType
			}
			if itemID, idErr := strconv.ParseInt(splittedID[2], 10, 8); idErr == nil {
				taskList[i].ItemID = utils.PtrOrNil(int8(itemID))
			}
		}

		return err
	}
	productFunc := func(ctx context.Context) error {
		var procucts []dtos.ProductInfo
		if err := r.db.WithContext(ctx).Model(new(model.Product)).
			Where("products.task_id IN ?", taskIDs).
			Where("products.deleted_at IS NULL").
			Select("products.task_id", "products.id", "products.name", "products.type").
			Find(&procucts).Error; err != nil {
			return err
		}
		for _, p := range procucts {
			if _, exists := productInfoMap[p.TaskID]; !exists {
				productInfoMap[p.TaskID] = make([]dtos.ProductInfo, 0)
			}
			productInfoMap[p.TaskID] = append(productInfoMap[p.TaskID], p)
		}
		return nil
	}
	contentFunc := func(ctx context.Context) error {
		var contents []dtos.ContentInfo
		if err := r.db.WithContext(ctx).Model(new(model.Content)).
			Where("contents.task_id IN ?", taskIDs).
			Where("contents.deleted_at IS NULL").
			Select("contents.task_id", "contents.id", "contents.title", "contents.description", "contents.type").
			Find(&contents).Error; err != nil {
			return err
		}
		for _, c := range contents {
			if _, exists := contentInfoMap[c.TaskID]; !exists {
				contentInfoMap[c.TaskID] = make([]dtos.ContentInfo, 0)
			}
			contentInfoMap[c.TaskID] = append(contentInfoMap[c.TaskID], c)
		}
		return nil
	}
	if err := utils.RunParallel(ctx, 3, taskFunc, productFunc, contentFunc); err != nil {
		return nil, err
	}
	// Map product and content info to tasks
	for i := range taskList {
		products, exists := productInfoMap[taskList[i].ID]
		if exists && len(products) > 0 {
			taskList[i].ProductInfo = &products[0]
		}
		contents, exists := contentInfoMap[taskList[i].ID]
		if exists && len(contents) > 0 {
			taskList[i].ContentInfo = &contents[0]
		}
	}

	return taskList, nil
}

func NewTaskRepository(db *gorm.DB) irepository.TaskRepository {
	return &TaskRepository{
		genericRepository: &genericRepository[model.Task]{db: db},
	}
}
