package handler

import (
	"core-backend/config"
	"core-backend/internal/application"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/iproxies"
	"core-backend/internal/application/service/helper"
	"core-backend/internal/domain/model"
	gormrepository "core-backend/internal/infrastructure/gorm_repository"
	"core-backend/pkg/utils"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type TestHandler struct {
	config         *config.AppConfig
	tiktokProxy    iproxies.TikTokProxy
	facebookProxy  iproxies.FacebookProxy
	dbRegistry     *gormrepository.DatabaseRegistry
	applicationReg *application.ApplicationRegistry
}

func NewTestHandler(config *config.AppConfig, applicationRegistry *application.ApplicationRegistry) *TestHandler {
	return &TestHandler{
		config:         config,
		tiktokProxy:    applicationRegistry.InfrastructureRegistry.ProxiesRegistry.TikTokProxy,
		facebookProxy:  applicationRegistry.InfrastructureRegistry.ProxiesRegistry.FacebookProxy,
		dbRegistry:     applicationRegistry.DatabaseRegistry,
		applicationReg: applicationRegistry,
	}
}

// TikTokExchangeCodeForToken godoc
//
//	@Summary		Exchange TikTok OAuth code for access token
//	@Description	Exchanges the authorization code received from TikTok OAuth for an access token.
//	@Tags			Test
//	@Accept			json
//	@Produce		json
//	@Param			code			query		string					true	"Authorization code received from TikTok OAuth"
//	@Param			redirect_uri	query		string					true	"Redirect URI used in the OAuth flow"
//	@Success		200				{object}	any						"Successfully exchanged code for token"
//	@Failure		400				{object}	responses.APIResponse	"Bad request due to invalid parameters"
//	@Security		BearerAuth
//	@Router			/api/v1/test/tiktok/exchange-code-for-token [get]
func (h *TestHandler) TikTokExchangeCodeForToken(c *gin.Context) {
	code := c.Query("code")
	redirectURI := c.Query("redirect_uri")
	if code == "" || redirectURI == "" {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid TikTok OAuth request: missing code or redirect_uri", http.StatusBadRequest))
		return
	}

	tokenResp, err := h.tiktokProxy.ExchangeCodeForToken(c.Request.Context(), code, redirectURI)
	if err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Failed to exchange code for token", http.StatusBadRequest))
		return
	}

	c.JSON(http.StatusOK, tokenResp)
}

// TikTokRefreshAccessToken godoc
//
//	@Summary		Refresh TikTok OAuth access token
//	@Description	Refreshes the TikTok OAuth access token using the provided refresh token.
//	@Tags			Test
//	@Accept			json
//	@Produce		json
//	@Param			refresh_token	query		string					true	"Refresh token used to obtain a new access token"
//	@Success		200				{object}	any						"Successfully refreshed access token"
//	@Failure		400				{object}	responses.APIResponse	"Bad request due to invalid parameters"
//	@Security		BearerAuth
//	@Router			/api/v1/test/tiktok/refresh-access-token [get]
func (h *TestHandler) TikTokRefreshAccessToken(c *gin.Context) {
	refreshToken := c.Query("refresh_token")
	if refreshToken == "" {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid TikTok OAuth request: missing refresh_token", http.StatusBadRequest))
		return
	}

	tokenResp, err := h.tiktokProxy.RefreshAccessToken(c.Request.Context(), refreshToken)
	if err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Failed to refresh access token", http.StatusBadRequest))
		return
	}

	c.JSON(http.StatusOK, tokenResp)
}

// TikTokGetUserProfile godoc
//
//	@Summary		Get TikTok user profile
//	@Description	Retrieves the TikTok user profile using the provided access token.
//	@Tags			Test
//	@Accept			json
//	@Produce		json
//	@Param			access_token	query		string					true	"Access token used to retrieve the user profile"
//	@Success		200				{object}	any						"Successfully retrieved user profile"
//	@Failure		400				{object}	responses.APIResponse	"Bad request due to invalid parameters"
//	@Security		BearerAuth
//	@Router			/api/v1/test/tiktok/get-user-profile [get]
func (h *TestHandler) TikTokGetUserProfile(c *gin.Context) {
	accessToken := c.Query("access_token")
	if accessToken == "" {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid TikTok OAuth request: missing access_token or open_id", http.StatusBadRequest))
		return
	}

	userProfile, err := h.tiktokProxy.GetUserProfile(c.Request.Context(), accessToken, "")
	if err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Failed to get user profile", http.StatusBadRequest))
		return
	}

	c.JSON(http.StatusOK, userProfile)
}

