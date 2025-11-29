package handler

import (
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/application/interfaces/iservice"
	"core-backend/internal/domain/constant"
	"core-backend/pkg/utils"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

type AdminConfigHandler struct {
	adminConfigService iservice.AdminConfigService
	unitOfWork         irepository.UnitOfWork
	validator          *validator.Validate
}

func NewAdminConfigHandler(
	adminConfigService iservice.AdminConfigService,
	unitOfWork irepository.UnitOfWork,
) *AdminConfigHandler {
	validator := validator.New()
	return &AdminConfigHandler{
		adminConfigService: adminConfigService,
		unitOfWork:         unitOfWork,
		validator:          validator,
	}
}

// GetAllConfigValues godoc
//
//	@Summary		Get all config values
//	@Description	Retrieve all config values
//	@Tags			Admin Config
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	responses.AdminConfigListResponse	"Config values retrieved successfully"
//	@Failure		500	{object}	responses.APIResponse				"Internal server error"
//	@Failure		401	{object}	responses.APIResponse				"Unauthorized"
//	@Failure		403	{object}	responses.APIResponse				"Forbidden"
//	@Security		BearerAuth
//	@Router			/api/v1/configs [get]
func (h *AdminConfigHandler) GetAllConfigValues(c *gin.Context) {
	configResponses, err := h.adminConfigService.GetAllConfig(c.Request.Context())
	if err != nil {
		response := responses.ErrorResponse("Failed to get all config values: "+err.Error(), http.StatusInternalServerError)
		c.JSON(http.StatusInternalServerError, response)
		return
	}

	response := responses.NewPaginationResponse("Successfully retrieved all config values",
		http.StatusOK,
		configResponses,
		responses.Pagination{
			Page:  1,
			Limit: len(configResponses),
			Total: int64(len(configResponses)),
		},
	)
	c.JSON(http.StatusOK, response)
}

// GetRepresentativeConfigs godoc
//
//	@Summary		Get representative config values
//	@Description	Retrieve representative config values
//	@Tags			Admin Config
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	responses.APIResponse	"Representative config values retrieved successfully"
//	@Failure		500	{object}	responses.APIResponse	"Internal server error"
//	@Failure		401	{object}	responses.APIResponse	"Unauthorized"
//	@Failure		403	{object}	responses.APIResponse	"Forbidden"
//	@Security		BearerAuth
//	@Router			/api/v1/configs/representative [get]
func (h *AdminConfigHandler) GetRepresentativeConfigs(c *gin.Context) {
	keys := []string{
		constant.ConfigKeyRepresentativeName.String(),
		constant.ConfigKeyRepresentativeRole.String(),
		constant.ConfigKeyRepresentativePhone.String(),
		constant.ConfigKeyRepresentativeEmail.String(),
		constant.ConfigKeyRepresentativeTaxNumber.String(),
		constant.ConfigKeyRepresentativeBankName.String(),
		constant.ConfigKeyRepresentativeBankAccountNumber.String(),
		constant.ConfigKeyRepresentativeBankAccountHolder.String(),
		constant.ConfigKeyRepresentativeCompanyAddress.String(),
	}

	configResponses, err := h.adminConfigService.GetConfigValuesByKeys(c.Request.Context(), keys)
	if err != nil {
		response := responses.ErrorResponse("Failed to get representative config values: "+err.Error(), http.StatusInternalServerError)
		c.JSON(http.StatusInternalServerError, response)
		return
	}

	resposne := responses.SuccessResponse("Successfully retrieved representative config values", utils.IntPtr(http.StatusOK), configResponses)
	c.JSON(http.StatusOK, resposne)
}
