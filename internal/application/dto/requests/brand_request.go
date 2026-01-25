package requests

import "core-backend/internal/domain/model"

// CreateBrandRequest represents the request payload for creating a new brand.
type CreateBrandRequest struct {
	Name         string  `json:"name" validate:"required,min=2,max=255" example:"Acme Corp"`
	Description  *string `json:"description" validate:"omitempty,max=1000" example:"A leading manufacturer of quality products."`
	ContactEmail string  `json:"contact_email" validate:"required,email,max=255" example:"acme@example.com"`
	ContactPhone string  `json:"contact_phone" validate:"omitempty,e164" example:"+1234567890"`
	Address      *string `json:"address,omitempty" validate:"omitempty,max=500" example:"123 Main St, Anytown, USA"`
	Website      *string `json:"website,omitempty" validate:"omitempty,url,max=255" example:"https://www.acme.com"`
	LogoURL      *string `json:"logo_url,omitempty" validate:"omitempty,url" example:"https://www.acme.com/logo.png"`
}

// CreateBrandWithUserRequest represents the request payload for creating a new brand with user information.
type CreateBrandWithUserRequest struct {
	CreateBrandRequest
	TaxNumber               *string `json:"tax_number" validate:"omitempty,max=100" example:"123456789"`
	RepresentativeName      *string `json:"representative_name" validate:"omitempty,max=255" example:"John Doe"`
	RepresentativeRole      *string `json:"representative_role" validate:"omitempty,max=255" example:"Manager"`
	RepresentativeEmail     *string `json:"representative_email" validate:"omitempty,email,max=255" example:"john.doe@example.com"`
	RepresentativePhone     *string `json:"representative_phone" validate:"omitempty,e164" example:"+1234567890"`
	RepresentativeCitizenID *string `json:"representative_citizen_id" validate:"omitempty,max=50" example:"A123456789"`
}

// UpdateBrandRequest represents the request payload for updating an existing brand.
type UpdateBrandRequest struct {
	Name         *string `json:"name,omitempty" validate:"omitempty,min=2,max=255" example:"Acme Corp"`
	Description  *string `json:"description,omitempty" validate:"omitempty,max=1000" example:"A leading manufacturer of quality products."`
	ContactEmail *string `json:"contact_email,omitempty" validate:"omitempty,email,max=255" example:"acme@example.com"`
	ContactPhone *string `json:"contact_phone,omitempty" validate:"omitempty,e164" example:"+1234567890"`
	Address      *string `json:"address,omitempty" validate:"omitempty,max=500" example:"123 Main St, Anytown, USA"`
	Website      *string `json:"website,omitempty" validate:"omitempty,url,max=255" example:"https://www.acme.com"`
	LogoURL      *string `json:"logo_url,omitempty" validate:"omitempty,url" example:"https://www.acme.com/logo.png"`
}

// ListBrandsRequest represents the request payload for listing brands with optional filters and pagination.
type ListBrandsRequest struct {
	PaginationRequest
	Keywords *string `json:"keywords" form:"keywords" validate:"omitempty,max=255" example:"Acme"`
	Status   *string `json:"status" form:"status" validate:"omitempty,oneof=ACTIVE INACTIVE" example:"ACTIVE"`
}

type ListProductsByBrandRequest struct {
	PaginationRequest
	Keywords *string `json:"keywords" form:"keywords" validate:"omitempty,max=255" example:"Acme"`
}

func (ubr UpdateBrandRequest) ToExistingBrand(brand *model.Brand) *model.Brand {
	if ubr.Name != nil {
		brand.Name = *ubr.Name
	}
	if ubr.Description != nil {
		brand.Description = ubr.Description
	}
	if ubr.ContactEmail != nil {
		brand.ContactEmail = *ubr.ContactEmail
	}
	if ubr.ContactPhone != nil {
		brand.ContactPhone = *ubr.ContactPhone
	}
	if ubr.Address != nil {
		brand.Address = ubr.Address
	}
	if ubr.Website != nil {
		brand.Website = ubr.Website
	}
	if ubr.LogoURL != nil {
		brand.LogoURL = ubr.LogoURL
	}
	return brand
}
