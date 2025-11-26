package handler

import (
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/iservice"
	"net/http"

	"github.com/gin-gonic/gin"
)

// SalesStaffAnalyticsHandler handles Sales Staff analytics endpoints
type SalesStaffAnalyticsHandler struct {
	service iservice.SalesStaffAnalyticsService
}

// NewSalesStaffAnalyticsHandler creates a new Sales Staff analytics handler
func NewSalesStaffAnalyticsHandler(service iservice.SalesStaffAnalyticsService) *SalesStaffAnalyticsHandler {
	return &SalesStaffAnalyticsHandler{service: service}
}

// GetDashboard returns the complete Sales Staff dashboard
// @Summary Get Sales Staff Dashboard
// @Description Returns comprehensive sales dashboard with overview metrics, orders breakdown, revenue by source, top brands, top products, recent orders, and revenue trend
// @Tags Sales Staff Analytics
// @Accept json
// @Produce json
// @Param year query int false "Year (defaults to current year)" minimum(2000) maximum(2100) example(2025)
// @Param month query int false "Month (defaults to current month)" minimum(1) maximum(12) example(11)
// @Success 200 {object} responses.APIResponse{data=responses.SalesStaffDashboardResponse}
// @Failure 400 {object} responses.APIResponse
// @Failure 401 {object} responses.APIResponse
// @Failure 403 {object} responses.APIResponse
// @Failure 500 {object} responses.APIResponse
// @Security BearerAuth
// @Router /api/v1/analytics/sales/dashboard [get]
func (h *SalesStaffAnalyticsHandler) GetDashboard(c *gin.Context) {
	var req requests.SalesStaffDashboardRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid request parameters", http.StatusBadRequest))
		return
	}

	result, err := h.service.GetDashboard(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse("Failed to get dashboard", http.StatusInternalServerError))
		return
	}

	c.JSON(http.StatusOK, responses.SuccessResponse("Sales Staff dashboard retrieved successfully", nil, result))
}

// GetOrdersOverview returns orders statistics by type and status
// @Summary Get Orders Overview
// @Description Returns orders statistics broken down by order type (STANDARD, LIMITED) and status
// @Tags Sales Staff Analytics
// @Accept json
// @Produce json
// @Param start_date query string false "Start date (RFC3339 format)" format(date-time)
// @Param end_date query string false "End date (RFC3339 format)" format(date-time)
// @Param order_type query string false "Filter by order type" Enums(STANDARD, LIMITED)
// @Success 200 {object} responses.APIResponse{data=responses.OrdersBreakdown}
// @Failure 400 {object} responses.APIResponse
// @Failure 401 {object} responses.APIResponse
// @Failure 403 {object} responses.APIResponse
// @Failure 500 {object} responses.APIResponse
// @Security BearerAuth
// @Router /api/v1/analytics/sales/orders [get]
func (h *SalesStaffAnalyticsHandler) GetOrdersOverview(c *gin.Context) {
	var req requests.OrdersOverviewRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid request parameters", http.StatusBadRequest))
		return
	}

	result, err := h.service.GetOrdersOverview(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse("Failed to get orders overview", http.StatusInternalServerError))
		return
	}

	c.JSON(http.StatusOK, responses.SuccessResponse("Orders overview retrieved successfully", nil, result))
}

// GetPreOrdersOverview returns pre-orders statistics
// @Summary Get Pre-Orders Overview
// @Description Returns pre-orders statistics including counts by status
// @Tags Sales Staff Analytics
// @Accept json
// @Produce json
// @Param start_date query string false "Start date (RFC3339 format)" format(date-time)
// @Param end_date query string false "End date (RFC3339 format)" format(date-time)
// @Success 200 {object} responses.APIResponse{data=responses.PreOrderStats}
// @Failure 400 {object} responses.APIResponse
// @Failure 401 {object} responses.APIResponse
// @Failure 403 {object} responses.APIResponse
// @Failure 500 {object} responses.APIResponse
// @Security BearerAuth
// @Router /api/v1/analytics/sales/pre-orders [get]
func (h *SalesStaffAnalyticsHandler) GetPreOrdersOverview(c *gin.Context) {
	var req requests.PreOrdersOverviewRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid request parameters", http.StatusBadRequest))
		return
	}

	result, err := h.service.GetPreOrdersOverview(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse("Failed to get pre-orders overview", http.StatusInternalServerError))
		return
	}

	c.JSON(http.StatusOK, responses.SuccessResponse("Pre-orders overview retrieved successfully", nil, result))
}

// GetRevenueBySource returns revenue breakdown by source
// @Summary Get Revenue by Source
// @Description Returns revenue breakdown by source (standard products, limited products, advertising, affiliate, ambassador, co-producing)
// @Tags Sales Staff Analytics
// @Accept json
// @Produce json
// @Param start_date query string false "Start date (RFC3339 format)" format(date-time)
// @Param end_date query string false "End date (RFC3339 format)" format(date-time)
// @Success 200 {object} responses.APIResponse{data=responses.RevenueBySource}
// @Failure 400 {object} responses.APIResponse
// @Failure 401 {object} responses.APIResponse
// @Failure 403 {object} responses.APIResponse
// @Failure 500 {object} responses.APIResponse
// @Security BearerAuth
// @Router /api/v1/analytics/sales/revenue [get]
func (h *SalesStaffAnalyticsHandler) GetRevenueBySource(c *gin.Context) {
	var req requests.RevenueBySourceRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid request parameters", http.StatusBadRequest))
		return
	}

	result, err := h.service.GetRevenueBySource(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse("Failed to get revenue by source", http.StatusInternalServerError))
		return
	}

	c.JSON(http.StatusOK, responses.SuccessResponse("Revenue by source retrieved successfully", nil, result))
}

