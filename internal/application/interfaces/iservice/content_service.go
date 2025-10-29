package iservice

import (
	"context"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/domain/enum"

	"github.com/google/uuid"
)

// ContentService handles content management business logic
type ContentService interface {
	// Create creates new content with DRAFT status
	Create(ctx context.Context, uow irepository.UnitOfWork, req *requests.CreateContentRequest) (*responses.ContentResponse, error)

	// Update updates existing content (DRAFT or REJECTED status only)
	Update(ctx context.Context, id uuid.UUID, req *requests.UpdateContentRequest) (*responses.ContentResponse, error)

	// GetByID retrieves content by ID with relationships
	GetByID(ctx context.Context, id uuid.UUID) (*responses.ContentResponse, error)

	// Delete soft deletes content (DRAFT or REJECTED status only)
	Delete(ctx context.Context, id uuid.UUID) error

	// List retrieves paginated content with filters and search
	List(ctx context.Context, req *requests.ContentFilterRequest) ([]*responses.ContentResponse, int64, error)

	// SetRejectionFeedback stores rejection feedback for a content item (transaction-aware)
	SetRejectionFeedback(ctx context.Context, uow irepository.UnitOfWork, contentID uuid.UUID, feedback string) error

	// SetPublishDate stores publish date for a content item (transaction-aware)
	SetPublishDate(ctx context.Context, uow irepository.UnitOfWork, contentID uuid.UUID, publishDate *string) error

	// ValidateForSubmission validates content is ready for submission
	ValidateForSubmission(ctx context.Context, contentID uuid.UUID) error

	// DetermineWorkflowRoute determines target status based on selected channels
	DetermineWorkflowRoute(ctx context.Context, contentID uuid.UUID) (enum.ContentStatus, error)
}
