package handler

import (
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/application/interfaces/iservice"
	"core-backend/internal/domain/enum"
	"core-backend/pkg/utils"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"gorm.io/gorm"
)

type ChannelHandler struct {
	channelService iservice.ChannelService
	unitOfWork     irepository.UnitOfWork
	validator      *validator.Validate
}

func NewChannelHandler(service iservice.ChannelService, unitOfWork irepository.UnitOfWork) *ChannelHandler {
	validator := validator.New()
	return &ChannelHandler{
		channelService: service,
		unitOfWork:     unitOfWork,
		validator:      validator,
	}
}

// GetAllChannels godoc
//
//	@Summary		Get all channels
//	@Description	Retrieve a list of all channels
//	@Tags			Channels
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	responses.ChannelListResponse	"Channels retrieved successfully"
//	@Failure		500	{object}	responses.APIResponse			"Internal server error"
//	@Router			/api/v1/channels [get]
func (h *ChannelHandler) GetAllChannels(c *gin.Context) {
	isReturningTokenInfo := false
	userRole, err := extractUserRoles(c)
	if err == nil && *userRole == enum.UserRoleAdmin.String() {
		isReturningTokenInfo = true
	}

	// Get all channels
	channels, err := h.channelService.GetAllChannels(c.Request.Context(), isReturningTokenInfo)
	if err != nil {
		var response *responses.APIResponse
		var statusCode int
		switch err.Error() {
		case gorm.ErrRecordNotFound.Error():
			response = responses.ErrorResponse("No channels found", http.StatusNotFound)
			statusCode = http.StatusNotFound
		default:
			response = responses.ErrorResponse("Failed to get all channels: "+err.Error(), http.StatusInternalServerError)
			statusCode = http.StatusInternalServerError
		}
		c.JSON(statusCode, response)
		return
	}

	// Return response
	response := responses.NewPaginationResponse(
		"Channels retrieved successfully",
		http.StatusOK,
		channels,
		responses.Pagination{
			Page:  1,
			Limit: len(channels),
			Total: int64(len(channels)),
		},
	)
	c.JSON(http.StatusOK, response)
}

