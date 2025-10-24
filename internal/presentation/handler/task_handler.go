package handler

import (
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/application/interfaces/iservice"
	"core-backend/pkg/utils"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

type TaskHandler struct {
	taskService iservice.TaskService
	unitOfWork  irepository.UnitOfWork
	validator   *validator.Validate
}

func NewTaskHandler(taskService iservice.TaskService, unitOfWork irepository.UnitOfWork) *TaskHandler {
	validator := validator.New()
	return &TaskHandler{
		taskService: taskService,
		unitOfWork:  unitOfWork,
		validator:   validator,
	}
}

// GetTasksByFilter godoc
//
//	@Summary		Get Tasks by Filter
//	@Description	Retrieve a list of tasks based on filter criteria and return them in a paginated response.
//	@Tags			Tasks
//	@Accept			json
//	@Produce		json
//	@Param			page				query		int									false	"Page number"					default(1)
//	@Param			limit				query		int									false	"Number of items per page"		default(10)
//	@Param			created_by_id		query		string								false	"Filter by creator ID"			format(uuid)
//	@Param			assigned_to_id		query		string								false	"Filter by assignee ID"			format(uuid)
//	@Param			milestone_id		query		string								false	"Filter by milestone ID"		format(uuid)
//	@Param			deadline_from_date	query		string								false	"Filter by deadline from date"	format(date-time)
//	@Param			deadline_to_date	query		string								false	"Filter by deadline to date"	format(date-time)
//	@Param			updated_from_date	query		string								false	"Filter by updated from date"	format(date-time)
//	@Param			updated_to_date		query		string								false	"Filter by updated to date"		format(date-time)
//	@Param			status				query		string								false	"Filter by task status"			Enums(TODO, IN_PROGRESS, CANCELLED, RECAP, DONE)
//	@Param			type				query		string								false	"Filter by task type"			Enums(PRODUCT, CONTENT, EVENT, OTHER)
//	@Param			sort_by				query		string								false	"Field to sort by"
//	@Param			sort_order			query		string								false	"Sort order (asc or desc)"
//	@Success		200					{object}	responses.PaginationTaskResponse	"Tasks retrieved successfully"
//	@Failure		400					{object}	responses.APIResponse				"Invalid query parameters"
//	@Failure		500					{object}	responses.APIResponse				"Internal server error"
//	@Security		BearerAuth
//	@Router			/api/v1/tasks [get]
func (h *TaskHandler) GetTasksByFilter(c *gin.Context) {
	var req requests.TaskFilterRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest,
			responses.ErrorResponse("Invalid query parameters: "+err.Error(), http.StatusBadRequest))
		return
	}
	if err := h.validator.Struct(&req); err != nil {
		c.JSON(http.StatusBadRequest, processValidationError(err))
		return
	}

	taskResponse, total, err := h.taskService.GetTaskByFilter(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError,
			responses.ErrorResponse("Failed to get tasks: "+err.Error(), http.StatusInternalServerError))
		return
	}

	response := responses.NewPaginationResponse(
		"Tasks retrieved successfully",
		http.StatusOK,
		taskResponse,
		responses.Pagination{
			Page:  req.Page,
			Limit: req.Limit,
			Total: total,
		},
	)
	c.JSON(http.StatusOK, response)
}

// GetTaskByID godoc
//
//	@Summary		Get Task by ID
//	@Description	Retrieve a task by its ID and return its details in the response.
//	@Tags			Tasks
//	@Accept			json
//	@Produce		json
//	@Param			task_id	path		string												true	"Task ID"	format(uuid)
//	@Success		200		{object}	responses.APIResponse{data=responses.TaskResponse}	"Task retrieved successfully"
//	@Failure		400		{object}	responses.APIResponse								"Invalid task ID"
//	@Failure		404		{object}	responses.APIResponse								"Task not found"
//	@Failure		500		{object}	responses.APIResponse								"Internal server error"
//	@Security		BearerAuth
//	@Router			/api/v1/tasks/{task_id} [get]
func (h *TaskHandler) GetTaskByID(c *gin.Context) {
	taskID, err := extractParamID(c, "task_id")
	if err != nil {
		c.JSON(http.StatusBadRequest,
			responses.ErrorResponse("Invalid task ID: "+err.Error(), http.StatusBadRequest))
		return
	}

	var taskResponse *responses.TaskResponse
	taskResponse, err = h.taskService.GetTaskByID(c.Request.Context(), taskID)
	if err != nil {
		var statusCode int
		var response *responses.APIResponse

		switch err.Error() {
		case "task not found":
			statusCode = http.StatusNotFound
			response = responses.ErrorResponse("Task not found", statusCode)
		default:
			statusCode = http.StatusInternalServerError
			response = responses.ErrorResponse("Failed to get task: "+err.Error(), statusCode)
		}
		c.JSON(statusCode, response)
		return
	}

	c.JSON(http.StatusOK, responses.SuccessResponse("Task retrieved successfully", utils.PtrOrNil(http.StatusOK), taskResponse))
}
