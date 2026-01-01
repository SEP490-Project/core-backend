package handler

import (
	"net/http"
	"slices"
	"strconv"

	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/iservice"
	"core-backend/internal/domain/enum"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ScheduleHandler handles generic schedule endpoints
type ScheduleHandler struct {
	scheduleService iservice.ScheduleService
}

// NewScheduleHandler creates a new ScheduleHandler
func NewScheduleHandler(scheduleService iservice.ScheduleService) *ScheduleHandler {
	return &ScheduleHandler{
		scheduleService: scheduleService,
	}
}

// ListSchedules godoc
//
//	@Summary		List all schedules
//	@Description	Returns list of schedules with filtering and pagination. Staff can only see their own schedules.
//	@Tags			Schedules
//	@Accept			json
//	@Produce		json
//	@Param			status			query		string	false	"Filter by status (PENDING, PROCESSING, COMPLETED, FAILED, CANCELLED)"
//	@Param			reference_type	query		string	false	"Filter by schedule type (CONTENT_PUBLISH, CONTRACT_NOTIFICATION, OTHER)"
//	@Param			from_date		query		string	false	"Filter schedules from this date (YYYY-MM-DD)"
//	@Param			to_date			query		string	false	"Filter schedules until this date (YYYY-MM-DD)"
//	@Param			page			query		int		false	"Page number"		default(1)
//	@Param			limit			query		int		false	"Items per page"	default(20)
//	@Success		200				{object}	responses.SchedulePaginationResponse
//	@Failure		400				{object}	responses.APIResponse
//	@Failure		401				{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/schedules [get]
func (h *ScheduleHandler) ListSchedules(c *gin.Context) {
	var filter requests.ScheduleFilterRequest
	if err := c.ShouldBindQuery(&filter); err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid filter parameters: "+err.Error(), http.StatusBadRequest))
		return
	}

	// Role-based filtering: staff can only see their own schedules
	roles := c.GetStringSlice("roles")
	userIDStr, _ := c.Get("user_id")
	if userIDStr != nil {
		userID, err := uuid.Parse(userIDStr.(string))
		if err == nil {
			// If not admin or marketing manager, filter by created_by
			if !slices.Contains(roles, "ADMIN") && !slices.Contains(roles, "MARKETING_MANAGER") {
				filter.CreatedBy = &userID
			}
		}
	}

	schedules, total, err := h.scheduleService.List(c.Request.Context(), &filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse("Failed to list schedules: "+err.Error(), http.StatusInternalServerError))
		return
	}

	c.JSON(http.StatusOK, responses.NewPaginationResponse(
		"Schedules retrieved successfully",
		http.StatusOK,
		schedules,
		responses.Pagination{
			Page:  filter.Page,
			Limit: filter.Limit,
			Total: total,
		},
	))
}

// GetSchedule godoc
//
//	@Summary		Get schedule details
//	@Description	Returns details of a specific schedule with type-specific information
//	@Tags			Schedules
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string	true	"Schedule ID"
//	@Success		200	{object}	responses.APIResponse{data=dtos.ScheduleDTO}
//	@Failure		400	{object}	responses.APIResponse
//	@Failure		404	{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/schedules/{id} [get]
func (h *ScheduleHandler) GetSchedule(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid schedule ID", http.StatusBadRequest))
		return
	}

	schedule, err := h.scheduleService.GetByIDWithDetails(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, responses.ErrorResponse("Schedule not found: "+err.Error(), http.StatusNotFound))
		return
	}

	c.JSON(http.StatusOK, responses.SuccessResponse("Schedule retrieved", nil, schedule))
}

// CancelSchedule godoc
//
//	@Summary		Cancel a schedule
//	@Description	Cancels a pending schedule
//	@Tags			Schedules
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string	true	"Schedule ID"
//	@Success		200	{object}	responses.APIResponse
//	@Failure		400	{object}	responses.APIResponse
//	@Failure		404	{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/schedules/{id} [delete]
func (h *ScheduleHandler) CancelSchedule(c *gin.Context) {
	id, err := extractParamID(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid schedule ID", http.StatusBadRequest))
		return
	}

	userID, err := extractUserID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Failed to extract user ID: "+err.Error(), http.StatusBadRequest))
		return
	}

	if err := h.scheduleService.Cancel(c.Request.Context(), id, userID); err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Failed to cancel schedule: "+err.Error(), http.StatusBadRequest))
		return
	}

	c.JSON(http.StatusOK, responses.SuccessResponse("Schedule cancelled successfully", nil, nil))
}

// GetUpcomingSchedules godoc
//
//	@Summary		Get upcoming schedules
//	@Description	Returns schedules for the next N days
//	@Tags			Schedules
//	@Accept			json
//	@Produce		json
//	@Param			days	query		int		false	"Number of days to look ahead"	default(7)
//	@Param			type	query		string	false	"Filter by schedule type (CONTENT_PUBLISH, CONTRACT_NOTIFICATION, OTHER)"
//	@Param			limit	query		int		false	"Maximum number of results"	default(50)
//	@Success		200		{object}	responses.APIResponse{data=[]dtos.ScheduleDTO}
//	@Failure		400		{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/schedules/upcoming [get]
func (h *ScheduleHandler) GetUpcomingSchedules(c *gin.Context) {
	days := 7
	if daysQuery := c.Query("days"); daysQuery != "" {
		if d, err := strconv.Atoi(daysQuery); err == nil && d > 0 {
			days = d
		}
	}

	limit := 50
	if limitQuery := c.Query("limit"); limitQuery != "" {
		if l, err := strconv.Atoi(limitQuery); err == nil && l > 0 {
			limit = l
		}
	}

	var scheduleType *enum.ScheduleType
	if typeQuery := c.Query("type"); typeQuery != "" {
		st := enum.ScheduleType(typeQuery)
		if st.IsValid() {
			scheduleType = &st
		}
	}

	schedules, err := h.scheduleService.GetUpcoming(c.Request.Context(), days, scheduleType, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse("Failed to get upcoming schedules: "+err.Error(), http.StatusInternalServerError))
		return
	}

	c.JSON(http.StatusOK, responses.SuccessResponse("Upcoming schedules retrieved", nil, schedules))
}
