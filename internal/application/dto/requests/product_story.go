package requests

import (
	"core-backend/internal/domain/model"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

type CreateProductStoryRequest struct {
	VariantID uuid.UUID      `json:"variant_id" form:"variant_id" gorm:"type:uuid;column:variant_id;not null" example:"550e8400-e29b-41d4-a716-446655440000"`
	Content   datatypes.JSON `json:"content" form:"content" validate:"required" example:"{\"description\":\"This is a sample story\"}" swaggerignore:"true"`
}

func (ps *CreateProductStoryRequest) ToModel() *model.ProductStory {
	now := time.Now().UTC()

	// Ensure Content is valid JSON; if not, store as empty JSON object
	var raw datatypes.JSON
	if len(ps.Content) == 0 {
		raw = datatypes.JSON([]byte("{}"))
	} else {
		raw = datatypes.JSON(ps.Content)
	}

	return &model.ProductStory{
		VariantID: ps.VariantID,
		Content:   raw,
		CreatedAt: now,
		UpdatedAt: now,
	}
}
