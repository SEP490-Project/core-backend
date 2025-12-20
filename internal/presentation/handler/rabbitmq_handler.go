package handler

import (
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/infrastructure/rabbitmq"
	"net/http"
	"slices"
	"strings"

	"github.com/gin-gonic/gin"
)

// RabbitMQHandler handles RabbitMQ management API endpoints
type RabbitMQHandler struct {
	managementService *rabbitmq.ManagementService
	rabbitmq          *rabbitmq.RabbitMQ
}

// NewRabbitMQHandler creates a new RabbitMQ handler
func NewRabbitMQHandler(managementService *rabbitmq.ManagementService, rmq *rabbitmq.RabbitMQ) *RabbitMQHandler {
	return &RabbitMQHandler{
		managementService: managementService,
		rabbitmq:          rmq,
	}
}

// GetOverview godoc
//
//	@Summary		Get RabbitMQ Overview
//	@Description	Returns an overview of RabbitMQ queues, exchanges, and message counts
//	@Tags			RabbitMQ Admin
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	responses.APIResponse{data=responses.RabbitMQOverviewResponse}
//	@Failure		401	{object}	responses.APIResponse
//	@Failure		403	{object}	responses.APIResponse
//	@Failure		500	{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/admin/rabbitmq/overview [get]
func (h *RabbitMQHandler) GetOverview(c *gin.Context) {
	ctx := c.Request.Context()

	if h.managementService == nil {
		c.JSON(http.StatusServiceUnavailable, responses.ErrorResponse("RabbitMQ management service not available", http.StatusServiceUnavailable))
		return
	}

	// Get all queues
	queues, err := h.managementService.ListQueues(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse("Failed to list queues: "+err.Error(), http.StatusInternalServerError))
		return
	}

	// Get all exchanges
	exchanges, err := h.managementService.ListExchanges(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse("Failed to list exchanges: "+err.Error(), http.StatusInternalServerError))
		return
	}

	// Calculate statistics
	var totalMessages, totalReady, totalUnacked, totalDLQMessages int64
	var mainQueues, retryQueues, dlqQueues, delayedQueues int

	for _, q := range queues {
		totalMessages += q.Messages
		totalReady += q.MessagesReady
		totalUnacked += q.MessagesUnacked

		qType := h.managementService.ClassifyQueue(q.Name)
		switch qType {
		case "main":
			mainQueues++
		case "retry":
			retryQueues++
		case "dead_letter":
			dlqQueues++
			totalDLQMessages += q.Messages
		case "delayed":
			delayedQueues++
		}
	}

	// Get connection status
	connectionStatus := "disconnected"
	if h.rabbitmq != nil && h.rabbitmq.IsConnected() {
		connectionStatus = "connected"
	}

	// Build topology summary from config
	var topologySummary *responses.RabbitMQTopologySummary
	if topologyConfig := h.managementService.GetTopologyConfig(); topologyConfig != nil {
		var exchangeNames []string
		var queueCount int
		for _, ex := range topologyConfig.Exchanges {
			exchangeNames = append(exchangeNames, ex.Name)
			queueCount += len(ex.Queues)
		}
		topologySummary = &responses.RabbitMQTopologySummary{
			ExchangeCount: len(topologyConfig.Exchanges),
			QueueCount:    queueCount,
			ExchangeNames: exchangeNames,
		}
	}

	overview := responses.RabbitMQOverviewResponse{
		TotalQueues:          len(queues),
		TotalExchanges:       len(exchanges),
		TotalMessages:        totalMessages,
		TotalMessagesReady:   totalReady,
		TotalMessagesUnacked: totalUnacked,
		ConnectionStatus:     connectionStatus,
		QueueSummary: responses.RabbitMQQueueSummary{
			MainQueues:       mainQueues,
			RetryQueues:      retryQueues,
			DeadLetterQueues: dlqQueues,
			DelayedQueues:    delayedQueues,
			TotalDLQMessages: totalDLQMessages,
		},
		ConfiguredTopology: topologySummary,
	}

	c.JSON(http.StatusOK, responses.SuccessResponse("RabbitMQ overview retrieved successfully", nil, overview))
}

