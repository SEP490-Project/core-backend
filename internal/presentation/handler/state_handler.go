package handler

import (
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/application/interfaces/iservice"
	"core-backend/internal/domain/enum"
	"core-backend/pkg/utils"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type StateHandler struct {
	iservice.StateTransferService
	irepository.UnitOfWork
	*validator.Validate
}

func NewStateHandler(StateTransferService iservice.StateTransferService, unitOfWork irepository.UnitOfWork, validate *validator.Validate) *StateHandler {
	return &StateHandler{
		StateTransferService: StateTransferService,
		UnitOfWork:           unitOfWork,
		Validate:             validate,
	}
}

// UpdateTaskStateRequest defines the request body for updating a task state
type UpdateTaskStateRequest struct {
	State string `json:"state" validate:"required,oneof=TODO IN_PROGRESS CANCELLED RECAP DONE"`
}

// UpdateTaskState godoc
//
//	@Summary		Update Task State
//	@Description	Move a task to a target state (TODO, IN_PROGRESS, CANCELLED, RECAP, DONE)
//	@Tags			State Transfer
//	@Accept			json
//	@Produce		json
//	@Param			task_id	path		string					true	"Task ID (UUID)"
//	@Param			body	body		UpdateTaskStateRequest	true	"Target state payload"
//	@Success		200		{object}	responses.APIResponse	"Task state updated"
//	@Failure		400		{object}	responses.APIResponse	"Invalid request"
//	@Failure		404		{object}	responses.APIResponse	"Task not found"
//	@Failure		409		{object}	responses.APIResponse	"Invalid state transition"
//	@Failure		500		{object}	responses.APIResponse	"Internal server error"
//	@Security		BearerAuth
//	@Router			/api/v1/tasks/{task_id}/state [patch]
func (h *StateHandler) UpdateTaskState(c *gin.Context) {
	id, err := extractParamID(c, "task_id")
	if err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("invalid task id: "+err.Error(), http.StatusBadRequest))
		return
	}

	var req UpdateTaskStateRequest
	if err = c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("invalid request body: "+err.Error(), http.StatusBadRequest))
		return
	}
	if err = h.Struct(&req); err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("validation failed: "+err.Error(), http.StatusBadRequest))
		return
	}

	target := enum.TaskStatus(req.State)
	if !target.IsValid() {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("invalid target state", http.StatusBadRequest))
		return
	}

	// // Authorization rule: only BRAND_PARTNER can move to REVISION or APPROVED
	var roleStr *string
	roleStr, err = extractUserRoles(c)
	if err != nil {
		c.JSON(http.StatusForbidden, responses.ErrorResponse("missing role in context: "+err.Error(), http.StatusForbidden))
		return
	}

	if *roleStr == string(enum.UserRoleAdmin) {
		goto SkipAdminRoleCheck
	}

	if target == enum.TaskStatusDone {
		if *roleStr != string(enum.UserRoleBrandPartner) { // could extend to Admin if desired
			c.JSON(http.StatusForbidden, responses.ErrorResponse("only BRAND_PARTNER can move Task to DONE", http.StatusForbidden))
			return
		}
	} else if *roleStr == string(enum.UserRoleBrandPartner) {
		c.JSON(http.StatusForbidden, responses.ErrorResponse("BRAND_PARTNER do not have this permission", http.StatusForbidden))
		return
	}

SkipAdminRoleCheck:

	userID, err := extractUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("invalid user_id in context: "+err.Error(), http.StatusBadRequest))
		return
	}

	if err := h.MoveTaskToState(c.Request.Context(), id, target, userID); err != nil {
		// naive mapping of errors; customize if you propagate error kinds
		c.JSON(http.StatusConflict, responses.ErrorResponse("failed to move task: "+err.Error(), http.StatusConflict))
		return
	}

	c.JSON(http.StatusOK, responses.SuccessResponse("Task state updated", nil, map[string]any{
		"id":    id.String(),
		"state": target,
	}))
}

// UpdateProductStateRequest defines the request body for updating a product state
// swagger:model UpdateProductStateRequest
type UpdateProductStateRequest struct {
	// State is the desired target state.
	// Enum: DRAFT,SUBMITTED,REVISION,APPROVED,ACTIVED,INACTIVED
	// example: SUBMITTED
	State string `json:"state" validate:"required,oneof=DRAFT SUBMITTED REVISION APPROVED ACTIVED INACTIVED"`
}

