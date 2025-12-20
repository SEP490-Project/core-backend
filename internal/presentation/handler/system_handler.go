package handler

import (
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/iservice"
	"net/http"

	"github.com/gin-gonic/gin"
)

type SystemHandler struct {
	systemService iservice.SystemService
}

func NewSystemHandler(systemService iservice.SystemService) *SystemHandler {
	return &SystemHandler{
		systemService: systemService,
	}
}

// GetSystemSpecs godoc
//
//	@Summary		Get System Specs
//	@Description	Returns the system specifications and runtime statistics
//	@Tags			System
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	responses.APIResponse{data=responses.SystemSpecsResponse}
//	@Failure		500	{object}	responses.APIResponse
//	@Router			/admin/system/specs [get]
//	@Security		BearerAuth
func (h *SystemHandler) GetSystemSpecs(c *gin.Context) {
	specs, err := h.systemService.GetSystemSpecs(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse(err.Error(), http.StatusInternalServerError))
		return
	}

	c.JSON(http.StatusOK, responses.SuccessResponse("System specs retrieved successfully", nil, specs))
}
