package handler

import (
	"context"
	"core-backend/internal/application/dto/dtos"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/iproxies"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/domain/enum"
	"fmt"
	"net/http"

	"github.com/aws/smithy-go/ptr"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// GHNHandler handles GHN delivery-related operations (delivery fee, service availability)
type GHNHandler struct {
	ghnProxy iproxies.GHNProxy
	uow      irepository.UnitOfWork
}

// CalculateDeliveryPriceByOrderID godoc
//
//	@Summary		Calculate delivery fee for a given order
//	@Description	Compute GHN delivery fee based on an existing order ID and selected delivery service
//	@Tags			ghn
//	@Accept			json
//	@Produce		json
//	@Param			order-id	path		string								true	"Order ID (UUID)"
//	@Success		200			{object}	dtos.DeliveryFeeSuccess
//	@Failure		400			{object}	map[string]string
//	@Failure		500			{object}	map[string]string
//	@Security		BearerAuth
//	@Router			/api/v1/ghn/order/{order-id}/calculate [post]
func (h *GHNHandler) CalculateDeliveryPriceByOrderID(c *gin.Context) {
	orderIDStr := c.Param("order-id")
	orderID, err := uuid.Parse(orderIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("invalid order ID format", http.StatusBadRequest))
		return
	}

	ctx := context.Background()
	result, err := h.ghnProxy.CalculateDeliveryPriceByID(ctx, orderID, h.uow)
	if err != nil {
		zap.L().Error("failed to calculate delivery fee", zap.Error(err))
		c.JSON(http.StatusBadRequest, responses.ErrorResponse(fmt.Sprintf("failed to calculate delivery price: %s", err.Error()), http.StatusBadRequest))
		return
	}

	c.JSON(http.StatusOK, responses.SuccessResponse("Delivery price calculated successfully", ptr.Int(http.StatusOK), result))
}

// GetAvailableDeliveryServicesByOrderID godoc
//
//	 @Deprecated
//		@Summary		Get available GHN delivery services for an order
//		@Description	Retrieve list of GHN delivery service options based on the order's destination
//		@Tags			ghn
//		@Accept			json
//		@Produce		json
//		@Param			order-id	path		string	true	"Order ID (UUID)"
//		@Success		200			{array}		dtos.DeliveryAvailableServiceDTO
//		@Failure		400			{object}	map[string]string
//		@Failure		500			{object}	map[string]string
//		@Security		BearerAuth
//		@Router			/api/v1/ghn/order/{order-id}/shipping-services [get]
func (h *GHNHandler) GetAvailableDeliveryServicesByOrderID(c *gin.Context) {
	orderIDStr := c.Param("order-id")
	orderID, err := uuid.Parse(orderIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("invalid order ID format", http.StatusBadRequest))
		return
	}

	ctx := context.Background()
	result, err := h.ghnProxy.GetAvailableDeliveryServicesByOrderID(ctx, orderID, h.uow)
	if err != nil {
		zap.L().Error("failed to fetch delivery services", zap.Error(err))
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse(fmt.Sprintf("failed to fetch delivery services: %s", err.Error()), http.StatusInternalServerError))
		return
	}

	c.JSON(http.StatusOK, responses.SuccessResponse("Available delivery services fetched successfully", ptr.Int(http.StatusOK), result))
}

// GetAvailableDeliveryServicesByDistrictID godoc
//
//	 @Deprecated
//		@Summary		Get GHN delivery services by district ID (public endpoint)
//		@Description	Fetch GHN delivery service options available for a specific district
//		@Tags			ghn
//		@Accept			json
//		@Produce		json
//		@Param			district-id	path		int	true	"District ID"
//		@Success		200			{array}		dtos.DeliveryAvailableServiceDTO
//		@Failure		400			{object}	map[string]string
//		@Failure		500			{object}	map[string]string
//		@Router			/api/v1/ghn/{district-id}/shipping-services [get]
func (h *GHNHandler) GetAvailableDeliveryServicesByDistrictID(c *gin.Context) {
	districtIDStr := c.Param("district-id")
	var districtID int
	_, err := fmt.Sscanf(districtIDStr, "%d", &districtID)
	if err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("district-id must be an integer", http.StatusBadRequest))
		return
	}

	ctx := context.Background()
	result, err := h.ghnProxy.GetAvailableDeliveryServicesByDistrictID(ctx, districtID, h.uow)
	if err != nil {
		zap.L().Error("failed to fetch GHN delivery services by district", zap.Error(err))
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse(fmt.Sprintf("failed to fetch GHN delivery services: %s", err.Error()), http.StatusInternalServerError))
		return
	}

	c.JSON(http.StatusOK, responses.SuccessResponse("Delivery services fetched successfully", ptr.Int(http.StatusOK), result))
}

// CalculateDeliveryPriceByDimensionRequest defines the request body for calculating delivery fee by dimensions/items
type CalculateDeliveryPriceByDimensionRequest struct {
	ToDistrictID int                               `json:"to_district_id" example:"1454"`
	ToWardCode   string                            `json:"to_ward_code" example:"012345"`
	Items        []dtos.ApplicationDeliveryFeeItem `json:"items"`
}

