package handler

import (
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/iservice"
	"fmt"
	"github.com/aws/smithy-go/ptr"
	"github.com/gin-gonic/gin"
	"net/http"
)

type LocationHandler struct {
	locationService iservice.LocationService
}

// GetProvinces godoc
//
//		@Summary	    Get list of provinces from GiaoHangNhanh API
//		@Description	Fetch all provinces	and front-end have to filter himself
//		@Tags			location
//		@Accept			json
//		@Produce		json
//		@Success		200		{object}	responses.ProvinceResponse	"Provinces response"
//		@Router			/api/v1/location/provinces [get]
//	 	@Security		BearerAuth
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
//		@Summary	    Get list of districts from a province
//		@Description	Fetch all districts	and front-end have to filter himself
//		@Tags			location
//		@Accept			json
//		@Produce		json
//	    @Param			province-id	path		int	true	"Province ID"
//		@Success		200		{object}	responses.DistrictResponse	"Provinces response"
//		@Router			/api/v1/location/districts/{province-id} [get]
//	 	@Security		BearerAuth
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
//		@Summary	    Get list of districts from a province
//		@Description	Fetch all districts	and front-end have to filter himself
//		@Tags			location
//		@Accept			json
//		@Produce		json
//	    @Param			district-id	path		int	true	"District ID"
//		@Success		200		{object}	responses.WardResponse	"Ward response"
//		@Router			/api/v1/location/wards/{district-id} [get]
//	 	@Security		BearerAuth
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

func NewLocationHandler(locationService iservice.LocationService) *LocationHandler {
	return &LocationHandler{
		locationService: locationService,
	}
}
