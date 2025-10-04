package service

import (
	"context"
	"core-backend/config"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/application/interfaces/iservice"
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
	"errors"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AuthService struct {
	jwtService              iservice.JWTService
	userRepository          irepository.GenericRepository[model.User]
	loggedSessionRepository irepository.GenericRepository[model.LoggedSession]
}

func NewAuthService(
	jwtService iservice.JWTService,
	userRepository irepository.GenericRepository[model.User],
	loggedSessionRepository irepository.GenericRepository[model.LoggedSession],
) *AuthService {
	return &AuthService{
		jwtService:              jwtService,
		userRepository:          userRepository,
		loggedSessionRepository: loggedSessionRepository,
	}
}

func (s *AuthService) Login(ctx context.Context, request *requests.LoginRequest, deviceFingerprint string) (*responses.LoginResponse, error) {
	zap.L().Info("User login attempt",
		zap.String("login_identifier", request.LoginIdentifier),
		zap.String("device_fingerprint", deviceFingerprint))

	// Validate input
	if request.LoginIdentifier == "" || request.Password == "" {
		zap.L().Debug("Login validation failed: missing credentials")
		return nil, errors.New("login identifier and password are required")
	}

	// Get user by username or email
	filters := func(db *gorm.DB) *gorm.DB {
		return db.Where("username = ? OR email = ?", request.LoginIdentifier, request.LoginIdentifier)
	}
	user, err := s.userRepository.GetByCondition(ctx, filters, nil)
	if err != nil {
		zap.L().Error("Failed to retrieve user during login",
			zap.String("login_identifier", request.LoginIdentifier),
			zap.Error(err))
		return nil, errors.New("invalid credentials")
	}
	if user == nil {
		zap.L().Debug("Login failed: user not found",
			zap.String("login_identifier", request.LoginIdentifier))
		return nil, errors.New("invalid credentials")
	}

	zap.L().Debug("User found for login",
		zap.String("user_id", user.ID.String()),
		zap.String("username", user.Username))

	// Check if user is active
	if !user.IsActive {
		zap.L().Debug("Login failed: account deactivated",
			zap.String("user_id", user.ID.String()),
			zap.String("username", user.Username))
		return nil, errors.New("account is deactivated")
	}

	// Verify password
	if err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(request.Password)); err != nil {
		zap.L().Debug("Login failed: invalid password",
			zap.String("user_id", user.ID.String()),
			zap.String("username", user.Username))
		return nil, errors.New("invalid credentials")
	}

	zap.L().Debug("Password verification successful",
		zap.String("user_id", user.ID.String()))

	// Generate token pair
	accessToken, refreshToken, err := s.jwtService.GenerateTokenPair(
		user.ID.String(),
		user.Username,
		user.Email,
		string(user.Role),
	)
	if err != nil {
		zap.L().Error("Failed to generate token pair during login",
			zap.String("user_id", user.ID.String()),
			zap.Error(err))
		return nil, errors.New("failed to generate tokens")
	}

	zap.L().Debug("Token pair generated successfully",
		zap.String("user_id", user.ID.String()))

	// Hash the refresh token for storage
	refreshTokenHash := s.jwtService.HashRefreshToken(refreshToken)

	// Create session record
	refreshTokenExpiry := time.Now().Add(time.Duration(config.GetAppConfig().JWT.RefreshExpiryHours) * time.Hour)
	session := &model.LoggedSession{
		UserID:            user.ID,
		RefreshTokenHash:  refreshTokenHash,
		DeviceFingerprint: deviceFingerprint,
		ExpiryAt:          &refreshTokenExpiry,
		IsRevoked:         false,
	}

	if err := s.loggedSessionRepository.Add(ctx, session); err != nil {
		zap.L().Error("Failed to create session during login",
			zap.String("user_id", user.ID.String()),
			zap.Error(err))
		return nil, errors.New("failed to create session")
	}

	zap.L().Info("Session created successfully",
		zap.String("user_id", user.ID.String()),
		zap.String("session_id", session.ID.String()))

	// Update user last login
	now := time.Now()
	user.LastLogin = &now
	if err := s.userRepository.Update(ctx, user); err != nil {
		zap.L().Warn("Failed to update user last login",
			zap.String("user_id", user.ID.String()),
			zap.Error(err))
	}

	// Build response
	userInfo := &responses.UserInfo{
		ID:       user.ID,
		Username: user.Username,
		Email:    user.Email,
		Role:     string(user.Role),
		IsActive: user.IsActive,
	}

	zap.L().Info("User login successful",
		zap.String("user_id", user.ID.String()),
		zap.String("username", user.Username),
		zap.String("role", string(user.Role)))

	return &responses.LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(config.GetAppConfig().JWT.AccessExpiryHours * 3600), // Convert to seconds
		User:         userInfo,
	}, nil
}

