package handler

import (
	"net/http"

	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/iservice"
	"core-backend/pkg/utils"

	"github.com/gin-gonic/gin"
)

type SalesStaffAnalyticsHandler struct {
	service iservice.SalesStaffAnalyticsService
}

func NewSalesStaffAnalyticsHandler(service iservice.SalesStaffAnalyticsService) *SalesStaffAnalyticsHandler {
	return &SalesStaffAnalyticsHandler{
		service: service,
	}
}

// GetFinancialsDashboard godoc
//
//	@Summary		Get Sales Staff Financials Dashboard
//	@Description	Get aggregated financial metrics, charts, and top lists
//	@Tags			SalesStaffAnalytics
//	@Accept			json
//	@Produce		json
//	@Param			from_date		query		string	false	"From Date (YYYY-MM-DD)"
//	@Param			to_date			query		string	false	"To Date (YYYY-MM-DD)"
//	@Param			limit			query		int		false	"Limit for top lists (default 5)"
//	@Param			period_gap		query		string	false	"Period Gap (day, week, month, quarter, year)"			enums(day, week, month, quarter, year)
//	@Param			compare_with	query		string	false	"Compare With (previous day/week/month/quarter/year)"	enums(day, week, month, quarter, year)
//	@Success		200				{object}	responses.APIResponse{data=responses.FinancialsDashboardResponse}
//	@Failure		400				{object}	responses.APIResponse
//	@Failure		500				{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/analytics/sales/financials/dashboard [get]
func (h *SalesStaffAnalyticsHandler) GetFinancialsDashboard(c *gin.Context) {
	var req requests.SalesDashboardFilter
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid query parameters", http.StatusBadRequest))
		return
	}

	ctx := c.Request.Context()
	result, err := h.service.GetFinancialsDashboard(ctx, h.getDefaultFilterRequest(&req))
	if err != nil {
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse(err.Error(), http.StatusInternalServerError))
		return
	}

	c.JSON(http.StatusOK, responses.SuccessResponse("Financials dashboard retrieved successfully", nil, result))
}

// GetOrdersDashboard godoc
//
//	@Summary		Get Sales Staff Orders Dashboard
//	@Description	Get aggregated order metrics, charts, and top lists
//	@Tags			SalesStaffAnalytics
//	@Accept			json
//	@Produce		json
//	@Param			from_date	query		string	false	"From Date (YYYY-MM-DD)"
//	@Param			to_date		query		string	false	"To Date (YYYY-MM-DD)"
//	@Param			limit		query		int		false	"Limit for top lists (default 5)"
//	@Param			period_gap	query		string	false	"Period Gap (day, week, month, quarter, year)"	enums(day, week, month, quarter, year)
//	@Success		200			{object}	responses.APIResponse{data=responses.OrdersDashboardResponse}
//	@Failure		400			{object}	responses.APIResponse
//	@Failure		500			{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/analytics/sales/orders/dashboard [get]
func (h *SalesStaffAnalyticsHandler) GetOrdersDashboard(c *gin.Context) {
	var req requests.SalesDashboardFilter
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid query parameters", http.StatusBadRequest))
		return
	}

	ctx := c.Request.Context()
	result, err := h.service.GetOrdersDashboard(ctx, h.getDefaultFilterRequest(&req))
	if err != nil {
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse(err.Error(), http.StatusInternalServerError))
		return
	}

	c.JSON(http.StatusOK, responses.SuccessResponse("Orders dashboard retrieved successfully", nil, result))
}

