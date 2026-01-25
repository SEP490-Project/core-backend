package model

import (
	"time"

	"gorm.io/gorm"
)

// District represents an administrative district (GHN District)
type District struct {
	ID             int    `json:"id" gorm:"primaryKey;column:id"` // GHN DistrictID
	ProvinceID     int    `json:"province_id" gorm:"column:province_id;not null;index"`
	Name           string `json:"name" gorm:"column:name;not null"`
	Code           string `json:"code" gorm:"column:code"`
	Type           int    `json:"type" gorm:"column:type"`
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
	Province Province `json:"province,omitempty" gorm:"foreignKey:ProvinceID;references:ID"`
}

func (District) TableName() string { return "districts" }
