package consumer

import (
	"context"
	"core-backend/internal/application/dto/consumers"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/application/interfaces/iservice"
	"core-backend/internal/application/interfaces/iservice_third_party"
	"core-backend/internal/application/service/helper"
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
	"core-backend/internal/infrastructure"
	gormrepository "core-backend/internal/infrastructure/gorm_repository"
	"encoding/json"
	"errors"
	"time"

	"firebase.google.com/go/v4/messaging"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// NotificationPushConsumer handles push notification messages from RabbitMQ
type NotificationPushConsumer struct {
	fcmService             iservice_third_party.FCMService
	expoPushService        iservice_third_party.ExpoPushService
	deviceTokenRepository  irepository.DeviceTokenRepository
	notificationRepository irepository.NotificationRepository
	userService            iservice.UserService
	healthMonitor          iservice_third_party.HealthMonitor
	unitOfWork             irepository.UnitOfWork
}

// NewNotificationPushConsumer creates a new push notification consumer
func NewNotificationPushConsumer(
	infraReg *infrastructure.InfrastructureRegistry,
	dbRegistry *gormrepository.DatabaseRegistry,
	userService iservice.UserService,
) *NotificationPushConsumer {
	return &NotificationPushConsumer{
		fcmService:             infraReg.FCMService,
		expoPushService:        infraReg.ExpoPushService,
		deviceTokenRepository:  dbRegistry.DeviceTokenRepository,
		notificationRepository: dbRegistry.NotificationRepository,
		userService:            userService,
		healthMonitor:          infraReg.HealthMonitor,
		unitOfWork:             infraReg.UnitOfWork,
	}
}

// Handle processes push notification messages
func (c *NotificationPushConsumer) Handle(ctx context.Context, body []byte) error {
	zap.L().Info("Received push notification message",
		zap.Int("message_size", len(body)))

	// Parse message
	var msg consumers.PushNotificationMessage
	if err := json.Unmarshal(body, &msg); err != nil {
		// Try to unmarshal as UnifiedNotificationMessage
		var unifiedMsg consumers.UnifiedNotificationMessage
		if errUnified := json.Unmarshal(body, &unifiedMsg); errUnified == nil && unifiedMsg.UserID != uuid.Nil {
			// Convert Unified to Push message
			msg = *unifiedMsg.ToPushRequest()
		} else {
			zap.L().Error("Failed to unmarshal push notification message",
				zap.Error(err),
				zap.String("body", string(body)))
			return err // Parsing errors should not retry
		}
	}

	zap.L().Info("Processing push notification",
		zap.String("notification_id", msg.NotificationID.String()),
		zap.String("user_id", msg.UserID.String()),
		zap.String("title", msg.Title))

	// Check user notification preferences
	_, pushEnabled, err := c.userService.GetOrCreateDefault(ctx, msg.UserID)
	if err != nil {
		zap.L().Error("Failed to get notification preferences",
			zap.String("user_id", msg.UserID.String()),
			zap.String("notification_id", msg.NotificationID.String()),
			zap.Error(err))
		// Continue with send on error (fail-open approach)
	} else if !pushEnabled {
		zap.L().Info("Push notifications disabled for user, skipping send",
			zap.String("user_id", msg.UserID.String()),
			zap.String("notification_id", msg.NotificationID.String()))

		// Log attempt as completed with user preference
		c.logDeliveryAttempt(ctx, msg.NotificationID, 0, 0, "Push notifications disabled by user")

		// Return success (no retry needed)
		return nil
	}

	// Check FCM service health before attempting to send
	if c.healthMonitor != nil && !c.healthMonitor.IsFCMHealthy() {
		health := c.healthMonitor.GetFCMHealth()
		zap.L().Warn("FCM service is unhealthy, skipping send",
			zap.String("notification_id", msg.NotificationID.String()),
			zap.Error(health.LastError))

		// Log attempt as failed with service unavailable
		c.logDeliveryAttempt(ctx, msg.NotificationID, 0, 0, "FCM service is temporarily unavailable")

		// Return error to trigger RabbitMQ retry
		return errors.New("FCM service unhealthy")
	}

	// Fetch device tokens for user
	tokens, err := c.deviceTokenRepository.FindByUserID(ctx, msg.UserID)
	if err != nil {
		zap.L().Error("Failed to fetch device tokens for user",
			zap.String("user_id", msg.UserID.String()),
			zap.Error(err))
		return err // Database errors should retry
	}

	if len(tokens) == 0 {
		zap.L().Warn("No device tokens found for user",
			zap.String("user_id", msg.UserID.String()))
		// Not an error - user simply has no registered devices
		// Log attempt as completed with no recipients
		c.logDeliveryAttempt(ctx, msg.NotificationID, 0, 0, "No device tokens registered")
		return nil
	}

	// Separate tokens by type (Expo vs FCM)
	expoTokens := make([]string, 0)
	fcmTokens := make([]string, 0)

	for _, token := range tokens {
		if c.expoPushService != nil && c.expoPushService.IsExpoToken(token.Token) {
			expoTokens = append(expoTokens, token.Token)
		} else {
			fcmTokens = append(fcmTokens, token.Token)
		}
	}

	zap.L().Info("Categorized device tokens by type",
		zap.Int("expo_tokens", len(expoTokens)),
		zap.Int("fcm_tokens", len(fcmTokens)))

	totalSuccess := 0
	totalFailure := 0
	var lastError error

	// Send to Expo tokens if any
	if len(expoTokens) > 0 && c.expoPushService != nil {
		successCount, failureCount, invalidTokens, err := c.sendViaExpo(ctx, msg, expoTokens)
		totalSuccess += successCount
		totalFailure += failureCount

		if err != nil {
			zap.L().Error("Failed to send Expo notifications",
				zap.String("notification_id", msg.NotificationID.String()),
				zap.Error(err))
			lastError = err
		}

		// Mark invalid Expo tokens
		for _, token := range invalidTokens {
			if err := c.deviceTokenRepository.MarkInvalid(ctx, token); err != nil {
				zap.L().Error("Failed to mark Expo token as invalid",
					zap.String("token", token),
					zap.Error(err))
			}
		}
	}

	// Send to FCM tokens if any
	if len(fcmTokens) > 0 && c.fcmService != nil {
		successCount, failureCount, err := c.sendViaFCM(ctx, msg, fcmTokens)
		totalSuccess += successCount
		totalFailure += failureCount

		if err != nil {
			zap.L().Error("Failed to send FCM notifications",
				zap.String("notification_id", msg.NotificationID.String()),
				zap.Error(err))
			lastError = err
		}
	}

	// Log combined delivery attempt
	errorMsg := ""
	if lastError != nil {
		errorMsg = lastError.Error()
	}
	c.logDeliveryAttempt(ctx, msg.NotificationID, totalSuccess, totalFailure, errorMsg)

	zap.L().Info("Push notification processing completed",
		zap.String("notification_id", msg.NotificationID.String()),
		zap.Int("total_success", totalSuccess),
		zap.Int("total_failure", totalFailure))

	// If all deliveries failed, return error to trigger retry
	if totalSuccess == 0 && totalFailure > 0 {
		return errors.New("all push notifications failed to deliver")
	}

	return nil
}

// region: 1. ============== Helper Functions ==============

// region: 2. ============== Sending Functions ==============

// sendViaExpo sends notifications via Expo push service
func (c *NotificationPushConsumer) sendViaExpo(
	ctx context.Context,
	msg consumers.PushNotificationMessage,
	tokens []string,
) (successCount, failureCount int, invalidTokens []string, err error) {
	zap.L().Info("Sending via Expo push service",
		zap.Int("token_count", len(tokens)))

	// Expo doesn't support complex platform configs like FCM,
	// so we use simple title/body/data sending
	successCount, failureCount, invalidTokens, err = c.expoPushService.SendMulticast(
		ctx,
		tokens,
		msg.Title,
		msg.Body,
		msg.Data,
	)

	return successCount, failureCount, invalidTokens, err
}

// sendViaFCM sends notifications via Firebase Cloud Messaging
func (c *NotificationPushConsumer) sendViaFCM(
	ctx context.Context,
	msg consumers.PushNotificationMessage,
	tokens []string,
) (successCount, failureCount int, err error) {
	zap.L().Info("Sending via FCM service",
		zap.Int("token_count", len(tokens)))

	// Send via FCM with platform config
	var batchResp *messaging.BatchResponse
	if msg.PlatformConfig != nil {
		apnsConfig := c.buildAPNSConfig(msg.PlatformConfig.IOS)
		androidConfig := c.buildAndroidConfig(msg.PlatformConfig.Android)
		batchResp, err = c.fcmService.SendMulticastWithPlatformConfig(
			ctx,
			tokens,
			msg.Title,
			msg.Body,
			msg.Data,
			apnsConfig,
			androidConfig,
		)
	} else {
		batchResp, err = c.fcmService.SendMulticast(
			ctx,
			tokens,
			msg.Title,
			msg.Body,
			msg.Data,
		)
	}

	if err != nil {
		return 0, len(tokens), err
	}

	// Handle batch response and mark invalid FCM tokens
	c.processBatchResponse(ctx, batchResp, tokens)

	return batchResp.SuccessCount, batchResp.FailureCount, nil
}

// endregion 2.

// processBatchResponse handles the FCM batch response and marks invalid tokens
func (c *NotificationPushConsumer) processBatchResponse(ctx context.Context, batchResp *messaging.BatchResponse, tokens []string) {
	for i, resp := range batchResp.Responses {
		if resp.Error != nil {
			token := tokens[i]
			zap.L().Warn("FCM delivery failed for token",
				zap.String("token", token),
				zap.Error(resp.Error))

			// Check if token is invalid
			if messaging.IsUnregistered(resp.Error) ||
				messaging.IsInvalidArgument(resp.Error) {
				zap.L().Info("Marking token as invalid",
					zap.String("token", token))
				if err := c.deviceTokenRepository.MarkInvalid(ctx, token); err != nil {
					zap.L().Error("Failed to mark token as invalid",
						zap.String("token", token),
						zap.Error(err))
				}
			}
		}
	}
}

// logDeliveryAttempt logs the delivery attempt to the notifications table
func (c *NotificationPushConsumer) logDeliveryAttempt(ctx context.Context, notificationID uuid.UUID, successCount, failureCount int, errorMsg string) {
	// Create delivery attempt record
	status := "success"
	if successCount == 0 && failureCount > 0 {
		status = "failed"
	} else if successCount > 0 && failureCount > 0 {
		status = "partial"
	}

	attempt := model.DeliveryAttempt{
		Timestamp: time.Now(),
		Status:    status,
		Error:     errorMsg,
	}

	if err := helper.WithTransaction(ctx, c.unitOfWork, func(ctx context.Context, uow irepository.UnitOfWork) error {
		// Use atomic update for delivery attempt
		if err := c.notificationRepository.UpdateDeliveryAttempt(ctx, notificationID, attempt); err != nil {
			zap.L().Error("Failed to log delivery attempt",
				zap.String("notification_id", notificationID.String()),
				zap.Error(err))
		}

		// Update status
		if successCount > 0 {
			if err := c.notificationRepository.UpdateStatus(ctx, notificationID, enum.NotificationStatusSent); err != nil {
				zap.L().Error("Failed to update notification status",
					zap.String("notification_id", notificationID.String()),
					zap.Error(err))
			}
		} else if failureCount > 0 {
			// Fetch notification to check retry count
			notification, err := c.notificationRepository.GetByID(ctx, notificationID, nil)
			if err != nil {
				zap.L().Error("Failed to fetch notification for status update",
					zap.String("notification_id", notificationID.String()),
					zap.Error(err))
			} else {
				newStatus := enum.NotificationStatusRetrying
				if len(notification.DeliveryAttempts) >= 3 { // Max retries reached
					newStatus = enum.NotificationStatusFailed
				}

				if err := c.notificationRepository.UpdateStatus(ctx, notificationID, newStatus); err != nil {
					zap.L().Error("Failed to update notification status",
						zap.String("notification_id", notificationID.String()),
						zap.Error(err))
				}
			}
		}

		// Update error details if present
		if errorMsg != "" {
			now := time.Now()
			// Use model.ErrorDetails compatible with UpdateErrorDetails
			errorDetails := model.ErrorDetails{
				ErrorCode:     "push_delivery_failed",
				ErrorMessage:  errorMsg,
				LastAttemptAt: &now,
			}

			if err := c.notificationRepository.UpdateErrorDetails(ctx, notificationID, errorDetails); err != nil {
				zap.L().Error("Failed to update notification error details",
					zap.String("notification_id", notificationID.String()),
					zap.Error(err))
			}
		}

		return nil
	}); err != nil {
		zap.L().Error("Transaction failed while logging delivery attempt",
			zap.String("notification_id", notificationID.String()),
			zap.Error(err))
	}
}

// buildAPNSConfig constructs APNs configuration from DTO
func (c *NotificationPushConsumer) buildAPNSConfig(iosConfig *consumers.IOSConfig) *messaging.APNSConfig {
	if iosConfig == nil {
		return nil
	}

	apnsPayload := &messaging.Aps{}

	if iosConfig.Badge != nil {
		apnsPayload.Badge = iosConfig.Badge
	}

	if iosConfig.Sound != "" {
		apnsPayload.Sound = iosConfig.Sound
	}

	if iosConfig.Category != "" {
		apnsPayload.Category = iosConfig.Category
	}

	if iosConfig.ThreadID != "" {
		apnsPayload.ThreadID = iosConfig.ThreadID
	}

	if iosConfig.ContentAvailable {
		apnsPayload.ContentAvailable = true
	}

	if iosConfig.MutableContent {
		apnsPayload.MutableContent = true
	}

	config := &messaging.APNSConfig{
		Payload: &messaging.APNSPayload{
			Aps: apnsPayload,
		},
	}

	// Add custom data if present
	if len(iosConfig.CustomData) > 0 {
		config.Payload.CustomData = iosConfig.CustomData
	}

	return config
}

// buildAndroidConfig constructs Android configuration from DTO
func (c *NotificationPushConsumer) buildAndroidConfig(androidConfig *consumers.AndroidConfig) *messaging.AndroidConfig {
	if androidConfig == nil {
		return nil
	}

	config := &messaging.AndroidConfig{}

	if androidConfig.Priority != "" {
		config.Priority = androidConfig.Priority
	}

	if androidConfig.CollapseKey != "" {
		config.CollapseKey = androidConfig.CollapseKey
	}

	if androidConfig.TTL != "" {
		// Parse TTL string as duration (e.g., "3600s", "1h", "30m")
		if ttl, err := time.ParseDuration(androidConfig.TTL); err == nil {
			config.TTL = &ttl
		} else {
			zap.L().Warn("Invalid TTL format, skipping",
				zap.String("ttl", androidConfig.TTL),
				zap.Error(err))
		}
	}

	// Build notification
	notification := &messaging.AndroidNotification{}

	if androidConfig.ChannelID != "" {
		notification.ChannelID = androidConfig.ChannelID
	}

	if androidConfig.Sound != "" {
		notification.Sound = androidConfig.Sound
	}

	if androidConfig.Color != "" {
		notification.Color = androidConfig.Color
	}

	if androidConfig.Icon != "" {
		notification.Icon = androidConfig.Icon
	}

	if androidConfig.Tag != "" {
		notification.Tag = androidConfig.Tag
	}

	if androidConfig.ClickAction != "" {
		notification.ClickAction = androidConfig.ClickAction
	}

	config.Notification = notification

	// Add custom data if present
	if len(androidConfig.CustomData) > 0 {
		config.Data = androidConfig.CustomData
	}

	return config
}

// endregion
