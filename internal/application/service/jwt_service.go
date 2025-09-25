package service

import (
	"core-backend/config"
	"core-backend/internal/application/interfaces/iservice"
	"core-backend/internal/domain/model"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
)

type JWTService struct {
	publicKey  *rsa.PublicKey
	privateKey *rsa.PrivateKey
}

func NewJwtService() iservice.JWTService {
	jwtConfig := config.GetAppConfig().JWT

	if jwtConfig.GetPublicKey() == nil || jwtConfig.GetPrivateKey() == nil {
		panic("Failed to load RSA keys from configuration")
	}

	return &JWTService{
		publicKey:  jwtConfig.GetPublicKey(),
		privateKey: jwtConfig.GetPrivateKey(),
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
	accessToken, err = s.GenerateAccessToken(userID, username, email, role, 15*time.Minute)
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

func GenerateKeyPair(privateKeyPath, publicKeyPath string, keySize int) error {
	zap.L().Info("Generating RSA key pair",
		zap.String("private_key_path", privateKeyPath),
		zap.String("public_key_path", publicKeyPath),
		zap.Int("key_size", keySize))

	// Generate private key
	privateKey, err := rsa.GenerateKey(rand.Reader, keySize)
	if err != nil {
		zap.L().Error("Failed to generate RSA private key",
			zap.Int("key_size", keySize),
			zap.Error(err))
		return fmt.Errorf("failed to generate private key: %w", err)
	}

	// Save private key
	privateKeyBytes := x509.MarshalPKCS1PrivateKey(privateKey)
	privateKeyPEM := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privateKeyBytes,
	}

	privateKeyFile, err := os.Create(privateKeyPath)
	if err != nil {
		zap.L().Error("Failed to create private key file",
			zap.String("private_key_path", privateKeyPath),
			zap.Error(err))
		return fmt.Errorf("failed to create private key file: %w", err)
	}
	defer privateKeyFile.Close()

	if err = pem.Encode(privateKeyFile, privateKeyPEM); err != nil {
		zap.L().Error("Failed to write private key to file",
			zap.String("private_key_path", privateKeyPath),
			zap.Error(err))
		return fmt.Errorf("failed to write private key: %w", err)
	}

	zap.L().Debug("Successfully saved private key",
		zap.String("private_key_path", privateKeyPath))

	// Save public key
	publicKeyBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		zap.L().Error("Failed to marshal public key",
			zap.Error(err))
		return fmt.Errorf("failed to marshal public key: %w", err)
	}

	publicKeyPEM := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: publicKeyBytes,
	}

	publicKeyFile, err := os.Create(publicKeyPath)
	if err != nil {
		zap.L().Error("Failed to create public key file",
			zap.String("public_key_path", publicKeyPath),
			zap.Error(err))
		return fmt.Errorf("failed to create public key file: %w", err)
	}
	defer publicKeyFile.Close()

	if err := pem.Encode(publicKeyFile, publicKeyPEM); err != nil {
		zap.L().Error("Failed to write public key to file",
			zap.String("public_key_path", publicKeyPath),
			zap.Error(err))
		return fmt.Errorf("failed to write public key: %w", err)
	}

	zap.L().Info("Successfully generated and saved RSA key pair",
		zap.String("private_key_path", privateKeyPath),
		zap.String("public_key_path", publicKeyPath),
		zap.Int("key_size", keySize))

	return nil
}
