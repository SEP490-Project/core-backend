package iservice

import (
	"context"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"

	"github.com/google/uuid"
)

// ContentService handles content management business logic
type ContentService interface {
	// Create creates new content with DRAFT status
	Create(ctx context.Context, req *requests.CreateContentRequest) (*responses.ContentResponse, error)

	// Update updates existing content (DRAFT or REJECTED status only)
	Update(ctx context.Context, id uuid.UUID, req *requests.UpdateContentRequest) (*responses.ContentResponse, error)

	// GetByID retrieves content by ID with relationships
	GetByID(ctx context.Context, id uuid.UUID) (*responses.ContentResponse, error)

	// Delete soft deletes content (DRAFT or REJECTED status only)
	Delete(ctx context.Context, id uuid.UUID) error

	// List retrieves paginated content with filters and search
	List(ctx context.Context, req *requests.ContentListRequest) ([]*responses.ContentResponse, int64, error)

	// Submit submits content for review (simplified signature for MVP)
	Submit(ctx context.Context, contentID uuid.UUID, submitterID uuid.UUID) error

	// Approve approves submitted content (simplified signature for MVP)
	Approve(ctx context.Context, contentID uuid.UUID, approverID uuid.UUID, comment string) error

	// Reject rejects submitted content (simplified signature for MVP)
	Reject(ctx context.Context, contentID uuid.UUID, reviewerID uuid.UUID, reason string) error

	// Publish publishes approved content with optional publish date
	Publish(ctx context.Context, contentID uuid.UUID, publisherID uuid.UUID, publishDate *string) error
}
