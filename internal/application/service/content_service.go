package service

import (
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
	contentRepo          irepository.GenericRepository[model.Content]
	blogRepo             irepository.GenericRepository[model.Blog]
	contentChannelRepo   irepository.GenericRepository[model.ContentChannel]
	taskRepo             irepository.TaskRepository
	affiliateLinkRepo    irepository.AffiliateLinkRepository
	uow                  irepository.UnitOfWork
	affiliateLinkService iservice.AffiliateLinkService
	channelService       iservice.ChannelService
}

func NewContentService(
	config *config.AppConfig,
	databaseReg *gormrepository.DatabaseRegistry,
	uow irepository.UnitOfWork,
	affiliateLinkService iservice.AffiliateLinkService,
	channelService iservice.ChannelService,
) iservice.ContentService {
	return &ContentService{
		config:               config,
		contentRepo:          databaseReg.ContentRepository,
		blogRepo:             databaseReg.BlogRepository,
		contentChannelRepo:   databaseReg.ContentChannelRepository,
		taskRepo:             databaseReg.TaskRepository,
		affiliateLinkRepo:    databaseReg.AffiliateLinkRepository,
		uow:                  uow,
		affiliateLinkService: affiliateLinkService,
		channelService:       channelService,
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

	var affiliateLink *model.AffiliateLink

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
	if req.AffiliateLink != nil || req.AffiliateLinkID != nil {
		validationFuncs = append(validationFuncs, func(ctx context.Context) error {
			affiliateLinkQuery := func(db *gorm.DB) *gorm.DB {
				if req.AffiliateLinkID != nil {
					db = db.Where("id = ?", *req.AffiliateLinkID)
				}
				if req.AffiliateLink != nil {
					db = db.Where("hash = ?", strings.Split(*req.AffiliateLink, "/r/")[1])
				}
				return db
			}
			var err error
			if affiliateLink, err = s.affiliateLinkRepo.GetByCondition(ctx, affiliateLinkQuery, nil); err != nil {
				zap.L().Error("Failed to retrieve affiliate link for content channel",
					zap.Error(err))
				return err
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

	if err = contentRepo.Add(ctx, content); err != nil {
		zap.L().Error("Failed to create content", zap.Error(err))
		return nil, errors.New("failed to create content")
	}

	// Create blog if type is POST
	if content.Type == enum.ContentTypePost && req.BlogFields != nil {
		creatingTags := utils.MapSlice(req.BlogFields.Tags, func(tag string) model.Tag {
			return model.Tag{Name: tag, CreatedByID: &req.BlogFields.AuthorID}
		})
		var createdTags []model.Tag
		createdTags, err = tagRepo.CreateIfNotExists(ctx, creatingTags)
		if err != nil {
			zap.L().Error("Failed to create or retrieve tags", zap.Error(err))
			return nil, errors.New("failed to create or retrieve tags")
		}

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

		// NOTE: Similarly, because the current frontend implementations only passed one channel per content creation flow.
		// Thus, it is possible to just update the created affiliate link with the current content ID and channel ID.
		// However, in the future, if multiple channels are supported in one content creation flow, it is necessary to revisit
		// how to handle the affiliate link in the content body.
		if affiliateLink != nil {
			if err = uow.AffiliateLinks().UpdateByCondition(ctx, func(db *gorm.DB) *gorm.DB {
				return db.Where("id = ?", affiliateLink.ID)
			}, map[string]any{"content_id": content.ID, "channel_id": channelID}); err != nil {
				zap.L().Error("Failed to associate affiliate link with content and channel", zap.Error(err))
			}
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
	if req.AffiliateLinkID == nil && req.AffiliateLink == nil {
		if err := s.createAffiliateLinkIfNeeded(ctx, content, req.Channels); err != nil {
			zap.L().Warn("Failed to create affiliate links for content",
				zap.String("content_id", content.ID.String()),
				zap.Error(err))
			// Don't fail content creation if affiliate link creation fails
		}
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

	return s.contentRepo.Delete(ctx, content)
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
	var affiliateLinkURL string
	for _, channelID := range channelIDs {
		affiliateReq := &requests.CreateAffiliateLinkRequest{
			TrackingURL: trackingLink,
			ContractID:  &contractID,
			ContentID:   &content.ID,
			ChannelID:   &channelID,
		}

		// Use CreateOrGet to ensure idempotency (no duplicates)
		var affiliateLink *responses.AffiliateLinkResponse
		affiliateLink, err = s.affiliateLinkService.CreateOrGet(ctx, affiliateReq)
		if err != nil {
			zap.L().Error("Failed to create affiliate link for content channel",
				zap.String("content_id", content.ID.String()),
				zap.String("channel_id", channelID.String()),
				zap.String("tracking_url", trackingLink),
				zap.Error(err))
			// Log error but don't fail content creation
			continue
		}
		affiliateLinkURL = affiliateLink.ShortURL
		zap.L().Info("Affiliate link created for content channel",
			zap.String("affiliate_link_id", affiliateLink.ID.String()),
			zap.String("hash", affiliateLink.Hash),
			zap.String("content_id", content.ID.String()),
			zap.String("channel_id", channelID.String()))
	}

	// NOTE: Currently, content have a 1-N relationship with Channels. However, the current Frontend implementation only
	// pass one channel per content creation. In that case, it is possible to append the affiliate link at the end of the body
	// in the content model.
	// Further on, if multiple channels are supported on the frontend, it is necessary to revisit how to handlC:q
	// affiliate links in the content body.
	builder, err := tiptap.FromJSON(content.Body)
	if err == nil {
		builder.AddLinkParagraph("Check it out ", "right now", affiliateLinkURL)
		var newBody []byte
		newBody, err = builder.Build()
		if err != nil {
			zap.L().Error("Failed to build new body with affiliate link",
				zap.String("content_id", content.ID.String()),
				zap.Error(err))
		}

		content.Body = newBody
		if err := s.contentRepo.Update(ctx, content); err != nil {
			zap.L().Error("Failed to update content body after affiliate link creation",
				zap.String("content_id", content.ID.String()),
				zap.Error(err))
			// Don't fail content creation if affiliate link creation fails
		}
	}

	return nil
}

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

// endregion
