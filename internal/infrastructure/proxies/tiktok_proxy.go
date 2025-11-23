package proxies

import (
	"context"
	"core-backend/config"
	"core-backend/internal/application/dto/dtos"
	"core-backend/internal/application/interfaces/iproxies"
	"core-backend/pkg/utils"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"

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

// region: ========= User Profile Methods =========

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

// endregion

// region: ========= Content Posting Methods =========

// GetCreatorInfo implements iproxies.TikTokProxy.
func (t *TikTokProxy) GetCreatorInfo(ctx context.Context, accessToken string) (*dtos.TikTokCreatorInfoResponse, error) {
	zap.L().Info("TikTokProxy - GetCreatorInfo called",
		zap.Int("access_token_length", len(accessToken)))

	path := "post/publish/creator_info/query/"

	headers := map[string]string{
		"Authorization": fmt.Sprintf("Bearer %s", accessToken),
		"Content-Type":  "application/json; charset=UTF-8",
	}

	var creatorInfo dtos.TikTokCreatorInfoResponse
	if err := PostGeneric(t.BaseProxy, ctx, path, headers, map[string]any{}, &creatorInfo); err != nil {
		zap.L().Error("Failed to get TikTok creator info", zap.Error(err))
		return nil, fmt.Errorf("failed to get creator info: %w", err)
	}

	if creatorInfo.Error.Code != "" && creatorInfo.Error.Code != "ok" {
		zap.L().Error("TikTok API returned error on getting creator info",
			zap.Any("error", creatorInfo.Error))
		return nil, fmt.Errorf("TikTok API error: %s - %s", creatorInfo.Error.Code, creatorInfo.Error.Message)
	}

	return &creatorInfo, nil
}

// region: ========= Content Publishing API methods (Direct Post) =========

// InitVideoPost implements iproxies.TikTokProxy.
func (t *TikTokProxy) InitVideoPost(ctx context.Context, accessToken string, req *dtos.TikTokVideoInitRequest) (*dtos.TikTokVideoInitResponse, error) {
	zap.L().Info("TikTokProxy - InitVideoPost called",
		zap.Any("request", req))

	path := "post/publish/video/init/"

	headers := map[string]string{
		"Authorization": fmt.Sprintf("Bearer %s", accessToken),
		"Content-Type":  "application/json; charset=UTF-8",
	}

	var initResp dtos.TikTokVideoInitResponse
	if err := PostGeneric(t.BaseProxy, ctx, path, headers, req, &initResp); err != nil {
		zap.L().Error("Failed to initialize TikTok video post", zap.Error(err))
		return nil, fmt.Errorf("failed to init video post: %w", err)
	}

	if initResp.Error.Code != "" && initResp.Error.Code != "ok" {
		zap.L().Error("TikTok API returned error on video init",
			zap.Any("error", initResp.Error))
		return nil, fmt.Errorf("TikTok API error: %s - %s", initResp.Error.Code, initResp.Error.Message)
	}

	zap.L().Info("TikTok video init successful",
		zap.String("publish_id", initResp.Data.PublishID))

	return &initResp, nil
}

// UploadVideoChunk implements iproxies.TikTokProxy.
func (t *TikTokProxy) UploadVideoChunk(ctx context.Context, uploadURL string, videoData []byte, chunkIndex int, totalChunks int, fileSize int64) error {
	zap.L().Info("TikTokProxy - UploadVideoChunk called",
		zap.Int("chunk_index", chunkIndex),
		zap.Int("total_chunks", totalChunks),
		zap.Int("chunk_size", len(videoData)))

	// Calculate byte range for this chunk
	chunkSize := int64(len(videoData))
	startByte := int64(chunkIndex) * chunkSize
	endByte := startByte + chunkSize - 1
	if endByte >= fileSize {
		endByte = fileSize - 1
	}

	headers := map[string]string{
		"Content-Type":   "video/mp4",
		"Content-Range":  fmt.Sprintf("bytes %d-%d/%d", startByte, endByte, fileSize),
		"Content-Length": fmt.Sprintf("%d", chunkSize),
	}

	// Create a temporary BaseProxy with the upload URL
	uploadProxy := NewBaseProxy(t.httpClient, uploadURL, t.BaseProxy.config)

	resp, err := uploadProxy.Put(ctx, "", headers, videoData)
	if err != nil {
		zap.L().Error("Failed to upload video chunk", zap.Error(err))
		return fmt.Errorf("failed to upload chunk: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		zap.L().Error("TikTok video chunk upload failed",
			zap.Int("status_code", resp.StatusCode),
			zap.Int("chunk_index", chunkIndex))
		return fmt.Errorf("video chunk upload failed with status: %d", resp.StatusCode)
	}

	zap.L().Info("TikTok video chunk uploaded successfully",
		zap.Int("chunk_index", chunkIndex),
		zap.Int("total_chunks", totalChunks))

	return nil
}

// endregion

// CheckPostStatus implements iproxies.TikTokProxy.
func (t *TikTokProxy) CheckPostStatus(ctx context.Context, publishID string, accessToken string) (*dtos.TikTokPostStatusResponse, error) {
	zap.L().Info("TikTokProxy - CheckPostStatus called",
		zap.String("publish_id", publishID))

	path := "post/publish/status/fetch/"

	headers := map[string]string{
		"Authorization": fmt.Sprintf("Bearer %s", accessToken),
		"Content-Type":  "application/json; charset=UTF-8",
	}

	body := map[string]any{
		"publish_id": publishID,
	}

	var statusResp dtos.TikTokPostStatusResponse
	if err := PostGeneric(t.BaseProxy, ctx, path, headers, body, &statusResp); err != nil {
		zap.L().Error("Failed to check TikTok post status", zap.Error(err))
		return nil, fmt.Errorf("failed to check post status: %w", err)
	}

	if statusResp.Error.Code != "" && statusResp.Error.Code != "ok" {
		zap.L().Error("TikTok API returned error on status check",
			zap.Any("error", statusResp.Error))
		return nil, fmt.Errorf("TikTok API error: %s - %s", statusResp.Error.Code, statusResp.Error.Message)
	}

	zap.L().Info("TikTok post status fetched",
		zap.String("status", string(statusResp.Data.Status)),
		zap.String("publish_id", publishID))

	return &statusResp, nil
}

// endregion

func NewTikTokProxy(httpClient *http.Client, config *config.AppConfig) iproxies.TikTokProxy {
	tiktokConfig := config.Social.TikTok
	baseURL := fmt.Sprintf("%s/v%s/", tiktokConfig.BaseURL, tiktokConfig.APIVersion)
	return &TikTokProxy{
		BaseProxy: NewBaseProxy(httpClient, baseURL, config),
		config:    &tiktokConfig,
	}
}

// region: ========= Helper Methods =========

func (t *TikTokProxy) ValidateContentRequest(
	ctx context.Context, accessToken string, req *dtos.TikTokVideoInitRequest, creatorInfo *dtos.TikTokCreatorInfo,
) []error {
	errorsSlice := make([]error, 0)

	// Validate Privacy Level
	if !slices.Contains(creatorInfo.PrivacyLevelOptions, req.PostInfo.PrivacyLevel) {
		errorsSlice = append(errorsSlice, fmt.Errorf("invalid privacy level: %s. Should be one of %s",
			req.PostInfo.PrivacyLevel,
			utils.JoinSliceFunc(creatorInfo.PrivacyLevelOptions, ", ", func(p dtos.TikTokPrivacyLevelOption) string {
				return fmt.Sprintf("'%s'", p)
			})))
	}

	// Validate Title
	if req.PostInfo.Title != "" {
		if utils.UTF16RuneCount(req.PostInfo.Title) > 2200 {
			errorsSlice = append(errorsSlice, errors.New("title is too long. Max length is 2200 UTF-16 rune count"))
		}
	}

	// Validate Source Info
	switch req.SourceInfo.Source {
	case dtos.TikTokSourceFileUpload:
		if req.SourceInfo.VideoSize == nil || *req.SourceInfo.VideoSize <= 0 {
			errorsSlice = append(errorsSlice, errors.New("video size is required for FILE_UPLOAD source"))
		}
		if req.SourceInfo.TotalChunkCount == nil || *req.SourceInfo.TotalChunkCount <= 0 {
			errorsSlice = append(errorsSlice, errors.New("total chunk count is required for FILE_UPLOAD source"))
		} else if *req.SourceInfo.TotalChunkCount <= 1 || *req.SourceInfo.TotalChunkCount > 1000 {
			errorsSlice = append(errorsSlice, errors.New("total chunk count must be between 1 and 1000"))
		}
		if req.SourceInfo.ChunkSize == nil || *req.SourceInfo.ChunkSize <= 0 {
			errorsSlice = append(errorsSlice, errors.New("chunk size is required for FILE_UPLOAD source"))
		}

		if *req.SourceInfo.ChunkSize < 5*1024*1024 || *req.SourceInfo.ChunkSize > 64*1024*1024 {
			errorsSlice = append(errorsSlice, errors.New("chunk size must be between 5MB and 64MB"))
		}

	case dtos.TikTokSourcePullFromURL:
		if req.SourceInfo.VideoURL == nil || *req.SourceInfo.VideoURL == "" {
			errorsSlice = append(errorsSlice, errors.New("video URL is required for PULL_FROM_URL source"))
		}

	default:
		errorsSlice = append(errorsSlice, fmt.Errorf("invalid source option: %s", req.SourceInfo.Source))
	}

	if len(req.FileInfoMetadata) > 0 {
		allowedExtensions := []string{".mp4", ".mov", ".webm"}
		if !utils.ContainsSlice(allowedExtensions, req.FileInfoMetadata["extension"]) {
			errorsSlice = append(errorsSlice, fmt.Errorf("invalid video file extension: %s. Allowed extensions are: %s",
				req.FileInfoMetadata["extension"], utils.JoinSliceFunc(allowedExtensions, ", ", func(s string) string { return "'" + s + "'" })))
		}

		allowedCodecs := []string{"h264", "h265", "vp8", "vp9"}
		if !utils.ContainsSlice(allowedCodecs, strings.ToLower(req.FileInfoMetadata["codec"])) {
			errorsSlice = append(errorsSlice, fmt.Errorf("invalid video codec: %s. Allowed codecs are: %s",
				req.FileInfoMetadata["video_codec"], utils.JoinSliceFunc(allowedCodecs, ", ", func(s string) string { return "'" + s + "'" })))
		}

		duration, err := strconv.Atoi(req.FileInfoMetadata["duration_seconds"])
		if err != nil && duration > creatorInfo.MaxVideoPostDuration {
			errorsSlice = append(errorsSlice, fmt.Errorf("video duration is too long. Max duration is %d seconds. Current duration is %d seconds",
				creatorInfo.MaxVideoPostDuration, duration))
		}
	}

	return errorsSlice
}

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
