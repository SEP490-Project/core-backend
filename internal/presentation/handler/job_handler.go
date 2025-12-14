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
//	@Success		200	{object}	responses.APIResponse
//	@Failure		401	{object}	responses.APIResponse
//	@Failure		403	{object}	responses.APIResponse
//	@Failure		500	{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/jobs/ctr-aggregation [post]
func (h *JobHandler) TriggerCTRAggregationJob(c *gin.Context) {
	h.triggerJob(c, "ctr_aggregation_job")
}

// TriggerExpiredLinkCleanupJob godoc
//
//	@Summary		Trigger Expired Link Cleanup Job
//	@Description	Manually trigger the expired link cleanup job
//	@Tags			Jobs
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	responses.APIResponse
//	@Failure		401	{object}	responses.APIResponse
//	@Failure		403	{object}	responses.APIResponse
//	@Failure		500	{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/jobs/expired-link-cleanup [post]
func (h *JobHandler) TriggerExpiredLinkCleanupJob(c *gin.Context) {
	h.triggerJob(c, "expired_link_cleanup_job")
}

// TriggerPayOSExpiryCheckJob godoc
//
//	@Summary		Trigger PayOS Expiry Check Job
//	@Description	Manually trigger the PayOS expiry check job
//	@Tags			Jobs
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	responses.APIResponse
//	@Failure		401	{object}	responses.APIResponse
//	@Failure		403	{object}	responses.APIResponse
//	@Failure		500	{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/jobs/payos-expiry-check [post]
func (h *JobHandler) TriggerPayOSExpiryCheckJob(c *gin.Context) {
	h.triggerJob(c, "payos_expiry_check_job")
}

// TriggerPreOrderOpeningCheckJob godoc
//
//	@Summary		Trigger Pre-Order Opening Check Job
//	@Description	Manually trigger the pre-order opening check job
//	@Tags			Jobs
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	responses.APIResponse
//	@Failure		401	{object}	responses.APIResponse
//	@Failure		403	{object}	responses.APIResponse
//	@Failure		500	{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/jobs/pre-order-opening-check [post]
func (h *JobHandler) TriggerPreOrderOpeningCheckJob(c *gin.Context) {
	h.triggerJob(c, "pre_order_opening_check_job")
}

// TriggerTikTokStatusPollerJob godoc
//
//	@Summary		Trigger TikTok Status Poller Job
//	@Description	Manually trigger the TikTok status poller job
//	@Tags			Jobs
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	responses.APIResponse
//	@Failure		401	{object}	responses.APIResponse
//	@Failure		403	{object}	responses.APIResponse
//	@Failure		500	{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/jobs/tiktok-status-poller [post]
func (h *JobHandler) TriggerTikTokStatusPollerJob(c *gin.Context) {
	h.triggerJob(c, "tiktok_status_poller_job")
}

// TriggerContentMetricsPollerJob godoc
//
//	@Summary		Trigger Content Metrics Poller Job
//	@Description	Manually trigger the content metrics poller job
//	@Tags			Jobs
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	responses.APIResponse
//	@Failure		401	{object}	responses.APIResponse
//	@Failure		403	{object}	responses.APIResponse
//	@Failure		500	{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/jobs/content-metrics-poller [post]
func (h *JobHandler) TriggerContentMetricsPollerJob(c *gin.Context) {
	h.triggerJob(c, "content_metrics_poller_job")
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

func (h *JobHandler) triggerJob(c *gin.Context, jobName string) {
	job, exists := h.cronJobRegistry.GetJobByName(jobName)
	if !exists {
		c.JSON(http.StatusNotFound, responses.ErrorResponse("Job not found", http.StatusNotFound))
		return
	}

	go job.Run()

	c.JSON(http.StatusOK, responses.SuccessResponse("Job triggered successfully", nil, nil))
}
