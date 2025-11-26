package notification_builder

import (
	"context"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
	"fmt"
	"github.com/aws/smithy-go/ptr"
	"time"

	"github.com/google/uuid"
)

// notificationBuilders maps order status to a builder function that returns
// a PublishNotificationRequest for that status. Builders live in the service
// package so they can call the shared buildNotificationRequest helper.
var notificationOrderBuilders = map[enum.OrderStatus]func(context.Context, *model.Order, *model.User) (requests.PublishNotificationRequest, error){
	enum.OrderStatusPending:             nil,
	enum.OrderStatusPaid:                buildOrderPaidNotification,
	enum.OrderStatusRefundRequested:     buildRefundRequestedNotification,
	enum.OrderStatusRefunded:            buildRefundedNotification,
	enum.OrderStatusConfirmed:           buildOrderConfirmedNotification,
	enum.OrderStatusCancelled:           nil,
	enum.OrderStatusShipped:             buildOrderShippedNotification,
	enum.OrderStatusInTransit:           buildOrderInTransitNotification,
	enum.OrderStatusDelivered:           buildOrderDeliveredNotification,
	enum.OrderStatusReceived:            buildOrderReceivedNotification,
	enum.OrderStatusCompensateRequested: buildCompensationRequestedNotification,
	enum.OrderStatusCompensated:         nil,
	enum.OrderStatusAwaitingPickUp:      buildAwaitingPickupNotification,
}

// helper to build a user-facing order link
func orderLink(id uuid.UUID) string {
	return "https://yourdomain.com/user/orders/" + id.String()
}

// helper to build push data consistently across builders
func pushDataForOrder(order *model.Order) map[string]string {
	return map[string]string{"data": orderLink(order.ID)}
}

// common channel sets
var (
	channelEmail     = []string{"EMAIL"}
	channelEmailPush = []string{"EMAIL", "PUSH"}
	channelPush      = []string{"PUSH"}
)

func buildOrderPaidNotification(ctx context.Context, order *model.Order, actionBy *model.User) (requests.PublishNotificationRequest, error) {
	_ = ctx
	_ = actionBy
	emailSubject := "Thanks you for choosing us"
	selectedTemplate := "order_created"
	emailPayload := EmailNotificationPayload{
		EmailSubject:      ptr.String(emailSubject),
		EmailTemplateName: ptr.String(selectedTemplate),
		EmailTemplateData: nil,
		EmailHTMLBody:     nil,
	}
	pushPayload := PushNotificationPayload{
		Title: "Payment Successful",
		Body:  "Thank you for your payment. We will process your order as soon as possible.",
		Data:  pushDataForOrder(order),
	}
	return buildNotificationRequest(order.UserID, channelEmail, emailPayload, pushPayload), nil
}

func buildRefundRequestedNotification(ctx context.Context, order *model.Order, actionBy *model.User) (requests.PublishNotificationRequest, error) {
	_ = ctx
	_ = actionBy
	emailSubject := "📩 Refund Request Received"
	selectedTemplate := "refund_request_received"
	emailPayload := EmailNotificationPayload{
		EmailSubject:      ptr.String(emailSubject),
		EmailTemplateName: ptr.String(selectedTemplate),
		EmailTemplateData: map[string]interface{}{
			"RefundCode":       order.ID.String(),
			"RequestDate":      order.UpdatedAt.Format("02 Jan 2006 15:04"),
			"RefundAmount":     fmt.Sprintf("%d VND", order.TotalAmount),
			"Reason":           order.GetLatestActionNote().Reason,
			"RefundStatusLink": orderLink(order.ID),
			"Year":             time.Now().Year(),
		},
		EmailHTMLBody: nil,
	}
	pushPayload := PushNotificationPayload{
		Title: "Refund Request Received",
		Body:  "We have received your refund request and will process it shortly.",
		Data:  pushDataForOrder(order),
	}
	return buildNotificationRequest(order.UserID, channelEmailPush, emailPayload, pushPayload), nil
}

func buildRefundedNotification(ctx context.Context, order *model.Order, actionBy *model.User) (requests.PublishNotificationRequest, error) {
	_ = ctx
	_ = actionBy
	emailSubject := "💰 Your Refund Has Been Approved!"
	selectedTemplate := "refund_request_received"
	emailPayload := EmailNotificationPayload{
		EmailSubject:      ptr.String(emailSubject),
		EmailTemplateName: ptr.String(selectedTemplate),
		EmailTemplateData: map[string]interface{}{
			"CustomerName":  order.User.FullName,
			"OrderCode":     order.ID.String(),
			"RefundAmount":  fmt.Sprintf("%d VND", order.TotalAmount),
			"RefundDate":    order.UpdatedAt.Format("02 Jan 2006 15:04"),
			"PaymentMethod": "PAYOS",
			"OrderLink":     orderLink(order.ID),
			"Year":          time.Now().Year(),
		},
		EmailHTMLBody: nil,
	}
	pushPayload := PushNotificationPayload{
		Title: "Ding Ding Ding 💰... Your Refund Has Been Approved!",
		Body:  "",
		Data:  pushDataForOrder(order),
	}
	return buildNotificationRequest(order.UserID, channelEmailPush, emailPayload, pushPayload), nil
}

