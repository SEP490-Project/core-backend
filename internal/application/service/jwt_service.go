package service

import (
	"core-backend/config"
	"core-backend/internal/domain/model"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
)

type JWTService struct {
	publicKey  *rsa.PublicKey
	privateKey *rsa.PrivateKey
}

func NewJwtService(config *config.AppConfig) *JWTService {
	if config.GetPublicKey() == nil || config.GetPrivateKey() == nil {
		panic("Failed to load RSA keys from configuration")
	}

	return &JWTService{
		publicKey:  config.GetPublicKey(),
		privateKey: config.GetPrivateKey(),
	}
}

func (s *JWTService) GenerateAccessToken(userID, username, email, role string, expiration time.Duration) (string, error) {
	now := time.Now()
	claims := &model.JWTClaims{
		UserID:   userID,
		Roles:    role,
		Username: username,
		Email:    email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(expiration)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Subject:   username,
			Issuer:    config.GetAppConfig().Server.ServiceName,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	return token.SignedString(s.privateKey)
}

func (s *JWTService) GenerateRefreshToken() (string, error) {
	// Generate 32 random bytes
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate refresh token: %w", err)
	}

	// Return base64 encoded string
	return base64.URLEncoding.EncodeToString(bytes), nil
}

func (s *JWTService) HashRefreshToken(refreshToken string) string {
	hash := sha256.Sum256([]byte(refreshToken))
	return hex.EncodeToString(hash[:])
}

func (s *JWTService) ValidateAccessToken(tokenString string) (*model.JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &model.JWTClaims{}, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.publicKey, nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	claims, ok := token.Claims.(*model.JWTClaims)
	if !ok {
		return nil, fmt.Errorf("invalid claims type")
	}

	return claims, nil
}

func (s *JWTService) GenerateTokenPair(userID, username, email, role string) (accessToken, refreshToken string, err error) {
	zap.L().Debug("Generating token pair for user",
		zap.String("user_id", userID),
		zap.String("username", username),
		zap.String("email", email),
		zap.String("role", string(role)))

	// Generate access token (short-lived)
	//accessToken, err = s.GenerateAccessToken(userID, username, email, role, 15*time.Minute)
	accessToken, err = s.GenerateAccessToken(userID, username, email, role, 15*time.Hour)
	if err != nil {
		zap.L().Error("Failed to generate access token",
			zap.String("user_id", userID),
			zap.String("username", username),
			zap.Error(err))
		return "", "", fmt.Errorf("failed to generate access token: %w", err)
	}

	// Generate refresh token (long-lived)
	refreshToken, err = s.GenerateRefreshToken()
	if err != nil {
		zap.L().Error("Failed to generate refresh token",
			zap.String("user_id", userID),
			zap.String("username", username),
			zap.Error(err))
		return "", "", fmt.Errorf("failed to generate refresh token: %w", err)
	}

	zap.L().Info("Successfully generated token pair for user",
		zap.String("user_id", userID),
		zap.String("username", username))

	return accessToken, refreshToken, nil
}