// CalculateDeliveryPriceByDimension godoc
//
//	@Summary		Calculate delivery fee by explicit destination and items
//	@Description	Compute GHN delivery fee by providing destination district/ward and a list of items (dimensions/weight)
//	@Tags			ghn
//	@Accept			json
//	@Produce		json
//	@Param			body	body		CalculateDeliveryPriceByDimensionRequest	true	"Delivery fee request"
//	@Success		200		{object}	dtos.DeliveryFeeSuccess
//	@Security		BearerAuth
//	@Router			/api/v1/ghn/delivery/calculate-by-dimension [post]
func (h *GHNHandler) CalculateDeliveryPriceByDimension(c *gin.Context) {
	var req CalculateDeliveryPriceByDimensionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("invalid request body: "+err.Error(), http.StatusBadRequest))
		return
	}

	ctx := context.Background()
	result, err := h.ghnProxy.CalculateDeliveryPriceByDimensionItems(ctx, req.ToDistrictID, req.ToWardCode, req.Items, h.uow)
	if err != nil {
		zap.L().Error("failed to calculate delivery price by dimension", zap.Error(err))
		c.JSON(http.StatusBadRequest, responses.ErrorResponse(fmt.Sprintf("failed to calculate delivery price: %s", err.Error()), http.StatusBadRequest))
		return
	}

	c.JSON(http.StatusOK, responses.SuccessResponse("Delivery price calculated successfully", ptr.Int(http.StatusOK), result))
}

// GetOrderInfo godoc
//
//	@Summary		Get GHN order info by GHN order code
//	@Description	Fetch GHN order details for a given GHN order code
//	@Tags		ghn
//	@Accept		json
//	@Produce	json
//	@Param		order-id	path	string	true	"ID of Order that related to GHN order (not Limited)"
//	@Success	200		{object}	dtos.OrderInfo
//	@Failure	400		{object}	map[string]string
//	@Failure	500		{object}	map[string]string
//	@Security	BearerAuth
//	@Router		/api/v1/ghn/order/info/{order-id} [get]
func (h *GHNHandler) GetOrderInfo(c *gin.Context) {
	orderID := c.Param("order-id")
	if orderID == "" {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("order code is required", http.StatusBadRequest))
		return
	}

	ctx := context.Background()
	result, err := h.ghnProxy.GetOrderInfo(ctx, orderID)
	if err != nil {
		zap.L().Error("failed to fetch GHN order info", zap.Error(err), zap.String("order-id", orderID))
		c.JSON(http.StatusBadRequest, responses.ErrorResponse(fmt.Sprintf("failed to fetch GHN order info: %s", err.Error()), http.StatusBadRequest))
		return
	}

	c.JSON(http.StatusOK, responses.SuccessResponse("GHN order info fetched successfully", ptr.Int(http.StatusOK), result))
}

// GetExpectedDeliveryTime godoc
//
//	@Summary		Get expected delivery time from GHN
//	@Description	Retrieve estimated delivery lead time for a destination (district + ward)
//	@Tags		ghn
//	@Accept		json
//	@Produce	json
//	@Param		to_district_id	query	int	true	"Destination district ID"
//	@Param		to_ward_code	query	string	true	"Destination ward code"
//	@Success	200		{object}	dtos.ExpectedDeliveryTime
//	@Failure	400		{object}	map[string]string
//	@Failure	500		{object}	map[string]string
//	@Router		/api/v1/ghn/expected-delivery-time [get]
func (h *GHNHandler) GetExpectedDeliveryTime(c *gin.Context) {
	districtStr := c.Query("to_district_id")
	wardCode := c.Query("to_ward_code")
	if districtStr == "" || wardCode == "" {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("to_district_id and to_ward_code query params are required", http.StatusBadRequest))
		return
	}

	var districtID int
	_, err := fmt.Sscanf(districtStr, "%d", &districtID)
	if err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("to_district_id must be an integer", http.StatusBadRequest))
		return
	}

	ctx := context.Background()
	result, err := h.ghnProxy.GetExpectedDeliveryTime(ctx, districtID, wardCode)
	if err != nil {
		zap.L().Error("failed to fetch expected delivery time", zap.Error(err))
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse(fmt.Sprintf("failed to fetch expected delivery time: %s", err.Error()), http.StatusInternalServerError))
		return
	}

	c.JSON(http.StatusOK, responses.SuccessResponse("Expected delivery time fetched successfully", ptr.Int(http.StatusOK), result))
}

