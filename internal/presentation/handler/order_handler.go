package handler

import (
	"core-backend/internal/application/dto/dtos"
	"core-backend/internal/application/interfaces/iproxies"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

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
	fileService          iservice.FileService
}

func NewOrderHandler(orderSvc iservice.OrderService, ghnProxy iproxies.GHNProxy, uow irepository.UnitOfWork, stateTransferService iservice.StateTransferService, fileService iservice.FileService) *OrderHandler {
	return &OrderHandler{
		orderService:         orderSvc,
		ghnProxy:             ghnProxy,
		unitOfWork:           uow,
		stateTransferService: stateTransferService,
		fileService:          fileService,
	}
}

// GetOrdersByUserIDWithPagination handles HTTP GET requests to retrieve paginated orders for a specific user.
//
//	@Summary		Get paginated orders by user ID
//	@Description	This handler extracts pagination parameters (`page`, `limit`) and optional filters from query string.
//	@Tags			Orders
//	@Accept			json
//	@Produce		json
//	@Param			page		query		int		false	"Page number (default: 1)"
//	@Param			limit		query		int		false	"Number of items per page (default: 10, max: 100)"
//	@Param			search		query		string	false	"Search term for filtering orders by order number or GHN code"
//	@Param			status		query		string	false	"Filter by order status"
//	@Param			createdFrom	query		string	false	"Filter by start date (YYYY-MM-DD)"
//	@Param			createdTo	query		string	false	"Filter by end date (YYYY-MM-DD)"
//	@Success		200			{object}	responses.APIResponse{data=[]model.Order,pagination=responses.Pagination}
//	@Failure		401			{object}	responses.APIResponse	"Unauthorized"
//	@Failure		500			{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/orders [get]
func (h *OrderHandler) GetOrdersByUserIDWithPagination(c *gin.Context) {
	// Pagination
	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil || page < 1 {
		page = 1
	}
	limit, err := strconv.Atoi(c.DefaultQuery("limit", "10"))
	if err != nil || limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	// Extract userID from token/context
	userID, err := extractUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, responses.ErrorResponse("unauthorized: "+err.Error(), http.StatusUnauthorized))
		return
	}

	// Filters
	search := c.DefaultQuery("search", "")
	status := c.DefaultQuery("status", "")
	createdFrom := c.DefaultQuery("createdFrom", "")
	createdTo := c.DefaultQuery("createdTo", "")

	// Call service
	orders, total, err := h.orderService.GetOrdersByUserIDWithPagination(userID, limit, page, search, status, createdFrom, createdTo)
	if err != nil {
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse("failed to fetch orders: "+err.Error(), http.StatusInternalServerError))
		return
	}

	// Pagination metadata
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

// CreateOrder godoc
//
//	@Summary		Place an order and initiate payment
//	@Description	Create an order -> calculate delivery fee -> create payment transaction
//	@Tags			Orders
//	@Accept			json
//	@Produce		json
//	@Param			data	body		requests.PlaceAndPayRequest	true	"Place and pay payload"
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

	var deliveryFee *dtos.DeliveryFeeSuccess

	if req.Order.IsSelfPickup == false {
		//*1 Calcucate delivery fee first as we need to validate order dimensions/weight before placing order
		deliveryFee, err = h.ghnProxy.CalculateDeliveryPriceByShippingAddressAndOrderItem(ctx, req.Order.AddressID, req.Order.Items, uow)
		if err != nil {
			zap.L().Error("failed to calculate delivery fee", zap.Error(err))
			c.JSON(http.StatusBadRequest, responses.ErrorResponse("failed to calculate delivery fee: "+err.Error(), http.StatusBadRequest))
			return
		}
	} else {
		deliveryFee = &dtos.DeliveryFeeSuccess{
			Total:                 0,
			ServiceFee:            0,
			InsuranceFee:          0,
			PickStationFee:        0,
			CouponValue:           0,
			R2SFee:                0,
			ReturnAgain:           0,
			DocumentReturn:        0,
			DoubleCheck:           0,
			CodFee:                0,
			PickRemoteAreasFee:    0,
			DeliverRemoteAreasFee: 0,
			CodFailedFee:          0,
		}
	}
	//*2 Create order with payment
	order, err := h.orderService.PlaceOrder(ctx, userID, req.Order, deliveryFee.Total, false, uow)
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

