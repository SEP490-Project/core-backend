package handler

import (
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/infrastructure/asynq"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	asynqlib "github.com/hibiken/asynq"
	"go.uber.org/zap"
)

// AsynqHandler handles Asynq management API endpoints
type AsynqHandler struct {
	asynqClient *asynq.AsynqClient
}

// NewAsynqHandler creates a new Asynq handler
func NewAsynqHandler(asynqClient *asynq.AsynqClient) *AsynqHandler {
	return &AsynqHandler{
		asynqClient: asynqClient,
	}
}

// GetOverview godoc
//
//	@Summary		Get Asynq Overview
//	@Description	Returns an overview of Asynq queues, tasks, and status
//	@Tags			Asynq Admin
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	responses.APIResponse{data=responses.AsynqOverviewResponse}
//	@Failure		401	{object}	responses.APIResponse
//	@Failure		403	{object}	responses.APIResponse
//	@Failure		500	{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/admin/asynq/overview [get]
func (h *AsynqHandler) GetOverview(c *gin.Context) {
	if h.asynqClient == nil {
		c.JSON(http.StatusServiceUnavailable, responses.ErrorResponse("Asynq client not available", http.StatusServiceUnavailable))
		return
	}

	inspector := h.asynqClient.GetInspector()

	// Get all queues
	queueNames, err := inspector.Queues()
	if err != nil {
		zap.L().Error("Failed to list Asynq queues", zap.Error(err))
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse("Failed to list queues: "+err.Error(), http.StatusInternalServerError))
		return
	}

	// Get stats for each queue
	var totalActive, totalPending, totalScheduled, totalRetry, totalArchived, totalCompleted, totalProcessed int
	var queues []responses.AsynqQueueInfo

	for _, queueName := range queueNames {
		queueInfo, err := inspector.GetQueueInfo(queueName)
		if err != nil {
			zap.L().Warn("Failed to get queue info", zap.String("queue", queueName), zap.Error(err))
			continue
		}

		totalActive += queueInfo.Active
		totalPending += queueInfo.Pending
		totalScheduled += queueInfo.Scheduled
		totalRetry += queueInfo.Retry
		totalArchived += queueInfo.Archived
		totalCompleted += queueInfo.Completed
		totalProcessed += queueInfo.Processed

		queues = append(queues, responses.AsynqQueueInfo{
			Name:      queueName,
			Size:      queueInfo.Size,
			Active:    queueInfo.Active,
			Pending:   queueInfo.Pending,
			Scheduled: queueInfo.Scheduled,
			Retry:     queueInfo.Retry,
			Archived:  queueInfo.Archived,
			Completed: queueInfo.Completed,
			Processed: queueInfo.Processed,
			Paused:    queueInfo.Paused,
			Timestamp: queueInfo.Timestamp,
		})
	}

	// Health check
	healthStatus := "healthy"
	if err := h.asynqClient.HealthCheck(); err != nil {
		healthStatus = "unhealthy"
	}

	overview := responses.AsynqOverviewResponse{
		Status:         healthStatus,
		TotalQueues:    len(queueNames),
		TotalActive:    totalActive,
		TotalPending:   totalPending,
		TotalScheduled: totalScheduled,
		TotalRetry:     totalRetry,
		TotalArchived:  totalArchived,
		TotalCompleted: totalCompleted,
		TotalProcessed: totalProcessed,
		Queues:         queues,
	}

	c.JSON(http.StatusOK, responses.SuccessResponse("Asynq overview retrieved successfully", nil, overview))
}

