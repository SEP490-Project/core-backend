package handler

import (
	"context"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/iservice"
	"core-backend/internal/infrastructure/scheduler"
	"fmt"
	"net/http"

	"github.com/aws/smithy-go/ptr"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type LocationHandler struct {
	locationService  iservice.LocationService
	locationSyncTask scheduler.TaskScheduler
}

// GetProvinces godoc
//
//	@Summary		Get list of provinces from GiaoHangNhanh API
//	@Description	Fetch all provinces	and front-end have to filter himself
//	@Tags			location
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	responses.ProvinceResponse	"Provinces response"
//	@Router			/api/v1/location/provinces [get]
//	@Security		BearerAuth
func (h *LocationHandler) GetProvinces(c *gin.Context) {
	result, err := h.locationService.GetProvinces()
	if err != nil {
		resp := responses.ErrorResponse(fmt.Sprintf("Failed to fetch Provinces: %s", err.Error()), http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, resp)
		return
	}
	resp := responses.SuccessResponse("Provinces fetched successfully", ptr.Int(http.StatusOK), result)
	c.JSON(http.StatusOK, resp)
}

// GetDistricts godoc
//
//	@Summary		Get list of districts from a province
//	@Description	Fetch all districts	and front-end have to filter himself
//	@Tags			location
//	@Accept			json
//	@Produce		json
//	@Param			province-id	path		int							true	"Province ID"
//	@Success		200			{object}	responses.DistrictResponse	"Provinces response"
//	@Router			/api/v1/location/districts/{province-id} [get]
//	@Security		BearerAuth
func (h *LocationHandler) GetDistricts(c *gin.Context) {
	provinceIDStr := c.Param("province-id")
	if provinceIDStr == "" {
		resp := responses.ErrorResponse("province_id query parameter is required", http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, resp)
		return
	}
	var provinceID int
	_, err := fmt.Sscanf(provinceIDStr, "%d", &provinceID)
	if err != nil {
		resp := responses.ErrorResponse("province-id must be an integer", http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, resp)
		return
	}

	result, err := h.locationService.GetDistrictsByProvinceID(provinceID)
	if err != nil {
		resp := responses.ErrorResponse(fmt.Sprintf("Failed to fetch Districts: %s", err.Error()), http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, resp)
		return
	}
	resp := responses.SuccessResponse("Districts fetched successfully", ptr.Int(http.StatusOK), result)
	c.JSON(http.StatusOK, resp)
}

// GetWards godoc
//
//	@Summary		Get list of districts from a province
//	@Description	Fetch all districts	and front-end have to filter himself
//	@Tags			location
//	@Accept			json
//	@Produce		json
//	@Param			district-id	path		int						true	"District ID"
//	@Success		200			{object}	responses.WardResponse	"Ward response"
//	@Router			/api/v1/location/wards/{district-id} [get]
//	@Security		BearerAuth
func (h *LocationHandler) GetWards(c *gin.Context) {
	districtIDStr := c.Param("district-id")
	if districtIDStr == "" {
		resp := responses.ErrorResponse("district-id query parameter is required", http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, resp)
		return
	}
	var districtID int
	_, err := fmt.Sscanf(districtIDStr, "%d", &districtID)
	if err != nil {
		resp := responses.ErrorResponse("district-id must be an integer", http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, resp)
		return
	}

	result, err := h.locationService.GetWardsByDistrictID(districtID)
	if err != nil {
		resp := responses.ErrorResponse(fmt.Sprintf("Failed to fetch Wards: %s", err.Error()), http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, resp)
		return
	}
	resp := responses.SuccessResponse("Wards fetched successfully", ptr.Int(http.StatusOK), result)
	c.JSON(http.StatusOK, resp)
}

// InputUserAddress godoc
//
//	@Summary		Create a new shipping address for the authenticated user
//	@Description	Persist a shipping address for the current authenticated user
//	@Tags			location
//	@Accept			json
//	@Produce		json
//	@Param			body	body		requests.InputAddressRequest		true	"Address payload"
//	@Success		201		{object}	responses.ShippingAddressResponse	"Address created"
//	@Failure		400		{object}	map[string]string
//	@Failure		401		{object}	map[string]string
//	@Security		BearerAuth
//	@Router			/api/v1/location/address [post]
func (h *LocationHandler) InputUserAddress(c *gin.Context) {
	var req requests.InputAddressRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		resp := responses.ErrorResponse("invalid request body: "+err.Error(), http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, resp)
		return
	}

	userID, err := extractUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, responses.ErrorResponse(err.Error(), http.StatusUnauthorized))
		return
	}
	addr, err := h.locationService.InputUserAddress(userID, req)
	if err != nil {
		resp := responses.ErrorResponse(fmt.Sprintf("failed to create address: %s", err.Error()), http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, resp)
		return
	}

	resp := responses.ShippingAddressResponse{}.ToResponse(addr)
	c.JSON(http.StatusCreated, responses.SuccessResponse("Address created successfully", ptr.Int(http.StatusCreated), resp))
}