// GetTopBrands returns top brands by revenue
// @Summary Get Top Brands
// @Description Returns top brands ranked by total revenue
// @Tags Sales Staff Analytics
// @Accept json
// @Produce json
// @Param start_date query string false "Start date (RFC3339 format)" format(date-time)
// @Param end_date query string false "End date (RFC3339 format)" format(date-time)
// @Param limit query int false "Number of results (default: 10, max: 50)" minimum(1) maximum(50) default(10)
// @Success 200 {object} responses.APIResponse{data=[]responses.BrandSalesMetric}
// @Failure 400 {object} responses.APIResponse
// @Failure 401 {object} responses.APIResponse
// @Failure 403 {object} responses.APIResponse
// @Failure 500 {object} responses.APIResponse
// @Security BearerAuth
// @Router /api/v1/analytics/sales/brands [get]
func (h *SalesStaffAnalyticsHandler) GetTopBrands(c *gin.Context) {
	var req requests.TopBrandsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid request parameters", http.StatusBadRequest))
		return
	}

	result, err := h.service.GetTopBrands(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse("Failed to get top brands", http.StatusInternalServerError))
		return
	}

	c.JSON(http.StatusOK, responses.SuccessResponse("Top brands retrieved successfully", nil, result))
}

// GetTopProducts returns top products by revenue
// @Summary Get Top Products
// @Description Returns top products ranked by revenue
// @Tags Sales Staff Analytics
// @Accept json
// @Produce json
// @Param start_date query string false "Start date (RFC3339 format)" format(date-time)
// @Param end_date query string false "End date (RFC3339 format)" format(date-time)
// @Param product_type query string false "Filter by product type" Enums(STANDARD, LIMITED)
// @Param limit query int false "Number of results (default: 10, max: 50)" minimum(1) maximum(50) default(10)
// @Success 200 {object} responses.APIResponse{data=[]responses.ProductSalesMetric}
// @Failure 400 {object} responses.APIResponse
// @Failure 401 {object} responses.APIResponse
// @Failure 403 {object} responses.APIResponse
// @Failure 500 {object} responses.APIResponse
// @Security BearerAuth
// @Router /api/v1/analytics/sales/products [get]
func (h *SalesStaffAnalyticsHandler) GetTopProducts(c *gin.Context) {
	var req requests.TopProductsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid request parameters", http.StatusBadRequest))
		return
	}

	result, err := h.service.GetTopProducts(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse("Failed to get top products", http.StatusInternalServerError))
		return
	}

	c.JSON(http.StatusOK, responses.SuccessResponse("Top products retrieved successfully", nil, result))
}

// GetRevenueTrend returns revenue time-series data
// @Summary Get Revenue Trend
// @Description Returns revenue time-series data with configurable granularity (DAY, WEEK, MONTH)
// @Tags Sales Staff Analytics
// @Accept json
// @Produce json
// @Param start_date query string false "Start date (RFC3339 format)" format(date-time)
// @Param end_date query string false "End date (RFC3339 format)" format(date-time)
// @Param granularity query string false "Time bucket granularity" Enums(DAY, WEEK, MONTH) default(DAY)
// @Success 200 {object} responses.APIResponse{data=[]responses.RevenueTrendPoint}
// @Failure 400 {object} responses.APIResponse
// @Failure 401 {object} responses.APIResponse
// @Failure 403 {object} responses.APIResponse
// @Failure 500 {object} responses.APIResponse
// @Security BearerAuth
// @Router /api/v1/analytics/sales/trend [get]
func (h *SalesStaffAnalyticsHandler) GetRevenueTrend(c *gin.Context) {
	var req requests.RevenueGrowthRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid request parameters", http.StatusBadRequest))
		return
	}

	result, err := h.service.GetRevenueTrend(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse("Failed to get revenue trend", http.StatusInternalServerError))
		return
	}

	c.JSON(http.StatusOK, responses.SuccessResponse("Revenue trend retrieved successfully", nil, result))
}

// GetPaymentStatus returns contract payment status overview
// @Summary Get Payment Status
// @Description Returns contract payment status overview with counts and amounts
// @Tags Sales Staff Analytics
// @Accept json
// @Produce json
// @Param start_date query string false "Start date (RFC3339 format)" format(date-time)
// @Param end_date query string false "End date (RFC3339 format)" format(date-time)
// @Param contract_id query string false "Filter by specific contract ID" format(uuid)
// @Success 200 {object} responses.APIResponse{data=responses.PaymentStatusOverview}
// @Failure 400 {object} responses.APIResponse
// @Failure 401 {object} responses.APIResponse
// @Failure 403 {object} responses.APIResponse
// @Failure 500 {object} responses.APIResponse
// @Security BearerAuth
// @Router /api/v1/analytics/sales/payments [get]
func (h *SalesStaffAnalyticsHandler) GetPaymentStatus(c *gin.Context) {
	var req requests.PaymentStatusRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid request parameters", http.StatusBadRequest))
		return
	}

	result, err := h.service.GetPaymentStatus(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse("Failed to get payment status", http.StatusInternalServerError))
		return
	}

	c.JSON(http.StatusOK, responses.SuccessResponse("Payment status retrieved successfully", nil, result))
}
