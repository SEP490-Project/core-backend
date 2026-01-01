package handler

import (
	"net/http"
	"strconv"

	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/iservice"

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
//	@Success		201		{object}	responses.APIResponse{data=responses.ScheduleResponse}
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
//	@Success		201			{object}	responses.APIResponse{data=responses.BatchScheduleResponse}
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

	statusCode := http.StatusCreated
	c.JSON(http.StatusCreated, responses.SuccessResponse("Batch scheduling completed", &statusCode, result))
}

// GetSchedule godoc
//
//	@Summary		Get schedule details
//	@Description	Returns details of a specific schedule
//	@Tags			Content Scheduling
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string	true	"Schedule ID"
//	@Success		200	{object}	responses.APIResponse{data=responses.ScheduleItemResponse}
//	@Failure		400	{object}	responses.APIResponse
//	@Failure		404	{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/schedules/contents/{id} [get]
func (h *ContentScheduleHandler) GetSchedule(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid schedule ID", http.StatusBadRequest))
		return
	}

	schedule, err := h.scheduleService.GetSchedule(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, responses.ErrorResponse("Schedule not found: "+err.Error(), http.StatusNotFound))
		return
	}

	c.JSON(http.StatusOK, responses.SuccessResponse("Schedule retrieved", nil, schedule))
}

// ListSchedules godoc
//
//	@Summary		List content schedules
//	@Description	Returns list of content schedules with filtering
//	@Tags			Content Scheduling
//	@Accept			json
//	@Produce		json
//	@Param			reference_id	query		string	false	"Filter by reference ID (e.g. content channel ID)"
//	@Param			status			query		string	false	"Filter by status (PENDING, PROCESSING, COMPLETED, FAILED, CANCELLED)"
//	@Param			from			query		string	false	"Filter schedules from this date (RFC3339)"
//	@Param			to				query		string	false	"Filter schedules until this date (RFC3339)"
//	@Param			page			query		int		false	"Page number"		default(1)
//	@Param			limit			query		int		false	"Items per page"	default(20)
//	@Success		200				{object}	responses.APIResponse{data=responses.ScheduleListResponse}
//	@Failure		400				{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/schedules/contents [get]
func (h *ContentScheduleHandler) ListSchedules(c *gin.Context) {
	var filter requests.ScheduleFilterRequest
	if err := c.ShouldBindQuery(&filter); err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid filter parameters: "+err.Error(), http.StatusBadRequest))
		return
	}

	result, err := h.scheduleService.ListSchedules(c.Request.Context(), &filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse("Failed to list schedules: "+err.Error(), http.StatusInternalServerError))
		return
	}

	c.JSON(http.StatusOK, responses.SuccessResponse("Schedules retrieved", nil, result))
}

// CancelSchedule godoc
//
//	@Summary		Cancel a scheduled content
//	@Description	Cancels a pending schedule
//	@Tags			Content Scheduling
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string	true	"Schedule ID"
//	@Success		200	{object}	responses.APIResponse
//	@Failure		400	{object}	responses.APIResponse
//	@Failure		404	{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/schedules/contents/{id}/cancel [post]
func (h *ContentScheduleHandler) CancelSchedule(c *gin.Context) {
	id, err := extractParamID(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid schedule ID", http.StatusBadRequest))
		return
	}

	if err := h.scheduleService.CancelSchedule(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Failed to cancel schedule: "+err.Error(), http.StatusBadRequest))
		return
	}

	c.JSON(http.StatusOK, responses.SuccessResponse("Schedule cancelled successfully", nil, nil))
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
//	@Success		200		{object}	responses.APIResponse{data=responses.ScheduleResponse}
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

// GetUpcomingSchedules godoc
//
//	@Summary		Get upcoming scheduled content
//	@Description	Returns content scheduled for the next N days
//	@Tags			Content Scheduling
//	@Accept			json
//	@Produce		json
//	@Param			days	query		int	false	"Number of days to look ahead"	default(7)
//	@Success		200		{object}	responses.APIResponse{data=[]responses.ScheduledContentItem}
//	@Failure		400		{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/schedules/contents/upcoming [get]
func (h *ContentScheduleHandler) GetUpcomingSchedules(c *gin.Context) {
	days := 7 // default
	if daysQuery := c.Query("days"); daysQuery != "" {
		if d, err := strconv.Atoi(daysQuery); err == nil && d > 0 {
			days = d
		}
	}

	schedules, err := h.scheduleService.GetUpcomingSchedules(c.Request.Context(), days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse("Failed to get upcoming schedules: "+err.Error(), http.StatusInternalServerError))
		return
	}

	c.JSON(http.StatusOK, responses.SuccessResponse("Upcoming schedules retrieved", nil, schedules))
}
