package handler

import (
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/application/interfaces/iservice"
	"core-backend/pkg/utils"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

type CampaignHandler struct {
	campaignService iservice.CampaignService
	uow             irepository.UnitOfWork
	validartor      *validator.Validate
}

func NewCampaignHandler(
	campaignService iservice.CampaignService,
	uow irepository.UnitOfWork,
) *CampaignHandler {
	return &CampaignHandler{
		campaignService: campaignService,
		uow:             uow,
		validartor:      validator.New(),
	}
}

// CreateCampaignFromContract godoc
// @Summary 	Create Campaign from Contract
// @Description Create a new campaign based on the provided contract details.
// @Tags 		Campaigns
// @Accept 		json
// @Produce 	json
// @Param 		data body requests.CreateCampaignRequest true "Campaign creation data"
// @Success 	201 {object} responses.APIResponse{data=responses.CampaignResponse} "Campaign created successfully"
// @Failure 	400 {object} responses.APIResponse "Invalid request or validation error"
// @Failure 	401 {object} responses.APIResponse "Unauthorized"
// @Failure 	403 {object} responses.APIResponse "Forbidden"
// @Failure 	500 {object} responses.APIResponse "Internal server error"
// @Security    BearerAuth
// @Router      /api/v1/contracts [post]
func (h *CampaignHandler) CreateCampaignFromContract(c *gin.Context) {
	var request *requests.CreateCampaignRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		responses := responses.ErrorResponse("Invalid request format: "+err.Error(), http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, responses)
		return
	}
	if err := h.validartor.Struct(request); err != nil {
		responses := responses.ErrorResponse("Validation error: "+err.Error(), http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, responses)
		return
	}

	activeUow := h.uow.Begin()
	var campaignResponse *responses.CampaignResponse
	var err error
	if campaignResponse, err = h.campaignService.CreateCampaignFromContract(c.Request.Context(), request, activeUow); err != nil {
		activeUow.Rollback()
		responses := responses.ErrorResponse("Failed to create campaign: "+err.Error(), http.StatusInternalServerError)
		c.JSON(http.StatusInternalServerError, responses)
		return
	}

	activeUow.Commit()
	responses := responses.SuccessResponse("Campaign created successfully", utils.IntPtr(http.StatusCreated), campaignResponse)
	c.JSON(http.StatusCreated, responses)
}