func (s *AuthService) RefreshToken(ctx context.Context, request *requests.RefreshTokenRequest, deviceFingerprint string) (*responses.LoginResponse, error) {
	zap.L().Debug("Token refresh attempt")

	if request.RefreshToken == "" {
		zap.L().Debug("Token refresh failed: missing refresh token")
		return nil, errors.New("refresh token is required")
	}

	refreshTokenHash := s.jwtService.HashRefreshToken(request.RefreshToken)

	query := func(db *gorm.DB) *gorm.DB {
		return db.Where("refresh_token_hash = ?", refreshTokenHash)
	}
	session, err := s.loggedSessionRepository.GetByCondition(ctx, query, nil)
	if err != nil {
		zap.L().Error("Failed to retrieve session during token refresh",
			zap.Error(err))
		return nil, errors.New("invalid refresh token")
	}
	if session == nil {
		zap.L().Debug("Token refresh failed: session not found")
		return nil, errors.New("invalid refresh token")
	}

	zap.L().Debug("Session found for token refresh",
		zap.String("session_id", session.ID.String()),
		zap.String("user_id", session.UserID.String()))

	// Check if session is expired or revoked
	if session.IsRevoked || (session.ExpiryAt != nil && time.Now().After(*session.ExpiryAt)) {
		zap.L().Debug("Token refresh failed: session expired or revoked",
			zap.String("session_id", session.ID.String()),
			zap.Bool("is_revoked", session.IsRevoked))
		return nil, errors.New("refresh token expired or revoked")
	}

	if session.DeviceFingerprint != deviceFingerprint {
		zap.L().Warn("Refresh token used from different device",
			zap.String("session_id", session.ID.String()),
			zap.String("expected", session.DeviceFingerprint),
			zap.String("actual", deviceFingerprint))

		// Optionally revoke all sessions for security
		return nil, errors.New("invalid device fingerprint")
	}

	// Get user details
	user, err := s.userRepository.GetByID(ctx, session.UserID, nil)
	if err != nil || user == nil {
		zap.L().Error("Failed to retrieve user during token refresh",
			zap.String("user_id", session.UserID.String()),
			zap.Error(err))
		return nil, errors.New("user not found")
	}
	if !user.IsActive {
		zap.L().Debug("Token refresh failed: user account deactivated",
			zap.String("user_id", user.ID.String()))
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
		zap.L().Error("Failed to generate new token pair during refresh",
			zap.String("user_id", user.ID.String()),
			zap.Error(err))
		return nil, errors.New("failed to generate tokens")
	}

	zap.L().Debug("New token pair generated successfully",
		zap.String("user_id", user.ID.String()))

	// Update session with new refresh token hash
	newRefreshTokenHash := s.jwtService.HashRefreshToken(newRefreshToken)
	now := time.Now()
	session.RefreshTokenHash = newRefreshTokenHash
	expiryAt := now.Add(time.Duration(config.GetAppConfig().JWT.RefreshExpiryHours) * time.Hour)
	session.ExpiryAt = &expiryAt
	session.LastUsedAt = &now

	if err := s.loggedSessionRepository.Update(ctx, session); err != nil {
		zap.L().Error("Failed to update session during token refresh",
			zap.String("session_id", session.ID.String()),
			zap.Error(err))
		return nil, errors.New("failed to update session")
	}

	// Build response
	userInfo := &responses.UserInfo{
		ID:       user.ID,
		Username: user.Username,
		Email:    user.Email,
		Role:     string(user.Role),
		IsActive: user.IsActive,
	}

	zap.L().Info("Token refresh successful",
		zap.String("user_id", user.ID.String()),
		zap.String("session_id", session.ID.String()))

	return &responses.LoginResponse{
		AccessToken:  newAccessToken,
		RefreshToken: newRefreshToken,
		ExpiresIn:    int64(config.GetAppConfig().JWT.AccessExpiryHours * 3600), // Convert to seconds
		User:         userInfo,
	}, nil
}

