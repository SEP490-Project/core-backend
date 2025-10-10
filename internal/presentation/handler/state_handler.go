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
//	@Param			id		path		string					true	"Task ID (UUID)"
//	@Param			body	body		UpdateTaskStateRequest	true	"Target state payload"
//	@Success		200		{object}	responses.APIResponse	"Task state updated"
//	@Failure		400		{object}	responses.APIResponse	"Invalid request"
//	@Failure		404		{object}	responses.APIResponse	"Task not found"
//	@Failure		409		{object}	responses.APIResponse	"Invalid state transition"
//	@Failure		500		{object}	responses.APIResponse	"Internal server error"
//	@Security		BearerAuth
//	@Router			/api/v1/tasks/{id}/state [patch]
func (h *StateHandler) UpdateTaskState(c *gin.Context) {
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
	if err := h.Validate.Struct(&req); err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("validation failed: "+err.Error(), http.StatusBadRequest))
		return
	}

	target := enum.TaskStatus(req.State)
	if !target.IsValid() {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("invalid target state", http.StatusBadRequest))
		return
	}

	// Authorization rule: only BRAND_PARTNER can move to REVISION or APPROVED
	roleVal, ok := c.Get("roles")
	if !ok || roleVal == nil {
		c.JSON(http.StatusForbidden, responses.ErrorResponse("missing role in context", http.StatusForbidden))
		return
	}

	roleStr, _ := roleVal.(string)

	if roleStr == string(enum.UserRoleAdmin) {
		goto SkipAdminRoleCheck
	}

	if target == enum.TaskStatusDone {
		if roleStr != string(enum.UserRoleBrandPartner) { // could extend to Admin if desired
			c.JSON(http.StatusForbidden, responses.ErrorResponse("only BRAND_PARTNER can move Task to DONE", http.StatusForbidden))
			return
		}
	} else if roleStr == string(enum.UserRoleBrandPartner) {
		c.JSON(http.StatusForbidden, responses.ErrorResponse("BRAND_PARTNER do not have this permission", http.StatusForbidden))
		return
	}

SkipAdminRoleCheck:

	userId, err := extractUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("invalid user_id in context: "+err.Error(), http.StatusBadRequest))
		return
	}

	if err := h.StateTransferService.MoveTaskToState(id, target, userId); err != nil {
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
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("invalid product id: "+err.Error(), http.StatusBadRequest))
		return
	}

	var req UpdateProductStateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("invalid request body: "+err.Error(), http.StatusBadRequest))
		return
	}
	if err := h.Validate.Struct(&req); err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("validation failed: "+err.Error(), http.StatusBadRequest))
		return
	}

	target := enum.ProductStatus(req.State)
	if !target.IsValid() {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("invalid target state", http.StatusBadRequest))
		return
	}

	// Authorization rule with Admin bypass
	roleVal, ok := c.Get("roles")
	if !ok || roleVal == nil {
		c.JSON(http.StatusForbidden, responses.ErrorResponse("missing role in context", http.StatusForbidden))
		return
	}
	roleStr, _ := roleVal.(string)

	if roleStr == string(enum.UserRoleAdmin) {
		goto SkipAdminRoleCheck
	}

	if target == enum.ProductStatusRevision || target == enum.ProductStatusApproved {
		if roleStr != string(enum.UserRoleBrandPartner) {
			c.JSON(http.StatusForbidden, responses.ErrorResponse("only BRAND_PARTNER can move product to REVISION or APPROVED", http.StatusForbidden))
			return
		}
	} else if roleStr == string(enum.UserRoleBrandPartner) {
		c.JSON(http.StatusForbidden, responses.ErrorResponse("BRAND_PARTNER do not have this permission", http.StatusForbidden))
		return
	}

SkipAdminRoleCheck:

	userId, err := extractUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("invalid user_id in context: "+err.Error(), http.StatusBadRequest))
		return
	}

	if err := h.StateTransferService.MoveProductToState(id, target, userId); err != nil {
		c.JSON(http.StatusConflict, responses.ErrorResponse("failed to move product: "+err.Error(), http.StatusConflict))
		return
	}

	c.JSON(http.StatusOK, responses.SuccessResponse("Product state updated", nil, map[string]any{
		"id":    id.String(),
		"state": target,
	}))
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
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("invalid request body: "+err.Error(), http.StatusBadRequest))
		return
	}
	if err := h.Validate.Struct(&req); err != nil {
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

	userId, err := extractUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("invalid user_id in context: "+err.Error(), http.StatusBadRequest))
		return
	}

	if err := h.StateTransferService.MoveMileStoneToState(id, target, userId); err != nil {
		c.JSON(http.StatusConflict, responses.ErrorResponse("failed to move milestone: "+err.Error(), http.StatusConflict))
		return
	}

	c.JSON(http.StatusOK, responses.SuccessResponse("Milestone state updated", nil, map[string]any{
		"id":    id.String(),
		"state": target,
	}))
}

func extractUserIDFromContext(c *gin.Context) (uuid.UUID, error) {
	userIDVal, ok := c.Get("user_id")
	if !ok || userIDVal == nil {
		return uuid.Nil, http.ErrNoCookie
	}
	userIDStr, _ := userIDVal.(string)
	return uuid.Parse(userIDStr)
}