// GetRevenueTrend godoc
//
//	@Summary		Get Revenue Trend
//	@Description	Get revenue trend charts
//	@Tags			SalesStaffAnalytics
//	@Accept			json
//	@Produce		json
//	@Param			from_date	query		string	false	"From Date (YYYY-MM-DD)"
//	@Param			to_date		query		string	false	"To Date (YYYY-MM-DD)"
//	@Param			period_gap	query		string	false	"Period Gap (day, week, month, quarter, year)"	enums(day, week, month, quarter, year)
//	@Success		200			{object}	responses.APIResponse{data=responses.RevenueTrendCharts}
//	@Failure		400			{object}	responses.APIResponse
//	@Failure		500			{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/analytics/sales/financials/trend [get]
func (h *SalesStaffAnalyticsHandler) GetRevenueTrend(c *gin.Context) {
	var req requests.SalesDashboardFilter
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid query parameters", http.StatusBadRequest))
		return
	}

	ctx := c.Request.Context()
	result, err := h.service.GetRevenueTrend(ctx, h.getDefaultFilterRequest(&req))
	if err != nil {
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse(err.Error(), http.StatusInternalServerError))
		return
	}

	c.JSON(http.StatusOK, responses.SuccessResponse("Revenue trend retrieved successfully", nil, result))
}

// GetOrdersTrend godoc
//
//	@Summary		Get Orders Trend
//	@Description	Get orders trend charts
//	@Tags			SalesStaffAnalytics
//	@Accept			json
//	@Produce		json
//	@Param			from_date	query		string	false	"From Date (YYYY-MM-DD)"
//	@Param			to_date		query		string	false	"To Date (YYYY-MM-DD)"
//	@Param			period_gap	query		string	false	"Period Gap (day, week, month, quarter, year)"	enums(day, week, month, quarter, year)
//	@Success		200			{object}	responses.APIResponse{data=responses.OrdersTrendCharts}
//	@Failure		400			{object}	responses.APIResponse
//	@Failure		500			{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/analytics/sales/orders/trend [get]
func (h *SalesStaffAnalyticsHandler) GetOrdersTrend(c *gin.Context) {
	var req requests.SalesDashboardFilter
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid query parameters", http.StatusBadRequest))
		return
	}

	ctx := c.Request.Context()
	result, err := h.service.GetOrdersTrend(ctx, h.getDefaultFilterRequest(&req))
	if err != nil {
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse(err.Error(), http.StatusInternalServerError))
		return
	}

	c.JSON(http.StatusOK, responses.SuccessResponse("Orders trend retrieved successfully", nil, result))
}

// GetRevenueGrowth godoc
//
//	@Summary		Get Revenue Growth
//	@Description	Get revenue growth percentage
//	@Tags			SalesStaffAnalytics
//	@Accept			json
//	@Produce		json
//	@Param			compare_with	query		string	false	"Compare With (previous day/week/month/quarter/year)"	enums(day, week, month, quarter, year)
//	@Success		200				{object}	responses.APIResponse{data=float64}
//	@Failure		400				{object}	responses.APIResponse
//	@Failure		500				{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/analytics/sales/financials/growth [get]
func (h *SalesStaffAnalyticsHandler) GetRevenueGrowth(c *gin.Context) {
	var req requests.SalesDashboardFilter
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid query parameters", http.StatusBadRequest))
		return
	}
	// Override irrelevant fields
	req.FromDateStr = nil
	req.ToDateStr = nil
	req.Limit = 0
	req.PeriodGap = ""

	ctx := c.Request.Context()
	result, err := h.service.GetRevenueGrowth(ctx, h.getDefaultFilterRequest(&req))
	if err != nil {
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse(err.Error(), http.StatusInternalServerError))
		return
	}

	c.JSON(http.StatusOK, responses.SuccessResponse("Revenue growth retrieved successfully", nil, result))
}

func (h *SalesStaffAnalyticsHandler) getDefaultFilterRequest(filter *requests.SalesDashboardFilter) *requests.SalesDashboardFilter {
	if filter == nil {
		return nil
	}

	if filter.Limit <= 0 {
		filter.Limit = 5
	}
	if filter.FromDateStr != nil && *filter.FromDateStr != "" {
		filter.FromDate = utils.BestEffortParseLocalTime(*filter.FromDateStr)
	}
	if filter.ToDateStr != nil && *filter.ToDateStr != "" {
		filter.ToDate = utils.BestEffortParseLocalTime(*filter.ToDateStr)
	}
	if filter.PeriodGap == "" {
		filter.PeriodGap = "month"
	}
	if filter.CompareWith == "" {
		filter.CompareWith = "month"
	}

	return filter
}
