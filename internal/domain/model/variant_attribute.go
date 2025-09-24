package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type VariantAttribute struct {
	ID          uuid.UUID      `json:"id" gorm:"type:uuid;column:id;primaryKey;default"`
	Ingredient  string         `json:"ingredient" gorm:"column:ingredient;not null"`
	Description *string        `json:"description" gorm:"column:description"`
	CreatedAt   time.Time      `json:"created_at" gorm:"column:created_at;autoCreateTime"`
	UpdatedAt   time.Time      `json:"updated_at" gorm:"column:updated_at;autoUpdateTime"`
	DeletedAt   gorm.DeletedAt `json:"deleted_at" gorm:"column:deleted_at;index"`
}

func (VariantAttribute) TableName() string { return "variant_attribute" }

func (va *VariantAttribute) BeforeCreate(tx *gorm.DB) (err error) {
	if va.ID == uuid.Nil {
		va.ID = uuid.New()
	}

	return nil
}
