package dtos

// TikTokCreatorInfoResponse represents the creator information from TikTok
type TikTokCreatorInfoResponse struct {
	Data  TikTokCreatorInfoData `json:"data"`
	Error TikTokError           `json:"error,omitempty"`
}

type TikTokCreatorInfoData struct {
	CreatorAvatar        TikTokAvatar `json:"creator_avatar"`
	CreatorUsername      string       `json:"creator_username"`
	CreatorNickname      string       `json:"creator_nickname"`
	PrivacyLevelOptions  []string     `json:"privacy_level_options"` // ["SELF_ONLY", "MUTUAL_FOLLOW_FRIENDS", etc.]
	CommentDisabled      bool         `json:"comment_disabled"`
	DuetDisabled         bool         `json:"duet_disabled"`
	StitchDisabled       bool         `json:"stitch_disabled"`
	MaxVideoPostDuration int          `json:"max_video_post_duration_sec"`
}

type TikTokAvatar struct {
	AvatarURL      string `json:"avatar_url"`
	AvatarURL100   string `json:"avatar_url_100"`
	AvatarLargeURL string `json:"avatar_large_url"`
}

// TikTokVideoInitRequest represents the request to initialize a video post
type TikTokVideoInitRequest struct {
	PostInfo   TikTokPostInfo   `json:"post_info"`
	SourceInfo TikTokSourceInfo `json:"source_info"`
}

type TikTokPostInfo struct {
	Title              string `json:"title"`
	PrivacyLevel       string `json:"privacy_level"` // "SELF_ONLY", "MUTUAL_FOLLOW_FRIENDS", "FOLLOWER_OF_CREATOR", "PUBLIC_TO_EVERYONE"
	DisableDuet        bool   `json:"disable_duet"`
	DisableComment     bool   `json:"disable_comment"`
	DisableStitch      bool   `json:"disable_stitch"`
	BrandContentToggle bool   `json:"brand_content_toggle"`
	BrandOrganicToggle bool   `json:"brand_organic_toggle"`
}

type TikTokSourceInfo struct {
	Source          string `json:"source"`                      // "FILE_UPLOAD" or "PULL_FROM_URL"
	VideoURL        string `json:"video_url,omitempty"`         // For PULL_FROM_URL
	VideoSize       int64  `json:"video_size,omitempty"`        // For FILE_UPLOAD
	ChunkSize       int64  `json:"chunk_size,omitempty"`        // For FILE_UPLOAD
	TotalChunkCount int    `json:"total_chunk_count,omitempty"` // For FILE_UPLOAD
}

// TikTokVideoInitResponse represents the response after initializing a video post
type TikTokVideoInitResponse struct {
	Data  TikTokVideoInitData `json:"data"`
	Error TikTokError         `json:"error,omitempty"`
}

type TikTokVideoInitData struct {
	PublishID string `json:"publish_id"`
	UploadURL string `json:"upload_url"` // Valid for 1 hour
}

// TikTokPostStatusResponse represents the status of a TikTok post
type TikTokPostStatusResponse struct {
	Data  TikTokPostStatusData `json:"data"`
	Error TikTokError          `json:"error,omitempty"`
}

type TikTokPostStatusData struct {
	Status                  string  `json:"status"` // "PROCESSING_UPLOAD", "PROCESSING_DOWNLOAD", "PUBLISH_COMPLETE", "FAILED", "PROCESSING_PUBLISH"
	PubliclyAvailablePostID *string `json:"publicly_available_post_id,omitempty"`
	FailReason              *string `json:"fail_reason,omitempty"`
}

// TikTokError represents error responses from TikTok API
type TikTokError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	LogID   string `json:"log_id"`
}