// ListQueues godoc
//
//	@Summary		List RabbitMQ Queues
//	@Description	Returns all RabbitMQ queues with their message counts and status
//	@Tags			RabbitMQ Admin
//	@Accept			json
//	@Produce		json
//	@Param			type			query		string	false	"Filter by queue type (main, retry, dead_letter, delayed, all)"
//	@Param			search			query		string	false	"Search by queue name"
//	@Param			has_messages	query		bool	false	"Filter queues with messages"
//	@Success		200				{object}	responses.APIResponse{data=[]responses.RabbitMQQueueResponse}
//	@Failure		401				{object}	responses.APIResponse
//	@Failure		403				{object}	responses.APIResponse
//	@Failure		500				{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/admin/rabbitmq/queues [get]
func (h *RabbitMQHandler) ListQueues(c *gin.Context) {
	ctx := c.Request.Context()

	if h.managementService == nil {
		c.JSON(http.StatusServiceUnavailable, responses.ErrorResponse("RabbitMQ management service not available", http.StatusServiceUnavailable))
		return
	}

	var filter requests.RabbitMQQueueFilterRequest
	if err := c.ShouldBindQuery(&filter); err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid filter parameters: "+err.Error(), http.StatusBadRequest))
		return
	}

	queues, err := h.managementService.ListQueues(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse("Failed to list queues: "+err.Error(), http.StatusInternalServerError))
		return
	}

	// Build a map for quick lookup of related queues
	queueMap := make(map[string]*rabbitmq.QueueInfo)
	for i := range queues {
		queueMap[queues[i].Name] = &queues[i]
	}

	var result []responses.RabbitMQQueueResponse
	for _, q := range queues {
		qType := h.managementService.ClassifyQueue(q.Name)

		// Apply filters
		if filter.Type != "" && filter.Type != "all" && filter.Type != qType {
			continue
		}
		if filter.Search != "" && !strings.Contains(strings.ToLower(q.Name), strings.ToLower(filter.Search)) {
			continue
		}
		if filter.HasMessages && q.Messages == 0 {
			continue
		}

		queueResp := responses.RabbitMQQueueResponse{
			Name:            q.Name,
			Type:            qType,
			VHost:           q.VHost,
			Durable:         q.Durable,
			AutoDelete:      q.AutoDelete,
			Messages:        q.Messages,
			MessagesReady:   q.MessagesReady,
			MessagesUnacked: q.MessagesUnacked,
			Consumers:       q.Consumers,
			State:           q.State,
			Arguments:       q.Arguments,
		}

		// For main queues, check if retry/DLQ exists
		if qType == "main" {
			retryName := q.Name + ".retry"
			dlqName := q.Name + ".dlq"

			if _, exists := queueMap[retryName]; exists {
				queueResp.HasRetryQueue = true
				queueResp.RetryQueueName = retryName
			}
			if _, exists := queueMap[dlqName]; exists {
				queueResp.HasDLQ = true
				queueResp.DLQName = dlqName
			}
		}

		// For retry/DLQ queues, add main queue reference
		if qType != "main" {
			queueResp.MainQueueName = h.managementService.GetQueueMainName(q.Name)
		}

		// Add rate information if available
		if q.MessageStats != nil {
			queueResp.MessageRate = &responses.RabbitMQRateInfo{
				PublishRate:   q.MessageStats.PublishDetails.Rate,
				DeliverRate:   q.MessageStats.DeliverDetails.Rate,
				AckRate:       q.MessageStats.AckDetails.Rate,
				RedeliverRate: q.MessageStats.RedeliverDetails.Rate,
			}
		}

		result = append(result, queueResp)
	}

	c.JSON(http.StatusOK, responses.SuccessResponse("Queues retrieved successfully", nil, result))
}

