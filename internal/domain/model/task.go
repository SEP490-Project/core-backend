package model

import (
	"core-backend/internal/domain/enum"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type Task struct {
	ID           uuid.UUID       `json:"id" gorm:"type:uuid;column:id;primaryKey;default"`
	MilestoneID  uuid.UUID       `json:"milestone_id" gorm:"type:uuid;column:milestone_id;not null"`
	Name         string          `json:"name" gorm:"column:name;not null"`
	Description  datatypes.JSON  `json:"description" gorm:"column:description;type:jsonb"`
	Deadline     time.Time       `json:"deadline" gorm:"column:deadline;not null"`
	Type         enum.TaskType   `json:"type" gorm:"column:type;not null;check:type in ('PRODUCT', 'CONTENT', 'EVENT', 'OTHER')"`
	Status       enum.TaskStatus `json:"status" gorm:"column:status;not null;check:status in ('TODO', 'IN_PROGRESS', 'CANCELLED', 'RECAP', 'DONE')"`
	AssignedToID *uuid.UUID      `json:"assigned_to" gorm:"type:uuid;column:assigned_to"`
	CreatedAt    time.Time       `json:"created_at" gorm:"column:created_at;autoCreateTime"`
	UpdatedAt    time.Time       `json:"updated_at" gorm:"column:updated_at;autoUpdateTime"`
	DeletedAt    gorm.DeletedAt

	// Relationships
	Milestone *Milestone `json:"-" gorm:"foreignKey:MilestoneID"`
}

func (Task) TableName() string { return "tasks" }

func (t *Task) BeforeCreate(tx *gorm.DB) (err error) {
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}
	// default status if not set
	if !t.Status.IsValid() {
		t.Status = enum.TaskStatusToDo
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
