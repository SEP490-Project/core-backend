package handler

import (
	"net/http"

	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/application/interfaces/iservice"
	"core-backend/internal/domain/enum"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type ContentHandler struct {
	contentService       iservice.ContentService
	stateTransferService iservice.StateTransferService
	unitOfWork           irepository.UnitOfWork
	validator            *validator.Validate
}

func NewContentHandler(
	contentService iservice.ContentService,
	stateTransferService iservice.StateTransferService,
	unitOfWork irepository.UnitOfWork,
) *ContentHandler {
	return &ContentHandler{
		contentService:       contentService,
		stateTransferService: stateTransferService,
		unitOfWork:           unitOfWork,
		validator:            validator.New(),
	}
}

// Create creates new content draft
//
//	@Summary		Create content draft
//	@Description	Creates new blog post or video content with DRAFT status
//	@Tags			Content
//	@Accept			json
//	@Produce		json
//	@Param			request	body		requests.CreateContentRequest	true	"Content creation data"
//	@Success		201		{object}	responses.APIResponse{data=responses.ContentResponse}
//	@Failure		400		{object}	responses.APIResponse	"Validation error or invalid request"
//	@Failure		404		{object}	responses.APIResponse	"Task not found"
//	@Failure		500		{object}	responses.APIResponse	"Internal server error"
//	@Security		BearerAuth
//	@Router			/api/v1/contents [post]
func (h *ContentHandler) Create(c *gin.Context) {
	var req requests.CreateContentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		zap.L().Error("Failed to bind CreateContentRequest", zap.Error(err))
		response := responses.ErrorResponse("Invalid request format", http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}
	if err := h.validator.Struct(&req); err != nil {
		c.JSON(http.StatusBadRequest, processValidationError(err))
		return
	}

	uow := h.unitOfWork.Begin(c.Request.Context())
	content, err := h.contentService.Create(c.Request.Context(), uow, &req)
	if err != nil {
		uow.Rollback()
		statusCode := http.StatusInternalServerError
		message := "Failed to create content"

		switch err.Error() {
		case "task not found":
			statusCode = http.StatusNotFound
			message = err.Error()
		case "failed to create content", "failed to create blog", "failed to create content channel":
			statusCode = http.StatusInternalServerError
			message = err.Error()
		default:
			statusCode = http.StatusBadRequest
			message = err.Error()
		}

		response := responses.ErrorResponse(message, statusCode)
		c.JSON(statusCode, response)
		return
	}

	uow.Commit()
	statusCode := http.StatusCreated
	response := responses.SuccessResponse("Content created successfully", &statusCode, content)
	c.JSON(http.StatusCreated, response)
}

// Update updates existing content draft
//
//	@Summary		Update content draft
//	@Description	Updates content in DRAFT or REJECTED status
//	@Tags			Content
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string							true	"Content ID (UUID)"
//	@Param			request	body		requests.UpdateContentRequest	true	"Content update data"
//	@Success		200		{object}	responses.APIResponse{data=responses.ContentResponse}
//	@Failure		400		{object}	responses.APIResponse	"Validation error or invalid request"
//	@Failure		404		{object}	responses.APIResponse	"Content not found"
//	@Failure		409		{object}	responses.APIResponse	"Content status not editable"
//	@Failure		500		{object}	responses.APIResponse	"Internal server error"
//	@Security		BearerAuth
//	@Router			/api/v1/contents/{id} [put]
func (h *ContentHandler) Update(c *gin.Context) {
	id, err := extractParamID(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid content ID format", http.StatusBadRequest))
		return
	}

	var req requests.UpdateContentRequest
	if err = c.ShouldBindJSON(&req); err != nil {
		response := responses.ErrorResponse("Invalid request format", http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}
	if err = h.validator.Struct(&req); err != nil {
		c.JSON(http.StatusBadRequest, processValidationError(err))
		return
	}

	content, err := h.contentService.Update(c.Request.Context(), id, &req)
	if err != nil {
		statusCode := http.StatusInternalServerError
		message := "Failed to update content"

		switch err.Error() {
		case "content not found":
			statusCode = http.StatusNotFound
			message = err.Error()
		case "only DRAFT or REJECTED content can be updated":
			statusCode = http.StatusConflict
			message = err.Error()
		case "failed to update content":
			statusCode = http.StatusInternalServerError
			message = err.Error()
		default:
			statusCode = http.StatusBadRequest
			message = err.Error()
		}

		response := responses.ErrorResponse(message, statusCode)
		c.JSON(statusCode, response)
		return
	}

	response := responses.SuccessResponse("Content updated successfully", nil, content)
	c.JSON(http.StatusOK, response)
}

// GetByID retrieves content by ID
//
//	@Summary		Get content by ID
//	@Description	Retrieves content with all relationships (blog, author, channels)
//	@Tags			Content
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string	true	"Content ID (UUID)"
//	@Success		200	{object}	responses.APIResponse{data=responses.ContentResponse}
//	@Failure		400	{object}	responses.APIResponse	"Invalid content ID"
//	@Failure		404	{object}	responses.APIResponse	"Content not found"
//	@Failure		500	{object}	responses.APIResponse	"Internal server error"
//	@Security		BearerAuth
//	@Router			/api/v1/contents/{id} [get]
func (h *ContentHandler) GetByID(c *gin.Context) {
	id, err := extractParamID(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid content ID format", http.StatusBadRequest))
		return
	}

	content, err := h.contentService.GetByID(c.Request.Context(), id)
	if err != nil {
		statusCode := http.StatusInternalServerError
		message := "Failed to retrieve content"

		if err.Error() == "content not found" {
			statusCode = http.StatusNotFound
			message = err.Error()
		}

		response := responses.ErrorResponse(message, statusCode)
		c.JSON(statusCode, response)
		return
	}

	response := responses.SuccessResponse("Content retrieved successfully", nil, content)
	c.JSON(http.StatusOK, response)
}

// Delete soft deletes content
//
//	@Summary		Delete content draft
//	@Description	Soft deletes content in DRAFT or REJECTED status
//	@Tags			Content
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string					true	"Content ID (UUID)"
//	@Success		200	{object}	responses.APIResponse	"Content deleted successfully"
//	@Failure		400	{object}	responses.APIResponse	"Invalid content ID"
//	@Failure		404	{object}	responses.APIResponse	"Content not found"
//	@Failure		409	{object}	responses.APIResponse	"Content status not deletable"
//	@Failure		500	{object}	responses.APIResponse	"Internal server error"
//	@Security		BearerAuth
//	@Router			/api/v1/contents/{id} [delete]
func (h *ContentHandler) Delete(c *gin.Context) {
	id, err := extractParamID(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid content ID format", http.StatusBadRequest))
		return
	}

	err = h.contentService.Delete(c.Request.Context(), id)
	if err != nil {
		statusCode := http.StatusInternalServerError
		message := "Failed to delete content"

		switch err.Error() {
		case "content not found":
			statusCode = http.StatusNotFound
			message = err.Error()
		case "only DRAFT or REJECTED content can be deleted":
			statusCode = http.StatusConflict
			message = err.Error()
		default:
			statusCode = http.StatusInternalServerError
			message = err.Error()
		}

		response := responses.ErrorResponse(message, statusCode)
		c.JSON(statusCode, response)
		return
	}

	response := responses.SuccessResponse("Content deleted successfully", nil, nil)
	c.JSON(http.StatusOK, response)
}

// Submit godoc
//
//	@Summary		Submit content for review
//	@Description	Transitions content from DRAFT/REJECTED to AWAIT_STAFF or AWAIT_BRAND based on channels
//	@Tags			Content
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string	true	"Content ID"
//	@Success		200	{object}	responses.APIResponse{data=map[string]any}
//	@Failure		400	{object}	responses.APIResponse
//	@Failure		404	{object}	responses.APIResponse
//	@Failure		409	{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/contents/{id}/submit [patch]
func (h *ContentHandler) Submit(c *gin.Context) {
	// Parse content ID from path
	contentID, err := extractParamID(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid content ID format", http.StatusBadRequest))
		return
	}

	// Extract user ID from context
	var userID uuid.UUID
	userID, err = extractUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, responses.ErrorResponse(err.Error(), http.StatusUnauthorized))
		return
	}

	// Validate content exists and get current status
	content, err := h.contentService.GetByID(c.Request.Context(), contentID)
	if err != nil {
		c.JSON(http.StatusNotFound, responses.ErrorResponse("content not found", http.StatusNotFound))
		return
	}

	// Validate current status (must be DRAFT or REJECTED)
	if content.Status != "DRAFT" && content.Status != "REJECTED" {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("only DRAFT or REJECTED content can be submitted", http.StatusBadRequest))
		return
	}

	// Validate required fields and affiliate link through ContentService
	if err = h.contentService.ValidateForSubmission(c.Request.Context(), contentID); err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse(err.Error(), http.StatusBadRequest))
		return
	}

	// Determine target status based on workflow routing
	var targetStatus enum.ContentStatus
	targetStatus, err = h.contentService.DetermineWorkflowRoute(c.Request.Context(), contentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse(err.Error(), http.StatusInternalServerError))
		return
	}

	// Begin transaction
	uow := h.unitOfWork.Begin(c.Request.Context())
	defer func() {
		if r := recover(); r != nil {
			uow.Rollback()
			panic(r)
		}
	}()

	// Transition state through FSM with UnitOfWork
	if err := h.stateTransferService.MoveContentToState(c.Request.Context(), uow, contentID, targetStatus, userID); err != nil {
		uow.Rollback()
		c.JSON(http.StatusConflict, responses.ErrorResponse("failed to submit content: "+err.Error(), http.StatusConflict))
		return
	}

	// Commit transaction
	if err := uow.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse("failed to commit transaction", http.StatusInternalServerError))
		return
	}

	zap.L().Info("Content submitted successfully",
		zap.String("content_id", contentID.String()),
		zap.String("submitter_id", userID.String()),
		zap.String("new_status", string(targetStatus)))

	c.JSON(http.StatusOK, responses.SuccessResponse("Content submitted successfully", nil, map[string]any{
		"id":     contentID.String(),
		"status": targetStatus,
	}))
}

