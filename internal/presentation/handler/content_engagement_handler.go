package handler

import (
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/iservice"
	customvalidator "core-backend/pkg/custom_validator"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

// ContentEngagementHandler handles content engagement API requests (website only)
type ContentEngagementHandler struct {
	engagementService iservice.ContentEngagementService
	validatorBuilder  *customvalidator.ValidatorBuilder
	validator         *validator.Validate
}

// NewContentEngagementHandler creates a new content engagement handler
func NewContentEngagementHandler(engagementService iservice.ContentEngagementService) *ContentEngagementHandler {
	validatorBuilder := customvalidator.NewValidatorBuilder().
		AddStructValidation(requests.ValidateContentEngagementRequest, requests.ContentEngagementRequest{}).
		AddTranslation("invalid_action", "The action provided is not valid.").
		AddTranslation("add_reaction_required_reaction_type", "reaction_type is required when action is add_reaction.").
		AddTranslation("add_comment_required_comment_text", "comment_text is required when action is add_comment.").
		AddTranslation("edit_comment_required_comment_id", "comment_id is required when action is edit_comment.").
		AddTranslation("edit_comment_required_comment_text", "comment_text is required when action is edit_comment.").
		AddTranslation("delete_comment_required_comment_id", "comment_id is required when action is delete_comment.").
		AddTranslation("add_comment_reaction_required_comment_id", "comment_id is required when action is add_comment_reaction.").
		AddTranslation("add_comment_reaction_required_reaction_type", "reaction_type is required when action is add_comment_reaction.").
		AddTranslation("remove_comment_reaction_required_comment_id", "comment_id is required when action is remove_comment_reaction.")

	return &ContentEngagementHandler{
		engagementService: engagementService,
		validatorBuilder:  validatorBuilder,
		validator:         validatorBuilder.Validate,
	}
}

// RecordEngagement godoc
//
//	@Summary		Record content engagement
//	@Description	Unified endpoint for all content engagement actions on website channel.
//	@Description	Supported actions: add_reaction, remove_reaction, share, add_comment, edit_comment, delete_comment, add_comment_reaction, remove_comment_reaction
//	@Tags			Content Engagement
//	@Accept			json
//	@Produce		json
//	@Param			content_id	path		string								true	"Content ID"
//	@Param			request		body		requests.ContentEngagementRequest	true	"Engagement request"
//	@Success		200			{object}	responses.APIResponse{data=responses.ContentEngagementResponse}
//	@Failure		400			{object}	responses.APIResponse
//	@Failure		401			{object}	responses.APIResponse
//	@Failure		404			{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/contents/{content_id}/engagement [post]
func (h *ContentEngagementHandler) RecordEngagement(c *gin.Context) {
	contentID, err := extractParamID(c, "content_id")
	if err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid content ID", http.StatusBadRequest))
		return
	}
	userID, err := extractUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, responses.ErrorResponse("User not authenticated", http.StatusUnauthorized))
		return
	}

	var req requests.ContentEngagementRequest
	if err = c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid request body", http.StatusBadRequest))
		return
	}
	if errors := h.validatorBuilder.Check(&req); len(errors) > 0 {
		c.JSON(http.StatusBadRequest, responses.ValidationErrorResponse(http.StatusBadRequest, "Invalid request body", errors...))
		return
	}

	req.ContentID = contentID
	req.UserID = userID

	result, err := h.engagementService.RecordEngagement(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse(err.Error(), http.StatusBadRequest))
		return
	}

	c.JSON(http.StatusOK, responses.SuccessResponse("Engagement recorded", nil, result))
}

// GetEngagementSummary godoc
//
//	@Summary		Get engagement summary for content
//	@Description	Returns engagement summary (reactions, comments, shares) for a specific content on website channel
//	@Tags			Content Engagement
//	@Accept			json
//	@Produce		json
//	@Param			content_id	path		string	true	"Content ID"
//	@Success		200			{object}	responses.APIResponse{data=responses.WebsiteEngagementSummary}
//	@Failure		400			{object}	responses.APIResponse
//	@Failure		404			{object}	responses.APIResponse
//	@Router			/api/v1/contents/{content_id}/engagement [get]
func (h *ContentEngagementHandler) GetEngagementSummary(c *gin.Context) {
	contentID, err := extractParamID(c, "content_id")
	if err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid content ID", http.StatusBadRequest))
		return
	}

	summary, err := h.engagementService.GetEngagementSummary(c.Request.Context(), contentID)
	if err != nil {
		c.JSON(http.StatusNotFound, responses.ErrorResponse(err.Error(), http.StatusNotFound))
		return
	}

	c.JSON(http.StatusOK, responses.SuccessResponse("Engagement summary retrieved", nil, summary))
}

// GetUserEngagementStatus godoc
//
//	@Summary		Get user's engagement status on content
//	@Description	Returns whether the current user has liked/shared the content
//	@Tags			Content Engagement
//	@Accept			json
//	@Produce		json
//	@Param			content_id	path		string	true	"Content ID"
//	@Success		200			{object}	responses.APIResponse{data=responses.UserEngagementStatus}
//	@Failure		400			{object}	responses.APIResponse
//	@Failure		401			{object}	responses.APIResponse
//	@Failure		404			{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/contents/{content_id}/engagement/status [get]
func (h *ContentEngagementHandler) GetUserEngagementStatus(c *gin.Context) {
	contentID, err := extractParamID(c, "content_id")
	if err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid content ID", http.StatusBadRequest))
		return
	}
	userID, err := extractUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, responses.ErrorResponse("User not authenticated", http.StatusUnauthorized))
		return
	}

	status, err := h.engagementService.GetUserEngagementStatus(c.Request.Context(), contentID, userID)
	if err != nil {
		c.JSON(http.StatusNotFound, responses.ErrorResponse(err.Error(), http.StatusNotFound))
		return
	}

	c.JSON(http.StatusOK, responses.SuccessResponse("User engagement status retrieved", nil, status))
}
