package handler

import (
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/application/interfaces/iservice"
	"core-backend/internal/domain/enum"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/aws/smithy-go/ptr"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type PreOrderHandler struct {
	preOrderService      iservice.PreOrderService
	unitOfWork           irepository.UnitOfWork
	stateTransferService iservice.StateTransferService
	fileService          iservice.FileService
}

// GetAllPreorders godoc
//
//	@Summary		Get paginated preorders by current user
//	@Description	Returns user's preorders with pagination, optional status filter and search by product name or receiver name
//	@Tags			Preorders
//	@Accept			json
//	@Produce		json
//	@Param			page	query		int		false	"Page number (default: 1)"
//	@Param			limit	query		int		false	"Items per page (default: 10, max: 100)"
//	@Param			search	query		string	false	"Search by product name or receiver full name"
//	@Param			status	query		string	false	"Filter by status (PENDING, PAID, PRE_ORDERED, STOCK_READY, STOCK_PREPARING, AWAITING_PICKUP, CANCELLED, IN_TRANSIT, DELIVERED, RECEIVED)"
//	@Success		200		{object}	responses.APIResponse{data=[]responses.PreOrderResponse,pagination=responses.Pagination}
//	@Failure		401		{object}	responses.APIResponse
//	@Failure		500		{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/preorders [get]
func (p *PreOrderHandler) GetAllPreorders(c *gin.Context) {
	pageStr := c.DefaultQuery("page", "1")
	limitStr := c.DefaultQuery("limit", "10")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	// Extract user
	userID, err := extractUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, responses.ErrorResponse("unauthorized: "+err.Error(), http.StatusUnauthorized))
		return
	}

	search := c.DefaultQuery("search", "")
	statusParam := c.DefaultQuery("status", "")

	// parse comma-separated statuses into slice
	var statuses []string
	if strings.TrimSpace(statusParam) != "" {
		parts := strings.Split(statusParam, ",")
		for _, s := range parts {
			s = strings.TrimSpace(s)
			if s != "" {
				statuses = append(statuses, s)
			}
		}
	}

	items, total, err := p.preOrderService.GetPreOrdersByUserIDWithPagination(userID, limit, page, search, statuses)
	if err != nil {
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse("failed to fetch preorders: "+err.Error(), http.StatusInternalServerError))
		return
	}

	totalPages := 0
	if limit > 0 {
		totalPages = int(total) / limit
		if total%limit != 0 {
			totalPages++
		}
	}

	pagination := responses.Pagination{
		Page:       page,
		Limit:      limit,
		Total:      int64(total),
		TotalPages: totalPages,
		HasNext:    page < totalPages,
		HasPrev:    page > 1,
	}

	resp := responses.NewPaginationResponse(
		"Preorders retrieved successfully",
		http.StatusOK,
		items,
		pagination,
	)

	c.JSON(http.StatusOK, resp)
}

// CreatePreOrderAndPay  godoc
//
//	@Summary		Create a PreOrder (reserve stock)
//	@Description	Reserve a product variant as a preorder. This will decrement variant stock and create a preorder record.
//	@Tags			Preorders
//	@Accept			json
//	@Produce		json
//	@Param			data	body		requests.PreOrderRequest	true	"PreOrder payload"
//	@Success		201		{object}	responses.APIResponse
//	@Failure		400		{object}	responses.APIResponse
//	@Failure		401		{object}	responses.APIResponse
//	@Failure		500		{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/preorders [post]
func (p *PreOrderHandler) CreatePreOrderAndPay(c *gin.Context) {
	var req requests.PreOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("invalid request body: "+err.Error(), http.StatusBadRequest))
		return
	}

	userID, err := extractUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, responses.ErrorResponse("unauthorized: "+err.Error(), http.StatusUnauthorized))
		return
	}

	ctx := c.Request.Context()
	uow := p.unitOfWork.Begin(ctx)
	defer func() {
		if uow.InTransaction() {
			_ = uow.Rollback()
		}
	}()

	//1. Preserve order
	preorder, err := p.preOrderService.PreserverOrder(ctx, req, uow, userID)
	if err != nil {
		_ = uow.Rollback()
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("failed to create preorder: "+err.Error(), http.StatusBadRequest))
		return
	}

	//2. Initiate payment
	var paymentTx *responses.PayOSLinkResponse
	paymentTx, err = p.preOrderService.PayForPreservationSlot(ctx, preorder.ID, req.SuccessURL, req.CancelURL, uow)
	if err != nil {
		_ = uow.Rollback()
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse("failed to initiate payment: "+err.Error(), http.StatusInternalServerError))
		return
	}

	//3. Commit transaction
	if err := uow.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse("failed to commit transaction: "+err.Error(), http.StatusInternalServerError))
		return
	}

	// 4) Return response
	resp := responses.SuccessResponse("Order placed and payment initiated", ptr.Int(http.StatusOK), map[string]any{
		"pre-order":  preorder,
		"payment_tx": paymentTx,
	})

	c.JSON(http.StatusCreated, resp)
}

