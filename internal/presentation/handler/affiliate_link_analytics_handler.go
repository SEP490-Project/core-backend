package handler

import (
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/iservice"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type AffiliateLinkAnalyticsHandler struct {
	analyticsService iservice.AffiliateLinkAnalyticsService
}

func NewAffiliateLinkAnalyticsHandler(analyticsService iservice.AffiliateLinkAnalyticsService) *AffiliateLinkAnalyticsHandler {
	return &AffiliateLinkAnalyticsHandler{
		analyticsService: analyticsService,
	}
}

// GetMetricsByContract godoc
//
//	@Summary		Get analytics metrics for a specific contract
//	@Description	Retrieves click metrics, CTR, top channels, and top links for a contract
//	@Tags			Analytics/AffiliateLinks
//	@Accept			json
//	@Produce		json
//	@Param			contract_id	path		string	true	"Contract ID"			format(uuid)
//	@Param			start_date	query		string	false	"Start date (ISO 8601)"	format(date-time)
//	@Param			end_date	query		string	false	"End date (ISO 8601)"	format(date-time)
//	@Success		200			{object}	responses.APIResponse{data=responses.ContractMetricsResponse}
//	@Failure		400			{object}	responses.APIResponse
//	@Failure		403			{object}	responses.APIResponse
//	@Failure		404			{object}	responses.APIResponse
//	@Failure		500			{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/analytics/affiliate-links/by-contract/{contract_id} [get]
func (h *AffiliateLinkAnalyticsHandler) GetMetricsByContract(c *gin.Context) {
	// Parse contract ID from path
	contractIDStr := c.Param("contract_id")
	contractID, err := uuid.Parse(contractIDStr)
	if err != nil {
		response := responses.ErrorResponse("Invalid contract ID", http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	// Get user ID from context for access validation
	userIDVal, exists := c.Get("user_id")
	if !exists {
		response := responses.ErrorResponse("User not authenticated", http.StatusUnauthorized)
		c.JSON(http.StatusUnauthorized, response)
		return
	}
	userID := userIDVal.(uuid.UUID)

	// Validate contract access (for BRAND_PARTNER role)
	if err = h.analyticsService.ValidateContractAccess(c.Request.Context(), userID, contractID); err != nil {
		zap.L().Warn("Contract access denied", zap.Error(err), zap.String("user_id", userID.String()))
		response := responses.ErrorResponse("Access denied to this contract's analytics", http.StatusForbidden)
		c.JSON(http.StatusForbidden, response)
		return
	}

	// Bind query parameters
	var req requests.ContractMetricsRequest
	req.ContractID = contractID
	if err = c.ShouldBindQuery(&req); err != nil {
		response := responses.ErrorResponse("Invalid query parameters", http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	// Get metrics
	metrics, err := h.analyticsService.GetMetricsByContract(c.Request.Context(), &req)
	if err != nil {
		zap.L().Error("Failed to get contract metrics", zap.Error(err))
		response := responses.ErrorResponse("Failed to retrieve contract metrics", http.StatusInternalServerError)
		c.JSON(http.StatusInternalServerError, response)
		return
	}

	response := responses.SuccessResponse("Contract metrics retrieved successfully", nil, metrics)
	c.JSON(http.StatusOK, response)
}

// GetMetricsByChannel godoc
//
//	@Summary		Get analytics metrics grouped by channel
//	@Description	Retrieves aggregated metrics for all channels with comparison data
//	@Tags			Analytics/AffiliateLinks
//	@Accept			json
//	@Produce		json
//	@Param			start_date	query		string	false	"Start date (ISO 8601)"	format(date-time)
//	@Param			end_date	query		string	false	"End date (ISO 8601)"	format(date-time)
//	@Success		200			{object}	responses.APIResponse{data=responses.ChannelMetricsResponse}
//	@Failure		400			{object}	responses.APIResponse
//	@Failure		500			{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/analytics/affiliate-links/by-channel [get]
func (h *AffiliateLinkAnalyticsHandler) GetMetricsByChannel(c *gin.Context) {
	// Bind query parameters
	var req requests.ChannelMetricsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response := responses.ErrorResponse("Invalid query parameters", http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	// Get metrics
	metrics, err := h.analyticsService.GetMetricsByChannel(c.Request.Context(), &req)
	if err != nil {
		zap.L().Error("Failed to get channel metrics", zap.Error(err))
		response := responses.ErrorResponse("Failed to retrieve channel metrics", http.StatusInternalServerError)
		c.JSON(http.StatusInternalServerError, response)
		return
	}

	response := responses.SuccessResponse("Channel metrics retrieved successfully", nil, metrics)
	c.JSON(http.StatusOK, response)
}

// GetTimeSeriesData godoc
//
//	@Summary		Get time-series data for a specific affiliate link
//	@Description	Retrieves time-bucketed click data for trend analysis
//	@Tags			Analytics/AffiliateLinks
//	@Accept			json
//	@Produce		json
//	@Param			affiliate_link_id	path		string	true	"Affiliate Link ID"		format(uuid)
//	@Param			start_date			query		string	false	"Start date (ISO 8601)"	format(date-time)
//	@Param			end_date			query		string	false	"End date (ISO 8601)"	format(date-time)
//	@Param			granularity			query		string	false	"Time bucket size"		Enums(HOUR, DAY, WEEK, MONTH)	default(DAY)
//	@Success		200					{object}	responses.APIResponse{data=responses.TimeSeriesDataResponse}
//	@Failure		400					{object}	responses.APIResponse
//	@Failure		404					{object}	responses.APIResponse
//	@Failure		500					{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/analytics/affiliate-links/time-series/{affiliate_link_id} [get]
func (h *AffiliateLinkAnalyticsHandler) GetTimeSeriesData(c *gin.Context) {
	// Parse affiliate link ID from path
	linkIDStr := c.Param("affiliate_link_id")
	linkID, err := uuid.Parse(linkIDStr)
	if err != nil {
		response := responses.ErrorResponse("Invalid affiliate link ID", http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	// Bind query parameters
	var req requests.TimeSeriesRequest
	req.AffiliateLinkID = linkID
	if err = c.ShouldBindQuery(&req); err != nil {
		response := responses.ErrorResponse("Invalid query parameters", http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	// Get time series data
	data, err := h.analyticsService.GetTimeSeriesData(c.Request.Context(), &req)
	if err != nil {
		zap.L().Error("Failed to get time series data", zap.Error(err))
		response := responses.ErrorResponse("Failed to retrieve time series data", http.StatusInternalServerError)
		c.JSON(http.StatusInternalServerError, response)
		return
	}

	response := responses.SuccessResponse("Time series data retrieved successfully", nil, data)
	c.JSON(http.StatusOK, response)
}

// GetTopPerformers godoc
//
//	@Summary		Get top performing affiliate links
//	@Description	Retrieves ranked list of best performing links based on sorting criteria
//	@Tags			Analytics/AffiliateLinks
//	@Accept			json
//	@Produce		json
//	@Param			start_date	query		string	false	"Start date (ISO 8601)"	format(date-time)
//	@Param			end_date	query		string	false	"End date (ISO 8601)"	format(date-time)
//	@Param			sort_by		query		string	false	"Sort criteria"			Enums(CLICKS, CTR, ENGAGEMENT)	default(CLICKS)
//	@Param			limit		query		int		false	"Number of results"		minimum(1)						maximum(50)	default(10)
//	@Success		200			{object}	responses.APIResponse{data=responses.TopPerformerResponse}
//	@Failure		400			{object}	responses.APIResponse
//	@Failure		500			{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/analytics/affiliate-links/top-performers [get]
func (h *AffiliateLinkAnalyticsHandler) GetTopPerformers(c *gin.Context) {
	// Bind query parameters
	var req requests.TopPerformersRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response := responses.ErrorResponse("Invalid query parameters", http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	// Get top performers
	performers, err := h.analyticsService.GetTopPerformers(c.Request.Context(), &req)
	if err != nil {
		zap.L().Error("Failed to get top performers", zap.Error(err))
		response := responses.ErrorResponse("Failed to retrieve top performers", http.StatusInternalServerError)
		c.JSON(http.StatusInternalServerError, response)
		return
	}

	response := responses.SuccessResponse("Top performers retrieved successfully", nil, performers)
	c.JSON(http.StatusOK, response)
}

// GetDashboard godoc
//
//	@Summary		Get dashboard metrics with parallel aggregation
//	@Description	Retrieves overview metrics, top contracts, channels, recent activity, and trends
//	@Tags			Analytics/AffiliateLinks
//	@Accept			json
//	@Produce		json
//	@Param			start_date	query		string	false	"Start date (ISO 8601)"	format(date-time)
//	@Param			end_date	query		string	false	"End date (ISO 8601)"	format(date-time)
//	@Success		200			{object}	responses.APIResponse{data=responses.DashboardMetricsResponse}
//	@Failure		400			{object}	responses.APIResponse
//	@Failure		500			{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/analytics/affiliate-links/dashboard [get]
func (h *AffiliateLinkAnalyticsHandler) GetDashboard(c *gin.Context) {
	// Bind query parameters
	var req requests.DashboardRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response := responses.ErrorResponse("Invalid query parameters", http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	// Get dashboard metrics
	dashboard, err := h.analyticsService.GetDashboardMetrics(c.Request.Context(), &req)
	if err != nil {
		zap.L().Error("Failed to get dashboard metrics", zap.Error(err))
		response := responses.ErrorResponse("Failed to retrieve dashboard metrics", http.StatusInternalServerError)
		c.JSON(http.StatusInternalServerError, response)
		return
	}

	response := responses.SuccessResponse("Dashboard metrics retrieved successfully", nil, dashboard)
	c.JSON(http.StatusOK, response)
}