// ListQueueGroups godoc
//
//	@Summary		List RabbitMQ Queue Groups
//	@Description	Returns queues grouped by their main queue (main + retry + DLQ)
//	@Tags			RabbitMQ Admin
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	responses.APIResponse{data=[]responses.RabbitMQQueueGroupResponse}
//	@Failure		401	{object}	responses.APIResponse
//	@Failure		403	{object}	responses.APIResponse
//	@Failure		500	{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/admin/rabbitmq/queues/grouped [get]
func (h *RabbitMQHandler) ListQueueGroups(c *gin.Context) {
	ctx := c.Request.Context()

	if h.managementService == nil {
		c.JSON(http.StatusServiceUnavailable, responses.ErrorResponse("RabbitMQ management service not available", http.StatusServiceUnavailable))
		return
	}

	queues, err := h.managementService.ListQueues(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse("Failed to list queues: "+err.Error(), http.StatusInternalServerError))
		return
	}

	// Build groups
	groups := make(map[string]*responses.RabbitMQQueueGroupResponse)
	queueMap := make(map[string]*rabbitmq.QueueInfo)

	for i := range queues {
		queueMap[queues[i].Name] = &queues[i]
	}

	for _, q := range queues {
		qType := h.managementService.ClassifyQueue(q.Name)
		mainName := h.managementService.GetQueueMainName(q.Name)

		if _, exists := groups[mainName]; !exists {
			groups[mainName] = &responses.RabbitMQQueueGroupResponse{}
		}

		queueResp := &responses.RabbitMQQueueResponse{
			Name:            q.Name,
			Type:            qType,
			VHost:           q.VHost,
			Durable:         q.Durable,
			AutoDelete:      q.AutoDelete,
			Messages:        q.Messages,
			MessagesReady:   q.MessagesReady,
			MessagesUnacked: q.MessagesUnacked,
			Consumers:       q.Consumers,
			State:           q.State,
			Arguments:       q.Arguments,
		}

		groups[mainName].TotalMessages += q.Messages

		switch qType {
		case "main":
			groups[mainName].MainQueue = queueResp
		case "retry":
			groups[mainName].RetryQueue = queueResp
		case "dead_letter":
			groups[mainName].DLQ = queueResp
		case "delayed":
			groups[mainName].DelayQueue = queueResp
		}
	}

	// Convert to slice
	var result []responses.RabbitMQQueueGroupResponse
	for _, group := range groups {
		if group.MainQueue != nil { // Only include groups with main queue
			result = append(result, *group)
		}
	}
	slices.SortFunc(result, func(a, b responses.RabbitMQQueueGroupResponse) int {
		return strings.Compare(a.MainQueue.Name, b.MainQueue.Name)
	})

	c.JSON(http.StatusOK, responses.SuccessResponse("Queue groups retrieved successfully", nil, result))
}

