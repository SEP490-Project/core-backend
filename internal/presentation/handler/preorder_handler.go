package handler

import (
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/application/interfaces/iservice"
	"net/http"
	"strconv"

	"github.com/aws/smithy-go/ptr"

	"github.com/gin-gonic/gin"
)

type PreOrderHandler struct {
	preOrderService iservice.PreOrderService
	unitOfWork      irepository.UnitOfWork
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
//	@Param			status	query		string	false	"Filter by status (PENDING, PRE_ORDERED, AWAITING_RELEASE, AWAITING_PICKUP, CONFIRMED, CANCELLED, IN_TRANSIT, DELIVERED, RECEIVED)"
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
	status := c.DefaultQuery("status", "")

	items, total, err := p.preOrderService.GetPreOrdersByUserIDWithPagination(userID, limit, page, search, status)
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

	preorders, total, err := p.preOrderService.GetStaffAvailablePreOrdersWithPagination(limit, page, search, status.String(), fullName, phone, provinceID, districtID, wardCode)
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

func NewPreOrderHandler(preOrderService iservice.PreOrderService, uow irepository.UnitOfWork) *PreOrderHandler {
	return &PreOrderHandler{
		preOrderService: preOrderService,
		unitOfWork:      uow,
	}
}
