package service

import (
	"context"
	"core-backend/config"
	"core-backend/internal/application/dto/dtos"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/iproxies"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/application/interfaces/irepository_third_party"
	"core-backend/internal/application/interfaces/iservice"
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
	"core-backend/internal/infrastructure"
	gormrepository "core-backend/internal/infrastructure/gorm_repository"
	"core-backend/pkg/logging"
	"core-backend/pkg/utils"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type contentPublishingService struct {
	contentRepo          irepository.GenericRepository[model.Content]
	contentChannelRepo   irepository.GenericRepository[model.ContentChannel]
	channelRepo          irepository.GenericRepository[model.Channel]
	facebookProxy        iproxies.FacebookProxy
	tiktokProxy          iproxies.TikTokProxy
	channelService       iservice.ChannelService
	stateTransferService iservice.StateTransferService
	fileService          iservice.FileService
	notificationService  iservice.NotificationService
	s3Storage            irepository_third_party.S3Storage
	s3StreamingStorage   irepository_third_party.S3StreamingStorage
	uow                  irepository.UnitOfWork
	config               *config.AppConfig
}

// PublishToChannel implements iservice.ContentPublishingService.
func (s *contentPublishingService) PublishToChannel(ctx context.Context, contentID uuid.UUID, channelID uuid.UUID, userID uuid.UUID) (*responses.PublishContentResponse, error) {
	zap.L().Info("PublishToChannel called",
		zap.String("content_id", contentID.String()),
		zap.String("channel_id", channelID.String()),
		zap.String("user_id", userID.String()))

	// 1. Load content with preloads
	content, err := s.contentRepo.GetByID(ctx, contentID, []string{"ContentChannels", "Task"})
	if err != nil {
		zap.L().Error("Failed to load content", zap.Error(err))
		return nil, errors.New("content not found")
	}

	// 2. Validate content status (must be APPROVED)
	if content.Status != enum.ContentStatusApproved {
		zap.L().Warn("Content not approved for publishing",
			zap.String("content_id", contentID.String()),
			zap.String("status", string(content.Status)))
		return nil, errors.New("content must be APPROVED before publishing")
	}

	// 3. Load channel with OAuth credentials
	channel, err := s.channelRepo.GetByID(ctx, channelID, nil)
	if err != nil {
		zap.L().Error("Failed to load channel", zap.Error(err))
		return nil, errors.New("channel not found")
	}

	// 4. Get decrypted access token
	var accessToken string
	if channel.Code != "WEBSITE" {
		accessToken, err = s.channelService.GetDecryptedToken(ctx, channel.Name)
		if err != nil {
			zap.L().Error("Failed to decrypt access token",
				zap.String("channel_name", channel.Name),
				zap.Error(err))
			return nil, fmt.Errorf("failed to decrypt access token: %w", err)
		}
	}

	// 5. Find or create ContentChannel record
	contentChannel, err := s.findOrCreateContentChannel(ctx, contentID, channelID)
	if err != nil {
		return nil, err
	}

	// 6. Route to appropriate platform
	var externalPostID string
	var postURL string
	var externalPostType *enum.ExternalPostType

	switch channel.Code {
	case "FACEBOOK":
		externalPostID, postURL, externalPostType, err = s.publishToFacebook(ctx, content, channel, contentChannel, accessToken)
	case "TIKTOK":
		externalPostType = utils.PtrOrNil(enum.ExternalPostTypeVideo)
		externalPostID, postURL, err = s.publishToTikTok(ctx, content, channel, contentChannel, accessToken)
	case "WEBSITE":
		externalPostID, postURL, err = s.publishToWebiste(ctx, content, channel)
	default:
		return nil, fmt.Errorf("unsupported channel code: %s", channel.Code)
	}

	if err != nil {
		// Update ContentChannel with error
		uow := s.uow.Begin(ctx)
		defer func() {
			if r := recover(); r != nil {
				uow.Rollback()
				panic(r)
			}
		}()

		errorMsg := err.Error()
		contentChannel.AutoPostStatus = enum.AutoPostStatusFailed
		contentChannel.LastError = &errorMsg

		if updateErr := uow.ContentChannels().Update(ctx, contentChannel); updateErr != nil {
			uow.Rollback()
			zap.L().Error("Failed to update content channel with error", zap.Error(updateErr))
		} else {
			uow.Commit()
		}

		return nil, err
	}

	// 7-8. Update ContentChannel and Content status
	uow := s.uow.Begin(ctx)
	defer func() {
		if r := recover(); r != nil {
			uow.Rollback()
			panic(r)
		}
	}()

	// Update ContentChannel with success
	now := time.Now()
	contentChannel.ExternalPostID = &externalPostID
	contentChannel.ExternalPostURL = &postURL
	contentChannel.ExternalPostType = externalPostType
	contentChannel.PublishedAt = &now
	contentChannel.AutoPostStatus = enum.AutoPostStatusPosted
	contentChannel.LastError = nil

	if err = uow.ContentChannels().Update(ctx, contentChannel); err != nil {
		uow.Rollback()
		zap.L().Error("Failed to update content channel after successful publish", zap.Error(err))
		return nil, fmt.Errorf("failed to update content channel: %w", err)
	}

	// Check if all channels posted → update content status to POSTED
	allPosted, err := s.checkAllChannelsPosted(ctx, uow, contentID)
	if err != nil {
		uow.Rollback()
		zap.L().Error("Failed to check all channels posted", zap.Error(err))
		return nil, fmt.Errorf("failed to check channel status: %w", err)
	}

	if allPosted {
		if err := s.stateTransferService.MoveContentToState(ctx, uow, content.ID, enum.ContentStatusPosted, userID); err != nil {
			uow.Rollback()
			zap.L().Error("Failed to update content status to POSTED", zap.Error(err))
			return nil, fmt.Errorf("failed to update content status: %w", err)
		}
	}

	// Commit transaction
	if err := uow.Commit(); err != nil {
		zap.L().Error("Failed to commit transaction", zap.Error(err))
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	zap.L().Info("Content published successfully",
		zap.String("content_id", contentID.String()),
		zap.String("channel_code", channel.Code),
		zap.String("external_post_id", externalPostID))

	return &responses.PublishContentResponse{
		ContentChannelID: contentChannel.ID,
		ExternalPostID:   externalPostID,
		PostURL:          postURL,
		PublishedAt:      now,
		Channel:          channel.Name,
	}, nil
}

// PublishToAllChannels implements iservice.ContentPublishingService.
func (s *contentPublishingService) PublishToAllChannels(ctx context.Context, contentID uuid.UUID, userID uuid.UUID) (*responses.PublishAllChannelsResponse, error) {
	zap.L().Info("PublishToAllChannels called",
		zap.String("content_id", contentID.String()),
		zap.String("user_id", userID.String()))

	// Load content with content channels
	content, err := s.contentRepo.GetByID(ctx, contentID, []string{"ContentChannels"})
	if err != nil {
		return nil, errors.New("content not found")
	}

	if len(content.ContentChannels) == 0 {
		return nil, errors.New("no channels assigned to this content")
	}

	response := &responses.PublishAllChannelsResponse{
		TotalChannels: len(content.ContentChannels),
		Results:       []responses.PublishContentResponse{},
		Errors:        []responses.PublishChannelError{},
	}

	// Publish to each channel
	for _, cc := range content.ContentChannels {
		result, err := s.PublishToChannel(ctx, contentID, cc.ChannelID, userID)
		if err != nil {
			response.FailureCount++
			// Load channel name for error response
			channel, _ := s.channelRepo.GetByID(ctx, cc.ChannelID, nil)
			channelName := "Unknown"
			if channel != nil {
				channelName = channel.Name
			}
			response.Errors = append(response.Errors, responses.PublishChannelError{
				ChannelID:   cc.ChannelID,
				ChannelName: channelName,
				Error:       err.Error(),
			})
		} else {
			response.SuccessCount++
			response.Results = append(response.Results, *result)
		}
	}

	return response, nil
}

// GetPublishingStatus implements iservice.ContentPublishingService.
func (s *contentPublishingService) GetPublishingStatus(ctx context.Context, contentChannelID uuid.UUID) (*responses.PublishingStatusResponse, error) {
	zap.L().Info("GetPublishingStatus called",
		zap.String("content_channel_id", contentChannelID.String()))

	// Load content channel
	contentChannel, err := s.contentChannelRepo.GetByID(ctx, contentChannelID, nil)
	if err != nil {
		return nil, errors.New("content channel not found")
	}

	// Load channel for name
	channel, err := s.channelRepo.GetByID(ctx, contentChannel.ChannelID, nil)
	if err != nil {
		return nil, errors.New("channel not found")
	}

	// Parse metrics if available
	var metrics map[string]any
	if contentChannel.Metrics != nil {
		if err := json.Unmarshal(contentChannel.Metrics, &metrics); err != nil {
			zap.L().Warn("Failed to parse metrics", zap.Error(err))
		}
	}

	// Extract post URL from metrics if not set directly
	var postURL *string
	if metrics != nil {
		if url, ok := metrics["post_url"].(string); ok {
			postURL = &url
		}
	}

	return &responses.PublishingStatusResponse{
		ContentChannelID: contentChannel.ID,
		ContentID:        contentChannel.ContentID,
		ChannelID:        contentChannel.ChannelID,
		ChannelName:      channel.Name,
		Status:           contentChannel.AutoPostStatus,
		ExternalPostID:   contentChannel.ExternalPostID,
		ExternalPostURL:  contentChannel.ExternalPostURL,
		PostURL:          postURL,
		PublishedAt:      contentChannel.PublishedAt,
		LastError:        contentChannel.LastError,
		Metrics:          metrics,
		CreatedAt:        contentChannel.CreatedAt,
		UpdatedAt:        contentChannel.UpdatedAt,
	}, nil
}

// RetryPublish implements iservice.ContentPublishingService.
func (s *contentPublishingService) RetryPublish(ctx context.Context, contentChannelID uuid.UUID, userID uuid.UUID) error {
	zap.L().Info("RetryPublish called",
		zap.String("content_channel_id", contentChannelID.String()))

	// Load content channel
	contentChannel, err := s.contentChannelRepo.GetByID(ctx, contentChannelID, nil)
	if err != nil {
		return errors.New("content channel not found")
	}

	// Reset status to PENDING
	contentChannel.AutoPostStatus = enum.AutoPostStatusPending
	contentChannel.LastError = nil

	if err = s.contentChannelRepo.Update(ctx, contentChannel); err != nil {
		return fmt.Errorf("failed to reset content channel status: %w", err)
	}

	// Attempt publish again
	_, err = s.PublishToChannel(ctx, contentChannel.ContentID, contentChannel.ChannelID, userID)
	return err
}

// region: 1. ============ Helper methods ===========

func (s *contentPublishingService) findOrCreateContentChannel(ctx context.Context, contentID uuid.UUID, channelID uuid.UUID) (*model.ContentChannel, error) {
	// Try to find existing content channel
	contentChannels, _, err := s.contentChannelRepo.GetAll(ctx, func(db *gorm.DB) *gorm.DB {
		return db.Where("content_id = ? AND channel_id = ?", contentID, channelID)
	}, nil, 1, 1)

	if err != nil {
		return nil, fmt.Errorf("failed to query content channels: %w", err)
	}

	if len(contentChannels) > 0 {
		return &contentChannels[0], nil
	}

	// Create new content channel
	newContentChannel := &model.ContentChannel{
		ID:             uuid.New(),
		ContentID:      contentID,
		ChannelID:      channelID,
		AutoPostStatus: enum.AutoPostStatusPending,
	}

	if err := s.contentChannelRepo.Add(ctx, newContentChannel); err != nil {
		return nil, fmt.Errorf("failed to create content channel: %w", err)
	}

	return newContentChannel, nil
}

// checkAllChannelsPosted checks if all channels are posted using a UnitOfWork transaction
func (s *contentPublishingService) checkAllChannelsPosted(ctx context.Context, uow irepository.UnitOfWork, contentID uuid.UUID) (bool, error) {
	// Get all content channels for this content within the transaction
	contentChannels, _, err := uow.ContentChannels().GetAll(ctx, func(db *gorm.DB) *gorm.DB {
		return db.Where("content_id = ?", contentID)
	}, nil, 0, 0)

	if err != nil {
		return false, err
	}

	if len(contentChannels) == 0 {
		return false, nil
	}

	// Check if all are posted
	for _, cc := range contentChannels {
		if cc.AutoPostStatus != enum.AutoPostStatusPosted &&
			cc.AutoPostStatus != enum.AutoPostStatusSkipped {
			return false, nil
		}
	}

	return true, nil
}

// region: 2. =========== Platform-specific publishing ===========

// region: 3. =========== Facebook Publishing ===========

func (s *contentPublishingService) publishToFacebook(ctx context.Context, content *model.Content, channel *model.Channel, contentChannel *model.ContentChannel, accessToken string) (string, string, *enum.ExternalPostType, error) {
	zap.L().Info("contentPublishingService - publishToFacebook called",
		zap.String("content_id", content.ID.String()),
		zap.String("channel_code", channel.Code),
		zap.String("channel_name", channel.Name))

	if channel.ExternalID == nil {
		return "", "", nil, errors.New("facebook page ID not set for channel")
	}

	pageID := *channel.ExternalID

	// Parse Tiptap content body to extract text and images
	parseResult, err := utils.ParseTiptapJSON(content.Body)
	if err != nil {
		return "", "", nil, fmt.Errorf("failed to parse content body: %w", err)
	}

	switch content.Type {
	case enum.ContentTypePost:
		imageLen := len(parseResult.ImageURLs)
		if imageLen == 1 {
			return s.publishSinglePhotoPostToFacebook(ctx, contentChannel.ID, accessToken, pageID, parseResult)
		} else if imageLen > 1 {
			return s.publishMultiPhotoPostToFacebook(ctx, contentChannel.ID, accessToken, pageID, parseResult)
		} else {
			return s.publishTextPostToFacebook(ctx, contentChannel.ID, accessToken, pageID, parseResult.PlainText)
		}

	case enum.ContentTypeVideo:
		return s.publishVideoPostToFacebook(ctx, contentChannel.ID, accessToken, pageID, content, parseResult)
	default:
		return "", "", nil, fmt.Errorf("unsupported content type for Facebook: %s", content.Type)
	}
}

// region: 4. ======== Facebook Publishing Content of Post Types ========

// publishTextPostToFacebook creates a simple text post on Facebook
func (s *contentPublishingService) publishTextPostToFacebook(ctx context.Context, contentChannelID uuid.UUID, accessToken, pageID, message string) (string, string, *enum.ExternalPostType, error) {
	zap.L().Info("contentPublishingService - publishTextPostToFacebook called",
		zap.String("page_id", pageID))

	resp, err := s.facebookProxy.CreateTextPost(ctx, pageID, message, true, accessToken)
	if err != nil {
		return "", "", nil, fmt.Errorf("facebook text post failed: %w", err)
	}
	postURL := fmt.Sprintf("https://www.facebook.com/%s", resp.ID)

	// Save external post ID asynchronously
	s.saveExternalPostIDAsync(ctx, contentChannelID, resp.ID, postURL, enum.ExternalPostTypeText)

	return resp.ID, postURL, utils.PtrOrNil(enum.ExternalPostTypeText), nil
}

// publishSinglePhotoPostToFacebook creates a photo post with a single image on Facebook
func (s *contentPublishingService) publishSinglePhotoPostToFacebook(ctx context.Context, contentChannelID uuid.UUID, accessToken, pageID string, parseResult *utils.TiptapParseResult) (string, string, *enum.ExternalPostType, error) {
	// Photo post with caption using first image
	publishingRequest := &dtos.FacebookPhotoPostPublishRequest{
		PageID:                 pageID,
		Caption:                parseResult.PlainText,
		Published:              true,
		UnpublishedContentType: dtos.UnpublishedContentTypeScheduled,
		URL:                    parseResult.ImageURLs[0],
	}
	resp, err := s.facebookProxy.CreateSinglePhotoPost(ctx, accessToken, publishingRequest)
	if err != nil {
		return "", "", nil, fmt.Errorf("facebook photo post failed: %w", err)
	}
	postURL := fmt.Sprintf("https://www.facebook.com/%s", resp.ID)

	// Save external post ID asynchronously
	s.saveExternalPostIDAsync(ctx, contentChannelID, resp.ID, postURL, enum.ExternalPostTypeSingleImage)
	return resp.ID, postURL, utils.PtrOrNil(enum.ExternalPostTypeSingleImage), nil
}

// publishMultiPhotoPostToFacebook creates a multi-photo post on Facebook
func (s *contentPublishingService) publishMultiPhotoPostToFacebook(
	ctx context.Context, contentChannelID uuid.UUID, accessToken, pageID string, parseResult *utils.TiptapParseResult,
) (string, string, *enum.ExternalPostType, error) {
	// Step 0: Save external post ID asynchronously (without external ID or URL yet)
	s.saveExternalPostIDAsync(ctx, contentChannelID, "", "", enum.ExternalPostTypeSingleImage)

	// Step 1: Upload each image to get media_fbid
	attachedMedia := make([]string, len(parseResult.ImageURLs))
	var funcs []func(ctx context.Context) error

	for idx, imageURL := range parseResult.ImageURLs {
		uploadReq := &dtos.FacebookImageUploadRequest{
			PageID:    pageID,
			URL:       imageURL,
			Published: false,
			Temporary: false,
		}

		i := idx // copy index for closure safety

		funcs = append(funcs, func(ctx context.Context) error {
			mediaID, err := s.facebookProxy.UploadImage(ctx, accessToken, uploadReq)
			if err != nil {
				zap.L().Error("Failed to upload image",
					zap.Int("index", i),
					zap.String("url", imageURL),
					zap.Error(err))
				return fmt.Errorf("failed to upload image: %w", err)
			}
			attachedMedia[i] = mediaID // SAFE (each goroutine writes to its own cell)
			return nil
		})
	}
	if err := utils.RunParallelWithRetry(ctx, 3, utils.RetryOptions{
		MaxAttempts:       3,
		BaseBackoff:       1 * time.Second,
		BackoffMultiplier: 1.5,
		AttemptTimeout:    20 * time.Second,
	}, funcs...); err != nil {
		zap.L().Error("Failed to upload images in parallel", zap.Error(err))
		return "", "", nil, fmt.Errorf("failed to upload images: %w", err)
	}

	// Step 2: Create multi-photo post
	publishingRequest := &dtos.FacebookPhotoPostPublishRequest{
		PageID:                 pageID,
		Caption:                parseResult.PlainText,
		Published:              true,
		UnpublishedContentType: dtos.UnpublishedContentTypeScheduled,
		AttachedMedia:          attachedMedia,
	}
	resp, err := s.facebookProxy.CreateMultiPhotoPost(ctx, accessToken, publishingRequest)
	if err != nil {
		zap.L().Error("Failed to create multi-photo post", zap.Error(err))
		return "", "", nil, fmt.Errorf("failed to create multi-photo post: %w", err)
	}

	postURL := fmt.Sprintf("https://www.facebook.com/%s", resp.ID)
	return resp.ID, postURL, utils.PtrOrNil(enum.ExternalPostTypeMultiImage), nil
}

// endregion 4.

// region: 4. ======== Facebook Publishing Content of Video Types ========

func (s *contentPublishingService) publishVideoPostToFacebook(
	ctx context.Context, contentChannelID uuid.UUID, accessToken, pageID string, content *model.Content, parseResult *utils.TiptapParseResult,
) (string, string, *enum.ExternalPostType, error) {
	// Step 0: Save external post ID asynchronously (without external ID or URL yet)
	s.saveExternalPostIDAsync(ctx, contentChannelID, "", "", enum.ExternalPostTypeSingleImage)

	// Step 1: Extract video info from content body
	videoInfo, err := s.extractVideoInfoFromContentBody(ctx, content)
	if err != nil {
		zap.L().Error("Failed to extract video info from content body", zap.Error(err))
		return "", "", nil, fmt.Errorf("failed to extract video info from content body: %w", err)
	}
	if parseResult != nil && parseResult.PlainText != "" {
		videoInfo.Description = parseResult.PlainText
	}

	zap.L().Info("Initiating Facebook video upload",
		zap.String("content_id", content.ID.String()),
		zap.String("video_url", videoInfo.VideoURL),
		zap.String("title", videoInfo.Title))

	// Step 2: Upload video to Facebook via URL
	videoPublishRequest := &dtos.FacebookVideoPostPublishRequest{
		PageID:                 pageID,
		Title:                  videoInfo.Title,
		Description:            videoInfo.Description,
		FileURL:                videoInfo.VideoURL,
		Published:              true,
		ScheduledPublishTime:   0,
		UnpublishedContentType: nil,
		SocialActions:          true,
		Secret:                 false,
	}
	videoID, err := s.facebookProxy.CreateVideoPostFromURL(ctx, accessToken, videoPublishRequest)
	if err != nil {
		zap.L().Error("Failed to create Facebook video post from URL",
			zap.Error(err))
		return "", "", nil, fmt.Errorf("failed to create Facebook video post from URL: %w", err)
	}

	return videoID, fmt.Sprintf("https://www.facebook.com/%s/videos/%s", pageID, videoID), utils.PtrOrNil(enum.ExternalPostTypeVideo), nil
}

/* Deprecated Video publishing logic via resumable upload
func (s *contentPublishingService) uploadVideoToFacebookResumable(ctx context.Context, accessToken string, content *model.Content, videoS3Key string) (string, string, error) {
 // Step 1: Download video from S3
	videoReader, fileSize, err := s.s3StreamingStorage.Get(ctx, videoS3Key)
	if err != nil {
		zap.L().Error("Failed to download video from S3",
			zap.String("s3_key", videoS3Key),
			zap.Error(err))
		return "", "", fmt.Errorf("failed to download video from S3: %w", err)
	}
	defer videoReader.Close()

	zap.L().Info("Video downloaded from S3",
		zap.String("content_id", content.ID.String()),
		zap.Int64("file_size", fileSize))

	// Step 2: Initialize video upload with Facebook using Resumable Upload API
	// Generate filename from content ID
	fileName := fmt.Sprintf("video_%s.mp4", content.ID.String())

	uploadSessionID, err := s.facebookProxy.InitVideoUpload(ctx, fileName, fileSize, accessToken)
	if err != nil {
		return "", "", fmt.Errorf("facebook video init failed: %w", err)
	}

	zap.L().Info("Facebook video upload session initialized",
		zap.String("content_id", content.ID.String()),
		zap.String("upload_session_id", uploadSessionID))

	// Step 3: Upload video in chunks with resume capability
	// const chunkSize = 50 * 1024 * 1024 // 50MB chunks (recommended by Facebook)
	// maxRetries := 3
	chunkSize := s.config.AdminConfig.FacebookVideoUploadChunkSizeInMB * 1024 * 1024
	maxRetries := s.config.AdminConfig.FacebookVideoUploadMaxRetries
	var fileHandle *string
	var bytesUploaded int64 = 0

	// Check if there's any existing progress (for resume)
	existingProgress, err := s.facebookProxy.GetUploadStatus(ctx, uploadSessionID, accessToken)
	if err == nil && existingProgress > 0 {
		bytesUploaded = existingProgress
		zap.L().Info("Resuming upload from existing progress",
			zap.String("content_id", content.ID.String()),
			zap.Int64("bytes_uploaded", bytesUploaded))

		// Skip bytes already uploaded
		_, err = io.CopyN(io.Discard, videoReader, bytesUploaded)
		if err != nil {
			return "", "", fmt.Errorf("failed to skip uploaded bytes: %w", err)
		}
	}

	// Upload chunks
	for bytesUploaded < fileSize {
		// Calculate chunk size for this iteration
		remainingBytes := fileSize - bytesUploaded
		currentChunkSize := min(remainingBytes, int64(chunkSize))

		// Read chunk from S3 reader
		chunk := make([]byte, currentChunkSize)
		var n int
		n, err = io.ReadFull(videoReader, chunk)
		if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
			return "", "", fmt.Errorf("failed to read video chunk: %w", err)
		}

		// Retry logic for chunk upload
		var uploadErr error
		chunkToUpload := chunk[:n]     // Save chunk for retries
		currentOffset := bytesUploaded // Save offset for this chunk

		for attempt := range maxRetries {
			zap.L().Info("Uploading video chunk",
				zap.String("content_id", content.ID.String()),
				zap.Int64("offset", currentOffset),
				zap.Int("chunk_size", n),
				zap.Int("attempt", attempt+1))

			// Upload chunk and potentially get file handle on last chunk
			fileHandle, uploadErr = s.facebookProxy.UploadVideoChunk(ctx, uploadSessionID, &chunkToUpload, currentOffset, accessToken)
			if uploadErr == nil {
				break // Success
			}

			// Check if we can resume from server state
			if attempt < maxRetries-1 {
				zap.L().Warn("Chunk upload failed, checking server state",
					zap.String("content_id", content.ID.String()),
					zap.Int64("offset", currentOffset),
					zap.Error(uploadErr))

				// Query server for current progress
				serverProgress, statusErr := s.facebookProxy.GetUploadStatus(ctx, uploadSessionID, accessToken)
				if statusErr == nil {
					if serverProgress > currentOffset {
						// Server already has this chunk (and possibly more)
						zap.L().Info("Server already has this chunk",
							zap.String("content_id", content.ID.String()),
							zap.Int64("server_offset", serverProgress),
							zap.Int64("our_offset", currentOffset))

						// Update our position and skip to next chunk
						uploadErr = nil // Clear error
						break
					} else if serverProgress == currentOffset {
						// Server is at the position we're trying to upload, retry
						zap.L().Info("Server waiting at expected offset, retrying",
							zap.String("content_id", content.ID.String()),
							zap.Int64("offset", serverProgress))
					}
				}

				// Wait before retry with exponential backoff
				backoff := time.Duration(attempt+1) * 2 * time.Second
				time.Sleep(backoff)
			}
		}

		if uploadErr != nil {
			return "", "", fmt.Errorf("failed to upload video chunk after %d retries: %w", maxRetries, uploadErr)
		}

		bytesUploaded += int64(n)
		zap.L().Info("Chunk uploaded successfully",
			zap.String("content_id", content.ID.String()),
			zap.Int64("total_uploaded", bytesUploaded),
			zap.Float64("progress_percent", float64(bytesUploaded)/float64(fileSize)*100))
	}

	if fileHandle == nil {
		return "", "", errors.New("upload completed but no file handle received")
	}

	zap.L().Info("Video upload completed, received file handle",
		zap.String("content_id", content.ID.String()),
		zap.Int("file_handle", len(*fileHandle)))

	// Step 4: Publish video to page using file handle
	publishResp, err := s.facebookProxy.PublishVideo(ctx, fileHandle, pageID, title, description, accessToken)
	if err != nil {
		return "", "", fmt.Errorf("facebook video publish failed: %w", err)
	}

	postURL := fmt.Sprintf("https://www.facebook.com/%s", publishResp.ID)

	zap.L().Info("Facebook video published successfully",
		zap.String("content_id", content.ID.String()),
		zap.String("post_id", publishResp.ID),
		zap.String("post_url", postURL))

	return publishResp.ID, postURL, nil
}
*/

// endregion 4.

// endregion 3.

// region: 3. =========== TikTok Publishing ===========

// TODO: Refactor publishToTikTok according to docs
func (s *contentPublishingService) publishToTikTok(ctx context.Context, content *model.Content, _ *model.Channel, contentChannel *model.ContentChannel, accessToken string) (string, string, error) {
	// TikTok only supports video
	if content.Type != enum.ContentTypeVideo {
		return "", "", errors.New("TikTok only supports video content")
	}

	// Step 0: Save external post ID asynchronously (without external ID or URL yet)
	s.saveExternalPostIDAsync(ctx, contentChannel.ID, "", "", enum.ExternalPostTypeVideo)

	// 1. Get creator info (required - validates token and gets allowed privacy levels)
	creatorInfo, err := s.tiktokProxy.GetCreatorInfo(ctx, accessToken)
	if err != nil {
		return "", "", fmt.Errorf("failed to get TikTok creator info: %w", err)
	}

	zap.L().Info("TikTok creator info retrieved",
		zap.String("username", creatorInfo.Data.CreatorUsername),
		zap.Any("privacy_options", creatorInfo.Data.PrivacyLevelOptions))

	// 2. Parse content body for video metadata
	videoInfo, err := s.extractVideoInfoFromContentBody(ctx, content)
	if err != nil {
		zap.L().Error("Failed to extract video info from content body", zap.Error(err))
		return "", "", fmt.Errorf("failed to extract video info from content body: %w", err)
	}

	// 3. Init video post
	// Note: This logic use the PULL_FROM_URL method to let TikTok fetch the video from a public URL
	// This will only required one step.
	initReq := &dtos.TikTokVideoInitRequest{
		PostInfo: dtos.TikTokPostInfo{
			PrivacyLevel:       dtos.TikTokPrivacyLevelSelfOnly,
			Title:              videoInfo.Title,
			DisableDuet:        creatorInfo.Data.DuetDisabled,
			DisableStitch:      creatorInfo.Data.StitchDisabled,
			DisableComment:     creatorInfo.Data.CommentDisabled,
			BrandContentToggle: false,
			BrandOrganicToggle: false,
			IsAIGC:             false,
		},
		SourceInfo: dtos.TikTokSourceInfo{
			Source:   dtos.TikTokSourcePullFromURL,
			VideoURL: &videoInfo.VideoURL, // Must be publicly accessible
		},
	}
	if content.AIGeneratedText != nil {
		initReq.PostInfo.IsAIGC = true
	}
	// NOTE: Currently, the logic for specifying brandContent info only based on if the content is from a task
	// Later if there is a need, implement the logic to check if the content is based on contract or not
	if content.Task != nil {
		initReq.PostInfo.BrandContentToggle = true
		initReq.PostInfo.BrandOrganicToggle = false
	} else {
		initReq.PostInfo.BrandContentToggle = false
		initReq.PostInfo.BrandOrganicToggle = true
	}

	// initReq.FileInfo
	if videoInfo.S3Key != nil {
		var fileInfoResp *responses.FileDetailResponse
		fileInfoResp, err = s.fileService.GetFileByS3Key(ctx, *videoInfo.S3Key)
		if err == nil {
			initReq.FileInfoMetadata = fileInfoResp.Metadata
		}
	}
	// 4. Validate init Content request before sending to TikTok
	if errList := s.tiktokProxy.ValidateContentRequest(ctx, accessToken, initReq, &creatorInfo.Data); len(errList) > 0 {
		zap.L().Error("TikTok content request validation failed",
			zap.Strings("errors", utils.MapSlice(errList, func(e error) string { return e.Error() })))
		return "", "", fmt.Errorf("tiktok content request validation failed: %v", errList)
	}

	// 5. Send init request to TikTok
	initResp, err := s.tiktokProxy.InitVideoPost(ctx, accessToken, initReq)
	if err != nil {
		return "", "", fmt.Errorf("failed to init TikTok video post: %w", err)
	}

	zap.L().Info("TikTok video post initiated",
		zap.String("publish_id", initResp.Data.PublishID))

	// TikTok will send webhook to notify the status of the video post
	// For now, persist publishID to check status
	return initResp.Data.PublishID, "", nil
}

// endregion 3.

// publishToWebiste publish the content to the website channel,
// This only return the content id, and the created URL on the website
// The publish date and status is handle outside
// Example URL: https://bshowsell.site/blog/48ae1866-ad46-4b5a-89ee-40183cf29eca
func (s *contentPublishingService) publishToWebiste(_ context.Context, content *model.Content, _ *model.Channel) (string, string, error) {
	zap.L().Info("ContentPublishingService - publishToWebiste called")
	return content.ID.String(), fmt.Sprintf("%s/blog/%s", s.config.Server.BaseFrontendURL, content.ID.String()), nil
}

// endregion 2.

// saveExternalPostIDAsync saves the external post ID and URL asynchronously with retries
func (s *contentPublishingService) saveExternalPostIDAsync(
	ctx context.Context, contentChannelID uuid.UUID, externalPostID, externalPostURL string, externalPostType enum.ExternalPostType,
) {
	zap.L().Info("ContentPublishingService - saveExternalPostIDAsync called",
		zap.String("content_channel_id", contentChannelID.String()),
		zap.String("external_post_id", externalPostID),
		zap.String("external_post_url", externalPostURL))

	requestID := logging.GetRequestID()
	saveFunc := func(ctx context.Context) error {
		logging.SetRequestID(requestID)
		if err := s.contentChannelRepo.UpdateByCondition(ctx, func(db *gorm.DB) *gorm.DB {
			return db.Where("id = ?", contentChannelID)
		}, map[string]any{
			"external_post_id":   externalPostID,
			"external_post_url":  externalPostURL,
			"auto_post_status":   enum.AutoPostStatusInProgress,
			"external_post_type": externalPostType,
		}); err != nil {
			zap.L().Error("Failed to save external post ID asynchronously", zap.Error(err))
			return fmt.Errorf("failed to save external post ID: %w", err)
		}
		return nil
	}
	if err := utils.RunWithRetry(ctx, utils.DefaultRetryOptions, saveFunc); err != nil {
		zap.L().Error("Failed to save external post ID asynchronously", zap.Error(err))
		return
	}

	zap.L().Info("Successfully saved external post ID asynchronously",
		// zap.String("content_id", contentID.String()),
		// zap.String("channel_id", channelID.String()),
		zap.String("content_channel_id", contentChannelID.String()),
		zap.String("external_post_id", externalPostID),
		zap.String("external_post_url", externalPostURL))
}

// extractVideoInfoFromContentBody extracts video metadata from the content body JSON
func (s *contentPublishingService) extractVideoInfoFromContentBody(
	_ context.Context, content *model.Content,
) (*struct {
	Title       string
	Description string
	VideoURL    string
	S3Key       *string
}, error) {
	var videoBody map[string]any
	if err := json.Unmarshal(content.Body, &videoBody); err != nil {
		return nil, fmt.Errorf("failed to parse video content body: %w", err)
	}

	// Extract video S3 key
	videoS3URLStr, ok := videoBody["video_url"].(string)
	if !ok || videoS3URLStr == "" {
		return nil, errors.New("content body must contain 'video_s3_key' field for VIDEO type")
	}
	videoURL, err := url.Parse(videoS3URLStr)
	if err != nil {
		zap.L().Error("Failed to parse video URL",
			zap.String("video_url", videoS3URLStr),
			zap.Error(err))
		return nil, fmt.Errorf("failed to parse video URL: %w", err)
	}
	videoS3Key := strings.Trim(videoURL.Path, "/")
	var encodedURL string
	if encodedURL, err = utils.EncodeIndividualPathSegments(videoURL.String()); err != nil {
		zap.L().Error("Failed to encode video S3 URL",
			zap.Error(err))
		return nil, fmt.Errorf("failed to encode video S3 URL: %w", err)
	}

	// Use title from content model
	title := content.Title
	if title == "" {
		title, ok = videoBody["title"].(string)
		if !ok || title == "" {
			title = "Untitled Video"
		}
	}

	description, ok := videoBody["description"].(string)
	if !ok {
		description = ""
	}
	result := &struct {
		Title       string
		Description string
		VideoURL    string
		S3Key       *string
	}{
		Title:       title,
		Description: description,
		VideoURL:    encodedURL,
		S3Key:       &videoS3Key,
	}
	return result, nil
}

// endregion 1.

// NewContentPublishingService creates a new instance of ContentPublishingService
func NewContentPublishingService(
	infraReg *infrastructure.InfrastructureRegistry,
	databaseReg *gormrepository.DatabaseRegistry,
	channelService iservice.ChannelService,
	stateTransferService iservice.StateTransferService,
	fileService iservice.FileService,
	notificationService iservice.NotificationService,
	config *config.AppConfig,
) iservice.ContentPublishingService {
	return &contentPublishingService{
		contentRepo:          databaseReg.ContentRepository,
		contentChannelRepo:   databaseReg.ContentChannelRepository,
		channelRepo:          databaseReg.ChannelRepository,
		facebookProxy:        infraReg.ProxiesRegistry.FacebookProxy,
		tiktokProxy:          infraReg.ProxiesRegistry.TikTokProxy,
		channelService:       channelService,
		stateTransferService: stateTransferService,
		fileService:          fileService,
		notificationService:  notificationService,
		s3Storage:            infraReg.ThirdPartyStorage.S3Storage,
		s3StreamingStorage:   infraReg.ThirdPartyStorage.S3StreamStorage,
		uow:                  infraReg.UnitOfWork,
		config:               config,
	}
}