func buildOrderConfirmedNotification(ctx context.Context, order *model.Order, actionBy *model.User) (requests.PublishNotificationRequest, error) {
	_ = ctx
	_ = actionBy
	pushPayload := PushNotificationPayload{
		Title: "Your Order Has Been Confirm!",
		Body:  "Happy Happy Happy",
		Data:  pushDataForOrder(order),
	}
	return buildNotificationRequest(order.UserID, channelPush, EmailNotificationPayload{}, pushPayload), nil
}

func buildOrderShippedNotification(ctx context.Context, order *model.Order, actionBy *model.User) (requests.PublishNotificationRequest, error) {
	_ = ctx
	_ = actionBy
	pushPayload := PushNotificationPayload{
		Title: "Your Order is on the way!",
		Body:  "Your Order had delivered to transportation Unit!",
		Data:  pushDataForOrder(order),
	}
	return buildNotificationRequest(order.UserID, channelPush, EmailNotificationPayload{}, pushPayload), nil
}

func buildOrderInTransitNotification(ctx context.Context, order *model.Order, actionBy *model.User) (requests.PublishNotificationRequest, error) {
	_ = ctx
	_ = actionBy
	pushPayload := PushNotificationPayload{
		Title: "Your Order is almost there!",
		Body:  "Your Order will reach you soon!",
		Data:  pushDataForOrder(order),
	}
	return buildNotificationRequest(order.UserID, channelPush, EmailNotificationPayload{}, pushPayload), nil
}

func buildOrderDeliveredNotification(ctx context.Context, order *model.Order, actionBy *model.User) (requests.PublishNotificationRequest, error) {
	_ = ctx
	_ = actionBy
	pushPayload := PushNotificationPayload{
		Title: "I'm Here!",
		Body:  "The delivery person will contact you soon!",
		Data:  pushDataForOrder(order),
	}
	return buildNotificationRequest(order.UserID, channelPush, EmailNotificationPayload{}, pushPayload), nil
}

func buildOrderReceivedNotification(ctx context.Context, order *model.Order, actionBy *model.User) (requests.PublishNotificationRequest, error) {
	_ = ctx
	_ = actionBy
	pushPayload := PushNotificationPayload{
		Title: "Thanks for using our service!",
		Body:  "We hope to see you again soon.",
		Data:  pushDataForOrder(order),
	}
	return buildNotificationRequest(order.UserID, channelPush, EmailNotificationPayload{}, pushPayload), nil
}

func buildCompensationRequestedNotification(ctx context.Context, order *model.Order, actionBy *model.User) (requests.PublishNotificationRequest, error) {
	_ = ctx
	_ = actionBy
	emailSubject := "💰 Your Refund Has Been Approved!"
	selectedTemplate := "refund_request_received"
	emailPayload := EmailNotificationPayload{
		EmailSubject:      ptr.String(emailSubject),
		EmailTemplateName: ptr.String(selectedTemplate),
		EmailTemplateData: map[string]interface{}{
			"CustomerName":  order.User.FullName,
			"OrderCode":     order.ID.String(),
			"RefundAmount":  fmt.Sprintf("%d VND", order.TotalAmount),
			"RefundDate":    order.UpdatedAt.Format("02 Jan 2006 15:04"),
			"PaymentMethod": "PAYOS",
			"OrderLink":     orderLink(order.ID),
			"Year":          time.Now().Year(),
		},
		EmailHTMLBody: nil,
	}
	pushPayload := PushNotificationPayload{
		Title: "Ding Ding Ding 💰... Your Refund Has Been Approved!",
		Body:  "",
		Data:  pushDataForOrder(order),
	}
	return buildNotificationRequest(order.UserID, channelEmailPush, emailPayload, pushPayload), nil
}

func buildAwaitingPickupNotification(ctx context.Context, order *model.Order, actionBy *model.User) (requests.PublishNotificationRequest, error) {
	_ = ctx
	_ = actionBy
	pushPayload := PushNotificationPayload{
		Title: "Your Order is ready for pick-up!",
		Body:  "Please visit our store to collect your order.",
		Data:  pushDataForOrder(order),
	}
	return buildNotificationRequest(order.UserID, channelPush, EmailNotificationPayload{}, pushPayload), nil
}

// buildNotificationRequest is a helper to create the PublishNotificationRequest used by the notification service.
func buildNotificationRequest(userID uuid.UUID, channel []string, emailPayload EmailNotificationPayload, pushPayload PushNotificationPayload) requests.PublishNotificationRequest {
	return requests.PublishNotificationRequest{
		UserID:   userID,
		Channels: channel,
		//Push
		Title: pushPayload.Title,
		Body:  pushPayload.Body,
		Data:  pushPayload.Data,
		//Email
		EmailSubject:      emailPayload.EmailSubject,
		EmailTemplateName: emailPayload.EmailTemplateName,
		EmailTemplateData: emailPayload.EmailTemplateData,
		EmailHTMLBody:     emailPayload.EmailHTMLBody,
	}
}
