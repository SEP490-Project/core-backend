package model

import (
	"core-backend/internal/domain/enum"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type Task struct {
	ID                uuid.UUID       `json:"id" gorm:"type:uuid;column:id;primaryKey;default"`
	MilestoneID       uuid.UUID       `json:"milestone_id" gorm:"type:uuid;column:milestone_id;not null"`
	Name              string          `json:"name" gorm:"column:name;not null"`
	Description       datatypes.JSON  `json:"description" gorm:"column:description;type:jsonb"`
	Deadline          time.Time       `json:"deadline" gorm:"column:deadline;not null"`
	Type              enum.TaskType   `json:"type" gorm:"column:type;not null;check:type in ('PRODUCT', 'CONTENT', 'EVENT', 'OTHER')"`
	Status            enum.TaskStatus `json:"status" gorm:"column:status;not null;check:status in ('TODO', 'IN_PROGRESS', 'CANCELLED', 'RECAP', 'DONE')"`
	AssignedToID      *uuid.UUID      `json:"assigned_to" gorm:"type:uuid;column:assigned_to"`
	ScopeOfWorkItemID *string         `json:"scope_of_work_item_id" gorm:"type:varchar(50);column:scope_of_work_item_id"`
	CreatedAt         time.Time       `json:"created_at" gorm:"column:created_at;autoCreateTime"`
	UpdatedAt         time.Time       `json:"updated_at" gorm:"column:updated_at;autoUpdateTime"`
	DeletedAt         gorm.DeletedAt  `json:"deleted_at" gorm:"index;column:deleted_at"`
	CreatedByID       uuid.UUID       `json:"created_by" gorm:"type:uuid;column:created_by;not null"`
	UpdatedByID       *uuid.UUID      `json:"updated_by" gorm:"type:uuid;column:updated_by"`

	// Relationships
	AssignedStaff *User      `json:"-" gorm:"foreignKey:AssignedToID"`
	Milestone     *Milestone `json:"-" gorm:"foreignKey:MilestoneID"`
	Product       *Product   `json:"product" gorm:"foreignKey:TaskID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	Contents      []*Content `json:"contents" gorm:"foreignKey:TaskID;references:ID"`
}

func (Task) TableName() string { return "tasks" }

func (t *Task) BeforeCreate(_ *gorm.DB) (err error) {
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}
	// default status if not set
	if !t.Status.IsValid() {
		t.Status = enum.TaskStatusToDo
	}
	return nil
}

func (t *Task) AfterSave(tx *gorm.DB) (err error) {
	return t.SyncMilestoneProgress(tx)
}

func (t *Task) AfterDelete(tx *gorm.DB) (err error) {
	return t.SyncMilestoneProgress(tx)
}

// Validate ensures the Task has a valid enum combination before persisting.
func (t *Task) Validate() error {
	if !t.Type.IsValid() {
		return gorm.ErrInvalidData
	}
	if !t.Status.IsValid() {
		return gorm.ErrInvalidData
	}
	return nil
}

func (t *Task) SyncMilestoneProgress(tx *gorm.DB) error {
	percentageExpr := `
		COALESCE(
			(
				SELECT 
					(COUNT(*) FILTER (WHERE status = 'DONE')::NUMERIC / NULLIF(COUNT(*), 0)::NUMERIC) * 100
				FROM tasks 
				WHERE tasks.milestone_id = milestones.id 
				AND tasks.deleted_at IS NULL
			), 
			0
		)
	`

	completedAtExpr := `
		CASE 
			WHEN (
				SELECT COUNT(*) > 0 AND COUNT(*) = COUNT(*) FILTER (WHERE status = 'DONE')
				FROM tasks 
				WHERE tasks.milestone_id = milestones.id 
				AND tasks.deleted_at IS NULL
			) 
			THEN COALESCE(completed_at, NOW()) 
			ELSE NULL 
		END
	`

	return tx.Model(&Milestone{}).
		Where("id = ?", t.MilestoneID).
		Updates(map[string]interface{}{
			"completion_percentage": gorm.Expr(percentageExpr),
			"completed_at":          gorm.Expr(completedAtExpr),
		}).Error
}
