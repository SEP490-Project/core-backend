package iservice

import (
	"context"
	"core-backend/internal/application/dto/requests"

	"github.com/google/uuid"
)

// BlogService handles blog-specific operations for POST type content
type BlogService interface {
	// UpdateBlogDetails updates blog-specific attributes (tags, excerpt, read_time)
	UpdateBlogDetails(ctx context.Context, contentID uuid.UUID, req *requests.UpdateBlogRequest) error
}