// GetQueue godoc
//
//	@Summary		Get Queue Details
//	@Description	Returns detailed information about a specific queue
//	@Tags			RabbitMQ Admin
//	@Accept			json
//	@Produce		json
//	@Param			queueName	path		string	true	"Queue name"
//	@Success		200			{object}	responses.APIResponse{data=responses.RabbitMQQueueResponse}
//	@Failure		401			{object}	responses.APIResponse
//	@Failure		403			{object}	responses.APIResponse
//	@Failure		404			{object}	responses.APIResponse
//	@Failure		500			{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/admin/rabbitmq/queues/{queueName} [get]
func (h *RabbitMQHandler) GetQueue(c *gin.Context) {
	ctx := c.Request.Context()
	queueName := c.Param("queueName")

	if h.managementService == nil {
		c.JSON(http.StatusServiceUnavailable, responses.ErrorResponse("RabbitMQ management service not available", http.StatusServiceUnavailable))
		return
	}

	queue, err := h.managementService.GetQueue(ctx, queueName)
	if err != nil {
		c.JSON(http.StatusNotFound, responses.ErrorResponse("Queue not found: "+err.Error(), http.StatusNotFound))
		return
	}

	qType := h.managementService.ClassifyQueue(queue.Name)
	queueResp := responses.RabbitMQQueueResponse{
		Name:            queue.Name,
		Type:            qType,
		VHost:           queue.VHost,
		Durable:         queue.Durable,
		AutoDelete:      queue.AutoDelete,
		Messages:        queue.Messages,
		MessagesReady:   queue.MessagesReady,
		MessagesUnacked: queue.MessagesUnacked,
		Consumers:       queue.Consumers,
		State:           queue.State,
		Arguments:       queue.Arguments,
	}

	if qType != "main" {
		queueResp.MainQueueName = h.managementService.GetQueueMainName(queue.Name)
	}

	if queue.MessageStats != nil {
		queueResp.MessageRate = &responses.RabbitMQRateInfo{
			PublishRate:   queue.MessageStats.PublishDetails.Rate,
			DeliverRate:   queue.MessageStats.DeliverDetails.Rate,
			AckRate:       queue.MessageStats.AckDetails.Rate,
			RedeliverRate: queue.MessageStats.RedeliverDetails.Rate,
		}
	}

	c.JSON(http.StatusOK, responses.SuccessResponse("Queue retrieved successfully", nil, queueResp))
}

// GetQueueMessages godoc
//
//	@Summary		Get Messages from Queue
//	@Description	Returns messages from a queue without consuming them (peek)
//	@Tags			RabbitMQ Admin
//	@Accept			json
//	@Produce		json
//	@Param			queueName	path		string	true	"Queue name"
//	@Param			count		query		int		false	"Number of messages to retrieve (default: 10, max: 100)"
//	@Param			ack_mode	query		string	false	"Ack mode (ack_requeue_true, ack_requeue_false, reject_requeue_true, reject_requeue_false)"
//	@Param			encoding	query		string	false	"Encoding (auto, base64)"
//	@Success		200			{object}	responses.APIResponse{data=[]responses.RabbitMQMessageResponse}
//	@Failure		401			{object}	responses.APIResponse
//	@Failure		403			{object}	responses.APIResponse
//	@Failure		404			{object}	responses.APIResponse
//	@Failure		500			{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/admin/rabbitmq/queues/{queueName}/messages [get]
func (h *RabbitMQHandler) GetQueueMessages(c *gin.Context) {
	ctx := c.Request.Context()
	queueName := c.Param("queueName")

	if h.managementService == nil {
		c.JSON(http.StatusServiceUnavailable, responses.ErrorResponse("RabbitMQ management service not available", http.StatusServiceUnavailable))
		return
	}

	var req requests.RabbitMQGetMessagesRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid parameters: "+err.Error(), http.StatusBadRequest))
		return
	}

	count := req.Count
	if count == 0 {
		count = 10
	}
	if count > 100 {
		count = 100
	}

	ackMode := req.AckMode
	if ackMode == "" {
		ackMode = "ack_requeue_true" // Peek only, don't consume
	}

	messages, err := h.managementService.GetMessages(ctx, queueName, count, ackMode, req.Encoding)
	if err != nil {
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse("Failed to get messages: "+err.Error(), http.StatusInternalServerError))
		return
	}

	var result []responses.RabbitMQMessageResponse
	for _, msg := range messages {
		result = append(result, responses.RabbitMQMessageResponse{
			PayloadBytes:    msg.PayloadBytes,
			Redelivered:     msg.Redelivered,
			Exchange:        msg.Exchange,
			RoutingKey:      msg.RoutingKey,
			MessageCount:    msg.MessageCount,
			Payload:         msg.Payload,
			PayloadEncoding: msg.PayloadEncoding,
			Properties: responses.RabbitMQMessagePropertiesResponse{
				ContentType:   msg.Properties.ContentType,
				Headers:       msg.Properties.Headers,
				DeliveryMode:  msg.Properties.DeliveryMode,
				Priority:      msg.Properties.Priority,
				CorrelationID: msg.Properties.CorrelationID,
				MessageID:     msg.Properties.MessageID,
				Timestamp:     msg.Properties.Timestamp,
				Type:          msg.Properties.Type,
				AppID:         msg.Properties.AppID,
			},
		})
	}

	c.JSON(http.StatusOK, responses.SuccessResponse("Messages retrieved successfully", nil, result))
}

