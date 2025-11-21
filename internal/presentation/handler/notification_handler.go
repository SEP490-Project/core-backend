package handler

import (
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/iservice"
	"core-backend/internal/domain/enum"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// NotificationHandler handles notification monitoring HTTP requests
type NotificationHandler struct {
	notificationService iservice.NotificationService
}

// NewNotificationHandler creates a new notification handler instance
func NewNotificationHandler(notificationService iservice.NotificationService) *NotificationHandler {
	return &NotificationHandler{
		notificationService: notificationService,
	}
}

// GetByID godoc
//
//	@Summary		Get notification by ID
//	@Description	Retrieve detailed information about a specific notification including delivery attempts
//	@Tags			Notifications
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string	true	"Notification ID (UUID)"
//	@Success		200	{object}	responses.APIResponse{data=responses.NotificationResponse}
//	@Failure		400	{object}	responses.APIResponse
//	@Failure		404	{object}	responses.APIResponse
//	@Failure		500	{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/notifications/{id} [get]
func (h *NotificationHandler) GetByID(c *gin.Context) {
	idParam := c.Param("id")
	notificationID, err := uuid.Parse(idParam)
	if err != nil {
		zap.L().Warn("Invalid notification ID format",
			zap.String("id", idParam))
		response := responses.ErrorResponse("Invalid notification ID format", http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	notification, err := h.notificationService.GetByID(c.Request.Context(), notificationID)
	if err != nil {
		if err.Error() == "notification not found" {
			response := responses.ErrorResponse("Notification not found", http.StatusNotFound)
			c.JSON(http.StatusNotFound, response)
			return
		}
		zap.L().Error("Failed to fetch notification", zap.Error(err))
		response := responses.ErrorResponse("Failed to fetch notification", http.StatusInternalServerError)
		c.JSON(http.StatusInternalServerError, response)
		return
	}

	notificationResponse := responses.ToNotificationResponse(notification)
	response := responses.SuccessResponse("Notification retrieved successfully", nil, notificationResponse)
	c.JSON(http.StatusOK, response)
}

// List godoc
//
//	@Summary		List notifications with filters
//	@Description	Retrieve notifications with optional filtering by user, type, status, and date range
//	@Tags			Notifications
//	@Accept			json
//	@Produce		json
//	@Param			user_id		query		string	false	"Filter by user ID (UUID)"
//	@Param			type		query		string	false	"Filter by notification type (EMAIL, PUSH)"
//	@Param			status		query		string	false	"Filter by status (PENDING, SENT, FAILED, RETRYING)"
//	@Param			start_date	query		string	false	"Filter by start date (RFC3339 or YYYY-MM-DD)"
//	@Param			end_date	query		string	false	"Filter by end date (RFC3339 or YYYY-MM-DD)"
//	@Param			page		query		int		false	"Page number (default: 1)"
//	@Param			limit		query		int		false	"Items per page (default: 20, max: 100)"
//	@Success		200			{object}	responses.APIResponse{data=responses.NotificationListResponse}
//	@Failure		400			{object}	responses.APIResponse
//	@Failure		500			{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/notifications [get]
func (h *NotificationHandler) List(c *gin.Context) {
	// Parse query parameters
	var userID *uuid.UUID
	if userIDStr := c.Query("user_id"); userIDStr != "" {
		parsedID, err := uuid.Parse(userIDStr)
		if err != nil {
			response := responses.ErrorResponse("Invalid user_id format", http.StatusBadRequest)
			c.JSON(http.StatusBadRequest, response)
			return
		}
		userID = &parsedID
	}

	var notificationType *enum.NotificationType
	if typeStr := c.Query("type"); typeStr != "" {
		nType := enum.NotificationType(typeStr)
		if !nType.IsValid() {
			response := responses.ErrorResponse("Invalid notification type", http.StatusBadRequest)
			c.JSON(http.StatusBadRequest, response)
			return
		}
		notificationType = &nType
	}

	var status *enum.NotificationStatus
	if statusStr := c.Query("status"); statusStr != "" {
		nStatus := enum.NotificationStatus(statusStr)
		if !nStatus.IsValid() {
			response := responses.ErrorResponse("Invalid notification status", http.StatusBadRequest)
			c.JSON(http.StatusBadRequest, response)
			return
		}
		status = &nStatus
	}

	var startDate, endDate *string
	if startDateStr := c.Query("start_date"); startDateStr != "" {
		startDate = &startDateStr
	}
	if endDateStr := c.Query("end_date"); endDateStr != "" {
		endDate = &endDateStr
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	notifications, total, err := h.notificationService.GetByFilters(
		c.Request.Context(),
		userID,
		notificationType,
		status,
		startDate,
		endDate,
		page,
		limit,
	)

	if err != nil {
		zap.L().Error("Failed to fetch notifications", zap.Error(err))
		response := responses.ErrorResponse(err.Error(), http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	listResponse := responses.ToNotificationListResponse(notifications, page, limit, total)
	response := responses.SuccessResponse("Notifications retrieved successfully", nil, listResponse)
	c.JSON(http.StatusOK, response)
}

// GetFailedNotifications godoc
//
//	@Summary		Get failed notifications with retries
//	@Description	Retrieve notifications that failed after multiple retry attempts
//	@Tags			Notifications
//	@Accept			json
//	@Produce		json
//	@Param			min_retries	query		int	false	"Minimum number of retry attempts (default: 3)"
//	@Param			page		query		int	false	"Page number (default: 1)"
//	@Param			limit		query		int	false	"Items per page (default: 20, max: 100)"
//	@Success		200			{object}	responses.APIResponse{data=responses.NotificationListResponse}
//	@Failure		400			{object}	responses.APIResponse
//	@Failure		500			{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/notifications/failed [get]
func (h *NotificationHandler) GetFailedNotifications(c *gin.Context) {
	minRetries, _ := strconv.Atoi(c.DefaultQuery("min_retries", "3"))
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	notifications, total, err := h.notificationService.GetFailedWithRetries(
		c.Request.Context(),
		minRetries,
		page,
		limit,
	)

	if err != nil {
		zap.L().Error("Failed to fetch failed notifications", zap.Error(err))
		response := responses.ErrorResponse("Failed to fetch failed notifications", http.StatusInternalServerError)
		c.JSON(http.StatusInternalServerError, response)
		return
	}

	listResponse := responses.ToNotificationListResponse(notifications, page, limit, total)
	response := responses.SuccessResponse("Failed notifications retrieved successfully", nil, listResponse)
	c.JSON(http.StatusOK, response)
}

// PublishNotification godoc
//
//	@Summary		Publish notification to multiple channels
//	@Description	Create and publish a notification to one or many channels (EMAIL, PUSH). Admin only.
//	@Tags			Notifications
//	@Accept			json
//	@Produce		json
//	@Param			request	body		requests.PublishNotificationRequest		true	"Notification data"
//	@Success		201		{object}	responses.APIResponse{data=[]string}	"Returns array of notification IDs"
//	@Failure		400		{object}	responses.APIResponse
//	@Failure		500		{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/notifications/publish [post]
func (h *NotificationHandler) PublishNotification(c *gin.Context) {
	var req requests.PublishNotificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		zap.L().Warn("Invalid request body for publish notification", zap.Error(err))
		response := responses.ErrorResponse("Invalid request body: "+err.Error(), http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	notificationIDs, err := h.notificationService.CreateAndPublishNotification(c.Request.Context(), &req)
	if err != nil {
		zap.L().Error("Failed to publish notification", zap.Error(err))
		response := responses.ErrorResponse(err.Error(), http.StatusInternalServerError)
		c.JSON(http.StatusInternalServerError, response)
		return
	}

	// Convert UUIDs to strings for response
	idStrings := make([]string, len(notificationIDs))
	for i, id := range notificationIDs {
		idStrings[i] = id.String()
	}

	statusCode := http.StatusCreated
	response := responses.SuccessResponse("Notifications published successfully", &statusCode, map[string]any{
		"notification_ids": idStrings,
		"count":            len(idStrings),
	})
	c.JSON(http.StatusCreated, response)
}

// PublishEmail godoc
//
//	@Summary		Publish email notification
//	@Description	Create and publish an email notification. Supports template or HTML body. Admin only.
//	@Tags			Notifications
//	@Accept			json
//	@Produce		json
//	@Param			request	body		requests.PublishEmailRequest					true	"Email notification data"
//	@Success		201		{object}	responses.APIResponse{data=map[string]string}	"Returns notification_id"
//	@Failure		400		{object}	responses.APIResponse
//	@Failure		500		{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/notifications/publish/email [post]
func (h *NotificationHandler) PublishEmail(c *gin.Context) {
	var req requests.PublishEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		zap.L().Warn("Invalid request body for publish email", zap.Error(err))
		response := responses.ErrorResponse("Invalid request body: "+err.Error(), http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	notificationID, err := h.notificationService.CreateAndPublishEmail(c.Request.Context(), &req)
	if err != nil {
		zap.L().Error("Failed to publish email notification", zap.Error(err))
		response := responses.ErrorResponse(err.Error(), http.StatusInternalServerError)
		c.JSON(http.StatusInternalServerError, response)
		return
	}

	statusCode := http.StatusCreated
	response := responses.SuccessResponse("Email notification published successfully", &statusCode, map[string]any{
		"notification_id": notificationID.String(),
	})
	c.JSON(http.StatusCreated, response)
}

// PublishPush godoc
//
//	@Summary		Publish push notification
//	@Description	Create and publish a push notification to user's registered devices. Admin only.
//	@Tags			Notifications
//	@Accept			json
//	@Produce		json
//	@Param			request	body		requests.PublishPushRequest						true	"Push notification data"
//	@Success		201		{object}	responses.APIResponse{data=map[string]string}	"Returns notification_id"
//	@Failure		400		{object}	responses.APIResponse
//	@Failure		500		{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/notifications/publish/push [post]
func (h *NotificationHandler) PublishPush(c *gin.Context) {
	var req requests.PublishPushRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		zap.L().Warn("Invalid request body for publish push", zap.Error(err))
		response := responses.ErrorResponse("Invalid request body: "+err.Error(), http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	notificationID, err := h.notificationService.CreateAndPublishPush(c.Request.Context(), &req)
	if err != nil {
		zap.L().Error("Failed to publish push notification", zap.Error(err))
		response := responses.ErrorResponse(err.Error(), http.StatusInternalServerError)
		c.JSON(http.StatusInternalServerError, response)
		return
	}

	statusCode := http.StatusCreated
	response := responses.SuccessResponse("Push notification published successfully", &statusCode, map[string]any{
		"notification_id": notificationID.String(),
	})
	c.JSON(http.StatusCreated, response)
}

// RepublishFailed godoc
//
//	@Summary		Republish failed notifications
//	@Description	Retry sending failed notifications based on filter criteria. Admin only.
//	@Tags			Notifications
//	@Accept			json
//	@Produce		json
//	@Param			request	body		requests.RepublishFailedNotificationRequest	true	"Filter criteria for failed notifications"
//	@Success		200		{object}	responses.APIResponse{data=map[string]int}	"Returns success_count"
//	@Failure		400		{object}	responses.APIResponse
//	@Failure		500		{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/notifications/republish-failed [post]
func (h *NotificationHandler) RepublishFailed(c *gin.Context) {
	var req requests.RepublishFailedNotificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		zap.L().Warn("Invalid request body for republish failed", zap.Error(err))
		response := responses.ErrorResponse("Invalid request body: "+err.Error(), http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	successCount, err := h.notificationService.RepublishFailedNotifications(c.Request.Context(), &req)
	if err != nil {
		zap.L().Error("Failed to republish notifications", zap.Error(err))
		response := responses.ErrorResponse(err.Error(), http.StatusInternalServerError)
		c.JSON(http.StatusInternalServerError, response)
		return
	}

	response := responses.SuccessResponse("Failed notifications republished successfully", nil, map[string]any{
		"success_count": successCount,
	})
	c.JSON(http.StatusOK, response)
}
