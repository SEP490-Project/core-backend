package dtos

import (
	"core-backend/internal/domain/model"
	"time"
)

type TikTokErrorCode string

const (
	TikTokOK                    TikTokErrorCode = "ok"
	TikTokAccessTokenInvalid    TikTokErrorCode = "access_token_invalid"
	TikTokInternalError         TikTokErrorCode = "internal_error"
	TikTokInvalidFileUpload     TikTokErrorCode = "invalid_file_upload"
	TikTokInvalidParams         TikTokErrorCode = "invalid_params"
	TikTokRateLimitExceeded     TikTokErrorCode = "rate_limit_exceeded"
	TikTokScopeNotAuthorized    TikTokErrorCode = "scope_not_authorized"
	TikTokScopePermissionMissed TikTokErrorCode = "scope_permission_missed"
)

func (e TikTokErrorCode) IsSuccess() bool {
	return e == TikTokOK
}

func (e TikTokErrorCode) String() string { return string(e) }

type TikTokTokenResponse struct {
	AccessToken      string `json:"access_token"`
	ExpiresIn        int    `json:"expires_in"` // seconds
	RefreshToken     string `json:"refresh_token"`
	RefreshExpiresIn int    `json:"refresh_expires_in"` // seconds
	OpenID           string `json:"open_id"`            // TikTok user identifier
	Scope            string `json:"scope"`
	TokenType        string `json:"token_type"` // "Bearer"
}

type TikTokErrorResponse struct {
	Code    TikTokErrorCode `json:"code"`
	Message string          `json:"message"`
	LogID   string          `json:"log_id"`
}

type TikTokResponseWrapper[T any] struct {
	Data  T                   `json:"data"`
	Error TikTokErrorResponse `json:"error"`
}

// TikTokUserProfile represents the structure of TikTok user profile data
// Needed scope: user.info.basic
// Optional scopes: user.info.profile, user.info.stats
type TikTokUserProfile struct {
	User struct {
		OpenID          string  `json:"open_id"`                     // scope: user.info.basic
		UnionID         string  `json:"union_id"`                    // scope: user.info.basic
		AvatarURL       string  `json:"avatar_url"`                  // scope: user.info.basic
		DisplayName     string  `json:"display_name"`                // scope: user.info.basic
		BioDescription  *string `json:"bio_description,omitempty"`   // scope: user.info.profile
		ProfileDeepLink *string `json:"profile_deep_link,omitempty"` // scope: user.info.profile
		IsVerified      *bool   `json:"is_verified,omitempty"`       // scope: user.info.profile
		UserName        *string `json:"username,omitempty"`          // scope: user.info.profile
		FollowerCount   *int64  `json:"follower_count,omitempty"`    // scope: user.info.stats
		FollowingCount  *int64  `json:"following_count,omitempty"`   // scope: user.info.stats
		LikesCount      *int64  `json:"likes_count,omitempty"`       // scope: user.info.stats
		VideoCount      *int64  `json:"video_count,omitempty"`       // scope: user.info.stats
	} `json:"user"`
}

// TikTokVideos represents the structure of TikTok video data
// Needed scope: video.list
type TikTokVideos struct {
	Videos struct {
		ID               string `json:"id"`
		CreateTime       string `json:"create_time"`
		CoverImageURL    string `json:"cover_image_url"`
		ShareURL         string `json:"share_url"`
		VideoDescription string `json:"video_description"`
		Duration         int64  `json:"duration"`
		Height           int64  `json:"height"`
		Width            int64  `json:"width"`
		Title            string `json:"title"`
		EmbedHTML        string `json:"embed_html"`
		EmbedLink        string `json:"embed_link"`
		LikeCount        int64  `json:"like_count"`
		CommentCount     int64  `json:"comment_count"`
		ShareCount       int64  `json:"share_count"`
		ViewCount        int64  `json:"view_count"`
	} `json:"videos"`
}

type TikTokUserProfileResponse TikTokResponseWrapper[TikTokUserProfile]

type TikTokVideosResponse TikTokResponseWrapper[TikTokVideos]

func (userProfile *TikTokUserProfileResponse) ToMetadata() *model.TikTokOAuthMetadata {
	return &model.TikTokOAuthMetadata{
		User:      userProfile.Data.User,
		UpdatedAt: time.Now(),
	}
}
