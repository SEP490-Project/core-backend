package handler

import (
	"core-backend/config"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/application/interfaces/iservice"
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

type FacebookSocialHandler struct {
	config                *config.AppConfig
	facebookSocialService iservice.FacebookSocialService
	unitOfWork            irepository.UnitOfWork
	validator             *validator.Validate
}

func NewFacebookSocialHandler(config *config.AppConfig, facebookSocialService iservice.FacebookSocialService, unitOfWork irepository.UnitOfWork) *FacebookSocialHandler {
	return &FacebookSocialHandler{
		config:                config,
		facebookSocialService: facebookSocialService,
		unitOfWork:            unitOfWork,
		validator:             validator.New(),
	}
}

// HandleLogin godoc
//
//	@Summary		Initiate Facebook OAuth HandleLogin process
//	@Description	Redirects the user to Facebook's OAuth authorization URL
//	@Tags			Social Authentication
//	@Accept			json
//	@Produce		json
//	@Param			redirect_url	query		string					false	"URL to redirect after successful login"
//	@Param			cancel_url		query		string					false	"URL to redirect if the user cancels the login"
//	@Param			is_internal		query		bool					false	"Whether to use internal scopes for admin users"
//	@Success		200				{object}	responses.APIResponse	"Redirect URL to Facebook OAuth"
//	@Failure		500				{object}	responses.APIResponse	"Internal server error"
//	@Security		BearerAuth
//	@Router			/api/v1/auth/facebook/login [get]
func (h *FacebookSocialHandler) HandleLogin(c *gin.Context) {
	var req requests.FacebookOAuthRequest
	if err := c.BindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid Facebook OAuth request: "+err.Error(), http.StatusBadRequest))
		return
	}
	if err := h.validator.Struct(&req); err != nil {
		c.JSON(http.StatusBadRequest, processValidationError(err))
		return
	}

	// If is_internal is true, only allow admin users to proceed
	userRole, err := extractUserRoles(c)
	if req.IsInternal && (err != nil || *userRole != enum.UserRoleAdmin.String()) {
		zap.L().Debug("User is not an admin, refusing to login with internal Facebook OAuth URL",
			zap.String("user_role", utils.DerefPtr(userRole, "nil")),
			zap.Error(err))
		c.JSON(http.StatusForbidden, responses.ErrorResponse("Forbidden: only admins can use internal Facebook OAuth login", http.StatusForbidden))
		return
	}

	redirectURL, err := h.buildBackendCallbackURL(req.IsInternal, req.RedirectURL, req.CancelURL)
	if err != nil {
		zap.L().Error("Failed to build backend callback URL", zap.Error(err))
		c.JSON(http.StatusInternalServerError, responses.ErrorResponse("Failed to construct OAuth URL", http.StatusInternalServerError))
		return
	}
	encodedRedirectURL := url.QueryEscape(redirectURL)
	facebookConfig := h.config.Social.Facebook
	scopeStr := strings.Join(facebookConfig.Scopes, ",")
	stateToken, err := crypto.GenerateStateToken(h.config.GetPrivateKey(), nil, map[string]string{"redirect_uri": encodedRedirectURL})
	if err != nil {
		zap.L().Debug("Failed to generate state token for Facebook OAuth", zap.Error(err))
		c.JSON(http.StatusInternalServerError,
			responses.ErrorResponse("Failed to generate state token for Facebook OAuth", http.StatusInternalServerError))
		return
	}

	urlStr := `https://www.facebook.com/v%s/dialog/oauth?client_id=%s&redirect_uri=%s&state=%s&scope=%s&response_type=%s`

	authorizationURL := fmt.Sprintf(urlStr,
		facebookConfig.APIVersion, facebookConfig.ClientID, encodedRedirectURL, stateToken, scopeStr, facebookConfig.ResponseType,
	)
	// safeURL := fmt.Sprintf(urlStr,
	// 	facebookConfig.APIVersion, "******", encodedRedirectURL, "******", scopeStr, facebookConfig.ResponseType,
	// )

	zap.L().Debug("Redirecting to Facebook OAuth URL", zap.String("url", authorizationURL))
	// c.Redirect(http.StatusFound, authorizationURL)
	c.JSON(http.StatusOK, responses.SuccessResponse("Redirect to Facebook OAuth URL", utils.PtrOrNil(http.StatusOK), map[string]any{
		"url": authorizationURL,
	}))
}

// HandleCallback godoc
//
//	@Summary		Handle Facebook OAuth callback
//	@Description	Handles the callback from Facebook OAuth after the user has authorized the application
//	@Tags			Social Authentication
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	responses.APIResponse{data=responses.LoginResponse}	"Login response"
//	@Failure		400	{object}	responses.APIResponse								"Invalid request body"
//	@Failure		500	{object}	responses.APIResponse								"Internal server error"
//	@Router			/api/v1/auth/facebook/callback [get]
func (h *FacebookSocialHandler) HandleCallback(c *gin.Context) {
	zap.L().Debug("FacebookSocialHandler HandleCallback called",
		zap.Any("query_params", c.Request.URL.Query()))

	code := c.Query("code")
	errorParam := c.Query("error")

	if errorParam != "" {
		h.handleCancelCallback(c)
		return
	}
	if code != "" {
		h.handleSuccessCallback(c)
		return
	}

	c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid Facebook OAuth callback: missing code or error parameter", http.StatusBadRequest))
}

// region: ====================== Helper Methods ======================