// TikTokGetSystemUserProfile godoc
//
//	@Summary		Get TikTok system user profile
//	@Description	Retrieves the TikTok system user profile using the provided access token.
//	@Tags			Test
//	@Accept			json
//	@Produce		json
//	@Param			access_token	query		string					true	"Access token used to retrieve the system user profile"
//	@Success		200				{object}	any						"Successfully retrieved system user profile"
//	@Failure		400				{object}	responses.APIResponse	"Bad request due to invalid parameters"
//	@Security		BearerAuth
//	@Router			/api/v1/test/tiktok/get-system-user-profile [get]
func (h *TestHandler) TikTokGetSystemUserProfile(c *gin.Context) {
	accessToken := c.Query("access_token")
	if accessToken == "" {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid TikTok OAuth request: missing access_token or open_id", http.StatusBadRequest))
		return
	}

	userProfile, err := h.tiktokProxy.GetSystemUserProfile(c.Request.Context(), accessToken)
	if err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Failed to get system user profile", http.StatusBadRequest))
		return
	}

	c.JSON(http.StatusOK, userProfile)
}

// TikTokGetCreatorInfo godoc
//
//	@Summary		Get TikTok creator info
//	@Description	Retrieves the TikTok creator info using the provided access token.
//	@Tags			Test
//	@Accept			json
//	@Produce		json
//	@Param			access_token	query		string					true	"Access token used to retrieve the creator info"
//	@Success		200				{object}	any						"Successfully retrieved creator info"
//	@Failure		400				{object}	responses.APIResponse	"Bad request due to invalid parameters"
//	@Security		BearerAuth
//	@Router			/api/v1/test/tiktok/get-creator-info [get]
func (h *TestHandler) TikTokGetCreatorInfo(c *gin.Context) {
	accessToken := c.Query("access_token")
	if accessToken == "" {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid TikTok OAuth request: missing access_token or open_id", http.StatusBadRequest))
		return
	}

	creatorInfo, err := h.tiktokProxy.GetCreatorInfo(c.Request.Context(), accessToken)
	if err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Failed to get creator info", http.StatusBadRequest))
		return
	}

	c.JSON(http.StatusOK, creatorInfo)
}