// ListExchanges godoc
//
//	@Summary		List RabbitMQ Exchanges
//	@Description	Returns all RabbitMQ exchanges
//	@Tags			RabbitMQ Admin
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	responses.APIResponse{data=[]responses.RabbitMQExchangeResponse}
//	@Failure		401	{object}	responses.APIResponse
//	@Failure		403	{object}	responses.APIResponse
//	@Failure		500	{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/admin/rabbitmq/exchanges [get]
func (h *RabbitMQHandler) ListExchanges(c *gin.Context) {
	ctx := c.Request.Context()

	if h.managementService == nil {
		c.JSON(http.StatusServiceUnavailable, responses.ErrorResponse("RabbitMQ management service not available", http.StatusServiceUnavailable))
		return
	}

	exchanges, err := h.managementService.ListExchanges(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse("Failed to list exchanges: "+err.Error(), http.StatusInternalServerError))
		return
	}

	var result []responses.RabbitMQExchangeResponse
	for _, ex := range exchanges {
		result = append(result, responses.RabbitMQExchangeResponse{
			Name:       ex.Name,
			VHost:      ex.VHost,
			Type:       ex.Type,
			Durable:    ex.Durable,
			AutoDelete: ex.AutoDelete,
			Internal:   ex.Internal,
			Arguments:  ex.Arguments,
		})
	}

	c.JSON(http.StatusOK, responses.SuccessResponse("Exchanges retrieved successfully", nil, result))
}

// PurgeQueue godoc
//
//	@Summary		Purge Queue
//	@Description	Removes all messages from a queue
//	@Tags			RabbitMQ Admin
//	@Accept			json
//	@Produce		json
//	@Param			queueName	path		string	true	"Queue name"
//	@Success		200			{object}	responses.APIResponse{data=responses.RabbitMQPurgeResponse}
//	@Failure		401			{object}	responses.APIResponse
//	@Failure		403			{object}	responses.APIResponse
//	@Failure		404			{object}	responses.APIResponse
//	@Failure		500			{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/admin/rabbitmq/queues/{queueName}/purge [delete]
func (h *RabbitMQHandler) PurgeQueue(c *gin.Context) {
	ctx := c.Request.Context()
	queueName := c.Param("queueName")

	if h.managementService == nil {
		c.JSON(http.StatusServiceUnavailable, responses.ErrorResponse("RabbitMQ management service not available", http.StatusServiceUnavailable))
		return
	}

	// Get queue info first to get message count
	queue, err := h.managementService.GetQueue(ctx, queueName)
	if err != nil {
		c.JSON(http.StatusNotFound, responses.ErrorResponse("Queue not found: "+err.Error(), http.StatusNotFound))
		return
	}

	messagesDeleted := queue.Messages

	if err := h.managementService.PurgeQueue(ctx, queueName); err != nil {
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse("Failed to purge queue: "+err.Error(), http.StatusInternalServerError))
		return
	}

	result := responses.RabbitMQPurgeResponse{
		QueueName:       queueName,
		MessagesDeleted: messagesDeleted,
		Status:          "purged",
	}

	c.JSON(http.StatusOK, responses.SuccessResponse("Queue purged successfully", nil, result))
}

