package requests

import (
	"core-backend/internal/domain/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"time"
)

type CreateVariantImagesRequest struct {
	VariantID uuid.UUID      `json:"variant_id" form:"variant_id" gorm:"type:uuid;column:variant_id;not null"`
	ImageURL  string         `json:"image_url" form:"image_url" gorm:"type:text;column:variant_id;not null"`
	AltText   *string        `json:"alt_text" form:"alt_text" gorm:"type:text;column:variant_id;not null"`
	IsPrimary bool           `json:"is_primary" form:"is_primary" gorm:"type:boolean;column:variant_id;not null"`
	CreatedAt time.Time      `json:"created_at" form:"created_at" gorm:"type:timestamp;column:variant_id;not null"`
	DeletedAt gorm.DeletedAt `json:"deleted_at" form:"deleted_at" gorm:"type:timestamp;column:variant_id;not null"`
}

func (v *CreateVariantImagesRequest) ToModel() *model.VariantImage {
	now := time.Now().UTC()
	return &model.VariantImage{
		VariantID: v.VariantID,
		ImageURL:  v.ImageURL,
		AltText:   v.AltText,
		CreatedAt: now,
		UpdatedAt: now,
	}
}
