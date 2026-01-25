package model

import (
	"time"

	"gorm.io/gorm"
)

// Ward represents an administrative ward/commune (GHN Ward)
type Ward struct {
	Code           string `json:"code" gorm:"primaryKey;column:code"` // GHN WardCode
	DistrictID     int    `json:"district_id" gorm:"column:district_id;not null;index"`
	Name           string `json:"name" gorm:"column:name;not null"`
	SupportType    int    `json:"support_type" gorm:"column:support_type"`
	PickType       int    `json:"pick_type" gorm:"column:pick_type"`
	DeliverType    int    `json:"deliver_type" gorm:"column:deliver_type"`
	GovernmentCode string `json:"government_code" gorm:"column:government_code"`
	IsEnable       int    `json:"is_enable" gorm:"column:is_enable"`
	CanUpdateCOD   bool   `json:"can_update_cod" gorm:"column:can_update_cod"`
	Status         int    `json:"status" gorm:"column:status"`

	CreatedAt time.Time      `json:"created_at" gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time      `json:"updated_at" gorm:"column:updated_at;autoUpdateTime"`
	DeletedAt gorm.DeletedAt `json:"deleted_at" gorm:"column:deleted_at;index"`

	// Relations
	District District `json:"district,omitempty" gorm:"foreignKey:DistrictID;references:ID"`
}

func (Ward) TableName() string { return "wards" }
