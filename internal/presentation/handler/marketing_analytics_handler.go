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
//	@Tags			Analytics.Marketing
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
//	@Tags			Analytics.Marketing
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
//	@Tags			Analytics.Marketing
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

// GetGrossContractRevenue godoc
//
//	@Summary		Get gross contract revenue for period
//	@Description	Returns total gross revenue from paid contract payments (before refund deductions)
//	@Tags			Analytics.Marketing
//	@Accept			json
//	@Produce		json
//	@Param			period		query		string	false	"Period preset"						Enums(TODAY, YESTERDAY, THIS_WEEK, LAST_WEEK, THIS_MONTH, LAST_MONTH, THIS_QUARTER, LAST_QUARTER, THIS_YEAR, LAST_YEAR, LAST_7_DAYS, LAST_30_DAYS, CUSTOM)
//	@Param			from_date	query		string	false	"Start date (when period=CUSTOM)"	Format(date)
//	@Param			to_date		query		string	false	"End date (when period=CUSTOM)"		Format(date)
//	@Success		200			{object}	responses.APIResponse{data=float64}
//	@Failure		400			{object}	responses.APIResponse
//	@Failure		500			{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/analytics/marketing/gross-revenue [get]
func (h *MarketingAnalyticsHandler) GetGrossContractRevenue(c *gin.Context) {
	ctx := c.Request.Context()

	var filter requests.DashboardFilterRequest
	if err := c.ShouldBindQuery(&filter); err != nil {
		zap.L().Error("Invalid request parameters", zap.Error(err))
		response := responses.ErrorResponse("Invalid request parameters", http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	revenue, err := h.marketingAnalyticsService.GetGrossContractRevenue(ctx, &filter)
	if err != nil {
		zap.L().Error("Failed to get gross contract revenue",
			zap.String("period", filter.GetPresetLabel()),
			zap.Error(err))
		response := responses.ErrorResponse("Failed to retrieve gross contract revenue", http.StatusInternalServerError)
		c.JSON(http.StatusInternalServerError, response)
		return
	}

	response := responses.SuccessResponse("Gross contract revenue retrieved successfully", nil, revenue)
	c.JSON(http.StatusOK, response)
}

// NetRevenueResponse represents the net revenue with breakdown
type NetRevenueResponse struct {
	GrossRevenue float64 `json:"gross_revenue" example:"100000000.00"`
	NetRevenue   float64 `json:"net_revenue" example:"85000000.00"`
	TotalRefunds float64 `json:"total_refunds" example:"15000000.00"`
}

// GetNetContractRevenue godoc
//
//	@Summary		Get net contract revenue for period
//	@Description	Returns net revenue (gross - refunds) from contract payments including breakdown
//	@Tags			Analytics.Marketing
//	@Accept			json
//	@Produce		json
//	@Param			period		query		string	false	"Period preset"						Enums(TODAY, YESTERDAY, THIS_WEEK, LAST_WEEK, THIS_MONTH, LAST_MONTH, THIS_QUARTER, LAST_QUARTER, THIS_YEAR, LAST_YEAR, LAST_7_DAYS, LAST_30_DAYS, CUSTOM)
//	@Param			from_date	query		string	false	"Start date (when period=CUSTOM)"	Format(date)
//	@Param			to_date		query		string	false	"End date (when period=CUSTOM)"		Format(date)
//	@Success		200			{object}	responses.APIResponse{data=NetRevenueResponse}
//	@Failure		400			{object}	responses.APIResponse
//	@Failure		500			{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/analytics/marketing/net-revenue [get]
func (h *MarketingAnalyticsHandler) GetNetContractRevenue(c *gin.Context) {
	ctx := c.Request.Context()

	var filter requests.DashboardFilterRequest
	if err := c.ShouldBindQuery(&filter); err != nil {
		zap.L().Error("Invalid request parameters", zap.Error(err))
		response := responses.ErrorResponse("Invalid request parameters", http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	gross, net, refunds, err := h.marketingAnalyticsService.GetNetContractRevenue(ctx, &filter)
	if err != nil {
		zap.L().Error("Failed to get net contract revenue",
			zap.String("period", filter.GetPresetLabel()),
			zap.Error(err))
		response := responses.ErrorResponse("Failed to retrieve net contract revenue", http.StatusInternalServerError)
		c.JSON(http.StatusInternalServerError, response)
		return
	}

	result := NetRevenueResponse{
		GrossRevenue: gross,
		NetRevenue:   net,
		TotalRefunds: refunds,
	}

	response := responses.SuccessResponse("Net contract revenue retrieved successfully", nil, result)
	c.JSON(http.StatusOK, response)
}

// GetTopBrandsByRevenue godoc
//
//	@Summary		Get top brands by revenue
//	@Description	Returns top brands by total revenue (contract payments + standard product sales)
//	@Tags			Analytics.Marketing
//	@Accept			json
//	@Produce		json
//	@Param			period		query		string	false	"Period preset"									Enums(TODAY, YESTERDAY, THIS_WEEK, LAST_WEEK, THIS_MONTH, LAST_MONTH, THIS_QUARTER, LAST_QUARTER, THIS_YEAR, LAST_YEAR, LAST_7_DAYS, LAST_30_DAYS, CUSTOM)
//	@Param			from_date	query		string	false	"Start date (when period=CUSTOM)"				Format(date)
//	@Param			to_date		query		string	false	"End date (when period=CUSTOM)"					Format(date)
//	@Param			limit		query		int		false	"Number of top brands to return (default: 5)"	minimum(1)	maximum(50)
//	@Success		200			{object}	responses.APIResponse{data=[]responses.BrandRevenueResponse}
//	@Failure		400			{object}	responses.APIResponse
//	@Failure		500			{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/analytics/marketing/top-brands [get]
func (h *MarketingAnalyticsHandler) GetTopBrandsByRevenue(c *gin.Context) {
	ctx := c.Request.Context()

	var filter requests.DashboardFilterRequest
	if err := c.ShouldBindQuery(&filter); err != nil {
		zap.L().Error("Invalid request parameters", zap.Error(err))
		response := responses.ErrorResponse("Invalid request parameters", http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	brands, err := h.marketingAnalyticsService.GetTopBrandsByRevenue(ctx, &filter)
	if err != nil {
		zap.L().Error("Failed to get top brands by revenue",
			zap.String("period", filter.GetPresetLabel()),
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
//	@Tags			Analytics.Marketing
//	@Accept			json
//	@Produce		json
//	@Param			period		query		string	false	"Period preset"						Enums(TODAY, YESTERDAY, THIS_WEEK, LAST_WEEK, THIS_MONTH, LAST_MONTH, THIS_QUARTER, LAST_QUARTER, THIS_YEAR, LAST_YEAR, LAST_7_DAYS, LAST_30_DAYS, CUSTOM)
//	@Param			from_date	query		string	false	"Start date (when period=CUSTOM)"	Format(date)
//	@Param			to_date		query		string	false	"End date (when period=CUSTOM)"		Format(date)
//	@Success		200			{object}	responses.APIResponse{data=responses.RevenueByTypeResponse}
//	@Failure		400			{object}	responses.APIResponse
//	@Failure		500			{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/analytics/marketing/revenue-by-type [get]
func (h *MarketingAnalyticsHandler) GetRevenueByContractType(c *gin.Context) {
	ctx := c.Request.Context()

	var filter requests.DashboardFilterRequest
	if err := c.ShouldBindQuery(&filter); err != nil {
		zap.L().Error("Invalid request parameters", zap.Error(err))
		response := responses.ErrorResponse("Invalid request parameters", http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	revenueBreakdown, err := h.marketingAnalyticsService.GetRevenueByContractType(ctx, &filter)
	if err != nil {
		zap.L().Error("Failed to get revenue by contract type",
			zap.String("period", filter.GetPresetLabel()),
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
//	@Tags			Analytics.Marketing
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
//	@Tags			Analytics.Marketing
//	@Accept			json
//	@Produce		json
//	@Param			period		query		string	false	"Period preset (defaults to THIS_MONTH)"	Enums(TODAY, YESTERDAY, THIS_WEEK, LAST_WEEK, THIS_MONTH, LAST_MONTH, THIS_QUARTER, LAST_QUARTER, THIS_YEAR, LAST_YEAR, LAST_7_DAYS, LAST_30_DAYS, CUSTOM)
//	@Param			from_date	query		string	false	"Start date (when period=CUSTOM)"			Format(date)
//	@Param			to_date		query		string	false	"End date (when period=CUSTOM)"				Format(date)
//	@Param			limit		query		int		false	"Limit for top-N queries (default: 5)"		minimum(1)	maximum(50)
//	@Success		200			{object}	responses.APIResponse{data=responses.MarketingDashboardResponse}
//	@Failure		400			{object}	responses.APIResponse
//	@Failure		500			{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/analytics/marketing/dashboard [get]
func (h *MarketingAnalyticsHandler) GetDashboard(c *gin.Context) {
	ctx := c.Request.Context()

	var filter requests.DashboardFilterRequest
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

// GetContractStatusDistribution godoc
//
//	@Summary		Get contract status distribution
//	@Description	Returns contracts grouped by status (Draft, Active, Completed, Terminated, Brand Violations, KOL Violations)
//	@Tags			Analytics.Marketing
//	@Produce		json
//	@Param			period		query		string	false	"Period preset"						Enums(TODAY, YESTERDAY, THIS_WEEK, LAST_WEEK, THIS_MONTH, LAST_MONTH, THIS_QUARTER, LAST_QUARTER, THIS_YEAR, LAST_YEAR, LAST_7_DAYS, LAST_30_DAYS, CUSTOM)
//	@Param			from_date	query		string	false	"Start date (when period=CUSTOM)"	Format(date)
//	@Param			to_date		query		string	false	"End date (when period=CUSTOM)"		Format(date)
//	@Success		200			{object}	responses.APIResponse{data=responses.ContractStatusDistributionResponse}
//	@Failure		400			{object}	responses.APIResponse
//	@Failure		500			{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/analytics/marketing/contract-status-distribution [get]
func (h *MarketingAnalyticsHandler) GetContractStatusDistribution(c *gin.Context) {
	ctx := c.Request.Context()

	var filter requests.DashboardFilterRequest
	if err := c.ShouldBindQuery(&filter); err != nil {
		zap.L().Error("Invalid request parameters", zap.Error(err))
		response := responses.ErrorResponse("Invalid query parameters: "+err.Error(), http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	result, err := h.marketingAnalyticsService.GetContractStatusDistribution(ctx, &filter)
	if err != nil {
		zap.L().Error("Failed to get contract status distribution", zap.Error(err))
		response := responses.ErrorResponse("Failed to get contract status distribution: "+err.Error(), http.StatusInternalServerError)
		c.JSON(http.StatusInternalServerError, response)
		return
	}

	response := responses.SuccessResponse("Contract status distribution retrieved successfully", nil, result)
	c.JSON(http.StatusOK, response)
}

// GetTaskStatusDistribution godoc
//
//	@Summary		Get task status distribution
//	@Description	Returns tasks grouped by status (ToDo, InProgress, Done, Cancelled)
//	@Tags			Analytics.Marketing
//	@Produce		json
//	@Param			period		query		string	false	"Period preset"						Enums(TODAY, YESTERDAY, THIS_WEEK, LAST_WEEK, THIS_MONTH, LAST_MONTH, THIS_QUARTER, LAST_QUARTER, THIS_YEAR, LAST_YEAR, LAST_7_DAYS, LAST_30_DAYS, CUSTOM)
//	@Param			from_date	query		string	false	"Start date (when period=CUSTOM)"	Format(date)
//	@Param			to_date		query		string	false	"End date (when period=CUSTOM)"		Format(date)
//	@Success		200			{object}	responses.APIResponse{data=responses.TaskStatusDistributionResponse}
//	@Failure		400			{object}	responses.APIResponse
//	@Failure		500			{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/analytics/marketing/task-status-distribution [get]
func (h *MarketingAnalyticsHandler) GetTaskStatusDistribution(c *gin.Context) {
	ctx := c.Request.Context()

	var filter requests.DashboardFilterRequest
	if err := c.ShouldBindQuery(&filter); err != nil {
		zap.L().Error("Invalid request parameters", zap.Error(err))
		response := responses.ErrorResponse("Invalid query parameters: "+err.Error(), http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	result, err := h.marketingAnalyticsService.GetTaskStatusDistribution(ctx, &filter)
	if err != nil {
		zap.L().Error("Failed to get task status distribution", zap.Error(err))
		response := responses.ErrorResponse("Failed to get task status distribution: "+err.Error(), http.StatusInternalServerError)
		c.JSON(http.StatusInternalServerError, response)
		return
	}

	response := responses.SuccessResponse("Task status distribution retrieved successfully", nil, result)
	c.JSON(http.StatusOK, response)
}

// GetRevenueOverTime godoc
//
//	@Summary		Get revenue over time
//	@Description	Returns revenue breakdown by source over time for combo chart visualization
//	@Tags			Analytics.Marketing
//	@Produce		json
//	@Param			period				query		string	false	"Period preset"						Enums(TODAY, YESTERDAY, THIS_WEEK, LAST_WEEK, THIS_MONTH, LAST_MONTH, THIS_QUARTER, LAST_QUARTER, THIS_YEAR, LAST_YEAR, LAST_7_DAYS, LAST_30_DAYS, CUSTOM)
//	@Param			from_date			query		string	false	"Start date (when period=CUSTOM)"	Format(date)
//	@Param			to_date				query		string	false	"End date (when period=CUSTOM)"		Format(date)
//	@Param			trend_granularity	query		string	false	"Chart granularity"					Enums(HOUR, DAY, WEEK, MONTH)
//	@Success		200					{object}	responses.APIResponse{data=responses.RevenueOverTimeResponse}
//	@Failure		400					{object}	responses.APIResponse
//	@Failure		500					{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/analytics/marketing/revenue-over-time [get]
func (h *MarketingAnalyticsHandler) GetRevenueOverTime(c *gin.Context) {
	ctx := c.Request.Context()

	var filter requests.DashboardFilterRequest
	if err := c.ShouldBindQuery(&filter); err != nil {
		zap.L().Error("Invalid request parameters", zap.Error(err))
		response := responses.ErrorResponse("Invalid query parameters: "+err.Error(), http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	result, err := h.marketingAnalyticsService.GetRevenueOverTime(ctx, &filter)
	if err != nil {
		zap.L().Error("Failed to get revenue over time", zap.Error(err))
		response := responses.ErrorResponse("Failed to get revenue over time: "+err.Error(), http.StatusInternalServerError)
		c.JSON(http.StatusInternalServerError, response)
		return
	}

	response := responses.SuccessResponse("Revenue over time retrieved successfully", nil, result)
	c.JSON(http.StatusOK, response)
}

// GetRefundViolationStats godoc
//
//	@Summary		Get refund and violation statistics
//	@Description	Returns system-wide refund and violation statistics
//	@Tags			Analytics.Marketing
//	@Produce		json
//	@Param			period		query		string	false	"Period preset"						Enums(TODAY, YESTERDAY, THIS_WEEK, LAST_WEEK, THIS_MONTH, LAST_MONTH, THIS_QUARTER, LAST_QUARTER, THIS_YEAR, LAST_YEAR, LAST_7_DAYS, LAST_30_DAYS, CUSTOM)
//	@Param			from_date	query		string	false	"Start date (when period=CUSTOM)"	Format(date)
//	@Param			to_date		query		string	false	"End date (when period=CUSTOM)"		Format(date)
//	@Success		200			{object}	responses.APIResponse{data=responses.RefundViolationStatsResponse}
//	@Failure		400			{object}	responses.APIResponse
//	@Failure		500			{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/analytics/marketing/refund-violation-stats [get]
func (h *MarketingAnalyticsHandler) GetRefundViolationStats(c *gin.Context) {
	ctx := c.Request.Context()

	var filter requests.DashboardFilterRequest
	if err := c.ShouldBindQuery(&filter); err != nil {
		zap.L().Error("Invalid request parameters", zap.Error(err))
		response := responses.ErrorResponse("Invalid query parameters: "+err.Error(), http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	result, err := h.marketingAnalyticsService.GetRefundViolationStats(ctx, &filter)
	if err != nil {
		zap.L().Error("Failed to get refund violation stats", zap.Error(err))
		response := responses.ErrorResponse("Failed to get refund violation stats: "+err.Error(), http.StatusInternalServerError)
		c.JSON(http.StatusInternalServerError, response)
		return
	}

	response := responses.SuccessResponse("Refund violation stats retrieved successfully", nil, result)
	c.JSON(http.StatusOK, response)
}

// GetContractRevenueBreakdown godoc
//
//	@Summary		Get contract revenue breakdown over time
//	@Description	Returns detailed breakdown of contract revenue for ComposedChart visualization
//	@Description	Components: Base Cost (Line), Affiliate Revenue (Line), Limited Product Brand/System Shares (Lines), Total (Bar)
//	@Tags			Analytics.Marketing
//	@Accept			json
//	@Produce		json
//	@Param			period				query		string	false	"Period preset"						Enums(TODAY, YESTERDAY, THIS_WEEK, LAST_WEEK, THIS_MONTH, LAST_MONTH, THIS_QUARTER, LAST_QUARTER, THIS_YEAR, LAST_YEAR, LAST_7_DAYS, LAST_30_DAYS, CUSTOM)
//	@Param			from_date			query		string	false	"Start date (when period=CUSTOM)"	Format(date)
//	@Param			to_date				query		string	false	"End date (when period=CUSTOM)"		Format(date)
//	@Param			trend_granularity	query		string	false	"Chart granularity"					Enums(HOUR, DAY, WEEK, MONTH)
//	@Success		200					{object}	responses.APIResponse{data=responses.ContractRevenueBreakdownResponse}
//	@Failure		400					{object}	responses.APIResponse
//	@Failure		500					{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/analytics/marketing/contract-revenue-breakdown [get]
func (h *MarketingAnalyticsHandler) GetContractRevenueBreakdown(c *gin.Context) {
	ctx := c.Request.Context()

	var filter requests.DashboardFilterRequest
	if err := c.ShouldBindQuery(&filter); err != nil {
		zap.L().Error("Invalid request parameters", zap.Error(err))
		response := responses.ErrorResponse("Invalid query parameters: "+err.Error(), http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	// Get granularity from filter (defaults to DAY)
	granularity := filter.GetTrendGranularity()

	result, err := h.marketingAnalyticsService.GetContractRevenueBreakdown(ctx, &filter, granularity)
	if err != nil {
		zap.L().Error("Failed to get contract revenue breakdown", zap.Error(err))
		response := responses.ErrorResponse("Failed to get contract revenue breakdown: "+err.Error(), http.StatusInternalServerError)
		c.JSON(http.StatusInternalServerError, response)
		return
	}

	response := responses.SuccessResponse("Contract revenue breakdown retrieved successfully", nil, result)
	c.JSON(http.StatusOK, response)
}
