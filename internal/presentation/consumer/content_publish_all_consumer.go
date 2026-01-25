package consumer

import (
	"context"
	"encoding/json"
	"fmt"

	"core-backend/internal/application"
	"core-backend/internal/application/dto/consumers"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/iservice"
	"core-backend/internal/domain/enum"
	"core-backend/pkg/utils"

	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"
)

// ContentPublishAllConsumer handles batch content publishing messages from RabbitMQ
type ContentPublishAllConsumer struct {
	contentPublishingService iservice.ContentPublishingService
	alertService             iservice.AlertManagerService
	notificationService      iservice.NotificationService
	validator                *validator.Validate
}

// NewContentPublishAllConsumer creates a new content publish all consumer
func NewContentPublishAllConsumer(appReg *application.ApplicationRegistry) *ContentPublishAllConsumer {
	return &ContentPublishAllConsumer{
		contentPublishingService: appReg.ContentPublishingService,
		alertService:             appReg.AlertManagerService,
		notificationService:      appReg.NotificationService,
		validator:                validator.New(),
	}
}

// Handle processes content publishing messages for all channels
func (c *ContentPublishAllConsumer) Handle(ctx context.Context, body []byte) error {
	zap.L().Info("Received content publish-all message",
		zap.Int("message_size", len(body)))

	// Parse message
	var msg consumers.PublishAllChannelsMessage
	if err := json.Unmarshal(body, &msg); err != nil {
		zap.L().Error("Failed to unmarshal content publish-all message",
			zap.Error(err),
			zap.ByteString("raw_message", body))
		return fmt.Errorf("failed to unmarshal message: %w", err)
	}

	// Validate message
	if err := c.validator.Struct(msg); err != nil {
		zap.L().Error("Invalid content publish-all message",
			zap.Error(err),
			zap.String("content_id", msg.ContentID.String()))
		c.alertUnexpectedError(ctx, msg, err)
		return fmt.Errorf("message validation failed: %w", err)
	}

	zap.L().Info("Processing content publish-all request",
		zap.String("content_id", msg.ContentID.String()),
		zap.String("user_id", msg.UserID.String()),
		zap.String("request_id", msg.RequestID))

	// Publish content to all channels
	result, err := c.contentPublishingService.PublishToAllChannels(
		ctx,
		msg.ContentID,
		msg.UserID,
	)

	if err != nil {
		zap.L().Error("Failed to publish content to all channels",
			zap.Error(err),
			zap.String("content_id", msg.ContentID.String()),
			zap.String("request_id", msg.RequestID))
		return fmt.Errorf("failed to publish content to all channels: %w", err)
	}

	zap.L().Info("Content published to all channels",
		zap.Int("total_channels", result.TotalChannels),
		zap.Int("success_count", result.SuccessCount),
		zap.Int("failure_count", result.FailureCount),
		zap.String("request_id", msg.RequestID))

	// Log any errors
	for _, channelError := range result.Errors {
		zap.L().Warn("Channel publish failed",
			zap.String("channel_id", channelError.ChannelID.String()),
			zap.String("channel_name", channelError.ChannelName),
			zap.String("error", channelError.Error))
		c.alertFailedPublishToChannel(ctx, msg, &channelError)
	}
	if len(result.Errors) > 0 {
		notiReq := requests.PublishInAppRequest{
			UserID: msg.UserID,
			Title:  "Content Publishing Failed",
			Body: fmt.Sprintf("Your content (ID: %s) failed to publish to %d out of %d channels.",
				msg.ContentID.String(), result.FailureCount, result.TotalChannels),
			Data: map[string]string{
				"content_id":      msg.ContentID.String(),
				"failed_channels": fmt.Sprintf("%d", result.FailureCount),
				"total_channels":  fmt.Sprintf("%d", result.TotalChannels),
				"failed_channel_ids": utils.ToString(utils.MapSlice(result.Errors, func(e responses.PublishChannelError) string {
					return e.ChannelID.String()
				})),
			},
		}
		if _, subErr := c.notificationService.CreateAndPublishInApp(ctx, &notiReq); subErr != nil {
			zap.L().Error("Failed to send in-app notification for content publishing failure",
				zap.Error(subErr),
				zap.String("content_id", msg.ContentID.String()),
				zap.String("request_id", msg.RequestID))
		}
	}

	// If all channels failed, return error to trigger retry
	if result.FailureCount == result.TotalChannels && result.TotalChannels > 0 {
		return fmt.Errorf("all channels failed to publish (%d/%d)", result.FailureCount, result.TotalChannels)
	}

	return nil
}

