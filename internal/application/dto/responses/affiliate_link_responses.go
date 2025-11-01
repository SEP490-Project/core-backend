package responses

import (
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// AffiliateLinkResponse represents the response for an affiliate link
type AffiliateLinkResponse struct {
	ID          uuid.UUID                `json:"id"`
	Hash        string                   `json:"hash"`
	ShortURL    string                   `json:"short_url"` // e.g., "https://domain.com/r/{hash}"
	ContractID  uuid.UUID                `json:"contract_id"`
	ContentID   uuid.UUID                `json:"content_id"`
	ChannelID   uuid.UUID                `json:"channel_id"`
	TrackingURL string                   `json:"tracking_url"`
	Status      enum.AffiliateLinkStatus `json:"status"`
	CreatedAt   *time.Time               `json:"created_at"`
	UpdatedAt   *time.Time               `json:"updated_at"`

	// Optional nested objects (using existing summary types)
	Contract *ContractSummary      `json:"contract,omitempty"`
	Content  *AffiliateLinkContent `json:"content,omitempty"`
	Channel  *AffiliateLinkChannel `json:"channel,omitempty"`
}

// AffiliateLinkContent represents minimal content info in affiliate link response
type AffiliateLinkContent struct {
	ID    uuid.UUID `json:"id"`
	Title *string   `json:"title"`
}

// AffiliateLinkChannel represents minimal channel info in affiliate link response
type AffiliateLinkChannel struct {
	ID   uuid.UUID `json:"id"`
	Name *string   `json:"name"`
}

// AffiliateLinkListResponse represents a paginated list of affiliate links
type AffiliateLinkListResponse struct {
	Links      []AffiliateLinkResponse `json:"links"`
	Pagination Pagination              `json:"pagination"`
}

func (AffiliateLinkResponse) ToResponse(link *model.AffiliateLink, baseURL string) *AffiliateLinkResponse {
	resp := &AffiliateLinkResponse{
		ID:          link.ID,
		Hash:        link.Hash,
		ShortURL:    fmt.Sprintf("%s/r/%s", baseURL, link.Hash),
		ContractID:  link.ContractID,
		ContentID:   link.ContentID,
		ChannelID:   link.ChannelID,
		TrackingURL: link.TrackingURL,
		Status:      link.Status,
		CreatedAt:   link.CreatedAt,
		UpdatedAt:   link.UpdatedAt,
	}

	// Include related entities if preloaded
	if link.Contract != nil {
		contractNumber := ""
		if link.Contract.ContractNumber != nil {
			contractNumber = *link.Contract.ContractNumber
		}
		title := ""
		if link.Contract.Title != nil {
			title = *link.Contract.Title
		}

		resp.Contract = &ContractSummary{
			ID:             link.Contract.ID.String(),
			ContractNumber: contractNumber,
			Title:          title,
			Type:           string(link.Contract.Type),
			Status:         string(link.Contract.Status),
		}
	}

	if link.Content != nil {
		resp.Content = &AffiliateLinkContent{
			ID:    link.Content.ID,
			Title: &link.Content.Title,
		}
	}

	if link.Channel != nil {
		resp.Channel = &AffiliateLinkChannel{
			ID:   link.Channel.ID,
			Name: &link.Channel.Name,
		}
	}

	return resp
}
