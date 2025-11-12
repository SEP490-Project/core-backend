package handler

import (
	"core-backend/internal/application/interfaces/iproxies"
	"net/http"
	"strconv"
	"strings"

	"go.uber.org/zap"

	"github.com/aws/smithy-go/ptr"

	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/application/interfaces/iservice"
	"core-backend/internal/domain/enum"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type OrderHandler struct {
	orderService         iservice.OrderService
	ghnProxy             iproxies.GHNProxy
	unitOfWork           irepository.UnitOfWork
	stateTransferService iservice.StateTransferService
}

func NewOrderHandler(orderSvc iservice.OrderService, ghnProxy iproxies.GHNProxy, uow irepository.UnitOfWork, stateTransferService iservice.StateTransferService) *OrderHandler {
	return &OrderHandler{
		orderService:         orderSvc,
		ghnProxy:             ghnProxy,
		unitOfWork:           uow,
		stateTransferService: stateTransferService,
	}
}

// GetOrdersByUserIDWithPagination handles HTTP GET requests to retrieve paginated orders for a specific user.
//
//	@Summary		Get paginated orders by user ID
//	@Description	This handler extracts pagination parameters (`page`, `limit`) and an optional search term from the query string.
//
//	It also extracts the user ID from the authentication token and fetches the paginated list of orders
//
//	@Tags			Orders
//	@Accept			json
//	@Produce		json
//	@Param		page	query		int		false	"Page number (default: 1)"
//	@Param		limit	query		int		false	"Number of items per page (default: 10, max: 100)"
//	@Param		search	query		string	false	"Search term for filtering orders by order number"
//	@Success		200		{object}	responses.APIResponse{data=[]model.Order,pagination=responses.Pagination}
//	@Failure		401		{object}	responses.APIResponse	"Unauthorized"
//	@Failure		500		{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/orders [get]
func (h *OrderHandler) GetOrdersByUserIDWithPagination(c *gin.Context) {
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

	orders, total, err := h.orderService.GetOrdersByUserIDWithPagination(userID, limit, page, search)
	if err != nil {
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse("failed to fetch orders: "+err.Error(), http.StatusInternalServerError))
		return
	}

	totalPages := int(total) / limit
	if total%limit != 0 {
		totalPages++
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
		"Products retrieved successfully",
		http.StatusOK,
		orders,
		pagination,
	)

	c.JSON(http.StatusOK, resp)
}

// PlaceAndPayOrder godoc
//
//	@Summary		Place an order and initiate payment
//	@Description	Create an order -> calculate delivery fee -> create payment transaction
//	@Tags			Orders
//	@Accept			json
//	@Produce		json
//	@Param		data	body		requests.PlaceAndPayRequest	true	"Place and pay payload"
//	@Success		200		{object}	map[string]any
//	@Failure		400		{object}	map[string]string
//	@Failure		401		{object}	map[string]string
//	@Failure		500		{object}	map[string]string
//	@Security		BearerAuth
//	@Router			/api/v1/orders [post]
func (h *OrderHandler) CreateOrder(c *gin.Context) {
	// Bind request
	var req requests.PlaceAndPayRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("invalid request body: "+err.Error(), http.StatusBadRequest))
		return
	}

	// Extract user
	userID, err := extractUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, responses.ErrorResponse("unauthorized: "+err.Error(), http.StatusUnauthorized))
		return
	}

	if len(req.Order.Items) == 0 {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("order must contain at least one item", http.StatusBadRequest))
		return
	}

	ctx := c.Request.Context()
	uow := h.unitOfWork.Begin(ctx)
	defer func() {
		// ensure transaction ended if still open
		if uow.InTransaction() {
			_ = uow.Rollback()
		}
	}()

	//*1 Calcucate delivery fee first as we need to validate order dimensions/weight before placing order
	deliveryFee, err := h.ghnProxy.CalculateDeliveryPriceByShippingAddressAndOrderItem(ctx, req.Order.AddressID, req.Order.Items, uow)
	if err != nil {
		zap.L().Error("failed to calculate delivery fee", zap.Error(err))
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("failed to calculate delivery fee: "+err.Error(), http.StatusBadRequest))
		return
	}

	//*2 Create order with payment
	order, err := h.orderService.PlaceOrder(ctx, userID, req.Order, deliveryFee.Total, uow)
	if err != nil {
		_ = uow.Rollback()
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("failed to place order: "+err.Error(), http.StatusBadRequest))
		return
	}

	//*3 Initiate payment
	paymentTx, err := h.orderService.PayOrder(ctx, order.ID, deliveryFee.Total, req.SuccessURL, req.CancelURL, uow)
	if err != nil {
		_ = uow.Rollback()
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("failed to initiate payment: "+err.Error(), http.StatusBadRequest))
		return
	}

	// 4) Commit transaction
	if err := uow.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse("failed to commit transaction: "+err.Error(), http.StatusInternalServerError))
		return
	}

	// 5) Return response
	resp := responses.SuccessResponse("Order placed and payment initiated", ptr.Int(http.StatusOK), map[string]any{
		"order":        order,
		"payment_tx":   paymentTx,
		"delivery_fee": deliveryFee,
	})
	c.JSON(http.StatusOK, resp)
}

