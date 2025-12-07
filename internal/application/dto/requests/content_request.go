package requests

import (
	"core-backend/pkg/utils"
	"regexp"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

// CreateContentRequest DTO for creating new content
type CreateContentRequest struct {
	TaskID          *uuid.UUID     `json:"task_id" validate:"omitempty,uuid"`
	Title           string         `json:"title" validate:"required,max=500"`
	Description     *string        `json:"description,omitempty" validate:"omitempty,max=1000"`
	Body            any            `json:"body" validate:"required"`
	Type            string         `json:"type" validate:"required,oneof=POST VIDEO"`
	AIGeneratedText *string        `json:"ai_generated_text,omitempty"`
	Channels        []uuid.UUID    `json:"channels" validate:"required,min=1,dive,uuid"`
	BlogFields      *BlogFieldsDTO `json:"blog_fields,omitempty"`

	// Optional affiliateLink fields. Has three stages:
	// 1. If AffiliateLinkID is provided, it indicates an existing affiliate link to be associated.
	// 2. If AffiliateLink is provided (and AffiliateLinkID is nil), affiliateLink record is created and probably used in the Body.
	//	  Check in the Body field for usage of the new affiliate link.
	// 3. If neither is provided, no affiliate link is associated. However, if content associlated with contract of type AFFILIATE,
	// 	  forcefully created an affiliate link record. If the Body does not contains affiliate link,
	//	  then automatically added it at the end of the body.
	AffiliateLink   *string    `json:"affiliate_link,omitempty" validate:"omitempty,max=1000"`
	AffiliateLinkID *uuid.UUID `json:"affiliate_link_id,omitempty" validate:"omitempty,uuid"`
}

// UpdateContentRequest DTO for updating existing content
type UpdateContentRequest struct {
	Title           *string        `json:"title,omitempty" validate:"omitempty,max=500"`
	Description     *string        `json:"description,omitempty" validate:"omitempty,max=1000"`
	Body            *any           `json:"body,omitempty"`
	Type            *string        `json:"type,omitempty" validate:"omitempty,oneof=POST VIDEO"`
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
	Status     *string `form:"status" validate:"omitempty,oneof=DRAFT AWAIT_STAFF AWAIT_BRAND REJECTED APPROVED POSTED"`
	Type       *string `form:"type" validate:"omitempty,oneof=POST VIDEO"`
	BrandID    *string `form:"brand_id" validate:"omitempty,uuid" example:"550e8400-e29b-41d4-a716-446655440000"`
	UserID     *string `form:"user_id" validate:"omitempty,uuid" example:"550e8400-e29b-41d4-a716-446655440000"`
	AssignedTo *string `form:"assigned_to" validate:"omitempty,uuid" example:"550e8400-e29b-41d4-a716-446655440000"`
	TaskID     *string `form:"task_id" validate:"omitempty,uuid" example:"550e8400-e29b-41d4-a716-446655440000"`
	ChannelID  *string `form:"channel_id" validate:"omitempty,uuid" example:"550e8400-e29b-41d4-a716-446655440000"`
	Search     *string `form:"search" validate:"omitempty,max=500" example:"summer sale"`
	FromDate   *string `form:"from_date" validate:"omitempty,datetime=2006-01-02"`
	ToDate     *string `form:"to_date" validate:"omitempty,datetime=2006-01-02"`
}

func ValidateCreateContentRequest(sl validator.StructLevel) {
	request := sl.Current().Interface().(CreateContentRequest)

	affiliateLinkRegex := regexp.MustCompile(`https?://[^\s]+/r/[^\s]+`)
	if request.AffiliateLink != nil && !affiliateLinkRegex.MatchString(*request.AffiliateLink) {
		sl.ReportError(*request.AffiliateLink, "affiliate_link", "AffiliateLink", "affiliate_link.regex", affiliateLinkRegex.String())
	}

	bodyStr := utils.ToString(request.Body)
	if request.AffiliateLink != nil && !strings.Contains(bodyStr, *request.AffiliateLink) {
		sl.ReportError(request.Body, "body", "Body", "affiliate_link.body", *request.AffiliateLink)
	}
}
