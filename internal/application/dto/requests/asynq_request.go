package requests

// AsynqTaskFilterRequest for filtering Asynq tasks
type AsynqTaskFilterRequest struct {
	Queue string `form:"queue" binding:"required" example:"default"`
	State string `form:"state" binding:"required,oneof=scheduled pending active retry archived" example:"scheduled"`
	Limit int    `form:"limit" binding:"omitempty,min=1,max=100" example:"20"`
	Page  int    `form:"page" binding:"omitempty,min=1" example:"1"`
}

// AsynqDeleteTaskRequest for deleting an Asynq task
type AsynqDeleteTaskRequest struct {
	Queue  string `json:"queue" binding:"required" example:"default"`
	TaskID string `json:"task_id" binding:"required" example:"d:default:t:task_id"`
	State  string `json:"state" binding:"required,oneof=scheduled pending retry archived" example:"scheduled"`
}

// AsynqRunTaskRequest for running an Asynq task immediately
type AsynqRunTaskRequest struct {
	Queue  string `json:"queue" binding:"required" example:"default"`
	TaskID string `json:"task_id" binding:"required" example:"d:default:t:task_id"`
}

// AsynqArchiveTaskRequest for archiving an Asynq task
type AsynqArchiveTaskRequest struct {
	Queue  string `json:"queue" binding:"required" example:"default"`
	TaskID string `json:"task_id" binding:"required" example:"d:default:t:task_id"`
}

// AsynqQueueActionRequest for queue actions (pause/unpause)
type AsynqQueueActionRequest struct {
	Queue string `json:"queue" binding:"required" example:"default"`
}
