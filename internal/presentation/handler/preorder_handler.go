package handler

import (
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/application/interfaces/iservice"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type PreOrderHandler struct {
	preOrderService iservice.PreOrderService
	unitOfWork      irepository.UnitOfWork
}

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

	// TODO: wire to service when implemented
	_ = userID
	_ = search

	// placeholder response
	c.JSON(http.StatusOK, responses.SuccessResponse("Not implemented", nil, nil))
}

// CreatePreOrder godoc
// @Summary Create a PreOrder (reserve stock)
// @Description Reserve a product variant as a preorder. This will decrement variant stock and create a preorder record.
// @Tags Preorders
// @Accept json
// @Produce json
// @Param data body requests.PreOrderRequest true "PreOrder payload"
// @Success 201 {object} responses.APIResponse
// @Failure 400 {object} responses.APIResponse
// @Failure 401 {object} responses.APIResponse
// @Failure 500 {object} responses.APIResponse
// @Security BearerAuth
// @Router /api/v1/preorders [post]
func (p *PreOrderHandler) CreatePreOrder(c *gin.Context) {
	var req requests.PreOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("invalid request body: "+err.Error(), http.StatusBadRequest))
		return
	}

	//userID, err := extractUserID(c)
	//if err != nil {
	//	c.JSON(http.StatusUnauthorized, responses.ErrorResponse("unauthorized: "+err.Error(), http.StatusUnauthorized))
	//	return
	//}

	ctx := c.Request.Context()
	uow := p.unitOfWork.Begin(ctx)
	defer func() {
		if uow.InTransaction() {
			_ = uow.Rollback()
		}
	}()

	preorder, err := p.preOrderService.PreserverOrder(ctx, req, uow)
	if err != nil {
		_ = uow.Rollback()
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("failed to create preorder: "+err.Error(), http.StatusBadRequest))
		return
	}

	if err := uow.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse("failed to commit transaction: "+err.Error(), http.StatusInternalServerError))
		return
	}

	c.JSON(http.StatusCreated, responses.SuccessResponse("Created successful", nil, preorder))
}

func NewPreOrderHandler(preOrderService iservice.PreOrderService, uow irepository.UnitOfWork) *PreOrderHandler {
	return &PreOrderHandler{
		preOrderService: preOrderService,
		unitOfWork:      uow,
	}
}
