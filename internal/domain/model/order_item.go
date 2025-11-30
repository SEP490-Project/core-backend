package model

import (
	"core-backend/internal/domain/enum"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type OrderItem struct {
	ID        uuid.UUID `json:"id" gorm:"type:uuid;column:id;primaryKey;default"`
	OrderID   uuid.UUID `json:"order_id" gorm:"type:uuid;column:order_id;not null"`
	VariantID uuid.UUID `json:"variant_id" gorm:"type:uuid;column:variant_id;not null"`
	Quantity  int       `json:"quantity" gorm:"column:quantity;not null"`
	Subtotal  float64   `json:"subtotal" gorm:"column:subtotal;not null"`
	UnitPrice float64   `json:"unit_price" gorm:"column:unit_price;not null"`

	Capacity              *float64            `json:"capacity" gorm:"column:capacity"`
	CapacityUnit          *enum.CapacityUnit  `json:"capacity_unit" gorm:"column:capacity_unit"`
	ContainerType         *enum.ContainerType `json:"container_type" gorm:"type:varchar(255);column:container_type;check:container_type in ('BOTTLE', 'TUBE', 'JAR', 'STICK', 'PENCIL', 'COMPACT', 'PALLETE', 'SACHET', 'VIAL', 'ROLLER_BOTTLE')"`
	DispenserType         *enum.DispenserType `json:"dispenser_type" gorm:"type:varchar(255);column:dispenser_type;check:dispenser_type in ('PUMP', 'SPRAY', 'DROPPER', 'ROLL_ON', 'TWIST_UP', 'SQUEEZE', 'NONE')"`
	Uses                  *string             `json:"uses" gorm:"type:text;column:uses"`
	ManufactureDate       *time.Time          `json:"manufacturing_date" gorm:"column:manufacturing_date"`
	ExpiryDate            *time.Time          `json:"expiry_date" gorm:"column:expiry_date"`
	Instructions          *string             `json:"instructions" gorm:"type:text;column:instructions"`
	AttributesDescription *datatypes.JSON     `json:"attributes_description" gorm:"column:attributes_description;type:jsonb" swaggerignore:"true"`
	Weight                int                 `json:"weight" gorm:"column:weight"` // in grams
	Height                int                 `json:"height" gorm:"column:height"` // in centimeters
	Length                int                 `json:"length" gorm:"column:length"` // in centimeters
	Width                 int                 `json:"width" gorm:"column:width"`   // in centimeters

	//product fields
	ProductName string     `json:"product_name" gorm:"column:product_name;not null"`
	Description *string    `json:"description" gorm:"column:description"`
	Type        string     `json:"product_type" gorm:"column:product_type;not null"`
	BrandID     *uuid.UUID `json:"brand_id" gorm:"column:brand_id;"`
	CategoryID  uuid.UUID  `json:"category_id" gorm:"column:category_id;not null"`

	// Relationships
	Variant  ProductVariant   `json:"product_variant" gorm:"foreignKey:VariantID"`
	Order    *Order           `json:"-" gorm:"foreignKey:OrderID"`
	Brand    *Brand           `json:"brand" gorm:"foreignKey:BrandID" swaggerignore:"true"`
	Category *ProductCategory `json:"category" gorm:"foreignKey:CategoryID"`
}

func (OrderItem) TableName() string { return "order_items" }

func (ot *OrderItem) BeforeCreate(tx *gorm.DB) (err error) {
	if ot.ID == uuid.Nil {
		ot.ID = uuid.New()
	}
	if ot.Quantity < 1 {
		zap.L().Warn("Quantity is less than 1, setting to 1")
		ot.Quantity = 1
	}
	if ot.Subtotal < 0 {
		zap.L().Warn("Subtotal is less than 0, setting to 0")
		ot.Subtotal = 0
	}

	return nil
}