// GetStaffAvailablePreOrdersWithPagination handles staff-facing GET requests to list preorders with same query fields as orders
//
//	@Summary		Get staff-available preorders with pagination
//	@Description	Retrieve paginated preorders for staff, filterable by status and search (id/payment id/bin) and address fields.
//	@Tags			Preorders
//	@Accept			json
//	@Produce		json
//	@Param			query	query		requests.StaffPreOrdersQuery	false	"Staff preorders query"
//	@Success		200		{object}	responses.APIResponse{data=[]responses.PreOrderResponse,pagination=responses.Pagination}
//	@Failure		401		{object}	responses.APIResponse	"Unauthorized"
//	@Failure		500		{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/preorders/staff [get]
func (p *PreOrderHandler) GetStaffAvailablePreOrdersWithPagination(c *gin.Context) {
	var q requests.StaffPreOrdersQuery
	_ = c.ShouldBindQuery(&q)

	page := q.Page
	limit := q.Limit
	if page < 1 {
		page = 1
	}
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	search := q.Search
	status := q.Status
	fullName := q.FullName
	phone := q.Phone
	provinceID := q.ProvinceID
	districtID := q.DistrictID
	wardCode := q.WardCode

	// normalize staff status to []string for service
	var statuses []string
	for _, s := range status {
		if s == "" {
			continue
		}
		// Tách comma-separated
		parts := strings.Split(s.String(), ",")
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p != "" {
				statuses = append(statuses, p)
			}
		}
	}

	preorders, total, err := p.preOrderService.GetStaffAvailablePreOrdersWithPagination(limit, page, search, fullName, phone, provinceID, districtID, wardCode, statuses)
	if err != nil {
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse("failed to fetch staff preorders: "+err.Error(), http.StatusInternalServerError))
		return
	}

	totalPages := 0
	if limit > 0 {
		totalPages = int(total) / limit
		if total%limit != 0 {
			totalPages++
		}
	}

	pagination := responses.Pagination{
		Page:       page,
		Limit:      limit,
		Total:      int64(total),
		TotalPages: totalPages,
		HasNext:    page < totalPages,
		HasPrev:    page > 1,
	}

	resp := responses.NewPaginationResponse(
		"Preorders retrieved successfully",
		http.StatusOK,
		preorders,
		pagination,
	)

	c.JSON(http.StatusOK, resp)
}

// PreOrderCensorship godoc
//
//	@Summary		Censor an PREORDER (confirm or cancel)
//	@Description	Change preOrder state to PRE_ORDERED or CANCELLED. Use query param `action=PRE_ORDERED` or `action=CANCELLED`. If cancelling, provide optional `reason` query param.
//	@Tags			Preorders.States
//	@Accept			json
//	@Produce		json
//	@Param			orderID	path		string				true	"Order ID"
//	@Param			action	query		string				true	"Action (CONFIRM|CANCEL)"
//	@Param			reason	body		CensorOrderRequest	false	"Cancel reason (required when action=CANCEL)"
//	@Success		200		{object}	responses.APIResponse{data=[]responses.OrderResponse,pagination=responses.Pagination}
//	@Failure		401		{object}	responses.APIResponse	"Unauthorized"
//	@Failure		500		{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/preorders/staff/{orderID}/censorship [POST]
func (p *PreOrderHandler) PreOrderCensorship(c *gin.Context) {
	preOrderIDStr := c.Param("orderID")
	if preOrderIDStr == "" {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("order_id is required", http.StatusBadRequest))
		return
	}

	action := strings.ToUpper(strings.TrimSpace(c.DefaultQuery("action", "")))
	if action == "" {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("action query param is required", http.StatusBadRequest))
		return
	}

	var targetStatus enum.PreOrderStatus
	switch action {
	case "CONFIRM":
		targetStatus = enum.PreOrderStatusPreOrdered
	case "CANCEL":
		targetStatus = enum.PreOrderStatusCancelled
	default:
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("invalid action, allowed: CONFIRM, CANCEL", http.StatusBadRequest))
		return
	}

	// Extract acting user
	updatedBy, err := extractUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, responses.ErrorResponse("unauthorized: "+err.Error(), http.StatusUnauthorized))
		return
	}

	// If cancelling, require a JSON body with reason
	var reasonPtr *string
	if targetStatus == enum.PreOrderStatusCancelled {
		var req CensorOrderRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, responses.ErrorResponse("reason is required when action=CANCEL: "+err.Error(), http.StatusBadRequest))
			return
		}
		trimmed := strings.TrimSpace(req.Reason)
		if trimmed == "" {
			c.JSON(http.StatusBadRequest, responses.ErrorResponse("reason cannot be empty", http.StatusBadRequest))
			return
		}
		reasonPtr = &trimmed
	}

	ctx := c.Request.Context()
	uow := p.unitOfWork.Begin(ctx)
	defer func() {
		if uow.InTransaction() {
			_ = uow.Rollback()
		}
	}()

	// Perform state transfer using stateTransferService
	preOrderID, err := uuid.Parse(preOrderIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("invalid order_id: "+err.Error(), http.StatusBadRequest))
		return
	}

	// Perform state transfer using stateTransferService (non-transactional here)
	if err := p.stateTransferService.MovePreOrderToState(ctx, preOrderID, targetStatus, updatedBy, reasonPtr, nil); err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("failed to update preorder: "+err.Error(), http.StatusBadRequest))
		_ = uow.Rollback()
		return
	}

	_ = uow.Commit()

	resp := responses.SuccessResponse("Preorder censored successfully", ptr.Int(http.StatusOK), nil)
	c.JSON(http.StatusOK, resp)
}

