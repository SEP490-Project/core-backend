package model

import (
	"core-backend/internal/domain/enum"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type Order struct {
	ID          uuid.UUID        `json:"id" gorm:"type:uuid;column:id;primaryKey;default"`
	UserID      uuid.UUID        `json:"user_id" gorm:"type:uuid;column:user_id;not null"`
	Status      enum.OrderStatus `json:"status" gorm:"column:status;not null"`
	TotalAmount float64          `json:"total_amount" gorm:"column:total_amount;not null"`

	// Copied shipping address fields (migration moved from a foreign key to flat columns)
	FullName      string    `json:"full_name" gorm:"column:full_name"`
	PhoneNumber   string    `json:"phone_number" gorm:"column:phone_number"`
	Email         string    `json:"email" gorm:"column:email"`
	Street        string    `json:"street" gorm:"column:street"`
	AddressLine2  string    `json:"address_line2" gorm:"column:address_line2"`
	City          string    `json:"city" gorm:"column:city"`
	GhnProvinceID int       `json:"ghn_province_id" gorm:"column:ghn_province_id"`
	GhnDistrictID int       `json:"ghn_district_id" gorm:"column:ghn_district_id"`
	GhnWardCode   string    `json:"ghn_ward_code" gorm:"column:ghn_ward_code"`
	ProvinceName  string    `json:"province_name" gorm:"column:province_name"`
	DistrictName  string    `json:"district_name" gorm:"column:district_name"`
	WardName      string    `json:"ward_name" gorm:"column:ward_name"`
	ShippingFee   int       `json:"shipping_fee" gorm:"column:shipping_fee;default:0"`
	CreatedAt     time.Time `json:"created_at" gorm:"column:created_at;autoCreateTime"`
	UpdatedAt     time.Time `json:"updated_at" gorm:"column:updated_at;autoUpdateTime"`
	//DeletedAt   gorm.DeletedAt   `json:"deleted_at" gorm:"column:deleted_at;index"`

	// Relationships
	User       User        `json:"-" gorm:"foreignKey:UserID"`
	OrderItems []OrderItem `json:"order_items" gorm:"foreignKey:OrderID"`
}

func (Order) TableName() string { return "orders" }

func (o *Order) BeforeCreate(tx *gorm.DB) (err error) {
	_ = tx
	if o.ID == uuid.Nil {
		o.ID = uuid.New()
	}
	if o.TotalAmount < 0 {
		zap.L().Warn("TotalAmount is less than 0, setting to 0")
		o.TotalAmount = 0
	}

	return nil
}
