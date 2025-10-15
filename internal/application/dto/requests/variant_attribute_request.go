package requests

import (
	"core-backend/internal/domain/model"
	"github.com/google/uuid"
	"time"
)

type CreateVariantAttributeRequest struct {
	Ingredient  string  `json:"ingredient" validate:"required"`
	Description *string `json:"description" validate:"omitempty"`
}

func (va *CreateVariantAttributeRequest) ToModel(createdByID uuid.UUID) *model.VariantAttribute {
	now := time.Now().UTC()

	return &model.VariantAttribute{
		Ingredient:  va.Ingredient,
		Description: va.Description,
		CreatedAt:   now,
		UpdatedAt:   now,
		CreatedByID: createdByID,
		UpdatedByID: nil,
	}

}
