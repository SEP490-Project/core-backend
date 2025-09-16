// Package core contains core domain models
package core

import "github.com/golang-jwt/jwt/v5"

type JWTClaims struct {
	UserID   string `json:"user_id"`
	Roles    string `json:"roles"`
	Username string `json:"username"`
	Email    string `json:"email"`
	jwt.RegisteredClaims
}
