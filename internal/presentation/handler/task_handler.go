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
	"github.com/google/uuid"
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
//	@Param			campaign_id			query		string								false	"Filter by campaign ID"			format(uuid)
//	@Param			contract_id			query		string								false	"Filter by contract ID"			format(uuid)
//	@Param			deadline_from_date	query		string								false	"Filter by deadline from date"	format(date-time)
//	@Param			deadline_to_date	query		string								false	"Filter by deadline to date"	format(date-time)
//	@Param			updated_from_date	query		string								false	"Filter by updated from date"	format(date-time)
//	@Param			updated_to_date		query		string								false	"Filter by updated to date"		format(date-time)
//	@Param			status				query		string								false	"Filter by task status"			Enums(TODO, IN_PROGRESS, CANCELLED, RECAP, DONE)
//	@Param			has_content			query		bool								false	"Filter by tasks that have associated content"
//	@Param			has_product			query		bool								false	"Filter by tasks that have associated product"
//	@Param			type				query		string								false	"Filter by task type"	Enums(PRODUCT, CONTENT, EVENT, OTHER)
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

	taskResponse, total, err := h.taskService.GetTaskByFilter(c.Request.Context(), &req)
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

// GetTasksByProfile godoc
//
//	@Summary		Get Tasks by Profile
//	@Description	Retrieve a list of tasks based on filter criteria and return them in a paginated response.
//	@Tags			Tasks
//	@Accept			json
//	@Produce		json
//	@Param			page				query		int									false	"Page number"					default(1)
//	@Param			limit				query		int									false	"Number of items per page"		default(10)
//	@Param			created_by_id		query		string								false	"Filter by creator ID"			format(uuid)
//	@Param			assigned_to_id		query		string								false	"Filter by assignee ID"			format(uuid)
//	@Param			milestone_id		query		string								false	"Filter by milestone ID"		format(uuid)
//	@Param			campaign_id			query		string								false	"Filter by campaign ID"			format(uuid)
//	@Param			contract_id			query		string								false	"Filter by contract ID"			format(uuid)
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
//	@Router			/api/v1/tasks/profile [get]
func (h *TaskHandler) GetTasksByProfile(c *gin.Context) {
	userID, err := extractUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized,
			responses.ErrorResponse("Unauthorized: "+err.Error(), http.StatusUnauthorized))
		return
	}

	var filterReq requests.TaskFilterRequest
	if err = c.ShouldBindQuery(&filterReq); err != nil {
		c.JSON(http.StatusBadRequest,
			responses.ErrorResponse("Invalid query parameters: "+err.Error(), http.StatusBadRequest))
		return
	}
	if err = h.validator.Struct(&filterReq); err != nil {
		c.JSON(http.StatusBadRequest, processValidationError(err))
		return
	}
	filterReq.AssignedToID = utils.PtrOrNil(userID.String())

	var taskResponses []responses.TaskListResponse
	var total int64
	taskResponses, total, err = h.taskService.GetTaskByFilter(c.Request.Context(), &filterReq)
	if err != nil {
		response := responses.ErrorResponse("Failed to get tasks: "+err.Error(), http.StatusInternalServerError)
		c.JSON(http.StatusInternalServerError, response)
		return
	}

	response := responses.NewPaginationResponse(
		"Tasks retrieved successfully",
		http.StatusOK,
		taskResponses,
		responses.Pagination{
			Page:  filterReq.Page,
			Limit: filterReq.Limit,
			Total: total,
		},
	)
	c.JSON(http.StatusOK, response)
}

// GetTasksByContractID godoc
//
//	@Summary		Get Tasks by Contract ID
//	@Description	Retrieve a list of tasks based on filter criteria and return them in a paginated response. This is usually used by BRAND_PARTNER
//	@Tags			Tasks
//	@Accept			json
//	@Produce		json
//	@Param			page				query		int									false	"Page number"					default(1)
//	@Param			limit				query		int									false	"Number of items per page"		default(10)
//	@Param			milestone_id		query		string								false	"Filter by milestone ID"		format(uuid)
//	@Param			campaign_id			query		string								false	"Filter by campaign ID"			format(uuid)
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
//	@Router			/api/v1/tasks/contract/{contract_id} [get]
func (h *TaskHandler) GetTasksByContractID(c *gin.Context) {
	contractID, err := extractParamID(c, "contract_id")
	if err != nil {
		c.JSON(http.StatusBadRequest,
			responses.ErrorResponse("Invalid contract ID: "+err.Error(), http.StatusBadRequest))
		return
	}

	var filterReq requests.TaskFilterRequest
	if err = c.ShouldBindQuery(&filterReq); err != nil {
		c.JSON(http.StatusBadRequest,
			responses.ErrorResponse("Invalid query parameters: "+err.Error(), http.StatusBadRequest))
		return
	}
	if err = h.validator.Struct(&filterReq); err != nil {
		c.JSON(http.StatusBadRequest, processValidationError(err))
		return
	}
	filterReq.ContractID = utils.PtrOrNil(contractID.String())

	var taskResponses []responses.TaskListResponse
	var total int64
	taskResponses, total, err = h.taskService.GetTaskByFilter(c.Request.Context(), &filterReq)
	if err != nil {
		response := responses.ErrorResponse("Failed to get tasks: "+err.Error(), http.StatusInternalServerError)
		c.JSON(http.StatusInternalServerError, response)
		return
	}

	response := responses.NewPaginationResponse(
		"Tasks retrieved successfully",
		http.StatusOK,
		taskResponses,
		responses.Pagination{
			Page:  filterReq.Page,
			Limit: filterReq.Limit,
			Total: total,
		},
	)
	c.JSON(http.StatusOK, response)

}

