package consumers

import (
	"core-backend/internal/application/dto/requests"

	"github.com/google/uuid"
)

type CampaignCreateMessage struct {
	Data   requests.CreateCampaignRequest `json:"data"`
	UserID uuid.UUID                      `json:"user_id"`
}
