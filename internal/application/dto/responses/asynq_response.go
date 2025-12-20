package responses

import "time"

// AsynqOverviewResponse represents the overview of Asynq system
type AsynqOverviewResponse struct {
	Status         string           `json:"status" example:"healthy"`
	TotalQueues    int              `json:"total_queues" example:"3"`
	TotalActive    int              `json:"total_active" example:"5"`
	TotalPending   int              `json:"total_pending" example:"10"`
	TotalScheduled int              `json:"total_scheduled" example:"15"`
	TotalRetry     int              `json:"total_retry" example:"2"`
	TotalArchived  int              `json:"total_archived" example:"100"`
	TotalCompleted int              `json:"total_completed" example:"1000"`
	TotalProcessed int              `json:"total_processed" example:"1200"`
	Queues         []AsynqQueueInfo `json:"queues"`
}

// AsynqQueueInfo represents information about an Asynq queue
type AsynqQueueInfo struct {
	Name      string    `json:"name" example:"default"`
	Size      int       `json:"size" example:"25"`
	Active    int       `json:"active" example:"5"`
	Pending   int       `json:"pending" example:"10"`
	Scheduled int       `json:"scheduled" example:"5"`
	Retry     int       `json:"retry" example:"2"`
	Archived  int       `json:"archived" example:"3"`
	Completed int       `json:"completed" example:"100"`
	Processed int       `json:"processed" example:"120"`
	Paused    bool      `json:"paused" example:"false"`
	Timestamp time.Time `json:"timestamp" example:"2025-12-18T10:00:00Z"`
}

// AsynqTaskListResponse represents a list of Asynq tasks
type AsynqTaskListResponse struct {
	Tasks      []AsynqTaskResponse `json:"tasks"`
	TotalCount int                 `json:"total_count" example:"50"`
	Page       int                 `json:"page" example:"1"`
	Limit      int                 `json:"limit" example:"20"`
}

// AsynqTaskResponse represents an Asynq task
type AsynqTaskResponse struct {
	ID            string     `json:"id" example:"d:default:t:123"`
	Type          string     `json:"type" example:"task:content:schedule"`
	Queue         string     `json:"queue" example:"default"`
	State         string     `json:"state" example:"scheduled"`
	MaxRetry      int        `json:"max_retry" example:"3"`
	Retried       int        `json:"retried" example:"0"`
	LastErr       string     `json:"last_err,omitempty" example:""`
	NextProcessAt time.Time  `json:"next_process_at" example:"2025-12-18T15:00:00Z"`
	Timeout       int        `json:"timeout" example:"1800"` // seconds
	Deadline      *time.Time `json:"deadline,omitempty" example:"2025-12-18T16:00:00Z"`
	CompletedAt   *time.Time `json:"completed_at,omitempty" example:"2025-12-18T15:01:00Z"`
	Result        string     `json:"result,omitempty" example:""`
	Payload       any        `json:"payload,omitempty"`
}

// AsynqQueueStatsResponse represents detailed statistics for a queue
type AsynqQueueStatsResponse struct {
	Name           string    `json:"name" example:"default"`
	Size           int       `json:"size" example:"25"`
	Active         int       `json:"active" example:"5"`
	Pending        int       `json:"pending" example:"10"`
	Scheduled      int       `json:"scheduled" example:"5"`
	Retry          int       `json:"retry" example:"2"`
	Archived       int       `json:"archived" example:"3"`
	Completed      int       `json:"completed" example:"100"`
	Processed      int       `json:"processed" example:"120"`
	Failed         int       `json:"failed" example:"20"`
	Paused         bool      `json:"paused" example:"false"`
	Timestamp      time.Time `json:"timestamp" example:"2025-12-18T10:00:00Z"`
	ProcessedToday int       `json:"processed_today" example:"50"`
	CompletedToday int       `json:"completed_today" example:"45"`
	FailedToday    int       `json:"failed_today" example:"5"`
	AvgProcessTime float64   `json:"avg_process_time" example:"2.5"` // seconds
}