func (c *ContentPublishAllConsumer) alertUnexpectedError(ctx context.Context, msg consumers.PublishAllChannelsMessage, err error) {
	// Raise System Alert
	alertRequest := requests.RaiseAlertRequest{
		Type:           enum.AlertTypeError,
		Category:       enum.AlertCategoryContentPublishFailed,
		Severity:       enum.AlertSeverityHigh,
		Title:          "Content Publish Failed",
		Description:    fmt.Sprintf("Failed to publish content to all channels. Reason: %s", err.Error()),
		ReferenceID:    &msg.ContentID,
		ReferenceType:  utils.PtrOrNil(enum.ReferenceTypeContent),
		ActionURL:      nil,
		ExpiresInHours: nil,
		TargetRoles:    []enum.UserRole{enum.UserRoleContentStaff, enum.UserRoleAdmin},
	}
	if _, alertErr := c.alertService.RaiseAlert(ctx, &alertRequest); alertErr != nil {
		zap.L().Error("Failed to raise alert for content publish failure",
			zap.Error(alertErr),
			zap.String("content_id", msg.ContentID.String()),
			zap.String("request_id", msg.RequestID))
	}

	// Notify related users
	notiReq := requests.PublishInAppRequest{
		UserID: msg.UserID,
		Title:  "Content Publishing Failed",
		Body: fmt.Sprintf("Your content (ID: %s) failed to publish to all channels. Reason: %s",
			msg.ContentID.String(), err.Error()),
		Data: map[string]string{
			"content_id": msg.ContentID.String(),
			"error":      err.Error(),
		},
	}
	if _, subErr := c.notificationService.CreateAndPublishInApp(ctx, &notiReq); subErr != nil {
		zap.L().Error("Failed to send in-app notification for content publishing failure",
			zap.Error(subErr),
			zap.String("content_id", msg.ContentID.String()),
			zap.String("request_id", msg.RequestID))
		return // Continue to next message
	}
}

func (c *ContentPublishAllConsumer) alertFailedPublishToChannel(ctx context.Context, msg consumers.PublishAllChannelsMessage, channelError *responses.PublishChannelError) {
	// Raise System Alert
	alertRequest := requests.RaiseAlertRequest{
		Type:           enum.AlertTypeError,
		Category:       enum.AlertCategoryContentPublishFailed,
		Severity:       enum.AlertSeverityMedium,
		Title:          "Content Publish to Channel Failed",
		Description:    fmt.Sprintf("Failed to publish content to channel '%s'. Reason: %s", channelError.ChannelName, channelError.Error),
		ReferenceID:    &msg.ContentID,
		ReferenceType:  utils.PtrOrNil(enum.ReferenceTypeContent),
		ActionURL:      nil,
		ExpiresInHours: nil,
		TargetRoles:    []enum.UserRole{enum.UserRoleContentStaff, enum.UserRoleAdmin},
	}
	if _, alertErr := c.alertService.RaiseAlert(ctx, &alertRequest); alertErr != nil {
		zap.L().Error("Failed to raise alert for content publish failure",
			zap.Error(alertErr),
			zap.String("content_id", msg.ContentID.String()),
			zap.String("channel_id", channelError.ChannelID.String()),
			zap.String("channel_name", channelError.ChannelName),
			zap.String("request_id", msg.RequestID))
	}
}
