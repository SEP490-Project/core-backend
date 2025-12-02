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
