package service

import (
	"errors"
	"core-backend/config"
	"core-backend/internal/application/dto"
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
	"core-backend/internal/domain/repository"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	jwtService              *JWTService
	userRepository          repository.UserRepository
	loggedSessionRepository repository.LoggedSessionRepository
}

func NewAuthService(
	jwtService *JWTService,
	userRepository repository.UserRepository,
	loggedSessionRepository repository.LoggedSessionRepository,
) *AuthService {
	return &AuthService{
		jwtService:              jwtService,
		userRepository:          userRepository,
		loggedSessionRepository: loggedSessionRepository,
	}
}

func (s *AuthService) Login(request *dto.LoginRequest) (*dto.LoginResponse, error) {
	// Validate input
	if request.LoginIdentifier == "" || request.Password == "" {
		return nil, errors.New("login identifier and password are required")
	}

	// Get user by username or email
	user, err := s.userRepository.GetByUsernameOrEmail(request.LoginIdentifier)
	if err != nil {
		return nil, errors.New("invalid credentials")
	}
	if user == nil {
		return nil, errors.New("invalid credentials")
	}

	// Check if user is active
	if !user.IsActive {
		return nil, errors.New("account is deactivated")
	}

	// Verify password
	if err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(request.Password)); err != nil {
		return nil, errors.New("invalid credentials")
	}

	// Generate token pair
	accessToken, refreshToken, err := s.jwtService.GenerateTokenPair(
		user.ID.String(),
		user.Username,
		user.Email,
		string(user.Role),
	)
	if err != nil {
		return nil, errors.New("failed to generate tokens")
	}

	// Hash the refresh token for storage
	refreshTokenHash := s.jwtService.HashRefreshToken(refreshToken)

	// Create session record
	refreshTokenExpiry := time.Now().Add(time.Duration(config.GetAppConfig().JWT.RefreshExpiryHours) * time.Hour)
	session := &model.LoggedSession{
		UserID:            user.ID,
		RefreshTokenHash:  refreshTokenHash,
		DeviceFingerprint: request.DeviceFingerprint,
		ExpiryAt:          &refreshTokenExpiry,
		IsRevoked:         false,
	}

	if err := s.loggedSessionRepository.Create(session); err != nil {
		return nil, errors.New("failed to create session")
	}

	// Update user last login
	now := time.Now()
	user.LastLogin = now
	if err := s.userRepository.Update(user); err != nil {
		zap.L().Warn("failed to update user last login", zap.Error(err))
	}

	// Build response
	userInfo := &dto.UserInfo{
		ID:       user.ID,
		Username: user.Username,
		Email:    user.Email,
		Role:     string(user.Role),
		IsActive: user.IsActive,
	}

	return &dto.LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(config.GetAppConfig().JWT.AccessExpiryHours * 3600), // Convert to seconds
		TokenType:    "Bearer",
		User:         userInfo,
	}, nil
}

func (s *AuthService) RefreshToken(request *dto.RefreshTokenRequest) (*dto.LoginResponse, error) {
	if request.RefreshToken == "" {
		return nil, errors.New("refresh token is required")
	}

	refreshTokenHash := s.jwtService.HashRefreshToken(request.RefreshToken)

	session, err := s.loggedSessionRepository.GetByRefreshTokenHash(refreshTokenHash)
	if err != nil {
		return nil, errors.New("invalid refresh token")
	}
	if session == nil {
		return nil, errors.New("invalid refresh token")
	}

	// Check if session is expired or revoked
	if session.IsRevoked || (session.ExpiryAt != nil && time.Now().After(*session.ExpiryAt)) {
		return nil, errors.New("refresh token expired or revoked")
	}

	// Get user details
	user, err := s.userRepository.GetByID(session.UserID)
	if err != nil || user == nil {
		return nil, errors.New("user not found")
	}
	if !user.IsActive {
		return nil, errors.New("account is deactivated")
	}

	// Generate new token pair
	newAccessToken, newRefreshToken, err := s.jwtService.GenerateTokenPair(
		user.ID.String(),
		user.Username,
		user.Email,
		string(user.Role),
	)
	if err != nil {
		return nil, errors.New("failed to generate tokens")
	}

	// Update session with new refresh token hash
	newRefreshTokenHash := s.jwtService.HashRefreshToken(newRefreshToken)
	now := time.Now()
	session.RefreshTokenHash = newRefreshTokenHash
	expiryAt := now.Add(time.Duration(config.GetAppConfig().JWT.RefreshExpiryHours) * time.Hour)
	session.ExpiryAt = &expiryAt
	session.LastUsedAt = &now

	if err := s.loggedSessionRepository.Update(session); err != nil {
		return nil, errors.New("failed to update session")
	}

	// Build response
	userInfo := &dto.UserInfo{
		ID:       user.ID,
		Username: user.Username,
		Email:    user.Email,
		Role:     string(user.Role),
		IsActive: user.IsActive,
	}

	return &dto.LoginResponse{
		AccessToken:  newAccessToken,
		RefreshToken: newRefreshToken,
		ExpiresIn:    int64(config.GetAppConfig().JWT.AccessExpiryHours * 3600), // Convert to seconds
		TokenType:    "Bearer",
		User:         userInfo,
	}, nil
}

