package service

import (
	"context"
	"core-backend/config"
	"core-backend/internal/application/dto/dtos"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/iproxies"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/application/interfaces/iservice"
	"core-backend/internal/application/service/helper"
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
	"core-backend/pkg/utils"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type TikTokSocialService struct {
	config                  *config.AppConfig
	tiktokProxy             iproxies.TikTokProxy
	channelService          iservice.ChannelService
	jwtService              iservice.JWTService
	userRepository          irepository.GenericRepository[model.User]
	loggedSessionRepository irepository.GenericRepository[model.LoggedSession]
	unitOfWork              irepository.UnitOfWork
}

func NewTikTokSocialService(
	config *config.AppConfig,
	tiktokProxy iproxies.TikTokProxy,
	channelService iservice.ChannelService,
	jwtService iservice.JWTService,
	userRepository irepository.GenericRepository[model.User],
	loggedSessionRepository irepository.GenericRepository[model.LoggedSession],
	unitOfWork irepository.UnitOfWork,
) iservice.TikTokSocialService {
	return &TikTokSocialService{
		config:                  config,
		tiktokProxy:             tiktokProxy,
		channelService:          channelService,
		jwtService:              jwtService,
		userRepository:          userRepository,
		loggedSessionRepository: loggedSessionRepository,
		unitOfWork:              unitOfWork,
	}
}

// HandleOAuthLogin implements iservice.TikTokSocialService.
func (t *TikTokSocialService) HandleOAuthLogin(ctx context.Context, uow irepository.UnitOfWork, code string, redirectURL string, deviceFingerprint string) (*responses.LoginResponse, error) {
	zap.L().Info("TikTokSocialService - AuthenticateUser called")

	userRepo := uow.Users()

	// 1. Exchange code for access token + refresh token
	tokenResp, err := t.tiktokProxy.ExchangeCodeForToken(ctx, code, redirectURL)
	if err != nil {
		zap.L().Error("Failed to exchange code for token", zap.Error(err))
		return nil, errors.New("failed to authenticate with TikTok")
	}

	// 2. Get user profile from TikTok
	userProfile, err := t.tiktokProxy.GetUserProfile(ctx, tokenResp.AccessToken, tokenResp.OpenID)
	if err != nil {
		zap.L().Error("Failed to get user profile from TikTok", zap.Error(err))
		return nil, errors.New("failed to retrieve user information from TikTok")
	}

	tiktokUser := userProfile.Data.User
	// TikTok doesn't provide email, use open_id as unique identifier
	syntheticEmail := fmt.Sprintf("tiktok_%s@oauth.local", tiktokUser.OpenID)
	username := tiktokUser.UserName

	// 3. Check if user exists by synthetic email
	filters := func(db *gorm.DB) *gorm.DB {
		return db.Where("email = ? or username = ?", syntheticEmail, username)
	}
	user, err := userRepo.GetByCondition(ctx, filters, nil)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		zap.L().Error("Failed to query user", zap.Error(err))
		return nil, errors.New("failed to process authentication")
	}

	// 4. Create user if doesn't exist
	if user == nil {
		zap.L().Info("Creating new user from TikTok OAuth", zap.String("open_id", tiktokUser.OpenID))

		// Generate a random password hash (user won't use it for OAuth login)
		var passwordHash []byte
		passwordHash, err = bcrypt.GenerateFromPassword([]byte(uuid.New().String()), bcrypt.DefaultCost)
		if err != nil {
			zap.L().Error("Failed to generate password hash", zap.Error(err))
			return nil, errors.New("failed to create user")
		}

		// Extract avatar URL if available
		var avatarURL *string
		if tiktokUser.AvatarURL != "" {
			avatarURL = &tiktokUser.AvatarURL
		}

		user = &model.User{
			Username:      *username,
			Email:         syntheticEmail,
			PasswordHash:  string(passwordHash),
			FullName:      tiktokUser.DisplayName,
			Role:          enum.UserRoleCustomer,
			AvatarURL:     avatarURL,
			IsActive:      true,
			EmailEnabled:  true,
			PushEnabled:   true,
			IsTikTokOAuth: true,
			LastLogin:     utils.PtrOrNil(time.Now()),
			OAuthMetadata: &model.OAuthMetadata{
				TikTok: userProfile.ToMetadata(),
			},
		}

		if err = userRepo.Add(ctx, user); err != nil {
			zap.L().Error("Failed to create user", zap.Error(err))
			return nil, errors.New("failed to create user account")
		}

		zap.L().Info("User created successfully", zap.String("user_id", user.ID.String()))
	} else if !user.IsActive {
		zap.L().Warn("OAuth login attempt for deactivated account", zap.String("user_id", user.ID.String()))
		return nil, errors.New("account is deactivated")
	} else {
		zap.L().Info("Existing user found, updating TikTok flag", zap.String("user_id", user.ID.String()))

		user.IsTikTokOAuth = true
		user.LastLogin = utils.PtrOrNil(time.Now())
		user.OAuthMetadata.TikTok = userProfile.ToMetadata()
		if err = userRepo.Update(ctx, user); err != nil {
			zap.L().Error("Failed to update user profile", zap.Error(err))
			return nil, errors.New("failed to update user profile")
		}
	}

	// 5. Generate internal JWT tokens
	accessToken, refreshToken, err := t.jwtService.GenerateTokenPair(
		user.ID.String(),
		user.Username,
		user.Email,
		string(user.Role),
	)
	if err != nil {
		zap.L().Error("Failed to generate token pair", zap.Error(err))
		return nil, errors.New("failed to generate authentication tokens")
	}

	// 6. Hash refresh token and create session
	refreshTokenHash := t.jwtService.HashRefreshToken(refreshToken)
	refreshTokenExpiry := time.Now().Add(time.Duration(t.config.JWT.RefreshExpiryHours) * time.Hour)
	session := &model.LoggedSession{
		UserID:            user.ID,
		RefreshTokenHash:  refreshTokenHash,
		DeviceFingerprint: deviceFingerprint,
		ExpiryAt:          &refreshTokenExpiry,
		IsRevoked:         false,
	}
	if err := uow.LoggedSessions().Add(ctx, session); err != nil {
		zap.L().Error("Failed to create session", zap.Error(err))
		return nil, errors.New("failed to create session")
	}

	zap.L().Info("User authenticated successfully via TikTok OAuth",
		zap.String("user_id", user.ID.String()),
		zap.String("open_id", tiktokUser.OpenID))

	return &responses.LoginResponse{
		AccessToken:           accessToken,
		RefreshToken:          refreshToken,
		ExpiresIn:             int64(config.GetAppConfig().JWT.AccessExpiryHours * 3600),
		User:                  responses.UserInfoResponse{}.ToResponse(user),
		DeviceTokenRegistered: false,
	}, nil
}

