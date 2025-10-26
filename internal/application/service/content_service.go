package service

import (
	"context"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/application/interfaces/iservice"
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type ContentService struct {
	contentRepo        irepository.GenericRepository[model.Content]
	blogRepo           irepository.GenericRepository[model.Blog]
	contentChannelRepo irepository.GenericRepository[model.ContentChannel]
	channelRepo        irepository.GenericRepository[model.Channel]
	taskRepo           irepository.GenericRepository[model.Task]
	uow                irepository.UnitOfWork
}

func NewContentService(
	contentRepo irepository.GenericRepository[model.Content],
	blogRepo irepository.GenericRepository[model.Blog],
	contentChannelRepo irepository.GenericRepository[model.ContentChannel],
	channelRepo irepository.GenericRepository[model.Channel],
	taskRepo irepository.GenericRepository[model.Task],
	uow irepository.UnitOfWork,
) iservice.ContentService {
	return &ContentService{
		contentRepo:        contentRepo,
		blogRepo:           blogRepo,
		contentChannelRepo: contentChannelRepo,
		channelRepo:        channelRepo,
		taskRepo:           taskRepo,
		uow:                uow,
	}
}

// Create creates new content with DRAFT status
func (s *ContentService) Create(ctx context.Context, req *requests.CreateContentRequest) (*responses.ContentResponse, error) {
	zap.L().Info("Creating content", zap.String("title", req.Title), zap.String("type", req.Type))

	// Validate task exists if provided
	if req.TaskID != nil {
		_, err := s.taskRepo.GetByID(ctx, *req.TaskID, nil)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, errors.New("task not found")
			}
			return nil, err
		}
	}

	// Start transaction
	uow := s.uow.Begin()
	defer func() {
		if r := recover(); r != nil {
			uow.Rollback()
			panic(r)
		}
	}()

	// Create content entity
	content := &model.Content{
		TaskID:          req.TaskID,
		Title:           req.Title,
		Body:            req.Body,
		Type:            enum.ContentType(req.Type),
		Status:          enum.ContentStatusDraft,
		AffiliateLink:   req.AffiliateLink,
		AIGeneratedText: req.AIGeneratedText,
	}

	if err := uow.Contents().Add(ctx, content); err != nil {
		_ = uow.Rollback()
		zap.L().Error("Failed to create content", zap.Error(err))
		return nil, errors.New("failed to create content")
	}

	// Create blog if type is POST
	if content.Type == enum.ContentTypePost && req.BlogFields != nil {
		tagsJSON, err := json.Marshal(req.BlogFields.Tags)
		if err != nil {
			_ = uow.Rollback()
			return nil, errors.New("invalid tags format")
		}

		blog := &model.Blog{
			ContentID: content.ID,
			AuthorID:  req.BlogFields.AuthorID,
			Tags:      datatypes.JSON(tagsJSON),
			Excerpt:   req.BlogFields.Excerpt,
			ReadTime:  req.BlogFields.ReadTime,
		}

		if err := uow.Blogs().Add(ctx, blog); err != nil {
			_ = uow.Rollback()
			zap.L().Error("Failed to create blog", zap.Error(err))
			return nil, errors.New("failed to create blog")
		}
	}

	// Create ContentChannel records
	for _, channelID := range req.Channels {
		contentChannel := &model.ContentChannel{
			ContentID:      content.ID,
			ChannelID:      channelID,
			AutoPostStatus: "PENDING",
		}

		if err := s.contentChannelRepo.Add(ctx, contentChannel); err != nil {
			_ = uow.Rollback()
			zap.L().Error("Failed to create content channel", zap.Error(err))
			return nil, errors.New("failed to create content channel")
		}
	}

	// Commit transaction
	if err := uow.Commit(); err != nil {
		zap.L().Error("Failed to commit transaction", zap.Error(err))
		return nil, errors.New("failed to create content")
	}

	zap.L().Info("Content created successfully", zap.String("content_id", content.ID.String()))
	return s.GetByID(ctx, content.ID)
}

// GetByID retrieves content by ID with relationships
func (s *ContentService) GetByID(ctx context.Context, id uuid.UUID) (*responses.ContentResponse, error) {
	content, err := s.contentRepo.GetByCondition(ctx,
		func(db *gorm.DB) *gorm.DB {
			return db.Where("id = ?", id).
				Preload("Blog").
				Preload("Blog.Author").
				Preload("ContentChannels").
				Preload("ContentChannels.Channel")
		},
		nil,
	)

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("content not found")
		}
		return nil, err
	}

	return s.mapToContentResponse(content), nil
}

