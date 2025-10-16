package responses

import (
	"core-backend/internal/domain/model"
	"core-backend/pkg/utils"
)

// ModifiedHistoryResponse represents a ModifiedHistory model
type ModifiedHistoryResponse struct {
	ID            string `json:"id,omitempty"`
	ReferenceID   string `json:"reference_id,omitempty"`
	ReferenceType string `json:"reference_type,omitempty"`
	Operation     string `json:"operation,omitempty"`
	Description   string `json:"description,omitempty"`
	ChangedByID   string `json:"changed_by,omitempty"`
	ChangedAt     string `json:"changed_at,omitempty"`
}

// ToModifiedHistoryResponse converts a ModifiedHistory model to a ModifiedHistoryResponse
func (mhr ModifiedHistoryResponse) ToModifiedHistoryResponse(model *model.ModifiedHistory) *ModifiedHistoryResponse {
	return &ModifiedHistoryResponse{
		ID:            model.ID.String(),
		ReferenceID:   model.ReferenceID.String(),
		ReferenceType: model.ReferenceType.String(),
		Operation:     model.Operation.String(),
		Description:   model.Description,
		ChangedByID:   model.ChangedByID.String(),
		ChangedAt:     utils.FormatLocalTime(model.ChangedAt, ""),
	}
}

func (mhr ModifiedHistoryResponse) ToModifiedHistoryResponseList(models []model.ModifiedHistory) []ModifiedHistoryResponse {
	responses := make([]ModifiedHistoryResponse, len(models))
	for _, model := range models {
		responses = append(responses, *mhr.ToModifiedHistoryResponse(&model))
	}
	return responses
}
