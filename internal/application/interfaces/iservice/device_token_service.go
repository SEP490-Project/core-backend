package iservice

import (
	"context"
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"

	"github.com/google/uuid"
)

// DeviceTokenService defines the interface for device token operations
type DeviceTokenService interface {
	// RegisterToken registers a new device token or updates if it exists
	RegisterToken(ctx context.Context, userID uuid.UUID, sessionID *uuid.UUID, token string, platform enum.PlatformType) error

	// UpdateToken updates an existing device token
	UpdateToken(ctx context.Context, tokenID uuid.UUID, newToken string, platform enum.PlatformType) error

	// DeleteToken deletes a specific device token
	DeleteToken(ctx context.Context, tokenID uuid.UUID) error

	// DeleteTokenByToken deletes a device token by its token string
	DeleteTokenByToken(ctx context.Context, token string) error

	// DeleteTokensBySessionID deletes all device tokens associated with a session
	DeleteTokensBySessionID(ctx context.Context, sessionID uuid.UUID) error

	// DeleteAllTokens deletes all device tokens for a user
	DeleteAllTokens(ctx context.Context, userID uuid.UUID) error

	// GetUserTokens retrieves all valid device tokens for a user
	GetUserTokens(ctx context.Context, userID uuid.UUID) ([]model.DeviceToken, error)

	// CleanupInvalidTokens removes expired and invalid device tokens
	CleanupInvalidTokens(ctx context.Context) (int64, error)
}
