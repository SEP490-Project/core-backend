package handler

import (
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/iservice"
	"net/http"

	"github.com/gin-gonic/gin"
)

// ContentStaffAnalyticsHandler handles Content Staff analytics endpoints
type ContentStaffAnalyticsHandler struct {
	service iservice.ContentStaffAnalyticsService
}

// NewContentStaffAnalyticsHandler creates a new Content Staff analytics handler
func NewContentStaffAnalyticsHandler(service iservice.ContentStaffAnalyticsService) *ContentStaffAnalyticsHandler {
	return &ContentStaffAnalyticsHandler{service: service}
}

// GetDashboard returns the complete Content Staff dashboard
// @Summary Get Content Staff Dashboard
// @Description Returns comprehensive content dashboard with overview metrics, content status breakdown, platform metrics, top content, top channels, recent content, and engagement trend
// @Tags Content Staff Analytics
// @Accept json
// @Produce json
// @Param year query int false "Year (defaults to current year)" minimum(2000) maximum(2100) example(2025)
// @Param month query int false "Month (defaults to current month)" minimum(1) maximum(12) example(11)
// @Success 200 {object} responses.APIResponse{data=responses.ContentStaffDashboardResponse}
// @Failure 400 {object} responses.APIResponse
// @Failure 401 {object} responses.APIResponse
// @Failure 403 {object} responses.APIResponse
// @Failure 500 {object} responses.APIResponse
// @Security BearerAuth
// @Router /api/v1/analytics/content/dashboard [get]
func (h *ContentStaffAnalyticsHandler) GetDashboard(c *gin.Context) {
	var req requests.ContentStaffDashboardRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid request parameters", http.StatusBadRequest))
		return
	}

	result, err := h.service.GetDashboard(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse("Failed to get dashboard", http.StatusInternalServerError))
		return
	}

	c.JSON(http.StatusOK, responses.SuccessResponse("Content Staff dashboard retrieved successfully", nil, result))
}

// GetContentStatusBreakdown returns content counts by status
// @Summary Get Content Status Breakdown
// @Description Returns content counts broken down by status (DRAFT, PENDING, APPROVED, REJECTED, POSTED)
// @Tags Content Staff Analytics
// @Accept json
// @Produce json
// @Param start_date query string false "Start date (RFC3339 format)" format(date-time)
// @Param end_date query string false "End date (RFC3339 format)" format(date-time)
// @Success 200 {object} responses.APIResponse{data=responses.ContentStatusBreakdown}
// @Failure 400 {object} responses.APIResponse
// @Failure 401 {object} responses.APIResponse
// @Failure 403 {object} responses.APIResponse
// @Failure 500 {object} responses.APIResponse
// @Security BearerAuth
// @Router /api/v1/analytics/content/status [get]
func (h *ContentStaffAnalyticsHandler) GetContentStatusBreakdown(c *gin.Context) {
	var req requests.ContentStatusRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid request parameters", http.StatusBadRequest))
		return
	}

	result, err := h.service.GetContentStatusBreakdown(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse("Failed to get content status breakdown", http.StatusInternalServerError))
		return
	}

	c.JSON(http.StatusOK, responses.SuccessResponse("Content status breakdown retrieved successfully", nil, result))
}

// GetMetricsByPlatform returns metrics aggregated by platform
// @Summary Get Metrics by Platform
// @Description Returns content and engagement metrics aggregated by platform (FACEBOOK, TIKTOK, INSTAGRAM, YOUTUBE)
// @Tags Content Staff Analytics
// @Accept json
// @Produce json
// @Param start_date query string false "Start date (RFC3339 format)" format(date-time)
// @Param end_date query string false "End date (RFC3339 format)" format(date-time)
// @Success 200 {object} responses.APIResponse{data=[]responses.PlatformMetric}
// @Failure 400 {object} responses.APIResponse
// @Failure 401 {object} responses.APIResponse
// @Failure 403 {object} responses.APIResponse
// @Failure 500 {object} responses.APIResponse
// @Security BearerAuth
// @Router /api/v1/analytics/content/platforms [get]
func (h *ContentStaffAnalyticsHandler) GetMetricsByPlatform(c *gin.Context) {
	var req requests.PlatformMetricsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid request parameters", http.StatusBadRequest))
		return
	}

	result, err := h.service.GetMetricsByPlatform(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse("Failed to get platform metrics", http.StatusInternalServerError))
		return
	}

	c.JSON(http.StatusOK, responses.SuccessResponse("Platform metrics retrieved successfully", nil, result))
}

