package service

import (
	"context"
	"core-backend/config"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/application/interfaces/iservice"
	iservicethirdparty "core-backend/internal/application/interfaces/iservice_third_party"
	"core-backend/internal/domain/model"
	"core-backend/pkg/crypto"
	"core-backend/pkg/utils"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type channelService struct {
	channelRepo        irepository.GenericRepository[model.Channel]
	tokenEncryptionKey []byte
	vaultService       iservicethirdparty.VaultService
	config             *config.TokenStorageConfig
}

// CreateChannel implements iservice.ChannelService.
func (c *channelService) CreateChannel(ctx context.Context, request *requests.CreateChannelRequest, uow irepository.UnitOfWork) (*responses.ChannelResponse, error) {
	zap.L().Info("CreateChannel called", zap.Any("request", request))

	channelRepo := uow.Channels()

	creatingModel := &model.Channel{
		ID:          uuid.New(),
		Code:        request.Code,
		Name:        request.Name,
		Description: request.Description,
		HomePageURL: request.HomePageURL,
		IsActive:    request.IsActive,
	}

	if err := channelRepo.Add(ctx, creatingModel); err != nil {
		zap.L().Error("Failed to create channel", zap.Error(err))
		return nil, err
	}

	createdModel, err := channelRepo.GetByID(ctx, creatingModel.ID, nil)
	if err != nil {
		zap.L().Error("Failed to get created channel by ID", zap.Error(err))
		return nil, err
	}

	return responses.ChannelResponse{}.ToResponse(createdModel), nil
}

// DeleteChannel implements iservice.ChannelService.
func (c *channelService) DeleteChannel(ctx context.Context, channelID uuid.UUID, uow irepository.UnitOfWork) error {
	zap.L().Info("DeleteChannel called", zap.String("channel_id", channelID.String()))

	channelRepo := uow.Channels()

	channel, err := channelRepo.GetByID(ctx, channelID, nil)
	if err != nil {
		zap.L().Error("Failed to get channel by ID", zap.Error(err))
		return err
	} else if channel == nil {
		zap.L().Warn("Channel not found", zap.String("channel_id", channelID.String()))
		return errors.New("channel not found")
	}

	if err := channelRepo.Delete(ctx, channel); err != nil {
		zap.L().Error("Failed to delete channel", zap.Error(err))
		return err
	}

	return nil
}

// GetAllChannels implements iservice.ChannelService.
func (c *channelService) GetAllChannels(ctx context.Context, isReturnTokenInfo bool) ([]responses.ChannelResponse, error) {
	zap.L().Info("GetAllChannels called")

	channels, _, err := c.channelRepo.GetAll(ctx, nil, nil, 0, 0)
	if err != nil {
		zap.L().Error("Failed to get all channels", zap.Error(err))
		return nil, err
	}

	var channelResponses []responses.ChannelResponse
	for _, model := range channels {
		tempResp := (responses.ChannelResponse{}.ToResponse(&model))
		if isReturnTokenInfo {
			tempResp.TokenInfo = &responses.ChannelTokenInfo{
				ExternalID:            utils.DerefPtr(model.ExternalID, ""),
				AccountName:           utils.DerefPtr(model.AccountName, ""),
				AccessTokenExpiresAt:  utils.FormatLocalTime(model.AccessTokenExpiresAt, ""),
				RefreshTokenExpiresAt: utils.PtrOrNil(utils.FormatLocalTime(model.RefreshTokenExpiresAt, "")),
				LastSyncedAt:          utils.FormatLocalTime(model.LastSyncedAt, ""),
			}
		}

		channelResponses = append(channelResponses, *tempResp)

	}
	return channelResponses, nil
}

// GetChannelByID implements iservice.ChannelService.
func (c *channelService) GetChannelByID(ctx context.Context, channelID uuid.UUID) (*responses.ChannelResponse, error) {
	zap.L().Info("GetChannelByID called", zap.String("channel_id", channelID.String()))

	channel, err := c.channelRepo.GetByID(ctx, channelID, nil)
	if err != nil {
		zap.L().Error("Failed to get channel by ID", zap.Error(err))
		return nil, err
	}

	return responses.ChannelResponse{}.ToResponse(channel), nil
}

func (c *channelService) GetChannelByName(ctx context.Context, channelName string) (*responses.ChannelResponse, error) {
	zap.L().Info("GetChannelByName called", zap.String("channel_name", channelName))

	channel, err := c.getChannelByName(ctx, channelName)

	return responses.ChannelResponse{}.ToResponse(channel), err
}

// UpdateChannel implements iservice.ChannelService.
func (c *channelService) UpdateChannel(
	ctx context.Context,
	channelID uuid.UUID,
	request *requests.UpdateChannelRequest,
	uow irepository.UnitOfWork,
) (*responses.ChannelResponse, error) {
	zap.L().Info("UpdateChannel called", zap.String("channel_id", channelID.String()), zap.Any("request", request))

	channelRepo := uow.Channels()

	channel, err := channelRepo.GetByID(ctx, channelID, nil)
	if err != nil {
		zap.L().Error("Failed to get channel by ID", zap.Error(err))
		return nil, err
	} else if channel == nil {
		zap.L().Warn("Channel not found", zap.String("channel_id", channelID.String()))
		return nil, errors.New("channel not found")
	}

	if request.Code != nil {
		channel.Code = *request.Code
	}
	if request.Name != nil {
		channel.Name = *request.Name
	}
	if request.Description != nil {
		channel.Description = request.Description
	}
	if request.HomePageURL != nil {
		channel.HomePageURL = request.HomePageURL
	}
	if request.IsActive != nil && *request.IsActive != channel.IsActive {
		channel.IsActive = *request.IsActive
	}

	if err = channelRepo.Update(ctx, channel); err != nil {
		zap.L().Error("Failed to update channel", zap.Error(err))
		return nil, err
	}

	var updatedModel *model.Channel
	updatedModel, err = channelRepo.GetByID(ctx, channel.ID, nil)
	if err != nil {
		zap.L().Error("Failed to get updated channel by ID", zap.Error(err))
		return nil, err
	}

	return responses.ChannelResponse{}.ToResponse(updatedModel), nil
}

// UpdateChannelToken stores encrypted OAuth tokens for a channel
func (c *channelService) UpdateChannelToken(ctx context.Context, uow irepository.UnitOfWork,
	channelName string, externalID string, accountName string, accessToken string, refreshToken *string, expiresAt *time.Time, refreshExpiresAt *time.Time,
) error {
	zap.L().Info("ChannelService - UpdateChannelToken called",
		zap.String("channel_name", channelName),
		zap.String("external_id", externalID),
		zap.String("account_name", accountName),
		zap.Any("expires_at", expiresAt),
		zap.Any("refresh_expires_at", refreshExpiresAt))

	// channelRepo := uow.Channels()

	channel, err := c.getChannelByName(ctx, channelName)
	if err != nil && errors.Is(err, gorm.ErrRecordNotFound) {
		zap.L().Info("ChannelService - UpdateChannelToken - Channel not found, creating new channel record",
			zap.String("channel_name", channelName))

	}

	now := time.Now()
	if externalID != "" && (channel.ExternalID == nil || *channel.ExternalID != externalID) {
		channel.ExternalID = &externalID
	}
	if accountName != "" && (channel.AccountName == nil || *channel.AccountName != accountName) {
		channel.AccountName = &accountName
	}
	channel.AccessTokenExpiresAt = expiresAt
	channel.RefreshTokenExpiresAt = refreshExpiresAt
	channel.LastSyncedAt = &now

	// Choose storage backend based on configuration
	if c.config.UseVault && c.vaultService != nil {
		// Store vault path reference in database (not the actual token)
		vaultPath := fmt.Sprintf("%s/%s/%s", c.config.VaultPathPrefix, channelName, externalID)
		channel.HashedAccessToken = &vaultPath
		channel.HashedRefreshToken = nil // Not needed for vault backend

		data := map[string]any{
			"access_token": accessToken,
			"channel":      channelName,
			"external_id":  externalID,
			"stored_at":    time.Now().Unix(),
		}
		if refreshToken != nil && *refreshToken != "" {
			data["refresh_token"] = *refreshToken
		}

		if err = c.vaultService.PutSecret(ctx, vaultPath, data); err != nil {
			zap.L().Error("ChannelService - UpdateChannelToken - Failed to store token in Vault", zap.Error(err))
			return errors.New("failed to store token in Vault")
		}

		zap.L().Info("ChannelService - UpdateChannelToken - Token stored in Vault",
			zap.String("channel", channelName), zap.String("vault_path", vaultPath))
	} else {
		// Store in database with encryption (default)
		encryptedAccessToken, err := crypto.EncryptToken(accessToken, c.tokenEncryptionKey)
		if err != nil {
			zap.L().Error("ChannelService - UpdateChannelToken - Failed to encrypt access token", zap.Error(err))
			return errors.New("failed to encrypt access token")
		}

		var encryptedRefreshToken *string
		if refreshToken != nil && *refreshToken != "" {
			encrypted, err := crypto.EncryptToken(*refreshToken, c.tokenEncryptionKey)
			if err != nil {
				zap.L().Error("ChannelService - UpdateChannelToken - Failed to encrypt refresh token", zap.Error(err))
				return errors.New("failed to encrypt refresh token")
			}
			encryptedRefreshToken = &encrypted
		}

		channel.HashedAccessToken = &encryptedAccessToken
		channel.HashedRefreshToken = encryptedRefreshToken

		zap.L().Info("ChannelService - UpdateChannelToken - Token stored in database (encrypted)",
			zap.String("channel", channelName))
	}

	return c.channelRepo.Update(ctx, channel)
}

// GetDecryptedToken retrieves and decrypts the access token for a channel
func (c *channelService) GetDecryptedToken(ctx context.Context, channelName string) (string, error) {
	zap.L().Info("ChannelService - GetDecryptedToken called",
		zap.String("channel_name", channelName))
	channel, err := c.getChannelByName(ctx, channelName)
	if err != nil {
		return "", err
	}

	if channel.HashedAccessToken == nil {
		return "", errors.New("no access token stored for channel")
	}

	// Check if using Vault backend
	if c.config.UseVault && c.vaultService != nil {
		// HashedAccessToken contains vault path, not encrypted token
		vaultPath := *channel.HashedAccessToken
		zap.L().Debug("ChannelService - GetDecryptedToken - Using Vault backend",
			zap.String("vault_path", vaultPath))

		var secret map[string]any
		secret, err = c.vaultService.GetSecret(ctx, vaultPath)
		if err != nil {
			zap.L().Error("Failed to read token from Vault",
				zap.String("vault_path", vaultPath),
				zap.Error(err))
			return "", errors.New("failed to retrieve token from Vault")
		}

		accessToken, ok := secret["access_token"].(string)
		if !ok {
			return "", errors.New("invalid token format in Vault")
		}
		zap.L().Info("ChannelService - GetDecryptedToken - Token retrieved from Vault",
			zap.Int("access_token_length", len(accessToken)))

		return accessToken, nil
	}

	// Database backend: decrypt token
	decryptedToken, err := crypto.DecryptToken(*channel.HashedAccessToken, c.tokenEncryptionKey)
	if err != nil {
		return "", errors.New("failed to decrypt access token")
	}

	return decryptedToken, nil
}

// GetDecryptedRefreshToken retrieves and decrypts the refresh token for a channel
func (c *channelService) GetDecryptedRefreshToken(ctx context.Context, channelName string) (string, error) {
	channel, err := c.getChannelByName(ctx, channelName)
	if err != nil {
		return "", err
	}

	// Check if using Vault backend
	if c.config.UseVault && c.vaultService != nil {
		if channel.HashedAccessToken == nil {
			return "", errors.New("no vault path stored for channel")
		}

		vaultPath := *channel.HashedAccessToken
		var secret map[string]any
		secret, err = c.vaultService.GetSecret(ctx, vaultPath)
		if err != nil {
			return "", errors.New("failed to retrieve token from Vault")
		}

		refreshToken, ok := secret["refresh_token"].(string)
		if !ok {
			return "", errors.New("no refresh token in Vault")
		}

		return refreshToken, nil
	}

	// Database backend: decrypt token
	if channel.HashedRefreshToken == nil {
		return "", errors.New("no refresh token stored for channel")
	}

	decryptedToken, err := crypto.DecryptToken(*channel.HashedRefreshToken, c.tokenEncryptionKey)
	if err != nil {
		return "", errors.New("failed to decrypt refresh token")
	}

	return decryptedToken, nil
}

// IsTokenExpiringSoon checks if a token will expire within the given duration
func (c *channelService) IsTokenExpiringSoon(ctx context.Context, channelName string, threshold time.Duration) (bool, error) {
	channel, err := c.getChannelByName(ctx, channelName)
	if err != nil {
		return false, err
	}

	if channel.AccessTokenExpiresAt == nil {
		return true, nil // No expiry info = assume needs refresh
	}

	expiresIn := time.Until(*channel.AccessTokenExpiresAt)
	return expiresIn <= threshold, nil
}

// ClearChannelToken removes OAuth tokens from a channel
func (c *channelService) ClearChannelToken(ctx context.Context, uow irepository.UnitOfWork, channelName string) error {
	zap.L().Info("ChannelService - ClearChannelToken called",
		zap.String("channel_name", channelName))
	channel, err := c.getChannelByName(ctx, channelName)
	if err != nil {
		return err
	}

	channel.ExternalID = nil
	channel.AccountName = nil
	channel.HashedAccessToken = nil
	channel.HashedRefreshToken = nil
	channel.AccessTokenExpiresAt = nil
	channel.RefreshTokenExpiresAt = nil
	channel.LastSyncedAt = nil

	if err = uow.Channels().Update(ctx, channel); err != nil {
		zap.L().Error("ChannelService - ClearChannelToken - Failed to clear channel token", zap.Error(err))
		return err
	}

	return nil
}

func (c *channelService) GetDecryptedTokenPair(ctx context.Context, channelName string) (accesstoken, refreshToken string, err error) {
	zap.L().Info("ChannelService - GetDecryptedTokenPair called",
		zap.String("channel_name", channelName))

	channel, err := c.getChannelByName(ctx, channelName)
	if err != nil {
		return "", "", err
	}

	if channel.HashedAccessToken == nil {
		zap.L().Warn("No stored access token for channel", zap.String("channel_name", channelName))
		return "", "", iservice.ErrNoStoredToken
	}
	if channel.RefreshTokenExpiresAt != nil && time.Until(*channel.RefreshTokenExpiresAt) <= 0 {
		zap.L().Warn("Refresh token has expired", zap.String("channel_name", channelName))
		return "", "", iservice.ErrRefreshExpired
	}

	// Check if using Vault backend
	if c.config.UseVault && c.vaultService != nil {
		if channel.HashedAccessToken == nil {
			return "", "", errors.New("no vault path stored for channel")
		}

		vaultPath := *channel.HashedAccessToken
		var secret map[string]any
		secret, err = c.vaultService.GetSecret(ctx, vaultPath)
		if err != nil {
			return "", "", errors.New("failed to retrieve token from Vault")
		}

		refreshToken, _ := secret["refresh_token"].(string)
		// if !ok {
		// 	return "", "", errors.New("no refresh token in Vault")
		// }
		accessToken, ok := secret["access_token"].(string)
		if !ok {
			return "", "", errors.New("no access token in Vault")
		}

		if channel.AccessTokenExpiresAt != nil && time.Until(*channel.AccessTokenExpiresAt) <= 0 {
			zap.L().Warn("Access token has expired, need to refresh immediately to continue", zap.String("channel_name", channelName))
			return "", refreshToken, iservice.ErrAccessExpired
		}

		return accessToken, refreshToken, nil
	}

	// Database backend: decrypt token
	if channel.HashedRefreshToken == nil && channel.HashedAccessToken == nil {
		return "", "", errors.New("no refresh anh access token stored for channel")
	}

	var decryptedRefreshToken, decryptedAccessToken string
	if channel.HashedRefreshToken != nil {
		decryptedRefreshToken, err = crypto.DecryptToken(*channel.HashedRefreshToken, c.tokenEncryptionKey)
		if err != nil {
			return "", "", errors.New("failed to decrypt refresh token")
		}
	}
	decryptedAccessToken, err = crypto.DecryptToken(*channel.HashedAccessToken, c.tokenEncryptionKey)
	if err != nil {
		return "", "", errors.New("failed to decrypt access token")
	}

	if channel.AccessTokenExpiresAt != nil && time.Until(*channel.AccessTokenExpiresAt) <= 0 {
		zap.L().Warn("Access token has expired, need to refresh immediately to continue", zap.String("channel_name", channelName))
		return "", decryptedRefreshToken, iservice.ErrAccessExpired
	}

	return decryptedAccessToken, decryptedRefreshToken, nil
}

func NewChannelService(channelRepo irepository.GenericRepository[model.Channel], appConfig *config.AppConfig, vaultService iservicethirdparty.VaultService) iservice.ChannelService {
	// Decode hex-encoded encryption key to bytes
	var encryptionKey []byte
	if appConfig.TokenStorage.EncryptionKey != "" {
		decodedKey, err := hex.DecodeString(appConfig.TokenStorage.EncryptionKey)
		if err != nil {
			zap.L().Warn("Failed to decode token encryption key, using as-is", zap.Error(err))
			encryptionKey = []byte(appConfig.TokenStorage.EncryptionKey)
		} else {
			encryptionKey = decodedKey
		}
	}

	return &channelService{
		channelRepo:        channelRepo,
		tokenEncryptionKey: encryptionKey,
		vaultService:       vaultService,
		config:             &appConfig.TokenStorage,
	}
}

// region: ======= Helper Functions =========

// getChannelByName retrieves a channel by its name (FACEBOOK, TIKTOK, WEBSITE)
func (c *channelService) getChannelByName(ctx context.Context, name string) (*model.Channel, error) {
	channel, err := c.channelRepo.GetByCondition(ctx,
		func(db *gorm.DB) *gorm.DB {
			return db.Where("name ilike ? or code ilike ?", "%"+name+"%", name)
		}, nil)

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("channel '%s' not found by name or code", name)
		}
		return nil, err
	}

	return channel, nil
}

// endregion
