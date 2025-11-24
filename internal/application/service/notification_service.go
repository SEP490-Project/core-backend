package service

import (
	"context"
	"core-backend/internal/application/dto/consumers"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/application/interfaces/iservice"
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
	"core-backend/internal/infrastructure/rabbitmq"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

const (
	emailProducerName = "notification-email-producer"
	pushProducerName  = "notification-push-producer"
	inAppProducerName = "notification-in-app-producer"
	allProducerName   = "notification-all-producer"
)

// NotificationService implements notification monitoring operations
type NotificationService struct {
	notificationRepo irepository.NotificationRepository
	userRepo         irepository.GenericRepository[model.User]
	rabbitmq         *rabbitmq.RabbitMQ
	realTimeNotifier iservice.SSEService
}

// NewNotificationService creates a new notification service instance
func NewNotificationService(
	notificationRepo irepository.NotificationRepository,
	userRepo irepository.GenericRepository[model.User],
	rabbitmq *rabbitmq.RabbitMQ,
	realTimeNotifier iservice.SSEService,
) *NotificationService {
	return &NotificationService{
		notificationRepo: notificationRepo,
		userRepo:         userRepo,
		rabbitmq:         rabbitmq,
		realTimeNotifier: realTimeNotifier,
	}
}

// GetByID retrieves a notification by its ID
func (s *NotificationService) GetByID(ctx context.Context, id uuid.UUID) (*model.Notification, error) {
	zap.L().Info("Fetching notification by ID",
		zap.String("notification_id", id.String()))

	notification, err := s.notificationRepo.GetByID(ctx, id, nil)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			zap.L().Warn("Notification not found",
				zap.String("notification_id", id.String()))
			return nil, errors.New("notification not found")
		}
		zap.L().Error("Failed to fetch notification",
			zap.String("notification_id", id.String()),
			zap.Error(err))
		return nil, errors.New("failed to fetch notification")
	}

	return notification, nil
}

// GetByUser retrieves notifications for a specific user with pagination
func (s *NotificationService) GetByUser(ctx context.Context, userID uuid.UUID, page, limit int) ([]*model.Notification, int64, error) {
	zap.L().Info("Fetching notifications for user",
		zap.String("user_id", userID.String()),
		zap.Int("page", page),
		zap.Int("limit", limit))

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	notifications, total, err := s.notificationRepo.GetAll(
		ctx,
		func(db *gorm.DB) *gorm.DB {
			return db.Where("user_id = ?", userID).Order("created_at DESC")
		},
		nil,
		limit,
		page,
	)

	if err != nil {
		zap.L().Error("Failed to fetch notifications for user",
			zap.String("user_id", userID.String()),
			zap.Error(err))
		return nil, 0, errors.New("failed to fetch notifications")
	}

	zap.L().Info("Fetched notifications for user",
		zap.String("user_id", userID.String()),
		zap.Int("count", len(notifications)),
		zap.Int64("total", total))

	// Convert []model.Notification to []*model.Notification
	notificationPtrs := make([]*model.Notification, len(notifications))
	for i := range notifications {
		notificationPtrs[i] = &notifications[i]
	}

	return notificationPtrs, total, nil
}

// GetByStatus retrieves notifications by status with pagination
func (s *NotificationService) GetByStatus(ctx context.Context, status enum.NotificationStatus, page, limit int) ([]*model.Notification, int64, error) {
	zap.L().Info("Fetching notifications by status",
		zap.String("status", string(status)),
		zap.Int("page", page),
		zap.Int("limit", limit))

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	if !status.IsValid() {
		return nil, 0, errors.New("invalid notification status")
	}

	notifications, total, err := s.notificationRepo.GetAll(
		ctx,
		func(db *gorm.DB) *gorm.DB {
			return db.Where("status = ?", status).Order("created_at DESC")
		},
		nil,
		limit,
		page,
	)

	if err != nil {
		zap.L().Error("Failed to fetch notifications by status",
			zap.String("status", string(status)),
			zap.Error(err))
		return nil, 0, errors.New("failed to fetch notifications")
	}

	zap.L().Info("Fetched notifications by status",
		zap.String("status", string(status)),
		zap.Int("count", len(notifications)),
		zap.Int64("total", total))

	// Convert []model.Notification to []*model.Notification
	notificationPtrs := make([]*model.Notification, len(notifications))
	for i := range notifications {
		notificationPtrs[i] = &notifications[i]
	}

	return notificationPtrs, total, nil
}