// GetGHNSession godoc
//
//	@Summary		Get GHN session token
//	@Description	Retrieve a session token from GHN for authenticated requests
//	@Tags		ghn-mocking
//	@Accept		json
//	@Produce	json
//	@Success	200		{object}	dtos.GHNSessionResponse
//	@Failure	500		{object}	map[string]string
//	@Router		/api/v1/ghn/mocking/session [get]
func (h *GHNHandler) GetGHNSession(c *gin.Context) {
	ctx := context.Background()
	result, err := h.ghnProxy.GetSession(ctx)
	if err != nil {
		zap.L().Error("failed to fetch GHN session", zap.Error(err))
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse(fmt.Sprintf("failed to fetch GHN session: %s", err.Error()), http.StatusInternalServerError))
		return
	}

	c.JSON(http.StatusOK, responses.SuccessResponse("GHN session fetched successfully", ptr.Int(http.StatusOK), result))
}

// GetGHNServiceToken godoc
//
//	@Summary		Get GHN Service Token (step 2)
//	@Description	Retrieve GHN Service Token using GHN session
//	@Tags			ghn-mocking
//	@Accept			json
//	@Produce		json
//	@Param			token	query	string	true	"GHN Session Token"
//	@Success		200		{object}	dtos.GHNServiceToken
//	@Failure		400		{object}	map[string]string
//	@Failure		500		{object}	map[string]string
//	@Router			/api/v1/ghn/mocking/service-token [get]
func (h *GHNHandler) GetGHNServiceToken(c *gin.Context) {
	ghnSession := c.Query("token")
	if ghnSession == "" {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("token query param is required", http.StatusBadRequest))
		return
	}

	ctx := context.Background()
	result, err := h.ghnProxy.GetGHNServiceToken(ctx, ghnSession)
	if err != nil {
		zap.L().Error("failed to get GHN service token", zap.Error(err))
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse(fmt.Sprintf("failed to get GHN service token: %s", err.Error()), http.StatusInternalServerError))
		return
	}

	c.JSON(http.StatusOK, responses.SuccessResponse("GHN service token fetched successfully", ptr.Int(http.StatusOK), result))
}

// GetGHNGSOToken godoc
//
//	@Summary		Get GHN GSO Token (step 3)
//	@Description	Retrieve GHN GSO Token using Service Token
//	@Tags			ghn-mocking
//	@Accept			json
//	@Produce		json
//	@Param			service_token	query	string	true	"GHN Service Token"
//	@Success		200		{object}	dtos.GHNTokenGSO
//	@Failure		400		{object}	map[string]string
//	@Failure		500		{object}	map[string]string
//	@Router			/api/v1/ghn/mocking/gso-token [get]
func (h *GHNHandler) GetGHNGSOToken(c *gin.Context) {
	serviceToken := c.Query("service_token")
	if serviceToken == "" {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("service_token query param is required", http.StatusBadRequest))
		return
	}

	ctx := context.Background()
	result, err := h.ghnProxy.GetGHNGSOToken(ctx, serviceToken)
	if err != nil {
		zap.L().Error("failed to get GHN GSO token", zap.Error(err))
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse(fmt.Sprintf("failed to get GHN GSO token: %s", err.Error()), http.StatusInternalServerError))
		return
	}

	c.JSON(http.StatusOK, responses.SuccessResponse("GHN GSO token fetched successfully", ptr.Int(http.StatusOK), result))
}

type UpdateGHNDeliveryStatusRequest struct {
	OrderCode string                 `json:"order_code" example:"L4TFM8" binding:"required"`
	Status    enum.GHNDeliveryStatus `json:"status" example:"storing" binding:"required"`
}

// UpdateGHNDeliveryStatus godoc
// @Summary Update GHN Order Delivery Status
// @Description Allowed values: ready_to_pick, storing, delivering, delivered, cancel
// @Tags ghn
// @Accept json
// @Produce json
// @Param request body UpdateGHNDeliveryStatusRequest true "Order status update payload"
// @Success 200 {object} dtos.UpdateGHNDeliveryStatusResponse
// @Failure 400 {object} map[string]string "Bad Request"
// @Failure 500 {object} map[string]string "Internal Server Error"
// @Router /api/v1/ghn/order/status [post]
func (h *GHNHandler) UpdateGHNDeliveryStatus(c *gin.Context) {
	var req UpdateGHNDeliveryStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate enum
	if !req.Status.IsValid() {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("invalid status value: %s. Allowed values: ready_to_pick, storing, delivering, delivered, cancel", req.Status),
		})
		return
	}

	ctx := c.Request.Context()
	status := enum.GHNDeliveryStatus(req.Status)

	result, err := h.ghnProxy.UpdateGHNDeliveryStatus(ctx, req.OrderCode, status)
	if err != nil {
		zap.L().Error("failed to update GHN delivery status", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	resp := responses.SuccessResponse("Updated GHN delivery status successfully", ptr.Int(http.StatusOK), result)
	c.JSON(http.StatusOK, resp)
}

// NewGHNHandler creates a new GHNHandler instance
func NewGHNHandler(ghnProxy iproxies.GHNProxy, uow irepository.UnitOfWork) *GHNHandler {
	return &GHNHandler{
		ghnProxy: ghnProxy,
		uow:      uow,
	}
}
