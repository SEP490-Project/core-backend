package handler

import (
	"core-backend/config"
	"core-backend/internal/application"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/iproxies"
	"net/http"

	"github.com/gin-gonic/gin"
)

type TestHandler struct {
	config        *config.AppConfig
	tiktokProxy   iproxies.TikTokProxy
	facebookProxy iproxies.FacebookProxy
}

func NewTestHandler(config *config.AppConfig, applicationRegistry *application.ApplicationRegistry) *TestHandler {
	return &TestHandler{
		config:        config,
		tiktokProxy:   applicationRegistry.InfrastructureRegistry.ProxiesRegistry.TikTokProxy,
		facebookProxy: applicationRegistry.InfrastructureRegistry.ProxiesRegistry.FacebookProxy,
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
