package gormrepository

import (
	"context"
	"core-backend/internal/domain/model"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// DeviceTokenRepository implements the device token repository interface
type DeviceTokenRepository struct {
	*genericRepository[model.DeviceToken]
}

// NewDeviceTokenRepository creates a new device token repository
func NewDeviceTokenRepository(db *gorm.DB) *DeviceTokenRepository {
	return &DeviceTokenRepository{
		genericRepository: &genericRepository[model.DeviceToken]{db: db},
	}
}

// FindByUserID retrieves all device tokens for a specific user
func (r *DeviceTokenRepository) FindByUserID(ctx context.Context, userID uuid.UUID) ([]model.DeviceToken, error) {
	var tokens []model.DeviceToken
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND is_valid = ?", userID, true).
		Order("last_used_at DESC").
		Find(&tokens).Error

	if err != nil {
		return nil, err
	}

	return tokens, nil
}

// FindByToken retrieves a device token by its token string
func (r *DeviceTokenRepository) FindByToken(ctx context.Context, token string) (*model.DeviceToken, error) {
	var deviceToken model.DeviceToken
	err := r.db.WithContext(ctx).
		Where("token = ?", token).
		First(&deviceToken).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &deviceToken, nil
}

// MarkInvalid marks a device token as invalid (sets IsValid to false)
func (r *DeviceTokenRepository) MarkInvalid(ctx context.Context, token string) error {
	result := r.db.WithContext(ctx).
		Model(&model.DeviceToken{}).
		Where("token = ?", token).
		Update("is_valid", false)

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return errors.New("device token not found")
	}

	return nil
}

// CleanupExpired deletes device tokens that haven't been used since the cutoff date
func (r *DeviceTokenRepository) CleanupExpired(ctx context.Context, cutoffDate time.Time) (int64, error) {
	result := r.db.WithContext(ctx).
		Where("last_used_at < ? OR is_valid = ?", cutoffDate, false).
		Delete(&model.DeviceToken{})

	if result.Error != nil {
		return 0, result.Error
	}

	return result.RowsAffected, nil
}

// UpdateLastUsed updates the last used timestamp for a device token
func (r *DeviceTokenRepository) UpdateLastUsed(ctx context.Context, token string) error {
	now := time.Now()
	result := r.db.WithContext(ctx).
		Model(&model.DeviceToken{}).
		Where("token = ?", token).
		Update("last_used_at", now)

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return errors.New("device token not found")
	}

	return nil
}
