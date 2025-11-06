package handler

import (
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/application/interfaces/iservice"
	"core-backend/internal/domain/enum"
	"core-backend/pkg/utils"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

type CampaignHandler struct {
	campaignService      iservice.CampaignService
	stateTransferService iservice.StateTransferService
	uow                  irepository.UnitOfWork
	validartor           *validator.Validate
}

func NewCampaignHandler(
	campaignService iservice.CampaignService,
	stateTransferService iservice.StateTransferService,
	uow irepository.UnitOfWork,
) *CampaignHandler {
	validator := validator.New()
	validator.RegisterStructValidation(requests.ValidateCreateCampaignRequest, requests.CreateCampaignRequest{})

	return &CampaignHandler{
		campaignService:      campaignService,
		stateTransferService: stateTransferService,
		uow:                  uow,
		validartor:           validator,
	}
}

// CreateCampaignFromContract godoc
//
//	@Summary		Create Campaign from Contract
//	@Description	Create a new campaign based on the provided contract details.
//	@Tags			Campaigns
//	@Accept			json
//	@Produce		json
//	@Param			data	body		requests.CreateCampaignRequest									true	"Campaign creation data"
//	@Success		201		{object}	responses.APIResponse{data=responses.CampaignDetailsResponse}	"Campaign created successfully"
//	@Failure		400		{object}	responses.APIResponse											"Invalid request or validation error"
//	@Failure		401		{object}	responses.APIResponse											"Unauthorized"
//	@Failure		403		{object}	responses.APIResponse											"Forbidden"
//	@Failure		500		{object}	responses.APIResponse											"Internal server error"
//	@Security		BearerAuth
//	@Router			/api/v1/campaigns [post]
func (h *CampaignHandler) CreateCampaignFromContract(c *gin.Context) {
	userID, err := extractUserID(c)
	if err != nil {
		responses := responses.ErrorResponse("Unauthorized: "+err.Error(), http.StatusUnauthorized)
		c.JSON(http.StatusUnauthorized, responses)
		return
	}
	var request *requests.CreateCampaignRequest
	if err = c.ShouldBindJSON(&request); err != nil {
		responses := responses.ErrorResponse("Invalid request format: "+err.Error(), http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, responses)
		return
	}
	if err = h.validartor.Struct(request); err != nil {
		responses := responses.ErrorResponse("Validation error: "+err.Error(), http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, responses)
		return
	}
	if request.ContractID == "" {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("ContractID is required", http.StatusBadRequest))
		return
	}

	uow := h.uow.Begin(c.Request.Context())
	var campaignResponse *responses.CampaignDetailsResponse
	campaignResponse, err = h.campaignService.CreateCampaignFromContract(c.Request.Context(), userID, request, uow)
	if err != nil {
		uow.Rollback()
		var responsesDtos *responses.APIResponse
		switch err.Error() {
		case fmt.Sprintf("Campaign already exists for contract %s", request.ContractID):
			responsesDtos = responses.ErrorResponse(err.Error(), http.StatusBadRequest)
			c.JSON(http.StatusBadRequest, responsesDtos)
		default:
			responses := responses.ErrorResponse("Failed to create campaign: "+err.Error(), http.StatusInternalServerError)
			c.JSON(http.StatusInternalServerError, responses)
		}
		return
	}

	uow.Commit()
	responses := responses.SuccessResponse("Campaign created successfully", utils.IntPtr(http.StatusCreated), campaignResponse)
	c.JSON(http.StatusCreated, responses)
}

