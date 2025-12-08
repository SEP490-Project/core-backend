package responses

import (
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
	"core-backend/pkg/utils"
)

type BrandInfoResponse struct {
	ID      string  `json:"id"`
	Name    string  `json:"name"`
	LogoURL *string `json:"logo_url"`
	Status  string  `json:"status"`
}

// BrandResponse represents the response structure for a brand.
type BrandResponse struct {
	ID                      string  `json:"id"`
	Name                    string  `json:"name"`
	Description             *string `json:"description"`
	ContactEmail            string  `json:"contact_email"`
	ContactPhone            string  `json:"contact_phone"`
	Address                 *string `json:"address"`
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
func (br BrandResponse) ToBrandResponse(brandModel *model.Brand) *BrandResponse {
	br.ID = brandModel.ID.String()
	br.Name = brandModel.Name
	br.Description = brandModel.Description
	br.ContactEmail = brandModel.ContactEmail
	br.ContactPhone = brandModel.ContactPhone
	br.Address = brandModel.Address
	br.Website = brandModel.Website
	br.Status = string(brandModel.Status)
	br.LogoURL = brandModel.LogoURL
	br.CreatedAt = utils.FormatLocalTime(&brandModel.CreatedAt, "")

	if brandModel.Contracts != nil {
		br.NumberOfContracts = len(brandModel.Contracts)
		activeCount := len(utils.FilterSlice(brandModel.Contracts, func(c model.Contract) bool { return c.Status == enum.ContractStatusActive }))
		br.NumberOfActiveContracts = activeCount
	}

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
