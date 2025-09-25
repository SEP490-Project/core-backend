// Package model defines the data structures for application configuration settings.
package model

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Config struct {
	ID          uuid.UUID      `json:"id" gorm:"type:uuid;column:id;primaryKey;default"`
	Key         string         `json:"key" gorm:"type:varchar(255);column:key;not null"`
	Value       string         `json:"value" gorm:"type:text;column:value;not null"`
	Description *string        `json:"description" gorm:"type:text;column:description"`
	CreatedAt   int64          `json:"created_at" gorm:"column:created_at;autoCreateTime"`
	UpdatedAt   int64          `json:"updated_at" gorm:"column:updated_at;autoUpdateTime"`
	DeletedAt   gorm.DeletedAt `json:"deleted_at" gorm:"column:deleted_at;index"`
}

func (Config) TableName() string { return "config" }

func (c *Config) BeforeCreate(tx *gorm.DB) error {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	return nil
}
