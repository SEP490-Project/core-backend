package model

import (
	"core-backend/internal/domain/enum"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ModifiedHistory struct {
	ID            uuid.UUID              `json:"id" gorm:"type:uuid;column:id;primaryKey;default"`
	ReferenceID   uuid.UUID              `json:"reference_id" gorm:"type:uuid;column:reference_id"`
	ReferenceType enum.ModifiedType      `json:"reference_type" gorm:"type:varchar(50);column:reference_type;not null;check:reference_type in ('CONTRACT','CAMPAIGN','MILESTONE','TASK','CONTENT','PRODUCT','BLOG')"`
	Operation     enum.ModifiedOperation `json:"operation" gorm:"type:varchar(50);column:operation;not null;check:operation in ('CREATE','UPDATE','DELETE')"`
	Status        enum.ModifiedStatus    `json:"status" gorm:"type:varchar(50);column:status;not null;check:status in ('IN_PROGRESS','COMPLETED','FAILED')"`
	Description   string                 `json:"description" gorm:"type:text;column:description;not null"`
	ChangedByID   *uuid.UUID             `json:"changed_by" gorm:"type:uuid;column:changed_by"`
	ChangedAt     *time.Time             `json:"changed_at" gorm:"type:timestamptz;column:changed_at;autoUpdateTime"`
}

func (ModifiedHistory) TableName() string { return "modified_histories" }

func (m *ModifiedHistory) BeforeCreate(tx *gorm.DB) error {
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}

	return nil
}
