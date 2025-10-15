package responses

import (
	"core-backend/internal/domain/model"
	"time"
)

type VariantImageResponse struct {
	ID        string    `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	VariantID string    `json:"variant_id" example:"550e8400-e29b-41d4-a716-446655440001"`
	ImageURL  string    `json:"image_url" example:"https://example.com/image.jpg"`
	AltText   *string   `json:"alt_text" example:"Sample image"`
	IsPrimary bool      `json:"is_primary" example:false`
	CreatedAt time.Time `json:"created_at" example:"2023-10-01T00:00:00Z"`
	UpdatedAt time.Time `json:"updated_at" example:"2023-10-01T00:00:00Z"`
}

func (v VariantImageResponse) ToVariantImageResponse(m *model.VariantImage) *VariantImageResponse {
	if m == nil {
		return nil
	}
	return &VariantImageResponse{
		ID:        m.ID.String(),
		VariantID: m.VariantID.String(),
		ImageURL:  m.ImageURL,
		AltText:   m.AltText,
		IsPrimary: m.IsPrimary,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.UpdatedAt,
	}
}
