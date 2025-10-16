package consumers

import (
	"core-backend/internal/application/dto/requests"

	"github.com/google/uuid"
)

// ContractCreateMessage represents the message structure for contract creation
type ContractCreateMessage struct {
	Contract requests.CreateContractRequest `json:"contract"`
	UserID   uuid.UUID                      `json:"user_id"`
}
