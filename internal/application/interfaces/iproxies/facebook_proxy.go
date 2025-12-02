package iproxies

import (
	"context"
	"core-backend/internal/application/dto/dtos"
)

type FacebookProxy interface {
	// region: ======== OAuth Methods ========

	// ExchangeCodeForUserAccessToken exchanges an authorization code for a user access token.
	ExchangeCodeForUserAccessToken(ctx context.Context, code string, redirectURL string) (*dtos.FacebookAccessTokenResponse, error)

	// ExchangeUserAccessTokenForLongLivedToken exchanges a short-lived user access token for a long-lived token.
	ExchangeUserAccessTokenForLongLivedToken(ctx context.Context, userAccessToken string) (*dtos.FacebookAccessTokenResponse, error)

	// GetAccountPageInfo retrieves the Facebook account information using the provided user access token.
	// This call to the path '/me/acccounts' to get Page Access Tokens.
	GetAccountPageInfo(ctx context.Context, userAccessToken string) (*dtos.FacebookAccountInfoResponse, error)

	// endregion

	// GetUserProfile retrieves the Facebook user's basic profile information.
	// The included fields are: id, name, email, and picture.
	GetUserProfile(ctx context.Context, userAccessToken string) (*dtos.FacebookUserProfileResponse, error)

	// region: ======== Content Posting Methods ========

	// CreateTextPost creates a text post on a Facebook Page
	CreateTextPost(ctx context.Context, pageID string, message string, published bool, pageAccessToken string) (*dtos.FacebookPostResponse, error)

	// region: ======== Publish Single Photo Post Or Upload Multiple Photos for Post ========

	// CreatePhotoPost creates a photo post on a Facebook Page
	CreateSinglePhotoPost(ctx context.Context, pageAccessToken string, publishRequest *dtos.FacebookPhotoPostPublishRequest) (*dtos.FacebookPostResponse, error)

	UploadImage(ctx context.Context, accessToken string, uploadReq *dtos.FacebookImageUploadRequest) (string, error)

	CreateMultiPhotoPost(ctx context.Context, accessToken string, publishRequest *dtos.FacebookPhotoPostPublishRequest) (*dtos.FacebookPostResponse, error)

	// endregion

	// region: ======== Upload and publish Video To Facebook (3 phases) ========

	// InitVideoUpload initializes a resumable video upload session using Resumable Upload API (Step 1 of 3)
	// Returns upload session ID in format "upload:<UPLOAD_SESSION_ID>"
	InitVideoUpload(ctx context.Context, fileName string, fileSize int64, userAccessToken string) (string, error)

	// UploadVideoChunk uploads video binary data using Resumable Upload API (Step 2 of 3)
	// Returns file handle (e.g., "2:c2FtcGxl...") to use in PublishVideo
	UploadVideoChunk(ctx context.Context, uploadSessionID string, videoData *[]byte, fileOffset int64, userAccessToken string) (*string, error)

	// PublishVideo publishes an uploaded video using the file handle from UploadVideoChunk (Step 3 of 3)
	PublishVideo(ctx context.Context, fileHandle *string, pageID, title, description, pageAccessToken string) (*dtos.FacebookPostResponse, error)

	// endregion

	// region: ======== Publish Video Post From URL ========

	CreateVideoPostFromURL(ctx context.Context, pageAccessToken string, req *dtos.FacebookVideoPostPublishRequest) (string, error)

	// endregion

	// GetUploadStatus checks the current upload status and returns bytes uploaded (for resume capability)
	GetUploadStatus(ctx context.Context, uploadSessionID string, userAccessToken string) (int64, error)

	// endregion

	// region: ======== Metrics Methods ========

	// GetPostMetrics retrieves metrics for a specific post
	GetPostMetrics(ctx context.Context, postID string, accessToken string, metrics []string, period dtos.FacebookInsightsPeriod) (*dtos.FacebookPostMetricsResponse, error)

	// GetPageInsights retrieves insights for a page
	GetPageInsights(ctx context.Context, pageID string, accessToken string, metrics []string, period dtos.FacebookInsightsPeriod) (*dtos.FacebookPageInsightsResponse, error)

	// GetVideoInsights retrieves insights for a video
	GetVideoInsights(ctx context.Context, videoID string, accessToken string, metrics []string, period dtos.FacebookInsightsPeriod) (*dtos.FacebookVideoInsightsResponse, error)

	// endregion
}
