package handler

import (
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/infrastructure/jobs"
	"net/http"

	"github.com/gin-gonic/gin"
)

type JobHandler struct {
	cronJobRegistry *jobs.CronJobRegistry
}

func NewJobHandler(cronJobRegistry *jobs.CronJobRegistry) *JobHandler {
	return &JobHandler{
		cronJobRegistry: cronJobRegistry,
	}
}

// TriggerCTRAggregationJob godoc
//
//	@Summary		Trigger CTR Aggregation Job
//	@Description	Manually trigger the CTR aggregation job
//	@Tags			Jobs
//	@Accept			json
//	@Produce		json
//	@Param			async	query		bool	false	"Run job asynchronously (default: true)"
//	@Success		200		{object}	responses.APIResponse
//	@Failure		401		{object}	responses.APIResponse
//	@Failure		403		{object}	responses.APIResponse
//	@Failure		500		{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/jobs/ctr-aggregation [post]
func (h *JobHandler) TriggerCTRAggregationJob(c *gin.Context) {
	isAsync := true
	if val := c.Query("async"); val == "false" {
		isAsync = false
	}
	h.triggerJob(c, "ctr_aggregation_job", isAsync)
}

// TriggerExpiredLinkCleanupJob godoc
//
//	@Summary		Trigger Expired Link Cleanup Job
//	@Description	Manually trigger the expired link cleanup job
//	@Tags			Jobs
//	@Accept			json
//	@Produce		json
//	@Param			async	query		bool	false	"Run job asynchronously (default: true)"
//	@Success		200		{object}	responses.APIResponse
//	@Failure		401		{object}	responses.APIResponse
//	@Failure		403		{object}	responses.APIResponse
//	@Failure		500		{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/jobs/expired-link-cleanup [post]
func (h *JobHandler) TriggerExpiredLinkCleanupJob(c *gin.Context) {
	isAsync := true
	if val := c.Query("async"); val == "false" {
		isAsync = false
	}
	h.triggerJob(c, "expired_link_cleanup_job", isAsync)
}

// TriggerPayOSExpiryCheckJob godoc
//
//	@Summary		Trigger PayOS Expiry Check Job
//	@Description	Manually trigger the PayOS expiry check job
//	@Tags			Jobs
//	@Accept			json
//	@Produce		json
//	@Param			async	query		bool	false	"Run job asynchronously (default: true)"
//	@Success		200		{object}	responses.APIResponse
//	@Failure		401		{object}	responses.APIResponse
//	@Failure		403		{object}	responses.APIResponse
//	@Failure		500		{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/jobs/payos-expiry-check [post]
func (h *JobHandler) TriggerPayOSExpiryCheckJob(c *gin.Context) {
	isAsync := true
	if val := c.Query("async"); val == "false" {
		isAsync = false
	}
	h.triggerJob(c, "payos_expiry_check_job", isAsync)
}

// TriggerPreOrderOpeningCheckJob godoc
//
//	@Summary		Trigger Pre-Order Opening Check Job
//	@Description	Manually trigger the pre-order opening check job
//	@Tags			Jobs
//	@Accept			json
//	@Produce		json
//	@Param			async	query		bool	false	"Run job asynchronously (default: true)"
//	@Success		200		{object}	responses.APIResponse
//	@Failure		401		{object}	responses.APIResponse
//	@Failure		403		{object}	responses.APIResponse
//	@Failure		500		{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/jobs/pre-order-opening-check [post]
func (h *JobHandler) TriggerPreOrderOpeningCheckJob(c *gin.Context) {
	isAsync := true
	if val := c.Query("async"); val == "false" {
		isAsync = false
	}
	h.triggerJob(c, "pre_order_opening_check_job", isAsync)
}

// TriggerTikTokStatusPollerJob godoc
//
//	@Summary		Trigger TikTok Status Poller Job
//	@Description	Manually trigger the TikTok status poller job
//	@Tags			Jobs
//	@Accept			json
//	@Produce		json
//	@Param			async	query		bool	false	"Run job asynchronously (default: true)"
//	@Success		200		{object}	responses.APIResponse
//	@Failure		401		{object}	responses.APIResponse
//	@Failure		403		{object}	responses.APIResponse
//	@Failure		500		{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/jobs/tiktok-status-poller [post]
func (h *JobHandler) TriggerTikTokStatusPollerJob(c *gin.Context) {
	isAsync := true
	if val := c.Query("async"); val == "false" {
		isAsync = false
	}
	h.triggerJob(c, "tiktok_status_poller_job", isAsync)
}

