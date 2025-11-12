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
}