// CreateInternalCampaign godoc
//
//	@Summary		Create Internal Campaign
//	@Description	Create a new internal campaign not linked to any contract.
//	@Tags			Campaigns
//	@Accept			json
//	@Produce		json
//	@Param			data	body		requests.CreateCampaignRequest									true	"Internal campaign creation data"
//	@Success		201		{object}	responses.APIResponse{data=responses.CampaignDetailsResponse}	"Internal campaign created successfully"
//	@Failure		400		{object}	responses.APIResponse											"Invalid request or validation error"
//	@Failure		401		{object}	responses.APIResponse											"Unauthorized"
//	@Failure		403		{object}	responses.APIResponse											"Forbidden"
//	@Failure		500		{object}	responses.APIResponse											"Internal server error"
//	@Security		BearerAuth
//	@Router			/api/v1/campaigns/internal [post]
func (h *CampaignHandler) CreateInternalCampaign(c *gin.Context) {
	userID, err := extractUserID(c)
	if err != nil {
		responses := responses.ErrorResponse("Unauthorized: "+err.Error(), http.StatusUnauthorized)
		c.JSON(http.StatusUnauthorized, responses)
		return
	}
	var request *requests.CreateCampaignRequest
	if err = c.ShouldBindJSON(&request); err != nil {
		responses := responses.ErrorResponse("Invalid request format: "+err.Error(), http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, responses)
		return
	}
	if err = h.validartor.Struct(request); err != nil {
		responses := responses.ErrorResponse("Validation error: "+err.Error(), http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, responses)
		return
	}
	uow := h.uow.Begin(c.Request.Context())
	var campaignResponse *responses.CampaignDetailsResponse
	campaignResponse, err = h.campaignService.CreateInternalCampaign(c.Request.Context(), uow, request, userID)
	if err != nil {
		uow.Rollback()
		response := responses.ErrorResponse("Failed to create campaign: "+err.Error(), http.StatusInternalServerError)
		c.JSON(http.StatusInternalServerError, response)
		return
	}

	uow.Commit()
	responses := responses.SuccessResponse("Campaign created successfully", utils.IntPtr(http.StatusCreated), campaignResponse)
	c.JSON(http.StatusCreated, responses)
}

// GetCampaignInfoByID godoc
//
//	@Summary		Get Campaign Info by ID
//	@Description	Get basic information about a campaign by its ID.
//	@Tags			Campaigns
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string														true	"Campaign ID"	format(uuid)
//	@Success		200	{object}	responses.APIResponse{data=responses.CampaignInfoResponse}	"Campaign info retrieved successfully"
//	@Failure		400	{object}	responses.APIResponse										"Invalid campaign ID"
//	@Failure		404	{object}	responses.APIResponse										"Campaign not found"
//	@Failure		500	{object}	responses.APIResponse										"Internal server error"
//	@Security		BearerAuth
//	@Router			/api/v1/campaigns/id/{id} [get]
func (h *CampaignHandler) GetCampaignInfoByID(c *gin.Context) {
	campaignID, err := extractParamID(c, "id")
	if err != nil {
		responses := responses.ErrorResponse("Invalid campaign ID: "+err.Error(), http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, responses)
		return
	}

	var campaignInfo *responses.CampaignInfoResponse
	if campaignInfo, err = h.campaignService.GetCampaignInfoByID(c.Request.Context(), campaignID); err != nil {
		responses := responses.ErrorResponse("Failed to get campaign info: "+err.Error(), http.StatusInternalServerError)
		c.JSON(http.StatusInternalServerError, responses)
		return
	}

	responses := responses.SuccessResponse("Campaign info retrieved successfully", utils.IntPtr(http.StatusOK), campaignInfo)
	c.JSON(http.StatusOK, responses)
}

// GetCampaignDetailsByID godoc
//
//	@Summary		Get Campaign Details by ID
//	@Description	Get detailed information about a campaign by its ID including milestones and number of tasks.
//	@Tags			Campaigns
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string															true	"Campaign ID"	format(uuid)
//	@Success		200	{object}	responses.APIResponse{data=responses.CampaignDetailsResponse}	"Campaign details retrieved successfully"
//	@Failure		400	{object}	responses.APIResponse											"Invalid campaign ID"
//	@Failure		404	{object}	responses.APIResponse											"Campaign not found"
//	@Failure		500	{object}	responses.APIResponse											"Internal server error"
//	@Security		BearerAuth
//	@Router			/api/v1/campaigns/id/{id}/details [get]
func (h *CampaignHandler) GetCampaignDetailsByID(c *gin.Context) {
	campaignID, err := extractParamID(c, "id")
	if err != nil {
		responses := responses.ErrorResponse("Invalid campaign ID: "+err.Error(), http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, responses)
		return
	}

	var campaignDetails *responses.CampaignDetailsResponse
	if campaignDetails, err = h.campaignService.GetCampaignDetailsByID(c.Request.Context(), campaignID); err != nil {
		responses := responses.ErrorResponse("Failed to get campaign details: "+err.Error(), http.StatusInternalServerError)
		c.JSON(http.StatusInternalServerError, responses)
		return
	}

	responses := responses.SuccessResponse("Campaign details retrieved successfully", utils.IntPtr(http.StatusOK), campaignDetails)
	c.JSON(http.StatusOK, responses)
}

