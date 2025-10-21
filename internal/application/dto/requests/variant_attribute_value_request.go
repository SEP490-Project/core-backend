package requests

import (
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
	"time"

	"github.com/google/uuid"
)

type CreateVariantAttributeValueRequest struct {
	Value       float64            `json:"value" validate:"required,min=0.1" gorm:"column:value;not null"`
	Unit        enum.AttributeUnit `json:"unit" validate:"required,oneof=% MG G ML L IU PPM NONE" gorm:"column:unit;not null"`
	AttributeID uuid.UUID          `json:"attribute_id" validate:"required" gorm:"type:uuid;column:attribute_id;not null"`
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
