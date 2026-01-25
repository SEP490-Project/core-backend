package dtos

import (
	"core-backend/internal/domain/model"
	"time"
)

// FacebookResponseWrapper is a generic wrapper for Facebook API responses
type FacebookResponseWrapper[T any] struct {
	Data   T                  `json:"data"`
	Paging FacebookPagingInfo `json:"paging"`
}

type FacebookPagingInfo struct {
	Cursors FacebookCursorsInfo `json:"cursors"`
}

type FacebookCursorsInfo struct {
	Before string `json:"before"`
	After  string `json:"after"`
}

type FacebookAccountInfo struct {
	AccessToken  string                 `json:"access_token"`
	Category     string                 `json:"category"`
	CategoryList []FacebookCategoryInfo `json:"category_list"`
	Name         string                 `json:"name"`
	ID           string                 `json:"id"`
	Tasks        []string               `json:"tasks"`
}

type FacebookAccountInfoResponse FacebookResponseWrapper[[]FacebookAccountInfo]

type FacebookCategoryInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type FacebookAccessTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"` // seconds till expiration
}

type FacebookUserProfileResponse struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Email   string `json:"email"`
	Picture *struct {
		Data *struct {
			URL string `json:"url"`
		} `json:"data"`
	} `json:"picture"`
	Birthday *string `json:"birthday,omitempty"`
}

type FacebookUserProfile struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Email   string `json:"email"`
	Picture *struct {
		Data struct {
			URL string `json:"url"`
		} `json:"data"`
	} `json:"picture"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (userProfile *FacebookUserProfileResponse) ToMetadata() *model.FacebookOAuthMetadata {
	return &model.FacebookOAuthMetadata{
		ID:        userProfile.ID,
		Name:      userProfile.Name,
		Email:     userProfile.Email,
		Picture:   userProfile.Picture,
		Birthday:  userProfile.Birthday,
		UpdatedAt: time.Now(),
	}
}

type FacebookPostMetricsResponse struct {
	Data []FacebookMetricData `json:"data"`
}

type FacebookPageInsightsResponse struct {
	Data []FacebookMetricData `json:"data"`
}

type FacebookMetricData struct {
	Name        string                `json:"name"`
	Period      string                `json:"period"`
	Values      []FacebookMetricValue `json:"values"`
	Title       string                `json:"title"`
	Description string                `json:"description"`
	ID          string                `json:"id"`
}

type FacebookMetricValue struct {
	Value   any    `json:"value"` // Can be int or map[string]int
	EndTime string `json:"end_time"`
}

type FacebookInsightsPeriod string

const (
	FacebookInsightsPeriodDay      FacebookInsightsPeriod = "day"
	FacebookInsightsPeriodWeek     FacebookInsightsPeriod = "week"
	FacebookInsightsPeriodDays28   FacebookInsightsPeriod = "days_28"
	FacebookInsightsPeriodLifetime FacebookInsightsPeriod = "lifetime"
)

type FacebookVideoInsightsResponse struct {
	Data []FacebookMetricData `json:"data"`
}

// FacebookPageInfoResponse represents page-level metrics from Facebook Graph API
// Used by GET /{page-id}?fields=fan_count,followers_count,...
type FacebookPageInfoResponse struct {
	ID             string `json:"id"`
	Name           string `json:"name,omitempty"`
	FanCount       int    `json:"fan_count"`       // Total page likes
	FollowersCount int    `json:"followers_count"` // Total page followers
}

// FacebookPagePostsResponse represents paginated list of posts from a Facebook page
// Used by GET /{page-id}/posts?fields=...
type FacebookPagePostsResponse struct {
	Data   []FacebookPagePost     `json:"data"`
	Paging *FacebookPagingWithURL `json:"paging,omitempty"`
}

// FacebookPagingWithURL extends paging info with next/previous URLs
type FacebookPagingWithURL struct {
	Cursors  FacebookCursorsInfo `json:"cursors"`
	Next     *string             `json:"next,omitempty"`     // URL for next page
	Previous *string             `json:"previous,omitempty"` // URL for previous page
}

// FacebookPagePost represents a single post from a Facebook page
type FacebookPagePost struct {
	ID          string                `json:"id"`
	Message     string                `json:"message,omitempty"`
	CreatedTime string                `json:"created_time"`
	Reactions   *FacebookSummaryCount `json:"reactions,omitempty"`
	Comments    *FacebookSummaryCount `json:"comments,omitempty"`
	Shares      *FacebookSharesCount  `json:"shares,omitempty"`
	// insights
	Insights    *FacebookPostInsights `json:"insights,omitempty"`
	Attachments *FacebookAttachments  `json:"attachments,omitempty"`
}

// FacebookSummaryCount represents a count with summary (used for reactions, comments)
type FacebookSummaryCount struct {
	Summary struct {
		TotalCount int `json:"total_count"`
	} `json:"summary"`
}

// FacebookSharesCount represents shares count structure
type FacebookSharesCount struct {
	Count int `json:"count"`
}

// FacebookPostInsights represents insights data for a Facebook post
type FacebookPostInsights FacebookPageInsightsResponse

// FacebookAttachments represents media attachments on a post
type FacebookAttachments struct {
	Data []FacebookAttachment `json:"data"`
}

// FacebookAttachment represents a single attachment (photo, video, etc.)
type FacebookAttachment struct {
	MediaType string                    `json:"media_type"` // "video", "photo", "album", etc.
	Type      string                    `json:"type"`       // "video_inline", "photo", etc.
	Target    *FacebookAttachmentTarget `json:"target,omitempty"`
	URL       string                    `json:"url,omitempty"`
}

// FacebookAttachmentTarget contains the ID of the attachment (e.g., video ID)
type FacebookAttachmentTarget struct {
	ID  string `json:"id"`
	URL string `json:"url,omitempty"`
}

// FacebookPagePostIDFromVideoID represents the response from querying a post ID by video ID
type FacebookPagePostIDFromVideoID struct {
	ID          string `json:"id"`
	PageStoryID string `json:"page_story_id"`
}