// GetCampaignsByFilter godoc
//
//	@Summary		Get Campaigns List by filter
//	@Description	Get paginated list of campaigns with optional filters
//	@Tags			Campaigns
//	@Accept			json
//	@Produce		json
//	@Param			page	query		int											false	"Page number"		default(1)
//	@Param			limit	query		int											false	"Items per page"	default(10)
//	@Param			keyword	query		string										false	"Search keyword for campaign name"
//	@Param			status	query		string										false	"Filter by campaign status"	Enums(RUNNING, COMPLETED, CANCELLED)
//	@Param			type	query		string										false	"Filter by campaign type"	Enums(ADVERTISING, AFFILIATE, BRAND_AMBASSADOR, CO_PRODUCING)
//	@Success		200		{object}	responses.CampaignInfoPaginationResponse	"Campaigns retrieved successfully"
//	@Failure		400		{object}	responses.APIResponse						"Invalid query parameters"
//	@Failure		500		{object}	responses.APIResponse						"Internal server error"
//	@Security		BearerAuth
//	@Router			/api/v1/campaigns [get]
func (h *CampaignHandler) GetCampaignsByFilter(c *gin.Context) {
	var filterRequest *requests.CampaignFilterRequest
	if err := c.ShouldBindQuery(&filterRequest); err != nil {
		responses := responses.ErrorResponse("Invalid filter request: "+err.Error(), http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, responses)
		return
	}
	if err := h.validartor.Struct(filterRequest); err != nil {
		responses := responses.ErrorResponse("Validation error: "+err.Error(), http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, responses)
		return
	}

	campaigns, totalCount, err := h.campaignService.GetCampaignsByFilter(c.Request.Context(), filterRequest)
	if err != nil {
		responses := responses.ErrorResponse("Failed to get campaigns: "+err.Error(), http.StatusInternalServerError)
		c.JSON(http.StatusInternalServerError, responses)
		return
	}

	paginationResponse := responses.NewPaginationResponse(
		"Campaigns retrieved successfully",
		http.StatusOK,
		campaigns,
		responses.Pagination{
			Page:  filterRequest.Page,
			Limit: filterRequest.Limit,
			Total: totalCount,
		},
	)
	c.JSON(http.StatusOK, paginationResponse)
}

// DeleteCampaign godoc
//
//	@Summary		Delete Campaign
//	@Description	Soft delete a campaign by ID
//	@Tags			Campaigns
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string					true	"Campaign ID"	format(uuid)
//	@Success		200	{object}	responses.APIResponse	"Campaign deleted successfully"
//	@Failure		400	{object}	responses.APIResponse	"Invalid campaign ID"
//	@Failure		404	{object}	responses.APIResponse	"Campaign not found"
//	@Failure		500	{object}	responses.APIResponse	"Internal server error"
//	@Security		BearerAuth
//	@Router			/api/v1/campaigns/id/{id} [delete]
func (h *CampaignHandler) DeleteCampaign(c *gin.Context) {
	campaignID, err := extractParamID(c, "id")
	if err != nil {
		responses := responses.ErrorResponse("Invalid campaign ID: "+err.Error(), http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, responses)
		return
	}

	if err = h.campaignService.DeleteCampaign(c.Request.Context(), campaignID); err != nil {
		var responsesDtos *responses.APIResponse
		switch err.Error() {
		case fmt.Sprintf("campaign with ID %s not found", campaignID.String()):
			responsesDtos = responses.ErrorResponse(err.Error(), http.StatusNotFound)
			c.JSON(http.StatusNotFound, responsesDtos)
		default:
			responsesDtos = responses.ErrorResponse("Failed to delete campaign: "+err.Error(), http.StatusInternalServerError)
			c.JSON(http.StatusInternalServerError, responsesDtos)
		}
		return
	}

	responses := responses.SuccessResponse("Campaign deleted successfully", utils.IntPtr(http.StatusOK), nil)
	c.JSON(http.StatusOK, responses)
}