// GetFailedWithRetries retrieves notifications that failed after multiple retry attempts
func (s *NotificationService) GetFailedWithRetries(ctx context.Context, minRetries int, page, limit int) ([]*model.Notification, int64, error) {
	zap.L().Info("Fetching failed notifications with retries",
		zap.Int("min_retries", minRetries),
		zap.Int("page", page),
		zap.Int("limit", limit))

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	notifications, err := s.notificationRepo.FindFailedWithRetries(ctx, minRetries)
	if err != nil {
		zap.L().Error("Failed to fetch notifications with retries",
			zap.Int("min_retries", minRetries),
			zap.Error(err))
		return nil, 0, errors.New("failed to fetch failed notifications")
	}

	// Manual pagination since this uses a custom query
	total := int64(len(notifications))
	start := (page - 1) * limit
	end := start + limit

	if start >= len(notifications) {
		return []*model.Notification{}, total, nil
	}
	if end > len(notifications) {
		end = len(notifications)
	}

	paginatedNotifications := notifications[start:end]

	zap.L().Info("Fetched failed notifications with retries",
		zap.Int("count", len(paginatedNotifications)),
		zap.Int64("total", total))

	return paginatedNotifications, total, nil
}

// GetByFilters retrieves notifications with multiple filter criteria
func (s *NotificationService) GetByFilters(
	ctx context.Context,
	userID *uuid.UUID,
	notificationType *enum.NotificationType,
	status *enum.NotificationStatus,
	isRead *bool,
	startDate, endDate *string,
	page, limit int,
) ([]*model.Notification, int64, error) {
	zap.L().Info("Fetching notifications with filters",
		zap.Int("page", page),
		zap.Int("limit", limit))

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	// Validate enums if provided
	if notificationType != nil && !notificationType.IsValid() {
		return nil, 0, errors.New("invalid notification type")
	}
	if status != nil && !status.IsValid() {
		return nil, 0, errors.New("invalid notification status")
	}

	// Parse dates if provided
	var startTime, endTime *time.Time
	if startDate != nil && *startDate != "" {
		t, err := time.Parse(time.RFC3339, *startDate)
		if err != nil {
			t, err = time.Parse("2006-01-02", *startDate)
			if err != nil {
				return nil, 0, errors.New("invalid start date format, use RFC3339 or YYYY-MM-DD")
			}
		}
		startTime = &t
	}
	if endDate != nil && *endDate != "" {
		t, err := time.Parse(time.RFC3339, *endDate)
		if err != nil {
			t, err = time.Parse("2006-01-02", *endDate)
			if err != nil {
				return nil, 0, errors.New("invalid end date format, use RFC3339 or YYYY-MM-DD")
			}
		}
		endTime = &t
	}

	notifications, total, err := s.notificationRepo.GetAll(
		ctx,
		func(db *gorm.DB) *gorm.DB {
			query := db

			if userID != nil {
				query = query.Where("user_id = ?", *userID)
			}
			if notificationType != nil {
				query = query.Where("type = ?", *notificationType)
			}
			if status != nil {
				query = query.Where("status = ?", *status)
			}
			if isRead != nil {
				query = query.Where("is_read = ?", *isRead)
			}
			if startTime != nil {
				query = query.Where("created_at >= ?", *startTime)
			}
			if endTime != nil {
				query = query.Where("created_at <= ?", *endTime)
			}

			return query.Order("created_at DESC")
		},
		nil,
		limit,
		page,
	)

	if err != nil {
		zap.L().Error("Failed to fetch notifications with filters", zap.Error(err))
		return nil, 0, errors.New("failed to fetch notifications")
	}

	zap.L().Info("Fetched notifications with filters",
		zap.Int("count", len(notifications)),
		zap.Int64("total", total))

	// Convert []model.Notification to []*model.Notification
	notificationPtrs := make([]*model.Notification, len(notifications))
	for i := range notifications {
		notificationPtrs[i] = &notifications[i]
	}

	return notificationPtrs, total, nil
}

