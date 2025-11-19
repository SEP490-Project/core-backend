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
	"os"
	"path/filepath"
	"time"

	"github.com/aws/smithy-go/ptr"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type StateHandler struct {
	iservice.StateTransferService
	irepository.UnitOfWork
	*validator.Validate
	fileService iservice.FileService
}

func NewStateHandler(StateTransferService iservice.StateTransferService, unitOfWork irepository.UnitOfWork, validate *validator.Validate, fileService iservice.FileService) *StateHandler {
	return &StateHandler{
		StateTransferService: StateTransferService,
		fileService:          fileService,
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
//	@Description	Move a product to a target state (DRAFT, SUBMITTED, REVISION, ACTIVED, INACTIVED)
//	@Tags			Products.State
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
	if err = c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("invalid request body: "+err.Error(), http.StatusBadRequest))
		return
	}

	if err = h.Struct(&req); err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("validation failed: "+err.Error(), http.StatusBadRequest))
		return
	}

	target := enum.ProductStatus(req.State)
	if !target.IsValid() {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("invalid target state", http.StatusBadRequest))
		return
	}

	// Lấy role từ context
	roleVal, ok := c.Get("roles")
	if !ok || roleVal == nil {
		c.JSON(http.StatusForbidden, responses.ErrorResponse("missing role in context", http.StatusForbidden))
		return
	}
	roleStr, _ := roleVal.(string)
	if err := roleChecker("PRODUCT", roleStr, target); err != nil {
		resp := responses.ErrorResponse(err.Error(), http.StatusForbidden)
		c.JSON(http.StatusForbidden, resp)
		return
	}
	// Quy tắc role
	switch roleStr {
	case string(enum.UserRoleAdmin):
		// ADMIN có thể làm tất cả, không cần kiểm tra
	case string(enum.UserRoleBrandPartner):
		if target != enum.ProductStatusRevision && target != enum.ProductStatusActived {
			c.JSON(http.StatusForbidden, responses.ErrorResponse(
				fmt.Sprintf("BRAND_PARTNER cannot move product to state %s", target.String()),
				http.StatusForbidden,
			))
			return
		}
	case string(enum.UserRoleSalesStaff):
		if target == enum.ProductStatusRevision || target == enum.ProductStatusActived {
			c.JSON(http.StatusForbidden, responses.ErrorResponse(
				fmt.Sprintf("STAFF cannot move product to state %s", target.String()),
				http.StatusForbidden,
			))
			return
		}
	default:
		c.JSON(http.StatusForbidden, responses.ErrorResponse("unknown role", http.StatusForbidden))
		return
	}

	userID, err := extractUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, responses.ErrorResponse("invalid user_id in context: "+err.Error(), http.StatusUnauthorized))
		return
	}

	if err := h.MoveProductToState(c.Request.Context(), id, target, userID); err != nil {
		c.JSON(http.StatusConflict, responses.ErrorResponse("failed to move product: "+err.Error(), http.StatusConflict))
		return
	}

	c.JSON(http.StatusOK, responses.SuccessResponse("Product state updated", nil, map[string]any{
		"id":    id.String(),
		"state": target,
	}))
}

