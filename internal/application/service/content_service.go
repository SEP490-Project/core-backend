package service

import (
	"bytes"
	"context"
	"core-backend/config"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/application/interfaces/iservice"
	"core-backend/internal/application/service/helper"
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
	gormrepository "core-backend/internal/infrastructure/gorm_repository"
	"core-backend/pkg/tiptap"
	"core-backend/pkg/utils"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type ContentService struct {
	config               *config.AppConfig
	contentRepo          irepository.ContentRepository
	blogRepo             irepository.GenericRepository[model.Blog]
	contentChannelRepo   irepository.GenericRepository[model.ContentChannel]
	taskRepo             irepository.TaskRepository
	affiliateLinkRepo    irepository.AffiliateLinkRepository
	contractRepo         irepository.ContractRepository
	uow                  irepository.UnitOfWork
	affiliateLinkService iservice.AffiliateLinkService
	channelService       iservice.ChannelService
	contractService      iservice.ContractService
}

func NewContentService(
	config *config.AppConfig,
	databaseReg *gormrepository.DatabaseRegistry,
	uow irepository.UnitOfWork,
	affiliateLinkService iservice.AffiliateLinkService,
	channelService iservice.ChannelService,
	contractService iservice.ContractService,
) iservice.ContentService {
	return &ContentService{
		config:               config,
		contentRepo:          databaseReg.ContentRepository,
		blogRepo:             databaseReg.BlogRepository,
		contentChannelRepo:   databaseReg.ContentChannelRepository,
		taskRepo:             databaseReg.TaskRepository,
		affiliateLinkRepo:    databaseReg.AffiliateLinkRepository,
		contractRepo:         databaseReg.ContractRepository,
		uow:                  uow,
		affiliateLinkService: affiliateLinkService,
		channelService:       channelService,
		contractService:      contractService,
	}
}

// Create creates new content with DRAFT status
func (s *ContentService) Create(ctx context.Context, uow irepository.UnitOfWork, req *requests.CreateContentRequest) (*responses.ContentResponse, error) {
	zap.L().Info("Creating content", zap.String("title", req.Title), zap.String("type", req.Type))

	taskRepo := uow.Tasks()
	contentRepo := uow.Contents()
	blogRepo := uow.Blogs()
	tagRepo := uow.Tags()
	contentChannelRepo := uow.ContentChannels()

	validationFuncs := make([]func(ctx context.Context) error, 0)
	if req.TaskID != nil {
		validationFuncs = append(validationFuncs, func(ctx context.Context) error {
			if exists, err := taskRepo.ExistsByID(ctx, *req.TaskID); err != nil {
				zap.L().Error("Failed to check task existence", zap.Error(err))
				return err
			} else if !exists {
				return errors.New("task not found")
			}
			return nil
		})
	}
	if req.BlogFields != nil && req.BlogFields.AuthorID != uuid.Nil {
		validationFuncs = append(validationFuncs, func(ctx context.Context) error {
			if exists, err := uow.Users().ExistsByID(ctx, req.BlogFields.AuthorID); err != nil {
				zap.L().Error("Failed to check user existence", zap.Error(err))
				return err
			} else if !exists {
				return errors.New("user not found")
			}
			return nil
		})
	}
	utils.RunParallel(ctx, 3, validationFuncs...)

	// Start transaction
	rawBody, err := json.Marshal(req.Body)
	if err != nil {
		zap.L().Error("Failed to marshal body", zap.Error(err))
		return nil, errors.New("failed to marshal body")
	}

	// Create content entity
	content := &model.Content{
		ID:              uuid.New(),
		TaskID:          req.TaskID,
		Title:           req.Title,
		Description:     req.Description,
		Body:            rawBody,
		Type:            enum.ContentType(req.Type),
		Status:          enum.ContentStatusDraft,
		AIGeneratedText: req.AIGeneratedText,
	}
	if req.BlogFields != nil && req.BlogFields.AuthorID != uuid.Nil {
		content.CreatedBy = &req.BlogFields.AuthorID
	} else {
		content.CreatedBy = &req.UserID
	}

	if req.Description == nil || *req.Description == "" {
		switch req.Type {
		case enum.ContentTypeVideo.String():
			var bodyMap map[string]any
			if err = json.Unmarshal(rawBody, &bodyMap); err == nil {
				if description, ok := bodyMap["description"].(string); ok {
					content.Description = &description
				}
			}
		case enum.ContentTypePost.String():
			var parsedTipTap *tiptap.TiptapParseResult
			parsedTipTap, err = tiptap.ParseTiptapJSON(rawBody)
			minText := min(100, len(parsedTipTap.PlainText))
			if err == nil {
				content.Description = utils.PtrOrNil(parsedTipTap.PlainText[:minText])
			} else {
				zap.L().Warn("Failed to parse TipTap JSON for description", zap.Error(err))
			}
			if parsedTipTap.HasImages {
				content.ThumbnailURL = utils.PtrOrNil(parsedTipTap.ImageURLs[0])
			}
		}
	}

	var createdTags []model.Tag
	if req.BlogFields != nil {
		creatingTags := utils.MapSlice(req.BlogFields.Tags, func(tag string) model.Tag {
			return model.Tag{Name: tag, CreatedByID: &req.BlogFields.AuthorID}
		})
		createdTags, err = tagRepo.CreateIfNotExists(ctx, creatingTags)
		if err != nil {
			zap.L().Error("Failed to create or retrieve tags", zap.Error(err))
			return nil, errors.New("failed to create or retrieve tags")
		}
		content.Tags = utils.MapSlice(createdTags, func(tag model.Tag) string {
			return tag.Name
		})
	}

	if err = contentRepo.Add(ctx, content); err != nil {
		zap.L().Error("Failed to create content", zap.Error(err))
		return nil, errors.New("failed to create content")
	}

	// Create blog if type is POST
	if content.Type == enum.ContentTypePost && req.BlogFields != nil {
		blog := &model.Blog{
			ContentID: content.ID,
			AuthorID:  req.BlogFields.AuthorID,
			Tags:      createdTags,
			Excerpt:   req.BlogFields.Excerpt,
			ReadTime:  req.BlogFields.ReadTime,
		}

		if err = blogRepo.Add(ctx, blog); err != nil {
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

		if err = contentChannelRepo.Add(ctx, contentChannel); err != nil {
			zap.L().Error("Failed to create content channel", zap.Error(err))
			return nil, errors.New("failed to create content channel")
		}
	}

	if err = uow.Commit(); err != nil {
		zap.L().Error("Failed to commit transaction", zap.Error(err))
		return nil, errors.New("failed to commit transaction")
	}

	// Automatically create affiliate links for content channels if contract has tracking link
	// and no affiliate link info provided in the request
	// This is a best-effort operation - failures are logged but don't fail content creation
	if req.TrackingLink == nil {
		if err := s.createAffiliateLinkIfNeeded(ctx, content, req.Channels); err != nil {
			zap.L().Warn("Failed to create affiliate links for content",
				zap.String("content_id", content.ID.String()),
				zap.Error(err))
			// Don't fail content creation if affiliate link creation fails
		}
	}

	if req.TaskID != nil {
		go func() {
			if err := s.updateContractScopeOfWork(ctx, *req.TaskID); err != nil {
				zap.L().Warn("Failed to update contract scope of work after content creation",
					zap.String("content_id", content.ID.String()),
					zap.Error(err))
			}
		}()
	}

	zap.L().Info("Content created successfully", zap.String("content_id", content.ID.String()))
	return s.GetByID(ctx, content.ID)
}

// GetByID retrieves content by ID with relationships
func (s *ContentService) GetByID(ctx context.Context, id uuid.UUID) (*responses.ContentResponse, error) {
	content, err := s.contentRepo.GetByID(ctx, id, []string{"Blog", "Blog.Author", "ContentChannels.AffiliateLink", "ContentChannels.Channel", "Blog.Tags"})

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("content not found")
		}
		return nil, err
	}

	return responses.ContentResponse{}.ToResponse(content, s.config.Server.BaseURL), nil
}