// GetCampaignInfoByContractID godoc
//
//	@Summary		Get Campaign Info by Contract ID
//	@Description	Get basic information about a campaign by its contract ID.
//	@Tags			Campaigns
//	@Accept			json
//	@Produce		json
//	@Param			contract_id	path		string														true	"Contract ID"	format(uuid)
//	@Success		200			{object}	responses.APIResponse{data=responses.CampaignInfoResponse}	"Campaign info retrieved successfully"
//	@Failure		400			{object}	responses.APIResponse										"Invalid contract ID"
//	@Failure		404			{object}	responses.APIResponse										"Campaign not found"
//	@Failure		500			{object}	responses.APIResponse										"Internal server error"
//	@Security		BearerAuth
//	@Router			/api/v1/campaigns/contract/{contract_id} [get]
func (h *CampaignHandler) GetCampaignInfoByContractID(c *gin.Context) {
	contractID, err := extractParamID(c, "contract_id")
	if err != nil {
		responses := responses.ErrorResponse("Invalid contract ID: "+err.Error(), http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, responses)
		return
	}

	var campaignInfo *responses.CampaignInfoResponse
	if campaignInfo, err = h.campaignService.GetCampaignInfoByContractID(c.Request.Context(), contractID); err != nil {
		var responsesDTO *responses.APIResponse
		switch err.Error() {
		case "no campaign found for the given contract ID":
			responsesDTO = responses.ErrorResponse(err.Error(), http.StatusNotFound)
			c.JSON(http.StatusNotFound, responsesDTO)
		default:
			responsesDTO = responses.ErrorResponse("Failed to get campaign info: "+err.Error(), http.StatusInternalServerError)
			c.JSON(http.StatusInternalServerError, responsesDTO)
		}
		return
	}

	resposnes := responses.SuccessResponse("Campaign info retrieved successfully", utils.IntPtr(http.StatusOK), campaignInfo)
	c.JSON(http.StatusOK, resposnes)
}

// GetCampaignDetailsByContractID godoc
//
//	@Summary		Get Campaign Details by Contract ID
//	@Description	Get detailed information about a campaign by its contract ID including milestones and number of tasks.
//	@Tags			Campaigns
//	@Accept			json
//	@Produce		json
//	@Param			contract_id	path		string															true	"Contract ID"	format(uuid)
//	@Success		200			{object}	responses.APIResponse{data=responses.CampaignDetailsResponse}	"Campaign details retrieved successfully"
//	@Failure		400			{object}	responses.APIResponse											"Invalid contract ID"
//	@Failure		404			{object}	responses.APIResponse											"Campaign not found"
//	@Failure		500			{object}	responses.APIResponse											"Internal server error"
//	@Security		BearerAuth
//	@Router			/api/v1/campaigns/contract/{contract_id}/details [get]
func (h *CampaignHandler) GetCampaignDetailsByContractID(c *gin.Context) {
	contractID, err := extractParamID(c, "contract_id")
	if err != nil {
		responses := responses.ErrorResponse("Invalid contract ID: "+err.Error(), http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, responses)
		return
	}

	var campaignDetails *responses.CampaignDetailsResponse
	if campaignDetails, err = h.campaignService.GetCampaignDetailsByContractID(c.Request.Context(), contractID); err != nil {
		var responsesDTO *responses.APIResponse
		switch err.Error() {
		case "no campaign found for the given contract ID":
			responsesDTO = responses.ErrorResponse(err.Error(), http.StatusNotFound)
			c.JSON(http.StatusNotFound, responsesDTO)
		default:
			responsesDTO = responses.ErrorResponse("Failed to get campaign info: "+err.Error(), http.StatusInternalServerError)
			c.JSON(http.StatusInternalServerError, responsesDTO)
		}
		return
	}

	resposnes := responses.SuccessResponse("Campaign details retrieved successfully", utils.IntPtr(http.StatusOK), campaignDetails)
	c.JSON(http.StatusOK, resposnes)
}

