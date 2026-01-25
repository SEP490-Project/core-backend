package handler

import (
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/iservice"
	"net/http"

	"github.com/gin-gonic/gin"
)

type AdminAnalyticsHandler struct {
	analyticsService iservice.AdminAnalyticsService
}

func NewAdminAnalyticsHandler(analyticsService iservice.AdminAnalyticsService) *AdminAnalyticsHandler {
	return &AdminAnalyticsHandler{
		analyticsService: analyticsService,
	}
}

// GetDashboard returns the complete Admin analytics dashboard
//
//	@Summary		Get Admin Dashboard
//	@Description	Returns comprehensive platform-wide analytics dashboard for Admin including user metrics, revenue breakdown, contracts, campaigns, and growth trends
//	@Tags			Analytics.Admin
//	@Accept			json
//	@Produce		json
//	@Param			year	query		int	false	"Year for filtering (defaults to current year)"
//	@Param			month	query		int	false	"Month for filtering (defaults to current month)"
//	@Success		200		{object}	responses.APIResponse{data=responses.AdminDashboardResponse}
//	@Failure		400		{object}	responses.APIResponse
//	@Failure		401		{object}	responses.APIResponse
//	@Failure		403		{object}	responses.APIResponse
//	@Failure		500		{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/analytics/admin/dashboard [get]
func (h *AdminAnalyticsHandler) GetDashboard(c *gin.Context) {
	ctx := c.Request.Context()

	var req requests.AdminDashboardRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid query parameters: "+err.Error(), http.StatusBadRequest))
		return
	}

	dashboard, err := h.analyticsService.GetDashboard(ctx, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse("Failed to fetch dashboard: "+err.Error(), http.StatusInternalServerError))
		return
	}

	c.JSON(http.StatusOK, responses.SuccessResponse("Dashboard fetched successfully", nil, dashboard))
}

// GetUsersOverview returns user statistics and growth
//
//	@Summary		Get Users Overview
//	@Description	Returns user statistics including role breakdown and growth trends
//	@Tags			Analytics.Admin
//	@Accept			json
//	@Produce		json
//	@Param			role		query		string	false	"Filter by user role (ADMIN, MARKETING_STAFF, SALES_STAFF, CONTENT_STAFF, BRAND_PARTNER, CUSTOMER)"
//	@Param			start_date	query		string	false	"Start date (ISO 8601 format)"
//	@Param			end_date	query		string	false	"End date (ISO 8601 format)"
//	@Success		200			{object}	responses.APIResponse{data=responses.UsersOverviewResponse}
//	@Failure		400			{object}	responses.APIResponse
//	@Failure		401			{object}	responses.APIResponse
//	@Failure		403			{object}	responses.APIResponse
//	@Failure		500			{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/analytics/admin/users [get]
func (h *AdminAnalyticsHandler) GetUsersOverview(c *gin.Context) {
	ctx := c.Request.Context()

	var req requests.UsersOverviewRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid query parameters: "+err.Error(), http.StatusBadRequest))
		return
	}

	overview, err := h.analyticsService.GetUsersOverview(ctx, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse("Failed to fetch users overview: "+err.Error(), http.StatusInternalServerError))
		return
	}

	c.JSON(http.StatusOK, responses.SuccessResponse("Users overview fetched successfully", nil, overview))
}

// GetPlatformRevenue returns platform-wide revenue analytics
//
//	@Summary		Get Platform Revenue
//	@Description	Returns platform-wide revenue analytics including breakdown by source and trends
//	@Tags			Analytics.Admin
//	@Accept			json
//	@Produce		json
//	@Param			start_date	query		string	false	"Start date (ISO 8601 format)"
//	@Param			end_date	query		string	false	"End date (ISO 8601 format)"
//	@Param			granularity	query		string	false	"Time granularity (DAY, WEEK, MONTH - default: MONTH)"
//	@Success		200			{object}	responses.APIResponse{data=responses.PlatformRevenueResponse}
//	@Failure		400			{object}	responses.APIResponse
//	@Failure		401			{object}	responses.APIResponse
//	@Failure		403			{object}	responses.APIResponse
//	@Failure		500			{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/analytics/admin/revenue [get]
func (h *AdminAnalyticsHandler) GetPlatformRevenue(c *gin.Context) {
	ctx := c.Request.Context()

	var req requests.PlatformRevenueRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid query parameters: "+err.Error(), http.StatusBadRequest))
		return
	}

	revenue, err := h.analyticsService.GetPlatformRevenue(ctx, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse("Failed to fetch platform revenue: "+err.Error(), http.StatusInternalServerError))
		return
	}

	c.JSON(http.StatusOK, responses.SuccessResponse("Platform revenue fetched successfully", nil, revenue))
}