// CreateAndPublishNotification creates notification records and publishes them to specified channels
func (s *NotificationService) CreateAndPublishNotification(ctx context.Context, req *requests.PublishNotificationRequest) ([]uuid.UUID, error) {
	zap.L().Info("Creating and publishing notification to multiple channels",
		zap.String("user_id", req.UserID.String()),
		zap.Strings("channels", req.Channels))

	// Verify user exists
	user, err := s.validateUserExists(ctx, req.UserID, nil)
	if err != nil {
		return nil, err
	}

	// Process each channel
	notificationIDs := make([]uuid.UUID, 0, len(req.Channels))
	for _, channel := range req.Channels {
		notificationType := enum.NotificationType(channel)
		if !notificationType.IsValid() {
			zap.L().Warn("Invalid notification channel", zap.String("channel", channel))
			continue
		}

		switch notificationType {
		case enum.NotificationTypeEmail:
			emailReq := req.ToEmailRequest()
			emailReq.To = user.Email
			var notificationID uuid.UUID
			notificationID, err = s.CreateAndPublishEmail(ctx, emailReq)
			if err != nil {
				zap.L().Error("Failed to publish email notification", zap.Error(err))
				continue
			}
			notificationIDs = append(notificationIDs, notificationID)

		case enum.NotificationTypePush:
			pushReq := req.ToPushRequest()
			var notificationID uuid.UUID
			notificationID, err = s.CreateAndPublishPush(ctx, pushReq)
			if err != nil {
				zap.L().Error("Failed to publish push notification", zap.Error(err))
				continue
			}
			notificationIDs = append(notificationIDs, notificationID)

		case enum.NotificationTypeInApp:
			inAppReq := req.ToInAppRequest()
			var notificationID uuid.UUID
			notificationID, err = s.CreateAndPublishInApp(ctx, inAppReq)
			if err != nil {
				zap.L().Error("Failed to publish in-app notification", zap.Error(err))
				continue
			}
			notificationIDs = append(notificationIDs, notificationID)

		default:
			zap.L().Warn("Unsupported notification channel, skipping requests",
				zap.String("channel", channel),
				zap.Any("request", req))
			continue
		}
	}

	if len(notificationIDs) == 0 {
		return nil, errors.New("failed to publish notifications to any channel")
	}

	zap.L().Info("Successfully published notifications",
		zap.Int("count", len(notificationIDs)))

	return notificationIDs, nil
}

// CreateAndPublishEmail creates an email notification record and publishes it
func (s *NotificationService) CreateAndPublishEmail(ctx context.Context, req *requests.PublishEmailRequest) (uuid.UUID, error) {
	zap.L().Info("Creating and publishing email notification",
		zap.String("user_id", req.UserID.String()),
		zap.String("to", req.To))

	// Verify user exists
	if _, err := s.validateUserExists(ctx, req.UserID, &req.To); err != nil {
		return uuid.Nil, err
	}

	// Validate that either template or body is provided
	if req.TemplateName == nil && req.HTMLBody == nil {
		return uuid.Nil, errors.New("either template_name or html_body must be provided")
	}

	// Create persisting notification record and consumer message
	notificationID := uuid.New()
	notification := &model.Notification{
		ID:            notificationID,
		UserID:        req.UserID,
		Type:          enum.NotificationTypeEmail,
		Status:        enum.NotificationStatusPending,
		RecipientInfo: model.JSONBRecipientInfo{Email: req.To},
		ContentData:   model.JSONBContentData{Subject: req.Subject},
	}

	emailMsg := &consumers.EmailNotificationMessage{
		NotificationID: notificationID,
		UserID:         req.UserID,
		To:             req.To,
		Subject:        req.Subject,
		Priority:       req.Priority,
		Metadata:       req.Metadata,
	}

	if req.TemplateName != nil {
		notification.ContentData.TemplateName = *req.TemplateName
		if req.TemplateData != nil {
			// Store template data as JSON
			notification.ContentData.TemplateData = req.TemplateData
		}
		emailMsg.TemplateName = *req.TemplateName
		emailMsg.TemplateData = req.TemplateData
	} else if req.HTMLBody != nil {
		notification.ContentData.Body = *req.HTMLBody
		emailMsg.HTMLBody = *req.HTMLBody
	}

	if err := s.notificationRepo.Add(ctx, notification); err != nil {
		zap.L().Error("Failed to create email notification record", zap.Error(err))
		return uuid.Nil, errors.New("failed to create notification record")
	}

	if err := s.sendEmailNotification(ctx, emailMsg); err != nil {
		return uuid.Nil, fmt.Errorf("failed to publish notification: %s", err.Error())
	}

	zap.L().Info("Successfully published email notification",
		zap.String("notification_id", notificationID.String()))

	// Push unread count to user
	go s.pushUnreadCount(context.Background(), req.UserID)

	return notificationID, nil
}

