package irepository

import (
	"context"
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"

	"github.com/google/uuid"
)

type ContentRepository interface {
	GenericRepository[model.Content]

	GetContentByIDWIthDetails(ctx context.Context, id uuid.UUID) (*model.Content, error)

	// GetContentIDsByCampaignID returns all content IDs associated with tasks in a campaign
	GetContentIDsByCampaignID(ctx context.Context, campaignID uuid.UUID, excludeStatus ...enum.ContentStatus) ([]uuid.UUID, error)
}
