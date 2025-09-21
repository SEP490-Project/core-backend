package model

import (
	"time"
)

type OrderItem struct {
	ID                    int        `gorm:"primaryKey;column:id"`
	OrderID               int        `gorm:"column:order_id;not null"`
	VariantID             int        `gorm:"column:variant_id;not null"`
	Quantity              int        `gorm:"column:quantity;not null"`
	Subtotal              float64    `gorm:"column:subtotal;not null"`
	UnitPrice             float64    `gorm:"column:unit_price;not null"`
	Capacity              float64    `gorm:"column:capacity"`
	CapacityUnit          string     `gorm:"column:capacity_unit"`
	ContainerType         string     `gorm:"column:container_type"`
	DispenserType         string     `gorm:"column:dispenser_type"`
	Uses                  string     `gorm:"column:uses"`
	ManufactureDate       *time.Time `gorm:"column:manufactring_date"`
	ExpiryDate            *time.Time `gorm:"column:expiry_date"`
	Instructions          string     `gorm:"column:instructions"`
	AttributesDescription string     `gorm:"column:attributes_description;type:jsonb"`
	ItemStatus            string     `gorm:"column:item_status;not null"`
	UpdatedAt             time.Time  `gorm:"column:updated_at"`
}

func (OrderItem) TableName() string { return "order_item" }