func (s *AuthService) SignUp(request *dto.SignUpRequest) (*dto.SignUpResponse, error) {
	// Validate input
	if request.Username == "" || request.Email == "" || request.Password == "" {
		return nil, errors.New("username, email, and password are required")
	}

	// Check if username exists
	if exists, err := s.userRepository.IsUsernameExists(request.Username); err != nil {
		return nil, errors.New("failed to check username availability")
	} else if exists {
		return nil, errors.New("username already exists")
	}

	// Check if email exists
	if exists, err := s.userRepository.IsEmailExists(request.Email); err != nil {
		return nil, errors.New("failed to check email availability")
	} else if exists {
		return nil, errors.New("email already exists")
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(request.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, errors.New("failed to hash password")
	}

	// Create user
	user := &model.User{
		Username:     request.Username,
		Email:        request.Email,
		PasswordHash: string(hashedPassword),
		Role:         enum.RoleCustomer, // Default role
		IsActive:     true,
	}

	if err := s.userRepository.Create(user); err != nil {
		return nil, errors.New("failed to create user")
	}

	// Build response
	userInfo := &dto.UserInfo{
		ID:       user.ID,
		Username: user.Username,
		Email:    user.Email,
		Role:     string(user.Role),
		IsActive: user.IsActive,
	}

	return &dto.SignUpResponse{
		Message: "User created successfully",
		User:    userInfo,
	}, nil
}

func (s *AuthService) Logout(request *dto.LogoutRequest) (*dto.LogoutResponse, error) {
	if request.RefreshToken == "" {
		return nil, errors.New("refresh token is required")
	}

	refreshTokenHash := s.jwtService.HashRefreshToken(request.RefreshToken)
	session, err := s.loggedSessionRepository.GetByRefreshTokenHash(refreshTokenHash)
	if err != nil || session == nil {
		return nil, errors.New("session not found")
	}

	if err := s.loggedSessionRepository.RevokeSession(session.ID); err != nil {
		return nil, errors.New("failed to revoke session")
	}

	return &dto.LogoutResponse{
		Message: "Logged out successfully",
	}, nil
}

func (s *AuthService) LogoutAll(userID uuid.UUID) (*dto.LogoutResponse, error) {
	if err := s.loggedSessionRepository.RevokeAllUserSessions(userID); err != nil {
		return nil, errors.New("failed to revoke all sessions")
	}

	return &dto.LogoutResponse{
		Message: "All sessions revoked successfully",
	}, nil
}

func (s *AuthService) GetActiveSessions(userID uuid.UUID) ([]*dto.SessionInfo, error) {
	sessions, err := s.loggedSessionRepository.GetActiveSessionsByUserID(userID)
	if err != nil {
		return nil, errors.New("failed to get active sessions")
	}

	sessionInfos := make([]*dto.SessionInfo, len(sessions))
	for i, session := range sessions {
		sessionInfos[i] = &dto.SessionInfo{
			ID:                session.ID,
			DeviceFingerprint: session.DeviceFingerprint,
			CreatedAt:         session.CreatedAt,
			LastUsedAt:        session.LastUsedAt,
			ExpiryAt:          session.ExpiryAt,
			IsRevoked:         session.IsRevoked,
		}
	}

	return sessionInfos, nil
}

func (s *AuthService) RevokeSession(sessionID uuid.UUID) (*dto.LogoutResponse, error) {
	if err := s.loggedSessionRepository.RevokeSession(sessionID); err != nil {
		return nil, errors.New("failed to revoke session")
	}

	return &dto.LogoutResponse{
		Message: "Session revoked successfully",
	}, nil
}