// GetTopContent returns top content by views
// @Summary Get Top Content
// @Description Returns top performing content ranked by views
// @Tags Content Staff Analytics
// @Accept json
// @Produce json
// @Param start_date query string false "Start date (RFC3339 format)" format(date-time)
// @Param end_date query string false "End date (RFC3339 format)" format(date-time)
// @Param platform query string false "Filter by platform" Enums(FACEBOOK, TIKTOK, INSTAGRAM, YOUTUBE)
// @Param limit query int false "Number of results (default: 10, max: 50)" minimum(1) maximum(50) default(10)
// @Success 200 {object} responses.APIResponse{data=[]responses.ContentMetric}
// @Failure 400 {object} responses.APIResponse
// @Failure 401 {object} responses.APIResponse
// @Failure 403 {object} responses.APIResponse
// @Failure 500 {object} responses.APIResponse
// @Security BearerAuth
// @Router /api/v1/analytics/content/top [get]
func (h *ContentStaffAnalyticsHandler) GetTopContent(c *gin.Context) {
	var req requests.TopContentRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid request parameters", http.StatusBadRequest))
		return
	}

	result, err := h.service.GetTopContent(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse("Failed to get top content", http.StatusInternalServerError))
		return
	}

	c.JSON(http.StatusOK, responses.SuccessResponse("Top content retrieved successfully", nil, result))
}

// GetTopChannels returns top channels by engagement
// @Summary Get Top Channels
// @Description Returns top performing channels ranked by total engagement
// @Tags Content Staff Analytics
// @Accept json
// @Produce json
// @Param start_date query string false "Start date (RFC3339 format)" format(date-time)
// @Param end_date query string false "End date (RFC3339 format)" format(date-time)
// @Param platform query string false "Filter by platform" Enums(FACEBOOK, TIKTOK, INSTAGRAM, YOUTUBE)
// @Param limit query int false "Number of results (default: 10, max: 50)" minimum(1) maximum(50) default(10)
// @Success 200 {object} responses.APIResponse{data=[]responses.ChannelMetric}
// @Failure 400 {object} responses.APIResponse
// @Failure 401 {object} responses.APIResponse
// @Failure 403 {object} responses.APIResponse
// @Failure 500 {object} responses.APIResponse
// @Security BearerAuth
// @Router /api/v1/analytics/content/channels [get]
func (h *ContentStaffAnalyticsHandler) GetTopChannels(c *gin.Context) {
	var req requests.TopChannelsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid request parameters", http.StatusBadRequest))
		return
	}

	result, err := h.service.GetTopChannels(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse("Failed to get top channels", http.StatusInternalServerError))
		return
	}

	c.JSON(http.StatusOK, responses.SuccessResponse("Top channels retrieved successfully", nil, result))
}

// GetEngagementTrend returns engagement time-series data
// @Summary Get Engagement Trend
// @Description Returns engagement time-series data with configurable granularity (DAY, WEEK, MONTH)
// @Tags Content Staff Analytics
// @Accept json
// @Produce json
// @Param start_date query string false "Start date (RFC3339 format)" format(date-time)
// @Param end_date query string false "End date (RFC3339 format)" format(date-time)
// @Param granularity query string false "Time bucket granularity" Enums(DAY, WEEK, MONTH) default(DAY)
// @Success 200 {object} responses.APIResponse{data=[]responses.EngagementTrendPoint}
// @Failure 400 {object} responses.APIResponse
// @Failure 401 {object} responses.APIResponse
// @Failure 403 {object} responses.APIResponse
// @Failure 500 {object} responses.APIResponse
// @Security BearerAuth
// @Router /api/v1/analytics/content/trend [get]
func (h *ContentStaffAnalyticsHandler) GetEngagementTrend(c *gin.Context) {
	var req requests.EngagementTrendRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid request parameters", http.StatusBadRequest))
		return
	}

	result, err := h.service.GetEngagementTrend(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse("Failed to get engagement trend", http.StatusInternalServerError))
		return
	}

	c.JSON(http.StatusOK, responses.SuccessResponse("Engagement trend retrieved successfully", nil, result))
}

// GetCampaignContentMetrics returns content metrics by campaign
// @Summary Get Campaign Content Metrics
// @Description Returns content metrics aggregated by campaign
// @Tags Content Staff Analytics
// @Accept json
// @Produce json
// @Param start_date query string false "Start date (RFC3339 format)" format(date-time)
// @Param end_date query string false "End date (RFC3339 format)" format(date-time)
// @Param campaign_id query string false "Filter by specific campaign ID" format(uuid)
// @Param limit query int false "Number of results (default: 10, max: 50)" minimum(1) maximum(50) default(10)
// @Success 200 {object} responses.APIResponse{data=[]responses.CampaignContentMetric}
// @Failure 400 {object} responses.APIResponse
// @Failure 401 {object} responses.APIResponse
// @Failure 403 {object} responses.APIResponse
// @Failure 500 {object} responses.APIResponse
// @Security BearerAuth
// @Router /api/v1/analytics/content/campaigns [get]
func (h *ContentStaffAnalyticsHandler) GetCampaignContentMetrics(c *gin.Context) {
	var req requests.CampaignContentRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid request parameters", http.StatusBadRequest))
		return
	}

	result, err := h.service.GetCampaignContentMetrics(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse("Failed to get campaign content metrics", http.StatusInternalServerError))
		return
	}

	c.JSON(http.StatusOK, responses.SuccessResponse("Campaign content metrics retrieved successfully", nil, result))
}
