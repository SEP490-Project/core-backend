package irepository

import (
	"context"
	"core-backend/internal/domain/model"
	"time"

	"github.com/google/uuid"
)

// DeviceTokenRepository defines the interface for device token data access
type DeviceTokenRepository interface {
	GenericRepository[model.DeviceToken]

	// FindByUserID retrieves all device tokens for a specific user
	FindByUserID(ctx context.Context, userID uuid.UUID) ([]model.DeviceToken, error)

	// FindByToken retrieves a device token by its token string
	FindByToken(ctx context.Context, token string) (*model.DeviceToken, error)

	// MarkInvalid marks a device token as invalid (sets IsValid to false)
	MarkInvalid(ctx context.Context, token string) error

	// CleanupExpired deletes device tokens that haven't been used since the cutoff date
	CleanupExpired(ctx context.Context, cutoffDate time.Time) (int64, error)

	// UpdateLastUsed updates the last used timestamp for a device token
	UpdateLastUsed(ctx context.Context, token string) error
}
