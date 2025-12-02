package iproxies

import (
	"context"
	"core-backend/internal/application/dto/dtos"
)

type TikTokProxy interface {
	// ExchangeCodeForToken exchanges an authorization code for access token + refresh token
	ExchangeCodeForToken(ctx context.Context, code string, redirectURL string) (*dtos.TikTokTokenResponse, error)

	// RefreshAccessToken exchanges a refresh token for a new access token
	RefreshAccessToken(ctx context.Context, refreshToken string) (*dtos.TikTokTokenResponse, error)

	// GetUserProfile retrieves TikTok user profile information using access token
	// Scopes required: user.info.basic
	// The included fields are: open_id, union_id, avatar_url, display_name
	GetUserProfile(ctx context.Context, accessToken string, openID string) (*dtos.TikTokUserProfileResponse, error)

	// GetSystemUserProfile retrieves TikTok user profile information for system users using access token
	// Scopes required: user.info.basic, user.info.profile, user.info.stats
	// The included fields are: open_id, union_id, avatar_url, display_name, bio_description, follower_count,
	// 							following_count, heart_count, video_count
	GetSystemUserProfile(ctx context.Context, accessToken string) (*dtos.TikTokUserProfileResponse, error)

	// region: ======== Content Posting Methods ========

	// GetCreatorInfo retrieves creator information including allowed privacy levels
	GetCreatorInfo(ctx context.Context, accessToken string) (*dtos.TikTokCreatorInfoResponse, error)

	// InitVideoPost initializes a video post upload session
	InitVideoPost(ctx context.Context, accessToken string, req *dtos.TikTokVideoInitRequest) (*dtos.TikTokVideoInitResponse, error)

	// UploadVideoChunk uploads a chunk of video data to TikTok
	UploadVideoChunk(ctx context.Context, uploadURL string, videoData []byte, chunkIndex int, totalChunks int, fileSize int64) error

	// CheckPostStatus checks the status of a video post upload
	CheckPostStatus(ctx context.Context, publishID string, accessToken string) (*dtos.TikTokPostStatusResponse, error)

	// endregion

	// region: ======== Metrics Methods ========

	// GetVideoMetrics retrieves metrics for a specific video
	GetVideoMetrics(ctx context.Context, videoID string, accessToken string) (*dtos.TikTokVideoMetricsResponse, error)

	// endregion

	ValidateContentRequest(
		ctx context.Context, accessToken string, req *dtos.TikTokVideoInitRequest, creatorInfo *dtos.TikTokCreatorInfo,
	) []error
}