// CreateAndPublishPush creates a push notification record and publishes it
func (s *NotificationService) CreateAndPublishPush(ctx context.Context, req *requests.PublishPushRequest) (uuid.UUID, error) {
	zap.L().Info("Creating and publishing push notification",
		zap.String("user_id", req.UserID.String()),
		zap.String("title", req.Title))

	// Verify user exists
	if _, err := s.validateUserExists(ctx, req.UserID, nil); err != nil {
		return uuid.Nil, err
	}

	// Create notification record and consumer message
	notificationID := uuid.New()
	notification := &model.Notification{
		ID:     notificationID,
		UserID: req.UserID,
		Type:   enum.NotificationTypePush,
		Status: enum.NotificationStatusPending,
		RecipientInfo: model.JSONBRecipientInfo{
			Tokens: []string{}, // Will be populated when sending
		},
		ContentData: model.JSONBContentData{
			Title: req.Title,
			Body:  req.Body,
		},
	}
	pushMsg := &consumers.PushNotificationMessage{
		NotificationID: notificationID,
		UserID:         req.UserID,
		Title:          req.Title,
		Body:           req.Body,
		Data:           req.Data,
	}

	// Add platform-specific configurations if provided
	if req.IOSBadge != nil || req.IOSSound != nil || req.AndroidPriority != nil || req.AndroidNotificationTag != nil {
		platformConfig := &model.PlatformConfig{}
		consumerPlatformConfig := &consumers.PlatformConfig{}

		if req.IOSBadge != nil || req.IOSSound != nil {
			platformConfig.IOSConfig = &model.IOSConfig{Badge: req.IOSBadge}
			consumerPlatformConfig.IOS = &consumers.IOSConfig{Badge: req.IOSBadge}
			if req.IOSSound != nil {
				platformConfig.IOSConfig.Sound = *req.IOSSound
				consumerPlatformConfig.IOS.Sound = *req.IOSSound
			}
		}

		if req.AndroidPriority != nil || req.AndroidNotificationTag != nil {
			platformConfig.AndroidConfig = &model.AndroidConfig{}
			consumerPlatformConfig.Android = &consumers.AndroidConfig{}
			if req.AndroidPriority != nil {
				platformConfig.AndroidConfig.Priority = *req.AndroidPriority
				consumerPlatformConfig.Android.Priority = *req.AndroidPriority
			}
			if req.AndroidNotificationTag != nil {
				platformConfig.AndroidConfig.Color = *req.AndroidNotificationTag // Using Color field as tag
				consumerPlatformConfig.Android.Tag = *req.AndroidNotificationTag
			}
		}

		notification.PlatformConfig = model.JSONBPlatformConfig(*platformConfig)
		pushMsg.PlatformConfig = consumerPlatformConfig
	}

	if err := s.notificationRepo.Add(ctx, notification); err != nil {
		zap.L().Error("Failed to create push notification record", zap.Error(err))
		return uuid.Nil, errors.New("failed to create notification record")
	}

	// Publish to RabbitMQ
	if err := s.sendPushNotification(ctx, pushMsg); err != nil {
		return uuid.Nil, fmt.Errorf("failed to publish notification: %s", err.Error())
	}

	zap.L().Info("Successfully published push notification",
		zap.String("notification_id", notificationID.String()))

	// Push unread count to user
	go s.pushUnreadCount(context.Background(), req.UserID)

	return notificationID, nil
}