// RetryDLQ godoc
//
//	@Summary		Retry Dead Letter Queue Messages
//	@Description	Moves messages from a DLQ back to the main queue for retry using a shovel
//	@Tags			RabbitMQ Admin
//	@Accept			json
//	@Produce		json
//	@Param			request	body		requests.RabbitMQRetryDLQRequest	true	"Retry request"
//	@Success		200		{object}	responses.APIResponse{data=responses.RabbitMQDLQRetryResponse}
//	@Failure		400		{object}	responses.APIResponse
//	@Failure		401		{object}	responses.APIResponse
//	@Failure		403		{object}	responses.APIResponse
//	@Failure		500		{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/admin/rabbitmq/dlq/retry [post]
func (h *RabbitMQHandler) RetryDLQ(c *gin.Context) {
	ctx := c.Request.Context()

	if h.managementService == nil {
		c.JSON(http.StatusServiceUnavailable, responses.ErrorResponse("RabbitMQ management service not available", http.StatusServiceUnavailable))
		return
	}

	var req requests.RabbitMQRetryDLQRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid request: "+err.Error(), http.StatusBadRequest))
		return
	}

	// Determine destination queue
	destQueue := req.DestinationQueue
	if destQueue == "" {
		// Default to main queue (remove .dlq suffix)
		destQueue = h.managementService.GetQueueMainName(req.SourceQueue)
	}

	// Create shovel to move messages
	err := h.managementService.MoveMessages(ctx, req.SourceQueue, destQueue, req.DestExchange, req.DestRoutingKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse("Failed to create shovel for retry: "+err.Error(), http.StatusInternalServerError))
		return
	}

	result := responses.RabbitMQDLQRetryResponse{
		SourceQueue:      req.SourceQueue,
		DestinationQueue: destQueue,
		Status:           "shovel_created",
		Message:          "A shovel has been created to move messages from the DLQ to the destination queue. Messages will be moved automatically.",
	}

	c.JSON(http.StatusOK, responses.SuccessResponse("DLQ retry initiated successfully", nil, result))
}

// ListShovels godoc
//
//	@Summary		List Shovels
//	@Description	Returns all active shovels in the vhost
//	@Tags			RabbitMQ Admin
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	responses.APIResponse{data=[]responses.RabbitMQShovelStatusResponse}
//	@Failure		401	{object}	responses.APIResponse
//	@Failure		403	{object}	responses.APIResponse
//	@Failure		500	{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/admin/rabbitmq/shovels [get]
func (h *RabbitMQHandler) ListShovels(c *gin.Context) {
	ctx := c.Request.Context()

	if h.managementService == nil {
		c.JSON(http.StatusServiceUnavailable, responses.ErrorResponse("RabbitMQ management service not available", http.StatusServiceUnavailable))
		return
	}

	shovels, err := h.managementService.ListShovels(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse("Failed to list shovels: "+err.Error(), http.StatusInternalServerError))
		return
	}

	var result []responses.RabbitMQShovelStatusResponse
	for _, s := range shovels {
		result = append(result, responses.RabbitMQShovelStatusResponse{
			Name:      s.Name,
			VHost:     s.VHost,
			Type:      s.Type,
			State:     s.State,
			Timestamp: s.Timestamp,
		})
	}

	c.JSON(http.StatusOK, responses.SuccessResponse("Shovels retrieved successfully", nil, result))
}

// DeleteShovel godoc
//
//	@Summary		Delete Shovel
//	@Description	Deletes a shovel by name
//	@Tags			RabbitMQ Admin
//	@Accept			json
//	@Produce		json
//	@Param			shovelName	path		string	true	"Shovel name"
//	@Success		200			{object}	responses.APIResponse
//	@Failure		401			{object}	responses.APIResponse
//	@Failure		403			{object}	responses.APIResponse
//	@Failure		404			{object}	responses.APIResponse
//	@Failure		500			{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/admin/rabbitmq/shovels/{shovelName} [delete]
func (h *RabbitMQHandler) DeleteShovel(c *gin.Context) {
	ctx := c.Request.Context()
	shovelName := c.Param("shovelName")

	if h.managementService == nil {
		c.JSON(http.StatusServiceUnavailable, responses.ErrorResponse("RabbitMQ management service not available", http.StatusServiceUnavailable))
		return
	}

	if err := h.managementService.DeleteShovel(ctx, shovelName); err != nil {
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse("Failed to delete shovel: "+err.Error(), http.StatusInternalServerError))
		return
	}

	c.JSON(http.StatusOK, responses.SuccessResponse("Shovel deleted successfully", nil, nil))
}

