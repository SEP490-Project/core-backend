package model

import (
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type LimitedProduct struct {
	Id                    uuid.UUID `json:"id" gorm:"type:uuid;column:id;primaryKey;not null"`
	MaxStock              int       `json:"max_stock" gorm:"column:max_stock;not null"`
	IsFreeShipping        bool      `json:"is_free_shipping" gorm:"column:is_free_shipping;not null;default:false"`
	BoughtLimit           int       `json:"bought_limit" gorm:"column:bought_limit;not null;default:1"`
	PremiereDate          time.Time `json:"premiere_date" gorm:"column:premiere_date;not null"`
	AvailabilityStartDate time.Time `json:"availability_start_date" gorm:"column:availability_start_date;not null"`
	AvailabilityEndDate   time.Time `json:"availability_end_date" gorm:"column:availability_end_date;not null"`

	// Relationships
	Product Product `json:"product" gorm:"foreignKey:Id;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
}

func (LimitedProduct) TableName() string { return "limited_products" }

func (lp *LimitedProduct) BeforeCreate(tx any) (err error) {
	if lp.MaxStock < 0 {
		zap.L().Warn("LimitedProduct MaxStock is less than 0, setting to 0")
		lp.MaxStock = 0
	}
	if lp.BoughtLimit < 1 {
		zap.L().Warn("LimitedProduct BoughtLimit is less than 1, setting to 1")
		lp.BoughtLimit = 1
	}

	return nil
}
