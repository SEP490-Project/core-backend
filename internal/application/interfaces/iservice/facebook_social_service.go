package iservice

import (
	"context"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/irepository"
)

type FacebookSocialService interface {
	// HandleOAuthLogin handles admin OAuth flow for connecting Facebook pages
	HandleOAuthLogin(ctx context.Context, uow irepository.UnitOfWork, code string, redirectURL string, deviceFingerprint string) (*responses.LoginResponse, error)

	// HandleRefreshPageAccessToken refreshes the page access token for admin
	HandleRefreshPageAccessToken(ctx context.Context, uow irepository.UnitOfWork, successReq *requests.FacebookOAuthSuccessRequest) error

	// IsFacebookTokenNearExpiry checks if the token is near expiry
	IsFacebookTokenNearExpiry(ctx context.Context, accessToken string) (bool, error)
}
