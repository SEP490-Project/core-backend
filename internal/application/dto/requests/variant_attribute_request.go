package requests

import (
	"core-backend/internal/domain/model"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"time"
)

type CreateVariantAttributeRequest struct {
	Ingredient  string         `json:"ingredient" gorm:"column:ingredient;not null"`
	Description *string        `json:"description" gorm:"column:description"`
	CreatedAt   time.Time      `json:"created_at" gorm:"column:created_at;autoCreateTime"`
	UpdatedAt   time.Time      `json:"updated_at" gorm:"column:updated_at;autoUpdateTime"`
	DeletedAt   gorm.DeletedAt `json:"deleted_at" gorm:"column:deleted_at;index"`
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
