package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type ProductStory struct {
	ID        uuid.UUID      `json:"id" gorm:"type:uuid;column:id;primaryKey;default"`
	VariantID uuid.UUID      `json:"variant_id" gorm:"type:uuid;column:variant_id;not null"`
	Content   datatypes.JSON `json:"content" gorm:"column:content;type:jsonb"`
	CreatedAt time.Time      `json:"created_at" gorm:"column:created_at"`
	UpdatedAt time.Time      `json:"updated_at" gorm:"column:updated_at"`
	DeletedAt gorm.DeletedAt `json:"deleted_at" gorm:"column:deleted_at;index"`
}

func (ProductStory) TableName() string { return "product_story" }

func (ps *ProductStory) BeforeCreate(tx *gorm.DB) (err error) {
	if ps.ID == uuid.Nil {
		ps.ID = uuid.New()
	}

	return nil
}
