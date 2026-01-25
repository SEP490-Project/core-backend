package requests

import (
	"core-backend/internal/domain/model"
	"time"

	"github.com/google/uuid"
)

type CreateVariantAttributeValueRequest struct {
	Value       float64   `json:"value" validate:"required,min=0.1" example:"10.5"`
	Unit        string    `json:"unit" validate:"required" example:"MG"`
	AttributeID uuid.UUID `json:"attribute_id" validate:"required" example:"550e8400-e29b-41d4-a716-446655440001"`
}

func (v *CreateVariantAttributeValueRequest) ToModel() *model.VariantAttributeValue {
	if v == nil {
		return nil
	}

	now := time.Now().UTC()

	return &model.VariantAttributeValue{
		AttributeID: v.AttributeID,
		Value:       v.Value,
		Unit:        v.Unit,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}
