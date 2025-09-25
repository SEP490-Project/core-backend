package iservice

import (
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"

	"github.com/google/uuid"
)

type AuthService interface {
	Login(request *requests.LoginRequest) (*responses.LoginResponse, error)
	RefreshToken(request *requests.RefreshTokenRequest) (*responses.LoginResponse, error)
	SignUp(request *requests.SignUpRequest) (*responses.SignUpResponse, error)
	Logout(request *requests.LogoutRequest) (*responses.LogoutResponse, error)
	LogoutAll(userID uuid.UUID) (*responses.LogoutResponse, error)
	GetActiveSessions(userID uuid.UUID) ([]*responses.SessionInfo, error)
	RevokeSession(sessionID uuid.UUID) (*responses.LogoutResponse, error)
}