// ListTasks godoc
//
//	@Summary		List Asynq Tasks
//	@Description	Returns tasks in a specific queue and state
//	@Tags			Asynq Admin
//	@Accept			json
//	@Produce		json
//	@Param			queue	query		string	true	"Queue name (default, critical, low)"
//	@Param			state	query		string	true	"Task state (scheduled, pending, active, retry, archived)"
//	@Param			limit	query		int		false	"Limit number of results (default: 20, max: 100)"
//	@Param			page	query		int		false	"Page number for pagination (default: 1)"
//	@Success		200		{object}	responses.APIResponse{data=responses.AsynqTaskListResponse}
//	@Failure		400		{object}	responses.APIResponse
//	@Failure		401		{object}	responses.APIResponse
//	@Failure		403		{object}	responses.APIResponse
//	@Failure		500		{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/admin/asynq/tasks [get]
func (h *AsynqHandler) ListTasks(c *gin.Context) {
	if h.asynqClient == nil {
		c.JSON(http.StatusServiceUnavailable, responses.ErrorResponse("Asynq client not available", http.StatusServiceUnavailable))
		return
	}

	var filter requests.AsynqTaskFilterRequest
	if err := c.ShouldBindQuery(&filter); err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid filter parameters: "+err.Error(), http.StatusBadRequest))
		return
	}

	// Set defaults
	if filter.Queue == "" {
		filter.Queue = "default"
	}
	if filter.Limit == 0 {
		filter.Limit = 20
	}
	if filter.Limit > 100 {
		filter.Limit = 100
	}
	if filter.Page < 1 {
		filter.Page = 1
	}

	inspector := h.asynqClient.GetInspector()

	// Calculate offset for pagination
	offset := (filter.Page - 1) * filter.Limit
	listOpts := []asynqlib.ListOption{
		asynqlib.PageSize(filter.Limit),
		asynqlib.Page(offset),
	}

	var tasks []*asynqlib.TaskInfo
	var err error

	switch strings.ToLower(filter.State) {
	case "scheduled":
		tasks, err = inspector.ListScheduledTasks(filter.Queue, listOpts...)
	case "pending":
		tasks, err = inspector.ListPendingTasks(filter.Queue, listOpts...)
	case "active":
		tasks, err = inspector.ListActiveTasks(filter.Queue)
	case "retry":
		tasks, err = inspector.ListRetryTasks(filter.Queue, listOpts...)
	case "archived":
		tasks, err = inspector.ListArchivedTasks(filter.Queue, listOpts...)
	default:
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid state. Must be one of: scheduled, pending, active, retry, archived", http.StatusBadRequest))
		return
	}

	if err != nil {
		zap.L().Error("Failed to list Asynq tasks",
			zap.String("queue", filter.Queue),
			zap.String("state", filter.State),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse("Failed to list tasks: "+err.Error(), http.StatusInternalServerError))
		return
	}

	// Convert to response format
	var taskResponses []responses.AsynqTaskResponse
	for _, task := range tasks {
		taskResponses = append(taskResponses, responses.AsynqTaskResponse{
			ID:            task.ID,
			Type:          task.Type,
			Queue:         task.Queue,
			State:         string(rune(task.State)),
			MaxRetry:      task.MaxRetry,
			Retried:       task.Retried,
			LastErr:       task.LastErr,
			NextProcessAt: task.NextProcessAt,
			Timeout:       int(task.Timeout.Seconds()),
			Deadline:      &task.Deadline,
			CompletedAt:   &task.CompletedAt,
			Result:        string(task.Result),
		})
	}

	result := responses.AsynqTaskListResponse{
		Tasks:      taskResponses,
		TotalCount: len(taskResponses),
		Page:       filter.Page,
		Limit:      filter.Limit,
	}

	c.JSON(http.StatusOK, responses.SuccessResponse("Tasks retrieved successfully", nil, result))
}

