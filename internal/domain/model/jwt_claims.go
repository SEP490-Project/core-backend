package model

import "github.com/golang-jwt/jwt/v5"

type JWTClaims struct {
	UserID    string `json:"user_id"`
	SessionID string `json:"session_id,omitempty"`
	Roles     string `json:"roles"`
	Username  string `json:"username"`
	Email     string `json:"email"`
	jwt.RegisteredClaims
}