// Approve godoc
//
//	@Summary		Approve content
//	@Description	Transitions content from AWAIT_STAFF/AWAIT_BRAND to APPROVED
//	@Tags			Content
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string	true	"Content ID"
//	@Success		200	{object}	responses.APIResponse{data=map[string]any}
//	@Failure		400	{object}	responses.APIResponse
//	@Failure		404	{object}	responses.APIResponse
//	@Failure		409	{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/contents/{id}/approve [patch]
func (h *ContentHandler) Approve(c *gin.Context) {
	// Parse content ID from path
	contentID, err := extractParamID(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid content ID format", http.StatusBadRequest))
		return
	}

	// Extract user ID from context
	var userID uuid.UUID
	userID, err = extractUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, responses.ErrorResponse(err.Error(), http.StatusUnauthorized))
		return
	}

	// Validate content exists and current status
	content, err := h.contentService.GetByID(c.Request.Context(), contentID)
	if err != nil {
		c.JSON(http.StatusNotFound, responses.ErrorResponse("content not found", http.StatusNotFound))
		return
	}

	if content.Status != enum.ContentStatusAwaitStaff && content.Status != enum.ContentStatusAwaitBrand {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("only content awaiting review can be approved", http.StatusBadRequest))
		return
	}

	// Begin transaction
	uow := h.unitOfWork.Begin(c.Request.Context())
	defer func() {
		if r := recover(); r != nil {
			uow.Rollback()
			panic(r)
		}
	}()

	// Transition state through FSM with UnitOfWork
	if err := h.stateTransferService.MoveContentToState(c.Request.Context(), uow, contentID, enum.ContentStatusApproved, userID); err != nil {
		uow.Rollback()
		c.JSON(http.StatusConflict, responses.ErrorResponse("failed to approve content: "+err.Error(), http.StatusConflict))
		return
	}

	// Commit transaction
	if err := uow.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse("failed to commit transaction", http.StatusInternalServerError))
		return
	}

	zap.L().Info("Content approved successfully",
		zap.String("content_id", contentID.String()),
		zap.String("approver_id", userID.String()))

	c.JSON(http.StatusOK, responses.SuccessResponse("Content approved successfully", nil, map[string]any{
		"id":     contentID.String(),
		"status": enum.ContentStatusApproved.String(),
	}))
}

