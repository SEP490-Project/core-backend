package service

import (
	"context"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/application/interfaces/iservice"
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
	"core-backend/pkg/utils"
	"errors"

	"github.com/google/uuid"
	"go.uber.org/zap"
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
func (s *BlogService) UpdateBlogDetails(ctx context.Context, uow irepository.UnitOfWork, contentID uuid.UUID, req *requests.UpdateBlogRequest) error {
	zap.L().Info("BlogService - UpdateBlogDetails called", zap.String("content_id", contentID.String()))
	blogRepo := uow.Blogs()
	contentRepo := uow.Contents()
	tagRepo := uow.Tags()

	// Retrieve content to validate it exists and is POST type
	content, err := contentRepo.GetByID(ctx, contentID, nil)
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
	blog, err := blogRepo.GetByCondition(ctx,
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
		creatingTags := utils.MapSlice(req.Tags, func(tag string) model.Tag {
			return model.Tag{
				Name:        tag,
				Description: &tag,
			}
		})
		createdTags, err := tagRepo.CreateIfNotExists(ctx, creatingTags)
		if err != nil {
			zap.L().Error("Failed to create tags", zap.Error(err))
			return errors.New("failed to create tags")
		}
		blog.Tags = createdTags
		updated = true
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
		if err := blogRepo.Update(ctx, blog); err != nil {
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