// CreateAndPublishInApp creates an in-app notification record and publishes it
func (s *NotificationService) CreateAndPublishInApp(ctx context.Context, req *requests.PublishInAppRequest) (uuid.UUID, error) {
	zap.L().Info("Creating and publishing in-app notification",
		zap.String("user_id", req.UserID.String()),
		zap.String("title", req.Title))

	// Verify user exists
	if _, err := s.validateUserExists(ctx, req.UserID, nil); err != nil {
		return uuid.Nil, err
	}

	// Create notification record
	notificationID := uuid.New()
	notification := &model.Notification{
		ID:            notificationID,
		UserID:        req.UserID,
		Type:          enum.NotificationTypeInApp,
		Status:        enum.NotificationStatusSent, // In-app notifications are immediately "sent"
		IsRead:        false,
		RecipientInfo: model.JSONBRecipientInfo{
			// No specific recipient info needed for in-app, but we can store user ID again or leave empty
		},
		ContentData: model.JSONBContentData{
			Title: req.Title,
			Body:  req.Body,
		},
	}

	if err := s.notificationRepo.Add(ctx, notification); err != nil {
		zap.L().Error("Failed to create in-app notification record", zap.Error(err))
		return uuid.Nil, errors.New("failed to create notification record")
	}

	// Publish to RabbitMQ
	producer, err := s.rabbitmq.GetProducer(inAppProducerName)
	if err != nil {
		zap.L().Error("Failed to get in-app notification producer", zap.Error(err))
		// Don't fail the request, just log error (notification is saved in DB)
	} else {
		msg := consumers.InAppNotificationMessage{
			NotificationID: notificationID,
			UserID:         req.UserID,
			Title:          req.Title,
			Message:        req.Body,
			Type:           "info", // Default type
			Data:           req.Data,
			CreatedAt:      time.Now().Format(time.RFC3339),
		}

		if err := producer.PublishJSON(ctx, msg); err != nil {
			zap.L().Error("Failed to publish in-app notification message",
				zap.String("notification_id", notificationID.String()),
				zap.Error(err))
		} else {
			zap.L().Info("Published in-app notification message",
				zap.String("notification_id", notificationID.String()))
		}
	}

	zap.L().Info("Successfully published in-app notification",
		zap.String("notification_id", notificationID.String()))

	// Push unread count to user (this is also done by consumer, but doing it here gives immediate feedback if connected)
	go s.pushUnreadCount(context.Background(), req.UserID)

	return notificationID, nil
}