// AssignTask godoc
//
//	@Summary		Assign Task
//	@Description	Assign a task to a user.
//	@Tags			Tasks
//	@Accept			json
//	@Produce		json
//	@Param			task_id			path		string					true	"Task ID"				format(uuid)
//	@Param			assigned_to_id	path		string					true	"Assigned to user ID"	format(uuid)
//	@Success		200				{object}	responses.TaskResponse	"Task retrieved successfully"
//	@Failure		400				{object}	responses.APIResponse	"Invalid task ID"
//	@Failure		400				{object}	responses.APIResponse	"Invalid assigned_to_id"
//	@Failure		400				{object}	responses.APIResponse	"Invalid updated_by_id"
//	@Failure		404				{object}	responses.APIResponse	"Task not found"
//	@Failure		404				{object}	responses.APIResponse	"User not found"
//	@Failure		500				{object}	responses.APIResponse	"Internal server error"
//	@Security		BearerAuth
//	@Router			/api/v1/tasks/{task_id}/assign/{assigned_to_id} [patch]
func (h *TaskHandler) AssignTask(c *gin.Context) {
	var (
		err          error
		taskID       uuid.UUID
		assignedToID uuid.UUID
		updatedByID  uuid.UUID
	)

	taskID, err = extractParamID(c, "task_id")
	if err != nil {
		c.JSON(http.StatusBadRequest,
			responses.ErrorResponse("Invalid task ID: "+err.Error(), http.StatusBadRequest))
		return
	}
	assignedToID, err = extractParamID(c, "assigned_to_id")
	if err != nil {
		c.JSON(http.StatusBadRequest,
			responses.ErrorResponse("Invalid assigned_to_id: "+err.Error(), http.StatusBadRequest))
		return
	}
	updatedByID, err = extractUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized,
			responses.ErrorResponse("Unauthorized: "+err.Error(), http.StatusUnauthorized))
		return
	}

	uow := h.unitOfWork.Begin(c.Request.Context())
	var taskResponse *responses.TaskResponse
	taskResponse, err = h.taskService.AssignTask(c.Request.Context(), uow, taskID, assignedToID, updatedByID)
	if err != nil {
		uow.Rollback()
		var response *responses.APIResponse
		var statusCode int
		switch err.Error() {
		case "task not found":
			statusCode = http.StatusNotFound
			response = responses.ErrorResponse("Task not found", statusCode)
		case "user not found":
			statusCode = http.StatusNotFound
			response = responses.ErrorResponse("User not found", statusCode)
		default:
			statusCode = http.StatusInternalServerError
			response = responses.ErrorResponse("Failed to assign task: "+err.Error(), statusCode)
		}
		c.JSON(statusCode, response)
		return
	}

	uow.Commit()
	c.JSON(http.StatusOK, responses.SuccessResponse("Task assigned successfully", utils.PtrOrNil(http.StatusOK), taskResponse))
}

// CreateTask godoc
//
//	@Summary		Create Task
//	@Description	Create a new task.
//	@Tags			Tasks
//	@Accept			json
//	@Produce		json
//	@Param			task	body		requests.CreateTaskRequest							true	"Task creation payload"
//	@Success		200		{object}	responses.APIResponse{data=responses.TaskResponse}	"Task created successfully"
//	@Failure		400		{object}	responses.APIResponse								"Invalid request body"
//	@Failure		400		{object}	responses.APIResponse								"Validation error"
//	@Failure		500		{object}	responses.APIResponse								"Internal server error"
//	@Security		BearerAuth
//	@Router			/api/v1/tasks [post]
func (h *TaskHandler) CreateTask(c *gin.Context) {
	var (
		err           error
		createRequest requests.CreateTaskRequest
		createdByID   uuid.UUID
	)
	if err = c.ShouldBindJSON(&createRequest); err != nil {
		c.JSON(http.StatusBadRequest,
			responses.ErrorResponse("Invalid request body: "+err.Error(), http.StatusBadRequest))
		return
	}
	if err = h.validator.Struct(&createRequest); err != nil {
		c.JSON(http.StatusBadRequest, processValidationError(err))
		return
	}

	createdByID, err = extractUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized,
			responses.ErrorResponse("Unauthorized: "+err.Error(), http.StatusUnauthorized))
		return
	}
	createRequest.CreatedByID = createdByID.String()

	var taskResponse *responses.TaskResponse
	taskResponse, err = h.taskService.CreateTask(c.Request.Context(), h.unitOfWork, &createRequest)
	if err != nil {
		response := responses.ErrorResponse("Failed to create task: "+err.Error(), http.StatusInternalServerError)
		c.JSON(http.StatusInternalServerError, response)
		return
	}

	c.JSON(http.StatusOK, responses.SuccessResponse("Task created successfully", utils.PtrOrNil(http.StatusOK), taskResponse))
}

