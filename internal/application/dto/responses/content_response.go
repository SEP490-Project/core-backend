package responses

import (
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
	"core-backend/pkg/utils"
	"time"

	"github.com/google/uuid"
)

// ContentResponse DTO for content API responses
type ContentResponse struct {
	ID                uuid.UUID             `json:"id"`
	TaskID            *uuid.UUID            `json:"task_id,omitempty"`
	Title             string                `json:"title"`
	Description       *string               `json:"description,omitempty"`
	ThumbnailURL      *string               `json:"thumbnail_url,omitempty"`
	Body              any                   `json:"body"`
	Type              enum.ContentType      `json:"type"`
	Status            enum.ContentStatus    `json:"status"`
	PublishDate       *time.Time            `json:"publish_date,omitempty"`
	AIGeneratedText   *string               `json:"ai_generated_text,omitempty"`
	RejectionFeedback *string               `json:"rejection_feedback,omitempty"`
	CreatedAt         string                `json:"created_at"`
	UpdatedAt         string                `json:"updated_at"`
	Blog              *BlogResponse         `json:"blog,omitempty"`
	ContentChannels   []ContentChannelBrief `json:"content_channels,omitempty"`
}

// ContentChannelBrief for nested content channel info
type ContentChannelBrief struct {
	ID             uuid.UUID                    `json:"id"`
	ChannelID      uuid.UUID                    `json:"channel_id"`
	ChannelName    string                       `json:"channel_name"`
	PostDate       *time.Time                   `json:"post_date,omitempty"`
	AutoPostStatus string                       `json:"auto_post_status"`
	AffiliateLink  *ContentChannelAffiliateInfo `json:"affiliate_link,omitempty"`
	Metrics        *ContentChannelMetric        `json:"metrics,omitempty"`
}

type ContentChannelMetric struct {
	Mapped  map[string]float64 `json:"mapped"`
	Fetched map[string]any     `json:"fetched"`
}

type ContentChannelAffiliateInfo struct {
	AffiliateLink *string `json:"affiliate_link,omitempty"`
	OriginalLink  *string `json:"original_link,omitempty"`
	TotalClicks   *int    `json:"total_clicks,omitempty"`
	Status        *string `json:"status,omitempty"`
}

type ContentListResponse struct {
	ID                uuid.UUID             `json:"id"`
	TaskID            *uuid.UUID            `json:"task_id,omitempty"`
	Title             string                `json:"title"`
	Description       *string               `json:"description,omitempty"`
	ThumbnailURL      *string               `json:"thumbnail_url,omitempty"`
	Type              enum.ContentType      `json:"type"`
	Status            enum.ContentStatus    `json:"status"`
	PublishDate       *time.Time            `json:"publish_date,omitempty"`
	RejectionFeedback *string               `json:"rejection_feedback,omitempty"`
	CreatedAt         string                `json:"created_at"`
	UpdatedAt         string                `json:"updated_at"`
	Blog              *BlogResponse         `json:"blog,omitempty"`
	ContentChannels   []ContentChannelBrief `json:"content_channels,omitempty"`
}