// BroadcastToUser sends a unified notification to a specific user across specified channels
func (s *NotificationService) BroadcastToUser(ctx context.Context, userID uuid.UUID, title, body string, data map[string]string, channels []string) error {
	zap.L().Info("Broadcasting notification to user",
		zap.String("user_id", userID.String()),
		zap.Strings("channels", channels))

	// Verify user exists
	if _, err := s.validateUserExists(ctx, userID, nil); err != nil {
		return err
	}

	// Create notification record (generic type or primary type?)
	// For broadcast, we might want to create one record per channel or one master record.
	// Current system seems to be 1 record = 1 type.
	// If we use "notification.all", we are sending to multiple queues.
	// But we need a record in DB for history.
	// Let's create an IN_APP record as the "master" record for history if InApp is included,
	// or just create one record and let the consumers handle it?
	// If we create one record, say Type=IN_APP, but send to Email queue, Email consumer might be confused if it expects Type=EMAIL.
	// However, our consumers now handle UnifiedNotificationMessage.

	// Strategy: Create one notification record of type "SYSTEM" or "BROADCAST" if we had it,
	// but for now let's default to IN_APP as it's the most generic.
	// OR, we iterate and create records like CreateAndPublishNotification does.
	// BUT the user wants to use the "notification.all" routing key to broadcast.

	// If we use "notification.all", the SAME message goes to all queues.
	// That message needs a NotificationID.
	// So we must create ONE notification record that represents this broadcast.
	// Let's use Type=IN_APP as the primary type for the record, as it's most likely to be viewed in the app.

	notificationID := uuid.New()
	notification := &model.Notification{
		ID:            notificationID,
		UserID:        userID,
		Type:          enum.NotificationTypeInApp, // Defaulting to InApp for broadcast records
		Status:        enum.NotificationStatusSent,
		IsRead:        false,
		RecipientInfo: model.JSONBRecipientInfo{
			// We could store target channels here if we extend the struct
		},
		ContentData: model.JSONBContentData{
			Title: title,
			Body:  body,
		},
	}

	if err := s.notificationRepo.Add(ctx, notification); err != nil {
		return fmt.Errorf("failed to create notification record: %w", err)
	}

	// Publish to "notification.all"
	producer, err := s.rabbitmq.GetProducer(allProducerName)
	if err != nil {
		return fmt.Errorf("failed to get all-producer: %w", err)
	}

	msg := consumers.UnifiedNotificationMessage{
		NotificationID: notificationID,
		UserID:         userID,
		Title:          title,
		Body:           body,
		Data:           data,
		Type:           "info",
		TargetChannels: channels,
		CreatedAt:      time.Now().Format(time.RFC3339),
	}

	if err := producer.PublishJSON(ctx, msg); err != nil {
		return fmt.Errorf("failed to publish broadcast message: %w", err)
	}

	return nil
}

// BroadcastToAll sends a unified notification to all users (optionally filtered by role)
func (s *NotificationService) BroadcastToAll(ctx context.Context, title, body string, data map[string]string, role *string) error {
	zap.L().Info("Broadcasting notification to all users",
		zap.Stringp("role", role))

	// Fetch users (with role filter if provided)
	// This could be heavy if there are many users. For a real production system,
	// we should use a batch job or a specific "Broadcast" queue that expands the list.
	// For now, we'll iterate.

	limit := 100
	page := 1

	for {
		users, total, err := s.userRepo.GetAll(ctx, func(db *gorm.DB) *gorm.DB {
			if role != nil {
				return db.Where("role = ?", *role)
			}
			return db
		}, nil, limit, page)

		if err != nil {
			return fmt.Errorf("failed to fetch users for broadcast: %w", err)
		}

		if len(users) == 0 {
			break
		}

		// Process batch
		for _, user := range users {
			// We use "all" channels by default for broadcast to all
			// Or we could check user preferences here?
			// Let's send to all channels and let consumers/preferences handle it.
			channels := []string{"EMAIL", "PUSH", "IN_APP"}

			// Call BroadcastToUser for each user
			// We do this asynchronously to not block the loop too much,
			// but we need to be careful about overwhelming the DB/RabbitMQ.
			// For safety in this implementation, we'll do it synchronously or with a worker pool.
			// Synchronous for simplicity and reliability for now.
			if err := s.BroadcastToUser(ctx, user.ID, title, body, data, channels); err != nil {
				zap.L().Error("Failed to broadcast to user",
					zap.String("user_id", user.ID.String()),
					zap.Error(err))
				// Continue to next user
			}
		}

		if int64(page*limit) >= total {
			break
		}
		page++
	}

	return nil
}