// GetCampaignsInfoByBrandID godoc
//
//	@Summary		Get Campaigns Info by Brand ID
//	@Description	Get paginated list of campaigns with optional filters
//	@Tags			Campaigns
//	@Accept			json
//	@Produce		json
//	@Param			brand_id	path		string										true	"Brand ID"	format(uuid)
//	@Success		200			{object}	responses.CampaignInfoPaginationResponse	"Campaigns retrieved successfully"
//	@Failure		400			{object}	responses.APIResponse						"Invalid query parameters"
//	@Failure		500			{object}	responses.APIResponse						"Internal server error"
//	@Security		BearerAuth
//	@Router			/api/v1/campaigns/brand/{brand_id} [get]
func (h *CampaignHandler) GetCampaignsInfoByBrandID(c *gin.Context) {
	brandID, err := extractParamID(c, "brand_id")
	if err != nil {
		responses := responses.ErrorResponse("Invalid brand ID: "+err.Error(), http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, responses)
		return
	}

	var campaignsInfo []*responses.CampaignInfoResponse
	var totalCount int64
	if campaignsInfo, totalCount, err = h.campaignService.GetCampaignsInfoByBrandID(c.Request.Context(), brandID); err != nil {
		responses := responses.ErrorResponse("Failed to get campaigns info: "+err.Error(), http.StatusInternalServerError)
		c.JSON(http.StatusInternalServerError, responses)
		return
	}

	responses := responses.NewPaginationResponse(
		"Campaigns retrieved successfully",
		http.StatusOK,
		campaignsInfo,
		responses.Pagination{
			Page:  1,
			Limit: int(totalCount),
			Total: totalCount,
		},
	)
	c.JSON(http.StatusOK, responses)
}

// GetCampaignsByBrandProfile godoc
//
//	@Summary		Get Campaigns Info by authenticated brand user
//	@Description	Get a list of campaigns associated with the authenticated brand user.
//	@Tags			Campaigns
//	@Accept			json
//	@Produce		json
//	@Param			page	query		int											false	"Page number"		default(1)
//	@Param			limit	query		int											false	"Items per page"	default(10)
//	@Param			keyword	query		string										false	"Search keywords for campaign name"
//	@Param			status	query		string										false	"Filter by campaign status"	Enums(RUNNING, COMPLETED, CANCELLED)
//	@Param			type	query		string										false	"Filter by campaign type"	Enums(ADVERTISING, AFFILIATE, BRAND_AMBASSADOR, CO_PRODUCING)
//	@Success		200		{object}	responses.CampaignInfoPaginationResponse	"Campaigns retrieved successfully"
//	@Failure		400		{object}	responses.APIResponse						"Invalid query parameters"
//	@Failure		401		{object}	responses.APIResponse						"Unauthorized"
//	@Failure		403		{object}	responses.APIResponse						"Forbidden"
//	@Failure		500		{object}	responses.APIResponse						"Internal server error"
//	@Security		BearerAuth
//	@Router			/api/v1/campaigns/brand/profile [get]
func (h *CampaignHandler) GetCampaignsByBrandProfile(c *gin.Context) {
	userID, err := extractUserID(c)
	var filterRequest *requests.CampaignFilterRequest
	if err = c.ShouldBindQuery(&filterRequest); err != nil {
		responses := responses.ErrorResponse("Invalid filter request: "+err.Error(), http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, responses)
		return
	}
	if err != nil {
		responses := responses.ErrorResponse("Unauthorized: "+err.Error(), http.StatusUnauthorized)
		c.JSON(http.StatusUnauthorized, responses)
		return
	}

	if err = h.validartor.Struct(filterRequest); err != nil {
		responses := responses.ErrorResponse("Validation error: "+err.Error(), http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, responses)
		return
	}

	campaignsInfo, totalCount, err := h.campaignService.GetCampaignsInfoByUserID(c.Request.Context(), userID, filterRequest)
	if err != nil {
		responses := responses.ErrorResponse("Failed to get campaigns info: "+err.Error(), http.StatusInternalServerError)
		c.JSON(http.StatusInternalServerError, responses)
		return
	}

	paginationResponse := responses.NewPaginationResponse(
		"Campaigns retrieved successfully",
		http.StatusOK,
		campaignsInfo,
		responses.Pagination{
			Page:  filterRequest.Page,
			Limit: int(totalCount),
			Total: totalCount,
		},
	)
	c.JSON(http.StatusOK, paginationResponse)
}

