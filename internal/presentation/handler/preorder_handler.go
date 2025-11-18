package handler

import (
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/application/interfaces/iservice"
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
	"net/http"
	"strconv"
	"strings"

	"github.com/aws/smithy-go/ptr"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type PreOrderHandler struct {
	preOrderService      iservice.PreOrderService
	unitOfWork           irepository.UnitOfWork
	stateTransferService iservice.StateTransferService
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
//	@Success		200		{object}	responses.APIResponse{data=[]model.PreOrder,pagination=responses.Pagination}
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

	ctx := c.Request.Context()
	uow := p.unitOfWork.Begin(ctx)
	defer func() {
		if uow.InTransaction() {
			_ = uow.Rollback()
		}
	}()

	//1. Preserve order
	preorder, err := p.preOrderService.PreserverOrder(ctx, req, uow)
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
//	@Param			query	query		requests.StaffOrdersQuery	false	"Staff preorders query"
//	@Success		200		{object}	responses.APIResponse{data=[]model.PreOrder,pagination=responses.Pagination}
//	@Failure		401		{object}	responses.APIResponse	"Unauthorized"
//	@Failure		500		{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/preorders/staff [get]
func (p *PreOrderHandler) GetStaffAvailablePreOrdersWithPagination(c *gin.Context) {
	var q requests.StaffOrdersQuery
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
	if strings.TrimSpace(string(status)) != "" {
		statuses = append(statuses, string(status))
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
//	@Success		200		{object}	responses.APIResponse{data=[]model.Order,pagination=responses.Pagination}
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
	// uow := p.unitOfWork.Begin(ctx)
	// defer func() {
	// 	if uow.InTransaction() {
	// 		_ = uow.Rollback()
	// 	}
	// }()

	// Perform state transfer using stateTransferService
	preOrderID, err := uuid.Parse(preOrderIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("invalid order_id: "+err.Error(), http.StatusBadRequest))
		return
	}

	// Perform state transfer using stateTransferService (non-transactional here)
	if err := p.stateTransferService.MovePreOrderToState(ctx, preOrderID, targetStatus, updatedBy, nil); err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("failed to update preorder: "+err.Error(), http.StatusBadRequest))
		return
	}

	// Begin a UnitOfWork transaction only for appending the action note
	uow := p.unitOfWork.Begin(ctx)
	defer func() {
		if uow.InTransaction() {
			_ = uow.Rollback()
		}
	}()

	// Append action note
	preOrder, err := uow.PreOrder().GetByID(ctx, preOrderID, nil)
	if err == nil && preOrder != nil {
		note := model.PreOrderActionNote{
			UserID:     updatedBy,
			UserName:   "", // optional: populate if available in context
			UserEmail:  "",
			ActionType: targetStatus,
			Reason:     "",
		}
		if reasonPtr != nil {
			note.Reason = *reasonPtr
		}
		preOrder.AddActionNote(note)
		_ = uow.PreOrder().Update(ctx, preOrder)
	}

	_ = uow.Commit()

	resp := responses.SuccessResponse("Preorder censored successfully", ptr.Int(http.StatusOK), nil)
	c.JSON(http.StatusOK, resp)
}

func NewPreOrderHandler(preOrderService iservice.PreOrderService, uow irepository.UnitOfWork, stateSvc iservice.StateTransferService) *PreOrderHandler {
	return &PreOrderHandler{
		preOrderService:      preOrderService,
		unitOfWork:           uow,
		stateTransferService: stateSvc,
	}
}
