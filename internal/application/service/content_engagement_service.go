package service

import (
	"context"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/application/interfaces/iservice"
	"core-backend/internal/domain/constant"
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
	"errors"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type contentEngagementService struct {
	contentChannelRepo irepository.GenericRepository[model.ContentChannel]
	channelRepo        irepository.GenericRepository[model.Channel]
	contentCommentRepo irepository.GenericRepository[model.ContentComment]
}

// NewContentEngagementService creates a new content engagement service
func NewContentEngagementService(
	contentChannelRepo irepository.GenericRepository[model.ContentChannel],
	channelRepo irepository.GenericRepository[model.Channel],
	contentCommentRepo irepository.GenericRepository[model.ContentComment],
) iservice.ContentEngagementService {
	return &contentEngagementService{
		contentChannelRepo: contentChannelRepo,
		channelRepo:        channelRepo,
		contentCommentRepo: contentCommentRepo,
	}
}

// RecordEngagement handles all engagement actions for website content
// Actions: add_reaction, remove_reaction, share, add_comment, edit_comment, delete_comment, add_comment_reaction, remove_comment_reaction
func (s *contentEngagementService) RecordEngagement(ctx context.Context, req *requests.ContentEngagementRequest) (*responses.ContentEngagementResponse, error) {
	// Get the website content channel
	contentChannel, err := s.getWebsiteContentChannel(ctx, req.ContentID)
	if err != nil {
		return nil, err
	}

	// Route to appropriate handler based on action
	switch req.Action {
	// Content-level reactions
	case requests.EngagementActionAddReaction:
		return s.handleAddReaction(ctx, contentChannel, req, &req.UserID)
	case requests.EngagementActionRemoveReaction:
		return s.handleRemoveReaction(ctx, contentChannel, &req.UserID)
	case requests.EngagementActionShare:
		return s.handleShare(ctx, contentChannel, &req.UserID)

	// Comment actions
	case requests.EngagementActionAddComment:
		return s.handleAddComment(ctx, contentChannel, req, &req.UserID)
	case requests.EngagementActionEditComment:
		return s.handleEditComment(ctx, contentChannel, req, &req.UserID)
	case requests.EngagementActionDeleteComment:
		return s.handleDeleteComment(ctx, contentChannel, req, &req.UserID)

	// Comment reactions
	case requests.EngagementActionAddCommentReaction:
		return s.handleAddCommentReaction(ctx, contentChannel, req, &req.UserID)
	case requests.EngagementActionRemoveCommentReaction:
		return s.handleRemoveCommentReaction(ctx, contentChannel, req, &req.UserID)

	default:
		return nil, errors.New("unsupported engagement action")
	}
}

// GetEngagementSummary returns engagement summary for a content on WEBSITE channel
func (s *contentEngagementService) GetEngagementSummary(ctx context.Context, contentID uuid.UUID) (*responses.WebsiteEngagementSummary, error) {
	contentChannel, err := s.getWebsiteContentChannel(ctx, contentID)
	if err != nil {
		return nil, err
	}

	return s.buildEngagementSummary(ctx, contentChannel, nil)
}

// GetUserEngagementStatus returns whether the current user has liked/shared the content
func (s *contentEngagementService) GetUserEngagementStatus(ctx context.Context, contentID, userID uuid.UUID) (*responses.UserEngagementStatus, error) {
	contentChannel, err := s.getWebsiteContentChannel(ctx, contentID)
	if err != nil {
		return nil, err
	}

	metrics := s.getMetrics(contentChannel)
	engagement := metrics.GetWebsiteEngagement()

	status := &responses.UserEngagementStatus{
		HasLiked:  false,
		HasShared: false,
	}

	// Check if user has reacted
	for _, reaction := range engagement.Reactions {
		if reaction.UserID == userID {
			status.HasLiked = true
			reactionType := string(reaction.Type)
			status.LikeType = &reactionType
			break
		}
	}

	return status, nil
}

// region: ======== Content Reaction Handlers ========

func (s *contentEngagementService) handleAddReaction(ctx context.Context, cc *model.ContentChannel, req *requests.ContentEngagementRequest, userID *uuid.UUID) (*responses.ContentEngagementResponse, error) {
	if userID == nil {
		return nil, errors.New("user must be authenticated to react")
	}

	reactionType := enum.ReactionTypeLike
	if req.ReactionType != nil {
		reactionType = *req.ReactionType
		if !reactionType.IsValid() {
			return nil, errors.New("invalid reaction type")
		}
	}

	metrics := s.getMetrics(cc)
	engagement := metrics.GetWebsiteEngagement()

	// Check if user already reacted - update existing reaction
	updated := false
	for i, reaction := range engagement.Reactions {
		if reaction.UserID == *userID {
			// Update existing reaction
			oldType := string(engagement.Reactions[i].Type)
			engagement.Reactions[i].Type = reactionType
			engagement.Reactions[i].ReactedAt = time.Now()
			updated = true

			// Update summary: decrement old, increment new
			if engagement.ReactionSummary[oldType] > 0 {
				engagement.ReactionSummary[oldType]--
			}
			engagement.ReactionSummary[string(reactionType)]++
			break
		}
	}

	if !updated {
		// Add new reaction
		engagement.Reactions = append(engagement.Reactions, model.WebsiteReactionEntry{
			UserID:    *userID,
			Type:      reactionType,
			ReactedAt: time.Now(),
		})
		engagement.ReactionSummary[string(reactionType)]++
	}

	// Save metrics
	metrics.SetWebsiteEngagement(engagement)
	if err := s.saveMetrics(ctx, cc, metrics); err != nil {
		return nil, err
	}

	summary, _ := s.buildEngagementSummary(ctx, cc, userID)
	return &responses.ContentEngagementResponse{
		Success: true,
		Message: "Reaction recorded",
		Metrics: summary,
	}, nil
}

func (s *contentEngagementService) handleRemoveReaction(ctx context.Context, cc *model.ContentChannel, userID *uuid.UUID) (*responses.ContentEngagementResponse, error) {
	if userID == nil {
		return nil, errors.New("user must be authenticated to remove reaction")
	}

	metrics := s.getMetrics(cc)
	engagement := metrics.GetWebsiteEngagement()

	// Find and remove user's reaction
	removed := false
	for i, reaction := range engagement.Reactions {
		if reaction.UserID == *userID {
			reactionType := string(reaction.Type)
			// Remove from list
			engagement.Reactions = append(engagement.Reactions[:i], engagement.Reactions[i+1:]...)
			// Update summary
			if engagement.ReactionSummary[reactionType] > 0 {
				engagement.ReactionSummary[reactionType]--
			}
			removed = true
			break
		}
	}

	if !removed {
		return nil, errors.New("no reaction to remove")
	}

	// Save metrics
	metrics.SetWebsiteEngagement(engagement)
	if err := s.saveMetrics(ctx, cc, metrics); err != nil {
		return nil, err
	}

	summary, _ := s.buildEngagementSummary(ctx, cc, userID)
	return &responses.ContentEngagementResponse{
		Success: true,
		Message: "Reaction removed",
		Metrics: summary,
	}, nil
}

func (s *contentEngagementService) handleShare(ctx context.Context, cc *model.ContentChannel, userID *uuid.UUID) (*responses.ContentEngagementResponse, error) {
	metrics := s.getMetrics(cc)
	engagement := metrics.GetWebsiteEngagement()

	engagement.SharesCount++

	metrics.SetWebsiteEngagement(engagement)
	if err := s.saveMetrics(ctx, cc, metrics); err != nil {
		return nil, err
	}

	summary, _ := s.buildEngagementSummary(ctx, cc, userID)
	return &responses.ContentEngagementResponse{
		Success: true,
		Message: "Share recorded",
		Metrics: summary,
	}, nil
}

// endregion

// region: ======== Comment Handlers ========

func (s *contentEngagementService) handleAddComment(ctx context.Context, cc *model.ContentChannel, req *requests.ContentEngagementRequest, userID *uuid.UUID) (*responses.ContentEngagementResponse, error) {
	if userID == nil {
		return nil, errors.New("user must be authenticated to comment")
	}
	if req.CommentText == nil || *req.CommentText == "" {
		return nil, errors.New("comment text is required")
	}

	comment := &model.ContentComment{
		ID:               uuid.New(),
		ContentChannelID: cc.ID,
		Comment:          *req.CommentText,
		CreatedBy:        userID,
		Reactions:        []model.ContentReaction{},
	}

	if err := s.contentCommentRepo.Add(ctx, comment); err != nil {
		zap.L().Error("Failed to add comment", zap.Error(err))
		return nil, errors.New("failed to add comment")
	}

	// Update metrics - increment comment count in CurrentMapped
	metrics := s.getMetrics(cc)
	if metrics.CurrentMapped == nil {
		metrics.CurrentMapped = make(map[enum.KPIValueType]float64)
	}
	metrics.CurrentMapped[enum.KPIValueTypeComments]++
	metrics.CurrentMapped[enum.KPIValueTypeEngagement]++
	if err := s.saveMetrics(ctx, cc, metrics); err != nil {
		zap.L().Warn("Failed to update comment metrics", zap.Error(err))
	}

	summary, _ := s.buildEngagementSummary(ctx, cc, userID)
	return &responses.ContentEngagementResponse{
		Success:   true,
		Message:   "Comment added",
		Metrics:   summary,
		CommentID: &comment.ID,
	}, nil
}

func (s *contentEngagementService) handleEditComment(ctx context.Context, cc *model.ContentChannel, req *requests.ContentEngagementRequest, userID *uuid.UUID) (*responses.ContentEngagementResponse, error) {
	if userID == nil {
		return nil, errors.New("user must be authenticated to edit comment")
	}
	if req.CommentID == nil {
		return nil, errors.New("comment_id is required")
	}
	if req.CommentText == nil || *req.CommentText == "" {
		return nil, errors.New("comment text is required")
	}

	comment, err := s.contentCommentRepo.GetByID(ctx, *req.CommentID, nil)
	if err != nil {
		return nil, errors.New("comment not found")
	}

	// Check ownership
	if comment.CreatedBy != nil && *comment.CreatedBy != *userID {
		return nil, errors.New("unauthorized to edit this comment")
	}

	comment.Comment = *req.CommentText
	comment.UpdatedBy = userID

	if err := s.contentCommentRepo.Update(ctx, comment); err != nil {
		zap.L().Error("Failed to update comment", zap.Error(err))
		return nil, errors.New("failed to update comment")
	}

	summary, _ := s.buildEngagementSummary(ctx, cc, userID)
	return &responses.ContentEngagementResponse{
		Success:   true,
		Message:   "Comment updated",
		Metrics:   summary,
		CommentID: &comment.ID,
	}, nil
}

func (s *contentEngagementService) handleDeleteComment(ctx context.Context, cc *model.ContentChannel, req *requests.ContentEngagementRequest, userID *uuid.UUID) (*responses.ContentEngagementResponse, error) {
	if userID == nil {
		return nil, errors.New("user must be authenticated to delete comment")
	}
	if req.CommentID == nil {
		return nil, errors.New("comment_id is required")
	}

	comment, err := s.contentCommentRepo.GetByID(ctx, *req.CommentID, nil)
	if err != nil {
		return nil, errors.New("comment not found")
	}

	// Check ownership
	if comment.CreatedBy != nil && *comment.CreatedBy != *userID {
		return nil, errors.New("unauthorized to delete this comment")
	}

	if err := s.contentCommentRepo.Delete(ctx, comment); err != nil {
		zap.L().Error("Failed to delete comment", zap.Error(err))
		return nil, errors.New("failed to delete comment")
	}

	// Update metrics - decrement comment count
	metrics := s.getMetrics(cc)
	if metrics.CurrentMapped[enum.KPIValueTypeComments] > 0 {
		metrics.CurrentMapped[enum.KPIValueTypeComments]--
	}
	if metrics.CurrentMapped[enum.KPIValueTypeEngagement] > 0 {
		metrics.CurrentMapped[enum.KPIValueTypeEngagement]--
	}
	if err := s.saveMetrics(ctx, cc, metrics); err != nil {
		zap.L().Warn("Failed to update comment metrics", zap.Error(err))
	}

	summary, _ := s.buildEngagementSummary(ctx, cc, userID)
	return &responses.ContentEngagementResponse{
		Success: true,
		Message: "Comment deleted",
		Metrics: summary,
	}, nil
}

// endregion

// region: ======== Comment Reaction Handlers ========

func (s *contentEngagementService) handleAddCommentReaction(ctx context.Context, cc *model.ContentChannel, req *requests.ContentEngagementRequest, userID *uuid.UUID) (*responses.ContentEngagementResponse, error) {
	if userID == nil {
		return nil, errors.New("user must be authenticated to react to comment")
	}
	if req.CommentID == nil {
		return nil, errors.New("comment_id is required")
	}
	if req.ReactionType == nil {
		return nil, errors.New("reaction_type is required")
	}

	comment, err := s.contentCommentRepo.GetByID(ctx, *req.CommentID, nil)
	if err != nil {
		return nil, errors.New("comment not found")
	}

	reactionType := *req.ReactionType
	if !reactionType.IsValid() {
		return nil, errors.New("invalid reaction type")
	}

	// Check if user already reacted - update existing reaction
	updated := false
	for i, reaction := range comment.Reactions {
		if reaction.UserID != nil && *reaction.UserID == *userID {
			comment.Reactions[i].Type = reactionType
			comment.Reactions[i].ReactedAt = time.Now()
			updated = true
			break
		}
	}

	if !updated {
		// Add new reaction
		comment.Reactions = append(comment.Reactions, model.ContentReaction{
			ID:        uuid.New(),
			UserID:    userID,
			Type:      reactionType,
			ReactedAt: time.Now(),
		})
	}

	if err := s.contentCommentRepo.Update(ctx, comment); err != nil {
		zap.L().Error("Failed to update comment reaction", zap.Error(err))
		return nil, errors.New("failed to add reaction to comment")
	}

	summary, _ := s.buildEngagementSummary(ctx, cc, userID)
	return &responses.ContentEngagementResponse{
		Success:   true,
		Message:   "Reaction added to comment",
		Metrics:   summary,
		CommentID: &comment.ID,
	}, nil
}

func (s *contentEngagementService) handleRemoveCommentReaction(ctx context.Context, cc *model.ContentChannel, req *requests.ContentEngagementRequest, userID *uuid.UUID) (*responses.ContentEngagementResponse, error) {
	if userID == nil {
		return nil, errors.New("user must be authenticated to remove reaction")
	}
	if req.CommentID == nil {
		return nil, errors.New("comment_id is required")
	}

	comment, err := s.contentCommentRepo.GetByID(ctx, *req.CommentID, nil)
	if err != nil {
		return nil, errors.New("comment not found")
	}

	// Find and remove user's reaction
	removed := false
	for i, reaction := range comment.Reactions {
		if reaction.UserID != nil && *reaction.UserID == *userID {
			comment.Reactions = append(comment.Reactions[:i], comment.Reactions[i+1:]...)
			removed = true
			break
		}
	}

	if !removed {
		return nil, errors.New("no reaction to remove")
	}

	if err := s.contentCommentRepo.Update(ctx, comment); err != nil {
		zap.L().Error("Failed to remove comment reaction", zap.Error(err))
		return nil, errors.New("failed to remove reaction from comment")
	}

	summary, _ := s.buildEngagementSummary(ctx, cc, userID)
	return &responses.ContentEngagementResponse{
		Success:   true,
		Message:   "Reaction removed from comment",
		Metrics:   summary,
		CommentID: &comment.ID,
	}, nil
}

// endregion

// region: ======== Helper Methods ========

func (s *contentEngagementService) getWebsiteContentChannel(ctx context.Context, contentID uuid.UUID) (*model.ContentChannel, error) {
	// Find WEBSITE channel
	channels, _, err := s.channelRepo.GetAll(ctx,
		func(db *gorm.DB) *gorm.DB {
			return db.Where("code = ?", constant.ChannelCodeWebsite)
		}, nil, 1, 1)
	if err != nil || len(channels) == 0 {
		zap.L().Error("Failed to find WEBSITE channel", zap.Error(err))
		return nil, errors.New("website channel not found")
	}

	// Find content channel for this content and website
	contentChannels, _, err := s.contentChannelRepo.GetAll(ctx,
		func(db *gorm.DB) *gorm.DB {
			return db.Where("content_id = ?", contentID).
				Where("channel_id = ?", channels[0].ID)
		}, nil, 1, 1)
	if err != nil || len(contentChannels) == 0 {
		zap.L().Warn("Content not published to website",
			zap.String("content_id", contentID.String()))
		return nil, errors.New("content not published to website")
	}

	return &contentChannels[0], nil
}

func (s *contentEngagementService) getMetrics(cc *model.ContentChannel) *model.ContentChannelMetrics {
	if cc.Metrics == nil {
		return &model.ContentChannelMetrics{
			CurrentFetched: make(map[string]any),
			CurrentMapped:  make(map[enum.KPIValueType]float64),
		}
	}
	return cc.Metrics
}

func (s *contentEngagementService) saveMetrics(ctx context.Context, cc *model.ContentChannel, metrics *model.ContentChannelMetrics) error {
	cc.Metrics = metrics
	return s.contentChannelRepo.Update(ctx, cc)
}

func (s *contentEngagementService) buildEngagementSummary(ctx context.Context, cc *model.ContentChannel, userID *uuid.UUID) (*responses.WebsiteEngagementSummary, error) {
	metrics := s.getMetrics(cc)
	engagement := metrics.GetWebsiteEngagement()

	// Calculate totals from reaction summary
	var totalReactions int64
	reactionsByType := make(map[string]int64)
	for reactionType, count := range engagement.ReactionSummary {
		reactionsByType[reactionType] = count
		totalReactions += count
	}

	// Get comment count from database (more accurate than metrics)
	_, totalComments, err := s.contentCommentRepo.GetAll(ctx,
		func(db *gorm.DB) *gorm.DB {
			return db.Where("content_channel_id = ?", cc.ID)
		}, nil, 1, 1)
	if err != nil {
		zap.L().Warn("Failed to get comment count", zap.Error(err))
	}

	// Check user's reaction
	var userReaction *string
	if userID != nil {
		for _, reaction := range engagement.Reactions {
			if reaction.UserID == *userID {
				reactionType := string(reaction.Type)
				userReaction = &reactionType
				break
			}
		}
	}

	return &responses.WebsiteEngagementSummary{
		TotalReactions:  totalReactions,
		ReactionsByType: reactionsByType,
		TotalComments:   totalComments,
		TotalShares:     engagement.SharesCount,
		UserReaction:    userReaction,
	}, nil
}

// endregion
