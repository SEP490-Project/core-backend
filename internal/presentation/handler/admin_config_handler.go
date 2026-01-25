package handler

import (
	"core-backend/internal/application/dto/requests"
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

// UpdateConfig godoc
//
//	@Summary		Update a config value
//	@Description	Update a single config value by key
//	@Tags			Admin Config
//	@Accept			json
//	@Produce		json
//	@Param			key		path		string								true	"Config Key"
//	@Param			request	body		requests.UpdateAdminConfigRequest	true	"Update Request"
//	@Success		200		{object}	responses.APIResponse				"Config updated successfully"
//	@Failure		400		{object}	responses.APIResponse				"Bad Request"
//	@Failure		500		{object}	responses.APIResponse				"Internal server error"
//	@Security		BearerAuth
//	@Router			/api/v1/configs/{key} [put]
func (h *AdminConfigHandler) UpdateConfig(c *gin.Context) {
	key := c.Param("key")
	var req requests.UpdateAdminConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid request body", http.StatusBadRequest))
		return
	}
	if err := h.validator.Struct(req); err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Validation failed: "+err.Error(), http.StatusBadRequest))
		return
	}
	userID, err := extractUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, responses.ErrorResponse("Unauthorized: "+err.Error(), http.StatusUnauthorized))
		return
	}

	ctx := c.Request.Context()
	uow := h.unitOfWork.Begin(ctx)
	defer func() {
		if r := recover(); r != nil {
			uow.Rollback()
			panic(r)
		}
	}()

	if err := h.adminConfigService.UpdateConfigByKey(ctx, key, req.Value, uow, userID); err != nil {
		uow.Rollback()
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Failed to update config: "+err.Error(), http.StatusBadRequest))
		return
	}

	if err := uow.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse("Failed to commit transaction", http.StatusInternalServerError))
		return
	}

	c.JSON(http.StatusOK, responses.SuccessResponse("Config updated successfully", nil, nil))
}

// UpdateConfigs godoc
//
//	@Summary		Bulk update config values
//	@Description	Update multiple config values at once
//	@Tags			Admin Config
//	@Accept			json
//	@Produce		json
//	@Param			request	body		requests.BulkUpdateAdminConfigRequest	true	"Bulk Update Request"
//	@Success		200		{object}	responses.APIResponse					"Configs updated successfully"
//	@Failure		400		{object}	responses.APIResponse					"Bad Request"
//	@Failure		500		{object}	responses.APIResponse					"Internal server error"
//	@Security		BearerAuth
//	@Router			/api/v1/configs [put]
func (h *AdminConfigHandler) UpdateConfigs(c *gin.Context) {
	var req requests.BulkUpdateAdminConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid request body", http.StatusBadRequest))
		return
	}
	userID, err := extractUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, responses.ErrorResponse("Unauthorized: "+err.Error(), http.StatusUnauthorized))
		return
	}

	ctx := c.Request.Context()
	uow := h.unitOfWork.Begin(ctx)
	defer func() {
		if r := recover(); r != nil {
			uow.Rollback()
			panic(r)
		}
	}()

	if err := h.adminConfigService.UpdateConfigs(ctx, req, uow, userID); err != nil {
		uow.Rollback()
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Failed to update configs: "+err.Error(), http.StatusBadRequest))
		return
	}

	if err := uow.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse("Failed to commit transaction", http.StatusInternalServerError))
		return
	}

	c.JSON(http.StatusOK, responses.SuccessResponse("Configs updated successfully", nil, nil))
}

