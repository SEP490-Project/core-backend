package model

import (
	"core-backend/internal/domain/enum"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ShippingAddress struct {
	ID           uuid.UUID        `json:"id" gorm:"type:uuid;column:id;primaryKey;default"`
	UserID       uuid.UUID        `json:"user_id" gorm:"type:uuid;column:user_id;not null"`
	Type         enum.AddressType `json:"type" gorm:"column:type;not null;check:type in ('BILLING', 'SHIPPING')"`
	FullName     string           `json:"full_name" gorm:"column:full_name;not null"`
	PhoneNumber  string           `json:"phone_number" gorm:"column:phone_number;not null"`
	Email        string           `json:"email" gorm:"column:email;not null"`
	Street       string           `json:"street" gorm:"column:street;not null"`
	AddressLine2 *string          `json:"address_line_2" gorm:"column:address_line_2"`
	City         string           `json:"city" gorm:"column:city;not null"`
	State        *string          `json:"state" gorm:"column:state;not null"`
	PostalCode   string           `json:"postal_code" gorm:"column:postal_code;not null"`
	Country      string           `json:"country" gorm:"column:country;not null"`
	Company      *string          `json:"company" gorm:"column:company"`
	IsDefault    bool             `json:"is_default" gorm:"column:is_default;not null;default:false"`
	CreatedAt    time.Time        `json:"created_at" gorm:"column:created_at;autoCreateTime"`
	UpdatedAt    time.Time        `json:"updated_at" gorm:"column:updated_at;autoUpdateTime"`
	DeletedAt    gorm.DeletedAt   `json:"deleted_at" gorm:"column:deleted_at;index"`

	// Relationships
	User *User `json:"-" gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
}

func (ShippingAddress) TableName() string { return "shipping_addresses" }

func (sa *ShippingAddress) BeforeCreate(tx *gorm.DB) (err error) {
	if sa.ID == uuid.Nil {
		sa.ID = uuid.New()
	}

	return nil
}
