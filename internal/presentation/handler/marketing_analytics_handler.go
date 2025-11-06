package handler

import (
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/iservice"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type MarketingAnalyticsHandler struct {
	marketingAnalyticsService iservice.MarketingAnalyticsService
}

// NewMarketingAnalyticsHandler creates a new marketing analytics handler
func NewMarketingAnalyticsHandler(marketingAnalyticsService iservice.MarketingAnalyticsService) *MarketingAnalyticsHandler {
	return &MarketingAnalyticsHandler{
		marketingAnalyticsService: marketingAnalyticsService,
	}
}

// GetActiveBrandsCount godoc
//
//	@Summary		Get active brands count
//	@Description	Returns the total count of brands with status = 'ACTIVE'
//	@Tags			Analytics/MarketingStaffs
//	@Produce		json
//	@Success		200	{object}	responses.APIResponse{data=int64}
//	@Failure		500	{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/analytics/marketing/active-brands [get]
func (h *MarketingAnalyticsHandler) GetActiveBrandsCount(c *gin.Context) {
	ctx := c.Request.Context()

	count, err := h.marketingAnalyticsService.GetActiveBrandsCount(ctx)
	if err != nil {
		zap.L().Error("Failed to get active brands count", zap.Error(err))
		response := responses.ErrorResponse("Failed to retrieve active brands count", http.StatusInternalServerError)
		c.JSON(http.StatusInternalServerError, response)
		return
	}

	response := responses.SuccessResponse("Active brands count retrieved successfully", nil, count)
	c.JSON(http.StatusOK, response)
}

// GetActiveCampaignsCount godoc
//
//	@Summary		Get active campaigns count
//	@Description	Returns the total count of campaigns with status = 'RUNNING'
//	@Tags			Analytics/MarketingStaffs
//	@Produce		json
//	@Success		200	{object}	responses.APIResponse{data=int64}
//	@Failure		500	{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/analytics/marketing/active-campaigns [get]
func (h *MarketingAnalyticsHandler) GetActiveCampaignsCount(c *gin.Context) {
	ctx := c.Request.Context()

	count, err := h.marketingAnalyticsService.GetActiveCampaignsCount(ctx)
	if err != nil {
		zap.L().Error("Failed to get active campaigns count", zap.Error(err))
		response := responses.ErrorResponse("Failed to retrieve active campaigns count", http.StatusInternalServerError)
		c.JSON(http.StatusInternalServerError, response)
		return
	}

	response := responses.SuccessResponse("Active campaigns count retrieved successfully", nil, count)
	c.JSON(http.StatusOK, response)
}

// GetDraftCampaignsCount godoc
//
//	@Summary		Get draft campaigns count
//	@Description	Returns the count of campaigns with status = 'DRAFT' and contract_id IS NOT NULL
//	@Tags			Analytics/MarketingStaffs
//	@Produce		json
//	@Success		200	{object}	responses.APIResponse{data=int64}
//	@Failure		500	{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/analytics/marketing/draft-campaigns [get]
func (h *MarketingAnalyticsHandler) GetDraftCampaignsCount(c *gin.Context) {
	ctx := c.Request.Context()

	count, err := h.marketingAnalyticsService.GetDraftCampaignsCount(ctx)
	if err != nil {
		zap.L().Error("Failed to get draft campaigns count", zap.Error(err))
		response := responses.ErrorResponse("Failed to retrieve draft campaigns count", http.StatusInternalServerError)
		c.JSON(http.StatusInternalServerError, response)
		return
	}

	response := responses.SuccessResponse("Draft campaigns count retrieved successfully", nil, count)
	c.JSON(http.StatusOK, response)
}

