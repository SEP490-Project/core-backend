package handler

import (
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/iservice"
	"net/http"

	"github.com/gin-gonic/gin"
)

type BrandPartnerAnalyticsHandler struct {
	analyticsService iservice.BrandPartnerAnalyticsService
}

func NewBrandPartnerAnalyticsHandler(analyticsService iservice.BrandPartnerAnalyticsService) *BrandPartnerAnalyticsHandler {
	return &BrandPartnerAnalyticsHandler{
		analyticsService: analyticsService,
	}
}

// GetDashboard returns the complete Brand Partner analytics dashboard
//
//	@Summary		Get Brand Partner Dashboard
//	@Description	Returns comprehensive analytics dashboard for Brand Partner including overview metrics, top products, campaigns, content, revenue trend, affiliate metrics, and contracts
//	@Tags			Brand Partner Analytics
//	@Accept			json
//	@Produce		json
//	@Param			year	query		int	false	"Year for filtering (defaults to current year)"
//	@Param			month	query		int	false	"Month for filtering (defaults to current month)"
//	@Success		200		{object}	responses.APIResponse{data=responses.BrandPartnerDashboardResponse}
//	@Failure		400		{object}	responses.APIResponse
//	@Failure		401		{object}	responses.APIResponse
//	@Failure		500		{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/analytics/brand-partner/dashboard [get]
func (h *BrandPartnerAnalyticsHandler) GetDashboard(c *gin.Context) {
	ctx := c.Request.Context()

	brandUserID, err := extractUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, responses.ErrorResponse("Unauthorized: "+err.Error(), http.StatusUnauthorized))
		return
	}

	var req requests.BrandPartnerDashboardRequest
	if err = c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid query parameters: "+err.Error(), http.StatusBadRequest))
		return
	}

	dashboard, err := h.analyticsService.GetDashboard(ctx, brandUserID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse("Failed to fetch dashboard: "+err.Error(), http.StatusInternalServerError))
		return
	}

	c.JSON(http.StatusOK, responses.SuccessResponse("Dashboard fetched successfully", nil, dashboard))
}

// GetTopProducts returns the top products by revenue for the brand
//
//	@Summary		Get Brand's Top Products
//	@Description	Returns top products by revenue for the brand partner
//	@Tags			Brand Partner Analytics
//	@Accept			json
//	@Produce		json
//	@Param			start_date	query		string	false	"Start date (ISO 8601 format)"
//	@Param			end_date	query		string	false	"End date (ISO 8601 format)"
//	@Param			limit		query		int		false	"Number of products to return (default: 10, max: 50)"
//	@Success		200			{object}	responses.APIResponse{data=[]responses.BrandProductMetric}
//	@Failure		400			{object}	responses.APIResponse
//	@Failure		401			{object}	responses.APIResponse
//	@Failure		500			{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/analytics/brand-partner/top-products [get]
func (h *BrandPartnerAnalyticsHandler) GetTopProducts(c *gin.Context) {
	ctx := c.Request.Context()

	brandUserID, err := extractUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, responses.ErrorResponse("Unauthorized: "+err.Error(), http.StatusUnauthorized))
		return
	}

	var req requests.BrandTopProductsRequest
	if err = c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid query parameters: "+err.Error(), http.StatusBadRequest))
		return
	}

	products, err := h.analyticsService.GetTopProducts(ctx, brandUserID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse("Failed to fetch top products: "+err.Error(), http.StatusInternalServerError))
		return
	}

	c.JSON(http.StatusOK, responses.SuccessResponse("Top products fetched successfully", nil, products))
}

// GetCampaignMetrics returns the campaign performance metrics for the brand
//
//	@Summary		Get Brand's Campaign Metrics
//	@Description	Returns campaign performance metrics for the brand partner
//	@Tags			Brand Partner Analytics
//	@Accept			json
//	@Produce		json
//	@Param			start_date	query		string	false	"Start date (ISO 8601 format)"
//	@Param			end_date	query		string	false	"End date (ISO 8601 format)"
//	@Param			status		query		string	false	"Filter by campaign status (DRAFT, ACTIVE, IN_PROGRESS, PENDING, FINISHED, CANCELLED)"
//	@Param			limit		query		int		false	"Number of campaigns to return (default: 10, max: 50)"
//	@Success		200			{object}	responses.APIResponse{data=[]responses.BrandCampaignMetric}
//	@Failure		400			{object}	responses.APIResponse
//	@Failure		401			{object}	responses.APIResponse
//	@Failure		500			{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/analytics/brand-partner/campaigns [get]
func (h *BrandPartnerAnalyticsHandler) GetCampaignMetrics(c *gin.Context) {
	ctx := c.Request.Context()

	brandUserID, err := extractUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, responses.ErrorResponse("Unauthorized: "+err.Error(), http.StatusUnauthorized))
		return
	}

	var req requests.BrandCampaignsRequest
	if err = c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid query parameters: "+err.Error(), http.StatusBadRequest))
		return
	}

	campaigns, err := h.analyticsService.GetCampaignMetrics(ctx, brandUserID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse("Failed to fetch campaign metrics: "+err.Error(), http.StatusInternalServerError))
		return
	}

	c.JSON(http.StatusOK, responses.SuccessResponse("Campaign metrics fetched successfully", nil, campaigns))
}

