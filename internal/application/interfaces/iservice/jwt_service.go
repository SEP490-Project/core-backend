// Package iservice defines the interfaces for the services in the application.
package iservice

import (
	"core-backend/internal/domain/model"
	"time"
)

type JWTService interface {
	GenerateAccessToken(userID, username, email, role string, expiration time.Duration) (string, error)
	GenerateRefreshToken() (string, error)
	HashRefreshToken(refreshToken string) string
	ValidateAccessToken(tokenString string) (*model.JWTClaims, error)
	GenerateTokenPair(userID, username, email, role string) (accessToken, refreshToken string, err error)
}
