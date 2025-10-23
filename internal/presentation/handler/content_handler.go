package handler

import (
	"net/http"

	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/iservice"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type ContentHandler struct {
	contentService iservice.ContentService
}

func NewContentHandler(contentService iservice.ContentService) *ContentHandler {
	return &ContentHandler{
		contentService: contentService,
	}
}

// Create creates new content draft
// @Summary      Create content draft
// @Description  Creates new blog post or video content with DRAFT status
// @Tags         Content
// @Accept       json
// @Produce      json
// @Param        request body requests.CreateContentRequest true "Content creation data"
// @Success      201 {object} responses.APIResponse{data=responses.ContentResponse}
// @Failure      400 {object} responses.APIResponse "Validation error or invalid request"
// @Failure      404 {object} responses.APIResponse "Task not found"
// @Failure      500 {object} responses.APIResponse "Internal server error"
// @Security     BearerAuth
// @Router       /api/v1/contents [post]
func (h *ContentHandler) Create(c *gin.Context) {
	var req requests.CreateContentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		zap.L().Error("Failed to bind request", zap.Error(err))
		response := responses.ErrorResponse("Invalid request format", http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	content, err := h.contentService.Create(c.Request.Context(), &req)
	if err != nil {
		zap.L().Error("Failed to create content", zap.Error(err))

		// Determine appropriate status code based on error message
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

	statusCode := http.StatusCreated
	response := responses.SuccessResponse("Content created successfully", &statusCode, content)
	c.JSON(http.StatusCreated, response)
}

// Update updates existing content draft
// @Summary      Update content draft
// @Description  Updates content in DRAFT or REJECTED status
// @Tags         Content
// @Accept       json
// @Produce      json
// @Param        id path string true "Content ID (UUID)"
// @Param        request body requests.UpdateContentRequest true "Content update data"
// @Success      200 {object} responses.APIResponse{data=responses.ContentResponse}
// @Failure      400 {object} responses.APIResponse "Validation error or invalid request"
// @Failure      404 {object} responses.APIResponse "Content not found"
// @Failure      409 {object} responses.APIResponse "Content status not editable"
// @Failure      500 {object} responses.APIResponse "Internal server error"
// @Security     BearerAuth
// @Router       /api/v1/contents/{id} [put]
func (h *ContentHandler) Update(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		response := responses.ErrorResponse("Invalid content ID", http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	var req requests.UpdateContentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		zap.L().Error("Failed to bind request", zap.Error(err))
		response := responses.ErrorResponse("Invalid request format", http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	content, err := h.contentService.Update(c.Request.Context(), id, &req)
	if err != nil {
		zap.L().Error("Failed to update content", zap.String("id", id.String()), zap.Error(err))

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
// @Summary      Get content by ID
// @Description  Retrieves content with all relationships (blog, author, channels)
// @Tags         Content
// @Accept       json
// @Produce      json
// @Param        id path string true "Content ID (UUID)"
// @Success      200 {object} responses.APIResponse{data=responses.ContentResponse}
// @Failure      400 {object} responses.APIResponse "Invalid content ID"
// @Failure      404 {object} responses.APIResponse "Content not found"
// @Failure      500 {object} responses.APIResponse "Internal server error"
// @Security     BearerAuth
// @Router       /api/v1/contents/{id} [get]
func (h *ContentHandler) GetByID(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		response := responses.ErrorResponse("Invalid content ID", http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	content, err := h.contentService.GetByID(c.Request.Context(), id)
	if err != nil {
		zap.L().Error("Failed to get content", zap.String("id", id.String()), zap.Error(err))

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
// @Summary      Delete content draft
// @Description  Soft deletes content in DRAFT or REJECTED status
// @Tags         Content
// @Accept       json
// @Produce      json
// @Param        id path string true "Content ID (UUID)"
// @Success      200 {object} responses.APIResponse "Content deleted successfully"
// @Failure      400 {object} responses.APIResponse "Invalid content ID"
// @Failure      404 {object} responses.APIResponse "Content not found"
// @Failure      409 {object} responses.APIResponse "Content status not deletable"
// @Failure      500 {object} responses.APIResponse "Internal server error"
// @Security     BearerAuth
// @Router       /api/v1/contents/{id} [delete]
func (h *ContentHandler) Delete(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		response := responses.ErrorResponse("Invalid content ID", http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	err = h.contentService.Delete(c.Request.Context(), id)
	if err != nil {
		zap.L().Error("Failed to delete content", zap.String("id", id.String()), zap.Error(err))

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

// Submit submits content for review
// @Summary      Submit content for review
// @Description  Submits content for staff or brand approval based on selected channels
// @Tags         Content
// @Accept       json
// @Produce      json
// @Param        id path string true "Content ID (UUID)"
// @Param        request body requests.SubmitContentRequest true "Submit request data"
// @Success      200 {object} responses.APIResponse{data=responses.ContentResponse}
// @Failure      400 {object} responses.APIResponse "Invalid ID or validation error"
// @Failure      404 {object} responses.APIResponse "Content not found"
// @Failure      409 {object} responses.APIResponse "Invalid content status"
// @Failure      500 {object} responses.APIResponse "Internal server error"
// @Security     BearerAuth
// @Router       /api/v1/contents/{id}/submit [post]
func (h *ContentHandler) Submit(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		response := responses.ErrorResponse("Invalid content ID", http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	var req requests.SubmitContentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		zap.L().Error("Failed to bind request", zap.Error(err))
		response := responses.ErrorResponse("Invalid request format", http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	// Extract user_id from context (set by auth middleware)
	userIDValue, exists := c.Get("user_id")
	if !exists {
		response := responses.ErrorResponse("User ID not found in context", http.StatusUnauthorized)
		c.JSON(http.StatusUnauthorized, response)
		return
	}

	userID, ok := userIDValue.(uuid.UUID)
	if !ok {
		response := responses.ErrorResponse("Invalid user ID format", http.StatusUnauthorized)
		c.JSON(http.StatusUnauthorized, response)
		return
	}

	// Call service to submit content
	err = h.contentService.Submit(c.Request.Context(), id, userID)
	if err != nil {
		zap.L().Error("Failed to submit content",
			zap.String("id", id.String()),
			zap.String("user_id", userID.String()),
			zap.Error(err))

		statusCode := http.StatusInternalServerError
		message := "Failed to submit content"

		switch err.Error() {
		case "content not found":
			statusCode = http.StatusNotFound
			message = err.Error()
		case "only DRAFT or REJECTED content can be submitted":
			statusCode = http.StatusConflict
			message = err.Error()
		case "title and body are required fields":
			statusCode = http.StatusBadRequest
			message = err.Error()
		case "affiliate link is required for AFFILIATE contract content":
			statusCode = http.StatusBadRequest
			message = err.Error()
		default:
			statusCode = http.StatusInternalServerError
			message = err.Error()
		}

		response := responses.ErrorResponse(message, statusCode)
		c.JSON(statusCode, response)
		return
	}

	// Get updated content to return in response
	content, err := h.contentService.GetByID(c.Request.Context(), id)
	if err != nil {
		zap.L().Error("Failed to retrieve submitted content", zap.String("id", id.String()), zap.Error(err))
		response := responses.ErrorResponse("Content submitted but failed to retrieve updated data", http.StatusInternalServerError)
		c.JSON(http.StatusInternalServerError, response)
		return
	}

	response := responses.SuccessResponse("Content submitted successfully", nil, content)
	c.JSON(http.StatusOK, response)
}

// Approve approves submitted content
// @Summary      Approve content
// @Description  Approves content that is awaiting review
// @Tags         Content
// @Accept       json
// @Produce      json
// @Param        id path string true "Content ID (UUID)"
// @Param        request body requests.ApproveContentRequest true "Approval request data"
// @Success      200 {object} responses.APIResponse{data=responses.ContentResponse}
// @Failure      400 {object} responses.APIResponse "Invalid ID or validation error"
// @Failure      403 {object} responses.APIResponse "Forbidden - insufficient permissions"
// @Failure      404 {object} responses.APIResponse "Content not found"
// @Failure      409 {object} responses.APIResponse "Invalid content status"
// @Failure      500 {object} responses.APIResponse "Internal server error"
// @Security     BearerAuth
// @Router       /api/v1/contents/{id}/approve [post]
func (h *ContentHandler) Approve(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		response := responses.ErrorResponse("Invalid content ID", http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	var req requests.ApproveContentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		zap.L().Error("Failed to bind request", zap.Error(err))
		response := responses.ErrorResponse("Invalid request format", http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	// Extract user_id from context (set by auth middleware)
	userIDValue, exists := c.Get("user_id")
	if !exists {
		response := responses.ErrorResponse("User ID not found in context", http.StatusUnauthorized)
		c.JSON(http.StatusUnauthorized, response)
		return
	}

	userID, ok := userIDValue.(uuid.UUID)
	if !ok {
		response := responses.ErrorResponse("Invalid user ID format", http.StatusUnauthorized)
		c.JSON(http.StatusUnauthorized, response)
		return
	}

	// Extract comment message
	comment := ""
	if req.Message != nil {
		comment = *req.Message
	}

	// Call service to approve content
	err = h.contentService.Approve(c.Request.Context(), id, userID, comment)
	if err != nil {
		zap.L().Error("Failed to approve content",
			zap.String("id", id.String()),
			zap.String("user_id", userID.String()),
			zap.Error(err))

		statusCode := http.StatusInternalServerError
		message := "Failed to approve content"

		switch err.Error() {
		case "content not found":
			statusCode = http.StatusNotFound
			message = err.Error()
		case "only content awaiting review can be approved":
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

	// Get updated content to return in response
	content, err := h.contentService.GetByID(c.Request.Context(), id)
	if err != nil {
		zap.L().Error("Failed to retrieve approved content", zap.String("id", id.String()), zap.Error(err))
		response := responses.ErrorResponse("Content approved but failed to retrieve updated data", http.StatusInternalServerError)
		c.JSON(http.StatusInternalServerError, response)
		return
	}

	response := responses.SuccessResponse("Content approved successfully", nil, content)
	c.JSON(http.StatusOK, response)
}

// Reject rejects submitted content with feedback
// @Summary      Reject content
// @Description  Rejects content that is awaiting review with feedback
// @Tags         Content
// @Accept       json
// @Produce      json
// @Param        id path string true "Content ID (UUID)"
// @Param        request body requests.RejectContentRequest true "Rejection request data"
// @Success      200 {object} responses.APIResponse{data=responses.ContentResponse}
// @Failure      400 {object} responses.APIResponse "Invalid ID or validation error"
// @Failure      403 {object} responses.APIResponse "Forbidden - insufficient permissions"
// @Failure      404 {object} responses.APIResponse "Content not found"
// @Failure      409 {object} responses.APIResponse "Invalid content status"
// @Failure      500 {object} responses.APIResponse "Internal server error"
// @Security     BearerAuth
// @Router       /api/v1/contents/{id}/reject [post]
func (h *ContentHandler) Reject(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		response := responses.ErrorResponse("Invalid content ID", http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	var req requests.RejectContentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		zap.L().Error("Failed to bind request", zap.Error(err))
		response := responses.ErrorResponse("Invalid request format", http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	// Extract user_id from context (set by auth middleware)
	userIDValue, exists := c.Get("user_id")
	if !exists {
		response := responses.ErrorResponse("User ID not found in context", http.StatusUnauthorized)
		c.JSON(http.StatusUnauthorized, response)
		return
	}

	userID, ok := userIDValue.(uuid.UUID)
	if !ok {
		response := responses.ErrorResponse("Invalid user ID format", http.StatusUnauthorized)
		c.JSON(http.StatusUnauthorized, response)
		return
	}

	// Call service to reject content
	err = h.contentService.Reject(c.Request.Context(), id, userID, req.Feedback)
	if err != nil {
		zap.L().Error("Failed to reject content",
			zap.String("id", id.String()),
			zap.String("user_id", userID.String()),
			zap.Error(err))

		statusCode := http.StatusInternalServerError
		message := "Failed to reject content"

		switch err.Error() {
		case "content not found":
			statusCode = http.StatusNotFound
			message = err.Error()
		case "only content awaiting review can be rejected":
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

	// Get updated content to return in response
	content, err := h.contentService.GetByID(c.Request.Context(), id)
	if err != nil {
		zap.L().Error("Failed to retrieve rejected content", zap.String("id", id.String()), zap.Error(err))
		response := responses.ErrorResponse("Content rejected but failed to retrieve updated data", http.StatusInternalServerError)
		c.JSON(http.StatusInternalServerError, response)
		return
	}

	response := responses.SuccessResponse("Content rejected successfully", nil, content)
	c.JSON(http.StatusOK, response)
}

// Publish godoc
// @Summary      Publish approved content
// @Description  Publishes approved content to POSTED status with optional publish date
// @Tags         Content
// @Accept       json
// @Produce      json
// @Param        id path string true "Content ID (UUID)"
// @Param        request body requests.PublishContentRequest false "Publish request with optional publish_date"
// @Success      200 {object} responses.APIResponse{data=responses.ContentResponse} "Content published successfully"
// @Failure      400 {object} responses.APIResponse "Invalid request or content not approved"
// @Failure      401 {object} responses.APIResponse "Authentication required"
// @Failure      403 {object} responses.APIResponse "Insufficient permissions"
// @Failure      404 {object} responses.APIResponse "Content not found"
// @Failure      500 {object} responses.APIResponse "Internal server error"
// @Security     BearerAuth
// @Router       /api/v1/contents/{id}/publish [post]
func (h *ContentHandler) Publish(c *gin.Context) {
	// Parse content ID from URL parameter
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response := responses.ErrorResponse("Invalid content ID format", http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	// Bind request
	var req requests.PublishContentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response := responses.ErrorResponse("Invalid request body: "+err.Error(), http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	// Extract user ID from context
	userID, exists := c.Get("user_id")
	if !exists {
		response := responses.ErrorResponse("User context not found", http.StatusUnauthorized)
		c.JSON(http.StatusUnauthorized, response)
		return
	}

	publisherID, ok := userID.(uuid.UUID)
	if !ok {
		response := responses.ErrorResponse("Invalid user ID format", http.StatusUnauthorized)
		c.JSON(http.StatusUnauthorized, response)
		return
	}

	// Call service to publish content
	if err := h.contentService.Publish(c.Request.Context(), id, publisherID, req.PublishDate); err != nil {
		zap.L().Error("Failed to publish content", zap.String("id", id.String()), zap.Error(err))

		errMsg := err.Error()
		if errMsg == "content not found" {
			response := responses.ErrorResponse("Content not found", http.StatusNotFound)
			c.JSON(http.StatusNotFound, response)
			return
		}
		if errMsg == "only approved content can be published" {
			response := responses.ErrorResponse("Only approved content can be published", http.StatusConflict)
			c.JSON(http.StatusConflict, response)
			return
		}
		if errMsg == "invalid publish_date format, use ISO8601 format (e.g., 2006-01-02T15:04:05Z07:00)" {
			response := responses.ErrorResponse(errMsg, http.StatusBadRequest)
			c.JSON(http.StatusBadRequest, response)
			return
		}

		response := responses.ErrorResponse("Failed to publish content", http.StatusInternalServerError)
		c.JSON(http.StatusInternalServerError, response)
		return
	}

	// Retrieve updated content
	content, err := h.contentService.GetByID(c.Request.Context(), id)
	if err != nil {
		zap.L().Error("Failed to retrieve published content", zap.String("id", id.String()), zap.Error(err))
		response := responses.ErrorResponse("Content published but failed to retrieve updated data", http.StatusInternalServerError)
		c.JSON(http.StatusInternalServerError, response)
		return
	}

	response := responses.SuccessResponse("Content published successfully", nil, content)
	c.JSON(http.StatusOK, response)
}

// List retrieves paginated content with filters
// @Summary      List content
// @Description  Retrieves paginated content with optional filters, search, and sorting
// @Tags         Content
// @Accept       json
// @Produce      json
// @Param        page query int false "Page number (default: 1)" default(1)
// @Param        limit query int false "Items per page (default: 10, max: 100)" default(10)
// @Param        status query string false "Filter by status" Enums(DRAFT, AWAIT_STAFF, AWAIT_BRAND, REJECTED, APPROVED, POSTED)
// @Param        type query string false "Filter by type" Enums(POST, VIDEO)
// @Param        task_id query string false "Filter by task ID (UUID)"
// @Param        channel_id query string false "Filter by channel ID (UUID)"
// @Param        search query string false "Search in title and body"
// @Param        from_date query string false "Filter from date (YYYY-MM-DD)"
// @Param        to_date query string false "Filter to date (YYYY-MM-DD)"
// @Param        sort query string false "Sort order" Enums(created_at_asc, created_at_desc, updated_at_desc, title_asc) default(created_at_desc)
// @Success      200 {object} responses.ContentPaginationResponse
// @Failure      400 {object} responses.APIResponse "Invalid request parameters"
// @Failure      401 {object} responses.APIResponse "Authentication required"
// @Failure      500 {object} responses.APIResponse "Internal server error"
// @Security     BearerAuth
// @Router       /api/v1/contents [get]
func (h *ContentHandler) List(c *gin.Context) {
	var req requests.ContentListRequest

	// Bind query parameters
	if err := c.ShouldBindQuery(&req); err != nil {
		zap.L().Error("Failed to bind query parameters", zap.Error(err))
		response := responses.ErrorResponse("Invalid query parameters: "+err.Error(), http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	// Call service to list content
	contents, total, err := h.contentService.List(c.Request.Context(), &req)
	if err != nil {
		zap.L().Error("Failed to list contents", zap.Error(err))
		response := responses.ErrorResponse("Failed to retrieve content list", http.StatusInternalServerError)
		c.JSON(http.StatusInternalServerError, response)
		return
	}

	// Calculate pagination
	page := 1
	if req.Page > 0 {
		page = req.Page
	}

	limit := 10
	if req.Limit > 0 {
		limit = req.Limit
	}
	if limit > 100 {
		limit = 100
	}

	totalPages := int(total) / limit
	if int(total)%limit > 0 {
		totalPages++
	}

	hasNext := page < totalPages
	hasPrev := page > 1

	pagination := responses.Pagination{
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: totalPages,
		HasNext:    hasNext,
		HasPrev:    hasPrev,
	}

	response := &responses.PaginationResponse[*responses.ContentResponse]{
		Success:    true,
		Status:     http.StatusText(http.StatusOK),
		StatusCode: http.StatusOK,
		Message:    "Content list retrieved successfully",
		Data:       contents,
		Pagination: pagination,
	}
	c.JSON(http.StatusOK, response)
}
