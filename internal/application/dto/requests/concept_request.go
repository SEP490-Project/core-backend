package requests

import (
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
	"time"
)

type ConceptRequest struct {
	Name           string  `json:"name" validate:"required,min=1,max=255" example:"Concept Name"`
	Description    *string `json:"description" validate:"omitempty,max=1000" example:"Concept description"`
	Status         string  `json:"status" validate:"required,oneof=DRAFT ACTIVE INACTIVE" example:"DRAFT"`
	StartDate      *string `json:"start_date" validate:"omitempty,datetime=2006-01-02T15:04:05Z07:00" example:"2006-01-02T15:04:05Z07:00"`
	EndDate        *string `json:"end_date" validate:"omitempty,datetime=2006-01-02T15:04:05Z07:00" example:"2006-01-02T15:04:05Z07:00"`
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
		Status:         enum.ConceptStatus(d.Status),
		StartDate:      parseNullableTime(d.StartDate),
		EndDate:        parseNullableTime(d.EndDate),
		BannerURL:      d.BannerURL,
		VideoThumbnail: d.VideoThumbnail,
	}
}

func parseNullableTime(date *string) *time.Time {
	if date == nil {
		return nil
	}
	parsedTime, err := time.Parse(time.RFC3339, *date)
	if err != nil {
		return nil
	}
	return &parsedTime
}