// CreateLimitedOrder godoc
//
//	@Summary		Place an order and initiate payment
//	@Description	Create an order -> calculate delivery fee -> create payment transaction
//	@Tags			Orders
//	@Accept			json
//	@Produce		json
//	@Param			data	body		requests.PlaceAndPayRequest	true	"Place and pay payload"
//	@Success		200		{object}	map[string]any
//	@Failure		400		{object}	map[string]string
//	@Failure		401		{object}	map[string]string
//	@Failure		500		{object}	map[string]string
//	@Security		BearerAuth
//	@Router			/api/v1/orders/limited [post]
func (h *OrderHandler) CreateLimitedOrder(c *gin.Context) {
	//1. Bind request
	var req requests.PlaceAndPayRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("invalid request body: "+err.Error(), http.StatusBadRequest))
		return
	}

	//2. Extract user
	userID, err := extractUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, responses.ErrorResponse("unauthorized: "+err.Error(), http.StatusUnauthorized))
		return
	}
	// Validate
	if len(req.Order.Items) == 0 {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("order must contain at least one item", http.StatusBadRequest))
		return
	}

	//Create context
	ctx := c.Request.Context()
	uow := h.unitOfWork.Begin(ctx)
	defer func() {
		// ensure transaction ended if still open
		if uow.InTransaction() {
			_ = uow.Rollback()
		}
	}()

	//*2 Create order with payment
	order, err := h.orderService.PlaceOrder(ctx, userID, req.Order, 0, true, uow)
	if err != nil {
		_ = uow.Rollback()
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("failed to place order: "+err.Error(), http.StatusBadRequest))
		return
	}

	//*3 Initiate payment
	paymentTx, err := h.orderService.PayOrder(ctx, order.ID, 0, req.SuccessURL, req.CancelURL, uow)
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
		"delivery_fee": nil,
	})
	c.JSON(http.StatusOK, resp)
}

// GetStaffAvailableOrdersWithPagination handles HTTP GET requests to retrieve paginated staff-available orders with optional status filter.
//
//	@Summary		Get staff-available orders with pagination
//	@Description	Retrieve paginated orders for staff, filterable by status and order number search.
//	@Tags			Orders[Staff]
//	@Accept			json
//	@Produce		json
//	@Param			query	query		requests.StaffOrdersQuery	false	"Staff orders query"
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
	orderType := q.OrderType

	orders, total, err := h.orderService.GetStaffAvailableOrdersWithPagination(limit, page, search, status.String(), fullName, phone, provinceID, districtID, wardCode, orderType.String())
	if err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("failed to fetch staff orders: "+err.Error(), http.StatusBadRequest))
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

// GetSelfDeliveringOrdersWithPagination handles HTTP GET requests to retrieve paginated staff-available orders with optional status filter.
//
//	@Summary		Get staff-available orders with pagination
//	@Description	Retrieve paginated orders for staff, filterable by status and order number search.
//	@Tags			Orders[Staff]
//	@Accept			json
//	@Produce		json
//	@Param			query	query		requests.SelfDeliveringQuery	false	"Staff orders query"
//	@Success		200		{object}	responses.APIResponse{data=[]model.Order,pagination=responses.Pagination}
//	@Failure		401		{object}	responses.APIResponse	"Unauthorized"
//	@Failure		500		{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/orders/staff/self-delivering [get]
func (h *OrderHandler) GetSelfDeliveringOrdersWithPagination(c *gin.Context) {
	// Bind query params into struct for swag and cleaner code
	var q requests.SelfDeliveringQuery
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

	orders, total, err := h.orderService.GetSelfDeliveringOrdersWithPagination(limit, page, search, status.String(), fullName, phone, provinceID, districtID, wardCode)
	if err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("failed to fetch staff orders: "+err.Error(), http.StatusBadRequest))
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

// OrderCensorship godoc
//
//	@Summary		Censor an order (confirm or cancel)
//	@Description	Change order state to CONFIRMED or CANCELLED. Use query param `action=CONFIRM` or `action=CANCEL`. If cancelling, provide optional `reason` query param.
//	@Tags			Orders.[Staff].States
//	@Accept			json
//	@Produce		json
//	@Param			orderID	path		string				true	"Order ID"
//	@Param			action	query		string				true	"Action (CONFIRM|CANCEL)"
//	@Param			reason	body		CensorOrderRequest	false	"Cancel reason (required when action=CANCEL)"
//	@Success		200		{object}	responses.APIResponse{data=[]model.Order,pagination=responses.Pagination}
//	@Failure		401		{object}	responses.APIResponse	"Unauthorized"
//	@Failure		500		{object}	responses.APIResponse
//	@Security		BearerAuth
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
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("failed to update order: "+err.Error(), http.StatusBadRequest))
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
//
//	@Summary		Mark order as received
//	@Description	Đánh dấu đơn hàng là "đã nhận" (Received) sau khi giao thành công
//	@Tags			Orders
//	@Accept			json
//	@Produce		json
//	@Param			orderID	path		string					true	"Order ID (UUID)"
//	@Success		200		{object}	map[string]interface{}	"Order marked as received successfully"
//	@Failure		400		{object}	map[string]string		"Invalid order ID"
//	@Failure		404		{object}	map[string]string		"Order not found"
//	@Failure		500		{object}	map[string]string		"Internal server error"
//	@Security		BearerAuth
//	@Router			/api/v1/orders/received/{orderID} [patch]
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