func (s *AuthService) SignUp(ctx context.Context, request *requests.SignUpRequest) (*responses.SignUpResponse, error) {
	zap.L().Info("User signup attempt",
		zap.String("username", request.Username),
		zap.String("email", request.Email),
		zap.String("full_name", request.FullName))

	// Validate input
	if request.Username == "" || request.Email == "" || request.Password == "" {
		zap.L().Debug("Signup validation failed: missing required fields")
		return nil, errors.New("username, email, and password are required")
	}

	// Check if username exists
	query := func(db *gorm.DB) *gorm.DB {
		return db.Where("username = ?", request.Username)
	}
	if exists, err := s.userRepository.Exists(ctx, query); err != nil {
		zap.L().Error("Failed to check username availability during signup",
			zap.String("username", request.Username),
			zap.Error(err))
		return nil, errors.New("failed to check username availability")
	} else if exists {
		zap.L().Debug("Signup failed: username already exists",
			zap.String("username", request.Username))
		return nil, errors.New("username already exists")
	}

	// Check if email exists
	query = func(db *gorm.DB) *gorm.DB {
		return db.Where("email = ?", request.Email)
	}
	if exists, err := s.userRepository.Exists(ctx, query); err != nil {
		zap.L().Error("Failed to check email availability during signup",
			zap.String("email", request.Email),
			zap.Error(err))
		return nil, errors.New("failed to check email availability")
	} else if exists {
		zap.L().Debug("Signup failed: email already exists",
			zap.String("email", request.Email))
		return nil, errors.New("email already exists")
	}

	zap.L().Debug("Username and email availability verified",
		zap.String("username", request.Username),
		zap.String("email", request.Email))

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(request.Password), bcrypt.DefaultCost)
	if err != nil {
		zap.L().Error("Failed to hash password during signup",
			zap.String("username", request.Username),
			zap.Error(err))
		return nil, errors.New("failed to hash password")
	}

	zap.L().Debug("Password hashed successfully",
		zap.String("username", request.Username))

	// Create user
	user := &model.User{
		Username:     request.Username,
		Email:        request.Email,
		PasswordHash: string(hashedPassword),
		Role:         enum.UserRoleCustomer, // Default role
		FullName:     request.FullName,
		IsActive:     true,
	}

	if err := s.userRepository.Add(ctx, user); err != nil {
		zap.L().Error("Failed to create user during signup",
			zap.String("username", request.Username),
			zap.String("full_name", request.FullName),
			zap.String("email", request.Email),
			zap.Error(err))
		return nil, errors.New("failed to create user")
	}

	// Build response
	userInfo := &responses.UserInfo{
		ID:       user.ID,
		Username: user.Username,
		Email:    user.Email,
		Role:     string(user.Role),
		IsActive: user.IsActive,
	}

	zap.L().Info("User signup successful",
		zap.String("user_id", user.ID.String()),
		zap.String("username", user.Username),
		zap.String("email", user.Email),
		zap.String("role", string(user.Role)))

	return &responses.SignUpResponse{
		Message: "User created successfully",
		User:    userInfo,
	}, nil
}

