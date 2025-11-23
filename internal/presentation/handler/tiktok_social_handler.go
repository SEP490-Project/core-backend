package handler

import (
	"core-backend/config"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/application/interfaces/iservice"
	"core-backend/internal/application/service"
	"core-backend/internal/domain/enum"
	"core-backend/pkg/crypto"
	"core-backend/pkg/utils"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"
)

type TikTokSocialHandler struct {
	config              *config.AppConfig
	tiktokSocialService iservice.TikTokSocialService
	unitOfWork          irepository.UnitOfWork
	validator           *validator.Validate
}

func NewTikTokSocialHandler(config *config.AppConfig, tiktokSocialService iservice.TikTokSocialService, unitOfWork irepository.UnitOfWork) *TikTokSocialHandler {
	return &TikTokSocialHandler{
		config:              config,
		tiktokSocialService: tiktokSocialService,
		validator:           validator.New(),
		unitOfWork:          unitOfWork,
	}
}

// region: ============== TikTok OAuth Handlers ==============

// HandleLogin godoc
//
//	@Summary		Initiate TikTok OAuth HandleLogin process
//	@Description	Redirects the user to TikTok's OAuth authorization URL
//	@Tags			Social Authentication
//	@Accept			json
//	@Produce		json
//	@Param			redirect_url	query		string					false	"URL to redirect after successful login"
//	@Param			cancel_url		query		string					false	"URL to redirect if the user cancels the login"
//	@Param			is_internal		query		bool					false	"Whether to use internal scopes for admin users"
//	@Success		200				{object}	responses.APIResponse	"TikTok OAuth URL generated successfully"
//	@Failure		500				{object}	responses.APIResponse	"Internal server error"
//	@Security		BearerAuth
//	@Router			/api/v1/auth/tiktok/login [get]
func (h *TikTokSocialHandler) HandleLogin(c *gin.Context) {
	var req requests.TikTokOAuthRequest
	if err := c.BindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid TikTok OAuth request: "+err.Error(), http.StatusBadRequest))
		return
	}
	if err := h.validator.Struct(&req); err != nil {
		c.JSON(http.StatusBadRequest, processValidationError(err))
		return
	}
	if req.RedirectURL == "" {
		req.RedirectURL = h.config.Social.TikTok.FrontendRedirectURL
	}
	if req.CancelURL == "" {
		req.CancelURL = h.config.Social.TikTok.FrontendCancelURL
	}

	// If is_internal is true, only allow admin users to proceed
	userRole, err := extractUserRoles(c)
	if req.IsInternal && (err != nil || *userRole != enum.UserRoleAdmin.String()) {
		zap.L().Debug("User is not an admin, refusing to login with internal TikTok OAuth URL",
			zap.String("user_role", *userRole))
		c.JSON(http.StatusForbidden, responses.ErrorResponse("Forbidden: only admins can use internal TikTok OAuth login", http.StatusForbidden))
		return
	}

	// redirectURL, err := h.buildBackendCallbackURL(req.IsInternal, req.RedirectURL, req.CancelURL)
	// if err != nil {
	// 	zap.L().Error("Failed to build backend callback URL", zap.Error(err))
	// 	c.JSON(http.StatusInternalServerError, responses.ErrorResponse("Failed to construct OAuth URL", http.StatusInternalServerError))
	// 	return
	// }
	tiktokConfig := h.config.Social.TikTok
	var scopeStr string
	if req.IsInternal {
		scopeStr = strings.Join(tiktokConfig.Scopes, ",")
	} else {
		scopeStr = strings.Join(tiktokConfig.UserScopes, ",")
	}
	encodedRedirectURL := url.QueryEscape(tiktokConfig.RedirectURL)
	stateData := map[string]string{
		"redirect_uri": tiktokConfig.RedirectURL,
		"is_internal":  strconv.FormatBool(req.IsInternal),
		"redirect_url": req.RedirectURL,
		"cancel_url":   req.CancelURL,
	}
	stateToken, err := crypto.GenerateStateToken(h.config.GetPrivateKey(), nil, stateData)
	if err != nil {
		zap.L().Debug("Failed to generate state token for TikTok OAuth", zap.Error(err))
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse("Internal server error", http.StatusInternalServerError))
		return
	}

	// TikTok uses client_key instead of client_id
	urlStr := "https://www.tiktok.com/v%s/auth/authorize/?client_key=%s&scope=%s&response_type=%s&redirect_uri=%s&state=%s"
	authorizationURL := fmt.Sprintf(urlStr,
		tiktokConfig.APIVersion,
		tiktokConfig.ClientKey,
		scopeStr,
		tiktokConfig.ResponseType,
		encodedRedirectURL,
		stateToken)

	// safeURL := fmt.Sprintf(urlStr,
	// 	tiktokConfig.APIVersion,
	// 	""+tiktokConfig.ClientKey[:4]+"****",
	// 	scopeStr,
	// 	tiktokConfig.ResponseType,
	// 	encodedRedirectURL,
	// 	""+stateToken[:4]+"****")

	zap.L().Debug("Redirecting to TikTok OAuth URL", zap.String("url", authorizationURL))
	// c.Redirect(http.StatusFound, authorizationURL)
	c.JSON(http.StatusOK, responses.SuccessResponse("TikTok OAuth URL generated successfully", utils.PtrOrNil(http.StatusOK), map[string]string{
		"url": authorizationURL,
	}))
}