// RepublishFailedNotifications republishes failed notifications based on filter criteria
func (s *NotificationService) RepublishFailedNotifications(ctx context.Context, req *requests.RepublishFailedNotificationRequest) (int, error) {
	zap.L().Info("Republishing failed notifications",
		zap.Int("notification_ids_count", len(req.NotificationIDs)))

	var notifications []*model.Notification
	var err error

	// Fetch notifications based on criteria
	if len(req.NotificationIDs) > 0 {
		// Fetch specific notifications by IDs
		for _, id := range req.NotificationIDs {
			var notification *model.Notification
			notification, err = s.notificationRepo.GetByID(ctx, id, nil)
			if err != nil {
				zap.L().Warn("Failed to fetch notification", zap.String("id", id.String()), zap.Error(err))
				continue
			}
			notifications = append(notifications, notification)
		}
	} else {
		// Fetch failed notifications with filters
		minRetries := 1
		if req.MinRetries != nil {
			minRetries = *req.MinRetries
		}

		notifications, err = s.notificationRepo.FindFailedWithRetries(ctx, minRetries)
		if err != nil {
			zap.L().Error("Failed to fetch failed notifications", zap.Error(err))
			return 0, errors.New("failed to fetch failed notifications")
		}

		// Filter by type if specified
		if req.Type != nil {
			notificationType := enum.NotificationType(*req.Type)
			if !notificationType.IsValid() {
				return 0, errors.New("invalid notification type")
			}

			filtered := make([]*model.Notification, 0)
			for _, n := range notifications {
				if n.Type == notificationType {
					filtered = append(filtered, n)
				}
			}
			notifications = filtered
		}
	}

	if len(notifications) == 0 {
		return 0, errors.New("no failed notifications found matching criteria")
	}

	successCount := 0

	// Republish each notification
	for _, notification := range notifications {
		var producer *rabbitmq.Producer
		var err error

		switch notification.Type {
		case enum.NotificationTypeEmail:
			producer, err = s.rabbitmq.GetProducer("notification-email-producer")
			if err != nil {
				zap.L().Error("Failed to get email producer", zap.Error(err))
				continue
			}

			// Reconstruct email message from notification record
			emailMsg := &consumers.EmailNotificationMessage{
				NotificationID: notification.ID,
				UserID:         notification.UserID,
				To:             notification.RecipientInfo.Email,
				Subject:        notification.ContentData.Subject,
				Body:           notification.ContentData.Body,
			}

			if notification.ContentData.TemplateName != "" {
				emailMsg.TemplateName = notification.ContentData.TemplateName
				emailMsg.TemplateData = notification.ContentData.TemplateData
			} else if notification.ContentData.Body != "" {
				emailMsg.HTMLBody = notification.ContentData.Body
			}

			if err = producer.PublishJSON(ctx, emailMsg); err != nil {
				zap.L().Error("Failed to republish email notification",
					zap.String("notification_id", notification.ID.String()),
					zap.Error(err))
				continue
			}

		case enum.NotificationTypePush:
			producer, err = s.rabbitmq.GetProducer("notification-push-producer")
			if err != nil {
				zap.L().Error("Failed to get push producer", zap.Error(err))
				continue
			}

			// Reconstruct push message from notification record
			pushMsg := &consumers.PushNotificationMessage{
				NotificationID: notification.ID,
				UserID:         notification.UserID,
				Title:          notification.ContentData.Title,
				Body:           notification.ContentData.Body,
				Data:           make(map[string]string), // Data is not stored in ContentData
			}

			// Note: PlatformConfig needs manual conversion if needed
			// For simplicity, we'll send without platform config on republish

			if err := producer.PublishJSON(ctx, pushMsg); err != nil {
				zap.L().Error("Failed to republish push notification",
					zap.String("notification_id", notification.ID.String()),
					zap.Error(err))
				continue
			}

		default:
			zap.L().Warn("Unknown notification type",
				zap.String("notification_id", notification.ID.String()),
				zap.String("type", string(notification.Type)))
			continue
		}

		// Update status to RETRYING
		notification.Status = enum.NotificationStatusRetrying
		if err := s.notificationRepo.Update(ctx, notification); err != nil {
			zap.L().Warn("Failed to update notification status",
				zap.String("notification_id", notification.ID.String()),
				zap.Error(err))
		}

		successCount++
	}

	zap.L().Info("Republished failed notifications",
		zap.Int("success_count", successCount),
		zap.Int("total_attempted", len(notifications)))

	return successCount, nil
}