// PublishMessage godoc
//
//	@Summary		Publish Message
//	@Description	Publishes a message to an exchange (for testing/debugging)
//	@Tags			RabbitMQ Admin
//	@Accept			json
//	@Produce		json
//	@Param			request	body		requests.RabbitMQPublishMessageRequest	true	"Message to publish"
//	@Success		200		{object}	responses.APIResponse
//	@Failure		400		{object}	responses.APIResponse
//	@Failure		401		{object}	responses.APIResponse
//	@Failure		403		{object}	responses.APIResponse
//	@Failure		500		{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/admin/rabbitmq/publish [post]
func (h *RabbitMQHandler) PublishMessage(c *gin.Context) {
	ctx := c.Request.Context()

	if h.managementService == nil {
		c.JSON(http.StatusServiceUnavailable, responses.ErrorResponse("RabbitMQ management service not available", http.StatusServiceUnavailable))
		return
	}

	var req requests.RabbitMQPublishMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid request: "+err.Error(), http.StatusBadRequest))
		return
	}

	properties := map[string]interface{}{}
	if req.ContentType != "" {
		properties["content_type"] = req.ContentType
	}
	if req.Headers != nil {
		properties["headers"] = req.Headers
	}
	if req.DeliveryMode != 0 {
		properties["delivery_mode"] = req.DeliveryMode
	}
	if req.Priority != 0 {
		properties["priority"] = req.Priority
	}
	if req.CorrelationID != "" {
		properties["correlation_id"] = req.CorrelationID
	}
	if req.MessageID != "" {
		properties["message_id"] = req.MessageID
	}

	if err := h.managementService.PublishMessage(ctx, req.Exchange, req.RoutingKey, req.Payload, properties); err != nil {
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse("Failed to publish message: "+err.Error(), http.StatusInternalServerError))
		return
	}

	c.JSON(http.StatusOK, responses.SuccessResponse("Message published successfully", nil, nil))
}

// GetHealth godoc
//
//	@Summary		Get RabbitMQ Health
//	@Description	Returns the health status of RabbitMQ connection and management API
//	@Tags			RabbitMQ Admin
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	responses.APIResponse{data=responses.RabbitMQHealthResponse}
//	@Failure		401	{object}	responses.APIResponse
//	@Failure		403	{object}	responses.APIResponse
//	@Failure		500	{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/admin/rabbitmq/health [get]
func (h *RabbitMQHandler) GetHealth(c *gin.Context) {
	ctx := c.Request.Context()

	health := responses.RabbitMQHealthResponse{
		Connected:     false,
		ManagementAPI: false,
	}

	// Check AMQP connection
	if h.rabbitmq != nil && h.rabbitmq.IsConnected() {
		health.Connected = true
	}

	// Check management API
	if h.managementService != nil {
		overview, err := h.managementService.GetOverview(ctx)
		if err == nil {
			health.ManagementAPI = true
			health.Details = overview

			// Extract specific fields from overview
			if clusterName, ok := overview["cluster_name"].(string); ok {
				health.ClusterName = clusterName
			}
			if rmqVersion, ok := overview["rabbitmq_version"].(string); ok {
				health.RabbitMQVersion = rmqVersion
			}
			if erlangVersion, ok := overview["erlang_version"].(string); ok {
				health.ErlangVersion = erlangVersion
			}
			if node, ok := overview["node"].(string); ok {
				health.NodeName = node
			}
		}
	}

	statusCode := http.StatusOK
	if !health.Connected || !health.ManagementAPI {
		statusCode = http.StatusServiceUnavailable
	}

	c.JSON(statusCode, responses.SuccessResponse("RabbitMQ health check completed", &statusCode, health))
}
