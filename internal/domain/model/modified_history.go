package model

import (
	"core-backend/internal/domain/enum"
	"time"

	"github.com/google/uuid"
)

type ModifiedHistory struct {
	ID            uuid.UUID         `json:"id" gorm:"type:uuid;column:id;primaryKey;default"`
	ReferenceID   uuid.UUID         `json:"reference_id" gorm:"type:uuid;column:reference_id;not null"`
	ReferenceType enum.ModifiedType `json:"reference_type" gorm:"type:varchar(50);column:reference_type;not null;check:reference_type in ('CAMPAIGN','MILESTONE','TASK','CONTENT','PRODUCT','BLOG')"`
	ChangeType    string            `json:"change_type" gorm:"type:varchar(50);column:change_type;not null"`
	Description   string            `json:"description" gorm:"type:text;column:description;not null"`
	ChangedByID   *uuid.UUID        `json:"changed_by" gorm:"type:uuid;column:changed_by"`
	ChangedAt     *time.Time        `json:"changed_at" gorm:"type:timestamptz;column:changed_at;default:current_timestamp"`
}

func (ModifiedHistory) TableName() string { return "modified_histories" }

func (m *ModifiedHistory) BeforeCreate() (err error) {
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}

	return nil
}
