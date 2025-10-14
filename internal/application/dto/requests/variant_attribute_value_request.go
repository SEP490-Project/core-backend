package requests

import (
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
	"github.com/google/uuid"
	"time"
)

type CreateVariantAttributeValueRequest struct {
	Value       float64            `json:"value" gorm:"column:value;not null" example:"10.5"`
	Unit        enum.AttributeUnit `json:"unit" gorm:"column:unit;not null;check:unit in ('%', 'MG', 'G', 'ML', 'L', 'IU', 'PPM', 'NONE')" example:"MG"`
	AttributeID uuid.UUID          `json:"attribute_id" gorm:"type:uuid;column:attribute_id;not null" example:"550e8400-e29b-41d4-a716-446655440001"`
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