// Reject godoc
//
//	@Summary		Reject content
//	@Description	Transitions content from AWAIT_STAFF/AWAIT_BRAND to REJECTED with feedback
//	@Tags			Content
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string							true	"Content ID"
//	@Param			body	body		requests.RejectContentRequest	true	"Rejection reason"	example({"reason":"Quality does not meet standards"})
//	@Success		200		{object}	responses.APIResponse{data=map[string]any}
//	@Failure		400		{object}	responses.APIResponse
//	@Failure		404		{object}	responses.APIResponse
//	@Failure		409		{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/contents/{id}/reject [patch]
func (h *ContentHandler) Reject(c *gin.Context) {
	// Parse content ID from path
	contentID, err := extractParamID(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid content ID format", http.StatusBadRequest))
		return
	}

	// Extract user ID from context
	var userID uuid.UUID
	userID, err = extractUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, responses.ErrorResponse(err.Error(), http.StatusUnauthorized))
		return
	}

	// Parse request body for required reason
	var req requests.RejectContentRequest
	if err = c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("reason is required (minimum 10 characters)", http.StatusBadRequest))
		return
	}

	// Validate content exists and current status
	var content *responses.ContentResponse
	content, err = h.contentService.GetByID(c.Request.Context(), contentID)
	if err != nil {
		c.JSON(http.StatusNotFound, responses.ErrorResponse("content not found", http.StatusNotFound))
		return
	}

	if content.Status != enum.ContentStatusAwaitStaff && content.Status != enum.ContentStatusAwaitBrand {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("only content awaiting review can be rejected", http.StatusBadRequest))
		return
	}

	// Begin transaction
	uow := h.unitOfWork.Begin(c.Request.Context())
	defer func() {
		if r := recover(); r != nil {
			uow.Rollback()
			panic(r)
		}
	}()

	// 1. Transition state through FSM with UnitOfWork
	if err = h.stateTransferService.MoveContentToState(c.Request.Context(), uow, contentID, enum.ContentStatusRejected, userID); err != nil {
		uow.Rollback()
		c.JSON(http.StatusConflict, responses.ErrorResponse("failed to reject content: "+err.Error(), http.StatusConflict))
		return
	}

	// 2. Store rejection feedback using UnitOfWork
	if err = h.contentService.SetRejectionFeedback(c.Request.Context(), uow, contentID, req.Feedback); err != nil {
		uow.Rollback()
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse(err.Error(), http.StatusInternalServerError))
		return
	}

	// Commit transaction
	if err = uow.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse("failed to commit transaction", http.StatusInternalServerError))
		return
	}

	zap.L().Info("Content rejected successfully",
		zap.String("content_id", contentID.String()),
		zap.String("reviewer_id", userID.String()),
		zap.String("feedback", req.Feedback))

	c.JSON(http.StatusOK, responses.SuccessResponse("Content rejected successfully", nil, map[string]any{
		"id":     contentID.String(),
		"status": "REJECTED",
	}))
}