// GetByIDPublic retrieves publicly accessible content by ID
// Filters at DB level to only return POSTED content (more efficient than post-fetch check)
func (s *ContentService) GetByIDPublic(ctx context.Context, id uuid.UUID) (*responses.ContentResponse, error) {
	content, err := s.contentRepo.GetByCondition(ctx, func(db *gorm.DB) *gorm.DB {
		return db.Where("id = ? AND status = ?", id, enum.ContentStatusPosted)
	}, []string{"Blog", "Blog.Author", "ContentChannels.AffiliateLink", "ContentChannels.Channel", "Blog.Tags"})

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("content not found")
		}
		return nil, err
	}

	if content == nil {
		return nil, errors.New("content not found")
	}

	return responses.ContentResponse{}.ToResponse(content, s.config.Server.BaseURL), nil
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
	if req.Description != nil {
		content.Description = req.Description
	}
	if req.Body != nil {
		if rawBody, err := json.Marshal(req.Body); err == nil {
			content.Body = rawBody
		} else {
			zap.L().Error("Failed to marshal body", zap.Error(err))
			return nil, errors.New("failed to marshal body field")
		}
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

	// return s.contentRepo.Delete(ctx, content)

	return helper.WithTransaction(ctx, s.uow, func(ctx context.Context, uow irepository.UnitOfWork) error {
		if err := uow.Contents().Delete(ctx, content); err != nil {
			zap.L().Error("Failed to delete content",
				zap.String("content_id", id.String()),
				zap.Error(err))
			return err
		}

		return nil
	})
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
	if content.Title == "" || content.Body == nil {
		return errors.New("title and body are required fields")
	}

	// Validate affiliate link if needed
	if err = s.validateAffiliateLink(ctx, content); err != nil {
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

// SetRejectionFeedback stores rejection feedback for a content item
// This is called AFTER state transition through FSM
func (s *ContentService) SetRejectionFeedback(ctx context.Context, uow irepository.UnitOfWork, contentID uuid.UUID, feedback string) error {
	content, err := uow.Contents().GetByID(ctx, contentID, nil)
	if err != nil {
		return errors.New("content not found")
	}

	content.RejectionFeedback = &feedback
	if err := uow.Contents().Update(ctx, content); err != nil {
		zap.L().Error("Failed to store rejection feedback",
			zap.String("content_id", contentID.String()),
			zap.Error(err))
		return errors.New("failed to store rejection feedback")
	}

	return nil
}

// SetPublishDate stores publish date for a content item
// This is called AFTER state transition through FSM
func (s *ContentService) SetPublishDate(ctx context.Context, uow irepository.UnitOfWork, contentID uuid.UUID, publishDate *string) error {
	content, err := uow.Contents().GetByID(ctx, contentID, nil)
	if err != nil {
		return errors.New("content not found")
	}

	// Set publish date
	if publishDate != nil && *publishDate != "" {
		// parsedDate, err := time.Parse(time.RFC3339, *publishDate)
		parsedDate := utils.ParseLocalTimeWithFallback(*publishDate, utils.TimeFormat)
		if parsedDate == nil {
			content.PublishDate = parsedDate
		}
	} else {
		now := time.Now()
		content.PublishDate = &now
	}

	if err := uow.Contents().Update(ctx, content); err != nil {
		zap.L().Error("Failed to update publish date",
			zap.String("content_id", contentID.String()),
			zap.Error(err))
		return errors.New("failed to update publish date")
	}

	return nil
}

// ValidateForSubmission validates content is ready for submission
func (s *ContentService) ValidateForSubmission(ctx context.Context, contentID uuid.UUID) error {
	content, err := s.contentRepo.GetByID(ctx, contentID, nil)
	if err != nil {
		return errors.New("content not found")
	}

	// Validate required fields
	if content.Title == "" || content.Body == nil {
		return errors.New("title and body are required fields")
	}

	// Validate affiliate link if needed
	return s.validateAffiliateLink(ctx, content)
}

// DetermineWorkflowRoute determines target status based on selected channels
func (s *ContentService) DetermineWorkflowRoute(ctx context.Context, contentID uuid.UUID) (enum.ContentStatus, error) {
	return s.determineWorkflowRoute(ctx, contentID)
}

// List retrieves paginated content with filters, search, and sorting
func (s *ContentService) List(ctx context.Context, req *requests.ContentFilterRequest) ([]*responses.ContentListResponse, int64, error) {
	zap.L().Info("Listing contents with filters", zap.Any("filters", req))

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

		if req.BrandID != nil {
			db = db.Where(`
        EXISTS (
            SELECT 1
            FROM tasks t
            JOIN milestones m ON m.id = t.milestone_id
            JOIN campaigns c ON c.id = m.campaign_id
            JOIN contracts ct ON ct.id = c.contract_id
            WHERE t.id = contents.task_id
              AND ct.brand_id = ?
        )
    `, *req.BrandID)
		}

		if req.UserID != nil {
			db = db.Where(`
        EXISTS (
            SELECT 1
            FROM tasks t
            JOIN milestones m ON m.id = t.milestone_id
            JOIN campaigns c ON c.id = m.campaign_id
            JOIN contracts ct ON ct.id = c.contract_id
            JOIN brands b ON b.id = ct.brand_id
            WHERE t.id = contents.task_id
              AND b.user_id = ?
        )
    `, *req.UserID)
		}

		// Filter by task_id
		if req.TaskID != nil {
			db = db.Where("task_id = ?", *req.TaskID)
		}

		if req.AssignedTo != nil {
			db = db.Joins("JOIN tasks ON tasks.id = contents.task_id").
				Where("tasks.assigned_to = ?", req.AssignedTo).
				Distinct()
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
			db = db.Where("title ILIKE ? OR body::text ILIKE ?", searchPattern, searchPattern)
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
		db = db.Order(helper.ConvertToSortString(req.PaginationRequest))

		return db
	}

	// Preload relationships
	includes := []string{"Blog", "Blog.Author", "Blog.Tags", "ContentChannels", "ContentChannels.Channel", "Task"}

	// Execute query
	contents, total, err := s.contentRepo.GetAll(ctx, filterFunc, includes, req.Limit, req.Page)
	if err != nil {
		zap.L().Error("Failed to list contents", zap.Error(err))
		return nil, 0, errors.New("failed to retrieve content list")
	}

	zap.L().Info("Content list retrieved",
		zap.Int("count", len(contents)),
		zap.Int64("total", total))

	return responses.ContentListResponse{}.ToResponseList(contents, s.config.Server.BaseURL), total, nil
}

// region: ======== Helper methods ========

// determineWorkflowRoute determines the target status based on selected channels
func (s *ContentService) determineWorkflowRoute(ctx context.Context, contentID uuid.UUID) (enum.ContentStatus, error) {
	// Get content channels with channel preload
	contentChannels, _, err := s.contentChannelRepo.GetAll(ctx,
		func(db *gorm.DB) *gorm.DB {
			return db.Joins("Content").Joins("Channel").Where("content_channels.content_id = ?", contentID)
		},
		nil, 0, 0,
	)
	if err != nil {
		return "", errors.New("failed to retrieve content channels")
	}

	// Check if any channel is FACEBOOK or TIKTOK (external brand channels)
	awaitBrandAllowedChannels := []string{"FACEBOOK", "TIKTOK"}
	for _, cc := range contentChannels {
		if cc.Channel != nil {
			if (utils.ContainsSlice(awaitBrandAllowedChannels, strings.ToUpper(cc.Channel.Code)) ||
				utils.ContainsSlice(awaitBrandAllowedChannels, strings.ToUpper(cc.Channel.Name))) &&
				cc.Content.TaskID != nil {
				return enum.ContentStatusAwaitBrand, nil
			}
		}
	}

	// Default to internal staff review
	return enum.ContentStatusAwaitStaff, nil
}

// validateAffiliateLink validates affiliate link requirement for AFFILIATE contracts
func (s *ContentService) validateAffiliateLink(ctx context.Context, content *model.Content) error {
	// If no task association, skip validation
	if content.TaskID == nil {
		return nil
	}

	// Get task with contract information
	_, err := s.taskRepo.GetByID(ctx, *content.TaskID, nil)
	if err != nil {
		zap.L().Warn("Failed to retrieve task for affiliate link validation",
			zap.String("task_id", content.TaskID.String()),
			zap.Error(err))
		return nil // Don't fail submission if task not found
	}

	// TODO: Implement actual contract type check when contract info is available

	// Parse task description to get contract info
	// if task.Description != nil {
	// 	var descMap map[string]any
	// 	if err := json.Unmarshal(task.Description, &descMap); err == nil {
	// 		if contractType, ok := descMap["contract_type"].(string); ok {
	// 			if contractType == "AFFILIATE" {
	// 				// For AFFILIATE contracts, affiliate_link is required
	// 				if content.AffiliateLink == nil || *content.AffiliateLink == "" {
	// 					return errors.New("affiliate link is required for AFFILIATE contract content")
	// 				}
	// 			}
	// 		}
	// 	}
	// }

	return nil
}

// createAffiliateLinkIfNeeded creates affiliate links for content channels if contract has tracking link
func (s *ContentService) createAffiliateLinkIfNeeded(ctx context.Context, content *model.Content, channelIDs []uuid.UUID) error {
	// Skip if no task association
	if content.TaskID == nil || len(channelIDs) == 0 {
		return nil
	}

	trackingLink, contractID, err := s.taskRepo.GetContractTrackingLinkByTaskID(ctx, *content.TaskID)
	if err != nil {
		zap.L().Warn("Failed to retrieve affiliate link for content",
			zap.String("task_id", content.TaskID.String()),
			zap.Error(err))
		return nil // Don't fail content creation if tracking link retrieval fails
	} else if trackingLink == "" || contractID == uuid.Nil {
		zap.L().Warn("No tracking link found for contract associated with content task",
			zap.String("task_id", content.TaskID.String()))
		return nil
	}

	// Create affiliate links for each channel
	for _, channelID := range channelIDs {
		affiliateReq := &requests.CreateAffiliateLinkRequest{
			TrackingURL: trackingLink,
			ContractID:  &contractID,
			ContentID:   &content.ID,
			ChannelID:   &channelID,
		}

		// Use CreateOrGet to ensure idempotency (no duplicates)
		affiliateLink, err := s.affiliateLinkService.CreateOrGet(ctx, affiliateReq)
		if err != nil {
			zap.L().Error("Failed to create affiliate link for content channel",
				zap.String("content_id", content.ID.String()),
				zap.String("channel_id", channelID.String()),
				zap.String("tracking_url", trackingLink),
				zap.Error(err))
			// Log error but don't fail content creation
			continue
		}
		zap.L().Info("Affiliate link created for content channel",
			zap.String("affiliate_link_id", affiliateLink.ID.String()),
			zap.String("hash", affiliateLink.Hash),
			zap.String("content_id", content.ID.String()),
			zap.String("channel_id", channelID.String()))
	}

	// NOTE: Affiliate link injection is now handled at publish/render time via ContentChannel.GetRenderedBody()
	// This method replaces the tracking URL with the channel-specific affiliate URL dynamically.
	// This supports multi-channel content where each channel has its own unique affiliate link.
	// The original body is preserved in content.Body and should contain the tracking URL.

	// If the tracking URL is not in the body, append it now (one time, shared across all channels)
	if !bytes.Contains(content.Body, []byte(trackingLink)) {
		builder, err := tiptap.FromJSON(content.Body)
		if err == nil {
			builder.AddLinkParagraph("Check it out ", "right now", trackingLink)
			newBody, err := builder.Build()
			if err != nil {
				zap.L().Error("Failed to build new body with tracking link",
					zap.String("content_id", content.ID.String()),
					zap.Error(err))
			} else {
				content.Body = newBody
				if err := s.contentRepo.Update(ctx, content); err != nil {
					zap.L().Error("Failed to update content body with tracking link",
						zap.String("content_id", content.ID.String()),
						zap.Error(err))
				}
			}
		}
	}

	return nil
}

// bodyContainsURL checks if the tiptap body contains the given URL in any link
// func (s *ContentService) bodyContainsURL(body []byte, url string) bool {
// 	return bytes.Contains(body, []byte(url))
// }

func (s *ContentService) appendDefaultHomePageURLToContent(body []byte) ([]byte, error) {
	builder, err := tiptap.FromJSON(body)
	if err != nil {
		return nil, err
	}
	builder.AddLinkParagraph("Visit our ", "website", s.config.Server.BaseFrontendURL)
	builder.AddLinkParagraph("Visit our ", "Facebook page", s.config.AdminConfig.FacebookHomepageURL)
	builder.AddLinkParagraph("Visit our ", "TikTok profile", s.config.AdminConfig.TikTokHomepageURL)

	return builder.Build()
}

func (s *ContentService) updateContractScopeOfWork(ctx context.Context, taskID uuid.UUID) error {
	// Find contract ID associate with task id
	contractID, err := s.contractRepo.GetContractIDByTaskID(ctx, taskID)
	if err != nil {
		zap.L().Error("Failed to retrieve contract ID by task ID", zap.Error(err))
		return err
	} else if contractID == uuid.Nil {
		zap.L().Warn("No contract found for task ID", zap.String("task_id", taskID.String()))
		return nil
	}

	return s.contractService.UpdateContractScopeOfWorkWithReferencinnTaskIDs(ctx, contractID)
}

// endregion
