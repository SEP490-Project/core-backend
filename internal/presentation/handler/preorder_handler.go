package handler

import (
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/application/interfaces/iservice"
	"core-backend/internal/domain/enum"
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

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
//	@Description	Returns user's preorders with pagination, optional status filter, search by product name or receiver name, and date range
//	@Tags			Preorders
//	@Accept			json
//	@Produce		json
//	@Param			page		query		int						false	"Page number (default: 1)"
//	@Param			limit		query		int						false	"Items per page (default: 10, max: 100)"
//	@Param			search		query		string					false	"Search by product name or receiver full name"
//	@Param			status		query		[]enum.PreOrderStatus	false	"example:"PAID"`
//	@Param			createdFrom	query		string					false	"Filter by start date (YYYY-MM-DD)"
//	@Param			createdTo	query		string					false	"Filter by end date (YYYY-MM-DD)"
//	@Success		200			{object}	responses.APIResponse{data=[]responses.PreOrderResponse,pagination=responses.Pagination}
//	@Failure		401			{object}	responses.APIResponse
//	@Failure		500			{object}	responses.APIResponse
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
	createdFrom := c.DefaultQuery("createdFrom", "")
	createdTo := c.DefaultQuery("createdTo", "")

	// Parse comma-separated statuses into slice
	var statuses []string
	if strings.TrimSpace(statusParam) != "" {
		parts := strings.SplitSeq(statusParam, ",")
		for s := range parts {
			s = strings.TrimSpace(s)
			if s != "" {
				statuses = append(statuses, s)
			}
		}
	}

	// Call the service with the new parameters
	ctx := c.Request.Context()
	items, total, err := p.preOrderService.GetPreOrdersByUserIDWithPagination(ctx, userID, limit, page, search, statuses, createdFrom, createdTo)
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
//	@Summary		1. Create a PreOrder (reserve stock)
//	@Description	Reserve a product variant as a preorder. This will decrement variant stock and create a preorder record.
//	@Tags			Preorders.States
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
	// paymentTx, err = p.preOrderService.PayForPreservationSlot(ctx, preorder.ID, req.SuccessURL, req.CancelURL, uow)
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
//	@Description	Retrieve paginated preorders for staff, filterable by status and search (id) and address fields.
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
		parts := strings.SplitSeq(s.String(), ",")
		for p := range parts {
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

// PreOrderApprove godoc
//
//	@Summary		Staff Censor an PREORDER (move to PRE_ORDERED or REFUNDED)
//	@Description	Change preOrder state to PRE_ORDERED or REFUNDED. Use query param `action=PRE_ORDERED` or `action=REFUNDED`. If cancelling, provide optional `reason` query param.
//	@Tags			Preorders.States
//	@Accept			json
//	@Produce		json
//	@Param			preOrderID	path		string				true	"Order ID"
//	@Param			action		query		string				true	"Action (CONFIRM|CANCEL)"
//	@Param			reason		body		CensorOrderRequest	false	"Cancel reason (required when action=CANCEL)"
//	@Success		200			{object}	responses.APIResponse{data=[]responses.OrderResponse,pagination=responses.Pagination}
//	@Failure		401			{object}	responses.APIResponse	"Unauthorized"
//	@Failure		500			{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/preorders/staff/{preOrderID}/approve [POST]
func (p *PreOrderHandler) PreOrderApprove(c *gin.Context) {
	preOrderIDStr := c.Param("preOrderID")
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
		if err = c.ShouldBindJSON(&req); err != nil {
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
//	@Summary		4 .1 Mark preorder as received (customer)
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
//	@Router			/api/v1/preorders/self-delivering/{id}/received [post]
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
	// --- Path ID ---
	preOrderID, err := parseUUIDParam(c, "id")
	if err != nil {
		return
	}

	// --- User ---
	updatedBy, err := extractUserID(c)
	if err != nil {
		msg := "unauthorized: " + err.Error()
		c.JSON(http.StatusUnauthorized, responses.ErrorResponse(msg, http.StatusUnauthorized))
		return
	}

	// --- Parse multipart form ---
	if err = parseMultipart(c, 32<<20); err != nil {
		return
	}

	// --- Extract required fields ---
	reasonPtr, err := extractRequiredFormField(c, "reason")
	if err != nil {
		return
	}

	fileHeader, err := extractRequiredFile(c, "file")
	if err != nil {
		return
	}

	fileURLPtr, err := p.handleFileUpload(c, updatedBy, fileHeader)
	if err != nil {
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse(err.Error(), http.StatusInternalServerError))
		return
	}

	// --- Begin transaction ---
	ctx := c.Request.Context()
	uow := p.unitOfWork.Begin(ctx)
	defer func() {
		if uow.InTransaction() {
			_ = uow.Rollback()
		}
	}()

	if err := p.preOrderService.RequestCompensation(ctx, preOrderID, updatedBy, reasonPtr, fileURLPtr); err != nil {
		zap.L().Error("failed to request preorder compensation", zap.Error(err))
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("failed to request compensation: "+err.Error(), http.StatusBadRequest))
		_ = uow.Rollback()
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
//	@Param			preOrderID	path		string	true	"PreOrder ID (UUID)"
//	@Param			isApproved	formData	string	true	"true|false"
//	@Param			reason		formData	string	false	"Reason (optional)"
//	@Param			file		formData	file	false	"Evidence file"
//	@Success		200			{object}	responses.APIResponse
//	@Failure		400			{object}	responses.APIResponse
//	@Failure		401			{object}	responses.APIResponse
//	@Failure		500			{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/preorders/staff/{preOrderID}/compensation [post]
func (p *PreOrderHandler) ProcessCompensation(c *gin.Context) {

	// --- Path param ---
	preOrderID, err := parseUUIDParam(c, "preOrderID")
	if err != nil {
		return
	}

	// --- User info ---
	updatedBy, err := extractUserID(c)
	if err != nil {
		msg := "unauthorized: " + err.Error()
		c.JSON(http.StatusUnauthorized, responses.ErrorResponse(msg, http.StatusUnauthorized))
		return
	}

	// --- Parse multipart ---
	if err = parseMultipart(c, 32<<20); err != nil {
		return
	}

	// --- Extract isApproved ---
	isApprovedStr := strings.TrimSpace(c.PostForm("isApproved"))
	if isApprovedStr == "" {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("isApproved is required", http.StatusBadRequest))
		return
	}
	isApproved := strings.EqualFold(isApprovedStr, "true") || isApprovedStr == "1"

	// --- Extract reason (optional unless rejecting) ---
	reason := strings.TrimSpace(c.PostForm("reason"))
	if !isApproved && reason == "" {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("reason cannot be empty when rejecting", http.StatusBadRequest))
		return
	}

	// --- File handling ---
	var fileURLPtr *string

	fileHeader, _ := c.FormFile("file")
	if isApproved && fileHeader == nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("file is required when approving", http.StatusBadRequest))
		return
	}

	if fileHeader != nil {
		fileURLPtr, err = p.handleFileUpload(c, updatedBy, fileHeader)
		if err != nil {
			c.JSON(http.StatusInternalServerError, responses.ErrorResponse(err.Error(), http.StatusInternalServerError))
			return
		}
	}

	// Convert nil to empty string for service call consistency
	var fileURL string
	if fileURLPtr != nil {
		fileURL = *fileURLPtr
	}

	// --- Perform service call ---
	ctx := c.Request.Context()
	err = p.preOrderService.ProcessCompensation(ctx, preOrderID, updatedBy, isApproved, &reason, &fileURL)
	if err != nil {
		zap.L().Error("failed to process preorder compensation", zap.Error(err))
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("failed to process compensation: "+err.Error(), http.StatusBadRequest))
		return
	}

	c.JSON(http.StatusOK,
		responses.SuccessResponse(
			"Compensation processed successfully",
			ptr.Int(http.StatusOK),
			map[string]any{"file_url": fileURL},
		),
	)
}

