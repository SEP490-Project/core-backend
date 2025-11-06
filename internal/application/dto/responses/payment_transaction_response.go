package responses

import (
	"core-backend/internal/domain/model"
	"core-backend/pkg/utils"

	"github.com/google/uuid"
)

type PaymentTransactionResponse struct {
	ID              uuid.UUID `json:"id" example:"b3e1f9d2-8c4e-4f5a-9f1e-2d3c4b5a6e7f"`
	ReferenceID     string    `json:"reference_id" example:"a1b2c3d4-e5f6-7g8h-9i0j-k1l2m3n4o5p6"`
	ReferenceType   string    `json:"reference_type" example:"ORDER"`
	Amount          string    `json:"amount" example:"99.99"`
	Method          string    `json:"method" example:"CREDIT_CARD"`
	Status          string    `json:"status" example:"COMPLETED"`
	TransactionDate string    `json:"transaction_date" example:"2024-01-01T12:00:00Z"`
	GatewayRef      string    `json:"gateway_ref,omitempty" example:"PAYOS123456789"`
	GatewayID       string    `json:"gateway_id,omitempty" example:"GATEWAY987654321"`
	UpdatedAt       string    `json:"updated_at" example:"2024-01-02T12:00:00Z"`
}

func (PaymentTransactionResponse) ToResponse(source *model.PaymentTransaction) *PaymentTransactionResponse {
	if source == nil {
		return nil
	}

	return &PaymentTransactionResponse{
		ID:              source.ID,
		ReferenceID:     source.ReferenceID.String(),
		ReferenceType:   source.ReferenceType.String(),
		Amount:          utils.ToString(source.Amount),
		Method:          source.Method,
		Status:          string(source.Status),
		TransactionDate: utils.FormatLocalTime(&source.TransactionDate, utils.TimeFormat),
		GatewayRef:      source.GatewayRef,
		GatewayID:       source.GatewayID,
		UpdatedAt:       utils.FormatLocalTime(&source.UpdatedAt, utils.TimeFormat),
	}
}

func (PaymentTransactionResponse) ToResponseList(source []model.PaymentTransaction) []PaymentTransactionResponse {
	if len(source) == 0 {
		return []PaymentTransactionResponse{}
	}

	responses := make([]PaymentTransactionResponse, len(source))
	for i, v := range source {
		responses[i] = *PaymentTransactionResponse{}.ToResponse(&v)
	}
	return responses
}
