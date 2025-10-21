package responses

import "core-backend/internal/domain/model"

type VariantAttributeResponse struct {
	ID          string  `json:"id"`
	Ingredient  string  `json:"ingredient"`
	Description *string `json:"description"`
}

func (v *VariantAttributeResponse) ToVariantAttributeResponse(attribute model.VariantAttribute) VariantAttributeResponse {
	return VariantAttributeResponse{
		ID:          attribute.ID.String(),
		Ingredient:  attribute.Ingredient,
		Description: attribute.Description,
	}
}
