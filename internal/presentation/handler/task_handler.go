package handler

import (
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/application/interfaces/iservice"
	"core-backend/internal/domain/enum"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

type TaskHandler struct {
	TaskService iservice.TaskService
	UnitOfWork  irepository.UnitOfWork
	Validator   *validator.Validate
}

func NewTaskHandler(taskService iservice.TaskService, unitOfWork irepository.UnitOfWork, validate *validator.Validate) *TaskHandler {
	return &TaskHandler{
		TaskService: taskService,
		UnitOfWork:  unitOfWork,
		Validator:   validate,
	}
}

// UpdateTaskStateRequest defines the request body for updating a task state
type UpdateTaskStateRequest struct {
	State string `json:"state" validate:"required,oneof=TODO IN_PROGRESS CANCELLED RECAP DONE"`
}

// UpdateTaskState godoc
// @Summary      Update Task State
// @Description  Move a task to a target state (TODO, IN_PROGRESS, CANCELLED, RECAP, DONE)
// @Tags         Tasks
// @Accept       json
// @Produce      json
// @Param        id    path   string                  true  "Task ID (UUID)"
// @Param        body  body   UpdateTaskStateRequest  true  "Target state payload"
// @Success      200   {object} responses.APIResponse  "Task state updated"
// @Failure      400   {object} responses.APIResponse  "Invalid request"
// @Failure      404   {object} responses.APIResponse  "Task not found"
// @Failure      409   {object} responses.APIResponse  "Invalid state transition"
// @Failure      500   {object} responses.APIResponse  "Internal server error"
// @Security     BearerAuth
// @Router       /api/v1/tasks/{id}/state [patch]
func (h *TaskHandler) UpdateTaskState(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("invalid task id: "+err.Error(), http.StatusBadRequest))
		return
	}

	var req UpdateTaskStateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("invalid request body: "+err.Error(), http.StatusBadRequest))
		return
	}
	if err := h.Validator.Struct(&req); err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("validation failed: "+err.Error(), http.StatusBadRequest))
		return
	}

	target := enum.TaskStatus(req.State)
	if !target.IsValid() {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("invalid target state", http.StatusBadRequest))
		return
	}

	if err := h.TaskService.MoveTaskToState(id, target); err != nil {
		// naive mapping of errors; customize if you propagate error kinds
		c.JSON(http.StatusConflict, responses.ErrorResponse("failed to move task: "+err.Error(), http.StatusConflict))
		return
	}

	c.JSON(http.StatusOK, responses.SuccessResponse("Task state updated", nil, map[string]any{
		"id":    id.String(),
		"state": target,
	}))
}
