package dtos

import (
	"core-backend/internal/domain/model"
	"time"
)

// region: 1. ========= TikTok Enums =========

// region: 2. ========= TikTok Error Codes =========

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

// endregion 2.

// region: 2. ========= Privacy Level Options =========

type TikTokPrivacyLevelOption string

const (
	TikTokPrivacyLevelPublicToEveryone    TikTokPrivacyLevelOption = "PUBLIC_TO_EVERYONE"
	TikTokPrivacyLevelFollowerOfCreator   TikTokPrivacyLevelOption = "FOLLOWER_OF_CREATOR"
	TikTokPrivacyLevelMutualFollowFriends TikTokPrivacyLevelOption = "MUTUAL_FOLLOW_FRIENDS"
	TikTokPrivacyLevelSelfOnly            TikTokPrivacyLevelOption = "SELF_ONLY"
)

func (p TikTokPrivacyLevelOption) String() string { return string(p) }

// endregion 2.

// region: 2. ========= Video Post Status =========

type TikTokVideoPostStatus string

const (
	TikTokVideoPostStatusProcessingUpload   TikTokVideoPostStatus = "PROCESSING_UPLOAD"   // For FILE_UPLOAD
	TikTokVideoPostStatusProcessingDownload TikTokVideoPostStatus = "PROCESSING_DOWNLOAD" // For PULL_FROM_URL
	TikTokVideoPostStatusSendToUserInbox    TikTokVideoPostStatus = "SEND_TO_USER_INBOX"
	TikTokVideoPostStatusPublishComplete    TikTokVideoPostStatus = "PUBLISH_COMPLETE"
	TikTokVideoPostStatusFailed             TikTokVideoPostStatus = "FAILED"
)

// endregion 2.

// region: 2. ========= Source Options =========

type TikTokSourceOption string

const (
	TikTokSourceFileUpload  TikTokSourceOption = "FILE_UPLOAD"
	TikTokSourcePullFromURL TikTokSourceOption = "PULL_FROM_URL"
)

// endregion

// endregion 1.

// region: 1. ========= TikTok Requests =========

// TikTokVideoInitRequest represents the request to initialize a video post
type TikTokVideoInitRequest struct {
	PostInfo   TikTokPostInfo   `json:"post_info"`
	SourceInfo TikTokSourceInfo `json:"source_info"`

	// Internal use only, not sent to TikTok
	VideoSize        int64             `json:"-"`
	IsSingleChunk    bool              `json:"-"`
	IsFinalChunk     bool              `json:"-"`
	FileInfoMetadata map[string]string `json:"-"`
}

type TikTokPostInfo struct {
	PrivacyLevel          TikTokPrivacyLevelOption `json:"privacy_level"`            // TikTok Post Privacy Level
	Title                 string                   `json:"title"`                    // Video Title
	DisableDuet           bool                     `json:"disable_duet"`             // Disable Duet
	DisableStitch         bool                     `json:"disable_stitch"`           // Disable Stitch
	DisableComment        bool                     `json:"disable_comment"`          // Disable Comments
	VideoCoverTimestampMs int                      `json:"video_cover_timestamp_ms"` // Timestamp in milliceonds to be used as image cover for video (thumbnail)
	BrandContentToggle    bool                     `json:"brand_content_toggle"`     // True if the content is paid partnership to promote third party brand
	BrandOrganicToggle    bool                     `json:"brand_organic_toggle"`     // True if the content is promoting own brand
	IsAIGC                bool                     `json:"is_aigc"`                  // True if contetn is AI Generated Content
}

type TikTokSourceInfo struct {
	Source          TikTokSourceOption `json:"source"`                      // "FILE_UPLOAD" or "PULL_FROM_URL"
	VideoURL        *string            `json:"video_url,omitempty"`         // For PULL_FROM_URL
	VideoSize       *int64             `json:"video_size,omitempty"`        // For FILE_UPLOAD
	ChunkSize       *int64             `json:"chunk_size,omitempty"`        // For FILE_UPLOAD
	TotalChunkCount *int               `json:"total_chunk_count,omitempty"` // For FILE_UPLOAD
}

// endregion 1.

// region: 1. ========= TikTok Responses =========

// region: 2. ========= TikTok OAuth Token Response =========

type TikTokTokenResponse struct {
	AccessToken      string `json:"access_token"`
	ExpiresIn        int    `json:"expires_in"` // seconds
	RefreshToken     string `json:"refresh_token"`
	RefreshExpiresIn int    `json:"refresh_expires_in"` // seconds
	OpenID           string `json:"open_id"`            // TikTok user identifier
	Scope            string `json:"scope"`
	TokenType        string `json:"token_type"` // "Bearer"
}

type TikTokTokenErrorResponse struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
	LogID            string `json:"log_id"`
}

// endregion 2.

type TikTokErrorResponse struct {
	Code    TikTokErrorCode `json:"code"`
	Message string          `json:"message"`
	LogID   string          `json:"log_id"`
}

type TikTokResponseWrapper[T any] struct {
	Data  T                   `json:"data"`
	Error TikTokErrorResponse `json:"error"`
}

// region: 2. ========= TikTok User Profile =========

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

type TikTokUserProfileResponse TikTokResponseWrapper[TikTokUserProfile]

func (userProfile *TikTokUserProfileResponse) ToMetadata() *model.TikTokOAuthMetadata {
	return &model.TikTokOAuthMetadata{
		User:      userProfile.Data.User,
		UpdatedAt: time.Now(),
	}
}

// endregion 2.

// region: 2. ========= TikTok Videos =========

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

type TikTokVideosResponse TikTokResponseWrapper[TikTokVideos]

// endregion 2.

// region: 2. ========= TikTok Creator Info =========

type TikTokCreatorInfo struct {
	CreatorAvatarURL     string                     `json:"creator_avatar_url"`
	CreatorUsername      string                     `json:"creator_username"`
	CreatorNickname      string                     `json:"creator_nickname"`
	PrivacyLevelOptions  []TikTokPrivacyLevelOption `json:"privacy_level_options"`
	CommentDisabled      bool                       `json:"comment_disabled"`
	DuetDisabled         bool                       `json:"duet_disabled"`
	StitchDisabled       bool                       `json:"stitch_disabled"`
	MaxVideoPostDuration int                        `json:"max_video_post_duration_sec"`
}

// TikTokCreatorInfoResponse represents the creator information from TikTok
type TikTokCreatorInfoResponse TikTokResponseWrapper[TikTokCreatorInfo]

// endregion 2.

// region: 2. ========= TikTok Video Initialization (Content Posting - Direct Post) =========

type TikTokVideoInitData struct {
	PublishID string `json:"publish_id"`
	UploadURL string `json:"upload_url"` // Valid for 1 hour
}

// TikTokVideoInitResponse represents the response after initializing a video post
type TikTokVideoInitResponse TikTokResponseWrapper[TikTokVideoInitData]

// endregion 2.

// region: 2. ========= TikTok Post Status =========

type TikTokPostStatusData struct {
	Status                  TikTokVideoPostStatus `json:"status"`
	PubliclyAvailablePostID []int64               `json:"publicaly_available_post_id"`
	FailReason              *string               `json:"fail_reason,omitempty"`
	UploadedBytes           *int64                `json:"uploaded_bytes,omitempty"`   // For FILE_UPLOAD
	DownloadedBytes         *int64                `json:"downloaded_bytes,omitempty"` // For PULL_FROM_URL
}

// TikTokPostStatusResponse represents the status of a TikTok post
type TikTokPostStatusResponse TikTokResponseWrapper[TikTokPostStatusData]

// endregion 2.

// endregion 1.
