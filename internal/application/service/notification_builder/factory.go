package notification_builder

import (
	"context"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
	"fmt"
)

// Notification payload helper types used by builders
type EmailNotificationPayload struct {
	EmailSubject      *string
	EmailTemplateName *string
	EmailTemplateData map[string]interface{}
	EmailHTMLBody     *string
}

type PushNotificationPayload struct {
	Title string
	Body  string
	Data  map[string]string
}

func BuildOrderNotification(
	ctx context.Context,
	status enum.OrderStatus,
	order *model.Order,
	user *model.User,
) (requests.PublishNotificationRequest, error) {
	builder, exists := notificationOrderBuilders[status]
	if !exists || builder == nil {
		return requests.PublishNotificationRequest{}, fmt.Errorf("notification_builder: no builder for order status %s", status.String())
	}
	return builder(ctx, order, user)
}
