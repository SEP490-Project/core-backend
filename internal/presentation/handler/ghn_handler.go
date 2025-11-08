package handler

import (
	"context"
	"core-backend/internal/application/dto/dtos"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/irepository"
	iExtService "core-backend/internal/application/interfaces/iservice_third_party"
	"fmt"
	"net/http"

	"github.com/aws/smithy-go/ptr"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// GHNHandler handles GHN delivery-related operations (delivery fee, service availability)
type GHNHandler struct {
	ghnService iExtService.GHNService
	uow        irepository.UnitOfWork
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
	result, err := h.ghnService.CalculateDeliveryPriceByID(ctx, orderID, h.uow)
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
	result, err := h.ghnService.GetAvailableDeliveryServicesByOrderID(ctx, orderID, h.uow)
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
	result, err := h.ghnService.GetAvailableDeliveryServicesByDistrictID(ctx, districtID, h.uow)
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
	result, err := h.ghnService.CalculateDeliveryPriceByDimensionItems(ctx, req.ToDistrictID, req.ToWardCode, req.Items, h.uow)
	if err != nil {
		zap.L().Error("failed to calculate delivery price by dimension", zap.Error(err))
		c.JSON(http.StatusBadRequest, responses.ErrorResponse(fmt.Sprintf("failed to calculate delivery price: %s", err.Error()), http.StatusBadRequest))
		return
	}

	c.JSON(http.StatusOK, responses.SuccessResponse("Delivery price calculated successfully", ptr.Int(http.StatusOK), result))
}

// NewGHNHandler creates a new GHNHandler instance
func NewGHNHandler(ghnService iExtService.GHNService, uow irepository.UnitOfWork) *GHNHandler {
	return &GHNHandler{
		ghnService: ghnService,
		uow:        uow,
	}
}