// GetTaskDetails godoc
//
//	@Summary		Get Task Details
//	@Description	Returns details of a specific task
//	@Tags			Asynq Admin
//	@Accept			json
//	@Produce		json
//	@Param			queue	query		string	true	"Queue name"
//	@Param			task_id	query		string	true	"Task ID"
//	@Success		200		{object}	responses.APIResponse{data=responses.AsynqTaskResponse}
//	@Failure		400		{object}	responses.APIResponse
//	@Failure		404		{object}	responses.APIResponse
//	@Failure		500		{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/admin/asynq/tasks/details [get]
func (h *AsynqHandler) GetTaskDetails(c *gin.Context) {
	if h.asynqClient == nil {
		c.JSON(http.StatusServiceUnavailable, responses.ErrorResponse("Asynq client not available", http.StatusServiceUnavailable))
		return
	}

	queue := c.Query("queue")
	taskID := c.Query("task_id")

	if queue == "" || taskID == "" {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("queue and task_id are required", http.StatusBadRequest))
		return
	}

	taskInfo, err := h.asynqClient.GetTaskInfo(queue, taskID)
	if err != nil {
		zap.L().Error("Failed to get task details",
			zap.String("queue", queue),
			zap.String("task_id", taskID),
			zap.Error(err))
		c.JSON(http.StatusNotFound, responses.ErrorResponse("Task not found: "+err.Error(), http.StatusNotFound))
		return
	}

	var payload any
	if err := json.Unmarshal(taskInfo.Payload, &payload); err != nil {
		zap.L().Warn("Failed to unmarshal task payload",
			zap.String("queue", queue),
			zap.String("task_id", taskID),
			zap.Error(err))
	}
	taskResp := responses.AsynqTaskResponse{
		ID:            taskInfo.ID,
		Type:          taskInfo.Type,
		Queue:         taskInfo.Queue,
		State:         string(rune(taskInfo.State)),
		MaxRetry:      taskInfo.MaxRetry,
		Retried:       taskInfo.Retried,
		LastErr:       taskInfo.LastErr,
		NextProcessAt: taskInfo.NextProcessAt,
		Timeout:       int(taskInfo.Timeout.Seconds()),
		Deadline:      &taskInfo.Deadline,
		CompletedAt:   &taskInfo.CompletedAt,
		Result:        string(taskInfo.Result),
		Payload:       payload,
	}

	c.JSON(http.StatusOK, responses.SuccessResponse("Task details retrieved successfully", nil, taskResp))
}

// DeleteTask godoc
//
//	@Summary		Delete/Cancel Task
//	@Description	Deletes or cancels a task (for scheduled, pending, retry, or archived tasks)
//	@Tags			Asynq Admin
//	@Accept			json
//	@Produce		json
//	@Param			request	body		requests.AsynqDeleteTaskRequest	true	"Delete task request"
//	@Success		200		{object}	responses.APIResponse
//	@Failure		400		{object}	responses.APIResponse
//	@Failure		500		{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/admin/asynq/tasks [delete]
func (h *AsynqHandler) DeleteTask(c *gin.Context) {
	if h.asynqClient == nil {
		c.JSON(http.StatusServiceUnavailable, responses.ErrorResponse("Asynq client not available", http.StatusServiceUnavailable))
		return
	}

	var req requests.AsynqDeleteTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid request: "+err.Error(), http.StatusBadRequest))
		return
	}

	inspector := h.asynqClient.GetInspector()

	var err error
	switch strings.ToLower(req.State) {
	case "scheduled":
		err = inspector.DeleteTask(req.Queue, req.TaskID)
	case "pending":
		err = inspector.DeleteTask(req.Queue, req.TaskID)
	case "retry":
		err = inspector.DeleteTask(req.Queue, req.TaskID)
	case "archived":
		err = inspector.DeleteTask(req.Queue, req.TaskID)
	default:
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Cannot delete active tasks. State must be: scheduled, pending, retry, or archived", http.StatusBadRequest))
		return
	}

	if err != nil {
		zap.L().Error("Failed to delete task",
			zap.String("queue", req.Queue),
			zap.String("task_id", req.TaskID),
			zap.String("state", req.State),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse("Failed to delete task: "+err.Error(), http.StatusInternalServerError))
		return
	}

	zap.L().Info("Task deleted",
		zap.String("queue", req.Queue),
		zap.String("task_id", req.TaskID),
		zap.String("state", req.State))

	c.JSON(http.StatusOK, responses.SuccessResponse("Task deleted successfully", nil, nil))
}