// Publish godoc
//
//	@Summary		Publish content
//	@Description	Transitions content from APPROVED to POSTED with optional publish date
//	@Tags			Content
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string							true	"Content ID"
//	@Param			body	body		requests.PublishContentRequest	false	"Optional publish date"	example({"publish_date":"2025-12-25T10:00:00Z"})
//	@Success		200		{object}	responses.APIResponse{data=map[string]any}
//	@Failure		400		{object}	responses.APIResponse
//	@Failure		404		{object}	responses.APIResponse
//	@Failure		409		{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/contents/{id}/publish [patch]
func (h *ContentHandler) Publish(c *gin.Context) {
	// Parse content ID from path
	contentID, err := extractParamID(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid content ID format", http.StatusBadRequest))
		return
	}

	// Extract user ID from context
	var userID uuid.UUID
	userID, err = extractUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, responses.ErrorResponse(err.Error(), http.StatusUnauthorized))
		return
	}

	// Parse request body for optional publish_date
	// Allow empty body - publish_date will be nil
	var req requests.PublishContentRequest
	_ = c.ShouldBindJSON(&req)

	// Validate content exists and current status
	content, err := h.contentService.GetByID(c.Request.Context(), contentID)
	if err != nil {
		c.JSON(http.StatusNotFound, responses.ErrorResponse("content not found", http.StatusNotFound))
		return
	}

	if content.Status != enum.ContentStatusApproved {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("only approved content can be published", http.StatusBadRequest))
		return
	}

	// Begin transaction
	uow := h.unitOfWork.Begin(c.Request.Context())
	defer func() {
		if r := recover(); r != nil {
			uow.Rollback()
			panic(r)
		}
	}()

	// 1. Transition state through FSM with UnitOfWork
	if err := h.stateTransferService.MoveContentToState(c.Request.Context(), uow, contentID, enum.ContentStatusPosted, userID); err != nil {
		uow.Rollback()
		c.JSON(http.StatusConflict, responses.ErrorResponse("failed to publish content: "+err.Error(), http.StatusConflict))
		return
	}

	// 2. Set publish date using UnitOfWork
	if err := h.contentService.SetPublishDate(c.Request.Context(), uow, contentID, req.PublishDate); err != nil {
		uow.Rollback()
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse(err.Error(), http.StatusInternalServerError))
		return
	}

	// Commit transaction
	if err := uow.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse("failed to commit transaction", http.StatusInternalServerError))
		return
	}

	// Get updated content for response
	updatedContent, _ := h.contentService.GetByID(c.Request.Context(), contentID)

	zap.L().Info("Content published successfully",
		zap.String("content_id", contentID.String()),
		zap.String("publisher_id", userID.String()))

	c.JSON(http.StatusOK, responses.SuccessResponse("Content published successfully", nil, map[string]any{
		"id":           contentID.String(),
		"status":       "POSTED",
		"publish_date": updatedContent.PublishDate,
	}))
}

