package requests

import (
	"core-backend/internal/domain/model"
	"time"

	"github.com/google/uuid"
)

type CreateProductStoryRequest struct {
	VariantID uuid.UUID `json:"variant_id" form:"variant_id" gorm:"type:uuid;column:variant_id;not null" example:"550e8400-e29b-41d4-a716-446655440000"`
	Content   []byte    `json:"content" form:"content" validate:"required,max=5000"`
}

func (ps *CreateProductStoryRequest) ToModel() *model.ProductStory {
	now := time.Now().UTC()

	return &model.ProductStory{
		VariantID: ps.VariantID,
		Content:   ps.Content,
		CreatedAt: now,
		UpdatedAt: now,
	}
}
