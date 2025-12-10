package proxies

import (
	"bytes"
	"context"
	"core-backend/config"
	"core-backend/internal/application/dto/dtos"
	"core-backend/internal/application/interfaces/iproxies"
	"core-backend/pkg/utils"
	"fmt"
	"net/http"
	"net/url"
	"strings"

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

	url, err := utils.AddQueryParams("oauth/access_token", map[string]string{
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

	url, err := utils.AddQueryParams("oauth/access_token", map[string]string{
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

	url, err := utils.AddQueryParam("me/accounts", "access_token", userAccessToken)
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

	url, err := utils.AddQueryParams("me", map[string]string{
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

// CreateTextPost implements iproxies.FacebookProxy.
func (f *FacebookProxy) CreateTextPost(ctx context.Context, pageID string, message string, published bool, pageAccessToken string) (*dtos.FacebookPostResponse, error) {
	zap.L().Info("FacebookProxy - CreateTextPost called",
		zap.String("page_id", pageID),
		zap.Bool("published", published))

	path := fmt.Sprintf("%s/feed", pageID)

	body := map[string]any{
		"message":      message,
		"published":    published,
		"access_token": pageAccessToken,
	}

	var postResp dtos.FacebookPostResponse
	if err := PostGeneric(f.BaseProxy, ctx, path, nil, body, &postResp); err != nil {
		zap.L().Error("Failed to create Facebook text post",
			zap.String("page_id", pageID),
			zap.Error(err))
		return nil, fmt.Errorf("failed to create Facebook text post: %w", err)
	}

	zap.L().Info("Facebook text post created successfully",
		zap.String("post_id", postResp.ID))

	return &postResp, nil
}

// region: =============== Publish Single Photo Post Or Upload Multiple Photos for Post ===============

// CreateSinglePhotoPost implements iproxies.FacebookProxy.
func (f *FacebookProxy) CreateSinglePhotoPost(ctx context.Context, pageAccessToken string, publishRequest *dtos.FacebookPhotoPostPublishRequest) (*dtos.FacebookPostResponse, error) {
	zap.L().Info("FacebookProxy - CreatePhotoPost called",
		zap.Any("request", publishRequest))

	path := fmt.Sprintf("/%s/photos", publishRequest.PageID)

	body := map[string]any{
		"url":          publishRequest.URL,
		"caption":      publishRequest.Caption,
		"published":    publishRequest.Published,
		"access_token": pageAccessToken,
	}

	if !publishRequest.Published {
		body["scheduled_publish_time"] = publishRequest.ScheduledPublishTime
		body["unpublished_content_type"] = publishRequest.UnpublishedContentType
	}

	var postResp dtos.FacebookPostResponse
	if err := PostGeneric(f.BaseProxy, ctx, path, nil, body, &postResp); err != nil {
		zap.L().Error("Failed to create Facebook photo post",
			zap.String("page_id", publishRequest.PageID),
			zap.Error(err))
		return nil, fmt.Errorf("failed to create Facebook photo post: %w", err)
	}

	zap.L().Info("Facebook photo post created successfully",
		zap.String("post_id", postResp.ID))

	return &postResp, nil
}

// UploadImage implements iproxies.FacebookProxy.
func (f *FacebookProxy) UploadImage(ctx context.Context, accessToken string, uploadReq *dtos.FacebookImageUploadRequest) (string, error) {
	zap.L().Info("FacebookProxy - UploadImage called",
		zap.Any("request", uploadReq))

	path := fmt.Sprintf("%s/photos", uploadReq.PageID)

	body := map[string]string{
		"access_token": accessToken,
		"url":          uploadReq.URL,
		"published":    "false",
		"temporary":    fmt.Sprintf("%t", uploadReq.Temporary),
	}

	var imageIDResp struct {
		ID string `json:"id"`
	}
	if err := PostGeneric(f.BaseProxy, ctx, path, nil, body, &imageIDResp); err != nil {
		zap.L().Error("Failed to upload image to Facebook",
			zap.Error(err))
		return "", fmt.Errorf("failed to upload image to Facebook: %w", err)
	}

	if imageIDResp.ID == "" {
		zap.L().Error("Facebook image upload returned empty ID")
		return "", fmt.Errorf("facebook image upload returned empty ID")
	}

	return imageIDResp.ID, nil
}

func (f *FacebookProxy) CreateMultiPhotoPost(ctx context.Context, accessToken string, publishRequest *dtos.FacebookPhotoPostPublishRequest) (*dtos.FacebookPostResponse, error) {
	zap.L().Info("FacebookProxy - CreateMultiPhotoPost called",
		zap.Any("request", publishRequest))

	path := fmt.Sprintf("%s/feed", publishRequest.PageID)

	body := map[string]any{
		"caption":      publishRequest.Caption,
		"published":    publishRequest.Published,
		"access_token": accessToken,
	}
	if !publishRequest.Published {
		body["scheduled_publish_time"] = publishRequest.ScheduledPublishTime
		body["unpublished_content_type"] = publishRequest.UnpublishedContentType
	}

	if len(publishRequest.AttachedMedia) > 0 {
		attachedMedias := make([]map[string]string, len(publishRequest.AttachedMedia))
		for i, mediaID := range publishRequest.AttachedMedia {
			attachedMedias[i] = map[string]string{"media_fbid": mediaID}
		}
		body["attached_media"] = attachedMedias
	}

	var postResp dtos.FacebookPostResponse
	if err := PostGeneric(f.BaseProxy, ctx, path, nil, body, &postResp); err != nil {
		zap.L().Error("Failed to create Facebook photo post",
			zap.String("page_id", postResp.PostID),
			zap.Error(err))
		return nil, fmt.Errorf("failed to create Facebook photo post: %w", err)
	}

	return &postResp, nil
}

// endregion

// region: =============== Upload and publish Video To Facebook (3 phases) ===============

// InitVideoUpload implements iproxies.FacebookProxy.
// Step 1: Start upload session using Resumable Upload API
func (f *FacebookProxy) InitVideoUpload(ctx context.Context, fileName string, fileSize int64, userAccessToken string) (string, error) {
	zap.L().Info("FacebookProxy - InitVideoUpload called",
		zap.String("app_id", f.config.ClientID),
		zap.String("file_name", fileName),
		zap.Int64("file_size", fileSize))

	// Use Resumable Upload API: POST /<APP_ID>/uploads
	path, err := utils.AddQueryParams(
		fmt.Sprintf("%s/uploads", f.config.ClientID),
		map[string]any{
			"file_name":    fileName,
			"file_length":  fileSize,
			"file_type":    "video/mp4",
			"access_token": userAccessToken,
		},
	)
	if err != nil {
		zap.L().Error("Failed to add query parameters to Facebook video upload init URL", zap.Error(err))
		return "", fmt.Errorf("failed to add query parameters to Facebook video upload init URL: %w", err)
	}
	path = strings.TrimLeft(path, "/")

	// headers := map[string]string{
	// 	"Authorization": fmt.Sprintf("OAuth %s", userAccessToken),
	// }

	var initResp dtos.FacebookVideoUploadInitResponse
	if err := PostGeneric(f.BaseProxy, ctx, path, nil, nil, &initResp); err != nil {
		zap.L().Error("Failed to initialize Facebook video upload session",
			zap.String("app_id", f.config.ClientID),
			zap.Error(err))
		return "", fmt.Errorf("failed to initialize video upload session: %w", err)
	}

	zap.L().Info("Facebook video upload session initialized",
		zap.String("upload_session_id", initResp.UploadSessionID))

	return initResp.UploadSessionID, nil
}

// UploadVideoChunk implements iproxies.FacebookProxy.
// Step 2: Upload video binary data using Resumable Upload API
func (f *FacebookProxy) UploadVideoChunk(ctx context.Context, uploadSessionID string, videoData *[]byte, fileOffset int64, userAccessToken string) (*string, error) {
	zap.L().Info("FacebookProxy - UploadVideoChunk called",
		zap.String("upload_session_id", uploadSessionID),
		zap.Int64("file_offset", fileOffset),
		zap.Int("chunk_size", len(*videoData)))

	// Use Resumable Upload API: POST /upload:<UPLOAD_SESSION_ID>
	path := fmt.Sprintf("%s%s", f.baseURL, uploadSessionID)

	headers := map[string]string{
		"Authorization": fmt.Sprintf("OAuth %s", userAccessToken),
		"file_offset":   fmt.Sprintf("%d", fileOffset),
	}

	// Convert []byte to io.Reader and send directly
	reader := bytes.NewReader(*videoData)
	var uploadResp dtos.FacebookVideoUploadResponse
	if err := PostGeneric(f.BaseProxy, ctx, path, headers, reader, &uploadResp); err != nil {
		zap.L().Error("Failed to upload Facebook video chunk",
			zap.String("upload_session_id", uploadSessionID),
			zap.Int64("file_offset", fileOffset),
			zap.Error(err))
		return nil, fmt.Errorf("failed to upload video chunk: %w", err)
	}

	zap.L().Info("Facebook video chunk uploaded successfully",
		zap.String("upload_session_id", uploadSessionID),
		zap.Int("file_handle", len(uploadResp.FileHandle)))

	return &uploadResp.FileHandle, nil
}

// PublishVideo implements iproxies.FacebookProxy.
// Step 3: Publish video using file handle from Step 2
func (f *FacebookProxy) PublishVideo(ctx context.Context, fileHandle *string, pageID, title, description, pageAccessToken string) (*dtos.FacebookPostResponse, error) {
	zap.L().Info("FacebookProxy - PublishVideo called",
		zap.String("page_id", pageID),
		zap.Int("file_handle", len(*fileHandle)))

	// Use graph-video.facebook.com for video publish (CRITICAL)
	videoBaseURL := fmt.Sprintf("https://graph-video.facebook.com/v%s/", f.config.APIVersion)
	path := fmt.Sprintf("%s/videos", pageID)

	multipartBody := utils.NewMultipartForm(map[string]string{
		"fbuploader_video_file_chunk": *fileHandle,
		"title":                       title,
		"description":                 description,
		"access_token":                pageAccessToken,
	})

	// Create temporary proxy with video-specific base URL
	videoProxy := NewBaseProxy(f.httpClient, videoBaseURL, f.BaseProxy.config)

	var postResp dtos.FacebookPostResponse
	if err := PostGeneric(videoProxy, ctx, path, nil, multipartBody, &postResp); err != nil {
		zap.L().Error("Failed to publish Facebook video",
			zap.String("page_id", pageID),
			zap.Int("file_handle", len(*fileHandle)),
			zap.Error(err))
		return nil, fmt.Errorf("failed to publish video: %w", err)
	}

	zap.L().Info("Facebook video published successfully",
		zap.String("post_id", postResp.ID))

	return &postResp, nil
}

// endregion

// region: =============== Upload and Publish Video To Facebook V2 ===============

func (f *FacebookProxy) CreateVideoPostFromURL(ctx context.Context, pageAccessToken string, req *dtos.FacebookVideoPostPublishRequest) (string, error) {
	zap.L().Info("FacebookProxy - PublishVideoPostFromURL called",
		zap.Any("request", req))

	if req.PageID == "" || req.FileURL == "" {
		zap.L().Error("Invalid request: PageID and FileURL are required")
		return "", fmt.Errorf("invalid request: PageID and VideoURL are required")
	}

	if !req.Published && req.ScheduledPublishTime == 0 {
		zap.L().Error("Invalid request: ScheduledPublishTime is required for unpublished posts")
		return "", fmt.Errorf("invalid request: ScheduledPublishTime is required for unpublished posts")
	}

	path, err := utils.AddQueryParams(fmt.Sprintf("%s/videos", req.PageID), map[string]string{"access_token": pageAccessToken})

	if err != nil {
		zap.L().Error("Failed to add query parameters to Facebook video post from URL URL",
			zap.Error(err))
		return "", fmt.Errorf("failed to add query parameters to Facebook video post from URL URL: %w", err)
	}

	urlEncodedData := url.Values{}
	urlEncodedData.Add("title", req.Title)
	urlEncodedData.Add("description", req.Description)
	urlEncodedData.Add("file_url", req.FileURL)
	urlEncodedData.Add("published", fmt.Sprintf("%t", req.Published))
	urlEncodedData.Add("social_actions", fmt.Sprintf("%t", req.SocialActions))
	urlEncodedData.Add("secret", fmt.Sprintf("%t", req.Secret))
	if req.UnpublishedContentType != nil && !req.Published {
		urlEncodedData.Add("scheduled_publish_time", fmt.Sprintf("%d", req.ScheduledPublishTime))
		urlEncodedData.Add("unpublished_content_type", string(*req.UnpublishedContentType))
	}

	var postResp dtos.FacebookPostResponse
	if err := PostGeneric(f.BaseProxy, ctx, path, nil, urlEncodedData, &postResp); err != nil {
		zap.L().Error("Failed to publish Facebook video post from URL",
			zap.Error(err))
		return "", fmt.Errorf("failed to publish Facebook video post from URL: %w", err)
	}

	zap.L().Debug("Facebook video post published from URL successfully",
		zap.String("id", postResp.ID))
	return postResp.ID, nil
}

// endregion

// GetUploadStatus implements iproxies.FacebookProxy.
// Check current upload progress for resume capability
func (f *FacebookProxy) GetUploadStatus(ctx context.Context, uploadSessionID string, userAccessToken string) (int64, error) {
	zap.L().Info("FacebookProxy - GetUploadStatus called",
		zap.String("upload_session_id", uploadSessionID))

	// GET /upload:<UPLOAD_SESSION_ID>
	path := fmt.Sprintf("%s%s", f.baseURL, uploadSessionID)

	headers := map[string]string{
		"Authorization": fmt.Sprintf("OAuth %s", userAccessToken),
	}

	var statusResp dtos.FacebookVideoUploadStatusResponse
	if err := GetGeneric(f.BaseProxy, ctx, path, headers, &statusResp); err != nil {
		zap.L().Error("Failed to get Facebook upload status",
			zap.String("upload_session_id", uploadSessionID),
			zap.Error(err))
		return 0, fmt.Errorf("failed to get upload status: %w", err)
	}

	zap.L().Info("Facebook upload status retrieved",
		zap.String("upload_session_id", uploadSessionID),
		zap.Int64("bytes_uploaded", statusResp.FileOffset))

	return statusResp.FileOffset, nil
}

// GetPageInfo implements iproxies.FacebookProxy.
// Retrieves page-level metrics like fan_count, followers_count
func (f *FacebookProxy) GetPageInfo(ctx context.Context, pageID string, accessToken string, fields []string) (*dtos.FacebookPageInfoResponse, error) {
	zap.L().Info("FacebookProxy - GetPageInfo called", zap.String("page_id", pageID), zap.Strings("fields", fields))

	queryParams := map[string]string{
		"access_token": accessToken,
	}
	if len(fields) > 0 {
		queryParams["fields"] = strings.Join(fields, ",")
	}

	url, err := utils.AddQueryParams(pageID, queryParams)
	if err != nil {
		zap.L().Error("Failed to construct URL for getting Facebook page info", zap.Error(err))
		return nil, fmt.Errorf("failed to construct URL: %w", err)
	}

	var pageInfoResp dtos.FacebookPageInfoResponse
	if err := GetGeneric(f.BaseProxy, ctx, url, nil, &pageInfoResp); err != nil {
		zap.L().Error("Failed to get Facebook page info", zap.Error(err))
		return nil, fmt.Errorf("failed to get page info: %w", err)
	}

	return &pageInfoResp, nil
}

// GetPagePosts implements iproxies.FacebookProxy.
// Retrieves paginated list of posts from a Facebook page with engagement metrics
func (f *FacebookProxy) GetPagePosts(ctx context.Context, pageID string, accessToken string, fields string, cursor *string) (*dtos.FacebookPagePostsResponse, error) {
	zap.L().Info("FacebookProxy - GetPagePosts called", zap.String("page_id", pageID))

	queryParams := map[string]string{
		"access_token": accessToken,
	}
	if fields != "" {
		queryParams["fields"] = fields
	}
	if cursor != nil && *cursor != "" {
		queryParams["after"] = *cursor
	}

	url, err := utils.AddQueryParams(fmt.Sprintf("%s/posts", pageID), queryParams)
	if err != nil {
		zap.L().Error("Failed to construct URL for getting Facebook page posts", zap.Error(err))
		return nil, fmt.Errorf("failed to construct URL: %w", err)
	}

	var postsResp dtos.FacebookPagePostsResponse
	if err := GetGeneric(f.BaseProxy, ctx, url, nil, &postsResp); err != nil {
		zap.L().Error("Failed to get Facebook page posts", zap.Error(err))
		return nil, fmt.Errorf("failed to get page posts: %w", err)
	}

	return &postsResp, nil
}

// GetPostMetrics implements iproxies.FacebookProxy.
func (f *FacebookProxy) GetPostMetrics(ctx context.Context, postID string, accessToken string, metrics []string, period dtos.FacebookInsightsPeriod) (*dtos.FacebookPostMetricsResponse, error) {
	zap.L().Info("FacebookProxy - GetPostMetrics called", zap.String("post_id", postID))

	queryParams := map[string]string{
		"metric":       strings.Join(metrics, ","),
		"access_token": accessToken,
	}
	if period != "" {
		queryParams["period"] = string(period)
	}

	url, err := utils.AddQueryParams(fmt.Sprintf("%s/insights", postID), queryParams)
	if err != nil {
		zap.L().Error("Failed to construct URL for getting Facebook post metrics", zap.Error(err))
		return nil, fmt.Errorf("failed to construct URL: %w", err)
	}

	var metricsResp dtos.FacebookPostMetricsResponse
	if err := GetGeneric(f.BaseProxy, ctx, url, nil, &metricsResp); err != nil {
		zap.L().Error("Failed to get Facebook post metrics", zap.Error(err))
		return nil, fmt.Errorf("failed to get post metrics: %w", err)
	}

	return &metricsResp, nil
}

// GetPageInsights implements iproxies.FacebookProxy.
func (f *FacebookProxy) GetPageInsights(ctx context.Context, pageID string, accessToken string, metrics []string, period dtos.FacebookInsightsPeriod) (*dtos.FacebookPageInsightsResponse, error) {
	zap.L().Info("FacebookProxy - GetPageInsights called", zap.String("page_id", pageID))

	queryParams := map[string]string{
		"metric":       strings.Join(metrics, ","),
		"access_token": accessToken,
	}
	if period != "" {
		queryParams["period"] = string(period)
	}

	url, err := utils.AddQueryParams(fmt.Sprintf("%s/insights", pageID), queryParams)
	if err != nil {
		zap.L().Error("Failed to construct URL for getting Facebook page insights", zap.Error(err))
		return nil, fmt.Errorf("failed to construct URL: %w", err)
	}

	var insightsResp dtos.FacebookPageInsightsResponse
	if err := GetGeneric(f.BaseProxy, ctx, url, nil, &insightsResp); err != nil {
		zap.L().Error("Failed to get Facebook page insights", zap.Error(err))
		return nil, fmt.Errorf("failed to get page insights: %w", err)
	}

	return &insightsResp, nil
}

// GetVideoInsights implements iproxies.FacebookProxy.
func (f *FacebookProxy) GetVideoInsights(ctx context.Context, videoID string, accessToken string, metrics []string, period dtos.FacebookInsightsPeriod) (*dtos.FacebookVideoInsightsResponse, error) {
	zap.L().Info("FacebookProxy - GetVideoInsights called", zap.String("video_id", videoID))

	queryParams := map[string]string{
		"access_token": accessToken,
	}
	if len(metrics) > 0 {
		queryParams["metric"] = strings.Join(metrics, ",")
	}
	if period != "" {
		queryParams["period"] = string(period)
	}

	url, err := utils.AddQueryParams(fmt.Sprintf("%s/video_insights", videoID), queryParams)
	if err != nil {
		zap.L().Error("Failed to construct URL for getting Facebook video insights", zap.Error(err))
		return nil, fmt.Errorf("failed to construct URL: %w", err)
	}

	var insightsResp dtos.FacebookVideoInsightsResponse
	if err := GetGeneric(f.BaseProxy, ctx, url, nil, &insightsResp); err != nil {
		zap.L().Error("Failed to get Facebook video insights", zap.Error(err))
		return nil, fmt.Errorf("failed to get video insights: %w", err)
	}

	return &insightsResp, nil
}

func NewFacebookProxy(httpClient *http.Client, config *config.AppConfig) iproxies.FacebookProxy {
	facebookConfig := config.Social.Facebook
	baseURL := fmt.Sprintf("%s/v%s/", facebookConfig.BaseURL, facebookConfig.APIVersion)
	return &FacebookProxy{
		BaseProxy: NewBaseProxy(httpClient, baseURL, config),
		config:    &facebookConfig,
	}
}
