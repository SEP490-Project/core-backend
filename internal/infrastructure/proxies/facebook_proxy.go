package proxies

import (
	"context"
	"core-backend/config"
	"core-backend/internal/application/dto/dtos"
	"core-backend/internal/application/interfaces/iproxies"
	"core-backend/pkg/utils"
	"fmt"
	"net/http"

	"go.uber.org/zap"
)

type FacebookProxy struct {
	*BaseProxy
	config *config.FacebookSocialConfig
}

// ExchangeCodeForUserAccessToken implements iproxies.FacebookProxy.
func (f *FacebookProxy) ExchangeCodeForUserAccessToken(ctx context.Context, code string, redirectURL string) (*dtos.FacebookAccessTokenResponse, error) {
	zap.L().Info("FacebookProxy - ExchangeCodeForUserAccessToken called",
		zap.Int("code", len(code)),
		zap.String("redirect_url", redirectURL))

	url, err := utils.AddQueryParams("/oauth/access_token", map[string]string{
		"client_id":     f.config.ClientID,
		"client_secret": f.config.ClientSecret,
		"redirect_uri":  redirectURL,
		"code":          code,
	})
	if err != nil {
		zap.L().Error("Failed to construct URL for exchanging code for user access token",
			zap.Error(err))
		return nil, fmt.Errorf("failed to construct URL: %w", err)
	}

	var tokenResp dtos.FacebookAccessTokenResponse
	if err := GetGeneric(f.BaseProxy, ctx, url, nil, &tokenResp); err != nil {
		zap.L().Error("Failed to exchange code for user access token",
			zap.Error(err))
		return nil, fmt.Errorf("failed to exchange code for user access token: %w", err)
	}

	return &tokenResp, nil
}

// ExchangeUserAccessTokenForLongLivedToken implements iproxies.FacebookProxy.
func (f *FacebookProxy) ExchangeUserAccessTokenForLongLivedToken(ctx context.Context, userAccessToken string) (*dtos.FacebookAccessTokenResponse, error) {
	zap.L().Info("FacebookProxy - ExchangeUserAccessTokenForLongLivedToken called",
		zap.Int("user_access_token_length", len(userAccessToken)))

	url, err := utils.AddQueryParams("/oauth/access_token", map[string]string{
		"grant_type":        "fb_exchange_token",
		"client_id":         f.config.ClientID,
		"client_secret":     f.config.ClientSecret,
		"fb_exchange_token": userAccessToken,
	})
	if err != nil {
		zap.L().Error("Failed to construct URL for exchanging user access token for long-lived token",
			zap.Error(err))
		return nil, fmt.Errorf("failed to construct URL: %w", err)
	}

	var tokenResp dtos.FacebookAccessTokenResponse
	if err := GetGeneric(f.BaseProxy, ctx, url, nil, &tokenResp); err != nil {
		zap.L().Error("Failed to exchange user access token for long-lived token",
			zap.Error(err))
		return nil, fmt.Errorf("failed to exchange user access token for long-lived token: %w", err)
	}

	return &tokenResp, nil
}

// GetAccountPageInfo implements iproxies.FacebookProxy.
func (f *FacebookProxy) GetAccountPageInfo(ctx context.Context, userAccessToken string) (*dtos.FacebookAccountInfoResponse, error) {
	zap.L().Info("FacebookProxy - GetAccountPageInfo called",
		zap.Int("user_access_token_length", len(userAccessToken)))

	url, err := utils.AddQueryParam("/me/accounts", "access_token", userAccessToken)
	if err != nil {
		zap.L().Error("Failed to construct URL for getting Facebook account info",
			zap.Error(err))
		return nil, fmt.Errorf("failed to construct URL: %w", err)
	}

	var accountInfoResp dtos.FacebookAccountInfoResponse
	if err := GetGeneric(f.BaseProxy, ctx, url, nil, &accountInfoResp); err != nil {
		zap.L().Error("Failed to get Facebook account info",
			zap.Error(err))
		return nil, fmt.Errorf("failed to get Facebook account info: %w", err)
	}

	return &accountInfoResp, nil
}

// GetUserProfile implements iproxies.FacebookProxy.
func (f *FacebookProxy) GetUserProfile(ctx context.Context, userAccessToken string) (*dtos.FacebookUserProfileResponse, error) {
	zap.L().Info("FacebookProxy - GetUserProfile called",
		zap.Int("user_access_token_length", len(userAccessToken)))

	url, err := utils.AddQueryParams("/me", map[string]string{
		"access_token": userAccessToken,
		"fields":       "id,name,email,picture,birthday",
	})
	if err != nil {
		zap.L().Error("Failed to construct URL for getting Facebook user profile",
			zap.Error(err))
		return nil, fmt.Errorf("failed to construct URL: %w", err)
	}

	var userProfile dtos.FacebookUserProfileResponse
	if err := GetGeneric(f.BaseProxy, ctx, url, nil, &userProfile); err != nil {
		zap.L().Error("Failed to get Facebook user profile",
			zap.Error(err))
		return nil, fmt.Errorf("failed to get Facebook user profile: %w", err)
	}

	return &userProfile, nil
}

func NewFacebookProxy(httpClient *http.Client, config *config.FacebookSocialConfig) iproxies.FacebookProxy {
	baseURL := fmt.Sprintf("%s/v%s", config.BaseURL, config.APIVersion)
	return &FacebookProxy{
		BaseProxy: NewBaseProxy(httpClient, baseURL),
		config:    config,
	}
}