// MarkPreOrderAsReceivedByStaff godoc
//
//	@Summary		3.2-[Self-Pick-Up]-END Mark preorder as RECEIVED (Staff)
//	@Description	Mark self-pickup preorder as RECEIVED by staff
//	@Tags			Preorders.States
//	@Accept			multipart/form-data
//	@Produce		json
//	@Param			preOrderID	path		string	true	"PreOrder ID (UUID)"
//	@Param			file		formData	file	true	"Evidence file"
//	@Success		200			{object}	responses.APIResponse
//	@Failure		400			{object}	responses.APIResponse
//	@Failure		401			{object}	responses.APIResponse
//	@Failure		500			{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/preorders/staff/{preOrderID}/received [post]
func (p *PreOrderHandler) MarkPreOrderAsReceivedByStaff(c *gin.Context) {
	// --- Path ID ---
	preOrderID, err := parseUUIDParam(c, "preOrderID")
	if err != nil {
		return
	}

	// --- User ---
	updatedBy, err := extractUserID(c)
	if err != nil {
		msg := "unauthorized: " + err.Error()
		c.JSON(http.StatusUnauthorized, responses.ErrorResponse(msg, http.StatusUnauthorized))
		return
	}

	// --- Parse multipart form ---
	if err = parseMultipart(c, 32<<20); err != nil {
		return
	}

	fileHeader, err := extractRequiredFile(c, "file")
	if err != nil {
		return
	}

	fileURLPtr, err := p.handleFileUpload(c, updatedBy, fileHeader)
	if err != nil {
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse(err.Error(), http.StatusInternalServerError))
		return
	}

	// --- Begin transaction ---
	ctx := c.Request.Context()
	uow := p.unitOfWork.Begin(ctx)
	defer func() {
		if uow.InTransaction() {
			_ = uow.Rollback()
		}
	}()

	// Perform state transfer using stateTransferService (non-transactional here)
	targetStatus := enum.PreOrderStatusReceived
	if err := p.stateTransferService.MovePreOrderToState(ctx, preOrderID, targetStatus, updatedBy, nil, fileURLPtr); err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("failed to update preorder: "+err.Error(), http.StatusBadRequest))
		_ = uow.Rollback()
		return
	}
	_ = uow.Commit()
	resp := responses.SuccessResponse("Mark PreOrder as \"Delivered\" successfully", ptr.Int(http.StatusOK), nil)
	c.JSON(http.StatusOK, resp)
}