// UpdatePreOrderState
//
//	@Tags		Preorders
//	@Accept		multipart/form-data
//	@Produce	json
//	@Param		id		path		string	true	"Pre-Order ID (UUID)"
//	@Param		state	formData	string	true	"Target state: 'PENDING', 'PAID', 'PRE_ORDERED', 'STOCK_READY', 'STOCK_PREPARING', 'AWAITING_PICKUP', 'IN_TRANSIT', 'DELIVERED', 'RECEIVED'"
//	@Param		files	formData	file	false	"Proof images (multiple)"
//	@Security	BearerAuth
//	@Router		/api/v1/preorders/{id}/state [patch]
func (h *StateHandler) UpdatePreOrderState(c *gin.Context) {
	// 1.Parse path param
	idParam := c.Param("id")
	preOrderID, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("invalid pre-order id", http.StatusBadRequest))
		return
	}

	// 2.Parse multipart form
	form, err := c.MultipartForm()
	if err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("failed to parse multipart form: "+err.Error(), http.StatusBadRequest))
		return
	}

	// 3. Parse state
	stateValues := form.Value["state"]
	if len(stateValues) == 0 {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("missing state", http.StatusBadRequest))
		return
	}
	targetState := enum.PreOrderStatus(stateValues[0])
	if !targetState.IsValid() {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("invalid state: "+stateValues[0], http.StatusBadRequest))
		return
	}

	// Validate enum
	if !targetState.IsValid() {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("invalid target state", http.StatusBadRequest))
		return
	}

	// 4.Validate user role
	roleVal, ok := c.Get("roles")
	if !ok {
		c.JSON(http.StatusForbidden, responses.ErrorResponse("missing role in context", http.StatusForbidden))
		return
	}
	roleStr := roleVal.(string)

	if err := roleChecker("PREORDER", roleStr, targetState); err != nil {
		c.JSON(http.StatusForbidden, responses.ErrorResponse(err.Error(), http.StatusForbidden))
		return
	}

	// 5.Parse userID in context
	userID, err := extractUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("invalid user_id in context: "+err.Error(), http.StatusBadRequest))
		return
	}

	// 6. Process files (optional)
	files := form.File["files"]
	fileURLs := make([]string, 0, len(files))

	isImageOptional := true
	isStatusDelivered := targetState == enum.PreOrderStatusDelivered
	isStatusReceived := targetState == enum.PreOrderStatusReceived

	if isStatusDelivered || isStatusReceived {
		uow := h.UnitOfWork.Begin(c.Request.Context())
		preOrder, err := uow.PreOrder().GetByID(c.Request.Context(), preOrderID, nil)
		if err != nil {
			c.JSON(http.StatusBadRequest, responses.ErrorResponse("failed to fetch pre-order: "+err.Error(), http.StatusBadRequest))
			return
		}
		if (!preOrder.IsSelfPickedUp && isStatusDelivered) || (preOrder.IsSelfPickedUp && isStatusReceived) {
			isImageOptional = false
		}
	}

	// Validate at least one file if required
	if !isImageOptional && len(files) == 0 {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("at least one proof image is required for this state transition", http.StatusBadRequest))
		return
	}

	// Process uploaded files if any
	if !isImageOptional && len(files) > 0 {
		tmpDir := "/tmp/preorder_uploads"
		if err := os.MkdirAll(tmpDir, os.ModePerm); err != nil {
			c.JSON(http.StatusInternalServerError, responses.ErrorResponse("failed to create tmp dir", http.StatusInternalServerError))
			return
		}

		for _, fileHeader := range files {
			timestamp := time.Now().Format("20060102_150405")
			newFileName := fmt.Sprintf("%s_%s", timestamp, fileHeader.Filename)
			localPath := filepath.Join(tmpDir, newFileName)

			if err := c.SaveUploadedFile(fileHeader, localPath); err != nil {
				c.JSON(http.StatusInternalServerError, responses.ErrorResponse("failed to save uploaded file: "+fileHeader.Filename, http.StatusInternalServerError))
				return
			}

			userID := c.GetString("userID")
			url, err := h.fileService.UploadFile(c.Request.Context(), userID, localPath, newFileName)
			if err != nil {
				_ = os.Remove(localPath)
				c.JSON(http.StatusInternalServerError, responses.ErrorResponse("failed to upload file: "+fileHeader.Filename, http.StatusInternalServerError))
				return
			}

			fileURLs = append(fileURLs, url)
			_ = os.Remove(localPath)
		}
	}

	// 7. Move state safely
	var fileURLPtr *string
	if len(fileURLs) > 0 {
		fileURLPtr = ptr.String(fileURLs[0]) // only first file is passed
	}

	ctx := c.Request.Context()
	if err := h.MovePreOrderToState(ctx, preOrderID, targetState, userID, fileURLPtr); err != nil {
		c.JSON(http.StatusConflict, responses.ErrorResponse("failed to move pre-order: "+err.Error(), http.StatusConflict))
		return
	}

	// 8. Response
	c.JSON(http.StatusOK, responses.SuccessResponse("Pre-order state updated", nil, map[string]any{
		"id":    preOrderID,
		"state": targetState,
		"files": fileURLs,
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

func roleChecker(t, roleStr string, target any) error {
	switch t {
	case "PREORDER":
		zap.L().Info("This is an preorder")
		parsedTarget := target.(enum.PreOrderStatus)
		forbiddenStates := make(map[enum.PreOrderStatus]bool)

		switch roleStr {
		case string(enum.UserRoleAdmin):
			// ADMIN có thể làm tất cả, không cần kiểm tra
		case string(enum.UserRoleBrandPartner):
			forbiddenStates = map[enum.PreOrderStatus]bool{
				enum.PreOrderStatusPaid:           true,
				enum.PreOrderStatusPreOrdered:     true,
				enum.PreOrderStatusStockReady:     true,
				enum.PreOrderStatusStockPreparing: true,
				enum.PreOrderStatusAwaitingPickup: true,
				enum.PreOrderStatusInTransit:      true,
				enum.PreOrderStatusDelivered:      true,
			}
		case string(enum.UserRoleSalesStaff):
			forbiddenStates = map[enum.PreOrderStatus]bool{
				enum.PreOrderStatusPaid: true,
			}
		default:
			return fmt.Errorf("unknown role: %s", roleStr)
		}

		if forbiddenStates[parsedTarget] {
			return fmt.Errorf("%s cannot move preorder to state %s", enum.UserRoleBrandPartner.String(), parsedTarget.String())
		}

	case "PRODUCT":
		parsedTarget := target.(enum.ProductStatus)
		forbiddenStates := make(map[enum.ProductStatus]bool)

		switch roleStr {
		case string(enum.UserRoleAdmin):
			// ADMIN có thể làm tất cả, không cần kiểm tra
		case string(enum.UserRoleBrandPartner):
			forbiddenStates = map[enum.ProductStatus]bool{
				enum.ProductStatusDraft:     true,
				enum.ProductStatusSubmitted: true,
			}
		case string(enum.UserRoleSalesStaff):
			forbiddenStates = map[enum.ProductStatus]bool{
				enum.ProductStatusRevision: true,
				enum.ProductStatusActived:  true,
			}
		default:
			return fmt.Errorf("unknown role: %s", roleStr)
		}

		if forbiddenStates[parsedTarget] {
			return fmt.Errorf("%s cannot move product to state %s", enum.UserRoleBrandPartner.String(), parsedTarget.String())
		}

	default:
		zap.L().Info("Type not found for: " + t)
		return fmt.Errorf("Type not found for role Checker: %s ", t)
	}
	return nil
}