// HandleCallback godoc
//
//	@Summary		Handle TikTok OAuth callback
//	@Description	Processes the OAuth callback from TikTok and redirects based on success or error
//	@Tags			Social Authentication
//	@Accept			json
//	@Produce		json
//	@Param			code				query		string					false	"Authorization code from TikTok"
//	@Param			state				query		string					false	"State token for CSRF protection"
//	@Param			error				query		string					false	"Error code if authorization failed"
//	@Param			error_description	query		string					false	"Error description if authorization failed"
//	@Success		302					{string}	string					"Redirect to success or cancel URL"
//	@Failure		400					{object}	responses.APIResponse	"Bad request"
//	@Failure		500					{object}	responses.APIResponse	"Internal server error"
//	@Router			/api/v1/auth/tiktok/callback [get]
func (h *TikTokSocialHandler) HandleCallback(c *gin.Context) {
	zap.L().Debug("TikTokSocialHandler HandleCallback called",
		zap.Any("query_params", c.Request.URL.Query()))
	code := c.Query("code")
	errorParam := c.Query("error")

	if code != "" {
		h.handleSuccess(c)
		return
	} else if errorParam != "" {
		h.handleError(c)
		return
	}

	c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid TikTok OAuth callback", http.StatusBadRequest))
}

// endregion

// region: ============== TikTok Content Handlers ==============

// endregion

// region: ============== TikTok Webhook Handler ==============

func (h *TikTokSocialHandler) HandleWebhook(c *gin.Context) {
	var body map[string]any
	if err := c.ShouldBindJSON(&body); err != nil {
		responses := responses.ErrorResponse("Invalid request body: "+err.Error(), http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, responses)
		return
	}

	zap.L().Debug("TikTokSocialHandler HandleWebhook called - Not Implemented",
		zap.Any("headers", c.Request.Header),
		zap.Any("query_params", c.Request.URL.Query()),
		zap.Any("body", body))
}

// endregion

// region: ============== TikTok Creator Info and User Profile Handlers ==============

