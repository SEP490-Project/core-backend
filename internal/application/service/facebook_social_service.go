package service

import (
	"context"
	"core-backend/config"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/iproxies"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/application/interfaces/iservice"
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
	"core-backend/pkg/utils"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type FacebookSocialService struct {
	config                  *config.AppConfig
	facebookProxy           iproxies.FacebookProxy
	channelService          iservice.ChannelService
	jwtService              iservice.JWTService
	userRepository          irepository.GenericRepository[model.User]
	loggedSessionRepository irepository.GenericRepository[model.LoggedSession]
}

// HandleOAuthLogin implements iservice.FacebookSocialService.
func (f *FacebookSocialService) HandleOAuthLogin(ctx context.Context, uow irepository.UnitOfWork, code string, redirectURL string, deviceFingerprint string) (*responses.LoginResponse, error) {
	zap.L().Info("FacebookSocialService - AuthenticateUser called",
		zap.String("code", code[:10]+"..."),
		zap.String("redirect_url", redirectURL),
		zap.String("device_fingerprint", deviceFingerprint))

	userRepo := uow.Users()

	// 1. Exchange code for access token
	tokenResp, err := f.facebookProxy.ExchangeCodeForUserAccessToken(ctx, code, redirectURL)
	if err != nil {
		zap.L().Error("FacebookSocialService - AuthenticateUser - Failed to exchange code for access token",
			zap.Error(err))
		return nil, errors.New("failed to authenticate with Facebook")
	}

	// 2. Get user profile from Facebook
	userProfile, err := f.facebookProxy.GetUserProfile(ctx, tokenResp.AccessToken)
	if err != nil {
		zap.L().Error("FacebookSocialService - AuthenticateUser - Failed to get user profile from Facebook",
			zap.Error(err))
		return nil, errors.New("failed to retrieve user information from Facebook")
	}

	if userProfile.Email == "" {
		zap.L().Error("FacebookSocialService - AuthenticateUser - Facebook user profile missing email",
			zap.Any("user_profile", userProfile))
		return nil, errors.New("email permission is required for authentication")
	}

	// 3. Check if user exists by email
	filters := func(db *gorm.DB) *gorm.DB {
		return db.Where("email = ?", userProfile.Email)
	}
	user, err := userRepo.GetByCondition(ctx, filters, nil)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		zap.L().Error("FacebookSocialService - AuthenticateUser - Failed to query user",
			zap.Error(err))
		return nil, errors.New("failed to process authentication")
	}

	// 4. Create user if doesn't exist
	if user == nil {
		zap.L().Info("FacebookSocialService - AuthenticateUser - Creating new user from Facebook OAuth",
			zap.String("email", userProfile.Email))

		// Generate a random password hash (user won't use it for OAuth login)
		var passwordHash []byte
		passwordHash, err = bcrypt.GenerateFromPassword([]byte(uuid.New().String()), bcrypt.DefaultCost)
		if err != nil {
			zap.L().Error("Failed to generate password hash", zap.Error(err))
			return nil, errors.New("failed to create user")
		}

		// Extract avatar URL if available
		var avatarURL *string
		if userProfile.Picture != nil && userProfile.Picture.Data != nil && userProfile.Picture.Data.URL != "" {
			avatarURL = &userProfile.Picture.Data.URL
		}

		// Extract birthday if available
		// Birthday format can be MM/DD/YYYY, MM/DD, or YYYY
		var birthday *time.Time
		if userProfile.Birthday != nil && *userProfile.Birthday != "" {
			birthdayStr := *userProfile.Birthday
			parts := strings.Split(birthdayStr, "/")
			if len(parts) == 3 {
				birthday, err = utils.ParseLocalTime(birthdayStr, "01-02-2006")
			} else if len(parts) == 2 {
				birthday, err = utils.ParseLocalTime(birthdayStr, "01-02")
			} else if len(parts) == 1 {
				birthday, err = utils.ParseLocalTime(birthdayStr, "2006")
			}
			if err == nil {
				// Use birthday as part of user creation logic if needed
				_ = birthday
			} else {
				zap.L().Warn("Failed to parse birthday from Facebook profile",
					zap.String("birthday_str", birthdayStr),
					zap.Error(err))
			}
		}

		user = &model.User{
			Username:        fmt.Sprintf("fb_%s", userProfile.ID),
			Email:           userProfile.Email,
			PasswordHash:    string(passwordHash),
			FullName:        userProfile.Name,
			DateOfBirth:     birthday,
			Role:            enum.UserRoleCustomer,
			AvatarURL:       avatarURL,
			LastLogin:       utils.PtrOrNil(time.Now()),
			IsActive:        true,
			EmailEnabled:    true,
			PushEnabled:     true,
			IsFacebookOAuth: true,
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
		zap.L().Info("Existing user found, update facebook flag", zap.String("user_id", user.ID.String()))

		// Update user profile info from Facebook
		user.IsFacebookOAuth = true
		user.LastLogin = utils.PtrOrNil(time.Now())
		if err = userRepo.Update(ctx, user); err != nil {
			zap.L().Error("Failed to update user profile", zap.Error(err))
			return nil, errors.New("failed to update user profile")
		}
	}

	// Generate Session ID
	sessionID := uuid.New()

	// 5. Generate internal JWT tokens
	accessToken, refreshToken, err := f.jwtService.GenerateTokenPair(
		user.ID.String(),
		sessionID.String(),
		user.Username,
		user.Email,
		string(user.Role),
	)
	if err != nil {
		zap.L().Error("Failed to generate token pair", zap.Error(err))
		return nil, errors.New("failed to generate authentication tokens")
	}

	// 6. Hash refresh token and create session
	refreshTokenHash := f.jwtService.HashRefreshToken(refreshToken)
	refreshTokenExpiry := time.Now().Add(time.Duration(f.config.JWT.RefreshExpiryHours) * time.Hour)
	session := &model.LoggedSession{
		ID:                sessionID,
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

	zap.L().Info("User authenticated successfully via Facebook OAuth",
		zap.String("user_id", user.ID.String()),
		zap.String("email", user.Email))
	return &responses.LoginResponse{
		AccessToken:           accessToken,
		RefreshToken:          refreshToken,
		ExpiresIn:             int64(config.GetAppConfig().JWT.AccessExpiryHours * 3600), // Convert to seconds
		User:                  responses.UserInfoResponse{}.ToResponse(user),
		DeviceTokenRegistered: false,
	}, nil
}

// HandleRefreshPageAccessToken implements iservice.FacebookSocialService.
func (f *FacebookSocialService) HandleRefreshPageAccessToken(ctx context.Context, uow irepository.UnitOfWork, successReq *requests.FacebookOAuthSuccessRequest) error {
	zap.L().Info("FacebookSocialService - HandleRefreshPageAccessToken called",
		zap.Any("request", successReq))

	// 1. Exchange code for user token
	userTokenResp, err := f.facebookProxy.ExchangeCodeForUserAccessToken(
		ctx, successReq.Code, successReq.BackendCallbackURL)
	if err != nil {
		zap.L().Error("Failed to exchange code for user access token", zap.Error(err))
		return errors.New("failed to exchange code for user access token")
	}

	// 2. Get long-lived token
	longLivedTokenResp, err := f.facebookProxy.ExchangeUserAccessTokenForLongLivedToken(
		ctx, userTokenResp.AccessToken)
	if err != nil {
		longLivedTokenResp = userTokenResp
	}

	// 3. Get page info to get Page Access Token
	accountInfoResp, err := f.facebookProxy.GetAccountPageInfo(ctx, longLivedTokenResp.AccessToken)
	if err != nil {
		zap.L().Error("Failed to get Facebook account page info", zap.Error(err))
		return errors.New("failed to retrieve Facebook pages")
	}
	if len(accountInfoResp.Data) == 0 {
		zap.L().Error("No Facebook pages found for the user")
		return errors.New("no Facebook pages found")
	}

	firstPage := accountInfoResp.Data[0]
	// Long lived token for pages is usually 60 days
	expiresAt := time.Now().Add(60 * 24 * time.Hour)

	// 4. Update channel token
	err = f.channelService.UpdateChannelToken(ctx, uow, "FACEBOOK", firstPage.ID,
		firstPage.Name, firstPage.AccessToken, nil, &expiresAt, nil)
	if err != nil {
		return errors.New("failed to update Facebook Page token")
	}

	zap.L().Info("Facebook Page token refreshed successfully")
	return nil
}

// IsFacebookTokenNearExpiry implements iservice.FacebookSocialService.
func (f *FacebookSocialService) IsFacebookTokenNearExpiry(ctx context.Context, accessToken string) (bool, error) {
	threshold := time.Duration(f.config.AdminConfig.FacebookExpiryThresholdNotifications) * 24 * time.Hour
	return f.channelService.IsTokenExpiringSoon(ctx, "FACEBOOK", threshold)
}

func NewFacebookSocialService(
	config *config.AppConfig,
	facebookProxy iproxies.FacebookProxy,
	channelService iservice.ChannelService,
	jwtService iservice.JWTService,
	userRepository irepository.GenericRepository[model.User],
	loggedSessionRepository irepository.GenericRepository[model.LoggedSession],
) iservice.FacebookSocialService {
	return &FacebookSocialService{
		config:                  config,
		facebookProxy:           facebookProxy,
		channelService:          channelService,
		jwtService:              jwtService,
		userRepository:          userRepository,
		loggedSessionRepository: loggedSessionRepository,
	}
}
