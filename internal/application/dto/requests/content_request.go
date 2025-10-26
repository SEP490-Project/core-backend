package requests

import "github.com/google/uuid"

// CreateContentRequest DTO for creating new content
type CreateContentRequest struct {
	TaskID          *uuid.UUID     `json:"task_id" validate:"omitempty,uuid"`
	Title           string         `json:"title" validate:"required,max=500"`
	Body            string         `json:"body" validate:"required"`
	Type            string         `json:"type" validate:"required,oneof=POST VIDEO"`
	AffiliateLink   *string        `json:"affiliate_link,omitempty" validate:"omitempty,max=1000"`
	AIGeneratedText *string        `json:"ai_generated_text,omitempty"`
	Channels        []uuid.UUID    `json:"channels" validate:"required,min=1,dive,uuid"`
	BlogFields      *BlogFieldsDTO `json:"blog_fields,omitempty"`
}

// UpdateContentRequest DTO for updating existing content
type UpdateContentRequest struct {
	Title           *string        `json:"title,omitempty" validate:"omitempty,max=500"`
	Body            *string        `json:"body,omitempty"`
	Type            *string        `json:"type,omitempty" validate:"omitempty,oneof=POST VIDEO"`
	AffiliateLink   *string        `json:"affiliate_link,omitempty" validate:"omitempty,max=1000"`
	AIGeneratedText *string        `json:"ai_generated_text,omitempty"`
	Channels        []uuid.UUID    `json:"channels,omitempty" validate:"omitempty,dive,uuid"`
	BlogFields      *BlogFieldsDTO `json:"blog_fields,omitempty"`
}

// SubmitContentRequest DTO for submitting content for review
type SubmitContentRequest struct {
	Message *string `json:"message,omitempty" validate:"omitempty,max=500"`
}

// ApproveContentRequest DTO for approving content
type ApproveContentRequest struct {
	Message *string `json:"message,omitempty" validate:"omitempty,max=500"`
}

// RejectContentRequest DTO for rejecting content
type RejectContentRequest struct {
	Feedback string `json:"feedback" validate:"required,max=1000"`
}

// PublishContentRequest DTO for publishing approved content
type PublishContentRequest struct {
	PublishDate *string `json:"publish_date,omitempty" validate:"omitempty,datetime=2006-01-02T15:04:05Z07:00"`
}

// ContentFilterRequest DTO for listing and filtering content
type ContentFilterRequest struct {
	PaginationRequest
	Status    *string    `form:"status" validate:"omitempty,oneof=DRAFT AWAIT_STAFF AWAIT_BRAND REJECTED APPROVED POSTED"`
	Type      *string    `form:"type" validate:"omitempty,oneof=POST VIDEO"`
	TaskID    *uuid.UUID `form:"task_id" validate:"omitempty,uuid"`
	ChannelID *uuid.UUID `form:"channel_id" validate:"omitempty,uuid"`
	Search    *string    `form:"search" validate:"omitempty,max=500"`
	FromDate  *string    `form:"from_date" validate:"omitempty,datetime=2006-01-02"`
	ToDate    *string    `form:"to_date" validate:"omitempty,datetime=2006-01-02"`
}