// GetCreatorInfo godoc
//
//	@Summary		Get TikTok Creator Information
//	@Description	Retrieves information about the TikTok creator associated with the stored access token
//	@Tags			Social/TikTok
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	responses.APIResponse{data=dtos.TikTokCreatorInfoResponse}	"TikTok creator info retrieved successfully"
//	@Failure		403	{object}	responses.APIResponse										"TikTok refresh token expired"
//	@Failure		404	{object}	responses.APIResponse										"No TikTok token found"
//	@Failure		500	{object}	responses.APIResponse										"Internal server error"
//	@Security		BearerAuth
//	@Router			/api/v1/social/tiktok/creator-info [get]
func (h *TikTokSocialHandler) GetCreatorInfo(c *gin.Context) {
	creatorInfo, err := h.tiktokSocialService.GetTikTokCreatorInfo(c.Request.Context())
	if err != nil {
		switch err {
		case service.TikTokRefreshExpiredErr:
			c.JSON(http.StatusForbidden, responses.ErrorResponse("TikTok refresh token expired", http.StatusForbidden))
		case service.TikTokNoStoredTokenErr:
			c.JSON(http.StatusNotFound, responses.ErrorResponse("No TikTok token found, please authenticate first", http.StatusNotFound))
		default:
			c.JSON(http.StatusInternalServerError, responses.ErrorResponse("Failed to get TikTok creator info: "+err.Error(), http.StatusInternalServerError))
		}
		return
	}

	c.JSON(http.StatusOK,
		responses.SuccessResponse("TikTok creator info retrieved successfully", utils.PtrOrNil(http.StatusOK), creatorInfo))
}

// GetSystemUserProfile godoc
//
//	@Summary		Get TikTok System User Profile
//	@Description	Retrieves the TikTok system user profile associated with the stored access token
//	@Tags			Social/TikTok
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	responses.APIResponse{data=dtos.TikTokUserProfileResponse}	"TikTok system user profile retrieved successfully"
//	@Failure		403	{object}	responses.APIResponse										"TikTok refresh token expired"
//	@Failure		404	{object}	responses.APIResponse										"No TikTok token found"
//	@Failure		500	{object}	responses.APIResponse										"Internal server error"
//	@Security		BearerAuth
//	@Router			/api/v1/social/tiktok/system-user-profile [get]
func (h *TikTokSocialHandler) GetSystemUserProfile(c *gin.Context) {
	systemUserProfile, err := h.tiktokSocialService.GetTikTokSystemUserProfile(c.Request.Context())
	if err != nil {
		switch err {
		case service.TikTokRefreshExpiredErr:
			c.JSON(http.StatusForbidden, responses.ErrorResponse("TikTok refresh token expired", http.StatusForbidden))
		case service.TikTokNoStoredTokenErr:
			c.JSON(http.StatusNotFound, responses.ErrorResponse("No TikTok token found, please authenticate first", http.StatusNotFound))
		default:
			c.JSON(http.StatusInternalServerError, responses.ErrorResponse("Failed to get TikTok system user profile: "+err.Error(), http.StatusInternalServerError))
		}
		return
	}
	c.JSON(http.StatusOK,
		responses.SuccessResponse("TikTok system user profile retrieved successfully", utils.PtrOrNil(http.StatusOK), systemUserProfile))
}

// endregion

// region: ============== Helper Methods ==============