// GetContentMetrics returns the content performance metrics for the brand
//
//	@Summary		Get Brand's Content Metrics
//	@Description	Returns content performance metrics summary for the brand partner
//	@Tags			Brand Partner Analytics
//	@Accept			json
//	@Produce		json
//	@Param			start_date	query		string	false	"Start date (ISO 8601 format)"
//	@Param			end_date	query		string	false	"End date (ISO 8601 format)"
//	@Success		200			{object}	responses.APIResponse{data=responses.BrandContentMetric}
//	@Failure		400			{object}	responses.APIResponse
//	@Failure		401			{object}	responses.APIResponse
//	@Failure		500			{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/analytics/brand-partner/content [get]
func (h *BrandPartnerAnalyticsHandler) GetContentMetrics(c *gin.Context) {
	ctx := c.Request.Context()

	brandUserID, err := extractUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, responses.ErrorResponse("Unauthorized: "+err.Error(), http.StatusUnauthorized))
		return
	}

	var req requests.BrandContentMetricsRequest
	if err = c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid query parameters: "+err.Error(), http.StatusBadRequest))
		return
	}

	content, err := h.analyticsService.GetContentMetrics(ctx, brandUserID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse("Failed to fetch content metrics: "+err.Error(), http.StatusInternalServerError))
		return
	}

	c.JSON(http.StatusOK, responses.SuccessResponse("Content metrics fetched successfully", nil, content))
}

// GetRevenueTrend returns the revenue time-series for the brand
//
//	@Summary		Get Brand's Revenue Trend
//	@Description	Returns revenue time-series data for the brand partner
//	@Tags			Brand Partner Analytics
//	@Accept			json
//	@Produce		json
//	@Param			start_date	query		string	false	"Start date (ISO 8601 format)"
//	@Param			end_date	query		string	false	"End date (ISO 8601 format)"
//	@Param			granularity	query		string	false	"Time granularity (DAY, WEEK, MONTH - default: DAY)"
//	@Success		200			{object}	responses.APIResponse{data=[]responses.BrandRevenueTrendPoint}
//	@Failure		400			{object}	responses.APIResponse
//	@Failure		401			{object}	responses.APIResponse
//	@Failure		500			{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/analytics/brand-partner/revenue-trend [get]
func (h *BrandPartnerAnalyticsHandler) GetRevenueTrend(c *gin.Context) {
	ctx := c.Request.Context()

	brandUserID, err := extractUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, responses.ErrorResponse("Unauthorized: "+err.Error(), http.StatusUnauthorized))
		return
	}

	var req requests.BrandRevenueTrendRequest
	if err = c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid query parameters: "+err.Error(), http.StatusBadRequest))
		return
	}

	trend, err := h.analyticsService.GetRevenueTrend(ctx, brandUserID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse("Failed to fetch revenue trend: "+err.Error(), http.StatusInternalServerError))
		return
	}

	c.JSON(http.StatusOK, responses.SuccessResponse("Revenue trend fetched successfully", nil, trend))
}

// GetAffiliateMetrics returns the affiliate link performance for the brand
//
//	@Summary		Get Brand's Affiliate Metrics
//	@Description	Returns affiliate link performance metrics for the brand partner
//	@Tags			Brand Partner Analytics
//	@Accept			json
//	@Produce		json
//	@Param			start_date	query		string	false	"Start date (ISO 8601 format)"
//	@Param			end_date	query		string	false	"End date (ISO 8601 format)"
//	@Success		200			{object}	responses.APIResponse{data=responses.BrandAffiliateMetric}
//	@Failure		400			{object}	responses.APIResponse
//	@Failure		401			{object}	responses.APIResponse
//	@Failure		500			{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/analytics/brand-partner/affiliates [get]
func (h *BrandPartnerAnalyticsHandler) GetAffiliateMetrics(c *gin.Context) {
	ctx := c.Request.Context()

	brandUserID, err := extractUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, responses.ErrorResponse("Unauthorized: "+err.Error(), http.StatusUnauthorized))
		return
	}

	var req requests.BrandAffiliateMetricsRequest
	if err = c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid query parameters: "+err.Error(), http.StatusBadRequest))
		return
	}

	affiliate, err := h.analyticsService.GetAffiliateMetrics(ctx, brandUserID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse("Failed to fetch affiliate metrics: "+err.Error(), http.StatusInternalServerError))
		return
	}

	c.JSON(http.StatusOK, responses.SuccessResponse("Affiliate metrics fetched successfully", nil, affiliate))
}

// GetContractDetails returns the contract details for the brand
//
//	@Summary		Get Brand's Contract Details
//	@Description	Returns contract details for the brand partner
//	@Tags			Brand Partner Analytics
//	@Accept			json
//	@Produce		json
//	@Param			status	query		string	false	"Filter by contract status (DRAFT, PENDING, ACTIVE, COMPLETED, CANCELLED)"
//	@Param			limit	query		int		false	"Number of contracts to return (default: 10, max: 50)"
//	@Success		200		{object}	responses.APIResponse{data=[]responses.BrandContractDetail}
//	@Failure		400		{object}	responses.APIResponse
//	@Failure		401		{object}	responses.APIResponse
//	@Failure		500		{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/analytics/brand-partner/contracts [get]
func (h *BrandPartnerAnalyticsHandler) GetContractDetails(c *gin.Context) {
	ctx := c.Request.Context()

	brandUserID, err := extractUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, responses.ErrorResponse("Unauthorized: "+err.Error(), http.StatusUnauthorized))
		return
	}

	var req requests.BrandContractsRequest
	if err = c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid query parameters: "+err.Error(), http.StatusBadRequest))
		return
	}

	contracts, err := h.analyticsService.GetContractDetails(ctx, brandUserID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse("Failed to fetch contract details: "+err.Error(), http.StatusInternalServerError))
		return
	}

	c.JSON(http.StatusOK, responses.SuccessResponse("Contract details fetched successfully", nil, contracts))
}
