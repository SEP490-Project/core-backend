package service

import (
	"context"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
	"errors"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// DeviceTokenService implements device token management operations
type DeviceTokenService struct {
	deviceTokenRepo irepository.DeviceTokenRepository
}

// NewDeviceTokenService creates a new device token service
func NewDeviceTokenService(deviceTokenRepo irepository.DeviceTokenRepository) *DeviceTokenService {
	return &DeviceTokenService{
		deviceTokenRepo: deviceTokenRepo,
	}
}

// RegisterToken registers a new device token or updates if it exists
func (s *DeviceTokenService) RegisterToken(ctx context.Context, userID uuid.UUID, sessionID *uuid.UUID, token string, platform enum.PlatformType) error {
	// Validate platform
	if !platform.IsValid() {
		return errors.New("invalid platform type")
	}

	// Check if token already exists (including soft-deleted)
	query := func(db *gorm.DB) *gorm.DB {
		return db.Unscoped().Where("token = ?", token)
	}
	existing, err := s.deviceTokenRepo.GetByCondition(ctx, query, nil)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		zap.L().Error("Failed to check existing device token",
			zap.String("token", token),
			zap.Error(err))
		return err
	}

	// If token exists
	if existing != nil {
		// If token belongs to a different user, reject
		if existing.UserID != userID {
			zap.L().Warn("Device token already registered to another user",
				zap.String("token", token),
				zap.String("existing_user_id", existing.UserID.String()),
				zap.String("new_user_id", userID.String()))
			return errors.New("device token already registered to another user")
		}

		// Restore if deleted
		if existing.DeletedAt.Valid {
			existing.DeletedAt = gorm.DeletedAt{}
		}

		// Update last used timestamp and ensure valid
		now := time.Now()
		existing.LastUsedAt = &now
		existing.IsValid = true
		existing.Platform = platform
		existing.LoggedSessionID = sessionID

		if err := s.deviceTokenRepo.Update(ctx, existing); err != nil {
			zap.L().Error("Failed to update device token",
				zap.String("token_id", existing.ID.String()),
				zap.Error(err))
			return err
		}

		zap.L().Info("Device token updated",
			zap.String("token_id", existing.ID.String()),
			zap.String("user_id", userID.String()))
		return nil
	}

	// Create new device token
	now := time.Now()
	deviceToken := &model.DeviceToken{
		UserID:          userID,
		LoggedSessionID: sessionID,
		Token:           token,
		Platform:        platform,
		IsValid:         true,
		LastUsedAt:      &now,
	}

	if err := s.deviceTokenRepo.Add(ctx, deviceToken); err != nil {
		zap.L().Error("Failed to register device token",
			zap.String("user_id", userID.String()),
			zap.Error(err))
		return err
	}

	zap.L().Info("Device token registered successfully",
		zap.String("token_id", deviceToken.ID.String()),
		zap.String("user_id", userID.String()),
		zap.String("platform", string(platform)))
	return nil
}

// UpdateToken updates an existing device token
func (s *DeviceTokenService) UpdateToken(ctx context.Context, tokenID uuid.UUID, newToken string, platform enum.PlatformType) error {
	// Validate platform
	if !platform.IsValid() {
		return errors.New("invalid platform type")
	}

	// Get existing token
	existing, err := s.deviceTokenRepo.GetByID(ctx, tokenID, nil)
	if err != nil {
		return err
	}

	if existing == nil {
		return errors.New("device token not found")
	}

	// Check if new token is already registered
	tokenExists, err := s.deviceTokenRepo.FindByToken(ctx, newToken)
	if err != nil {
		return err
	}

	if tokenExists != nil && tokenExists.ID != tokenID {
		return errors.New("new token already registered")
	}

	// Update token
	now := time.Now()
	existing.Token = newToken
	existing.Platform = platform
	existing.LastUsedAt = &now
	existing.IsValid = true

	if err := s.deviceTokenRepo.Update(ctx, existing); err != nil {
		zap.L().Error("Failed to update device token",
			zap.String("token_id", tokenID.String()),
			zap.Error(err))
		return err
	}

	zap.L().Info("Device token updated",
		zap.String("token_id", tokenID.String()))
	return nil
}

