package responses

import (
	"core-backend/internal/application/dto/dtos"
	"core-backend/pkg/utils"
)

type MilestoneResponse struct {
	ID                   string  `json:"id" example:"550e8400-e29b-41d4-a716-446655440000"`
	Description          *string `json:"description" gorm:"type:text;column:description"`
	DueDate              string  `json:"due_date" gorm:"type:timestamp;column:due_date;not null"`
	CompletedAt          string  `json:"completed_at" gorm:"type:timestamp;column:completed_at"`
	CompletionPercentage float64 `json:"completion_percentage" gorm:"column:completion_percentage"`
	Status               string  `json:"status" gorm:"column:status;not null;check:status in ('NOT_STARTED', 'ON_GOING', 'CANCELLED', 'COMPLETED')"`
	BehindSchedule       bool    `json:"behind_schedule" gorm:"column:behind_schedule"`
}

func (MilestoneResponse) ToResponse(dto *dtos.MilestoneDTO) *MilestoneResponse {
	if dto == nil {
		return nil
	}

	response := &MilestoneResponse{
		ID:                   dto.ID.String(),
		Description:          dto.Description,
		DueDate:              dto.DueDate,
		CompletedAt:          utils.FormatLocalTime(dto.CompletedAt, utils.TimeFormat),
		CompletionPercentage: dto.CompletionPercentage,
		Status:               dto.Status.String(),
		BehindSchedule:       dto.BehindSchedule,
	}

	return response
}
