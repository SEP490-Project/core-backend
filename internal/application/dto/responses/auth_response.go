package responses

import (
	"time"

	"github.com/google/uuid"
)

// Auth Response DTOs

type LoginResponse struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresIn    int64     `json:"expires_in"`
	User         *UserInfo `json:"user"`
}

type UserInfo struct {
	ID       uuid.UUID `json:"id"`
	Username string    `json:"username"`
	Email    string    `json:"email"`
	Role     string    `json:"role"`
	IsActive bool      `json:"is_active"`
}

type SessionInfo struct {
	ID                uuid.UUID  `json:"id"`
	DeviceFingerprint string     `json:"device_fingerprint"`
	CreatedAt         *time.Time `json:"created_at"`
	LastUsedAt        *time.Time `json:"last_used_at"`
	ExpiryAt          *time.Time `json:"expiry_at"`
	IsRevoked         bool       `json:"is_revoked"`
}

type ActiveSessionsResponse struct {
	Sessions []SessionInfo `json:"sessions"`
	Total    int           `json:"total"`
}

// SignUpResponse represents the sign up response data
type SignUpResponse struct {
	Message string    `json:"message"`
	User    *UserInfo `json:"user"`
}

// LogoutResponse represents the logout response data
type LogoutResponse struct {
	Message string `json:"message"`
}