// MarkAsReadyToPickedUp godoc
//
//	@Summary	Mark order as ready to picked up
//	@Description
//	@Tags		Orders[Staff].States
//	@Accept		json
//	@Produce	json
//	@Param		orderID	path		string					true	"Order ID (UUID)"
//	@Success	200		{object}	map[string]interface{}	"Order marked as received successfully"
//	@Failure	400		{object}	map[string]string		"Invalid order ID"
//	@Failure	404		{object}	map[string]string		"Order not found"
//	@Failure	500		{object}	map[string]string		"Internal server error"
//	@Security	BearerAuth
//	@Router		/api/v1/orders/staff/readyToPickedUp/{orderID} [patch]
func (h *OrderHandler) MarkAsReadyToPickedUp(c *gin.Context) {
	idParam := c.Param("orderID")
	orderID, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid order ID"})
		return
	}

	ctx := c.Request.Context()
	if err := h.orderService.MarkAsReadyToPickedUp(ctx, orderID); err != nil {
		resp := responses.ErrorResponse(err.Error(), http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, resp)
		return
	}

	resp := responses.SuccessResponse("Order marked as received successfully", ptr.Int(http.StatusOK), nil)
	c.JSON(http.StatusOK, resp)
}

// MarkAsReceivedAfterPickedUp godoc
//
//	@Summary		Mark order as received after self pick-up
//	@Description	Upload proof image and mark the order as received (only for orders awaiting pick-up)
//	@Tags			Orders[Staff].States
//	@Accept			multipart/form-data
//	@Produce		json
//	@Param			orderID	path		string					true	"Order ID (UUID)"
//	@Param			files	formData	file					true	"Proof image(s) of self pick-up"
//	@Success		200		{object}	map[string]interface{}	"Order marked as received successfully"
//	@Failure		400		{object}	map[string]string		"Invalid order ID or status"
//	@Failure		404		{object}	map[string]string		"Order not found"
//	@Failure		500		{object}	map[string]string		"Internal server error"
//	@Security		BearerAuth
//	@Router			/api/v1/orders/staff/receivedAfterPickup/{orderID} [patch]
func (h *OrderHandler) MarkAsReceivedAfterPickedUp(c *gin.Context) {
	idParam := c.Param("orderID")
	orderID, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid order ID"})
		return
	}

	form, err := c.MultipartForm()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to parse multipart form"})
		return
	}

	files := form.File["files"]
	if len(files) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no files uploaded"})
		return
	}

	userTmpDir := "/tmp/uploads"
	if err := os.MkdirAll(userTmpDir, os.ModePerm); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create tmp upload directory"})
		return
	}

	var uploadedURLs []string
	for _, fileHeader := range files {
		timestamp := time.Now().Format("20060102_150405")
		newFileName := fmt.Sprintf("%s_%s", timestamp, fileHeader.Filename)
		finalPath := fmt.Sprintf("%s/%s", userTmpDir, newFileName)

		// Save uploaded file temporarily
		if err := c.SaveUploadedFile(fileHeader, finalPath); err != nil {
			_ = os.Remove(finalPath)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save file: " + fileHeader.Filename})
			return
		}

		defer func(path string) { _ = os.Remove(path) }(finalPath)

		// Upload to remote storage (e.g., S3, GCS)
		userID := c.GetString("userID") // assuming userID is stored in context by auth middleware
		url, err := h.fileService.UploadFile(userID, finalPath, newFileName)
		if err != nil {
			_ = os.Remove(finalPath)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to upload file: " + fileHeader.Filename + ", " + err.Error()})
			return
		}

		// Cleanup tmp file after upload
		_ = os.Remove(finalPath)
		uploadedURLs = append(uploadedURLs, url)
	}

	// Use the first image as the proof image
	imageURL := uploadedURLs[0]

	ctx := c.Request.Context()
	if err := h.orderService.MarkAsReceivedAfterPickedUp(ctx, orderID, imageURL); err != nil {
		zap.L().Error("Failed to mark order as received after pickup", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp := responses.SuccessResponse("Order marked as received successfully", ptr.Int(http.StatusOK), gin.H{
		"image_url": imageURL,
	})
	c.JSON(http.StatusOK, resp)
}

// MarkSelfDeliveringOrderAsInTransit godoc
//
//	@Summary		Mark self-delivering limited order as In Transit
//	@Description	Only for LIMITED orders with self-delivery (not self pick-up). Requires current status = CONFIRMED.
//	@Tags			Orders[Staff].Limited.States
//	@Accept			json
//	@Produce		json
//	@Param			orderID	path		string					true	"Order ID (UUID)"
//	@Success		200		{object}	map[string]interface{}	"Order marked as in transit successfully"
//	@Failure		400		{object}	map[string]string		"Invalid order ID or status"
//	@Failure		404		{object}	map[string]string		"Order not found"
//	@Failure		500		{object}	map[string]string		"Internal server error"
//	@Security		BearerAuth
//	@Router			/api/v1/orders/staff/self-delivering/in-transit/{orderID} [patch]
func (h *OrderHandler) MarkSelfDeliveringOrderAsInTransit(c *gin.Context) {
	idParam := c.Param("orderID")
	orderID, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid order ID"})
		return
	}

	ctx := c.Request.Context()
	if err := h.orderService.MarkSelfDeliveringOrderAsInTransit(ctx, orderID); err != nil {
		resp := responses.ErrorResponse(err.Error(), http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, resp)
		return
	}

	resp := responses.SuccessResponse("Order marked as in transit successfully", ptr.Int(http.StatusOK), nil)
	c.JSON(http.StatusOK, resp)
}

