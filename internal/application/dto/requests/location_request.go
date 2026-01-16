package requests

import (
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"

	"github.com/google/uuid"
)

type InputAddressRequest struct {
	AddressType  enum.AddressType `json:"type" validate:"required,oneof=BILLING SHIPPING" example:"SHIPPING"`
	FullName     string           `json:"full_name" validate:"required"`
	PhoneNumber  string           `json:"phone_number" validate:"required"`
	Email        string           `json:"email"`
	Street       string           `json:"street" validate:"required"`
	AddressLine2 string           `json:"address_line_2" example:"Apartment, suite, unit, building, floor, etc."`
	City         string           `json:"city" validate:"required"`
	PostalCode   *string          `json:"postal_code"`
	Country      *string          `json:"country"`
	IsDefault    bool             `json:"is_default"`

	//from GHN
	GhnProvinceID int    `json:"ghn_province_id" example:"202"`
	GhnDistrictID int    `json:"ghn_district_id" example:"3176"`
	GhnWardCode   string `json:"ghn_ward_code" example:"21015"`
}

func (iar *InputAddressRequest) ToModel(userID uuid.UUID, ward model.Ward, district model.District, province model.Province) *model.ShippingAddress {
	if iar == nil {
		return nil
	}
	return &model.ShippingAddress{
		Type:          iar.AddressType,
		FullName:      iar.FullName,
		PhoneNumber:   iar.PhoneNumber,
		Email:         iar.Email,
		Street:        iar.Street,
		AddressLine2:  iar.AddressLine2,
		City:          iar.City,
		PostalCode:    iar.PostalCode,
		Country:       iar.Country,
		IsDefault:     iar.IsDefault,
		GhnProvinceID: iar.GhnProvinceID,
		GhnDistrictID: iar.GhnDistrictID,
		GhnWardCode:   iar.GhnWardCode,
		ProvinceName:  province.Name,
		DistrictName:  district.Name,
		WardName:      ward.Name,
		//relationship
		UserID: userID,
	}
}

// UpdateAddressRequest represents the request payload for updating a shipping address
type UpdateAddressRequest struct {
	AddressType  enum.AddressType `json:"type" validate:"omitempty,oneof=BILLING SHIPPING" example:"SHIPPING"`
	FullName     string           `json:"full_name" validate:"omitempty"`
	PhoneNumber  string           `json:"phone_number" validate:"omitempty"`
	Email        string           `json:"email"`
	Street       string           `json:"street" validate:"omitempty"`
	AddressLine2 string           `json:"address_line_2" example:"Apartment, suite, unit, building, floor, etc."`
	City         string           `json:"city" validate:"omitempty"`
	PostalCode   *string          `json:"postal_code"`
	Country      *string          `json:"country"`
	IsDefault    *bool            `json:"is_default"`

	//from GHN
	GhnProvinceID *int    `json:"ghn_province_id" example:"202"`
	GhnDistrictID *int    `json:"ghn_district_id" example:"3176"`
	GhnWardCode   *string `json:"ghn_ward_code" example:"21015"`
}
