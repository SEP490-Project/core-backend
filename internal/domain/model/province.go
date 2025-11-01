package model

import (
	"time"

	"gorm.io/gorm"
)

// Province represents an administrative province (GHN Province)
// We use GHN's ProvinceID as the primary key to stay compatible with existing ShippingAddress fields.
type Province struct {
	ID           int    `json:"id" gorm:"primaryKey;column:id"` // GHN ProvinceID
	Name         string `json:"name" gorm:"column:name;not null"`
	CountryID    int    `json:"country_id" gorm:"column:country_id"`
	Code         string `json:"code" gorm:"column:code"`
	RegionID     int    `json:"region_id" gorm:"column:region_id"`
	RegionCPN    int    `json:"region_cpn" gorm:"column:region_cpn"`
	IsEnable     int    `json:"is_enable" gorm:"column:is_enable"`
	CanUpdateCOD bool   `json:"can_update_cod" gorm:"column:can_update_cod"`
	Status       int    `json:"status" gorm:"column:status"`

	CreatedAt time.Time      `json:"created_at" gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time      `json:"updated_at" gorm:"column:updated_at;autoUpdateTime"`
	DeletedAt gorm.DeletedAt `json:"deleted_at" gorm:"column:deleted_at;index"`

	// Relations
	Districts []District `json:"districts,omitempty" gorm:"foreignKey:ProvinceID;references:ID"`
}

func (Province) TableName() string { return "provinces" }