// GetSystemHealth returns system health metrics
//
//	@Summary		Get System Health
//	@Description	Returns system health metrics including database, cache, and queue status
//	@Tags			Analytics.Admin
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	responses.APIResponse{data=responses.SystemHealthResponse}
//	@Failure		401	{object}	responses.APIResponse
//	@Failure		403	{object}	responses.APIResponse
//	@Failure		500	{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/analytics/admin/health [get]
func (h *AdminAnalyticsHandler) GetSystemHealth(c *gin.Context) {
	ctx := c.Request.Context()

	health, err := h.analyticsService.GetSystemHealth(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse("Failed to fetch system health: "+err.Error(), http.StatusInternalServerError))
		return
	}

	c.JSON(http.StatusOK, responses.SuccessResponse("System health fetched successfully", nil, health))
}

// GetUserGrowth returns user growth over time
//
//	@Summary		Get User Growth
//	@Description	Returns user registration growth over time
//	@Tags			Analytics.Admin
//	@Accept			json
//	@Produce		json
//	@Param			start_date	query		string	false	"Start date (ISO 8601 format)"
//	@Param			end_date	query		string	false	"End date (ISO 8601 format)"
//	@Param			granularity	query		string	false	"Time granularity (DAY, WEEK, MONTH - default: MONTH)"
//	@Param			role		query		string	false	"Filter by user role"
//	@Success		200			{object}	responses.APIResponse{data=[]responses.UserGrowthPoint}
//	@Failure		400			{object}	responses.APIResponse
//	@Failure		401			{object}	responses.APIResponse
//	@Failure		403			{object}	responses.APIResponse
//	@Failure		500			{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/analytics/admin/user-growth [get]
func (h *AdminAnalyticsHandler) GetUserGrowth(c *gin.Context) {
	ctx := c.Request.Context()

	var req requests.UserGrowthRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid query parameters: "+err.Error(), http.StatusBadRequest))
		return
	}

	growth, err := h.analyticsService.GetUserGrowth(ctx, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse("Failed to fetch user growth: "+err.Error(), http.StatusInternalServerError))
		return
	}

	c.JSON(http.StatusOK, responses.SuccessResponse("User growth fetched successfully", nil, growth))
}

// GetContractsSummary returns contract statistics
//
//	@Summary		Get Contracts Summary
//	@Description	Returns contract statistics including status breakdown and values
//	@Tags			Analytics.Admin
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	responses.APIResponse{data=responses.ContractsSummary}
//	@Failure		401	{object}	responses.APIResponse
//	@Failure		403	{object}	responses.APIResponse
//	@Failure		500	{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/analytics/admin/contracts [get]
func (h *AdminAnalyticsHandler) GetContractsSummary(c *gin.Context) {
	ctx := c.Request.Context()

	summary, err := h.analyticsService.GetContractsSummary(ctx, &requests.DashboardRequest{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse("Failed to fetch contracts summary: "+err.Error(), http.StatusInternalServerError))
		return
	}

	c.JSON(http.StatusOK, responses.SuccessResponse("Contracts summary fetched successfully", nil, summary))
}

// GetCampaignsSummary returns campaign statistics
//
//	@Summary		Get Campaigns Summary
//	@Description	Returns campaign statistics including status breakdown and content counts
//	@Tags			Analytics.Admin
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	responses.APIResponse{data=responses.AdminCampaignsSummary}
//	@Failure		401	{object}	responses.APIResponse
//	@Failure		403	{object}	responses.APIResponse
//	@Failure		500	{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/analytics/admin/campaigns [get]
func (h *AdminAnalyticsHandler) GetCampaignsSummary(c *gin.Context) {
	ctx := c.Request.Context()

	summary, err := h.analyticsService.GetCampaignsSummary(ctx, &requests.DashboardRequest{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse("Failed to fetch campaigns summary: "+err.Error(), http.StatusInternalServerError))
		return
	}

	c.JSON(http.StatusOK, responses.SuccessResponse("Campaigns summary fetched successfully", nil, summary))
}