// DeleteToken deletes a specific device token
func (s *DeviceTokenService) DeleteToken(ctx context.Context, tokenID uuid.UUID) error {
	// Check if token exists
	existing, err := s.deviceTokenRepo.GetByID(ctx, tokenID, nil)
	if err != nil {
		return err
	}

	if existing == nil {
		return errors.New("device token not found")
	}

	if err := s.deviceTokenRepo.Delete(ctx, existing); err != nil {
		zap.L().Error("Failed to delete device token",
			zap.String("token_id", tokenID.String()),
			zap.Error(err))
		return err
	}

	zap.L().Info("Device token deleted",
		zap.String("token_id", tokenID.String()),
		zap.String("user_id", existing.UserID.String()))
	return nil
}

// DeleteAllTokens deletes all device tokens for a user
func (s *DeviceTokenService) DeleteAllTokens(ctx context.Context, userID uuid.UUID) error {
	// Get all tokens for user
	tokens, err := s.deviceTokenRepo.FindByUserID(ctx, userID)
	if err != nil {
		return err
	}

	// Delete each token
	for _, token := range tokens {
		if err := s.deviceTokenRepo.Delete(ctx, &token); err != nil {
			zap.L().Error("Failed to delete device token",
				zap.String("token_id", token.ID.String()),
				zap.Error(err))
			// Continue deleting other tokens even if one fails
			continue
		}
	}

	zap.L().Info("All device tokens deleted for user",
		zap.String("user_id", userID.String()),
		zap.Int("count", len(tokens)))
	return nil
}

// GetUserTokens retrieves all valid device tokens for a user
func (s *DeviceTokenService) GetUserTokens(ctx context.Context, userID uuid.UUID) ([]model.DeviceToken, error) {
	tokens, err := s.deviceTokenRepo.FindByUserID(ctx, userID)
	if err != nil {
		zap.L().Error("Failed to get user device tokens",
			zap.String("user_id", userID.String()),
			zap.Error(err))
		return nil, err
	}

	return tokens, nil
}

// CleanupInvalidTokens removes expired and invalid device tokens
func (s *DeviceTokenService) CleanupInvalidTokens(ctx context.Context) (int64, error) {
	// Remove tokens not used in the last 90 days
	cutoffDate := time.Now().AddDate(0, 0, -90)

	count, err := s.deviceTokenRepo.CleanupExpired(ctx, cutoffDate)
	if err != nil {
		zap.L().Error("Failed to cleanup invalid device tokens", zap.Error(err))
		return 0, err
	}

	zap.L().Info("Cleaned up expired device tokens",
		zap.Int64("count", count),
		zap.Time("cutoff_date", cutoffDate))
	return count, nil
}

// DeleteTokenByToken deletes a device token by its token string
func (s *DeviceTokenService) DeleteTokenByToken(ctx context.Context, token string) error {
	existing, err := s.deviceTokenRepo.FindByToken(ctx, token)
	if err != nil {
		zap.L().Error("Failed to find device token for deletion",
			zap.String("token", token),
			zap.Error(err))
		return err
	}
	if existing == nil {
		return nil
	}

	if err := s.deviceTokenRepo.Delete(ctx, existing); err != nil {
		zap.L().Error("Failed to delete device token",
			zap.String("token_id", existing.ID.String()),
			zap.Error(err))
		return err
	}

	zap.L().Info("Device token deleted successfully",
		zap.String("token_id", existing.ID.String()))
	return nil
}

// DeleteTokensBySessionID deletes all device tokens associated with a session
func (s *DeviceTokenService) DeleteTokensBySessionID(ctx context.Context, sessionID uuid.UUID) error {
	query := func(db *gorm.DB) *gorm.DB {
		return db.Where("logged_session_id = ?", sessionID)
	}

	tokens, _, err := s.deviceTokenRepo.GetAll(ctx, query, nil, 0, 0)
	if err != nil {
		zap.L().Error("Failed to get device tokens by session ID",
			zap.String("session_id", sessionID.String()),
			zap.Error(err))
		return err
	}

	for _, token := range tokens {
		if err := s.deviceTokenRepo.Delete(ctx, &token); err != nil {
			zap.L().Error("Failed to delete device token",
				zap.String("token_id", token.ID.String()),
				zap.Error(err))
			// Continue deleting other tokens
		}
	}

	zap.L().Info("Device tokens deleted by session ID",
		zap.String("session_id", sessionID.String()),
		zap.Int("count", len(tokens)))
	return nil
}
