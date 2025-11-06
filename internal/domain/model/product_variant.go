package model

import (
	"core-backend/internal/domain/enum"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type ProductVariant struct {
	ID           uuid.UUID `json:"id" gorm:"type:uuid;column:id;primaryKey;default"`
	ProductID    uuid.UUID `json:"product_id" gorm:"type:uuid;column:product_id;not null"`
	Price        float64   `json:"price" gorm:"column:price;not null"`
	CurrentStock *int      `json:"current_stock" gorm:"column:current_stock"`

	Capacity        float64            `json:"capacity" gorm:"column:capacity"`
	CapacityUnit    enum.CapacityUnit  `json:"capacity_unit" gorm:"column:capacity_unit;not null;check:capacity_unit in ('ML', 'L', 'G', 'KG', 'OZ')"`
	ContainerType   enum.ContainerType `json:"container_type" gorm:"column:container_type;not null;check:container_type in ('BOTTLE', 'TUBE', 'JAR', 'STICK', 'PENCIL', 'COMPACT', 'PALLETE', 'SACHET', 'VIAL', 'ROLLER_BOTTLE')"`
	DispenserType   enum.DispenserType `json:"dispenser_type" gorm:"column:dispenser_type;not null;check:dispenser_type in ('PUMP', 'SPRAY', 'DROPPER', 'ROLL_ON', 'TWIST_UP', 'SQUEEZE', 'NONE')"`
	Uses            string             `json:"uses" gorm:"type:text;column:uses"`
	ManufactureDate *time.Time         `json:"manufacturing_date" gorm:"column:manufactring_date"`
	ExpiryDate      *time.Time         `json:"expiry_date" gorm:"column:expiry_date"`
	Instructions    string             `json:"instructions" gorm:"type:text;column:instructions"`
	IsDefault       bool               `json:"is_default" gorm:"column:is_default;not null;default:false"`
	CreatedAt       time.Time          `json:"created_at" gorm:"column:created_at"`
	UpdatedAt       time.Time          `json:"updated_at" gorm:"column:updated_at"`
	DeletedAt       gorm.DeletedAt     `json:"deleted_at" gorm:"column:deleted_at;index"`
	CreatedByID     uuid.UUID          `json:"created_by" gorm:"column:created_by;not null"`
	UpdatedByID     *uuid.UUID         `json:"updated_by" gorm:"column:updated_by"`

	Weight int `json:"weight" gorm:"column:weight"` // in grams
	Height int `json:"height" gorm:"column:height"` // in centimeters
	Length int `json:"length" gorm:"column:length"` // in centimeters
	Width  int `json:"width" gorm:"column:width"`   // in centimeters

	//Relationship ExistsByID
	Product         *Product                `json:"-" gorm:"foreignKey:ProductID"`
	Story           *ProductStory           `json:"story" gorm:"foreignKey:VariantID"`
	AttributeValues []VariantAttributeValue `json:"attributes" gorm:"foreignKey:VariantID"`
	Images          []VariantImage          `json:"images" gorm:"foreignKey:VariantID"`
}

func (ProductVariant) TableName() string { return "product_variants" }

func (pv *ProductVariant) BeforeCreate(tx *gorm.DB) (err error) {
	_ = tx
	if pv.ID == uuid.Nil {
		pv.ID = uuid.New()
	}
	if pv.Price < 0 {
		zap.L().Warn("Price is less than 0, setting to 0")
		pv.Price = 0
	}
	if pv.Capacity < 0 {
		zap.L().Warn("Capacity is less than 0, setting to 0")
		pv.Capacity = 0
	}
	if pv.CurrentStock != nil && *pv.CurrentStock < 0 {
		zap.L().Warn("CurrentStock is less than 0, setting to 0")
		*pv.CurrentStock = 0
	}

	return nil
}
