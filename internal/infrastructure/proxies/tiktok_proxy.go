package proxies

import (
	"context"
	"core-backend/config"
	"core-backend/internal/application/dto/dtos"
	"core-backend/internal/application/interfaces/iproxies"
	"core-backend/pkg/utils"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"go.uber.org/zap"
)

type TikTokProxy struct {
	*BaseProxy
	config *config.TikTokSocialConfig
}

// region: ========= Authentication Methods =========

func (t *TikTokProxy) ExchangeCodeForToken(ctx context.Context, code string, redirectURL string) (*dtos.TikTokTokenResponse, error) {
	zap.L().Info("TikTokProxy - ExchangeCodeForToken called",
		zap.Int("code_length", len(code)),
		zap.String("redirect_url", redirectURL))

	path := "oauth/token/"

	// TikTok expects application/x-www-form-urlencoded
	formData := url.Values{}
	formData.Set("client_key", t.config.ClientKey)
	formData.Set("client_secret", t.config.ClientSecret)
	formData.Set("code", code)
	formData.Set("grant_type", "authorization_code")
	formData.Set("redirect_uri", redirectURL)

	headers := map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}

	resp, err := t.Post(ctx, path, headers, formData)
	if err != nil {
		zap.L().Error("Failed to exchange code for token", zap.Error(err))
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}
	defer resp.Body.Close()

	if err = t.HandleNon2xxHTTPResponse(resp); err != nil {
		return nil, err
	}
	var tokenResp dtos.TikTokTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		zap.L().Error("Failed to decode token response", zap.Error(err))
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &tokenResp, nil
}

func (t *TikTokProxy) RefreshAccessToken(ctx context.Context, refreshToken string) (*dtos.TikTokTokenResponse, error) {
	zap.L().Info("TikTokProxy - RefreshAccessToken called",
		zap.Int("refresh_token_length", len(refreshToken)))

	path := "oauth/token/"

	formData := url.Values{}
	formData.Set("client_key", t.config.ClientKey)
	formData.Set("client_secret", t.config.ClientSecret)
	formData.Set("grant_type", "refresh_token")
	formData.Set("refresh_token", refreshToken)

	headers := map[string]string{
		"Content-Type": "application/x-www-form-urlencoded",
	}

	resp, err := t.Post(ctx, path, headers, formData)
	if err != nil {
		zap.L().Error("Failed to refresh access token", zap.Error(err))
		return nil, fmt.Errorf("failed to refresh token: %w", err)
	}
	defer resp.Body.Close()

	if err = t.HandleNon2xxHTTPResponse(resp); err != nil {
		var errResp dtos.TikTokTokenErrorResponse
		if decodeErr := json.NewDecoder(resp.Body).Decode(&errResp); decodeErr == nil {
			zap.L().Error("TikTok API returned error on token refresh",
				zap.Any("error", errResp))
			return nil, fmt.Errorf("TikTok API error: %s - %s", errResp.Error, errResp.ErrorDescription)
		}
	}
	var tokenResp dtos.TikTokTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		zap.L().Error("Failed to decode token response", zap.Error(err))
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &tokenResp, nil
}

// endregion

// GetUserProfile implements iproxies.TikTokProxy.
func (t *TikTokProxy) GetUserProfile(ctx context.Context, accessToken string, openID string) (*dtos.TikTokUserProfileResponse, error) {
	scopes := "open_id,union_id,avatar_url,display_name"
	zap.L().Info("TikTokProxy - GetUserProfile called",
		zap.Int("access_token_length", len(accessToken)),
		zap.String("open_id", openID),
		zap.String("scopes", scopes))

	return t.getUserProfile(ctx, accessToken, scopes)
}

func (t *TikTokProxy) GetSystemUserProfile(ctx context.Context, accessToken string) (*dtos.TikTokUserProfileResponse, error) {
	scopes := "open_id,union_id,avatar_url,display_name,bio_description,profile_deep_link,is_verified,username,follower_count,following_count,likes_count,video_count"
	zap.L().Info("TikTokProxy - GetSystemUserProfile called",
		zap.Int("access_token_length", len(accessToken)),
		zap.String("scopes", scopes))

	return t.getUserProfile(ctx, accessToken, scopes)
}

func NewTikTokProxy(httpClient *http.Client, config *config.AppConfig) iproxies.TikTokProxy {
	tiktokConfig := config.Social.TikTok
	baseURL := fmt.Sprintf("%s/v%s/", tiktokConfig.BaseURL, tiktokConfig.APIVersion)
	return &TikTokProxy{
		BaseProxy: NewBaseProxy(httpClient, baseURL, config),
		config:    &tiktokConfig,
	}
}

// region: ========= Helper Methods =========

func (t *TikTokProxy) getUserProfile(ctx context.Context, accessToken string, fields string) (*dtos.TikTokUserProfileResponse, error) {
	url, err := utils.AddQueryParams("user/info/", map[string]string{
		"fields": fields,
	})
	if err != nil {
		zap.L().Error("Failed to construct URL for getting TikTok user profile", zap.Error(err))
		return nil, fmt.Errorf("failed to construct URL: %w", err)
	}

	headers := map[string]string{
		"Authorization": fmt.Sprintf("Bearer %s", accessToken),
	}

	var userProfile dtos.TikTokUserProfileResponse
	if err := GetGeneric(t.BaseProxy, ctx, url, headers, &userProfile); err != nil {
		zap.L().Error("Failed to get TikTok user profile", zap.Error(err))
		return nil, fmt.Errorf("failed to get user profile: %w", err)
	}

	if !userProfile.Error.Code.IsSuccess() {
		zap.L().Error("TikTok API returned error",
			zap.Any("error", userProfile.Error))
		return nil, fmt.Errorf("TikTok API error: %s - %s", userProfile.Error.Code, userProfile.Error.Message)
	}

	return &userProfile, nil
}

// endregion
