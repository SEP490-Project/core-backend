package dtos

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// ChannelMetricsDTO represents aggregated metrics for a channel from the database
type ChannelMetricsDTO struct {
	ChannelID        uuid.UUID                       `json:"channel_id"`
	ChannelName      string                          `json:"channel_name"`
	ChannelCode      string                          `json:"channel_code"`
	PostCount        int64                           `json:"post_count"`
	TotalViews       int64                           `json:"total_views"`
	TotalLikes       int64                           `json:"total_likes"`
	TotalComments    int64                           `json:"total_comments"`
	TotalShares      int64                           `json:"total_shares"`
	TotalEngagement  int64                           `json:"total_engagement"`
	TotalReach       int64                           `json:"total_reach"`
	TotalClicks      int64                           `json:"total_clicks"`
	TotalImpressions int64                           `json:"total_impressions"`
	AverageCTR       float64                         `json:"average_ctr"`
	FollowersCount   int64                           `json:"followers_count"`                                                    // Channel followers from Channel.Metrics
	FetchedMetrics   ChannelMetricsDTOFetchedMetrics `json:"fetched_metrics,omitempty" gorm:"type:jsonb;column:fetched_metrics"` // Aggregated raw platform metrics
	MappedMetrics    ChannelMetricsDTOMappedMetrics  `json:"mapped_metrics,omitempty" gorm:"type:jsonb;column:mapped_metrics"`   // Mapped standardized metrics
}

type ChannelMetricsDTOFetchedMetrics map[string]any

func (fm *ChannelMetricsDTOFetchedMetrics) Scan(value any) error {
	if value == nil {
		*fm = nil
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, fm)
}

type ChannelMetricsDTOMappedMetrics map[string]float64

func (mm *ChannelMetricsDTOMappedMetrics) Scan(value any) error {
	if value == nil {
		*mm = nil
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, mm)
}

// TopPostDTO represents the top performing post for a channel
type TopPostDTO struct {
	ContentID uuid.UUID `json:"content_id"`
	Title     string    `json:"title"`
	Views     int64     `json:"views"`
	Likes     int64     `json:"likes"`
	Comments  int64     `json:"comments"`
	Shares    int64     `json:"shares"`
}

// TrendDataPointDTO represents a single point in a time series trend
type TrendDataPointDTO struct {
	Date        time.Time `json:"date"`
	Views       int64     `json:"views"`
	Likes       int64     `json:"likes"`
	Comments    int64     `json:"comments"`
	Shares      int64     `json:"shares"`
	Engagements int64     `json:"engagements"`
}

// ContentTypeDistributionDTO represents content count by type
type ContentTypeDistributionDTO struct {
	ContentType string  `json:"content_type"` // "POST", "VIDEO"
	Count       int64   `json:"count"`
	Percentage  float64 `json:"percentage"`
}

// ChannelDistributionDTO represents content count by channel (for pie chart)
type ChannelDistributionDTO struct {
	ChannelID   uuid.UUID `json:"channel_id"`
	ChannelName string    `json:"channel_name"`
	ChannelCode string    `json:"channel_code"`
	Count       int64     `json:"count"`
	Percentage  float64   `json:"percentage"`
}

// ContentPerformanceDTO represents content performance for ranking
type ContentPerformanceDTO struct {
	ContentID        uuid.UUID  `json:"content_id"`
	Title            string     `json:"title"`
	ContentType      string     `json:"content_type"`
	ChannelID        uuid.UUID  `json:"channel_id"`
	ChannelName      string     `json:"channel_name"`
	Views            int64      `json:"views"`
	Likes            int64      `json:"likes"`
	Comments         int64      `json:"comments"`
	Shares           int64      `json:"shares"`
	Engagement       int64      `json:"engagement"`
	CTR              float64    `json:"ctr"`
	PerformanceScore float64    `json:"performance_score"`
	PublishedAt      *time.Time `json:"published_at"`
	ThumbnailURL     *string    `json:"thumbnail_url"`
}

// ScheduleDTO represents schedule data from database query
type ScheduleDTO struct {
	ScheduleID       uuid.UUID  `json:"schedule_id"`
	ContentChannelID uuid.UUID  `json:"content_channel_id"`
	ContentID        uuid.UUID  `json:"content_id"`
	ContentTitle     string     `json:"content_title"`
	ContentType      string     `json:"content_type"`
	ChannelID        uuid.UUID  `json:"channel_id"`
	ChannelName      string     `json:"channel_name"`
	ChannelCode      string     `json:"channel_code"`
	ScheduledAt      time.Time  `json:"scheduled_at"`
	Status           string     `json:"status"`
	RetryCount       int        `json:"retry_count"`
	LastError        *string    `json:"last_error"`
	ExecutedAt       *time.Time `json:"executed_at"`
	CreatedAt        time.Time  `json:"created_at"`
	CreatedBy        uuid.UUID  `json:"created_by"`
	CreatedByName    string     `json:"created_by_name"`
}
