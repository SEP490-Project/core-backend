package requests

import "core-backend/internal/domain/enum"

type InputAddressRequest struct {
	AddressType  enum.AddressType `json:"type" validate:"required,oneof=BILLING SHIPPING" example:"SHIPPING"`
	FullName     string           `json:"full_name" validate:"required"`
	PhoneNumber  *string          `json:"phone_number" validate:"required"`
	Email        *string          `json:"email"`
	Street       string           `json:"street" validate:"required"`
	AddressLine2 *string          `json:"address_line_2" example:"Apartment, suite, unit, building, floor, etc."`
	City         string           `json:"city" validate:"required"`
	PostalCode   string           `json:"postal_code"`
	Country      *string          `json:"country"`
	IsDefault    *bool            `json:"is_default"`

	//from GHN
	GhnProvinceID *int    `json:"ghn_province_id,omitempty" gorm:"column:ghn_province_id"`
	GhnDistrictID *int    `json:"ghn_district_id,omitempty" gorm:"column:ghn_district_id"`
	GhnWardCode   *string `json:"ghn_ward_code,omitempty" gorm:"column:ghn_ward_code"`
}

func (iar *InputAddressRequest) ToModel() *InputAddressRequest {
	if iar == nil {
		return nil
	}
	return &InputAddressRequest{
		AddressType:   iar.AddressType,
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
	}
}