// MarkPreOrderAsReceived godoc
//
//	@Summary		Mark preorder as received (customer)
//	@Description	Mark a preorder as received by the customer
//	@Tags			Preorders.States
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string	true	"PreOrder ID (UUID)"
//	@Success		200	{object}	responses.APIResponse
//	@Failure		400	{object}	responses.APIResponse
//	@Failure		401	{object}	responses.APIResponse
//	@Failure		500	{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/preorders/{id}/received [patch]
func (p *PreOrderHandler) MarkPreOrderAsReceived(c *gin.Context) {
	idParam := c.Param("id")
	preOrderID, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("invalid preOrder ID", http.StatusBadRequest))
		return
	}

	updatedBy, err := extractUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, responses.ErrorResponse("unauthorized: "+err.Error(), http.StatusUnauthorized))
		return
	}

	ctx := c.Request.Context()
	if err := p.preOrderService.MarkPreOrderAsReceived(ctx, preOrderID, updatedBy); err != nil {
		zap.L().Error("failed to mark preorder as received", zap.Error(err))
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("failed to mark preorder as received: "+err.Error(), http.StatusBadRequest))
		return
	}

	c.JSON(http.StatusOK, responses.SuccessResponse("Preorder marked as received successfully", ptr.Int(http.StatusOK), nil))
}

// RequestCompensation godoc
//
//	@Summary		Request compensation for a preorder (customer)
//	@Description	Submit a compensation request with reason and supporting file
//	@Tags			Preorders.States
//	@Accept			multipart/form-data
//	@Produce		json
//	@Param			id		path		string	true	"PreOrder ID (UUID)"
//	@Param			reason	formData	string	true	"Reason for compensation"
//	@Param			file	formData	file	true	"Evidence file"
//	@Success		200		{object}	responses.APIResponse
//	@Failure		400		{object}	responses.APIResponse
//	@Failure		401		{object}	responses.APIResponse
//	@Failure		500		{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/preorders/{id}/compensation [post]
func (p *PreOrderHandler) RequestCompensation(c *gin.Context) {
	idParam := c.Param("id")
	preOrderID, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("invalid preOrder ID", http.StatusBadRequest))
		return
	}

	actionBy, err := extractUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, responses.ErrorResponse("unauthorized: "+err.Error(), http.StatusUnauthorized))
		return
	}

	reason := strings.TrimSpace(c.PostForm("reason"))
	if reason == "" {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("reason is required", http.StatusBadRequest))
		return
	}

	fileHeader, err := c.FormFile("file")
	if err != nil || fileHeader == nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("file is required", http.StatusBadRequest))
		return
	}

	userTmpDir := "/tmp/uploads"
	if err := os.MkdirAll(userTmpDir, os.ModePerm); err != nil {
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse("failed to create tmp upload directory", http.StatusInternalServerError))
		return
	}

	timestamp := time.Now().Format("20060102_150405")
	newFileName := fmt.Sprintf("%s_%s", timestamp, fileHeader.Filename)
	finalPath := fmt.Sprintf("%s/%s", userTmpDir, newFileName)

	if err := c.SaveUploadedFile(fileHeader, finalPath); err != nil {
		_ = os.Remove(finalPath)
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse("failed to save uploaded file: "+err.Error(), http.StatusInternalServerError))
		return
	}
	defer func(path string) { _ = os.Remove(path) }(finalPath)

	fileURL, err := p.fileService.UploadFile(c.Request.Context(), actionBy.String(), finalPath, newFileName)
	if err != nil {
		_ = os.Remove(finalPath)
		zap.L().Error("failed to upload compensation file", zap.Error(err))
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse("failed to upload file: "+err.Error(), http.StatusInternalServerError))
		return
	}

	ctx := c.Request.Context()
	if err := p.preOrderService.RequestCompensation(ctx, preOrderID, actionBy, &reason, &fileURL); err != nil {
		zap.L().Error("failed to request preorder compensation", zap.Error(err))
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("failed to request compensation: "+err.Error(), http.StatusBadRequest))
		return
	}

	c.JSON(http.StatusOK, responses.SuccessResponse("Compensation requested successfully", ptr.Int(http.StatusOK), nil))
}

