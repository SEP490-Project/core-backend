package responses

import (
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
	"core-backend/pkg/utils"

	"github.com/google/uuid"
)

type ContentCommentResponse struct {
	ID             uuid.UUID                   `json:"id"`
	Comment        string                      `json:"comment"`
	Reactions      map[enum.ReactionType]int64 `json:"reactions"`
	CreatedAt      string                      `json:"created_at"`
	IsEdit         bool                        `json:"is_edit"`
	IsCensored     *bool                       `json:"is_censored,omitempty"`
	CensoredReason *string                     `json:"censor_reason,omitempty"`
	UserID         uuid.UUID                   `json:"user_id"`
	Username       string                      `json:"username"`
	AvatarURL      *string                     `json:"avatar_url,omitempty"`
	UserReaction   *enum.ReactionType          `json:"user_reaction,omitempty"`
}

// ToResponse converts a ContentComment model to a ContentCommentResponse DTO
func (ContentCommentResponse) ToResponse(comment *model.ContentComment, userID *uuid.UUID) *ContentCommentResponse {
	if comment == nil {
		return &ContentCommentResponse{}
	}

	resp := &ContentCommentResponse{
		ID:             comment.ID,
		Comment:        comment.Comment,
		Reactions:      make(map[enum.ReactionType]int64),
		CreatedAt:      utils.FormatLocalTime(comment.CreatedAt, ""),
		IsEdit:         comment.UpdatedAt != nil && !comment.UpdatedAt.Equal(*comment.CreatedAt),
		IsCensored:     &comment.IsCensored,
		CensoredReason: comment.CensorReason,
	}

	if comment.CreatedUser != nil {
		resp.UserID = comment.CreatedUser.ID
		resp.Username = comment.CreatedUser.Username
		resp.AvatarURL = comment.CreatedUser.AvatarURL
	} else if comment.CreatedBy != nil {
		resp.UserID = *comment.CreatedBy
		resp.Username = "Unknown User"
	}

	for _, reaction := range comment.Reactions {
		utils.AddValueToMap(resp.Reactions, reaction.Type, 1)
		if userID != nil && reaction.UserID != nil && *reaction.UserID == *userID {
			resp.UserReaction = &reaction.Type
		}
	}

	return resp
}

// ToResponseList converts a list of ContentComment models to a list of ContentCommentResponse DTOs
func (ContentCommentResponse) ToResponseList(comments []model.ContentComment, userID *uuid.UUID) []ContentCommentResponse {
	if comments == nil {
		return []ContentCommentResponse{}
	}
	respList := make([]ContentCommentResponse, len(comments))
	for i, comment := range comments {
		respList[i] = *ContentCommentResponse{}.ToResponse(&comment, userID)
	}
	return respList
}