// GetTermOfService godoc
//
//	@Summary		Get term of service
//	@Description	Retrieve the term of service
//	@Tags			Admin Config
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	responses.APIResponse	"Term of service retrieved successfully"
//	@Failure		500	{object}	responses.APIResponse	"Internal server error"
//	@Router			/api/v1/configs/public/term-of-service [get]
func (h *AdminConfigHandler) GetTermOfService(c *gin.Context) {
	configResponse, err := h.adminConfigService.GetConfigByKey(c.Request.Context(), "term_of_service")
	if err != nil {
		response := responses.ErrorResponse("Failed to get term of service: "+err.Error(), http.StatusInternalServerError)
		c.JSON(http.StatusInternalServerError, response)
		return
	}

	resposne := responses.SuccessResponse("Successfully retrieved term of service", utils.IntPtr(http.StatusOK), configResponse.Value)
	c.JSON(http.StatusOK, resposne)
}

// GetPrivacyPolicy godoc
//
//	@Summary		Get privacy policy
//	@Description	Retrieve the privacy policy
//	@Tags			Admin Config
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	responses.APIResponse	"Privacy policy retrieved successfully"
//	@Failure		500	{object}	responses.APIResponse	"Internal server error"
//	@Router			/api/v1/configs/public/privacy-policy [get]
func (h *AdminConfigHandler) GetPrivacyPolicy(c *gin.Context) {
	configResponse, err := h.adminConfigService.GetConfigByKey(c.Request.Context(), "privacy_policy")
	if err != nil {
		response := responses.ErrorResponse("Failed to get privacy policy: "+err.Error(), http.StatusInternalServerError)
		c.JSON(http.StatusInternalServerError, response)
		return
	}

	resposne := responses.SuccessResponse("Successfully retrieved privacy policy", utils.IntPtr(http.StatusOK), configResponse.Value)
	c.JSON(http.StatusOK, resposne)
}

// GetConfigValueByKey godoc
//
//	@Summary		Get config value by key
//	@Description	Retrieve a config value by its key
//	@Tags			Admin Config
//	@Accept			json
//	@Produce		json
//	@Param			key	path		string					true	"Config Key"
//	@Success		200	{object}	responses.APIResponse	"Config value retrieved successfully"
//	@Failure		400	{object}	responses.APIResponse	"Bad Request"
//	@Failure		500	{object}	responses.APIResponse	"Internal server error"
//	@Security		BearerAuth
//	@Router			/api/v1/configs/public/{key}/value [get]
func (h *AdminConfigHandler) GetConfigValueByKey(c *gin.Context) {
	key := c.Param("key")
	if key == "" {
		response := responses.ErrorResponse("Config key is required", http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	value, err := h.adminConfigService.GetConfigValueByKey(c.Request.Context(), key)
	if err != nil {
		response := responses.ErrorResponse("Failed to get config value: "+err.Error(), http.StatusInternalServerError)
		c.JSON(http.StatusInternalServerError, response)
		return
	}

	resposne := responses.SuccessResponse("Successfully retrieved config value", utils.IntPtr(http.StatusOK), value)
	c.JSON(http.StatusOK, resposne)
}

// GetConfigByKey godoc
//
//	@Summary		Get config by key
//	@Description	Retrieve a config by its key
//	@Tags			Admin Config
//	@Accept			json
//	@Produce		json
//	@Param			key	path		string					true	"Config Key"
//	@Success		200	{object}	responses.APIResponse	"Config retrieved successfully"
//	@Failure		400	{object}	responses.APIResponse	"Bad Request"
//	@Failure		500	{object}	responses.APIResponse	"Internal server error"
//	@Security		BearerAuth
//	@Router			/api/v1/configs/{key} [get]
func (h *AdminConfigHandler) GetConfigByKey(c *gin.Context) {
	key := c.Param("key")
	if key == "" {
		response := responses.ErrorResponse("Config key is required", http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	configResponse, err := h.adminConfigService.GetConfigByKey(c.Request.Context(), key)
	if err != nil {
		response := responses.ErrorResponse("Failed to get config value: "+err.Error(), http.StatusInternalServerError)
		c.JSON(http.StatusInternalServerError, response)
		return
	}

	resposne := responses.SuccessResponse("Successfully retrieved config value", utils.IntPtr(http.StatusOK), configResponse)
	c.JSON(http.StatusOK, resposne)
}
