package requests

import "core-backend/internal/domain/model"

// CreateBrandRequest represents the request payload for creating a new brand.
type CreateBrandRequest struct {
	Name         string  `json:"name" validate:"required,min=2,max=255" example:"Acme Corp"`
	Description  *string `json:"description" validate:"omitempty,max=1000" example:"A leading manufacturer of quality products."`
	ContactEmail string  `json:"contact_email" validate:"required,email,max=255" example:"acme@example.com"`
	ContactPhone string  `json:"contact_phone" validate:"omitempty,e164" example:"+1234567890"`
	Website      *string `json:"website" validate:"omitempty,url,max=255" example:"https://www.acme.com"`
	LogoURL      *string `json:"logo_url" validate:"omitempty,url" example:"https://www.acme.com/logo.png"`
}

// UpdateBrandRequest represents the request payload for updating an existing brand.
type UpdateBrandRequest struct {
	Name         string  `json:"name" validate:"required,min=2,max=255" example:"Acme Corp"`
	Description  *string `json:"description" validate:"omitempty,max=1000" example:"A leading manufacturer of quality products."`
	ContactEmail string  `json:"contact_email" validate:"required,email,max=255" example:"acme@example.com"`
	ContactPhone string  `json:"contact_phone" validate:"omitempty,e164" example:"+1234567890"`
	Website      *string `json:"website" validate:"omitempty,url,max=255" example:"https://www.acme.com"`
	LogoURL      *string `json:"logo_url" validate:"omitempty,url" example:"https://www.acme.com/logo.png"`
}

// ListBrandsRequest represents the request payload for listing brands with optional filters and pagination.
type ListBrandsRequest struct {
	PaginationRequest
	Keywords *string `json:"keywords" form:"keywords" validate:"omitempty,max=255" example:"Acme"`
	Status   *string `json:"status" form:"status" validate:"omitempty,oneof=ACTIVE INACTIVE" example:"ACTIVE"`
}

func (ubr UpdateBrandRequest) ToExistingBrand(brand *model.Brand) *model.Brand {
	if ubr.Name != "" {
		brand.Name = ubr.Name
	}
	if ubr.Description != nil {
		brand.Description = ubr.Description
	}
	if ubr.ContactEmail != "" {
		brand.ContactEmail = ubr.ContactEmail
	}
	if ubr.ContactPhone != "" {
		brand.ContactPhone = ubr.ContactPhone
	}
	if ubr.Website != nil {
		brand.Website = ubr.Website
	}
	if ubr.LogoURL != nil {
		brand.LogoURL = ubr.LogoURL
	}
	return brand
}
