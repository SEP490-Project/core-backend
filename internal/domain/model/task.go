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

func (t *Task) BeforeUpdate(tx *gorm.DB) (err error) {
	var oldTask Task
	if err := tx.First(&oldTask, "id = ?", t.ID).Preload("Milestone", "Milestone.Tasks").Error; err != nil {
		return err
	}
	if oldTask.Status != t.Status {
		totalTasks := len(t.Milestone.Tasks)
		totalCompletedTasks := 0
		for _, task := range t.Milestone.Tasks {
			if task.Status == enum.TaskStatusDone {
				totalCompletedTasks++
			}
		}
		if t.Status == enum.TaskStatusDone {
			t.Milestone.CompletionPercentage = float64(totalCompletedTasks+1) / float64(totalTasks) * 100
		} else if oldTask.Status == enum.TaskStatusDone && t.Status != enum.TaskStatusDone {
			t.Milestone.CompletionPercentage = float64(totalCompletedTasks-1) / float64(totalTasks) * 100
		}
		if t.Milestone.CompletionPercentage < 0 {
			t.Milestone.CompletionPercentage = 0
		} else if t.Milestone.CompletionPercentage > 100 {
			t.Milestone.CompletionPercentage = 100
		}
	}

	return nil
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