// List retrieves paginated content with filters
//
//	@Summary		List content
//	@Description	Retrieves paginated content with optional filters, search, and sorting
//	@Tags			Content
//	@Accept			json
//	@Produce		json
//	@Param			page		query		int		false	"Page number (default: 1)"
//	@Param			limit		query		int		false	"Items per page (default: 10, max: 100)"
//	@Param			sort_by		query		string	false	"Sort by field"		Enums(created_at, updated_at, publish_date, title)
//	@Param			sort_order	query		string	false	"Sort order"		Enums(asc, desc)
//	@Param			status		query		string	false	"Filter by status"	Enums(DRAFT, AWAIT_STAFF, AWAIT_BRAND, REJECTED, APPROVED, POSTED)
//	@Param			type		query		string	false	"Filter by type"	Enums(POST, VIDEO)
//	@Param			task_id		query		string	false	"Filter by task ID (UUID)"
//	@Param			assigned_to	query		string	false	"Filter by assigned user ID (UUID)"
//	@Param			channel_id	query		string	false	"Filter by channel ID (UUID)"
//	@Param			search		query		string	false	"Search in title and body"
//	@Param			from_date	query		string	false	"Filter from date (YYYY-MM-DD)"
//	@Param			to_date		query		string	false	"Filter to date (YYYY-MM-DD)"
//	@Success		200			{object}	responses.ContentPaginationResponse
//	@Failure		400			{object}	responses.APIResponse	"Invalid request parameters"
//	@Failure		401			{object}	responses.APIResponse	"Authentication required"
//	@Failure		500			{object}	responses.APIResponse	"Internal server error"
//	@Security		BearerAuth
//	@Router			/api/v1/contents [get]
func (h *ContentHandler) List(c *gin.Context) {
	// Bind query parameters
	var req requests.ContentFilterRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response := responses.ErrorResponse("Invalid query parameters: "+err.Error(), http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	// Call service to list content
	contents, total, err := h.contentService.List(c.Request.Context(), &req)
	if err != nil {
		response := responses.ErrorResponse("Failed to retrieve content list", http.StatusInternalServerError)
		c.JSON(http.StatusInternalServerError, response)
		return
	}

	response := responses.NewPaginationResponse(
		"Content list retrieved successfully",
		http.StatusOK,
		contents,
		responses.Pagination{
			Page:  req.Page,
			Limit: req.Limit,
			Total: total,
		},
	)
	c.JSON(http.StatusOK, response)
}

// ListByAssignedUser retrieves paginated content assigned to the current authenticated staff user
//
// //	@Summary		List content assigned to ListByAssignedUser
//
//	@Description	Retrieves paginated content assigned to the current authenticated staff user
//	@Tags			Content
//	@Accept			json
//	@Produce		json
//	@Param			page		query		int		false	"Page number (default: 1)"
//	@Param			limit		query		int		false	"Items per page (default: 10, max: 100)"
//	@Param			sort_by		query		string	false	"Sort by field"		Enums(created_at, updated_at, publish_date, title)
//	@Param			sort_order	query		string	false	"Sort order"		Enums(asc, desc)
//	@Param			status		query		string	false	"Filter by status"	Enums(DRAFT, AWAIT_STAFF, AWAIT_BRAND, REJECTED, APPROVED, POSTED)
//	@Param			type		query		string	false	"Filter by type"	Enums(POST, VIDEO)
//	@Param			task_id		query		string	false	"Filter by task ID (UUID)"
//	@Param			channel_id	query		string	false	"Filter by channel ID (UUID)"
//	@Param			search		query		string	false	"Search in title and body"
//	@Param			from_date	query		string	false	"Filter from date (YYYY-MM-DD)"
//	@Param			to_date		query		string	false	"Filter to date (YYYY-MM-DD)"
//	@Success		200			{object}	responses.ContentPaginationResponse
//	@Failure		400			{object}	responses.APIResponse	"Invalid request parameters"
//	@Failure		401			{object}	responses.APIResponse	"Authentication required"
//	@Failure		500			{object}	responses.APIResponse	"Internal server error"
//	@Security		BearerAuth
//	@Router			/api/v1/contents/assigned_to [get]
func (h *ContentHandler) ListByAssignedUser(c *gin.Context) {
	assignedToID, err := extractUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized,
			responses.ErrorResponse("Authentication required: "+err.Error(), http.StatusUnauthorized))
		return
	}
	var req requests.ContentFilterRequest
	if err = c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest,
			responses.ErrorResponse("Invalid query parameters: "+err.Error(), http.StatusBadRequest))
		return
	}
	req.AssignedTo = &assignedToID
	if err = h.validator.Struct(&req); err != nil {
		c.JSON(http.StatusBadRequest, processValidationError(err))
		return
	}

	// Call service to list content
	contents, total, err := h.contentService.List(c.Request.Context(), &req)
	if err != nil {
		response := responses.ErrorResponse("Failed to retrieve content list", http.StatusInternalServerError)
		c.JSON(http.StatusInternalServerError, response)
		return
	}

	response := responses.NewPaginationResponse(
		"Content list retrieved successfully",
		http.StatusOK,
		contents,
		responses.Pagination{
			Page:  req.Page,
			Limit: req.Limit,
			Total: total,
		},
	)
	c.JSON(http.StatusOK, response)
}