// UpdateTaskByID godoc
//
//	@Summary		Update Task by ID
//	@Description	Update an existing task.
//	@Tags			Tasks
//	@Accept			json
//	@Produce		json
//	@Param			task_id	path		string						true	"Task ID"	format(uuid)
//	@Param			task	body		requests.UpdateTaskRequest	true	"Task update payload"
//	@Success		200		{object}	responses.TaskResponse		"Task updated successfully"
//	@Failure		400		{object}	responses.APIResponse		"Invalid task ID"
//	@Failure		400		{object}	responses.APIResponse		"Invalid request body"
//	@Failure		400		{object}	responses.APIResponse		"Invalid updated_by_id"
//	@Failure		404		{object}	responses.APIResponse		"Task not found"
//	@Failure		500		{object}	responses.APIResponse		"Internal server error"
//	@Security		BearerAuth
//	@Router			/api/v1/tasks/{task_id} [put]
func (h *TaskHandler) UpdateTaskByID(c *gin.Context) {
	var (
		err           error
		taskID        uuid.UUID
		updateRequest requests.UpdateTaskRequest
		updatedByID   uuid.UUID
	)
	updatedByID, err = extractUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized,
			responses.ErrorResponse("Unauthorized: "+err.Error(), http.StatusUnauthorized))
		return
	}

	taskID, err = extractParamID(c, "task_id")
	if err != nil {
		c.JSON(http.StatusBadRequest,
			responses.ErrorResponse("Invalid task ID: "+err.Error(), http.StatusBadRequest))
		return
	}
	if err = c.ShouldBindJSON(&updateRequest); err != nil {
		c.JSON(http.StatusBadRequest,
			responses.ErrorResponse("Invalid request body: "+err.Error(), http.StatusBadRequest))
		return
	}
	if err = h.validator.Struct(&updateRequest); err != nil {
		c.JSON(http.StatusBadRequest, processValidationError(err))
		return
	}

	updateRequest.ID = taskID.String()
	updateRequest.UpdatedByID = updatedByID.String()

	uow := h.unitOfWork.Begin(c.Request.Context())
	var taskResponse *responses.TaskResponse
	taskResponse, err = h.taskService.UpdateTaskByID(c.Request.Context(), uow, taskID, &updateRequest)
	if err != nil {
		uow.Rollback()
		var response *responses.APIResponse
		var statusCode int
		switch err.Error() {
		case "task not found":
			statusCode = http.StatusNotFound
			response = responses.ErrorResponse("Task not found", statusCode)
		default:
			statusCode = http.StatusInternalServerError
			response = responses.ErrorResponse("Failed to update task: "+err.Error(), statusCode)
		}
		c.JSON(statusCode, response)
		return
	}

	uow.Commit()
	c.JSON(http.StatusOK, responses.SuccessResponse("Task updated successfully", utils.PtrOrNil(http.StatusOK), taskResponse))
}

// DeleteTask godoc
//
//	@Summary		Delete Task
//	@Description	Delete a task by its ID.
//	@Tags			Tasks
//	@Accept			json
//	@Produce		json
//	@Param			task_id	path		string					true	"Task ID"	format(uuid)
//	@Success		200		{object}	responses.APIResponse	"Task deleted successfully"
//	@Failure		400		{object}	responses.APIResponse	"Invalid task ID"
//	@Failure		404		{object}	responses.APIResponse	"Task not found"
//	@Failure		500		{object}	responses.APIResponse	"Internal server error"
//	@Security		BearerAuth
//	@Router			/api/v1/tasks/{task_id} [delete]
func (h *TaskHandler) DeleteTask(c *gin.Context) {
	var (
		err    error
		taskID uuid.UUID
	)
	taskID, err = extractParamID(c, "task_id")
	if err != nil {
		c.JSON(http.StatusBadRequest,
			responses.ErrorResponse("Invalid task ID: "+err.Error(), http.StatusBadRequest))
		return
	}

	uow := h.unitOfWork.Begin(c.Request.Context())
	var taskResponse *responses.TaskResponse
	err = h.taskService.DeleteTask(c.Request.Context(), uow, taskID)
	if err != nil {
		uow.Rollback()
		response := responses.ErrorResponse("Failed to delete task: "+err.Error(), http.StatusInternalServerError)
		c.JSON(http.StatusInternalServerError, response)
		return
	}
	uow.Commit()

	c.JSON(http.StatusOK, responses.SuccessResponse("Task deleted successfully", utils.PtrOrNil(http.StatusOK), taskResponse))
}