// Update updates existing content
func (s *ContentService) Update(ctx context.Context, id uuid.UUID, req *requests.UpdateContentRequest) (*responses.ContentResponse, error) {
	content, err := s.contentRepo.GetByID(ctx, id, nil)
	if err != nil {
		return nil, errors.New("content not found")
	}

	if content.Status != enum.ContentStatusDraft && content.Status != enum.ContentStatusRejected {
		return nil, errors.New("only DRAFT or REJECTED content can be updated")
	}

	if req.Title != nil {
		content.Title = *req.Title
	}
	if req.Body != nil {
		content.Body = *req.Body
	}

	if err := s.contentRepo.Update(ctx, content); err != nil {
		return nil, errors.New("failed to update content")
	}

	return s.GetByID(ctx, id)
}

// Delete soft deletes content
func (s *ContentService) Delete(ctx context.Context, id uuid.UUID) error {
	content, err := s.contentRepo.GetByID(ctx, id, nil)
	if err != nil {
		return errors.New("content not found")
	}

	if content.Status != enum.ContentStatusDraft && content.Status != enum.ContentStatusRejected {
		return errors.New("only DRAFT or REJECTED content can be deleted")
	}

	return s.contentRepo.Delete(ctx, content)
}

// Helper methods
func (s *ContentService) mapToContentResponse(content *model.Content) *responses.ContentResponse {
	resp := &responses.ContentResponse{
		ID:                content.ID,
		TaskID:            content.TaskID,
		Title:             content.Title,
		Body:              content.Body,
		Type:              content.Type,
		Status:            content.Status,
		PublishDate:       content.PublishDate,
		AffiliateLink:     content.AffiliateLink,
		AIGeneratedText:   content.AIGeneratedText,
		RejectionFeedback: content.RejectionFeedback,
		CreatedAt:         content.CreatedAt,
		UpdatedAt:         content.UpdatedAt,
	}

	if content.Blog != nil {
		var tags []string
		_ = json.Unmarshal(content.Blog.Tags, &tags)

		resp.Blog = &responses.BlogResponse{
			ContentID: content.Blog.ContentID,
			AuthorID:  content.Blog.AuthorID,
			Tags:      tags,
			Excerpt:   content.Blog.Excerpt,
			ReadTime:  content.Blog.ReadTime,
			CreatedAt: content.Blog.CreatedAt,
			UpdatedAt: content.Blog.UpdatedAt,
		}

		if content.Blog.Author != nil {
			resp.Blog.Author = &responses.UserBrief{
				ID:       content.Blog.Author.ID,
				Username: content.Blog.Author.Username,
				Email:    content.Blog.Author.Email,
			}
		}
	}

	if len(content.ContentChannels) > 0 {
		resp.ContentChannels = make([]responses.ContentChannelBrief, 0)
		for _, cc := range content.ContentChannels {
			channelName := ""
			if cc.Channel != nil {
				channelName = cc.Channel.Name
			}

			resp.ContentChannels = append(resp.ContentChannels, responses.ContentChannelBrief{
				ID:             cc.ID,
				ChannelID:      cc.ChannelID,
				ChannelName:    channelName,
				PostDate:       cc.PostDate,
				AutoPostStatus: string(cc.AutoPostStatus),
			})
		}
	}

	return resp
}

// DetermineWorkflowRoute determines the target status based on selected channels
func (s *ContentService) determineWorkflowRoute(ctx context.Context, contentID uuid.UUID) (enum.ContentStatus, error) {
	// Get content channels with channel preload
	contentChannels, _, err := s.contentChannelRepo.GetAll(ctx,
		func(db *gorm.DB) *gorm.DB {
			return db.Where("content_id = ?", contentID).Preload("Channel")
		},
		nil, 0, 0,
	)
	if err != nil {
		return "", errors.New("failed to retrieve content channels")
	}

	// Check if any channel is FACEBOOK or TIKTOK (external brand channels)
	for _, cc := range contentChannels {
		if cc.Channel != nil {
			channelName := cc.Channel.Name
			if channelName == "FACEBOOK" || channelName == "TIKTOK" {
				return enum.ContentStatusAwaitBrand, nil
			}
		}
	}

	// Default to internal staff review
	return enum.ContentStatusAwaitStaff, nil
}