// SuggestCampaign godoc
//
//	@Summary		Suggest Campaign from Contract
//	@Description	Generate campaign suggestions based on contract deliverables. Only ACTIVE contracts can be used for suggestions.
//	@Tags			Campaigns
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string																true	"Campaign ID"	format(uuid)
//	@Success		200	{object}	responses.APIResponse{data=responses.CampaignSuggestionResponse}	"Campaign suggestion generated successfully"
//	@Failure		400	{object}	responses.APIResponse												"Invalid request or validation error"
//	@Failure		401	{object}	responses.APIResponse												"Unauthorized"
//	@Failure		403	{object}	responses.APIResponse												"Forbidden - Requires ADMIN or SALES_STAFF role"
//	@Failure		404	{object}	responses.APIResponse												"Contract not found"
//	@Failure		409	{object}	responses.APIResponse												"Contract is not ACTIVE or has no deliverables"
//	@Failure		500	{object}	responses.APIResponse												"Internal server error"
//	@Security		BearerAuth
//	@Router			/api/v1/campaigns/{id}/suggest [get]
func (h *CampaignHandler) SuggestCampaign(c *gin.Context) {
	campaignID, err := extractParamID(c, "id")
	if err != nil {
		response := responses.ErrorResponse("Invalid campaign ID: "+err.Error(), http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	// Call service to generate campaign suggestion
	suggestion, err := h.campaignService.SuggestCampaignFromContract(c.Request.Context(), campaignID)
	if err != nil {
		// Determine appropriate status code based on error message
		statusCode := http.StatusInternalServerError
		if err.Error() == "contract not found" {
			statusCode = http.StatusNotFound
		} else if err.Error() == "only ACTIVE contracts can be used for campaign suggestions" ||
			err.Error() == "contract has no deliverables defined in scope of work" ||
			err.Error() == "unsupported contract type" {
			statusCode = http.StatusConflict
		}

		response := responses.ErrorResponse("Failed to suggest campaign: "+err.Error(), statusCode)
		c.JSON(statusCode, response)
		return
	}

	// Return success response
	response := responses.SuccessResponse("Campaign suggestion generated successfully", nil, suggestion)
	c.JSON(http.StatusOK, response)
}

// ApproveCampaign godoc
//
//	@Summary		Approve Campaign
//	@Description	Approve a campaign by ID
//	@Tags			Campaigns
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string					true	"Campaign ID"	format(uuid)
//	@Success		200	{object}	responses.APIResponse	"Campaign approved successfully"
//	@Failure		400	{object}	responses.APIResponse	"Invalid campaign ID"
//	@Failure		401	{object}	responses.APIResponse	"Unauthorized"
//	@Failure		403	{object}	responses.APIResponse	"Forbidden"
//	@Failure		500	{object}	responses.APIResponse	"Internal server error"
//	@Security		BearerAuth
//	@Router			/api/v1/campaigns/{id}/approve [patch]
func (h *CampaignHandler) ApproveCampaign(c *gin.Context) {
	campaignID, err := extractParamID(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid campaign ID: "+err.Error(), http.StatusBadRequest))
		return
	}
	userID, err := extractUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, responses.ErrorResponse("Unauthorized: "+err.Error(), http.StatusUnauthorized))
		return
	}
	userRole, err := extractUserRoles(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, responses.ErrorResponse("Unauthorized: "+err.Error(), http.StatusUnauthorized))
		return
	}
	var campaignResponse *responses.CampaignInfoResponse
	campaignResponse, err = h.campaignService.GetCampaignInfoByID(c.Request.Context(), campaignID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse("Failed to approve campaign: "+err.Error(), http.StatusInternalServerError))
		return
	}
	if campaignResponse.Status == enum.CampaignDraft.String() &&
		!utils.ContainsSlice([]string{enum.UserRoleAdmin.String(), enum.UserRoleBrandPartner.String()}, *userRole) {
		c.JSON(http.StatusForbidden, responses.ErrorResponse("Forbidden: only ADMIN or BRAND_PARTNER can approve DRAFT campaigns", http.StatusForbidden))
		return
	}

	uow := h.uow.Begin(c.Request.Context())
	defer func() {
		if r := recover(); r != nil {
			uow.Rollback()
			panic(r)
		}
	}()

	if err = h.stateTransferService.MoveCampaignToState(
		c.Request.Context(), uow, campaignID, enum.CampaignRunning, userID,
	); err != nil {
		uow.Rollback()
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse("Failed to approve campaign: "+err.Error(), http.StatusInternalServerError))
		return
	}

	if err = uow.Commit(); err != nil {
		uow.Rollback()
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse("Failed to approve campaign: "+err.Error(), http.StatusInternalServerError))
		return
	}
	c.JSON(http.StatusOK,
		responses.SuccessResponse("Campaign approved successfully", utils.PtrOrNil(http.StatusOK), nil))
}

