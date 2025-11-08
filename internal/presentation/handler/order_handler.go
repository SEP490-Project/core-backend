package handler

import (
	"core-backend/internal/application/interfaces/iservice_third_party"
	"net/http"
	"strconv"

	"go.uber.org/zap"

	"github.com/aws/smithy-go/ptr"

	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/application/interfaces/iservice"

	"github.com/gin-gonic/gin"
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
	deliveryFee, err := h.ghnService.CalculateDeliveryPriceByShippingAddressAndOrderItem(ctx, req.Order.AddressID, req.Order.Items, uow)
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