// GetMonthlyContractRevenue godoc
//
//	@Summary		Get monthly contract revenue
//	@Description	Returns total revenue from paid contract payments for specified month
//	@Tags			Analytics/MarketingStaffs
//	@Accept			json
//	@Produce		json
//	@Param			year	query		int	true	"Year (e.g., 2024)"	minimum(2000)	maximum(2100)
//	@Param			month	query		int	true	"Month (1-12)"		minimum(1)		maximum(12)
//	@Success		200		{object}	responses.APIResponse{data=float64}
//	@Failure		400		{object}	responses.APIResponse
//	@Failure		500		{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/analytics/marketing/monthly-revenue [get]
func (h *MarketingAnalyticsHandler) GetMonthlyContractRevenue(c *gin.Context) {
	ctx := c.Request.Context()

	var req requests.MonthlyRevenueRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		zap.L().Error("Invalid request parameters", zap.Error(err))
		response := responses.ErrorResponse("Invalid request parameters", http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	revenue, err := h.marketingAnalyticsService.GetMonthlyContractRevenue(ctx, &req)
	if err != nil {
		zap.L().Error("Failed to get monthly contract revenue",
			zap.Int("year", req.Year),
			zap.Int("month", req.Month),
			zap.Error(err))
		response := responses.ErrorResponse("Failed to retrieve monthly contract revenue", http.StatusInternalServerError)
		c.JSON(http.StatusInternalServerError, response)
		return
	}

	response := responses.SuccessResponse("Monthly contract revenue retrieved successfully", nil, revenue)
	c.JSON(http.StatusOK, response)
}

// GetTopBrandsByRevenue godoc
//
//	@Summary		Get top brands by revenue
//	@Description	Returns top 4 brands by total revenue (contract payments + standard product sales)
//	@Tags			Analytics/MarketingStaffs
//	@Accept			json
//	@Produce		json
//	@Param			filter_type	query		string	true	"Filter type"							Enums(MONTH, QUARTER, YEAR)
//	@Param			year		query		int		true	"Year (e.g., 2024)"						minimum(2000)	maximum(2100)
//	@Param			month		query		int		false	"Month (required for MONTH filter)"		minimum(1)		maximum(12)
//	@Param			quarter		query		int		false	"Quarter (required for QUARTER filter)"	minimum(1)		maximum(4)
//	@Success		200			{object}	responses.APIResponse{data=[]responses.BrandRevenueResponse}
//	@Failure		400			{object}	responses.APIResponse
//	@Failure		500			{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/analytics/marketing/top-brands [get]
func (h *MarketingAnalyticsHandler) GetTopBrandsByRevenue(c *gin.Context) {
	ctx := c.Request.Context()

	var filter requests.TimeFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		zap.L().Error("Invalid request parameters", zap.Error(err))
		response := responses.ErrorResponse("Invalid request parameters", http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	brands, err := h.marketingAnalyticsService.GetTopBrandsByRevenue(ctx, &filter)
	if err != nil {
		zap.L().Error("Failed to get top brands by revenue",
			zap.String("filter_type", filter.FilterType),
			zap.Error(err))
		response := responses.ErrorResponse("Failed to retrieve top brands", http.StatusInternalServerError)
		c.JSON(http.StatusInternalServerError, response)
		return
	}

	response := responses.SuccessResponse("Top brands retrieved successfully", nil, brands)
	c.JSON(http.StatusOK, response)
}

// GetRevenueByContractType godoc
//
//	@Summary		Get revenue breakdown by contract type
//	@Description	Returns revenue breakdown by 4 contract types (ADVERTISING, AFFILIATE, BRAND_AMBASSADOR, CO_PRODUCING) + standard products
//	@Tags			Analytics/MarketingStaffs
//	@Accept			json
//	@Produce		json
//	@Param			filter_type	query		string	true	"Filter type"							Enums(MONTH, QUARTER, YEAR)
//	@Param			year		query		int		true	"Year (e.g., 2024)"						minimum(2000)	maximum(2100)
//	@Param			month		query		int		false	"Month (required for MONTH filter)"		minimum(1)		maximum(12)
//	@Param			quarter		query		int		false	"Quarter (required for QUARTER filter)"	minimum(1)		maximum(4)
//	@Success		200			{object}	responses.APIResponse{data=responses.RevenueByTypeResponse}
//	@Failure		400			{object}	responses.APIResponse
//	@Failure		500			{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/analytics/marketing/revenue-by-type [get]
func (h *MarketingAnalyticsHandler) GetRevenueByContractType(c *gin.Context) {
	ctx := c.Request.Context()

	var filter requests.TimeFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		zap.L().Error("Invalid request parameters", zap.Error(err))
		response := responses.ErrorResponse("Invalid request parameters", http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	revenueBreakdown, err := h.marketingAnalyticsService.GetRevenueByContractType(ctx, &filter)
	if err != nil {
		zap.L().Error("Failed to get revenue by contract type",
			zap.String("filter_type", filter.FilterType),
			zap.Error(err))
		response := responses.ErrorResponse("Failed to retrieve revenue breakdown", http.StatusInternalServerError)
		c.JSON(http.StatusInternalServerError, response)
		return
	}

	response := responses.SuccessResponse("Revenue breakdown retrieved successfully", nil, revenueBreakdown)
	c.JSON(http.StatusOK, response)
}

// GetUpcomingDeadlineCampaigns godoc
//
//	@Summary		Get campaigns approaching deadline
//	@Description	Returns campaigns with status = 'RUNNING' and end_date within specified days
//	@Tags			Analytics/MarketingStaffs
//	@Accept			json
//	@Produce		json
//	@Param			days	query		int	false	"Days before deadline (default: 10)"	minimum(1)	maximum(365)
//	@Success		200		{object}	responses.APIResponse{data=[]responses.UpcomingCampaignResponse}
//	@Failure		400		{object}	responses.APIResponse
//	@Failure		500		{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/analytics/marketing/upcoming-deadlines [get]
func (h *MarketingAnalyticsHandler) GetUpcomingDeadlineCampaigns(c *gin.Context) {
	ctx := c.Request.Context()

	var filter requests.UpcomingDeadlineFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		zap.L().Error("Invalid request parameters", zap.Error(err))
		response := responses.ErrorResponse("Invalid request parameters", http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	campaigns, err := h.marketingAnalyticsService.GetUpcomingDeadlineCampaigns(ctx, &filter)
	if err != nil {
		zap.L().Error("Failed to get upcoming deadline campaigns", zap.Error(err))
		response := responses.ErrorResponse("Failed to retrieve upcoming campaigns", http.StatusInternalServerError)
		c.JSON(http.StatusInternalServerError, response)
		return
	}

	response := responses.SuccessResponse("Upcoming deadline campaigns retrieved successfully", nil, campaigns)
	c.JSON(http.StatusOK, response)
}

// GetDashboard godoc
//
//	@Summary		Get marketing analytics dashboard
//	@Description	Returns aggregated analytics data including counts, revenue, top brands, and upcoming deadlines
//	@Tags			Analytics/MarketingStaffs
//	@Accept			json
//	@Produce		json
//	@Param			year	query		int	false	"Year (defaults to current)"	minimum(2000)	maximum(2100)
//	@Param			month	query		int	false	"Month (defaults to current)"	minimum(1)		maximum(12)
//	@Success		200		{object}	responses.APIResponse{data=responses.MarketingDashboardResponse}
//	@Failure		400		{object}	responses.APIResponse
//	@Failure		500		{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/analytics/marketing/dashboard [get]
func (h *MarketingAnalyticsHandler) GetDashboard(c *gin.Context) {
	ctx := c.Request.Context()

	var filter requests.DashboardFilter
	if err := c.ShouldBindQuery(&filter); err != nil {
		zap.L().Error("Invalid request parameters", zap.Error(err))
		response := responses.ErrorResponse("Invalid request parameters", http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	dashboard, err := h.marketingAnalyticsService.GetDashboard(ctx, &filter)
	if err != nil {
		zap.L().Error("Failed to get dashboard data", zap.Error(err))
		response := responses.ErrorResponse("Failed to retrieve dashboard data", http.StatusInternalServerError)
		c.JSON(http.StatusInternalServerError, response)
		return
	}

	response := responses.SuccessResponse("Dashboard data retrieved successfully", nil, dashboard)
	c.JSON(http.StatusOK, response)
}