// RejectCampaign godoc
//
//	@Summary		Reject Campaign
//	@Description	Reject a campaign by providing a reason.
//	@Tags			Campaigns
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string					true	"Campaign ID"	format(uuid)
//	@Param			reason	query		string					false	"Reason for rejecting the campaign"
//	@Success		200		{object}	responses.APIResponse	"Campaign rejected successfully"
//	@Failure		400		{object}	responses.APIResponse	"Invalid campaign ID"
//	@Failure		401		{object}	responses.APIResponse	"Unauthorized"
//	@Failure		500		{object}	responses.APIResponse	"Internal server error"
//	@Security		BearerAuth
//	@Router			/api/v1/campaigns/{id}/reject [patch]
func (h *CampaignHandler) RejectCampaign(c *gin.Context) {
	campaignID, err := extractParamID(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid campaign ID: "+err.Error(), http.StatusBadRequest))
		return
	}
	rejectReason := c.Query("reason")
	userID, err := extractUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, responses.ErrorResponse("Unauthorized: "+err.Error(), http.StatusUnauthorized))
		return
	}

	uow := h.uow.Begin(c.Request.Context())
	defer func() {
		if r := recover(); r != nil {
			uow.Rollback()
			panic(r)
		}
	}()

	if err = h.campaignService.SetRejectReason(c.Request.Context(), uow, campaignID, rejectReason, userID); err != nil {
		uow.Rollback()
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse("Failed to reject campaign: "+err.Error(), http.StatusInternalServerError))
		return
	}

	if err = uow.Commit(); err != nil {
		uow.Rollback()
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse("Failed to reject campaign: "+err.Error(), http.StatusInternalServerError))
		return
	}
	c.JSON(http.StatusOK,
		responses.SuccessResponse("Campaign rejected successfully", utils.PtrOrNil(http.StatusOK), nil))
}

// UpdateCampaign godoc
//
//	@Summary		Update Campaign
//	@Description	Update campaign details by ID.
//	@Tags			Campaigns
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string															true	"Campaign ID"	format(uuid)
//	@Param			data	body		requests.UpdateCampaignRequest									true	"Campaign update data"
//	@Success		200		{object}	responses.APIResponse{data=responses.CampaignDetailsResponse}	"Campaign updated successfully"
//	@Failure		400		{object}	responses.APIResponse											"Invalid request or validation error"
//	@Failure		401		{object}	responses.APIResponse											"Unauthorized"
//	@Failure		404		{object}	responses.APIResponse											"Campaign not found"
//	@Failure		500		{object}	responses.APIResponse											"Internal server error"
//	@Security		BearerAuth
//	@Router			/api/v1/campaigns/id/{id} [put]
func (h *CampaignHandler) UpdateCampaign(c *gin.Context) {
	campaignID, err := extractParamID(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid campaign ID: "+err.Error(), http.StatusBadRequest))
		return
	}
	userID, err := extractUserID(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, responses.ErrorResponse("Unauthorized: "+err.Error(), http.StatusUnauthorized))
		return
	}
	var request *requests.UpdateCampaignRequest
	if err = c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid request format: "+err.Error(), http.StatusBadRequest))
		return
	}
	request.UpdatedBy = &userID
	if err = h.validartor.Struct(request); err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Validation error: "+err.Error(), http.StatusBadRequest))
		return
	}

	uow := h.uow.Begin(c.Request.Context())
	defer func() {
		if r := recover(); r != nil {
			uow.Rollback()
			panic(r)
		}
	}()

	campaignDetailsResponse, err := h.campaignService.UpdateCampaign(c.Request.Context(), uow, campaignID, request)
	if err != nil {
		uow.Rollback()
		var response *responses.APIResponse
		var statusCode int
		switch err.Error() {
		case "campaign not found":
			statusCode = http.StatusNotFound
			response = responses.ErrorResponse(err.Error(), statusCode)
		case "only DRAFT campaigns can be updated":
			statusCode = http.StatusBadRequest
			response = responses.ErrorResponse(err.Error(), statusCode)
		default:
			statusCode = http.StatusInternalServerError
			response = responses.ErrorResponse("Failed to update campaign: "+err.Error(), statusCode)
		}
		c.JSON(statusCode, response)
		return
	}

	if err = uow.Commit(); err != nil {
		uow.Rollback()
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse("Failed to update campaign: "+err.Error(), http.StatusInternalServerError))
		return
	}

	response := responses.SuccessResponse("Campaign updated successfully", utils.PtrOrNil(http.StatusOK), campaignDetailsResponse)
	c.JSON(http.StatusOK, response)
}