// MarkSelfDeliveringOrderAsDelivered godoc
//
//	@Summary		Mark self-delivering limited order as Delivered
//	@Description	Upload proof image and mark the order as delivered. Only for LIMITED orders with self-delivery (not self pick-up). Requires current status = IN_TRANSIT.
//	@Tags			Orders[Staff].Limited.States
//	@Accept			multipart/form-data
//	@Produce		json
//	@Param			orderID	path		string					true	"Order ID (UUID)"
//	@Param			files	formData	file					true	"Proof image(s) of self delivering"
//	@Success		200		{object}	map[string]interface{}	"Order marked as delivered successfully"
//	@Failure		400		{object}	map[string]string		"Invalid order ID or status"
//	@Failure		404		{object}	map[string]string		"Order not found"
//	@Failure		500		{object}	map[string]string		"Internal server error"
//	@Security		BearerAuth
//	@Router			/api/v1/orders/staff/self-delivering/delivered/{orderID} [patch]
func (h *OrderHandler) MarkSelfDeliveringOrderAsDelivered(c *gin.Context) {
	idParam := c.Param("orderID")
	orderID, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid order ID"})
		return
	}

	form, err := c.MultipartForm()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to parse multipart form"})
		return
	}

	files := form.File["files"]
	if len(files) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no files uploaded"})
		return
	}

	userTmpDir := "/tmp/uploads"
	if err := os.MkdirAll(userTmpDir, os.ModePerm); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create tmp upload directory"})
		return
	}

	var uploadedURLs []string
	for _, fileHeader := range files {
		timestamp := time.Now().Format("20060102_150405")
		newFileName := fmt.Sprintf("%s_%s", timestamp, fileHeader.Filename)
		finalPath := fmt.Sprintf("%s/%s", userTmpDir, newFileName)

		// Save uploaded file temporarily
		if err := c.SaveUploadedFile(fileHeader, finalPath); err != nil {
			_ = os.Remove(finalPath)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save file: " + fileHeader.Filename})
			return
		}

		defer func(path string) { _ = os.Remove(path) }(finalPath)

		// Upload to remote storage (e.g., S3, GCS)
		userID := c.GetString("userID") // assuming userID is stored in context by auth middleware
		url, err := h.fileService.UploadFile(userID, finalPath, newFileName)
		if err != nil {
			_ = os.Remove(finalPath)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to upload file: " + fileHeader.Filename + ", " + err.Error()})
			return
		}

		// Cleanup tmp file after upload
		_ = os.Remove(finalPath)
		uploadedURLs = append(uploadedURLs, url)
	}

	imageURL := uploadedURLs[0]

	ctx := c.Request.Context()
	if err := h.orderService.MarkSelfDeliveringOrderAsDelivered(ctx, orderID, imageURL); err != nil {
		zap.L().Error("Failed to mark self-delivering order as delivered", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp := responses.SuccessResponse("Order marked as delivered successfully", ptr.Int(http.StatusOK), gin.H{
		"image_url": imageURL,
	})
	c.JSON(http.StatusOK, resp)
}