// HandleRefreshAccessToken implements iservice.TikTokSocialService.
func (t *TikTokSocialService) HandleRefreshAccessToken(ctx context.Context, uow irepository.UnitOfWork, successReq *requests.TikTokOAuthSuccessRequest) error {
	zap.L().Info("TikTokSocialService - HandleRefreshAccessToken called",
		zap.Any("request", successReq))

	// Get the stored refresh token from the channel
	refreshToken, err := t.channelService.GetDecryptedRefreshToken(ctx, "TIKTOK")
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		zap.L().Warn("Failed to get refresh token, generate new token for TikTok channel", zap.Error(err))
		err = t.generateTikTokChannelAccessToken(ctx, uow, successReq.Code, successReq.BackendCallbackURL)
		if err != nil {
			zap.L().Error("Failed to refresh TikTok access token", zap.Error(err))
			return errors.New("failed to refresh TikTok access token")
		}
	} else if err != nil {
		return err
	}

	if refreshToken == "" {
		zap.L().Info("No refresh token found, generating new access token using authorization code")
		err = t.generateTikTokChannelAccessToken(ctx, uow, successReq.Code, successReq.BackendCallbackURL)
	} else {
		zap.L().Info("Refreshing TikTok access token using stored refresh token")
		err = t.refreshTikTokChannelRefreshToken(ctx, uow, refreshToken)
	}
	if err != nil {
		zap.L().Error("Failed to refresh TikTok access token", zap.Error(err))
		return errors.New("failed to refresh TikTok access token")
	}

	zap.L().Info("TikTok access token refreshed successfully")
	return nil
}

// IsTikTokTokenNearExpiry implements iservice.TikTokSocialService.
func (t *TikTokSocialService) IsTikTokTokenNearExpiry(ctx context.Context, accessToken string) (bool, error) {
	threshold := time.Duration(t.config.AdminConfig.TikTokExpiryThresholdNotifications) * 24 * time.Hour
	return t.channelService.IsTokenExpiringSoon(ctx, "TIKTOK", threshold)
}

// region: ======= Creator Info & Sytem User Profile =======

func (t *TikTokSocialService) GetTikTokCreatorInfo(ctx context.Context) (*dtos.TikTokCreatorInfoResponse, error) {
	zap.L().Info("TikTokSocialService - GetTikTokCreatorInfo called")

	// 1. Get TikTok Access Token from channel service
	accessToken, err := t.GetTikTokAccessToken(ctx)
	if err != nil {
		zap.L().Error("Failed to get TikTok access token", zap.Error(err))
		return nil, err
	}

	// 2. Call TikTok Proxy to get creator info
	var creatorInfo *dtos.TikTokCreatorInfoResponse
	if creatorInfo, err = t.tiktokProxy.GetCreatorInfo(ctx, accessToken); err != nil {
		zap.L().Error("Failed to get TikTok creator info", zap.Error(err))
		return nil, err
	}

	return creatorInfo, nil
}