// ProcessCompensation godoc
//
//	@Summary		Process compensation for a preorder (staff)
//	@Description	Approve or reject a compensation request
//	@Tags			Preorders[Staff].States
//	@Accept			multipart/form-data
//	@Produce		json
//	@Param			orderID		path		string	true	"PreOrder ID (UUID)"
//	@Param			isApproved	formData	string	true	"true|false"
//	@Param			reason		formData	string	false	"Reason (optional)"
//	@Param			file		formData	file	false	"Confirmation file"
//	@Success		200			{object}	responses.APIResponse
//	@Failure		400			{object}	responses.APIResponse
//	@Failure		401			{object}	responses.APIResponse
//	@Failure		500			{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/preorders/staff/{orderID}/compensation [post]
func (p *PreOrderHandler) ProcessCompensation(c *gin.Context) {
	idParam := c.Param("orderID")
	preOrderID, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("invalid preOrder ID", http.StatusBadRequest))
		return
	}

	updatedBy, err := extractUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, responses.ErrorResponse("unauthorized: "+err.Error(), http.StatusUnauthorized))
		return
	}

	isApprovedStr := strings.TrimSpace(c.PostForm("isApproved"))
	if isApprovedStr == "" {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("isApproved form field is required", http.StatusBadRequest))
		return
	}
	isApproved := strings.EqualFold(isApprovedStr, "true") || isApprovedStr == "1"
	reason := strings.TrimSpace(c.PostForm("reason"))

	if !isApproved && reason == "" {
		resp := responses.ErrorResponse("reason cannot be left empty when rejecting", http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, resp)
		return
	}

	var fileURL string
	fileHeader, err := c.FormFile("file")
	if (err != nil || fileHeader == nil) && isApproved {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("file is required when approving", http.StatusBadRequest))
		return
	} else if err == nil && fileHeader != nil {
		userTmpDir := "/tmp/uploads"
		if err := os.MkdirAll(userTmpDir, os.ModePerm); err != nil {
			c.JSON(http.StatusInternalServerError, responses.ErrorResponse("failed to create tmp upload directory", http.StatusInternalServerError))
			return
		}

		timestamp := time.Now().Format("20060102_150405")
		newFileName := fmt.Sprintf("%s_%s", timestamp, fileHeader.Filename)
		finalPath := fmt.Sprintf("%s/%s", userTmpDir, newFileName)

		if err := c.SaveUploadedFile(fileHeader, finalPath); err != nil {
			_ = os.Remove(finalPath)
			c.JSON(http.StatusInternalServerError, responses.ErrorResponse("failed to save uploaded file: "+err.Error(), http.StatusInternalServerError))
			return
		}
		defer func(path string) { _ = os.Remove(path) }(finalPath)

		fileURL, err = p.fileService.UploadFile(c.Request.Context(), updatedBy.String(), finalPath, newFileName)
		if err != nil {
			_ = os.Remove(finalPath)
			zap.L().Error("failed to upload confirmation file", zap.Error(err))
			c.JSON(http.StatusInternalServerError, responses.ErrorResponse("failed to upload file: "+err.Error(), http.StatusInternalServerError))
			return
		}
	}

	ctx := c.Request.Context()
	if err := p.preOrderService.ProcessCompensation(ctx, preOrderID, updatedBy, isApproved, &reason, &fileURL); err != nil {
		zap.L().Error("failed to process preorder compensation", zap.Error(err))
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("failed to process compensation: "+err.Error(), http.StatusBadRequest))
		return
	}

	c.JSON(http.StatusOK, responses.SuccessResponse("Compensation processed successfully", ptr.Int(http.StatusOK), map[string]any{"file_url": fileURL}))
}

func NewPreOrderHandler(preOrderService iservice.PreOrderService, uow irepository.UnitOfWork, stateSvc iservice.StateTransferService, fileSvc iservice.FileService) *PreOrderHandler {
	return &PreOrderHandler{
		preOrderService:      preOrderService,
		unitOfWork:           uow,
		stateTransferService: stateSvc,
		fileService:          fileSvc,
	}
}
