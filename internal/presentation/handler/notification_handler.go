package handler

import (
	"context"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/iservice"
	"core-backend/internal/domain/enum"
	"net/http"
	"strconv"
	"time"

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
//	@Param			type		query		string	false	"Filter by notification type (EMAIL, PUSH, IN_APP)"
//	@Param			status		query		string	false	"Filter by status (PENDING, SENT, FAILED, RETRYING)"
//	@Param			is_read		query		bool	false	"Filter by read status"
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
	userIDStr := c.Query("user_id")
	typeStr := c.Query("type")
	statusStr := c.Query("status")
	isReadStr := c.Query("is_read")
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	var userID *uuid.UUID
	if userIDStr != "" {
		id, err := uuid.Parse(userIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid user ID format", http.StatusBadRequest))
			return
		}
		userID = &id
	}

	var notificationType *enum.NotificationType
	if typeStr != "" {
		nt := enum.NotificationType(typeStr)
		notificationType = &nt
	}

	var status *enum.NotificationStatus
	if statusStr != "" {
		s := enum.NotificationStatus(statusStr)
		status = &s
	}

	var isRead *bool
	if isReadStr != "" {
		val, err := strconv.ParseBool(isReadStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid is_read format", http.StatusBadRequest))
			return
		}
		isRead = &val
	}

	var startDatePtr, endDatePtr *string
	if startDate != "" {
		startDatePtr = &startDate
	}
	if endDate != "" {
		endDatePtr = &endDate
	}

	notifications, total, err := h.notificationService.GetByFilters(
		c.Request.Context(),
		userID,
		notificationType,
		status,
		isRead,
		startDatePtr,
		endDatePtr,
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
	idStrings := make([]string, 0, len(notificationIDs))
	for _, id := range notificationIDs {
		idStrings = append(idStrings, id.String())
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

// SubscribeSSE godoc
//
//	@Summary		Subscribe to real-time notifications (SSE)
//	@Description	Establishes a Server-Sent Events connection to receive real-time updates (e.g., unread count)
//	@Tags			Notifications
//	@Produce		text/event-stream
//	@Security		BearerAuth
//	@Router			/api/v1/notifications/sse [get]
func (h *NotificationHandler) SubscribeSSE(c *gin.Context) {
	/* userIDStr := c.GetString("user_id")
	if userIDStr == "" {
		c.AbortWithStatus(http.StatusUnauthorized)
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}*/

	userID, err := extractUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, responses.ErrorResponse("Invalid user ID", http.StatusUnauthorized))
		return
	}

	// Set headers for SSE
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("Transfer-Encoding", "chunked")
	c.Writer.Header().Set("Access-Control-Allow-Origin", "*")

	clientChan, unsubscribe := h.notificationService.SubscribeSSE(userID)
	defer unsubscribe()

	// Send initial connection message
	c.SSEvent("connected", "Connected to notification stream")
	c.Writer.Flush()

	// Create a ticker for heartbeats (keep-alive)
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	// Listen for messages
	ctx := c.Request.Context()
	for {
		select {
		case msg := <-clientChan:
			c.SSEvent(msg.Event, msg.Data)
			c.Writer.Flush()
		case <-ticker.C:
			c.SSEvent("heartbeat", "ping")
			c.Writer.Flush()
		case <-ctx.Done():
			return
		}
	}
}

// MarkAsRead godoc
//
//	@Summary		Mark notification as read
//	@Description	Marks a specific notification as read
//	@Tags			Notifications
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string	true	"Notification ID (UUID)"
//	@Success		200	{object}	responses.APIResponse
//	@Failure		400	{object}	responses.APIResponse
//	@Failure		404	{object}	responses.APIResponse
//	@Failure		500	{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/notifications/{id}/read [put]
func (h *NotificationHandler) MarkAsRead(c *gin.Context) {
	idParam := c.Param("id")
	notificationID, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid notification ID", http.StatusBadRequest))
		return
	}

	userIDStr := c.GetString("user_id")
	userID, _ := uuid.Parse(userIDStr)

	if err := h.notificationService.MarkAsRead(c.Request.Context(), notificationID, userID); err != nil {
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse(err.Error(), http.StatusInternalServerError))
		return
	}

	c.JSON(http.StatusOK, responses.SuccessResponse("Notification marked as read", nil, nil))
}