// MigrateScopeOfWorkIDs godoc
//
//	@Summary		Migrate ScopeOfWork IDs
//	@Description	Populates task_ids, product_ids, content_ids in ScopeOfWork based on existing data
//	@Tags			Test
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	any
//	@Security		BearerAuth
//	@Router			/api/v1/test/migrate-sow-ids [post]
func (h *TestHandler) MigrateScopeOfWorkIDs(c *gin.Context) {
	ctx := c.Request.Context()

	// 1. Get all contracts
	contracts, _, err := h.dbRegistry.ContractRepository.GetAll(ctx, func(db *gorm.DB) *gorm.DB {
		return db.
			Joins("INNER JOIN campaigns ON campaigns.contract_id = contracts.id").
			Where("contracts.scope_of_work IS NOT NULL AND contracts.scope_of_work != ?", "null")
	}, nil, 10000, 0)
	if err != nil {
		zap.L().Error("Failed to get contracts", zap.Error(err))
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse("Failed to get contracts", http.StatusInternalServerError))
		return
	}

	zap.L().Info("Starting migration", zap.Int("total_contracts", len(contracts)))

	updatedCount := 0
	for _, contract := range contracts {
		// 2. Get Campaign
		campaign, err := h.dbRegistry.CampaignRepository.GetByCondition(ctx, func(db *gorm.DB) *gorm.DB {
			return db.Where("contract_id = ?", contract.ID)
		}, []string{"Milestones.Tasks.Product", "Milestones.Tasks.Contents"})

		if err != nil {
			zap.L().Warn("Failed to get campaign for contract", zap.String("contract_id", contract.ID.String()), zap.Error(err))
			continue
		}
		if campaign == nil {
			zap.L().Info("No campaign found for contract", zap.String("contract_id", contract.ID.String()))
			continue
		}

		// 3. Map Tasks by ScopeOfWorkItemID
		zap.L().Info("Mapping tasks for contract",
			zap.String("contract_id", contract.ID.String()),
			zap.Int("milestone_count", len(campaign.Milestones)),
		)

		// Use helper to map tasks to SOW items (Best Effort)
		if err := helper.MapTasksToScopeOfWork(&contract, campaign.Milestones); err != nil {
			zap.L().Error("Failed to map tasks to SOW", zap.String("contract_id", contract.ID.String()), zap.Error(err))
			continue
		}

		// 4. Save Tasks (to persist ScopeOfWorkItemID)
		tasks := utils.FlatMapMapper(campaign.Milestones, func(m *model.Milestone) []*model.Task { return m.Tasks })
		tasksUpdated := 0
		for _, t := range tasks {
			if err := h.dbRegistry.TaskRepository.Update(ctx, t); err != nil {
				zap.L().Error("Failed to update task", zap.String("task_id", t.ID.String()), zap.Error(err))
			} else {
				tasksUpdated++
			}
		}
		zap.L().Info("Updated tasks for contract", zap.String("contract_id", contract.ID.String()), zap.Int("tasks_updated", tasksUpdated))

		// 5. Save Contract (to persist updated ScopeOfWork)
		if err := h.dbRegistry.ContractRepository.Update(ctx, &contract); err == nil {
			updatedCount++
			zap.L().Info("Successfully updated contract", zap.String("contract_id", contract.ID.String()))
		} else {
			zap.L().Error("Failed to update contract", zap.String("contract_id", contract.ID.String()), zap.Error(err))
		}
	}

	c.JSON(http.StatusOK, gin.H{"updated_contracts": updatedCount})
}

// UpdateContractScopeOfWork godoc
//
//	@Summary		Update Contract Scope of Work
//	@Description	Updates the scope of work for a specific contract based on its associated tasks.
//	@Tags			Test
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string					true	"Contract ID"
//	@Success		200	{object}	any						"Successfully updated contract scope of work"
//	@Failure		400	{object}	responses.APIResponse	"Bad request due to invalid parameters"
//	@Failure		500	{object}	responses.APIResponse	"Internal server error"
//	@Security		BearerAuth
//	@Router			/api/v1/test/contracts/{id}/update-sow [put]
func (h *TestHandler) UpdateContractScopeOfWork(c *gin.Context) {
	contractID, err := extractParamID(c, "id")
	if err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid contract ID", http.StatusBadRequest))
		return
	}

	contractService := h.applicationReg.ContractService
	if err := contractService.UpdateContractScopeOfWorkWithReferencinnTaskIDs(c.Request.Context(), contractID); err != nil {
		zap.L().Error("Failed to update contract scope of work", zap.String("contract_id", contractID.String()), zap.Error(err))
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse("Failed to update contract scope of work", http.StatusInternalServerError))
		return
	}

	c.JSON(http.StatusOK, responses.SuccessResponse("Contract scope of work updated successfully", nil, nil))
}

// UpdateAllContractScopeOfWork godoc
//
//	@Summary		Update All Contract Scope of Work
//	@Description	Updates the scope of work for all contracts
//	@Tags			Test
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	any						"Successfully updated all contract scope of work"
//	@Failure		500	{object}	responses.APIResponse	"Internal server error"
//	@Security		BearerAuth
//	@Router			/api/v1/test/contracts/update-all-contracts-sow [put]
func (h *TestHandler) UpdateAllContractScopeOfWork(c *gin.Context) {
	if err := h.applicationReg.ContractService.UpdateAllContractScopeOfWorkWithReferencinnTaskIDs(c.Request.Context()); err != nil {
		zap.L().Error("Failed to update all contract scope of work", zap.Error(err))
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse("Failed to update all contract scope of work", http.StatusInternalServerError))
		return
	}

	c.JSON(http.StatusOK, responses.SuccessResponse("All contract scope of work updated successfully", nil, nil))
}