func (s *AuthService) Logout(ctx context.Context, request *requests.LogoutRequest) (*responses.LogoutResponse, error) {
	zap.L().Debug("User logout attempt")

	if request.RefreshToken == "" {
		zap.L().Debug("Logout failed: missing refresh token")
		return nil, errors.New("refresh token is required")
	}

	refreshTokenHash := s.jwtService.HashRefreshToken(request.RefreshToken)
	query := func(db *gorm.DB) *gorm.DB {
		return db.Where("refresh_token_hash = ?", refreshTokenHash)
	}
	session, err := s.loggedSessionRepository.GetByCondition(ctx, query, nil)
	if err != nil || session == nil {
		zap.L().Debug("Logout failed: session not found",
			zap.Error(err))
		return nil, errors.New("session not found")
	}

	zap.L().Debug("Session found for logout",
		zap.String("session_id", session.ID.String()),
		zap.String("user_id", session.UserID.String()))

	conditions := func(db *gorm.DB) *gorm.DB {
		return db.Where("id = ? AND is_revoked = ?", session.ID, false)
	}
	if err := s.loggedSessionRepository.UpdateByCondition(ctx, conditions, map[string]any{"is_revoked": true}); err != nil {
		zap.L().Error("Failed to revoke session during logout",
			zap.String("session_id", session.ID.String()),
			zap.Error(err))
		return nil, errors.New("failed to revoke session")
	}

	zap.L().Info("User logout successful",
		zap.String("session_id", session.ID.String()),
		zap.String("user_id", session.UserID.String()))

	return &responses.LogoutResponse{
		Message: "Logged out successfully",
	}, nil
}

func (s *AuthService) LogoutAll(ctx context.Context, userID uuid.UUID) (*responses.LogoutResponse, error) {
	zap.L().Info("User logout all sessions attempt",
		zap.String("user_id", userID.String()))

	conditions := func(db *gorm.DB) *gorm.DB {
		return db.Where("user_id = ? AND is_revoked = ?", userID, false)
	}
	if err := s.loggedSessionRepository.UpdateByCondition(ctx, conditions, map[string]any{"is_revoked": true}); err != nil {
		zap.L().Error("Failed to revoke all sessions during logout all",
			zap.String("user_id", userID.String()),
			zap.Error(err))
		return nil, errors.New("failed to revoke all sessions")
	}

	zap.L().Info("User logout all sessions successful",
		zap.String("user_id", userID.String()))

	return &responses.LogoutResponse{
		Message: "All sessions revoked successfully",
	}, nil
}

func (s *AuthService) GetActiveSessions(ctx context.Context, userID uuid.UUID) ([]*responses.SessionInfo, error) {
	zap.L().Debug("Retrieving active sessions",
		zap.String("user_id", userID.String()))

	filtesr := func(db *gorm.DB) *gorm.DB {
		return db.Where("user_id = ? AND is_revoked = ? AND expiry_at > ?", userID, false, time.Now())
	}
	sessions, _, err := s.loggedSessionRepository.GetAll(ctx, filtesr, nil, 0, 100)
	if err != nil {
		zap.L().Error("Failed to get active sessions",
			zap.String("user_id", userID.String()),
			zap.Error(err))
		return nil, errors.New("failed to get active sessions")
	}

	sessionInfos := make([]*responses.SessionInfo, len(sessions))
	for i, session := range sessions {
		sessionInfos[i] = &responses.SessionInfo{
			ID:                session.ID,
			DeviceFingerprint: session.DeviceFingerprint,
			CreatedAt:         session.CreatedAt,
			LastUsedAt:        session.LastUsedAt,
			ExpiryAt:          session.ExpiryAt,
			IsRevoked:         session.IsRevoked,
		}
	}

	zap.L().Info("Active sessions retrieved successfully",
		zap.String("user_id", userID.String()),
		zap.Int("session_count", len(sessions)))

	return sessionInfos, nil
}

func (s *AuthService) RevokeSession(ctx context.Context, sessionID uuid.UUID) (*responses.LogoutResponse, error) {
	zap.L().Info("Session revocation attempt",
		zap.String("session_id", sessionID.String()))

	conditions := func(db *gorm.DB) *gorm.DB {
		return db.Where("id = ? AND is_revoked = ?", sessionID, false)
	}
	if err := s.loggedSessionRepository.UpdateByCondition(ctx, conditions, map[string]any{"is_revoked": true}); err != nil {
		zap.L().Error("Failed to revoke session",
			zap.String("session_id", sessionID.String()),
			zap.Error(err))
		return nil, errors.New("failed to revoke session")
	}

	zap.L().Info("Session revoked successfully",
		zap.String("session_id", sessionID.String()))

	return &responses.LogoutResponse{
		Message: "Session revoked successfully",
	}, nil
}