// ValidateAffiliateLink validates affiliate link requirement for AFFILIATE contracts
func (s *ContentService) validateAffiliateLink(ctx context.Context, content *model.Content) error {
	// If no task association, skip validation
	if content.TaskID == nil {
		return nil
	}

	// Get task with contract information
	task, err := s.taskRepo.GetByID(ctx, *content.TaskID, nil)
	if err != nil {
		zap.L().Warn("Failed to retrieve task for affiliate link validation",
			zap.String("task_id", content.TaskID.String()),
			zap.Error(err))
		return nil // Don't fail submission if task not found
	}

	// Parse task description to get contract info
	if task.Description != nil {
		var descMap map[string]interface{}
		if err := json.Unmarshal(task.Description, &descMap); err == nil {
			if contractType, ok := descMap["contract_type"].(string); ok {
				if contractType == "AFFILIATE" {
					// For AFFILIATE contracts, affiliate_link is required
					if content.AffiliateLink == nil || *content.AffiliateLink == "" {
						return errors.New("affiliate link is required for AFFILIATE contract content")
					}
				}
			}
		}
	}

	return nil
}

// Submit submits content for review with workflow routing
func (s *ContentService) Submit(ctx context.Context, contentID uuid.UUID, submitterID uuid.UUID) error {
	zap.L().Info("Submitting content for review",
		zap.String("content_id", contentID.String()),
		zap.String("submitter_id", submitterID.String()))

	// Get content with relationships
	content, err := s.contentRepo.GetByID(ctx, contentID, []string{"ContentChannels", "ContentChannels.Channel"})
	if err != nil {
		return errors.New("content not found")
	}

	// Validate current status (must be DRAFT or REJECTED)
	if content.Status != enum.ContentStatusDraft && content.Status != enum.ContentStatusRejected {
		return errors.New("only DRAFT or REJECTED content can be submitted")
	}

	// Validate required fields
	if content.Title == "" || content.Body == "" {
		return errors.New("title and body are required fields")
	}

	// Validate affiliate link if needed
	if err := s.validateAffiliateLink(ctx, content); err != nil {
		return err
	}

	// Determine workflow route based on channels
	targetStatus, err := s.determineWorkflowRoute(ctx, contentID)
	if err != nil {
		return err
	}

	// Update content status
	content.Status = targetStatus
	if err := s.contentRepo.Update(ctx, content); err != nil {
		zap.L().Error("Failed to update content status",
			zap.String("content_id", contentID.String()),
			zap.String("target_status", string(targetStatus)),
			zap.Error(err))
		return errors.New("failed to submit content")
	}

	zap.L().Info("Content submitted successfully",
		zap.String("content_id", contentID.String()),
		zap.String("new_status", string(targetStatus)),
		zap.String("submitter_id", submitterID.String()))

	return nil
}

// Approve approves submitted content
func (s *ContentService) Approve(ctx context.Context, contentID uuid.UUID, approverID uuid.UUID, comment string) error {
	zap.L().Info("Approving content",
		zap.String("content_id", contentID.String()),
		zap.String("approver_id", approverID.String()))

	// Get content
	content, err := s.contentRepo.GetByID(ctx, contentID, nil)
	if err != nil {
		return errors.New("content not found")
	}

	// Validate current status (must be AWAIT_STAFF or AWAIT_BRAND)
	if content.Status != enum.ContentStatusAwaitStaff && content.Status != enum.ContentStatusAwaitBrand {
		return errors.New("only content awaiting review can be approved")
	}

	// Update content status to APPROVED
	content.Status = enum.ContentStatusApproved
	if err := s.contentRepo.Update(ctx, content); err != nil {
		zap.L().Error("Failed to update content status to APPROVED",
			zap.String("content_id", contentID.String()),
			zap.Error(err))
		return errors.New("failed to approve content")
	}

	zap.L().Info("Content approved successfully",
		zap.String("content_id", contentID.String()),
		zap.String("approver_id", approverID.String()),
		zap.String("comment", comment))

	return nil
}

// Reject rejects submitted content with feedback
func (s *ContentService) Reject(ctx context.Context, contentID uuid.UUID, reviewerID uuid.UUID, reason string) error {
	zap.L().Info("Rejecting content",
		zap.String("content_id", contentID.String()),
		zap.String("reviewer_id", reviewerID.String()))

	// Get content
	content, err := s.contentRepo.GetByID(ctx, contentID, nil)
	if err != nil {
		return errors.New("content not found")
	}

	// Validate current status (must be AWAIT_STAFF or AWAIT_BRAND)
	if content.Status != enum.ContentStatusAwaitStaff && content.Status != enum.ContentStatusAwaitBrand {
		return errors.New("only content awaiting review can be rejected")
	}

	// Update content status to REJECTED and store feedback
	content.Status = enum.ContentStatusRejected
	content.RejectionFeedback = &reason
	if err := s.contentRepo.Update(ctx, content); err != nil {
		zap.L().Error("Failed to update content status to REJECTED",
			zap.String("content_id", contentID.String()),
			zap.Error(err))
		return errors.New("failed to reject content")
	}

	zap.L().Info("Content rejected successfully",
		zap.String("content_id", contentID.String()),
		zap.String("reviewer_id", reviewerID.String()),
		zap.String("reason", reason))

	return nil
}

