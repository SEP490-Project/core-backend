// Package model defines the data structures for application configuration settings.
package model

import (
	"core-backend/internal/domain/enum"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Config struct {
	ID          uuid.UUID            `json:"id" gorm:"type:uuid;column:id;primaryKey;default"`
	Key         string               `json:"key" gorm:"type:varchar(255);column:key;not null"`
	ValueType   enum.ConfigValueType `json:"value_type" gorm:"type:varchar(50);column:value_type;not null;check:value_type IN ('STRING', 'NUMBER', 'BOOLEAN', 'JSON')"`
	Value       string               `json:"value" gorm:"type:text;column:value;not null"`
	Description *string              `json:"description" gorm:"type:text;column:description"`
	CreatedAt   int64                `json:"created_at" gorm:"column:created_at;autoCreateTime"`
	UpdatedAt   int64                `json:"updated_at" gorm:"column:updated_at;autoUpdateTime"`
	UpdatedByID *uuid.UUID           `json:"updated_by" gorm:"type:uuid;column:updated_by"`
	DeletedAt   gorm.DeletedAt       `json:"deleted_at" gorm:"column:deleted_at;index"`
}

func (Config) TableName() string { return "configs" }

func (c *Config) BeforeCreate(tx *gorm.DB) error {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	return nil
}