// PreOrderObligateRefund godoc
//
//	@Summary		Staff actively cancel PREORDER (move to REFUNDED)
//	@Description	Change preOrder state to REFUNDED. Requires reason and evidence file.
//	@Tags			Preorders.States
//	@Accept			multipart/form-data
//	@Produce		json
//	@Param			preOrderID	path		string	true	"Order ID"
//	@Param			reason		formData	string	true	"Cancel reason"
//	@Param			file		formData	file	true	"Evidence file"
//	@Success		200			{object}	responses.APIResponse
//	@Failure		400			{object}	responses.APIResponse
//	@Failure		401			{object}	responses.APIResponse
//	@Failure		500			{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/preorders/staff/{preOrderID}/obligate-refund [post]
func (p *PreOrderHandler) PreOrderObligateRefund(c *gin.Context) {
	// --- Path ID ---
	preOrderID, err := parseUUIDParam(c, "preOrderID")
	if err != nil {
		return
	}

	// --- User ---
	updatedBy, err := extractUserID(c)
	if err != nil {
		msg := "unauthorized: " + err.Error()
		c.JSON(http.StatusUnauthorized, responses.ErrorResponse(msg, http.StatusUnauthorized))
		return
	}

	// --- Parse multipart form ---
	if err = parseMultipart(c, 32<<20); err != nil {
		return
	}

	// --- Extract required fields ---
	reasonPtr, err := extractRequiredFormField(c, "reason")
	if err != nil {
		return
	}

	fileHeader, err := extractRequiredFile(c, "file")
	if err != nil {
		return
	}

	fileURLPtr, err := p.handleFileUpload(c, updatedBy, fileHeader)
	if err != nil {
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse(err.Error(), http.StatusInternalServerError))
		return
	}

	// --- Begin transaction ---
	ctx := c.Request.Context()
	uow := p.unitOfWork.Begin(ctx)
	defer func() {
		if uow.InTransaction() {
			_ = uow.Rollback()
		}
	}()

	// --- Move PreOrder to REFUNDED ---
	err = p.preOrderService.ObligateRefund(ctx, preOrderID, updatedBy, reasonPtr, fileURLPtr)
	if err != nil {
		_ = uow.Rollback()
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("failed to obligate refund: "+err.Error(), http.StatusBadRequest))
		return
	}

	// --- Commit transaction ---
	_ = uow.Commit()
	resp := responses.SuccessResponse("Preorder refunded successfully", ptr.Int(http.StatusOK), nil)
	c.JSON(http.StatusOK, resp)
}