func (h *FacebookSocialHandler) handleCancelCallback(c *gin.Context) {
	var req requests.FacebookOAuthErrorRequest
	if err := c.BindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, responses.ErrorResponse("Invalid Facebook OAuth error query parameters", http.StatusBadRequest))
		return
	}

	redirectURL := req.CancelURL
	if redirectURL == "" {
		redirectURL = h.config.Social.Facebook.FrontendCancelURL
	}
	redirectURL, err := utils.AddQueryParams(redirectURL, map[string]string{
		"error_reason":      req.ErrorReason,
		"error":             req.Error,
		"error_description": req.ErrorDescription,
	})
	if err != nil {
		zap.L().Debug("Failed to add query parameters to Facebook OAuth error redirect URL", zap.Error(err))
		c.JSON(http.StatusBadRequest,
			responses.ErrorResponse("Failed to add query parameters to Facebook OAuth error redirect URL", http.StatusBadRequest))
		return
	}

	zap.L().Debug("Redirecting to Facebook OAuth error URL", zap.String("url", redirectURL))
	c.Redirect(http.StatusFound, redirectURL)
}

func (h *FacebookSocialHandler) handleSuccessCallback(c *gin.Context) {
	var (
		req                 requests.FacebookOAuthSuccessRequest
		err                 error
		redirectURL         = h.config.Social.Facebook.FrontendRedirectURL
		cancelURL           = h.config.Social.Facebook.FrontendCancelURL
		backendCallbackURL  string
		redirectQueryParams map[string]string
	)
	if err = c.ShouldBindQuery(&req); err != nil {
		zap.L().Error("Failed to bind Facebook OAuth success query parameters", zap.Error(err))
		cancelURL, _ = utils.AddQueryParams(cancelURL, map[string]string{
			"error_reason":      "invalid_request",
			"error":             "invalid_query_parameters",
			"error_description": "Failed to parse query parameters from Facebook OAuth callback.",
		})
		c.Redirect(http.StatusFound, cancelURL)
		return
	}
	if req.RedirectURL != "" {
		redirectURL = req.RedirectURL
	}
	if req.CancelURL != "" {
		cancelURL = req.CancelURL
	}

	if _, err = crypto.VerifyStateToken(h.config.GetPublicKey(), req.State); err != nil {
		zap.L().Error("Failed to verify state token,", zap.Error(err))
		cancelURL, _ = utils.AddQueryParams(cancelURL, map[string]string{
			"error_reason":      "invalid_state",
			"error":             "invalid_state_token",
			"error_description": err.Error(),
		})
		c.Redirect(http.StatusFound, cancelURL)
		return
	}
	backendCallbackURL, err = h.buildBackendCallbackURL(req.IsInternal, redirectURL, cancelURL)
	if err != nil {
		zap.L().Error("Failed to reconstruct original backend callback URL", zap.Error(err))
		cancelURL, _ = utils.AddQueryParams(cancelURL, map[string]string{
			"error_reason":      "internal_error",
			"error":             "url_construction_failed",
			"error_description": "Could not reconstruct the callback URL for token exchange.",
		})
		c.Redirect(http.StatusFound, cancelURL)
		return
	}
	req.BackendCallbackURL = backendCallbackURL

	withTransaction(c, h.unitOfWork, func(uow irepository.UnitOfWork) error {
		if req.IsInternal {
			err = h.facebookSocialService.HandleRefreshPageAccessToken(c.Request.Context(), uow, &req)
			if err != nil {
				zap.L().Error("Failed to refresh Facebook page access token", zap.Error(err))
				cancelURL, _ = utils.AddQueryParams(cancelURL, map[string]string{
					"error_reason":      "token_refresh_failed",
					"error":             "token_refresh_failed",
					"error_description": err.Error(),
				})
				c.Redirect(http.StatusFound, cancelURL)
				return err
			}

			redirectQueryParams = map[string]string{
				"is_internal": "true",
				"message":     "Facebook page access token refreshed successfully",
				"success":     "true",
			}
		} else {
			// Handle normal OAuth flow for user authentication
			deviceFingerprint := buildDeviceFingerprint(c)
			var loginResponse *responses.LoginResponse
			loginResponse, err = h.facebookSocialService.HandleOAuthLogin(c.Request.Context(), uow, req.Code, backendCallbackURL, deviceFingerprint)
			if err != nil {
				zap.L().Error("Failed to authenticate user via Facebook OAuth", zap.Error(err))
				cancelURL, _ = utils.AddQueryParams(cancelURL, map[string]string{
					"error_reason":      "authentication_failed",
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
	zap.L().Debug("Facebook OAuth login successful, redirecting to frontend", zap.String("redirect_url", redirectURL))
	c.Redirect(http.StatusFound, redirectURL)
}

func (h *FacebookSocialHandler) buildBackendCallbackURL(isInternal bool, finalRedirectURL, finalCancelURL string) (string, error) {
	if finalRedirectURL == "" {
		finalRedirectURL = h.config.Social.Facebook.FrontendRedirectURL
	}
	if finalCancelURL == "" {
		finalCancelURL = h.config.Social.Facebook.FrontendCancelURL
	}

	callbackParams := map[string]string{
		"is_internal":  strconv.FormatBool(isInternal),
		"redirect_url": finalRedirectURL,
		"cancel_url":   finalCancelURL,
	}

	// Use the base callback URL from the config and add the required parameters.
	return utils.AddQueryParams(h.config.Social.Facebook.RedirectURL, callbackParams)
}

// endregion