// ListPublic retrieves paginated public content (status=POSTED)
//
//	@Summary		List public content
//	@Description	Retrieves paginated content with status=POSTED. Filters like assigned_to or task_id are ignored for guests.
//	@Tags			Content
//	@Accept			json
//	@Produce		json
//	@Param			page		query		int		false	"Page number (default: 1)"
//	@Param			limit		query		int		false	"Items per page (default: 10, max: 100)"
//	@Param			sort_by		query		string	false	"Sort by field"		Enums(created_at, updated_at, publish_date, title)
//	@Param			sort_order	query		string	false	"Sort order"		Enums(asc, desc)
//	@Param			type		query		string	false	"Filter by type"	Enums(POST, VIDEO)
//	@Param			channel_id	query		string	false	"Filter by channel ID (UUID)"
//	@Param			search		query		string	false	"Search in title and body"
//	@Param			from_date	query		string	false	"Filter from date (YYYY-MM-DD)"
//	@Param			to_date		query		string	false	"Filter to date (YYYY-MM-DD)"
//	@Success		200			{object}	responses.ContentPaginationResponse
//	@Failure		400			{object}	responses.APIResponse	"Invalid request parameters"
//	@Failure		500			{object}	responses.APIResponse	"Internal server error"
//	@Router			/api/v1/contents/public [get]
func (h *ContentHandler) ListPublic(c *gin.Context) {
	var req requests.ContentFilterRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid query parameters: "+err.Error(), http.StatusBadRequest))
		return
	}

	// Force status = POSTED
	status := "POSTED"
	req.Status = &status

	// Remove filters that are not needed for public
	req.AssignedTo = nil
	req.TaskID = nil

	contents, total, err := h.contentService.List(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse("Failed to retrieve content list", http.StatusInternalServerError))
		return
	}

	response := responses.NewPaginationResponse(
		"Public content list retrieved successfully",
		http.StatusOK,
		contents,
		responses.Pagination{
			Page:  req.Page,
			Limit: req.Limit,
			Total: total,
		},
	)
	c.JSON(http.StatusOK, response)
}

