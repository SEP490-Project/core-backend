package requests

import (
	"core-backend/internal/domain/enum"
	"core-backend/pkg/utils"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

var (
	validReactionTypes = []string{enum.ReactionTypeLike.String(), enum.ReactionTypeLove.String(), enum.ReactionTypeWow.String(), enum.ReactionTypeHaha.String(), enum.ReactionTypeSad.String(), enum.ReactionTypeAngry.String(), enum.ReactionTypeThankful.String()}
	validActions       = []string{string(EngagementActionAddReaction), string(EngagementActionRemoveReaction), string(EngagementActionShare), string(EngagementActionAddComment), string(EngagementActionEditComment), string(EngagementActionDeleteComment), string(EngagementActionAddCommentReaction), string(EngagementActionRemoveCommentReaction)}
)

// ContentEngagementAction defines the type of engagement action
type ContentEngagementAction string

const (
	// Content-level actions (on WEBSITE ContentChannel)
	EngagementActionAddReaction    ContentEngagementAction = "add_reaction"
	EngagementActionRemoveReaction ContentEngagementAction = "remove_reaction"
	EngagementActionShare          ContentEngagementAction = "share"

	// Comment-level actions
	EngagementActionAddComment            ContentEngagementAction = "add_comment"
	EngagementActionEditComment           ContentEngagementAction = "edit_comment"
	EngagementActionDeleteComment         ContentEngagementAction = "delete_comment"
	EngagementActionAddCommentReaction    ContentEngagementAction = "add_comment_reaction"
	EngagementActionRemoveCommentReaction ContentEngagementAction = "remove_comment_reaction"
)

func (cea ContentEngagementAction) IsValid() bool {
	switch cea {
	case EngagementActionAddReaction, EngagementActionRemoveReaction, EngagementActionShare,
		EngagementActionAddComment, EngagementActionEditComment, EngagementActionDeleteComment,
		EngagementActionAddCommentReaction, EngagementActionRemoveCommentReaction:
		return true
	}
	return false
}

// ContentEngagementRequest represents a request to engage with content
// Validation rules:
// - add_reaction: requires reaction_type
// - remove_reaction: no additional fields needed (removes user's current reaction)
// - share: no additional fields
// - add_comment: requires comment_text
// - edit_comment: requires comment_id and comment_text
// - delete_comment: requires comment_id
// - add_comment_reaction: requires comment_id and reaction_type
// - remove_comment_reaction: requires comment_id
type ContentEngagementRequest struct {
	Action       ContentEngagementAction `json:"action" validate:"required,oneof=add_reaction remove_reaction share add_comment edit_comment delete_comment add_comment_reaction remove_comment_reaction"`
	ReactionType *enum.ReactionType      `json:"reaction_type,omitempty" validate:"omitempty,oneof=LIKE LOVE WOW HAHA SAD ANGRY THANKFUL"`
	CommentID    *uuid.UUID              `json:"comment_id,omitempty" validate:"omitempty,uuid"`
	CommentText  *string                 `json:"comment_text,omitempty" validate:"omitempty,max=2000"`

	// Internal fields
	ContentID uuid.UUID `json:"-"`
	UserID    uuid.UUID `json:"-"`
}

// IsValid validates the request based on the action type

// ReactToContentRequest represents a request to react to content
type ReactToContentRequest struct {
	ContentID    uuid.UUID `json:"content_id" validate:"required,uuid"`
	ReactionType string    `json:"reaction_type" validate:"required,oneof=LIKE LOVE WOW HAHA SAD ANGRY THANKFUL"`
	UserID       uuid.UUID `json:"-"` // Populated from JWT context
}

// CommentOnContentRequest represents a request to add a comment to content
type CommentOnContentRequest struct {
	ContentID   uuid.UUID  `json:"content_id" validate:"required,uuid"`
	ParentID    *uuid.UUID `json:"parent_id,omitempty" validate:"omitempty,uuid"` // For reply comments
	CommentText string     `json:"comment_text" validate:"required,min=1,max=2000"`
	UserID      uuid.UUID  `json:"-"` // Populated from JWT context
}

// ShareContentRequest represents a request to share content
type ShareContentRequest struct {
	ContentID uuid.UUID `json:"content_id" validate:"required,uuid"`
	Platform  *string   `json:"platform,omitempty" validate:"omitempty,oneof=FACEBOOK TWITTER LINKEDIN COPY_LINK"` // Optional share platform
	UserID    uuid.UUID `json:"-"`                                                                                 // Populated from JWT context
}

func ValidateContentEngagementRequest(sl validator.StructLevel) {
	request := sl.Current().Interface().(ContentEngagementRequest)

	if !request.Action.IsValid() {
		sl.ReportError(request.Action, "action", "Action", "invalid_action", utils.ToString(validActions))
		return
	}

	switch request.Action {
	case EngagementActionAddReaction:
		if request.ReactionType == nil {
			sl.ReportError(request.ReactionType, "reaction_type", "ReactionType", "add_reaction_required_reaction_type", utils.ToString(validReactionTypes))
		}

	case EngagementActionAddComment:
		if request.CommentText == nil || *request.CommentText == "" {
			sl.ReportError(request.CommentText, "comment_text", "CommentText", "add_comment_required_comment_text", "")
		}

	case EngagementActionEditComment:
		if request.CommentID == nil {
			sl.ReportError(request.CommentID, "comment_id", "CommentID", "edit_comment_required_comment_id", "")
		}
		if request.CommentText == nil || *request.CommentText == "" {
			sl.ReportError(request.CommentText, "comment_text", "CommentText", "edit_comment_required_comment_text", "")
		}

	case EngagementActionDeleteComment:
		if request.CommentID == nil {
			sl.ReportError(request.CommentID, "comment_id", "CommentID", "delete_comment_required_comment_id", "")
		}

	case EngagementActionAddCommentReaction:
		if request.CommentID == nil {
			sl.ReportError(request.CommentID, "comment_id", "CommentID", "add_comment_reaction_required_comment_id", "")
		}
		if request.ReactionType == nil {
			sl.ReportError(request.ReactionType, "reaction_type", "ReactionType", "add_comment_reaction_required_reaction_type", utils.ToString(validReactionTypes))
		}

	case EngagementActionRemoveCommentReaction:
		if request.CommentID == nil {
			sl.ReportError(request.CommentID, "comment_id", "CommentID", "remove_comment_reaction_required_comment_id", "")
		}
	}
}
