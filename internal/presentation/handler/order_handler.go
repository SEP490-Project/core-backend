package handler

import (
	"core-backend/internal/application/dto/dtos"
	"core-backend/internal/application/interfaces/iservice_third_party"
	"net/http"
	"strconv"

	"github.com/aws/smithy-go/ptr"

	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/application/interfaces/iservice"
	"core-backend/internal/domain/model"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type OrderHandler struct {
	orderService iservice.OrderService
	ghnService   iservice_third_party.GHNService
	unitOfWork   irepository.UnitOfWork
}

func NewOrderHandler(orderSvc iservice.OrderService, ghnService iservice_third_party.GHNService, uow irepository.UnitOfWork) *OrderHandler {
	return &OrderHandler{
		orderService: orderSvc,
		ghnService:   ghnService,
		unitOfWork:   uow,
	}
}

// PlaceOrder godoc
//
//	@Summary		Place an order
//	@Description	Create an order for the authenticated user
//	@Tags			Orders
//	@Accept			json
//	@Produce		json
//	@Param			data	body		requests.OrderRequest	true	"Order payload"
//	@Success		201		{object}	map[string]any
//	@Failure		400		{object}	map[string]string
//	@Failure		401		{object}	map[string]string
//	@Failure		500		{object}	map[string]string
//	@Security		BearerAuth
//	@Router			/api/v1/orders [post]
func (h *OrderHandler) PlaceOrder(c *gin.Context) {
	// Bind request
	var req requests.OrderRequest
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

	if len(req.Items) == 0 {
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

	order, err := h.orderService.PlaceOrder(ctx, userID, req, uow)
	if err != nil {
		// rollback already handled in service in many places; ensure rollback
		_ = uow.Rollback()
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse("failed to place order: "+err.Error(), http.StatusInternalServerError))
		return
	}

	if err := uow.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse("failed to commit transaction: "+err.Error(), http.StatusInternalServerError))
		return
	}

	resp := responses.SuccessResponse("Create order successfully", ptr.Int(http.StatusCreated), order)
	c.JSON(http.StatusCreated, resp)
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
//	@Param			page	query		int		false	"Page number (default: 1)"
//	@Param			limit	query		int		false	"Number of items per page (default: 10, max: 100)"
//	@Param			search	query		string	false	"Search term for filtering orders by order number"
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

// PayOrder godoc
//
//	@Summary		Initiate payment for an order
//	@Description	Generate payment link and create payment transaction for the given order
//	@Tags			Orders
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string	true	"Order ID (UUID)"
//	@Success		200	{object}	map[string]any
//	@Failure		400	{object}	map[string]string
//	@Failure		401	{object}	map[string]string
//	@Failure		404	{object}	map[string]string
//	@Failure		500	{object}	map[string]string
//	@Security		BearerAuth
//	@Router			/api/v1/orders/{id}/pay [post]
func (h *OrderHandler) PayOrder(c *gin.Context) {
	orderIDStr := c.Param("id")
	orderID, err := uuid.Parse(orderIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("invalid order id: "+err.Error(), http.StatusBadRequest))
		return
	}

	ctx := c.Request.Context()
	uow := h.unitOfWork.Begin(ctx)
	defer func() {
		if uow.InTransaction() {
			_ = uow.Rollback()
		}
	}()

	paymentTx, err := h.orderService.PayOrder(ctx, orderID, uow)
	if err != nil {
		_ = uow.Rollback()
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse("failed to initiate payment: "+err.Error(), http.StatusInternalServerError))
		return
	}

	if err := uow.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse("failed to commit transaction: "+err.Error(), http.StatusInternalServerError))
		return
	}

	resp := responses.SuccessResponse("Payment initiated", ptr.Int(http.StatusOK), paymentTx)
	c.JSON(http.StatusOK, resp)
}

// PlaceAndPayOrder godoc
//
//	@Summary		Place an order and initiate payment
//	@Description	Create an order and immediately calculate delivery fee and create payment transaction
//	@Tags			Orders
//	@Accept			json
//	@Produce		json
//	@Param			data	body		requests.PlaceAndPayRequest	true	"Place and pay payload"
//	@Success		200		{object}	map[string]any
//	@Failure		400		{object}	map[string]string
//	@Failure		401		{object}	map[string]string
//	@Failure		500		{object}	map[string]string
//	@Security		BearerAuth
//	@Router			/api/v1/orders/place-and-pay [post]
func (h *OrderHandler) PlaceAndPayOrder(c *gin.Context) {
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

	// 1) Create Order
	order, err := h.orderService.PlaceOrder(ctx, userID, req.Order, uow)
	if err != nil {
		_ = uow.Rollback()
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse("failed to place order: "+err.Error(), http.StatusInternalServerError))
		return
	}

	// 2) Calculate delivery fee using GHN service. If request contains a selected delivery service, pass it.
	var deliveryService dtos.DeliveryAvailableServiceDTO
	if req.DeliveryService != nil {
		deliveryService = *req.DeliveryService
	}

	var deliveryFee *dtos.DeliveryFeeSuccess
	deliveryFee, err = h.ghnService.CalculateDeliveryPriceByID(ctx, order.ID, deliveryService, uow)
	if err != nil {
		_ = uow.Rollback()
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse("failed to calculate delivery fee: "+err.Error(), http.StatusInternalServerError))
		return
	}

	// 3) Initiate payment
	var paymentTx *model.PaymentTransaction
	paymentTx, err = h.orderService.PayOrder(ctx, order.ID, uow)
	if err != nil {
		_ = uow.Rollback()
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse("failed to initiate payment: "+err.Error(), http.StatusInternalServerError))
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