func (h *TikTokSocialHandler) handleSuccess(c *gin.Context) {
	var (
		req                 requests.TikTokOAuthSuccessRequest
		err                 error
		redirectURL         = h.config.Social.TikTok.FrontendRedirectURL
		cancelURL           = h.config.Social.TikTok.FrontendCancelURL
		redirectQueryParams map[string]string
		stateToken          *crypto.StatePayload
	)
	if err = c.ShouldBindQuery(&req); err != nil {
		zap.L().Error("Failed to bind TikTok OAuth success query parameters", zap.Error(err))
		cancelURL, _ = utils.AddQueryParams(cancelURL, map[string]string{
			"error":             "invalid_query_parameters",
			"error_description": "Failed to parse query parameters from TikTok OAuth callback.",
		})
		c.Redirect(http.StatusFound, cancelURL)
		return
	}

	if stateToken, err = crypto.VerifyStateToken(h.config.GetPublicKey(), req.State); err != nil {
		zap.L().Error("Failed to verify state token", zap.Error(err))
		cancelURL, _ = utils.AddQueryParams(cancelURL, map[string]string{
			"error":             "invalid_state_token",
			"error_description": err.Error(),
		})
		c.Redirect(http.StatusFound, cancelURL)
		return
	}
	stateData := stateToken.Data
	if redirectURI, ok := stateData["redirect_uri"]; ok && redirectURI != "" {
		req.BackendCallbackURL = redirectURI
	}
	if isInternal, ok := stateData["is_internal"]; ok && isInternal != "" {
		req.IsInternal, _ = strconv.ParseBool(isInternal)
	}
	if redirectURLStr, ok := stateData["redirect_url"]; ok && redirectURLStr != "" {
		redirectURL = redirectURLStr
		req.RedirectURL = redirectURLStr
	}
	if cancelURLStr, ok := stateData["cancel_url"]; ok && cancelURLStr != "" {
		cancelURL = cancelURLStr
		req.CancelURL = cancelURLStr
	}

	withTransaction(c, h.unitOfWork, func(uow irepository.UnitOfWork) error {
		if req.IsInternal {
			err = h.tiktokSocialService.HandleRefreshAccessToken(c.Request.Context(), uow, &req)

			if err != nil {
				zap.L().Error("Failed to handle TikTok OAuth callback", zap.Error(err))
				cancelURL, _ = utils.AddQueryParams(cancelURL, map[string]string{
					"error":             "authentication_failed",
					"error_description": err.Error(),
				})
				c.Redirect(http.StatusFound, cancelURL)
				return err
			}

			redirectQueryParams = map[string]string{
				"is_internal": "true",
				"message":     "TikTok access token refreshed successfully",
				"success":     "true",
			}
		} else {
			// Handle normal OAuth flow for user authentication
			deviceFingerprint := buildDeviceFingerprint(c)
			var loginResponse *responses.LoginResponse
			loginResponse, err = h.tiktokSocialService.HandleOAuthLogin(c.Request.Context(), uow, req.Code, req.BackendCallbackURL, deviceFingerprint)
			if err != nil {
				zap.L().Error("Failed to authenticate user via TikTok OAuth", zap.Error(err))
				cancelURL, _ = utils.AddQueryParams(cancelURL, map[string]string{
					"error":             "authentication_failed",
					"error_description": err.Error(),
				})
				c.Redirect(http.StatusFound, cancelURL)
				return err
			}

			// Redirect with tokens and user info
			redirectQueryParams = map[string]string{
				"access_token":  loginResponse.AccessToken,
				"refresh_token": loginResponse.RefreshToken,
				"user_id":       loginResponse.User.ID.String(),
				"username":      loginResponse.User.Username,
				"email":         loginResponse.User.Email,
				"success":       "true",
			}
		}

		return nil
	})

	redirectURL, _ = utils.AddQueryParams(redirectURL, redirectQueryParams)
	zap.L().Debug("Redirecting to TikTok OAuth success URL", zap.String("url", redirectURL))
	c.Redirect(http.StatusFound, redirectURL)
}

func (h *TikTokSocialHandler) handleError(c *gin.Context) {
	var req requests.TikTokOAuthErrorRequest
	cancelURL := c.Query("cancel_url")
	if cancelURL == "" {
		cancelURL = h.config.Social.TikTok.FrontendCancelURL
	}

	if err := c.ShouldBindQuery(&req); err != nil {
		zap.L().Error("Failed to bind TikTok OAuth error query parameters", zap.Error(err))
		cancelURL, _ = utils.AddQueryParams(cancelURL, map[string]string{
			"error":             "invalid_request",
			"error_description": "Failed to parse error parameters from TikTok OAuth callback.",
		})
		c.Redirect(http.StatusFound, cancelURL)
		return
	}

	zap.L().Info("TikTok OAuth error",
		zap.String("error", req.Error),
		zap.String("error_description", req.ErrorDescription))

	redirectURL, err := utils.AddQueryParams(cancelURL, map[string]string{
		"error":             req.Error,
		"error_description": req.ErrorDescription,
	})
	if err != nil {
		zap.L().Debug("Failed to add query parameters to TikTok OAuth error redirect URL", zap.Error(err))
		c.JSON(http.StatusBadRequest,
			responses.ErrorResponse("Failed to add query parameters to TikTok OAuth error redirect URL", http.StatusBadRequest))
		return
	}

	zap.L().Debug("Redirecting to TikTok OAuth error URL", zap.String("url", redirectURL))
	c.Redirect(http.StatusFound, redirectURL)
}

// endregion
