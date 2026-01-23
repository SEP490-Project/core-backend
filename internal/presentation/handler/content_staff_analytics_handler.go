package handler

import (
	"net/http"

	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/iservice"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ContentStaffAnalyticsHandler handles content dashboard endpoints
type ContentStaffAnalyticsHandler struct {
	dashboardService iservice.ContentStaffAnalyticsService
}

// NewContentStaffAnalyticsHandler creates a new ContentStaffAnalyticsHandler
func NewContentStaffAnalyticsHandler(dashboardService iservice.ContentStaffAnalyticsService) *ContentStaffAnalyticsHandler {
	return &ContentStaffAnalyticsHandler{
		dashboardService: dashboardService,
	}
}

// GetDashboard godoc
//
//	@Summary		Get content staff dashboard
//	@Description	Returns complete dashboard with metrics, charts, schedules, and alerts
//	@Tags			Analytics.Content
//	@Accept			json
//	@Produce		json
//	@Param			period		query		string	false	"Period preset (TODAY, YESTERDAY, THIS_WEEK, LAST_WEEK, THIS_MONTH, LAST_MONTH, THIS_QUARTER, THIS_YEAR, LAST_7_DAYS, LAST_30_DAYS, CUSTOM)"	default(LAST_30_DAYS)
//	@Param			start_date	query		string	false	"Custom start date (YYYY-MM-DD) - required when period=CUSTOM"
//	@Param			end_date	query		string	false	"Custom end date (YYYY-MM-DD) - required when period=CUSTOM"
//	@Param			campaign_id	query		string	false	"Filter by campaign ID"
//	@Param			brand_id	query		string	false	"Filter by brand ID"
//	@Success		200			{object}	responses.APIResponse{data=responses.ContentDashboardResponse}
//	@Failure		400			{object}	responses.APIResponse
//	@Failure		401			{object}	responses.APIResponse
//	@Failure		500			{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/analytics/contents/dashboard [get]
func (h *ContentStaffAnalyticsHandler) GetDashboard(c *gin.Context) {
	var filter requests.ContentDashboardFilterRequest
	if err := c.ShouldBindQuery(&filter); err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid filter parameters: "+err.Error(), http.StatusBadRequest))
		return
	}

	// Get user ID from context for alert acknowledgment status
	var userID uuid.UUID
	userIDStr, exists := c.Get("user_id")
	if exists {
		userID, _ = uuid.Parse(userIDStr.(string))
	}

	dashboard, err := h.dashboardService.GetDashboard(c.Request.Context(), &filter, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse("Failed to load dashboard: "+err.Error(), http.StatusInternalServerError))
		return
	}

	c.JSON(http.StatusOK, responses.SuccessResponse("Dashboard loaded", nil, dashboard))
}

// GetChannelDetails godoc
//
//	@Summary		Get channel details
//	@Description	Returns detailed metrics for a specific channel
//	@Tags			Analytics.Content
//	@Accept			json
//	@Produce		json
//	@Param			id						path		string	true	"Channel ID"
//	@Param			period					query		string	false	"Period preset"	default(THIS_MONTH)
//	@Param			from_date				query		string	false	"Custom start date (YYYY-MM-DD)"
//	@Param			to_date					query		string	false	"Custom end date (YYYY-MM-DD)"
//	@Param			trend_granularity		query		string	false	"Trend granularity (HOUR, DAY, WEEK, MONTH)"	default(DAY)
//	@Param			top_content_limit		query		int		false	"Top content limit"								default(10)
//	@Param			recent_content_limit	query		int		false	"Recent content limit"							default(10)
//	@Success		200						{object}	responses.APIResponse{data=responses.ChannelDetailsResponse}
//	@Failure		400						{object}	responses.APIResponse
//	@Failure		404						{object}	responses.APIResponse
//	@Failure		500						{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/analytics/contents/channels/{id} [get]
func (h *ContentStaffAnalyticsHandler) GetChannelDetails(c *gin.Context) {
	channelIDStr := c.Param("id")
	channelID, err := uuid.Parse(channelIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid channel ID", http.StatusBadRequest))
		return
	}

	var filter requests.ChannelDetailsRequest
	if err := c.ShouldBindQuery(&filter); err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid filter parameters: "+err.Error(), http.StatusBadRequest))
		return
	}

	details, err := h.dashboardService.GetChannelDetails(c.Request.Context(), channelID, &filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse("Failed to load channel details: "+err.Error(), http.StatusInternalServerError))
		return
	}

	c.JSON(http.StatusOK, responses.SuccessResponse("Channel details loaded", nil, details))
}
