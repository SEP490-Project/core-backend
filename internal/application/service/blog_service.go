package service

import (
	"context"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/application/interfaces/iservice"
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
	"errors"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type BlogService struct {
	blogRepo    irepository.GenericRepository[model.Blog]
	contentRepo irepository.GenericRepository[model.Content]
}

func NewBlogService(
	blogRepo irepository.GenericRepository[model.Blog],
	contentRepo irepository.GenericRepository[model.Content],
) iservice.BlogService {
	return &BlogService{
		blogRepo:    blogRepo,
		contentRepo: contentRepo,
	}
}

// UpdateBlogDetails updates blog-specific attributes for POST type content
func (s *BlogService) UpdateBlogDetails(ctx context.Context, contentID uuid.UUID, req *requests.UpdateBlogRequest) error {
	// Retrieve content to validate it exists and is POST type
	content, err := s.contentRepo.GetByID(ctx, contentID, nil)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("content not found")
		}
		zap.L().Error("Failed to retrieve content", zap.String("content_id", contentID.String()), zap.Error(err))
		return errors.New("failed to retrieve content")
	}

	// Validate content type is POST
	if content.Type != enum.ContentTypePost {
		return errors.New("blog operations are only allowed for POST type content")
	}

	// Retrieve blog entity by content_id
	blog, err := s.blogRepo.GetByCondition(ctx,
		func(db *gorm.DB) *gorm.DB {
			return db.Where("content_id = ?", contentID)
		},
		nil)

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("blog not found for this content")
		}
		zap.L().Error("Failed to retrieve blog", zap.String("content_id", contentID.String()), zap.Error(err))
		return errors.New("failed to retrieve blog")
	}

	// Update blog fields if provided
	updated := false

	if req.Tags != nil {
		// Convert tags to JSONB
		tagsJSON, err := datatypes.NewJSONType(req.Tags).MarshalJSON()
		if err == nil {
			blog.Tags = tagsJSON
			updated = true
		}
	}

	if req.Excerpt != nil {
		blog.Excerpt = req.Excerpt
		updated = true
	}

	if req.ReadTime != nil {
		blog.ReadTime = req.ReadTime
		updated = true
	}

	// Save changes if any updates were made
	if updated {
		if err := s.blogRepo.Update(ctx, blog); err != nil {
			zap.L().Error("Failed to update blog", zap.String("content_id", contentID.String()), zap.Error(err))
			return errors.New("failed to update blog details")
		}

		zap.L().Info("Blog details updated successfully",
			zap.String("content_id", contentID.String()),
			zap.Bool("tags_updated", req.Tags != nil),
			zap.Bool("excerpt_updated", req.Excerpt != nil),
			zap.Bool("read_time_updated", req.ReadTime != nil))
	}

	return nil
}