// RunTask godoc
//
//	@Summary		Run Task Immediately
//	@Description	Forces a scheduled or retry task to run immediately
//	@Tags			Asynq Admin
//	@Accept			json
//	@Produce		json
//	@Param			request	body		requests.AsynqRunTaskRequest	true	"Run task request"
//	@Success		200		{object}	responses.APIResponse
//	@Failure		400		{object}	responses.APIResponse
//	@Failure		500		{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/admin/asynq/tasks/run [post]
func (h *AsynqHandler) RunTask(c *gin.Context) {
	if h.asynqClient == nil {
		c.JSON(http.StatusServiceUnavailable, responses.ErrorResponse("Asynq client not available", http.StatusServiceUnavailable))
		return
	}

	var req requests.AsynqRunTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid request: "+err.Error(), http.StatusBadRequest))
		return
	}

	inspector := h.asynqClient.GetInspector()

	err := inspector.RunTask(req.Queue, req.TaskID)
	if err != nil {
		zap.L().Error("Failed to run task",
			zap.String("queue", req.Queue),
			zap.String("task_id", req.TaskID),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse("Failed to run task: "+err.Error(), http.StatusInternalServerError))
		return
	}

	zap.L().Info("Task scheduled to run immediately",
		zap.String("queue", req.Queue),
		zap.String("task_id", req.TaskID))

	c.JSON(http.StatusOK, responses.SuccessResponse("Task scheduled to run immediately", nil, nil))
}

// ArchiveTask godoc
//
//	@Summary		Archive Task
//	@Description	Archives a task (moves it to archived state)
//	@Tags			Asynq Admin
//	@Accept			json
//	@Produce		json
//	@Param			request	body		requests.AsynqArchiveTaskRequest	true	"Archive task request"
//	@Success		200		{object}	responses.APIResponse
//	@Failure		400		{object}	responses.APIResponse
//	@Failure		500		{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/admin/asynq/tasks/archive [patch]
func (h *AsynqHandler) ArchiveTask(c *gin.Context) {
	if h.asynqClient == nil {
		c.JSON(http.StatusServiceUnavailable, responses.ErrorResponse("Asynq client not available", http.StatusServiceUnavailable))
		return
	}

	var req requests.AsynqArchiveTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid request: "+err.Error(), http.StatusBadRequest))
		return
	}

	inspector := h.asynqClient.GetInspector()

	err := inspector.ArchiveTask(req.Queue, req.TaskID)
	if err != nil {
		zap.L().Error("Failed to archive task",
			zap.String("queue", req.Queue),
			zap.String("task_id", req.TaskID),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse("Failed to archive task: "+err.Error(), http.StatusInternalServerError))
		return
	}

	zap.L().Info("Task archived",
		zap.String("queue", req.Queue),
		zap.String("task_id", req.TaskID))

	c.JSON(http.StatusOK, responses.SuccessResponse("Task archived successfully", nil, nil))
}

// PauseQueue godoc
//
//	@Summary		Pause Queue
//	@Description	Pauses task processing in a queue
//	@Tags			Asynq Admin
//	@Accept			json
//	@Produce		json
//	@Param			request	body		requests.AsynqQueueActionRequest	true	"Pause queue request"
//	@Success		200		{object}	responses.APIResponse
//	@Failure		400		{object}	responses.APIResponse
//	@Failure		500		{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/admin/asynq/queues/pause [patch]
func (h *AsynqHandler) PauseQueue(c *gin.Context) {
	if h.asynqClient == nil {
		c.JSON(http.StatusServiceUnavailable, responses.ErrorResponse("Asynq client not available", http.StatusServiceUnavailable))
		return
	}

	var req requests.AsynqQueueActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid request: "+err.Error(), http.StatusBadRequest))
		return
	}

	inspector := h.asynqClient.GetInspector()

	err := inspector.PauseQueue(req.Queue)
	if err != nil {
		zap.L().Error("Failed to pause queue", zap.String("queue", req.Queue), zap.Error(err))
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse("Failed to pause queue: "+err.Error(), http.StatusInternalServerError))
		return
	}

	zap.L().Info("Queue paused", zap.String("queue", req.Queue))
	c.JSON(http.StatusOK, responses.SuccessResponse("Queue paused successfully", nil, nil))
}

