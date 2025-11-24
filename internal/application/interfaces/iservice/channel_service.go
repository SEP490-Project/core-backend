package iservice

import (
	"context"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/irepository"
	"time"

	"github.com/google/uuid"
)

type ChannelService interface {
	// ChannelService defines the interface for channel-related operations
	GetAllChannels(ctx context.Context, isReturnTokenInfo bool) ([]responses.ChannelResponse, error)

	// GetChannelByID retrieves a channel by its ID
	GetChannelByID(ctx context.Context, channelID uuid.UUID) (*responses.ChannelResponse, error)

	// GetChannelByName retrieves a channel by its name
	GetChannelByName(ctx context.Context, channelName string) (*responses.ChannelResponse, error)

	// CreateChannel creates a new channel
	CreateChannel(ctx context.Context, request *requests.CreateChannelRequest, uow irepository.UnitOfWork) (*responses.ChannelResponse, error)

	// UpdateChannel updates an existing channel
	UpdateChannel(ctx context.Context, channelID uuid.UUID, request *requests.UpdateChannelRequest, uow irepository.UnitOfWork) (*responses.ChannelResponse, error)

	// DeleteChannel deletes a channel by its ID
	DeleteChannel(ctx context.Context, channelID uuid.UUID, uow irepository.UnitOfWork) error

	UpdateChannelToken(ctx context.Context, uow irepository.UnitOfWork, channelName string, externalID string,
		accountName string, accessToken string, refreshToken *string, expiresAt *time.Time, refreshExpiresAt *time.Time) error

	GetDecryptedToken(ctx context.Context, channelName string) (string, error)

	GetDecryptedRefreshToken(ctx context.Context, channelName string) (string, error)

	IsTokenExpiringSoon(ctx context.Context, channelName string, threshold time.Duration) (bool, error)

	ClearChannelToken(ctx context.Context, uow irepository.UnitOfWork, channelName string) error

	GetDecryptedTokenPair(ctx context.Context, channelName string) (accesstoken, refreshToken string, err error)
}