// MarkAsRead marks a notification as read
func (s *NotificationService) MarkAsRead(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	// Verify notification belongs to user
	notification, err := s.notificationRepo.GetByID(ctx, id, nil)
	if err != nil {
		return err
	}
	if notification.UserID != userID {
		return errors.New("notification not found")
	}

	if err := s.notificationRepo.MarkAsRead(ctx, id); err != nil {
		return err
	}

	// Push updated unread count
	go s.pushUnreadCount(context.Background(), userID)

	return nil
}

// MarkAllAsRead marks all notifications as read for a user
func (s *NotificationService) MarkAllAsRead(ctx context.Context, userID uuid.UUID) error {
	if err := s.notificationRepo.MarkAllAsRead(ctx, userID); err != nil {
		return err
	}

	// Push updated unread count (should be 0)
	go s.pushUnreadCount(context.Background(), userID)

	return nil
}

// SubscribeSSE subscribes a user to real-time notification updates
func (s *NotificationService) SubscribeSSE(userID uuid.UUID) (<-chan iservice.SSEMessage, func()) {
	return s.realTimeNotifier.Subscribe(userID)
}

func (s *NotificationService) GetUnreadCount(ctx context.Context, userID uuid.UUID) (int64, error) {
	zap.L().Info("NotificationService - GetUnreadCount called",
		zap.String("user_id", userID.String()))

	return s.notificationRepo.CountUnread(ctx, userID)
}

// region: ============ Helper Methods =============

func (s *NotificationService) validateUserExists(ctx context.Context, userID uuid.UUID, email *string) (*model.User, error) {
	user, err := s.userRepo.GetByCondition(ctx, func(db *gorm.DB) *gorm.DB {
		if userID != uuid.Nil {
			return db.Where("id = ?", userID)
		}
		if email != nil {
			return db.Where("email = ?", *email)
		}
		return db
	}, nil)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			zap.L().Warn("User not found", zap.String("user_id", userID.String()))
			return nil, errors.New("user not found")
		}
		zap.L().Error("Failed to fetch user", zap.String("user_id", userID.String()), zap.Error(err))
		return nil, errors.New("failed to fetch user")
	}
	return user, nil
}

func (s *NotificationService) sendEmailNotification(ctx context.Context, emailMessage *consumers.EmailNotificationMessage) error {
	emailProducer, err := s.rabbitmq.GetProducer(emailProducerName)
	if err != nil {
		zap.L().Error("Failed to get email producer",
			zap.String("producer_name", emailProducerName),
			zap.Error(err))
		return errors.New("failed to get email producer")
	}

	if err = emailProducer.PublishJSON(ctx, emailMessage); err != nil {
		zap.L().Error("Failed to publish email notification",
			zap.String("notification_id", emailMessage.NotificationID.String()),
			zap.Error(err))
		return fmt.Errorf("failed to publish email notification: %w", err)
	}
	return nil
}

func (s *NotificationService) sendPushNotification(ctx context.Context, pushMessage *consumers.PushNotificationMessage) error {
	pushProducer, err := s.rabbitmq.GetProducer(pushProducerName)
	if err != nil {
		zap.L().Error("Failed to get push producer",
			zap.String("producer_name", pushProducerName),
			zap.Error(err))
		return err
	}

	if err = pushProducer.PublishJSON(ctx, pushMessage); err != nil {
		zap.L().Error("Failed to publish push notification",
			zap.String("notification_id", pushMessage.NotificationID.String()),
			zap.Error(err))
		return fmt.Errorf("failed to publish push notification: %w", err)
	}

	return nil
}

func (s *NotificationService) pushUnreadCount(ctx context.Context, userID uuid.UUID) {
	count, err := s.notificationRepo.CountUnread(ctx, userID)
	if err != nil {
		zap.L().Error("Failed to count unread notifications", zap.Error(err), zap.String("user_id", userID.String()))
		return
	}

	if err := s.realTimeNotifier.SendUnreadCount(userID, count); err != nil {
		zap.L().Error("Failed to push unread count", zap.Error(err), zap.String("user_id", userID.String()))
	}
}

// endregion
