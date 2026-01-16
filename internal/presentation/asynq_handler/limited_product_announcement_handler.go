package asynqhandler

import (
	"context"
	asynqtask "core-backend/internal/application/dto/asynq_tasks"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/application/interfaces/iservice"
	"core-backend/internal/domain/enum"
	"core-backend/pkg/utils"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// LimitedProductAnnouncementHandler handles limited product announcement tasks
// It sends notifications to all customers about upcoming limited product events
type LimitedProductAnnouncementHandler struct {
	notificationService iservice.NotificationService
	uow                 irepository.UnitOfWork
}

// NewLimitedProductAnnouncementHandler creates a new handler instance
func NewLimitedProductAnnouncementHandler(
	notificationService iservice.NotificationService,
	uow irepository.UnitOfWork,
) *LimitedProductAnnouncementHandler {
	return &LimitedProductAnnouncementHandler{
		notificationService: notificationService,
		uow:                 uow,
	}
}

// ProcessTask implements the asynq.Handler interface
func (h *LimitedProductAnnouncementHandler) ProcessTask(ctx context.Context, task *asynq.Task) error {
	var payload asynqtask.LimitedProductAnnouncementPayload
	if err := json.Unmarshal(task.Payload(), &payload); err != nil {
		zap.L().Error("Failed to unmarshal limited product announcement payload",
			zap.Error(err),
			zap.String("payload", string(task.Payload())))
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	zap.L().Info("Processing limited product announcement",
		zap.String("product_id", payload.ProductID.String()),
		zap.String("product_name", payload.ProductName),
		zap.String("announcement_type", string(payload.AnnouncementType)),
		zap.Time("target_date", payload.TargetDate))

	// Build notification title and body based on announcement type
	title, body := h.buildNotificationContent(payload)

	// Get all customers to notify
	customers, err := h.getAllCustomers(ctx)
	if err != nil {
		zap.L().Error("Failed to get customers for notification",
			zap.String("product_id", payload.ProductID.String()),
			zap.Error(err))
		return fmt.Errorf("failed to get customers: %w", err)
	}

	if len(customers) == 0 {
		zap.L().Info("No customers to notify for limited product announcement",
			zap.String("product_id", payload.ProductID.String()))
		return nil
	}

	// Send notifications to all customers
	successCount := 0
	failCount := 0

	data := map[string]string{
		"product_id":        payload.ProductID.String(),
		"product_name":      payload.ProductName,
		"announcement_type": string(payload.AnnouncementType),
		"target_date":       utils.FormatLocalTime(&payload.TargetDate, utils.DateFormat),
	}

	for _, customerID := range customers {
		// Use notification service to send
		if err := h.sendNotification(ctx, customerID, title, body, data); err != nil {
			failCount++
			zap.L().Warn("Failed to send notification to customer",
				zap.String("customer_id", customerID.String()),
				zap.Error(err))
			continue
		}
		successCount++
	}

	zap.L().Info("Limited product announcement completed",
		zap.String("product_id", payload.ProductID.String()),
		zap.String("product_name", payload.ProductName),
		zap.String("announcement_type", string(payload.AnnouncementType)),
		zap.Int("total_customers", len(customers)),
		zap.Int("success_count", successCount),
		zap.Int("fail_count", failCount))

	return nil
}

// buildNotificationContent creates the notification title and body based on announcement type
func (h *LimitedProductAnnouncementHandler) buildNotificationContent(payload asynqtask.LimitedProductAnnouncementPayload) (string, string) {
	productName := payload.ProductName
	targetDateStr := utils.FormatLocalTime(&payload.TargetDate, utils.DateFormat)

	var title, body string

	switch payload.AnnouncementType {
	case asynqtask.AnnouncementTypePremiereDate3Days:
		title = "🎬 Premiere Coming Soon!"
		body = fmt.Sprintf("Don't miss it! \"%s\" premieres in 3 days on %s. Mark your calendar!", productName, targetDateStr)

	case asynqtask.AnnouncementTypePremiereDate1Day:
		title = "🎬 Premiere Tomorrow!"
		body = fmt.Sprintf("Get ready! \"%s\" premieres TOMORROW on %s. Don't miss the exclusive launch!", productName, targetDateStr)

	case asynqtask.AnnouncementTypeAvailability3Days:
		title = "🛒 Limited Product Available Soon!"
		body = fmt.Sprintf("Mark your calendar! \"%s\" will be available for purchase in 3 days on %s.", productName, targetDateStr)

	case asynqtask.AnnouncementTypeAvailability1Day:
		title = "🛒 Available Tomorrow!"
		body = fmt.Sprintf("Get ready to shop! \"%s\" goes on sale TOMORROW on %s. Limited quantities available!", productName, targetDateStr)

	default:
		title = "📢 Limited Product Update"
		body = fmt.Sprintf("Important update about \"%s\". Check it out!", productName)
	}

	return title, body
}

// getAllCustomers retrieves all customer user IDs for notification
func (h *LimitedProductAnnouncementHandler) getAllCustomers(ctx context.Context) ([]uuid.UUID, error) {
	userRepo := h.uow.Users()

	// Get all active customers
	users, _, err := userRepo.GetAll(ctx, func(db *gorm.DB) *gorm.DB {
		return db.Where("role = ? AND deleted_at IS NULL", enum.UserRoleCustomer.String())
	}, nil, 0, 0)

	if err != nil {
		return nil, err
	}

	customerIDs := make([]uuid.UUID, 0, len(users))
	for _, user := range users {
		customerIDs = append(customerIDs, user.ID)
	}

	return customerIDs, nil
}

// sendNotification sends a notification using the notification service
func (h *LimitedProductAnnouncementHandler) sendNotification(ctx context.Context, customerID uuid.UUID, title, body string, data map[string]string) error {
	// Create notification request
	notificationReq := &requests.PublishNotificationRequest{
		UserID: customerID,
		Title:  title,
		Body:   body,
		Types:  []enum.NotificationType{enum.NotificationTypePush, enum.NotificationTypeInApp},
		Data:   data,
	}

	if _, err := h.notificationService.CreateAndPublishNotification(ctx, notificationReq); err != nil {
		return err
	}

	return nil
}
