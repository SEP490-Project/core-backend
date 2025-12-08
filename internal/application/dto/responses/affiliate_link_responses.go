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
	TrackingURL string                   `json:"tracking_url"`
	Status      enum.AffiliateLinkStatus `json:"status"`
	CreatedAt   *time.Time               `json:"created_at"`
	UpdatedAt   *time.Time               `json:"updated_at"`
	Metadata    map[string]any           `json:"metadata,omitempty"`

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
		TrackingURL: link.TrackingURL,
		Status:      link.Status,
		CreatedAt:   link.CreatedAt,
		UpdatedAt:   link.UpdatedAt,
	}

	// Include related entities if preloaded
	if link.Contract != nil && link.ContractID != nil {
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
		resp.Metadata["contract_id"] = link.GetContractID().String()
	}

	if link.Content != nil && link.ContentID != nil {
		resp.Content = &AffiliateLinkContent{
			ID:    link.Content.ID,
			Title: &link.Content.Title,
		}
		resp.Metadata["content_id"] = link.GetContentID().String()
	}

	if link.Channel != nil && link.ChannelID != nil {
		resp.Channel = &AffiliateLinkChannel{
			ID:   link.Channel.ID,
			Name: &link.Channel.Name,
		}
		resp.Metadata["channel_id"] = link.GetChannelID().String()
	}

	return resp
}