// SetAddressAsDefault godoc
//
//	@Summary		Set an address as default for the authenticated user
//	@Description	Mark the given address as the default address for the current user
//	@Tags			location
//	@Accept			json
//	@Produce		json
//	@Param			address-id	path		string	true	"Address ID"
//	@Success		200			{object}	map[string]string
//	@Failure		400			{object}	map[string]string
//	@Failure		401			{object}	map[string]string
//	@Security		BearerAuth
//	@Router			/api/v1/location/address/{address-id}/default [patch]
func (h *LocationHandler) SetAddressAsDefault(c *gin.Context) {
	addressID := c.Param("address-id")
	if addressID == "" {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("address-id is required", http.StatusBadRequest))
		return
	}

	userIDVal, ok := c.Get("user_id")
	if !ok || userIDVal == nil {
		c.JSON(http.StatusUnauthorized, responses.ErrorResponse("missing user id in context", http.StatusUnauthorized))
		return
	}

	var userIDStr string
	switch v := userIDVal.(type) {
	case uuid.UUID:
		userIDStr = v.String()
	case string:
		userIDStr = v
	default:
		c.JSON(http.StatusUnauthorized, responses.ErrorResponse("invalid user id format in context", http.StatusUnauthorized))
		return
	}

	if err := h.locationService.SetAddressAsDefault(userIDStr, addressID); err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse(fmt.Sprintf("failed to set default address: %s", err.Error()), http.StatusBadRequest))
		return
	}

	c.JSON(http.StatusOK, responses.SuccessResponse("Address set as default", ptr.Int(http.StatusOK), nil))
}

// GetUserAddresses godoc
//
//	@Summary		Get addresses of authenticated user
//	@Description	Retrieve all shipping addresses belonging to the authenticated user
//	@Tags			location
//	@Accept			json
//	@Produce		json
//	@Success		200	{array}		responses.ShippingAddressResponse
//	@Failure		401	{object}	map[string]string
//	@Security		BearerAuth
//	@Router			/api/v1/location/addresses [get]
func (h *LocationHandler) GetUserAddresses(c *gin.Context) {
	userIDVal, ok := c.Get("user_id")
	if !ok || userIDVal == nil {
		c.JSON(http.StatusUnauthorized, responses.ErrorResponse("missing user id in context", http.StatusUnauthorized))
		return
	}

	var userID uuid.UUID
	switch v := userIDVal.(type) {
	case uuid.UUID:
		userID = v
	case string:
		uid, err := uuid.Parse(v)
		if err != nil {
			c.JSON(http.StatusUnauthorized, responses.ErrorResponse("invalid user id in context", http.StatusUnauthorized))
			return
		}
		userID = uid
	default:
		c.JSON(http.StatusUnauthorized, responses.ErrorResponse("invalid user id format in context", http.StatusUnauthorized))
		return
	}

	addrs, err := h.locationService.GetUserAddresses(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse("failed to fetch addresses: "+err.Error(), http.StatusInternalServerError))
		return
	}

	respList := responses.ShippingAddressResponse{}.ToResponseList(addrs)
	c.JSON(http.StatusOK, responses.SuccessResponse("Addresses fetched successfully", ptr.Int(http.StatusOK), respList))
}

// TriggerLocationSync godoc
//
//	@Summary		Trigger location synchronization
//	@Description	Manually trigger the synchronization of location data
//	@Tags			location
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	map[string]string	"Success message"
//	@Failure		400	{object}	map[string]string	"Error message"
//	@Security		BearerAuth
//	@Router			/api/v1/location/sync [post]
func (h *LocationHandler) TriggerLocationSync(c *gin.Context) {
	// spawn background sync using a detached context so the handler can return immediately
	go func() {
		ctx := context.Background()
		zap.L().Info("Starting location sync", zap.String("mode", "manual"))
		if err := h.locationSyncTask.StartOnce(ctx); err != nil {
			zap.L().Error("Location sync failed (manual)", zap.Error(err))
		}
	}()

	resp := responses.SuccessResponse("Location sync triggered successfully", ptr.Int(http.StatusOK), nil)
	c.JSON(http.StatusOK, resp)
}

func NewLocationHandler(locationService iservice.LocationService, locationSyncTask scheduler.TaskScheduler) *LocationHandler {
	return &LocationHandler{
		locationService:  locationService,
		locationSyncTask: locationSyncTask,
	}
}
