package notification_builder

import (
	"context"
	"core-backend/config"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/domain/model"
	"fmt"
	"gorm.io/gorm"
)

// common channel sets
var (
	channelEmail     = []string{"EMAIL"}
	channelEmailPush = []string{"EMAIL", "PUSH"}
	channelPush      = []string{"PUSH"}
)

// Notification payload helper types used by builders
type EmailNotificationPayload struct {
	CustomReceiver    *string
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

// ------- Order Notification Builder Factory -------//
type OrderNotificationType string

const (
	OrderNotifyPending             OrderNotificationType = "PENDING"
	OrderNotifyPaid                OrderNotificationType = "PAID"
	OrderNotifyRefundRequested     OrderNotificationType = "REFUND_REQUEST"
	OrderNotifyRefunded            OrderNotificationType = "REFUNDED"
	OrderNotifyConfirmed           OrderNotificationType = "CONFIRMED"
	OrderNotifyCancelled           OrderNotificationType = "CANCELLED"
	OrderNotifyShipped             OrderNotificationType = "SHIPPED"
	OrderNotifyInTransit           OrderNotificationType = "IN_TRANSIT"
	OrderNotifyDelivered           OrderNotificationType = "DELIVERED"
	OrderNotifyReceived            OrderNotificationType = "RECEIVED"
	OrderNotifyCompensateRequested OrderNotificationType = "COMPENSATE_REQUEST"
	OrderNotifyCompensated         OrderNotificationType = "COMPENSATED"
	OrderNotifyAwaitingPickUp      OrderNotificationType = "AWAITING_PICKUP"

	OrderNotifyCompensationDenied OrderNotificationType = "COMPENSATE_REJECTED"
	OrderNotifyObligateRefund     OrderNotificationType = "OBLIGATE_REFUND"
)

func (status OrderNotificationType) String() string {
	return string(status)
}

func BuildOrderNotifications(
	ctx context.Context,
	cfg config.AppConfig,
	db *gorm.DB,
	status OrderNotificationType,
	order *model.Order,
	user *model.User,
) ([]requests.PublishNotificationRequest, error) {
	builder, exists := notificationOrderBuilders[status]
	if !exists || builder == nil {
		return []requests.PublishNotificationRequest{}, fmt.Errorf("notification_builder: no builder for order status %s", status.String())
	}
	return builder(ctx, cfg, db, order, user)
}

//------- Payment Notification Builder Factory -------//

// ------- PreOrder Notification Builder Factory -------//
type PreOrderNotificationType string

const (
	PreOrderNotifyPending             PreOrderNotificationType = "PENDING"
	PreOrderNotifyPaid                PreOrderNotificationType = "PAID"
	PreOrderNotifyCancelled           PreOrderNotificationType = "CANCELLED"
	PreOrderNotifyPreOrdered          PreOrderNotificationType = "PRE_ORDERED"
	PreOrderNotifyAwaitingPickup      PreOrderNotificationType = "AWAITING_PICKUP"
	PreOrderNotifyInTransit           PreOrderNotificationType = "IN_TRANSIT"
	PreOrderNotifyDelivered           PreOrderNotificationType = "DELIVERED"
	PreOrderNotifyCompensateRequested PreOrderNotificationType = "COMPENSATE_REQUEST"
	PreOrderNotifyCompensated         PreOrderNotificationType = "COMPENSATED"
	PreOrderNotifyReceived            PreOrderNotificationType = "RECEIVED"

	PreOrderNotifyRefund         PreOrderNotificationType = "REFUND"
	PreOrderNotifyRefundRequest  PreOrderNotificationType = "REFUND_REQUEST"
	PreOrderNotifyObligateRefund PreOrderNotificationType = "OBLIGATE_REFUND"
)

func (status PreOrderNotificationType) String() string {
	return string(status)
}

func BuildPreOrderNotifications(ctx context.Context, cfg config.AppConfig, db *gorm.DB, status PreOrderNotificationType, preorder *model.PreOrder, user *model.User) ([]requests.PublishNotificationRequest, error) {
	builder, exists := notificationPreOrderBuilders[status]
	if !exists || builder == nil {
		return []requests.PublishNotificationRequest{}, fmt.Errorf("notification_builder: no builder for preorder status %s", status.String())
	}
	return builder(ctx, cfg, db, status, preorder, user)
}

//------- Payment Notification Builder Factory -------//
