// Package dto contains data transfer objects for the application layer
package dto

import (
	"github.com/google/uuid"
	"time"
)

// LoginRequest represents the login request data
type LoginRequest struct {
	LoginIdentifier   string `json:"login_identifier" validate:"required"`
	Password          string `json:"password" validate:"required,min=6"`
	DeviceFingerprint string `json:"device_fingerprint,omitempty"`
}

// SignUpRequest represents the sign up request data
type SignUpRequest struct {
	Username string `json:"username" validate:"required,min=3,max=50"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=6"`
}

// RefreshTokenRequest represents the refresh token request data
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

// LoginResponse represents the login response data
type LoginResponse struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresIn    int64     `json:"expires_in"`
	TokenType    string    `json:"token_type"`
	User         *UserInfo `json:"user"`
}

// UserInfo represents basic user information returned in auth responses
type UserInfo struct {
	ID       uuid.UUID `json:"id"`
	Username string    `json:"username"`
	Email    string    `json:"email"`
	Role     string    `json:"role"`
	IsActive bool      `json:"is_active"`
}

// SignUpResponse represents the sign up response data
type SignUpResponse struct {
	Message string    `json:"message"`
	User    *UserInfo `json:"user"`
}

// SessionInfo represents session information
type SessionInfo struct {
	ID                uuid.UUID  `json:"id"`
	DeviceFingerprint string     `json:"device_fingerprint"`
	CreatedAt         *time.Time `json:"created_at"`
	LastUsedAt        *time.Time `json:"last_used_at"`
	ExpiryAt          *time.Time `json:"expiry_at"`
	IsRevoked         bool       `json:"is_revoked"`
}

// LogoutRequest represents the logout request data
type LogoutRequest struct {
	RefreshToken string `json:"refresh_token,omitempty"`
}

// LogoutResponse represents the logout response data
type LogoutResponse struct {
	Message string `json:"message"`
}
