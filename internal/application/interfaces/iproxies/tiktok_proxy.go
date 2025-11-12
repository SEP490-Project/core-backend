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
}
