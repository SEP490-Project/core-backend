package responses

import "core-backend/internal/domain/model"

// BrandResponse represents the response structure for a brand.
type BrandResponse struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	Description  *string `json:"description"`
	ContactEmail string  `json:"contact_email"`
	ContactPhone string  `json:"contact_phone"`
	Website      *string `json:"website"`
	Status       string  `json:"status"`
	LogoURL      *string `json:"logo_url"`
	CreatedAt    string  `json:"created_at"`
}

// ToBrandResponse converts a Brand model to a BrandResponse DTO.
func (br BrandResponse) ToBrandResponse(model *model.Brand) *BrandResponse {
	br.ID = model.ID.String()
	br.Name = model.Name
	br.Description = model.Description
	br.ContactEmail = model.ContactEmail
	br.ContactPhone = model.ContactPhone
	br.Website = model.Website
	br.Status = string(model.Status)
	br.LogoURL = model.LogoURL
	br.CreatedAt = model.CreatedAt.Format(TimeFormat)

	return &br
}

// BrandPaginationResponse represents a paginated response for brands.
// Only used for Swaggo swagger docs generation
type BrandPaginationResponse PaginationResponse[BrandResponse]
