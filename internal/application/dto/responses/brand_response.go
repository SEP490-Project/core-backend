package responses

import (
	"core-backend/internal/domain/model"
	"core-backend/pkg/utils"
)

// BrandResponse represents the response structure for a brand.
type BrandResponse struct {
	ID                      string  `json:"id"`
	Name                    string  `json:"name"`
	Description             *string `json:"description"`
	ContactEmail            string  `json:"contact_email"`
	ContactPhone            string  `json:"contact_phone"`
	Website                 *string `json:"website"`
	Status                  string  `json:"status"`
	NumberOfContracts       int     `json:"number_of_contracts,omitempty"`
	NumberOfActiveContracts int     `json:"number_of_active_contracts,omitempty"`
	LogoURL                 *string `json:"logo_url"`
	CreatedAt               string  `json:"created_at"`
}

type BrandDetailResponse struct {
	BrandResponse
	TaxNumber               *string `json:"tax_number" example:"123456789"`
	RepresentativeName      *string `json:"representative_name" example:"John Doe"`
	RepresentativeRole      *string `json:"representative_role" example:"Manager"`
	RepresentativeEmail     *string `json:"representative_email" example:"john.doe@example.com"`
	RepresentativePhone     *string `json:"representative_phone" example:"+1234567890"`
	RepresentativeCitizenID *string `json:"representative_citizen_id" example:"A123456789"`
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
	br.CreatedAt = utils.FormatLocalTime(&model.CreatedAt, "")

	return &br
}

func (bdr BrandDetailResponse) ToBrandDetailResponse(model *model.Brand) *BrandDetailResponse {
	return &BrandDetailResponse{
		BrandResponse:           *bdr.ToBrandResponse(model),
		TaxNumber:               model.TaxNumber,
		RepresentativeName:      model.RepresentativeName,
		RepresentativeRole:      model.RepresentativeRole,
		RepresentativeEmail:     model.RepresentativeEmail,
		RepresentativePhone:     model.RepresentativePhone,
		RepresentativeCitizenID: model.RepresentativeCitizenID,
	}
}

// BrandPaginationResponse represents a paginated response for brands.
// Only used for Swaggo swagger docs generation
type BrandPaginationResponse PaginationResponse[BrandResponse]
