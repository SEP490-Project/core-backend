package requests

import (
	"core-backend/internal/domain/model"
	"time"

	"github.com/google/uuid"
)

type CreateVariantImagesRequest struct {
	VariantID uuid.UUID `json:"variant_id" form:"variant_id" validate:"required,uuid" example:"550e8400-e29b-41d4-a716-446655440000"`
	ImageURL  string    `json:"image_url" form:"image_url" validate:"required"`
	AltText   *string   `json:"alt_text" form:"alt_text" validate:"required"`
	IsPrimary bool      `json:"is_primary" form:"is_primary" gorm:"column:is_primary;type:boolean;not null"`
	CreatedAt time.Time `json:"created_at" form:"created_at"  validate:"omitempty,datetime=2006-01-02"`
}

func (v *CreateVariantImagesRequest) ToModel() *model.VariantImage {
	now := time.Now().UTC()
	return &model.VariantImage{
		VariantID: v.VariantID,
		ImageURL:  v.ImageURL,
		AltText:   v.AltText,
		IsPrimary: v.IsPrimary,
		CreatedAt: now,
		UpdatedAt: now,
	}
}