// PreOrderRefundRequest godoc
//
//	@Summary	Customer Request a refund (Available before staff confirm)
//	@Tags		Preorders.States
//	@Accept		multipart/form-data
//	@Produce	json
//	@Param		preOrderID	path		string	true	"Order ID"
//	@Param		reason		formData	string	true	"Cancel reason"
//	@Success	200			{object}	responses.APIResponse{data=[]responses.OrderResponse,pagination=responses.Pagination}
//	@Failure	401			{object}	responses.APIResponse	"Unauthorized"
//	@Failure	500			{object}	responses.APIResponse
//	@Security	BearerAuth
//	@Router		/api/v1/preorders/refund/{preOrderID} [POST]
func (p *PreOrderHandler) PreOrderRefundRequest(c *gin.Context) {
	// --- Path ID ---
	preOrderID, err := parseUUIDParam(c, "preOrderID")
	if err != nil {
		return
	}

	// --- User ---
	updatedBy, err := extractUserID(c)
	if err != nil {
		msg := "unauthorized: " + err.Error()
		c.JSON(http.StatusUnauthorized, responses.ErrorResponse(msg, http.StatusUnauthorized))
		return
	}

	// --- Parse multipart form ---
	if err = parseMultipart(c, 32<<20); err != nil {
		return
	}

	// --- Extract required fields ---
	reasonPtr, err := extractRequiredFormField(c, "reason")
	if err != nil {
		return
	}

	// --- Begin transaction ---
	ctx := c.Request.Context()
	uow := p.unitOfWork.Begin(ctx)
	defer func() {
		if uow.InTransaction() {
			_ = uow.Rollback()
		}
	}()

	// --- Move PreOrder to REFUNDED ---
	err = p.preOrderService.RefundRequest(ctx, preOrderID, updatedBy, reasonPtr)
	if err != nil {
		_ = uow.Rollback()
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("failed to refund request: "+err.Error(), http.StatusBadRequest))
		return
	}

	// --- Commit transaction ---
	_ = uow.Commit()
	resp := responses.SuccessResponse("Preorder refund requested successfully", ptr.Int(http.StatusOK), nil)
	c.JSON(http.StatusOK, resp)

}

// ApprovePreOrderRefund godoc
//
//	@Summary	3 Staff approve PREORDER refund request (REFUNDED_REQUEST -> REFUNDED)
//	@Tags		Preorders.States
//	@Accept		multipart/form-data
//	@Produce	json
//	@Param		preOrderID	path		string	true	"Order ID"
//	@Param		reason		formData	string	false	"Additional reason"
//	@Param		file		formData	file	true	"Evidence file"
//	@Success	200			{object}	responses.APIResponse{data=[]responses.OrderResponse,pagination=responses.Pagination}
//	@Failure	401			{object}	responses.APIResponse	"Unauthorized"
//	@Failure	500			{object}	responses.APIResponse
//	@Security	BearerAuth
//	@Router		/api/v1/preorders/staff/refund/{preOrderID}/approve [POST]
func (p *PreOrderHandler) ApprovePreOrderRefund(c *gin.Context) {
	// --- Path ID ---
	preOrderID, err := parseUUIDParam(c, "preOrderID")
	if err != nil {
		return
	}

	// --- User ---
	updatedBy, err := extractUserID(c)
	if err != nil {
		msg := "unauthorized: " + err.Error()
		c.JSON(http.StatusUnauthorized, responses.ErrorResponse(msg, http.StatusUnauthorized))
		return
	}

	// --- Parse multipart form ---
	if err = parseMultipart(c, 32<<20); err != nil {
		return
	}

	// --- Extract required fields ---
	var reasonPtr *string
	value := c.PostForm("reason")
	if value == "" {
		reasonPtr = &value
	} else {
		reasonPtr = nil
	}

	fileHeader, err := extractRequiredFile(c, "file")
	if err != nil {
		return
	}

	fileURLPtr, err := p.handleFileUpload(c, updatedBy, fileHeader)
	if err != nil {
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse(err.Error(), http.StatusInternalServerError))
		return
	}

	// --- Begin transaction ---
	ctx := c.Request.Context()
	uow := p.unitOfWork.Begin(ctx)
	defer func() {
		if uow.InTransaction() {
			_ = uow.Rollback()
		}
	}()

	err = p.preOrderService.ApproveRefundRequest(ctx, preOrderID, updatedBy, reasonPtr, fileURLPtr)
	if err != nil {
		_ = uow.Rollback()
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("failed to refund request: "+err.Error(), http.StatusBadRequest))
		return
	}

	// --- Commit transaction ---
	_ = uow.Commit()
	resp := responses.SuccessResponse("Preorder refund requested successfully", ptr.Int(http.StatusOK), nil)
	c.JSON(http.StatusOK, resp)
}

