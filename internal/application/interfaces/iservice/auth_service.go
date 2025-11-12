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

	// Password management
	ForgotPassword(ctx context.Context, request *requests.ForgotPasswordRequest) (string, error)
	ResetPassword(ctx context.Context, request *requests.ResetPasswordRequest) (string, error)
	ChangePassword(ctx context.Context, request *requests.ChangePasswordRequest) (string, error)
}