// Publish publishes approved content to POSTED status
func (s *ContentService) Publish(ctx context.Context, contentID uuid.UUID, publisherID uuid.UUID, publishDate *string) error {
	// Retrieve content
	content, err := s.contentRepo.GetByID(ctx, contentID, nil)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("content not found")
		}
		return err
	}

	// Validate status is APPROVED
	if content.Status != enum.ContentStatusApproved {
		return errors.New("only approved content can be published")
	}

	// Set publish date
	if publishDate != nil && *publishDate != "" {
		// Parse the provided publish date (ISO8601 format)
		parsedDate, err := time.Parse(time.RFC3339, *publishDate)
		if err != nil {
			return errors.New("invalid publish_date format, use ISO8601 format (e.g., 2006-01-02T15:04:05Z07:00)")
		}
		content.PublishDate = &parsedDate
	} else {
		// Use current timestamp if not provided
		now := time.Now()
		content.PublishDate = &now
	}

	// Update status to POSTED
	content.Status = enum.ContentStatusPosted

	// Save changes
	if err := s.contentRepo.Update(ctx, content); err != nil {
		return errors.New("failed to publish content")
	}

	zap.L().Info("Content published successfully",
		zap.String("content_id", contentID.String()),
		zap.String("publisher_id", publisherID.String()),
		zap.Time("publish_date", *content.PublishDate))

	return nil
}

// List retrieves paginated content with filters, search, and sorting
func (s *ContentService) List(ctx context.Context, req *requests.ContentListRequest) ([]*responses.ContentResponse, int64, error) {
	// Set defaults
	page := 1
	if req.Page > 0 {
		page = req.Page
	}

	limit := 10
	if req.Limit > 0 {
		limit = req.Limit
	}
	if limit > 100 {
		limit = 100 // Max limit enforcement
	}

	// Build filter function
	filterFunc := func(db *gorm.DB) *gorm.DB {
		// Filter by status
		if req.Status != nil && *req.Status != "" {
			db = db.Where("status = ?", *req.Status)
		}

		// Filter by type
		if req.Type != nil && *req.Type != "" {
			db = db.Where("type = ?", *req.Type)
		}

		// Filter by task_id
		if req.TaskID != nil {
			db = db.Where("task_id = ?", *req.TaskID)
		}

		// Filter by channel_id (requires join with content_channels)
		if req.ChannelID != nil {
			db = db.Joins("JOIN content_channels ON content_channels.content_id = contents.id").
				Where("content_channels.channel_id = ?", *req.ChannelID).
				Distinct()
		}

		// Full-text search on title and body
		if req.Search != nil && *req.Search != "" {
			searchPattern := "%" + *req.Search + "%"
			db = db.Where("title ILIKE ? OR body ILIKE ?", searchPattern, searchPattern)
		}

		// Date range filter
		if req.FromDate != nil && *req.FromDate != "" {
			fromDate, err := time.Parse("2006-01-02", *req.FromDate)
			if err == nil {
				db = db.Where("created_at >= ?", fromDate)
			}
		}

		if req.ToDate != nil && *req.ToDate != "" {
			toDate, err := time.Parse("2006-01-02", *req.ToDate)
			if err == nil {
				// Add 1 day to include the entire end date
				toDate = toDate.Add(24 * time.Hour)
				db = db.Where("created_at < ?", toDate)
			}
		}

		// Sorting
		sort := "created_at DESC" // Default sort
		if req.Sort != nil && *req.Sort != "" {
			switch *req.Sort {
			case "created_at_asc":
				sort = "created_at ASC"
			case "created_at_desc":
				sort = "created_at DESC"
			case "updated_at_desc":
				sort = "updated_at DESC"
			case "title_asc":
				sort = "title ASC"
			}
		}
		db = db.Order(sort)

		return db
	}

	// Preload relationships
	includes := []string{"Blog", "Blog.Author", "ContentChannels", "ContentChannels.Channel", "Task"}

	// Execute query
	contents, total, err := s.contentRepo.GetAll(ctx, filterFunc, includes, limit, page)
	if err != nil {
		zap.L().Error("Failed to list contents", zap.Error(err))
		return nil, 0, errors.New("failed to retrieve content list")
	}

	// Convert to response DTOs
	contentResponses := make([]*responses.ContentResponse, len(contents))
	for i := range contents {
		contentResponses[i] = s.mapToContentResponse(&contents[i])
	}

	zap.L().Info("Content list retrieved",
		zap.Int("count", len(contents)),
		zap.Int64("total", total),
		zap.Int("page", page),
		zap.Int("limit", limit))

	return contentResponses, total, nil
}