// UnpauseQueue godoc
//
//	@Summary		Unpause Queue
//	@Description	Resumes task processing in a paused queue
//	@Tags			Asynq Admin
//	@Accept			json
//	@Produce		json
//	@Param			request	body		requests.AsynqQueueActionRequest	true	"Unpause queue request"
//	@Success		200		{object}	responses.APIResponse
//	@Failure		400		{object}	responses.APIResponse
//	@Failure		500		{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/admin/asynq/queues/unpause [patch]
func (h *AsynqHandler) UnpauseQueue(c *gin.Context) {
	if h.asynqClient == nil {
		c.JSON(http.StatusServiceUnavailable, responses.ErrorResponse("Asynq client not available", http.StatusServiceUnavailable))
		return
	}

	var req requests.AsynqQueueActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid request: "+err.Error(), http.StatusBadRequest))
		return
	}

	inspector := h.asynqClient.GetInspector()

	err := inspector.UnpauseQueue(req.Queue)
	if err != nil {
		zap.L().Error("Failed to unpause queue", zap.String("queue", req.Queue), zap.Error(err))
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse("Failed to unpause queue: "+err.Error(), http.StatusInternalServerError))
		return
	}

	zap.L().Info("Queue unpaused", zap.String("queue", req.Queue))
	c.JSON(http.StatusOK, responses.SuccessResponse("Queue unpaused successfully", nil, nil))
}

// GetQueueStats godoc
//
//	@Summary		Get Queue Statistics
//	@Description	Returns detailed statistics for a specific queue
//	@Tags			Asynq Admin
//	@Accept			json
//	@Produce		json
//	@Param			queue	query		string	true	"Queue name"
//	@Success		200		{object}	responses.APIResponse{data=responses.AsynqQueueStatsResponse}
//	@Failure		400		{object}	responses.APIResponse
//	@Failure		500		{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/admin/asynq/queues/stats [get]
func (h *AsynqHandler) GetQueueStats(c *gin.Context) {
	if h.asynqClient == nil {
		c.JSON(http.StatusServiceUnavailable, responses.ErrorResponse("Asynq client not available", http.StatusServiceUnavailable))
		return
	}

	queue := c.Query("queue")
	if queue == "" {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("queue parameter is required", http.StatusBadRequest))
		return
	}

	inspector := h.asynqClient.GetInspector()

	queueInfo, err := inspector.GetQueueInfo(queue)
	if err != nil {
		zap.L().Error("Failed to get queue stats", zap.String("queue", queue), zap.Error(err))
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse("Failed to get queue stats: "+err.Error(), http.StatusInternalServerError))
		return
	}

	// Get historical stats for the last 24 hours
	now := time.Now()
	dailyStats, err := inspector.History(queue, 24)
	if err != nil {
		zap.L().Warn("Failed to get queue history", zap.String("queue", queue), zap.Error(err))
		// Continue without historical data
	}

	stats := responses.AsynqQueueStatsResponse{
		Name:           queue,
		Size:           queueInfo.Size,
		Active:         queueInfo.Active,
		Pending:        queueInfo.Pending,
		Scheduled:      queueInfo.Scheduled,
		Retry:          queueInfo.Retry,
		Archived:       queueInfo.Archived,
		Completed:      queueInfo.Completed,
		Processed:      queueInfo.Processed,
		Failed:         queueInfo.Processed - queueInfo.Completed, // Approximation
		Paused:         queueInfo.Paused,
		Timestamp:      queueInfo.Timestamp,
		ProcessedToday: 0,
		CompletedToday: 0,
		FailedToday:    0,
		AvgProcessTime: 0,
	}

	// Aggregate daily stats
	for _, ds := range dailyStats {
		if ds.Queue == queue && ds.Date.After(now.Add(-24*time.Hour)) {
			stats.ProcessedToday += ds.Processed
			stats.CompletedToday += ds.Processed - ds.Failed
			stats.FailedToday += ds.Failed
		}
	}

	c.JSON(http.StatusOK, responses.SuccessResponse("Queue stats retrieved successfully", nil, stats))
}