// TriggerContentMetricsPollerJob godoc
//
//	@Summary		Trigger Content Metrics Poller Job
//	@Description	Manually trigger the content metrics poller job
//	@Tags			Jobs
//	@Accept			json
//	@Produce		json
//	@Param			async	query		bool	false	"Run job asynchronously (default: true)"
//	@Success		200		{object}	responses.APIResponse
//	@Failure		401		{object}	responses.APIResponse
//	@Failure		403		{object}	responses.APIResponse
//	@Failure		500		{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/jobs/content-metrics-poller [post]
func (h *JobHandler) TriggerContentMetricsPollerJob(c *gin.Context) {
	isAsync := true
	if val := c.Query("async"); val == "false" {
		isAsync = false
	}
	// Trigger CTR Aggregation Job to start aggregate affilaite link data first,
	// This can be run asynchronously ctr_aggregation_job is more lightweight compared to content_metrics_poller_job
	h.triggerJob(c, "ctr_aggregation_job", true)
	h.triggerJob(c, "content_metrics_poller_job", isAsync)
}

// TriggerAllJobs godoc
//
//	@Summary		Trigger All Jobs
//	@Description	Manually trigger all registered jobs
//	@Tags			Jobs
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	responses.APIResponse
//	@Failure		401	{object}	responses.APIResponse
//	@Failure		403	{object}	responses.APIResponse
//	@Failure		500	{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/jobs/trigger-all [post]
func (h *JobHandler) TriggerAllJobs(c *gin.Context) {
	jobs := h.cronJobRegistry.GetAllJobs()
	triggered := make([]string, 0, len(jobs))

	for name, job := range jobs {
		go job.Run()
		triggered = append(triggered, name)
	}

	c.JSON(http.StatusOK, responses.SuccessResponse("All jobs triggered successfully", nil, triggered))
}

// GetAllRegisteredJobs godoc
//
//	@Summary		Get All Registered Jobs
//	@Description	Retrieve a list of all registered cron jobs with their statuses
//	@Tags			Jobs
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	responses.APIResponse
//	@Failure		401	{object}	responses.APIResponse
//	@Failure		403	{object}	responses.APIResponse
//	@Failure		500	{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/jobs [get]
func (h *JobHandler) GetAllRegisteredJobs(c *gin.Context) {
	jobs := h.cronJobRegistry.GetAllJobs()
	jobInfos := make(map[string]any, len(jobs))
	for name, cronJob := range jobs {
		jobInfos[name] = map[string]any{
			"name":     name,
			"enabled":  cronJob.IsEnabled(),
			"last_run": cronJob.GetLastRunTime(),
		}
	}

	c.JSON(http.StatusOK, responses.SuccessResponse("All jobs registered successfully", nil, jobInfos))
}

// TriggerJobByName godoc
//
//	@Summary		Trigger Job By Name
//	@Description	Manually trigger a specific job
//	@Tags			Jobs
//	@Accept			json
//	@Produce		json
//	@Param			jobName	path		string	true	"Job name"
//	@Param			async	query		bool	false	"Run job asynchronously (default: true)"
//	@Success		200		{object}	responses.APIResponse
//	@Failure		401		{object}	responses.APIResponse
//	@Failure		403		{object}	responses.APIResponse
//	@Failure		500		{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/jobs/trigger/{jobName} [post]
func (h *JobHandler) TriggerJobByName(c *gin.Context) {
	jobName := c.Param("jobName")
	isAsync := true
	if val := c.Query("async"); val == "false" {
		isAsync = false
	}
	h.triggerJob(c, jobName, isAsync)
}

func (h *JobHandler) triggerJob(c *gin.Context, jobName string, isAsync bool) {
	job, exists := h.cronJobRegistry.GetJobByName(jobName)
	if !exists {
		c.JSON(http.StatusNotFound, responses.ErrorResponse("Job not found", http.StatusNotFound))
		return
	}

	if isAsync {
		go job.Run()
	} else {
		job.Run()
	}

	c.JSON(http.StatusOK, responses.SuccessResponse("Job triggered successfully", nil, nil))
}
