package model

import (
	"time"
)

type ProductStory struct {
	ID        int       `gorm:"primaryKey;column:id"`
	VariantID int       `gorm:"column:variant_id;not null"`
	Content   string    `gorm:"column:content;type:jsonb;not null"`
	CreatedAt time.Time `gorm:"column:created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
	IsDeleted bool      `gorm:"column:is_deleted"`
}

func (ProductStory) TableName() string { return "product_story" }