// MarkPreOrderAsDelivered godoc
//
//	@Summary		3.1-[Self-Delivery] Mark preorder as delivered (Staff)
//	@Description	Mark self-delivering preorder as delivered by staff
//	@Tags			Preorders.States
//	@Accept			multipart/form-data
//	@Produce		json
//	@Param			preOrderID	path		string	true	"PreOrder ID (UUID)"
//	@Param			file		formData	file	false	"Evidence file"
//	@Success		200			{object}	responses.APIResponse
//	@Failure		400			{object}	responses.APIResponse
//	@Failure		401			{object}	responses.APIResponse
//	@Failure		500			{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/preorders/staff/self-delivering/{preOrderID}/delivered [post]
func (p *PreOrderHandler) MarkPreOrderAsDelivered(c *gin.Context) {
	// --- Path ID ---
	preOrderID, err := parseUUIDParam(c, "preOrderID")
	if err != nil {
		return
	}

	// --- User ---
	updatedBy, err := extractUserID(c)
	if err != nil {
		msg := "unauthorized: " + err.Error()
		c.JSON(http.StatusUnauthorized, responses.ErrorResponse(msg, http.StatusUnauthorized))
		return
	}

	// --- Parse multipart form ---
	if err = parseMultipart(c, 32<<20); err != nil {
		return
	}

	// --- Extract required fields ---
	fileHeader, err := extractRequiredFile(c, "file")
	if err != nil {
		return
	}

	fileURLPtr, err := p.handleFileUpload(c, updatedBy, fileHeader)
	if err != nil {
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse(err.Error(), http.StatusInternalServerError))
		return
	}

	// --- Begin transaction ---
	ctx := c.Request.Context()
	uow := p.unitOfWork.Begin(ctx)
	defer func() {
		if uow.InTransaction() {
			_ = uow.Rollback()
		}
	}()

	// Perform state transfer using stateTransferService (non-transactional here)
	targetStatus := enum.PreOrderStatusDelivered
	if err := p.stateTransferService.MovePreOrderToState(ctx, preOrderID, targetStatus, updatedBy, nil, fileURLPtr); err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("failed to update preorder: "+err.Error(), http.StatusBadRequest))
		_ = uow.Rollback()
		return
	}
	_ = uow.Commit()
	resp := responses.SuccessResponse("Mark PreOrder as \"Delivered\" successfully", ptr.Int(http.StatusOK), nil)
	c.JSON(http.StatusOK, resp)
}

// OpeningPreOrderEarly godoc
//
//	@Summary		Open PreOrder Early (Staff)
//	@Description	Allows staff to open pre-ordering for a product ahead of schedule.
//	@Tags			Preorders.States
//	@Accept			json
//	@Produce		json
//	@Param			productID	path		string	true	"Product ID (UUID)"
//	@Success		200			{object}	responses.APIResponse
//	@Failure		400			{object}	responses.APIResponse
//	@Failure		401			{object}	responses.APIResponse
//	@Failure		500			{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/preorders/staff/products/{productID}/open-early [patch]
func (p *PreOrderHandler) OpeningPreOrderEarly(c *gin.Context) {
	productID, err := parseUUIDParam(c, "productID")
	if err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("invalid product ID: "+err.Error(), http.StatusBadRequest))
		return
	}

	userID, err := extractUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, responses.ErrorResponse("unauthorized: "+err.Error(), http.StatusUnauthorized))
		return
	}

	if err = p.preOrderService.OpeningPreOrderEarly(c.Request.Context(), p.unitOfWork, productID, userID); err != nil {
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse("failed to open pre-order early: "+err.Error(), http.StatusInternalServerError))
		return
	}

	c.JSON(http.StatusOK, responses.SuccessResponse("Pre-order opened early successfully", ptr.Int(http.StatusOK), nil))
}

func NewPreOrderHandler(preOrderService iservice.PreOrderService, uow irepository.UnitOfWork, stateSvc iservice.StateTransferService, fileSvc iservice.FileService) *PreOrderHandler {
	return &PreOrderHandler{
		preOrderService:      preOrderService,
		unitOfWork:           uow,
		stateTransferService: stateSvc,
		fileService:          fileSvc,
	}
}

// handleFileUpload lưu file tạm, upload lên storage và trả về URL
func (p *PreOrderHandler) handleFileUpload(c *gin.Context, userID uuid.UUID, fileHeader *multipart.FileHeader) (*string, error) {
	// --- Tmp path ---
	tmpDir := os.TempDir()
	newFileName := uuid.New().String() + filepath.Ext(fileHeader.Filename)
	localPath := filepath.Join(tmpDir, newFileName)

	// --- Save file tạm ---
	if err := c.SaveUploadedFile(fileHeader, localPath); err != nil {
		return nil, fmt.Errorf("failed to save uploaded file: %w", err)
	}
	defer os.Remove(localPath)

	// --- Upload file lên S3 / storage ---
	fileURL, err := p.fileService.UploadFile(c.Request.Context(), userID.String(), localPath, newFileName)
	if err != nil {
		return nil, fmt.Errorf("failed to upload file: %w", err)
	}

	if fileURL == "" {
		return nil, errors.New("uploaded file URL is empty")
	}

	return &fileURL, nil
}
