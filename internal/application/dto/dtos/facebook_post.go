package dtos

// region: ============== Facebook Post Request DTOs ==============

type UnpublishedContentType string

const (
	UnpublishedContentTypeScheduled = "SCHEDULED"
)

// FacebookTextPostPublishRequest represents the request payload for publishing a Facebook post
// It can include message text, link, and scheduling options
// If 'Published' is false, 'ScheduledPublishTime' must be provided with the formats:
//   - Unix timestamp (e.g. 1530432000)
//   - ISO 8061 timestamp string (e.g. 2018-09-01T10:15:30+01:00)
//   - Any string parsable by PHP's strtotime() (e.g. +2 weeks, tomorrow)
//
// ScheduledPublishTime must be within 10 minutes to 30 days from the request time
// Reference: [https://developers.facebook.com/docs/pages-api/posts]
type FacebookTextPostPublishRequest struct {
	PageID                 string                 `json:"page_id"` // Facebook Page ID where the post will be published
	Message                string                 `json:"message,omitempty"`
	Link                   string                 `json:"link,omitempty"`
	Published              bool                   `json:"published,omitempty"`              // true for published, false for scheduled/draft
	ScheduledPublishTime   int64                  `json:"scheduled_publish_time,omitempty"` // Unix timestamp for scheduling
	UnpublishedContentType UnpublishedContentType `json:"unpublished_content_type,omitempty"`
	PageAccessToken        string                 `json:"page_access_token"`
}

type FacebookPhotoPostPublishRequest struct {
	PageID                 string                 `json:"page_id"` // Facebook Page ID where the post will be published
	Caption                string                 `json:"caption,omitempty"`
	Published              bool                   `json:"published,omitempty"`              // true for published, false for scheduled/draft
	ScheduledPublishTime   int64                  `json:"scheduled_publish_time,omitempty"` // Unix timestamp for scheduling
	UnpublishedContentType UnpublishedContentType `json:"unpublished_content_type,omitempty"`

	URL           string   `json:"url,omitempty"`            // For single photo post
	AttachedMedia []string `json:"attached_media,omitempty"` // For multiple photo post, list of media_fbid
}

// FacebookImageUploadRequest represents the request payload for uploading an image to Facebook
// For uploading file, the [Published] field should be false
// If the file is for a scheduled post, set [Temporary] to true
type FacebookImageUploadRequest struct {
	PageID    string `json:"page_id"`
	URL       string `json:"url"`
	Published bool   `json:"published,omitempty"`
	Temporary bool   `json:"temporary,omitempty"`
}

// FacebookVideoPostPublishRequest represents the request payload for publishing a Facebook video post
type FacebookVideoPostPublishRequest struct {
	PageID                 string                  `json:"page_id"` // Facebook Page ID where the post will be published
	Title                  string                  `json:"title,omitempty"`
	Description            string                  `json:"description,omitempty"`
	FileURL                string                  `json:"file_url"`                         // URL to the uploaded video file
	Published              bool                    `json:"published,omitempty"`              // true for published, false for scheduled/draft
	ScheduledPublishTime   int64                   `json:"scheduled_publish_time,omitempty"` // Unix timestamp for scheduling
	UnpublishedContentType *UnpublishedContentType `json:"unpublished_content_type,omitempty"`
	SocialActions          bool                    `json:"social_actions,omitempty"` // Whether to enable social actions (likes, comments, shares)
	Secret                 bool                    `json:"secret,omitempty"`         // Whether the video is secret (unlisted)
}

// endregion

// region: ============== Facebook Post Response DTOs ==============

// FacebookPostResponse represents the response from Facebook when creating a post
type FacebookPostResponse struct {
	ID           string `json:"id"`                      // Format: {page-id}_{post-id}
	PostID       string `json:"post_id,omitempty"`       // Extracted post ID only
	PermalinkURL string `json:"permalink_url,omitempty"` // Full URL to the post
}

// FacebookVideoUploadInitResponse represents the response when initializing a video upload
type FacebookVideoUploadInitResponse struct {
	UploadSessionID string `json:"id"` // Format: "upload:<UPLOAD_SESSION_ID>"
}

// FacebookVideoUploadResponse represents the response after uploading video chunks
type FacebookVideoUploadResponse struct {
	FileHandle string `json:"h"` // File handle for publishing (e.g., "2:c2FtcGxl...")
}

// FacebookVideoUploadStatusResponse represents the response when checking upload status
type FacebookVideoUploadStatusResponse struct {
	ID         string `json:"id"`          // Upload session ID
	FileOffset int64  `json:"file_offset"` // Bytes already uploaded (resume from this point)
}

// FacebookErrorResponse represents error responses from Facebook Graph API
type FacebookErrorResponse struct {
	Error FacebookError `json:"error"`
}

type FacebookError struct {
	Message      string `json:"message"`
	Type         string `json:"type"`
	Code         int    `json:"code"`
	ErrorSubcode int    `json:"error_subcode,omitempty"`
	FBTraceID    string `json:"fbtrace_id,omitempty"`
}

// endregion
