package response

import "github.com/google/uuid"

type LoginResponse struct {
	AccessToken  string   `json:"access_token"`
	RefreshToken string   `json:"refresh_token"`
	ExpiresIn    int64    `json:"expires_in"`
	TokenType    string   `json:"token_type"`
	User         UserInfo `json:"user"`
}

type UserInfo struct {
	ID       uuid.UUID `json:"id"`
	Username string    `json:"username"`
	Email    string    `json:"email"`
	Role     string    `json:"role"`
	IsActive bool      `json:"is_active"`
}

type SessionInfo struct {
	ID                uuid.UUID `json:"id"`
	DeviceFingerprint string    `json:"device_fingerprint"`
	CreatedAt         int64     `json:"created_at"`
	LastUsedAt        int64     `json:"last_used_at"`
	IsActive          bool      `json:"is_active"`
}

type ActiveSessionsResponse struct {
	Sessions []SessionInfo `json:"sessions"`
	Total    int           `json:"total"`
}