// MarkAllAsRead godoc
//
//	@Summary		Mark all notifications as read
//	@Description	Marks all notifications for the current user as read
//	@Tags			Notifications
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	responses.APIResponse
//	@Failure		500	{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/notifications/read-all [put]
func (h *NotificationHandler) MarkAllAsRead(c *gin.Context) {
	userIDStr := c.GetString("user_id")
	userID, _ := uuid.Parse(userIDStr)

	if err := h.notificationService.MarkAllAsRead(c.Request.Context(), userID); err != nil {
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse(err.Error(), http.StatusInternalServerError))
		return
	}

	c.JSON(http.StatusOK, responses.SuccessResponse("All notifications marked as read", nil, nil))
}

// BroadcastToUser godoc
//
//	@Summary		Broadcast notification to a specific user
//	@Description	Sends a unified notification to a specific user across specified channels
//	@Tags			Notifications
//	@Accept			json
//	@Produce		json
//	@Param			request	body		requests.BroadcastToUserRequest	true	"Broadcast Request"
//	@Success		200		{object}	responses.APIResponse
//	@Failure		400		{object}	responses.APIResponse
//	@Failure		500		{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/notifications/broadcast/user [post]
func (h *NotificationHandler) BroadcastToUser(c *gin.Context) {
	var req requests.BroadcastToUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid request body", http.StatusBadRequest))
		return
	}

	if err := h.notificationService.BroadcastToUser(c.Request.Context(), &req); err != nil {
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse(err.Error(), http.StatusInternalServerError))
		return
	}

	c.JSON(http.StatusOK, responses.SuccessResponse("Notification broadcasted to user", nil, nil))
}

// BroadcastToAll godoc
//
//	@Summary		Broadcast notification to all users
//	@Description	Sends a unified notification to all users (optionally filtered by role)
//	@Tags			Notifications
//	@Accept			json
//	@Produce		json
//	@Param			request	body		requests.BroadcastToAllRequest	true	"Broadcast Request"
//	@Success		200		{object}	responses.APIResponse
//	@Failure		400		{object}	responses.APIResponse
//	@Failure		500		{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/notifications/broadcast/all [post]
func (h *NotificationHandler) BroadcastToAll(c *gin.Context) {
	var req requests.BroadcastToAllRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid request body", http.StatusBadRequest))
		return
	}

	// This operation can be long-running, so we should probably run it asynchronously
	// and return 202 Accepted.
	go func() {
		// Create a background context since the request context will be cancelled
		ctx := context.Background()
		if err := h.notificationService.BroadcastToAll(ctx, &req); err != nil {
			zap.L().Error("Failed to broadcast to all users", zap.Error(err))
		}
	}()

	c.JSON(http.StatusAccepted, responses.SuccessResponse("Broadcast started", nil, nil))
}

// GetUnreadCount godoc
//
//	@Summary		Get unread notification count
//	@Description	Retrieve the count of unread notifications for the current user
//	@Tags			Notifications
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	responses.APIResponse{data=map[string]int64}
//	@Failure		400	{object}	responses.APIResponse
//	@Failure		500	{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/notifications/unread-count [get]
func (h *NotificationHandler) GetUnreadCount(c *gin.Context) {
	userID, err := extractUserID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid user ID", http.StatusBadRequest))
		return
	}

	unreadCount, err := h.notificationService.GetUnreadCount(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse(err.Error(), http.StatusInternalServerError))
		return
	}

	response := responses.SuccessResponse("Unread count retrieved successfully", nil, map[string]any{
		"unread_count": unreadCount,
	})
	c.JSON(http.StatusOK, response)
}