func (ContentResponse) ToResponse(content *model.Content, affiliateLinkBaseURL string) *ContentResponse {
	resp := &ContentResponse{
		ID:                content.ID,
		TaskID:            content.TaskID,
		Title:             content.Title,
		ThumbnailURL:      content.ThumbnailURL,
		Description:       content.Description,
		Body:              content.Body,
		Type:              content.Type,
		Status:            content.Status,
		PublishDate:       content.PublishDate,
		AIGeneratedText:   content.AIGeneratedText,
		RejectionFeedback: content.RejectionFeedback,
		CreatedAt:         utils.FormatLocalTime(content.CreatedAt, ""),
		UpdatedAt:         utils.FormatLocalTime(content.UpdatedAt, ""),
		// AffiliateLink:     content.AffiliateLink,
	}

	if content.Blog != nil {
		var tags []string
		if len(content.Blog.Tags) > 0 {
			tags = utils.MapSlice(content.Blog.Tags, func(tag model.Tag) string { return tag.Name })
		}

		resp.Blog = &BlogResponse{
			ContentID: content.Blog.ContentID,
			AuthorID:  content.Blog.AuthorID,
			Tags:      tags,
			Excerpt:   content.Blog.Excerpt,
			ReadTime:  content.Blog.ReadTime,
			CreatedAt: utils.FormatLocalTime(content.Blog.CreatedAt, ""),
			UpdatedAt: utils.FormatLocalTime(content.Blog.UpdatedAt, ""),
		}

		if content.Blog.Author != nil {
			resp.Blog.Author = &UserBrief{
				ID:       content.Blog.Author.ID,
				Username: content.Blog.Author.Username,
				Email:    content.Blog.Author.Email,
			}
		}
	}

	if len(content.ContentChannels) > 0 {
		resp.ContentChannels = make([]ContentChannelBrief, 0)
		for _, cc := range content.ContentChannels {
			channelName := ""
			if cc.Channel != nil {
				channelName = cc.Channel.Name
			}
			ccResp := ContentChannelBrief{
				ID:             cc.ID,
				ChannelID:      cc.ChannelID,
				ChannelName:    channelName,
				PostDate:       cc.PostDate,
				AutoPostStatus: string(cc.AutoPostStatus),
			}
			if cc.AffiliateLink != nil {
				ccResp.AffiliateLink = &ContentChannelAffiliateInfo{
					AffiliateLink: utils.PtrOrNil(affiliateLinkBaseURL + "/r/" + cc.AffiliateLink.Hash),
					OriginalLink:  utils.PtrOrNil(cc.AffiliateLink.TrackingURL),
					TotalClicks:   utils.PtrOrNil(len(cc.AffiliateLink.ClickEvents)),
					Status:        utils.PtrOrNil(cc.AffiliateLink.Status.String()),
				}
			}
			if modelMetrics, err := cc.GetMetrics(); cc.Metrics != nil && err == nil {
				ccResp.Metrics = &ContentChannelMetric{
					Mapped:  modelMetrics.CurrentMapped,
					Fetched: modelMetrics.CurrentFetched,
				}
			}

			resp.ContentChannels = append(resp.ContentChannels, ccResp)
		}
	}

	return resp

}

func (ContentListResponse) ToResponse(content *model.Content, affiliateLinkBaseURL string) *ContentListResponse {
	if content == nil {
		return nil
	}

	resp := &ContentListResponse{
		ID:                content.ID,
		TaskID:            content.TaskID,
		Title:             content.Title,
		ThumbnailURL:      content.ThumbnailURL,
		Description:       content.Description,
		Type:              content.Type,
		Status:            content.Status,
		PublishDate:       content.PublishDate,
		RejectionFeedback: content.RejectionFeedback,
		CreatedAt:         utils.FormatLocalTime(content.CreatedAt, utils.TimeFormat),
		UpdatedAt:         utils.FormatLocalTime(content.UpdatedAt, utils.TimeFormat),
	}

	if content.Blog != nil {
		var tags []string
		if len(content.Blog.Tags) > 0 {
			tags = utils.MapSlice(content.Blog.Tags, func(tag model.Tag) string { return tag.Name })
		}

		resp.Blog = &BlogResponse{
			ContentID: content.Blog.ContentID,
			AuthorID:  content.Blog.AuthorID,
			Tags:      tags,
			Excerpt:   content.Blog.Excerpt,
			ReadTime:  content.Blog.ReadTime,
			CreatedAt: utils.FormatLocalTime(content.Blog.CreatedAt, utils.TimeFormat),
			UpdatedAt: utils.FormatLocalTime(content.Blog.UpdatedAt, utils.TimeFormat),
		}

		if content.Blog.Author != nil {
			resp.Blog.Author = &UserBrief{
				ID:       content.Blog.Author.ID,
				Username: content.Blog.Author.Username,
				Email:    content.Blog.Author.Email,
			}
		}
	}

	if len(content.ContentChannels) > 0 {
		resp.ContentChannels = make([]ContentChannelBrief, 0)
		for _, cc := range content.ContentChannels {
			channelName := ""
			if cc.Channel != nil {
				channelName = cc.Channel.Name
			}
			ccResp := ContentChannelBrief{
				ID:             cc.ID,
				ChannelID:      cc.ChannelID,
				ChannelName:    channelName,
				PostDate:       cc.PostDate,
				AutoPostStatus: string(cc.AutoPostStatus),
				AffiliateLink:  nil,
				Metrics:        nil,
			}

			resp.ContentChannels = append(resp.ContentChannels, ccResp)
		}
	}

	return resp
}

func (r ContentListResponse) ToResponseList(contents []model.Content, affiliateLinkBaseURL string) []*ContentListResponse {
	if len(contents) == 0 {
		return []*ContentListResponse{}
	}

	resp := make([]*ContentListResponse, len(contents))
	for i, content := range contents {
		resp[i] = r.ToResponse(&content, affiliateLinkBaseURL)
	}
	return resp
}

// ContentPaginationResponse for paginated content responses
// Only used for Swaggo swagger docs generation
type ContentPaginationResponse PaginationResponse[ContentListResponse]
