package responses

import (
	"core-backend/internal/domain/model"
	"core-backend/pkg/utils"

	"github.com/google/uuid"
)

// region: ============== Contract Payment Response ==============

type ContractPaymenntResponse struct {
	ID                    string  `json:"id" example:"b3e1f9d2-8c4e-4f5a-9f1e-2d3c4b5a6e7f"`
	ContractID            string  `json:"contract_id" example:"a1b2c3d4-e5f6-7a8b-9c0d-e1f2a3b4c5d6"`
	ContractTitle         string  `json:"contract_title" example:"Website Development Contract"`
	ContractNumber        string  `json:"contract_number" example:"WD-2024-001"`
	BrandID               string  `json:"brand_id" example:"d4c3b2a1-0f9e-8d7c-6b5a-4c3b2a1f0e9d"`
	BrandName             string  `json:"brand_name" example:"Tech Solutions Inc."`
	InstallmentPercentage float64 `json:"installment_percentage" example:"50.0"`
	Amount                float64 `json:"amount" example:"5000.00"`
	Status                string  `json:"status" example:"PENDING"`
	DueDate               string  `json:"due_date" example:"2024-07-15T00:00:00Z"`
	PaymentMethod         string  `json:"payment_method" example:"BANK_TRANSFER"`
	Note                  *string `json:"note" example:"First installment payment"`
	CreatedAt             string  `json:"created_at" example:"2006-01-02T15:04:05Z07:00"`
	UpdatedAt             string  `json:"updated_at" example:"2006-01-02T15:04:05Z07:00"`
}

// ToResponse converts a ContractPayment model to a ContractPaymentResponse
func (ContractPaymenntResponse) ToResponse(model *model.ContractPayment) *ContractPaymenntResponse {
	if model == nil {
		return nil
	}

	response := &ContractPaymenntResponse{
		ID:                    model.ID.String(),
		ContractID:            model.ContractID.String(),
		InstallmentPercentage: model.InstallmentPercentage,
		Amount:                model.Amount,
		Status:                model.Status.String(),
		DueDate:               utils.FormatLocalTime(&model.DueDate, utils.DateFormat),
		PaymentMethod:         model.PaymentMethod.String(),
		Note:                  model.Note,
		CreatedAt:             utils.FormatLocalTime(&model.CreatedAt, utils.TimezoneFormat),
		UpdatedAt:             utils.FormatLocalTime(&model.UpdatedAt, utils.TimezoneFormat),
	}
	if model.ContractID != uuid.Nil && model.Contract != nil {
		response.ContractID = model.Contract.ID.String()
		response.ContractTitle = *model.Contract.Title
		response.ContractNumber = *model.Contract.ContractNumber
		if model.Contract.Brand != nil {
			response.BrandID = model.Contract.Brand.ID.String()
			response.BrandName = model.Contract.Brand.Name
		}
	}

	return response
}

// ToResponseList converts a list of ContractPayment models to a list of ContractPaymentResponses
func (ContractPaymenntResponse) ToResponseList(model []model.ContractPayment) []ContractPaymenntResponse {
	if len(model) == 0 {
		return []ContractPaymenntResponse{}
	}

	responses := make([]ContractPaymenntResponse, len(model))
	for i, v := range model {
		responses[i] = *ContractPaymenntResponse{}.ToResponse(&v)
	}
	return responses
}

// endregion

type ContractPaymentDetailResponse struct {
	ContractPaymenntResponse
}

type ContractPaymentPaginationResponse PaginationResponse[ContractPaymenntResponse]
