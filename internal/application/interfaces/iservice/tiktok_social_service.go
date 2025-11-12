package iservice

import (
	"context"
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
}