// UpdateProductState godoc
//
//	@Summary		Update Product State
//	@Description	Move a product to a target state (DRAFT, SUBMITTED, REVISION, APPROVED, ACTIVED, INACTIVED)
//	@Tags			State Transfer
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string					true	"Product ID (UUID)"
//	@Param			body	body		UpdateTaskStateRequest	true	"Target state payload"
//	@Success		200		{object}	responses.APIResponse	"Product state updated"
//	@Failure		400		{object}	responses.APIResponse	"Invalid request"
//	@Failure		404		{object}	responses.APIResponse	"Product not found"
//	@Failure		409		{object}	responses.APIResponse	"Invalid state transition"
//	@Failure		500		{object}	responses.APIResponse	"Internal server error"
//	@Security		BearerAuth
//	@Router			/api/v1/products/{id}/state [patch]
func (h *StateHandler) UpdateProductState(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		resp := responses.ErrorResponse("invalid product id: "+err.Error(), http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, resp)
		return
	}

	var req UpdateProductStateRequest
	if err = c.ShouldBindJSON(&req); err != nil {
		resp := responses.ErrorResponse("invalid request body: "+err.Error(), http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, resp)
		return
	}

	if err = h.Struct(&req); err != nil {
		resp := responses.ErrorResponse("validation failed: "+err.Error(), http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, resp)
		return
	}

	target := enum.ProductStatus(req.State)
	if !target.IsValid() {
		resp := responses.ErrorResponse("invalid target state", http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, resp)
		return
	}

	// Authorization rule with Admin bypass
	roleVal, ok := c.Get("roles")
	if !ok || roleVal == nil {
		resp := responses.ErrorResponse("missing role in context", http.StatusForbidden)
		c.JSON(http.StatusForbidden, resp)
		return
	}
	roleStr, _ := roleVal.(string)

	if roleStr == string(enum.UserRoleAdmin) {
		goto SkipAdminRoleCheck
	}

	if target == enum.ProductStatusRevision || target == enum.ProductStatusApproved {
		if roleStr != string(enum.UserRoleBrandPartner) {
			fmtMsg := fmt.Sprintf("insufficient permission to move product to this state: %s", target.String())
			resp := responses.ErrorResponse(fmtMsg, http.StatusForbidden)
			c.JSON(http.StatusForbidden, resp)
			return
		}
	} else if roleStr == string(enum.UserRoleBrandPartner) {
		resp := responses.ErrorResponse("BRAND_PARTNER do not have this permission", http.StatusBadRequest)
		c.JSON(http.StatusForbidden, resp)
		return
	}

SkipAdminRoleCheck:

	userID, err := extractUserIDFromContext(c)
	if err != nil {
		resp := responses.ErrorResponse("invalid user_id in context: "+err.Error(), http.StatusUnauthorized)
		c.JSON(http.StatusBadRequest, resp)
		return
	}

	if err := h.MoveProductToState(c.Request.Context(), id, target, userID); err != nil {
		resp := responses.ErrorResponse("failed to move product: "+err.Error(), http.StatusConflict)
		c.JSON(http.StatusConflict, resp)
		return
	}

	resp := responses.SuccessResponse("Product state updated", nil, map[string]any{
		"id":    id.String(),
		"state": target,
	})

	c.JSON(http.StatusOK, resp)
}

// UpdateMilestoneStateRequest defines the request body for updating a milestone state
// swagger:model UpdateMilestoneStateRequest
type UpdateMilestoneStateRequest struct {
	// State is the desired target state.
	// Enum: NOT_STARTED,ON_GOING,CANCELLED,COMPLETED
	// example: ON_GOING
	State string `json:"state" validate:"required,oneof=NOT_STARTED ON_GOING CANCELLED COMPLETED"`
}

// UpdateMilestoneState godoc
//
//	@Summary		Update Milestone State
//	@Description	Move a milestone to a target state (NOT_STARTED, ON_GOING, CANCELLED, COMPLETED)
//	@Tags			State Transfer
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string						true	"Milestone ID (UUID)"
//	@Param			body	body		UpdateMilestoneStateRequest	true	"Target state payload"
//	@Success		200		{object}	responses.APIResponse		"Milestone state updated"
//	@Failure		400		{object}	responses.APIResponse		"Invalid request"
//	@Failure		404		{object}	responses.APIResponse		"Milestone not found"
//	@Failure		409		{object}	responses.APIResponse		"Invalid state transition"
//	@Failure		500		{object}	responses.APIResponse		"Internal server error"
//	@Security		BearerAuth
//	@Router			/api/v1/milestones/{id}/state [patch]
func (h *StateHandler) UpdateMilestoneState(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("invalid milestone id: "+err.Error(), http.StatusBadRequest))
		return
	}

	var req UpdateMilestoneStateRequest
	if err = c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("invalid request body: "+err.Error(), http.StatusBadRequest))
		return
	}
	if err = h.Struct(&req); err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("validation failed: "+err.Error(), http.StatusBadRequest))
		return
	}

	target := enum.MilestoneStatus(req.State)
	if !target.IsValid() {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("invalid target state", http.StatusBadRequest))
		return
	}

	// Authorization logic (reuse task roles) - Admin bypass
	roleVal, ok := c.Get("roles")
	if !ok || roleVal == nil {
		c.JSON(http.StatusForbidden, responses.ErrorResponse("missing role in context", http.StatusForbidden))
		return
	}
	roleStr, _ := roleVal.(string)

	// Example rule: only Admin or Brand Partner can cancel a milestone (adjust as needed)
	if target == enum.MilestoneStatusCancelled && roleStr != string(enum.UserRoleAdmin) && roleStr != string(enum.UserRoleBrandPartner) {
		c.JSON(http.StatusForbidden, responses.ErrorResponse("insufficient permission to cancel milestone", http.StatusForbidden))
		return
	}

	userID, err := extractUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("invalid user_id in context: "+err.Error(), http.StatusBadRequest))
		return
	}

	if err := h.MoveMileStoneToState(c.Request.Context(), id, target, userID); err != nil {
		c.JSON(http.StatusConflict, responses.ErrorResponse("failed to move milestone: "+err.Error(), http.StatusConflict))
		return
	}

	c.JSON(http.StatusOK, responses.SuccessResponse("Milestone state updated", nil, map[string]any{
		"id":    id.String(),
		"state": target,
	}))
}

