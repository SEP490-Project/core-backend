package handler

import (
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/iservice"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// DeviceTokenHandler handles device token-related HTTP requests
type DeviceTokenHandler struct {
	deviceTokenService iservice.DeviceTokenService
	validator          *validator.Validate
}

// NewDeviceTokenHandler creates a new device token handler
func NewDeviceTokenHandler(deviceTokenService iservice.DeviceTokenService) *DeviceTokenHandler {
	return &DeviceTokenHandler{
		deviceTokenService: deviceTokenService,
		validator:          validator.New(),
	}
}

// Register godoc
//
//	@Summary		Register a new device token
//	@Description	Register a new FCM device token for push notifications
//	@Tags			Device Tokens
//	@Accept			json
//	@Produce		json
//	@Param			request	body		requests.RegisterDeviceTokenRequest							true	"Device token registration data"
//	@Success		201		{object}	responses.APIResponse{data=responses.DeviceTokenResponse}	"Device token registered successfully"
//	@Failure		400		{object}	responses.APIResponse										"Invalid request data"
//	@Failure		401		{object}	responses.APIResponse										"Unauthorized"
//	@Failure		409		{object}	responses.APIResponse										"Token already registered to another user"
//	@Failure		500		{object}	responses.APIResponse										"Internal server error"
//	@Security		BearerAuth
//	@Router			/api/v1/device-tokens [post]
func (h *DeviceTokenHandler) Register(c *gin.Context) {
	var req requests.RegisterDeviceTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response := responses.ErrorResponse("Invalid request data", http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	if err := h.validator.Struct(req); err != nil {
		response := responses.ErrorResponse(err.Error(), http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}
	userID, err := extractUserID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Unauthorized: "+err.Error(), http.StatusUnauthorized))
		return
	}

	var sessionID uuid.UUID
	sessionID, err = extractSessionID(c)
	if err != nil {
		zap.L().Warn("Failed to extract session ID from context", zap.Error(err))
	}

	// Register device token
	if err = h.deviceTokenService.RegisterToken(c.Request.Context(), userID, &sessionID, req.Token, req.Platform); err != nil {
		zap.L().Error("Failed to register device token",
			zap.String("user_id", userID.String()),
			zap.Error(err))

		statusCode := http.StatusInternalServerError
		if err.Error() == "device token already registered to another user" {
			statusCode = http.StatusConflict
		}

		response := responses.ErrorResponse(err.Error(), statusCode)
		c.JSON(statusCode, response)
		return
	}

	// Get the registered token
	tokens, err := h.deviceTokenService.GetUserTokens(c.Request.Context(), userID)
	if err != nil || len(tokens) == 0 {
		statusCode := http.StatusCreated
		response := responses.SuccessResponse("Device token registered successfully", &statusCode, nil)
		c.JSON(http.StatusCreated, response)
		return
	}

	// Find the token we just registered
	var registeredToken *responses.DeviceTokenResponse
	for i := range tokens {
		if tokens[i].Token == req.Token {
			registeredToken = responses.ToDeviceTokenResponse(&tokens[i])
			break
		}
	}

	statusCode := http.StatusCreated
	response := responses.SuccessResponse("Device token registered successfully", &statusCode, registeredToken)
	c.JSON(http.StatusCreated, response)
}

// Update godoc
//
//	@Summary		Update a device token
//	@Description	Update an existing device token
//	@Tags			Device Tokens
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string														true	"Device Token ID"	format(uuid)
//	@Param			request	body		requests.UpdateDeviceTokenRequest							true	"Updated device token data"
//	@Success		200		{object}	responses.APIResponse{data=responses.DeviceTokenResponse}	"Device token updated successfully"
//	@Failure		400		{object}	responses.APIResponse										"Invalid request data"
//	@Failure		401		{object}	responses.APIResponse										"Unauthorized"
//	@Failure		404		{object}	responses.APIResponse										"Device token not found"
//	@Failure		500		{object}	responses.APIResponse										"Internal server error"
//	@Security		BearerAuth
//	@Router			/api/v1/device-tokens/{id} [put]
func (h *DeviceTokenHandler) Update(c *gin.Context) {
	tokenIDStr := c.Param("id")
	tokenID, err := uuid.Parse(tokenIDStr)
	if err != nil {
		response := responses.ErrorResponse("Invalid device token ID", http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	var req requests.UpdateDeviceTokenRequest
	if err = c.ShouldBindJSON(&req); err != nil {
		response := responses.ErrorResponse("Invalid request data", http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	if err = h.validator.Struct(req); err != nil {
		response := responses.ErrorResponse(err.Error(), http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	// Update device token
	err = h.deviceTokenService.UpdateToken(c.Request.Context(), tokenID, req.NewToken, req.Platform)
	if err != nil {
		zap.L().Error("Failed to update device token",
			zap.String("token_id", tokenID.String()),
			zap.Error(err))

		statusCode := http.StatusInternalServerError
		if err.Error() == "device token not found" {
			statusCode = http.StatusNotFound
		}

		response := responses.ErrorResponse(err.Error(), statusCode)
		c.JSON(statusCode, response)
		return
	}

	response := responses.SuccessResponse("Device token updated successfully", nil, nil)
	c.JSON(http.StatusOK, response)
}

// List godoc
//
//	@Summary		List device tokens
//	@Description	Get all device tokens for the authenticated user
//	@Tags			Device Tokens
//	@Produce		json
//	@Success		200	{object}	responses.APIResponse{data=responses.DeviceTokenListResponse}	"Device tokens retrieved successfully"
//	@Failure		401	{object}	responses.APIResponse											"Unauthorized"
//	@Failure		500	{object}	responses.APIResponse											"Internal server error"
//	@Security		BearerAuth
//	@Router			/api/v1/device-tokens [get]
func (h *DeviceTokenHandler) List(c *gin.Context) {
	// Get user ID from context
	userID, err := extractUserID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Unauthorized: "+err.Error(), http.StatusUnauthorized))
		return
	}

	// Get user tokens
	tokens, err := h.deviceTokenService.GetUserTokens(c.Request.Context(), userID)
	if err != nil {
		zap.L().Error("Failed to get device tokens",
			zap.String("user_id", userID.String()),
			zap.Error(err))
		response := responses.ErrorResponse("Failed to retrieve device tokens", http.StatusInternalServerError)
		c.JSON(http.StatusInternalServerError, response)
		return
	}

	listResponse := responses.ToDeviceTokenListResponse(tokens)
	response := responses.SuccessResponse("Device tokens retrieved successfully", nil, listResponse)
	c.JSON(http.StatusOK, response)
}

// Delete godoc
//
//	@Summary		Delete a device token
//	@Description	Delete a specific device token
//	@Tags			Device Tokens
//	@Produce		json
//	@Param			id	path		string					true	"Device Token ID"	format(uuid)
//	@Success		200	{object}	responses.APIResponse	"Device token deleted successfully"
//	@Failure		400	{object}	responses.APIResponse	"Invalid device token ID"
//	@Failure		401	{object}	responses.APIResponse	"Unauthorized"
//	@Failure		404	{object}	responses.APIResponse	"Device token not found"
//	@Failure		500	{object}	responses.APIResponse	"Internal server error"
//	@Security		BearerAuth
//	@Router			/api/v1/device-tokens/{id} [delete]
func (h *DeviceTokenHandler) Delete(c *gin.Context) {
	tokenIDStr := c.Param("id")
	tokenID, err := uuid.Parse(tokenIDStr)
	if err != nil {
		response := responses.ErrorResponse("Invalid device token ID", http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	// Delete device token
	err = h.deviceTokenService.DeleteToken(c.Request.Context(), tokenID)
	if err != nil {
		zap.L().Error("Failed to delete device token",
			zap.String("token_id", tokenID.String()),
			zap.Error(err))

		statusCode := http.StatusInternalServerError
		if err.Error() == "device token not found" {
			statusCode = http.StatusNotFound
		}

		response := responses.ErrorResponse(err.Error(), statusCode)
		c.JSON(statusCode, response)
		return
	}

	response := responses.SuccessResponse("Device token deleted successfully", nil, nil)
	c.JSON(http.StatusOK, response)
}

// DeleteAll godoc
//
//	@Summary		Delete all device tokens
//	@Description	Delete all device tokens for the authenticated user
//	@Tags			Device Tokens
//	@Produce		json
//	@Success		200	{object}	responses.APIResponse	"All device tokens deleted successfully"
//	@Failure		401	{object}	responses.APIResponse	"Unauthorized"
//	@Failure		500	{object}	responses.APIResponse	"Internal server error"
//	@Security		BearerAuth
//	@Router			/api/v1/device-tokens [delete]
func (h *DeviceTokenHandler) DeleteAll(c *gin.Context) {
	// Get user ID from context
	userID, err := extractUserID(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Unauthorized: "+err.Error(), http.StatusUnauthorized))
		return
	}

	// Delete all user tokens
	if err = h.deviceTokenService.DeleteAllTokens(c.Request.Context(), userID); err != nil {
		zap.L().Error("Failed to delete all device tokens",
			zap.String("user_id", userID.String()),
			zap.Error(err))
		response := responses.ErrorResponse("Failed to delete device tokens", http.StatusInternalServerError)
		c.JSON(http.StatusInternalServerError, response)
		return
	}

	response := responses.SuccessResponse("All device tokens deleted successfully", nil, nil)
	c.JSON(http.StatusOK, response)
}
