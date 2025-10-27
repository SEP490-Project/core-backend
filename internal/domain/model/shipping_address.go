package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"core-backend/internal/domain/enum"
)

// ShippingAddress maps to the shipping_addresses table
type ShippingAddress struct {
	ID           uuid.UUID        `json:"id" gorm:"type:uuid;column:id;primaryKey;default:gen_random_uuid()"`
	UserID       uuid.UUID        `json:"user_id" gorm:"type:uuid;column:user_id;not null"`
	User         *User            `json:"user,omitempty" gorm:"foreignKey:UserID;references:ID"`
	Type         enum.AddressType `json:"type" gorm:"column:type;not null;check:type in ('BILLING','SHIPPING')"`
	FullName     string           `json:"full_name" gorm:"column:full_name;not null"`
	PhoneNumber  *string          `json:"phone_number,omitempty" gorm:"column:phone_number"`
	Email        *string          `json:"email,omitempty" gorm:"column:email"`
	Street       string           `json:"street" gorm:"column:street;not null"`
	AddressLine2 *string          `json:"address_line_2,omitempty" gorm:"column:address_line2"`
	City         string           `json:"city" gorm:"column:city;not null"`
	PostalCode   string           `json:"postal_code" gorm:"column:postal_code;not null"`
	Country      *string          `json:"country,omitempty" gorm:"column:country"`
	IsDefault    bool             `json:"is_default" gorm:"column:is_default;not null;default:false"`

	// GHN / courier related fields
	GhnProvinceID *int    `json:"ghn_province_id,omitempty" gorm:"column:ghn_province_id"`
	GhnDistrictID *int    `json:"ghn_district_id,omitempty" gorm:"column:ghn_district_id"`
	GhnWardCode   *string `json:"ghn_ward_code,omitempty" gorm:"column:ghn_ward_code"`
	ProvinceName  *string `json:"province_name,omitempty" gorm:"column:province_name"`
	DistrictName  *string `json:"district_name,omitempty" gorm:"column:district_name"`
	WardName      *string `json:"ward_name,omitempty" gorm:"column:ward_name"`

	CreatedAt time.Time      `json:"created_at" gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time      `json:"updated_at" gorm:"column:updated_at;autoUpdateTime"`
	DeletedAt gorm.DeletedAt `json:"deleted_at" gorm:"column:deleted_at;index"`
}

func (ShippingAddress) TableName() string { return "shipping_addresses" }

func (sa *ShippingAddress) BeforeCreate(_ *gorm.DB) (err error) {
	if sa.ID == uuid.Nil {
		sa.ID = uuid.New()
	}

	return nil
}
