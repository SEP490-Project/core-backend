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

type AffiliateLinkHandler struct {
	affiliateLinkService iservice.AffiliateLinkService
	validator            *validator.Validate
}

func NewAffiliateLinkHandler(affiliateLinkService iservice.AffiliateLinkService) *AffiliateLinkHandler {
	return &AffiliateLinkHandler{
		affiliateLinkService: affiliateLinkService,
		validator:            validator.New(),
	}
}

// Create godoc
//
//	@Summary		Create or Get Affiliate Link
//	@Description	Create a new affiliate link or return existing one for the same context
//	@Tags			AffiliateLinks
//	@Accept			json
//	@Produce		json
//	@Param			request	body		requests.CreateAffiliateLinkRequest							true	"Affiliate link creation data"
//	@Success		201		{object}	responses.APIResponse{data=responses.AffiliateLinkResponse}	"Affiliate link created successfully"
//	@Success		200		{object}	responses.APIResponse{data=responses.AffiliateLinkResponse}	"Existing affiliate link returned"
//	@Failure		400		{object}	responses.APIResponse										"Invalid request"
//	@Failure		404		{object}	responses.APIResponse										"Contract/Content/Channel not found"
//	@Failure		500		{object}	responses.APIResponse										"Internal server error"
//	@Security		BearerAuth
//	@Router			/api/v1/affiliate-links [post]
func (h *AffiliateLinkHandler) Create(c *gin.Context) {
	var req requests.CreateAffiliateLinkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response := responses.ErrorResponse("Invalid request format: "+err.Error(), http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	// Validate request
	if err := h.validator.Struct(req); err != nil {
		response := responses.ErrorResponse("Validation failed: "+err.Error(), http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	// Create or get affiliate link
	link, err := h.affiliateLinkService.CreateOrGet(c.Request.Context(), &req)
	if err != nil {
		zap.L().Error("Failed to create affiliate link", zap.Error(err))
		response := responses.ErrorResponse("Failed to create affiliate link: "+err.Error(), http.StatusInternalServerError)
		c.JSON(http.StatusInternalServerError, response)
		return
	}

	statusCode := http.StatusCreated
	response := responses.SuccessResponse("Affiliate link created successfully", &statusCode, link)
	c.JSON(http.StatusCreated, response)
}

// GetByID godoc
//
//	@Summary		Get Affiliate Link by ID
//	@Description	Retrieve a specific affiliate link by its ID
//	@Tags			AffiliateLinks
//	@Produce		json
//	@Param			id	path		string														true	"Affiliate Link ID (UUID)"
//	@Success		200	{object}	responses.APIResponse{data=responses.AffiliateLinkResponse}	"Affiliate link retrieved successfully"
//	@Failure		404	{object}	responses.APIResponse										"Affiliate link not found"
//	@Failure		500	{object}	responses.APIResponse										"Internal server error"
//	@Security		BearerAuth
//	@Router			/api/v1/affiliate-links/{id} [get]
func (h *AffiliateLinkHandler) GetByID(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		response := responses.ErrorResponse("Invalid affiliate link ID format", http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	link, err := h.affiliateLinkService.GetByID(c.Request.Context(), id, []string{"Contract", "Content", "Channel"})
	if err != nil {
		response := responses.ErrorResponse("Affiliate link not found", http.StatusNotFound)
		c.JSON(http.StatusNotFound, response)
		return
	}

	response := responses.SuccessResponse("Affiliate link retrieved successfully", nil, link)
	c.JSON(http.StatusOK, response)
}

// List godoc
//
//	@Summary		List Affiliate Links
//	@Description	Retrieve a paginated list of affiliate links with optional filtering
//	@Tags			AffiliateLinks
//	@Produce		json
//	@Param			contract_id	query		string															false	"Filter by Contract ID"
//	@Param			content_id	query		string															false	"Filter by Content ID"
//	@Param			channel_id	query		string															false	"Filter by Channel ID"
//	@Param			status		query		string															false	"Filter by status (active, inactive, expired)"
//	@Param			page_size	query		int																false	"Page size (default: 20, max: 100)"
//	@Param			page_number	query		int																false	"Page number (default: 1)"
//	@Success		200			{object}	responses.APIResponse{data=responses.AffiliateLinkListResponse}	"Affiliate links retrieved successfully"
//	@Failure		400			{object}	responses.APIResponse											"Invalid request parameters"
//	@Failure		500			{object}	responses.APIResponse											"Internal server error"
//	@Security		BearerAuth
//	@Router			/api/v1/affiliate-links [get]
func (h *AffiliateLinkHandler) List(c *gin.Context) {
	var req requests.GetAffiliateLinkRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response := responses.ErrorResponse("Invalid query parameters: "+err.Error(), http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	// Validate request
	if err := h.validator.Struct(req); err != nil {
		response := responses.ErrorResponse("Validation failed: "+err.Error(), http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	links, err := h.affiliateLinkService.List(c.Request.Context(), &req)
	if err != nil {
		zap.L().Error("Failed to list affiliate links", zap.Error(err))
		response := responses.ErrorResponse("Failed to list affiliate links: "+err.Error(), http.StatusInternalServerError)
		c.JSON(http.StatusInternalServerError, response)
		return
	}

	response := responses.NewPaginationResponse(
		"Affiliate links retrieved successfully",
		http.StatusOK,
		links.Links,
		links.Pagination,
	)
	c.JSON(http.StatusOK, response)
}

// Update godoc
//
//	@Summary		Update Affiliate Link
//	@Description	Update an affiliate link's status or tracking URL
//	@Tags			AffiliateLinks
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string														true	"Affiliate Link ID (UUID)"
//	@Param			request	body		requests.UpdateAffiliateLinkRequest							true	"Update data"
//	@Success		200		{object}	responses.APIResponse{data=responses.AffiliateLinkResponse}	"Affiliate link updated successfully"
//	@Failure		400		{object}	responses.APIResponse										"Invalid request"
//	@Failure		404		{object}	responses.APIResponse										"Affiliate link not found"
//	@Failure		500		{object}	responses.APIResponse										"Internal server error"
//	@Security		BearerAuth
//	@Router			/api/v1/affiliate-links/{id} [put]
func (h *AffiliateLinkHandler) Update(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		response := responses.ErrorResponse("Invalid affiliate link ID format", http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	var req requests.UpdateAffiliateLinkRequest
	if err = c.ShouldBindJSON(&req); err != nil {
		response := responses.ErrorResponse("Invalid request format: "+err.Error(), http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	// Validate request
	if err = h.validator.Struct(req); err != nil {
		response := responses.ErrorResponse("Validation failed: "+err.Error(), http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	link, err := h.affiliateLinkService.Update(c.Request.Context(), id, &req)
	if err != nil {
		zap.L().Error("Failed to update affiliate link", zap.Error(err))
		response := responses.ErrorResponse("Failed to update affiliate link: "+err.Error(), http.StatusInternalServerError)
		c.JSON(http.StatusInternalServerError, response)
		return
	}

	response := responses.SuccessResponse("Affiliate link updated successfully", nil, link)
	c.JSON(http.StatusOK, response)
}

// Delete godoc
//
//	@Summary		Delete Affiliate Link
//	@Description	Soft-delete an affiliate link
//	@Tags			AffiliateLinks
//	@Produce		json
//	@Param			id	path		string					true	"Affiliate Link ID (UUID)"
//	@Success		200	{object}	responses.APIResponse	"Affiliate link deleted successfully"
//	@Failure		404	{object}	responses.APIResponse	"Affiliate link not found"
//	@Failure		500	{object}	responses.APIResponse	"Internal server error"
//	@Security		BearerAuth
//	@Router			/api/v1/affiliate-links/{id} [delete]
func (h *AffiliateLinkHandler) Delete(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		response := responses.ErrorResponse("Invalid affiliate link ID format", http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	if err := h.affiliateLinkService.Delete(c.Request.Context(), id); err != nil {
		zap.L().Error("Failed to delete affiliate link", zap.Error(err))
		response := responses.ErrorResponse("Failed to delete affiliate link: "+err.Error(), http.StatusInternalServerError)
		c.JSON(http.StatusInternalServerError, response)
		return
	}

	response := responses.SuccessResponse("Affiliate link deleted successfully", nil, nil)
	c.JSON(http.StatusOK, response)
}
