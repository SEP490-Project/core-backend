package iservice

import (
	"context"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"

	"github.com/google/uuid"
)

type AuthService interface {
	Login(ctx context.Context, request *requests.LoginRequest, deviceFingerprint string) (*responses.LoginResponse, error)
	RefreshToken(ctx context.Context, request *requests.RefreshTokenRequest, deviceFingerprint string) (*responses.LoginResponse, error)
	SignUp(ctx context.Context, request *requests.SignUpRequest) (*responses.SignUpResponse, error)
	Logout(ctx context.Context, request *requests.LogoutRequest) (*responses.LogoutResponse, error)
	LogoutAll(ctx context.Context, userID uuid.UUID) (*responses.LogoutResponse, error)
	GetActiveSessions(ctx context.Context, userID uuid.UUID) ([]*responses.SessionInfo, error)
	RevokeSession(ctx context.Context, sessionID uuid.UUID) (*responses.LogoutResponse, error)
}
