package requests

import "core-backend/internal/domain/model"

// CreateBrandRequest represents the request payload for creating a new brand.
type CreateBrandRequest struct {
	Name         string  `json:"name" validate:"required,min=2,max=255"`
	Description  *string `json:"description" validate:"omitempty,max=1000"`
	ContactEmail string  `json:"contact_email" validate:"required,email,max=255"`
	ContactPhone string  `json:"contact_phone" validate:"omitempty,e164"`
	Website      *string `json:"website" validate:"omitempty,url,max=255"`
	LogoURL      *string `json:"logo_url" validate:"omitempty,url"`
}

// UpdateBrandRequest represents the request payload for updating an existing brand.
type UpdateBrandRequest struct {
	Name         string  `json:"name" validate:"required,min=2,max=255"`
	Description  *string `json:"description" validate:"omitempty,max=1000"`
	ContactEmail string  `json:"contact_email" validate:"required,email,max=255"`
	ContactPhone string  `json:"contact_phone" validate:"omitempty,e164"`
	Website      *string `json:"website" validate:"omitempty,url,max=255"`
	LogoURL      *string `json:"logo_url" validate:"omitempty,url"`
}

// ListBrandsRequest represents the request payload for listing brands with optional filters and pagination.
type ListBrandsRequest struct {
	PaginationRequest
	Keywords *string `json:"keywords" form:"keywords" validate:"omitempty,max=255"`
	Status   *string `json:"status" form:"status" validate:"omitempty,oneof=ACTIVE INACTIVE"`
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
