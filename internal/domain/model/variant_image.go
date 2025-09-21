package model

import (
	"time"
)

type VariantImage struct {
	ID        int       `gorm:"primaryKey;column:id"`
	VariantID int       `gorm:"column:variant_id;not null"`
	ImageURL  string    `gorm:"column:image_url;not null"`
	AltText   string    `gorm:"column:alt_text"`
	IsPrimary bool      `gorm:"column:is_primary"`
	CreatedAt time.Time `gorm:"column:created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
	IsDeleted bool      `gorm:"column:is_deleted"`
}

func (VariantImage) TableName() string { return "variant_image" }
