package iservice

import (
	"context"
	"core-backend/internal/application/dto/dtos"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/irepository"
)

type TikTokSocialService interface {
	// HandleOAuthLogin processes the TikTok OAuth callback and stores tokens for admin
	HandleOAuthLogin(ctx context.Context, uow irepository.UnitOfWork, code string, redirectURL string, deviceFingerprint string) (*responses.LoginResponse, error)

	// HandleRefreshAccessToken refreshes the TikTok access token using the refresh token
	HandleRefreshAccessToken(ctx context.Context, uow irepository.UnitOfWork, successReq *requests.TikTokOAuthSuccessRequest) error

	// IsTikTokTokenNearExpiry checks if the stored TikTok token is expiring soon
	IsTikTokTokenNearExpiry(ctx context.Context, accessToken string) (bool, error)

	GetTikTokCreatorInfo(ctx context.Context) (*dtos.TikTokCreatorInfoResponse, error)

	GetTikTokSystemUserProfile(ctx context.Context) (*dtos.TikTokUserProfileResponse, error)

	// GetTikTokAccessToken retrieves the TikTok access token for the system user
	// This method will automatically refresh access token if it is expired and refresh token is available
	GetTikTokAccessToken(ctx context.Context) (string, error)
}