func (t *TikTokSocialService) GetTikTokSystemUserProfile(ctx context.Context) (*dtos.TikTokUserProfileResponse, error) {
	zap.L().Info("TikTokSocialService - GetTikTokSystemUserProfile called")

	// 1. Get TikTok Access Token from channel service
	accessToken, err := t.GetTikTokAccessToken(ctx)
	if err != nil {
		zap.L().Error("Failed to get TikTok access token", zap.Error(err))
		return nil, err
	}

	// 2. Call TikTok Proxy to get system user profile
	var userProfile *dtos.TikTokUserProfileResponse
	if userProfile, err = t.tiktokProxy.GetSystemUserProfile(ctx, accessToken); err != nil {
		zap.L().Error("Failed to get TikTok system user profile", zap.Error(err))
		return nil, err
	}

	return userProfile, nil
}

// endregion

// region: ======= Helper Function =======

func (t *TikTokSocialService) refreshTikTokChannelRefreshToken(ctx context.Context, uow irepository.UnitOfWork, refreshToken string) error {
	// Refresh the access token
	tokenResp, err := t.tiktokProxy.RefreshAccessToken(ctx, refreshToken)
	if err != nil {
		zap.L().Error("Failed to refresh access token", zap.Error(err))
		return errors.New("failed to refresh TikTok access token")
	}

	// Update the stored tokens
	accessExpiresAt := time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
	refreshExpiresAt := time.Now().Add(time.Duration(tokenResp.RefreshExpiresIn) * time.Second)
	err = t.channelService.UpdateChannelToken(ctx, uow, "TIKTOK",
		"", "", tokenResp.AccessToken, &tokenResp.RefreshToken, &accessExpiresAt, &refreshExpiresAt)
	if err != nil {
		zap.L().Error("Failed to update TikTok tokens", zap.Error(err))
		return errors.New("failed to update TikTok credentials")
	}
	return nil
}

func (t *TikTokSocialService) generateTikTokChannelAccessToken(ctx context.Context, uow irepository.UnitOfWork, code, redirectURL string) error {
	tiktokTokenResp, err := t.tiktokProxy.ExchangeCodeForToken(ctx, code, redirectURL)
	if err != nil {
		zap.L().Error("Failed to exchange code for TikTok token", zap.Error(err))
		return errors.New("failed to authenticate with TikTok")
	}

	var tiktokUserProfile *dtos.TikTokUserProfileResponse
	if tiktokUserProfile, err = t.tiktokProxy.GetSystemUserProfile(ctx, tiktokTokenResp.AccessToken); err != nil {
		zap.L().Error("Failed to get TikTok user profile", zap.Error(err))
		return errors.New("failed to retrieve user information from TikTok")
	}

	// Store tokens in channel
	tiktokProfile := tiktokUserProfile.Data.User
	accessExpiresAt := time.Now().Add(time.Duration(tiktokTokenResp.ExpiresIn) * time.Second)
	refreshExpiresAt := time.Now().Add(time.Duration(tiktokTokenResp.RefreshExpiresIn) * time.Second)
	err = t.channelService.UpdateChannelToken(ctx, uow, "TIKTOK",
		tiktokProfile.OpenID, *tiktokProfile.UserName, tiktokTokenResp.AccessToken, &tiktokTokenResp.RefreshToken, &refreshExpiresAt, &accessExpiresAt)
	if err != nil {
		zap.L().Error("Failed to store TikTok channel tokens", zap.Error(err))
		return errors.New("failed to store TikTok credentials")
	}
	return nil
}

func (t *TikTokSocialService) GetTikTokAccessToken(ctx context.Context) (string, error) {
	zap.L().Info("TikTokSocialService - getTikTokAccessToken called")

	// Get TikTok token Pair from channel service
	accessToken, refreshToken, err := t.channelService.GetDecryptedTokenPair(ctx, "TIKTOK")
	if err != nil {
		switch err {
		case iservice.ErrRefreshExpired:
			zap.L().Warn("TikTok refresh token expired, need to re-authenticate")
			return "", errors.New("tiktok refresh token expired")

		case iservice.ErrAccessExpired:
			zap.L().Info("TikTok access token expired, refreshing using refresh token")
			// Refresh the access token
			uow := t.unitOfWork.Begin(ctx)
			if err = helper.WithTransaction(ctx, uow, func(ctx context.Context, uow irepository.UnitOfWork) error {
				err = t.refreshTikTokChannelRefreshToken(ctx, uow, refreshToken)
				if err != nil {
					return err
				}
				return nil
			}); err != nil {
				zap.L().Error("Failed to refresh TikTok access token", zap.Error(err))
				return "", errors.New("failed to refresh TikTok access token")
			}

			return t.channelService.GetDecryptedToken(ctx, "TIKTOK")

		default:
			zap.L().Error("Failed to get TikTok token pair", zap.Error(err))
			return "", errors.New("failed to retrieve TikTok tokens")
		}
	}

	return accessToken, nil
}

// endregion
