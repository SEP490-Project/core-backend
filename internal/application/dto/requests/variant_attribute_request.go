package requests

import (
	"core-backend/internal/domain/model"
	"time"

	"github.com/google/uuid"
)

type CreateVariantAttributeRequest struct {
	Ingredient  string  `json:"ingredient" validate:"required"`
	Description *string `json:"description" validate:"omitempty"`
}

func (va *CreateVariantAttributeRequest) ToCreationalModel(createdByID uuid.UUID) *model.VariantAttribute {
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