// GetByIDPublic retrieves public content by ID (status=POSTED)
//
//	@Summary		Get public content by ID
//	@Description	Retrieves content with all relationships (blog, author, channels). Only content with status=POSTED is returned.
//	@Tags			Content
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string	true	"Content ID (UUID)"
//	@Success		200	{object}	responses.APIResponse{data=responses.ContentResponse}
//	@Failure		400	{object}	responses.APIResponse	"Invalid content ID"
//	@Failure		404	{object}	responses.APIResponse	"Content not found or not POSTED"
//	@Failure		500	{object}	responses.APIResponse	"Internal server error"
//	@Router			/api/v1/contents/public/{id} [get]
func (h *ContentHandler) GetByIDPublic(c *gin.Context) {
	id, err := extractParamID(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid content ID format", http.StatusBadRequest))
		return
	}

	content, err := h.contentService.GetByID(c.Request.Context(), id)
	if err != nil {
		statusCode := http.StatusInternalServerError
		message := "Failed to retrieve content"

		if err.Error() == "content not found" {
			statusCode = http.StatusNotFound
			message = err.Error()
		}

		c.JSON(statusCode, responses.ErrorResponse(message, statusCode))
		return
	}

	// Check status
	if content.Status != "POSTED" {
		c.JSON(http.StatusNotFound, responses.ErrorResponse("Content not found", http.StatusNotFound))
		return
	}

	c.JSON(http.StatusOK, responses.SuccessResponse("Public content retrieved successfully", nil, content))
}