// GetStaffAvailableOrdersWithPagination handles HTTP GET requests to retrieve paginated staff-available orders with optional status filter.
//
//	@Summary		Get staff-available orders with pagination
//	@Description	Retrieve paginated orders for staff, filterable by status and order number search.
//	@Tags			Orders
//	@Accept			json
//	@Produce		json
//	@Param		query	query		requests.StaffOrdersQuery	false	"Staff orders query"
//	@Success		200		{object}	responses.APIResponse{data=[]model.Order,pagination=responses.Pagination}
//	@Failure		401		{object}	responses.APIResponse	"Unauthorized"
//	@Failure		500		{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/orders/staff [get]
func (h *OrderHandler) GetStaffAvailableOrdersWithPagination(c *gin.Context) {
	// Bind query params into struct for swag and cleaner code
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

	orders, total, err := h.orderService.GetStaffAvailableOrdersWithPagination(limit, page, search, status.String(), fullName, phone, provinceID, districtID, wardCode)
	if err != nil {
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse("failed to fetch staff orders: "+err.Error(), http.StatusInternalServerError))
		return
	}

	totalPages := int(total) / limit
	if total%limit != 0 {
		totalPages++
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
		"Orders retrieved successfully",
		http.StatusOK,
		orders,
		pagination,
	)

	c.JSON(http.StatusOK, resp)
}

// CensorOrderRequest represents payload for censoring an order (reason required when cancelling)
// swagger:model CensorOrderRequest
type CensorOrderRequest struct {
	// Reason for cancelling the order. Required when action=CANCEL
	// in: body
	// example: "Customer requested cancellation due to wrong size"
	Reason string `json:"reason" binding:"required"`
}

// OrderCensorship:
// @Summary	Censor an order (confirm or cancel)
// @Description	Change order state to CONFIRMED or CANCELLED. Use query param `action=CONFIRM` or `action=CANCEL`. If cancelling, provide optional `reason` query param.
// @Tags		Orders
// @Accept		json
// @Produce		json
// @Param	orderID	path	string	true	"Order ID"
// @Param	action	query	string	true	"Action (CONFIRM|CANCEL)"
// @Param   reason body CensorOrderRequest false "Cancel reason (required when action=CANCEL)"
// @Success		200		{object}	responses.APIResponse{data=[]model.Order,pagination=responses.Pagination}
// @Failure		401		{object}	responses.APIResponse	"Unauthorized"
// @Failure		500		{object}	responses.APIResponse
// @Security	BearerAuth
//
//	@Router			/api/v1/orders/staff/{orderID}/censorship [POST]
func (h *OrderHandler) OrderCensorship(c *gin.Context) {

	ctx := c.Request.Context()

	orderIDStr := c.Param("orderID")
	if orderIDStr == "" {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("order_id is required", http.StatusBadRequest))
		return
	}

	orderID, err := uuid.Parse(orderIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("invalid order_id: "+err.Error(), http.StatusBadRequest))
		return
	}

	action := strings.ToUpper(strings.TrimSpace(c.DefaultQuery("action", "")))
	if action == "" {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("action query param is required", http.StatusBadRequest))
		return
	}

	var targetStatus enum.OrderStatus
	switch action {
	case "CONFIRM":
		targetStatus = enum.OrderStatusConfirmed
	case "CANCEL":
		targetStatus = enum.OrderStatusCancelled
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
	if targetStatus == enum.OrderStatusCancelled {
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

	uow := h.unitOfWork.Begin(ctx)
	defer func() {
		if uow.InTransaction() {
			_ = uow.Rollback()
		}
	}()

	// Perform state transfer
	if err := h.stateTransferService.MoveOrderToState(ctx, orderID, targetStatus, updatedBy, reasonPtr); err != nil {
		_ = uow.Rollback()
		zap.L().Error("failed to censor order", zap.Error(err))
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse("failed to update order: "+err.Error(), http.StatusInternalServerError))
		return
	}

	//

	if err := uow.Commit(); err != nil {
		zap.L().Error("failed to commit transaction for censor order", zap.Error(err))
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse("failed to commit transaction: "+err.Error(), http.StatusInternalServerError))
		return
	}

	c.JSON(http.StatusOK, responses.SuccessResponse("Order updated successfully", ptr.Int(http.StatusOK), nil))
}

// MarkAsReceived godoc
// @Summary Mark order as received
// @Description Đánh dấu đơn hàng là "đã nhận" (Received) sau khi giao thành công
// @Tags Orders
// @Accept json
// @Produce json
// @Param orderID path string true "Order ID (UUID)"
// @Success 200 {object} map[string]interface{} "Order marked as received successfully"
// @Failure 400 {object} map[string]string "Invalid order ID"
// @Failure 404 {object} map[string]string "Order not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Security BearerAuth
// @Router /api/v1/orders/{orderID}/received [patch]
func (h *OrderHandler) MarkAsReceived(c *gin.Context) {
	idParam := c.Param("orderID")
	orderID, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid order ID"})
		return
	}

	ctx := c.Request.Context()
	if err := h.orderService.MarkAsReceived(ctx, orderID); err != nil {
		resp := responses.ErrorResponse("order not found", http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, resp)
		return
	}

	resp := responses.SuccessResponse("Order marked as received successfully", ptr.Int(http.StatusOK), nil)
	c.JSON(http.StatusOK, resp)
}
