package requests

import (
	"core-backend/internal/domain/model"
)

type ConceptRequest struct {
	Name           string  `json:"name" validate:"required,min=1" example:"Concept Name"`
	Description    *string `json:"description" validate:"omitempty" example:"Concept description"`
	BannerURL      *string `json:"banner_url" validate:"omitempty,url" example:"https://example.com/banner.jpg"`
	VideoThumbnail *string `json:"video_thumbnail" validate:"omitempty,url" example:"https://example.com/thumbnail.jpg"`
}

func (d *ConceptRequest) ToModel() *model.Concept {
	if d == nil {
		return nil
	}
	return &model.Concept{
		Name:           d.Name,
		Description:    d.Description,
		BannerURL:      d.BannerURL,
		VideoThumbnail: d.VideoThumbnail,
	}
}

type UpdateConceptRequest struct {
	Name           *string `json:"name" validate:"omitempty" example:"Concept Name"`
	Description    *string `json:"description" validate:"omitempty" example:"Concept description"`
	BannerURL      *string `json:"banner_url" validate:"omitempty,url" example:"https://example.com/banner.jpg"`
	VideoThumbnail *string `json:"video_thumbnail" validate:"omitempty,url" example:"https://example.com/thumbnail.jpg"`
}
