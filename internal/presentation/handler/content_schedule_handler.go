package handler

import (
	"net/http"

	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/iservice"
	"core-backend/pkg/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ContentScheduleHandler handles content scheduling endpoints
type ContentScheduleHandler struct {
	scheduleService iservice.ContentScheduleService
}

// NewContentScheduleHandler creates a new ContentScheduleHandler
func NewContentScheduleHandler(scheduleService iservice.ContentScheduleService) *ContentScheduleHandler {
	return &ContentScheduleHandler{
		scheduleService: scheduleService,
	}
}

// ScheduleContent godoc
//
//	@Summary		Schedule content for future publishing
//	@Description	Creates a schedule for content to be published at a specific time
//	@Tags			Content Scheduling
//	@Accept			json
//	@Produce		json
//	@Param			request	body		requests.ScheduleContentRequest	true	"Schedule request"
//	@Success		201		{object}	responses.APIResponse{data=responses.ScheduleDetailResponse}
//	@Failure		400		{object}	responses.APIResponse
//	@Failure		401		{object}	responses.APIResponse
//	@Failure		500		{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/schedules/contents [post]
func (h *ContentScheduleHandler) ScheduleContent(c *gin.Context) {
	var req requests.ScheduleContentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid request: "+err.Error(), http.StatusBadRequest))
		return
	}

	// Get user ID from context
	userIDStr, exists := c.Get("user_id")
	if exists {
		if userID, err := uuid.Parse(userIDStr.(string)); err == nil {
			req.UserID = userID
		}
	}

	schedule, err := h.scheduleService.ScheduleContent(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Failed to schedule content: "+err.Error(), http.StatusBadRequest))
		return
	}

	statusCode := http.StatusCreated
	c.JSON(http.StatusCreated, responses.SuccessResponse("Content scheduled successfully", &statusCode, schedule))
}

// BatchScheduleContent godoc
//
//	@Summary		Batch schedule content to multiple channels
//	@Description	Creates schedules for content to be published to multiple channels at specific times
//	@Tags			Content Scheduling
//	@Accept			json
//	@Produce		json
//	@Param			content_id	path		string							true	"Content ID"
//	@Param			request		body		requests.BatchScheduleRequest	true	"Batch schedule request"
//	@Success		201			{object}	responses.APIResponse{data=responses.BatchContentScheduleResponse}
//	@Failure		400			{object}	responses.APIResponse
//	@Failure		401			{object}	responses.APIResponse
//	@Failure		500			{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/schedules/contents/batch [post]
func (h *ContentScheduleHandler) BatchScheduleContent(c *gin.Context) {
	var req requests.BatchScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid request: "+err.Error(), http.StatusBadRequest))
		return
	}
	// Get user ID from context
	userID, err := extractUserID(c)
	if err == nil {
		req.UserID = userID
	}

	result, err := h.scheduleService.BatchScheduleContent(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Failed to batch schedule content: "+err.Error(), http.StatusBadRequest))
		return
	}
	if result.TotalFailed > 0 { // Partial failure
		c.JSON(http.StatusBadRequest, responses.SuccessResponse("Batch scheduling completed with some failures", utils.PtrOrNil(http.StatusBadRequest), result))
		return
	}

	statusCode := http.StatusCreated
	c.JSON(http.StatusCreated, responses.SuccessResponse("Batch scheduling completed", &statusCode, result))
}

// RescheduleContent godoc
//
//	@Summary		Reschedule content
//	@Description	Updates the scheduled time for content publishing
//	@Tags			Content Scheduling
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string								true	"Schedule ID"
//	@Param			request	body		requests.RescheduleContentRequest	true	"Reschedule request"
//	@Success		200		{object}	responses.APIResponse{data=responses.ScheduleDetailResponse}
//	@Failure		400		{object}	responses.APIResponse
//	@Failure		404		{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/schedules/contents/{id}/reschedule [post]
func (h *ContentScheduleHandler) RescheduleContent(c *gin.Context) {
	id, err := extractParamID(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid schedule ID", http.StatusBadRequest))
		return
	}

	var req requests.RescheduleContentRequest
	if err = c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid request: "+err.Error(), http.StatusBadRequest))
		return
	}

	schedule, err := h.scheduleService.RescheduleContent(c.Request.Context(), id, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Failed to reschedule content: "+err.Error(), http.StatusBadRequest))
		return
	}

	c.JSON(http.StatusOK, responses.SuccessResponse("Content rescheduled successfully", nil, schedule))
}
