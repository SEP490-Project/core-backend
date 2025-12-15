package handler

import (
	"net/http"

	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/iservice"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// AlertHandler handles alert management endpoints
type AlertHandler struct {
	alertService iservice.AlertManagerService
}

// NewAlertHandler creates a new AlertHandler
func NewAlertHandler(alertService iservice.AlertManagerService) *AlertHandler {
	return &AlertHandler{
		alertService: alertService,
	}
}

// GetAlerts godoc
//
//	@Summary		Get system alerts
//	@Description	Returns list of system alerts with filtering
//	@Tags			Alerts
//	@Accept			json
//	@Produce		json
//	@Param			category		query		string	false	"Filter by category (LOW_CTR_WARNING, MILESTONE_DEADLINE, etc.)"
//	@Param			severity		query		string	false	"Filter by severity (INFO, WARNING, CRITICAL)"
//	@Param			status			query		string	false	"Filter by status (ACTIVE, ACKNOWLEDGED, RESOLVED)"
//	@Param			reference_type	query		string	false	"Filter by reference type"
//	@Param			reference_id	query		string	false	"Filter by reference ID"
//	@Param			from			query		string	false	"Filter from date (RFC3339)"
//	@Param			to				query		string	false	"Filter until date (RFC3339)"
//	@Param			page			query		int		false	"Page number"		default(1)
//	@Param			limit			query		int		false	"Items per page"	default(20)
//	@Success		200				{object}	responses.APIResponse{data=responses.AlertsResponse}
//	@Failure		400				{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/alerts [get]
func (h *AlertHandler) GetAlerts(c *gin.Context) {
	var filter requests.AlertFilterRequest
	if err := c.ShouldBindQuery(&filter); err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid filter parameters: "+err.Error(), http.StatusBadRequest))
		return
	}

	result, _, err := h.alertService.GetAlertsWithPagination(c.Request.Context(), &filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse("Failed to get alerts: "+err.Error(), http.StatusInternalServerError))
		return
	}

	c.JSON(http.StatusOK, responses.SuccessResponse("Alerts retrieved", nil, result))
}

// GetAlert godoc
//
//	@Summary		Get alert details
//	@Description	Returns details of a specific alert including acknowledgments
//	@Tags			Alerts
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string	true	"Alert ID"
//	@Success		200	{object}	responses.APIResponse{data=responses.AlertResponse}
//	@Failure		400	{object}	responses.APIResponse
//	@Failure		404	{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/alerts/{id} [get]
func (h *AlertHandler) GetAlert(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid alert ID", http.StatusBadRequest))
		return
	}

	alert, err := h.alertService.GetAlert(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, responses.ErrorResponse("Alert not found: "+err.Error(), http.StatusNotFound))
		return
	}

	c.JSON(http.StatusOK, responses.SuccessResponse("Alert retrieved", nil, alert))
}

// AcknowledgeAlert godoc
//
//	@Summary		Acknowledge an alert
//	@Description	Marks an alert as acknowledged by the current user
//	@Tags			Alerts
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string								true	"Alert ID"
//	@Param			request	body		requests.AcknowledgeAlertRequest	false	"Acknowledgment details (optional)"
//	@Success		200		{object}	responses.APIResponse
//	@Failure		400		{object}	responses.APIResponse
//	@Failure		401		{object}	responses.APIResponse
//	@Failure		404		{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/alerts/{id}/acknowledge [post]
func (h *AlertHandler) AcknowledgeAlert(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid alert ID", http.StatusBadRequest))
		return
	}

	// Get user ID from context
	userIDStr, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, responses.ErrorResponse("User not authenticated", http.StatusUnauthorized))
		return
	}
	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid user ID", http.StatusBadRequest))
		return
	}

	var req requests.AcknowledgeAlertRequest
	_ = c.ShouldBindJSON(&req) // Notes is optional, ignore binding errors

	if err := h.alertService.AcknowledgeAlert(c.Request.Context(), id, userID, req.Notes); err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Failed to acknowledge alert: "+err.Error(), http.StatusBadRequest))
		return
	}

	c.JSON(http.StatusOK, responses.SuccessResponse("Alert acknowledged", nil, nil))
}

// ResolveAlert godoc
//
//	@Summary		Resolve an alert
//	@Description	Marks an alert as resolved
//	@Tags			Alerts
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string							true	"Alert ID"
//	@Param			request	body		requests.ResolveAlertRequest	false	"Resolution details (optional)"
//	@Success		200		{object}	responses.APIResponse
//	@Failure		400		{object}	responses.APIResponse
//	@Failure		401		{object}	responses.APIResponse
//	@Failure		404		{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/alerts/{id}/resolve [post]
func (h *AlertHandler) ResolveAlert(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid alert ID", http.StatusBadRequest))
		return
	}

	// Get user ID from context
	userIDStr, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, responses.ErrorResponse("User not authenticated", http.StatusUnauthorized))
		return
	}
	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid user ID", http.StatusBadRequest))
		return
	}

	var req requests.ResolveAlertRequest
	_ = c.ShouldBindJSON(&req) // Resolution is optional

	if err := h.alertService.ResolveAlert(c.Request.Context(), id, userID, req.Resolution); err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Failed to resolve alert: "+err.Error(), http.StatusBadRequest))
		return
	}

	c.JSON(http.StatusOK, responses.SuccessResponse("Alert resolved", nil, nil))
}

// GetAlertStats godoc
//
//	@Summary		Get alert statistics
//	@Description	Returns counts of alerts by status, severity, and category
//	@Tags			Alerts
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	responses.APIResponse{data=responses.AlertStatsResponse}
//	@Failure		500	{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/alerts/stats [get]
func (h *AlertHandler) GetAlertStats(c *gin.Context) {
	stats, err := h.alertService.GetAlertStats(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse("Failed to get alert stats: "+err.Error(), http.StatusInternalServerError))
		return
	}

	c.JSON(http.StatusOK, responses.SuccessResponse("Alert stats retrieved", nil, stats))
}

// GetUnacknowledgedCount godoc
//
//	@Summary		Get unacknowledged alert count
//	@Description	Returns count of alerts that haven't been acknowledged yet
//	@Tags			Alerts
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	responses.APIResponse{data=int}
//	@Failure		500	{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/alerts/unacknowledged-count [get]
func (h *AlertHandler) GetUnacknowledgedCount(c *gin.Context) {
	count, err := h.alertService.GetUnacknowledgedCount(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse("Failed to get unacknowledged count: "+err.Error(), http.StatusInternalServerError))
		return
	}

	c.JSON(http.StatusOK, responses.SuccessResponse("Unacknowledged count retrieved", nil, count))
}

// RaiseAlert godoc
//
//	@Summary		Raise a new alert
//	@Description	Creates a new system alert
//	@Tags			Alerts
//	@Accept			json
//	@Produce		json
//	@Param			request	body		requests.RaiseAlertRequest	true	"Alert details"
//	@Success		200		{object}	responses.APIResponse{data=responses.AlertResponse}
//	@Failure		400		{object}	responses.APIResponse
//	@Failure		500		{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/alerts [post]
func (h *AlertHandler) RaiseAlert(c *gin.Context) {
	var req requests.RaiseAlertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid request: "+err.Error(), http.StatusBadRequest))
		return
	}

	alert, err := h.alertService.RaiseAlert(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse("Failed to raise alert: "+err.Error(), http.StatusInternalServerError))
		return
	}

	c.JSON(http.StatusOK, responses.SuccessResponse("Alert raised", nil, alert))
}