// UpdateContractState godoc
//
//	@Summary		Update Contract State
//	@Description	Move a contract to a target state (DRAFT, APPROVED, ACTIVE, COMPLETED, TERMINATED, INACTIVE)
//	@Tags			State Transfer
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string								true	"Contract ID"	format(uuid)
//	@Param			request	body		requests.UpdateContractStateRequest	true	"Target state payload"
//	@Success		200		{object}	responses.APIResponse				"Contract state updated"
//	@Failure		400		{object}	responses.APIResponse				"Invalid request"
//	@Failure		401		{object}	responses.APIResponse				"Unauthorized"
//	@Failure		403		{object}	responses.APIResponse				"Forbidden"
//	@Failure		404		{object}	responses.APIResponse				"Contract not found"
//	@Failure		409		{object}	responses.APIResponse				"Invalid state transition"
//	@Failure		500		{object}	responses.APIResponse				"Internal server error"
//	@Security		BearerAuth
//	@Router			/api/v1/contracts/{id}/state [patch]
func (h *StateHandler) UpdateContractState(c *gin.Context) {
	userID, err := extractUserID(c)
	if err != nil {
		responses := responses.ErrorResponse("Unauthorized: "+err.Error(), http.StatusUnauthorized)
		c.JSON(http.StatusUnauthorized, responses)
		return
	}
	userRole, err := extractUserRoles(c)
	if err != nil {
		responses := responses.ErrorResponse("Unauthorized: "+err.Error(), http.StatusUnauthorized)
		c.JSON(http.StatusUnauthorized, responses)
		return
	}

	var contractID uuid.UUID
	contractID, err = extractParamID(c, "id")
	if err != nil {
		response := responses.ErrorResponse("Invalid contract ID: "+err.Error(), http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	var req requests.UpdateContractStateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		responses := responses.ErrorResponse("Invalid request body: "+err.Error(), http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, responses)
		return
	}
	if err := h.Struct(&req); err != nil {
		response := processValidationError(err)
		c.JSON(http.StatusBadRequest, response)
		return
	}
	targetState := enum.ContractStatus(req.State)
	if !targetState.IsValid() {
		responses := responses.ErrorResponse("Invalid target state", http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, responses)
		return
	}

	switch *userRole {
	case enum.UserRoleBrandPartner.String():
		if targetState != enum.ContractStatusApproved {
			resposnes := responses.ErrorResponse("Forbidden: BRAND_PARTNER can only move contract to APPROVED", http.StatusForbidden)
			c.JSON(http.StatusForbidden, resposnes)
			return
		}
	case enum.UserRoleMarketingStaff.String():
		if targetState != enum.ContractStatusCompleted && targetState != enum.ContractStatusTerminated {
			resposnes := responses.ErrorResponse("Forbidden: MARKETING_STAFF can only move contract to COMPLETED or TERMINATED", http.StatusForbidden)
			c.JSON(http.StatusForbidden, resposnes)
			return
		}
	}

	uow := h.Begin(c.Request.Context())
	defer func() {
		if r := recover(); r != nil {
			_ = uow.Rollback()
			zap.L().Error("panic recovered in MoveContractToState", zap.Any("recover", r))
			panic(r)
		}
	}()

	if err := h.MoveContractToState(c.Request.Context(), uow, contractID, targetState, userID); err != nil {
		uow.Rollback()
		response := responses.ErrorResponse("Failed to move contract: "+err.Error(), http.StatusInternalServerError)
		c.JSON(http.StatusInternalServerError, response)
		return
	}

	if err := uow.Commit(); err != nil {
		uow.Rollback()
		response := responses.ErrorResponse("Failed to commit transaction: "+err.Error(), http.StatusInternalServerError)
		c.JSON(http.StatusInternalServerError, response)
		return
	}
	response := responses.SuccessResponse("Contract state updated", utils.IntPtr(http.StatusOK), map[string]any{
		"id":    contractID.String(),
		"state": req.State,
	})
	c.JSON(http.StatusOK, response)
}

func extractUserIDFromContext(c *gin.Context) (uuid.UUID, error) {
	userIDVal, ok := c.Get("user_id")
	if !ok || userIDVal == nil {
		return uuid.Nil, http.ErrNoCookie
	}
	userIDStr, _ := userIDVal.(string)
	return uuid.Parse(userIDStr)
}