// GetChannelByID godoc
//
//	@Summary		Get channel by ID
//	@Description	Get channel by ID
//	@Tags			Channels
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string													true	"Channel ID"	example("123e4567-e89b-12d3-a456-426614174000")
//	@Success		200	{object}	responses.APIResponse{data=responses.ChannelResponse}	"Channel retrieved successfully"
//	@Failure		400	{object}	responses.APIResponse									"Invalid channel ID"
//	@Failure		500	{object}	responses.APIResponse									"Internal server error"
//	@Router			/api/v1/channels/{id} [get]
func (h *ChannelHandler) GetChannelByID(c *gin.Context) {
	channelID, err := extractParamID(c, "id")
	if err != nil {
		response := responses.ErrorResponse(err.Error(), http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	channel, err := h.channelService.GetChannelByID(c.Request.Context(), channelID)
	if err != nil {
		var response *responses.APIResponse
		var statusCode int
		switch err.Error() {
		case gorm.ErrRecordNotFound.Error():
			response = responses.ErrorResponse("Channel not found", http.StatusNotFound)
			statusCode = http.StatusNotFound
		default:
			response = responses.ErrorResponse("Failed to get channel by ID: "+err.Error(), http.StatusInternalServerError)
			statusCode = http.StatusInternalServerError
		}
		c.JSON(statusCode, response)
		return
	}

	if channel == nil {
		response := responses.ErrorResponse("Channel not found", http.StatusNotFound)
		c.JSON(http.StatusNotFound, response)
		return
	}

	response := responses.SuccessResponse("Channel retrieved successfully", utils.IntPtr(http.StatusOK), channel)
	c.JSON(http.StatusOK, response)
}

// CreateChannel godoc
//
//	@Summary		Create a channel
//	@Description	Create a new channel
//	@Tags			Channels
//	@Accept			json
//	@Produce		json
//	@Param			data	body		requests.CreateChannelRequest							true	"Channel to create"
//	@Success		201		{object}	responses.APIResponse{data=responses.ChannelResponse}	"Channel created successfully"
//	@Failure		400		{object}	responses.APIResponse									"Invalid request body"
//	@Failure		500		{object}	responses.APIResponse									"Failed to create channel"
//	@Security		BearerAuth
//	@Router			/api/v1/channels [post]
func (h *ChannelHandler) CreateChannel(c *gin.Context) {
	var req requests.CreateChannelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response := responses.ErrorResponse("Invalid request body: "+err.Error(), http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	if err := h.validator.Struct(&req); err != nil {
		response := processValidationError(err)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	uow := h.unitOfWork.Begin(c.Request.Context())

	channel, err := h.channelService.CreateChannel(c.Request.Context(), &req, uow)
	if err != nil {
		uow.Rollback()
		response := responses.ErrorResponse("Failed to create channel: "+err.Error(), http.StatusInternalServerError)
		c.JSON(http.StatusInternalServerError, response)
		return
	}

	uow.Commit()

	response := responses.SuccessResponse("Channel created successfully", utils.IntPtr(http.StatusCreated), channel)
	c.JSON(http.StatusCreated, response)
}

// UpdateChannel godoc
//
//	@Summary		Update a channel
//	@Description	Update an existing channel
//	@Tags			Channels
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string													true	"Channel ID"	example("123e4567-e89b-12d3-a456-426614174000")
//	@Param			data	body		requests.UpdateChannelRequest							true	"Channel to update"
//	@Success		200		{object}	responses.APIResponse{data=responses.ChannelResponse}	"Channel updated successfully"
//	@Failure		400		{object}	responses.APIResponse									"Invalid request body"
//	@Failure		404		{object}	responses.APIResponse									"Channel not found"
//	@Failure		500		{object}	responses.APIResponse									"Failed to update channel"
//	@Security		BearerAuth
//	@Router			/api/v1/channels/{id} [put]
func (h *ChannelHandler) UpdateChannel(c *gin.Context) {
	channelID, err := extractParamID(c, "id")
	if err != nil {
		response := responses.ErrorResponse(err.Error(), http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	var req requests.UpdateChannelRequest
	if err = c.ShouldBindJSON(&req); err != nil {
		response := responses.ErrorResponse("Invalid request body: "+err.Error(), http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	if err = h.validator.Struct(&req); err != nil {
		response := processValidationError(err)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	uow := h.unitOfWork.Begin(c.Request.Context())

	channel, err := h.channelService.UpdateChannel(c.Request.Context(), channelID, &req, uow)
	if err != nil {
		uow.Rollback()
		response := responses.ErrorResponse("Failed to update channel: "+err.Error(), http.StatusInternalServerError)
		c.JSON(http.StatusInternalServerError, response)
		return
	}

	uow.Commit()
	response := responses.SuccessResponse("Channel updated successfully", utils.IntPtr(http.StatusOK), channel)
	c.JSON(http.StatusOK, response)
}

// DeleteChannel godoc
//
//	@Summary		Delete a channel
//	@Description	Delete a channel by its ID
//	@Tags			Channels
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string					true	"Channel ID"	example("123e4567-e89b-12d3-a456-426614174000")
//	@Success		200	{object}	responses.APIResponse	"Channel deleted successfully"
//	@Failure		400	{object}	responses.APIResponse	"Invalid channel ID"
//	@Failure		404	{object}	responses.APIResponse	"Channel not found"
//	@Failure		500	{object}	responses.APIResponse	"Failed to delete channel"
//	@Security		BearerAuth
//	@Router			/api/v1/channels/{id} [delete]
func (h *ChannelHandler) DeleteChannel(c *gin.Context) {
	channelID, err := extractParamID(c, "id")
	if err != nil {
		response := responses.ErrorResponse(err.Error(), http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	uow := h.unitOfWork.Begin(c.Request.Context())

	err = h.channelService.DeleteChannel(c.Request.Context(), channelID, uow)
	if err != nil {
		uow.Rollback()
		response := responses.ErrorResponse("Failed to delete channel: "+err.Error(), http.StatusInternalServerError)
		c.JSON(http.StatusInternalServerError, response)
		return
	}

	uow.Commit()
	response := responses.SuccessResponse("Channel deleted successfully", utils.IntPtr(http.StatusOK), http.StatusNoContent)
	c.JSON(http.StatusOK, response)
}
