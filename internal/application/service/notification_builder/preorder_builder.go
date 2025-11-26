package notification_builder

import (
	"context"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
)

var notificationPreOrderBuilders = map[enum.PreOrderStatus]func(context.Context, *model.PreOrder, *model.User) (requests.PublishNotificationRequest, error){
	enum.PreOrderStatusPending: nil,
	enum.PreOrderStatusPaid:    nil,
}
