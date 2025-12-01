package notification_builder

import (
	"context"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/domain/model"
	"fmt"
	"time"
)

var notificationPreOrderBuilders = map[PreOrderNotificationType]func(context.Context, *model.PreOrder, *model.User) ([]requests.PublishNotificationRequest, error){
	PreOrderNotifyPending:             nil,
	PreOrderNotifyPaid:                buildPreorderPaidNotification,
	PreOrderNotifyPreOrdered:          buildPreorderPreOrderedNotification,
	PreOrderNotifyAwaitingPickup:      buildPreorderAwaitingPickupNotification,
	PreOrderNotifyInTransit:           buildPreorderInTransitNotification,
	PreOrderNotifyDelivered:           buildPreorderDeliveredNotification,
	PreOrderNotifyReceived:            buildPreorderReceivedNotification,
	PreOrderNotifyCompensateRequested: buildPreorderCompensateRequestedNotification,
	PreOrderNotifyCompensated:         buildPreorderCompensatedNotification,
	PreOrderNotifyCancelled:           buildPreorderCancelledNotification,
}

// helper to build a user-facing preorder link
func preorderLink(id interface{ String() string }) string {
	return "https://yourdomain.com/user/preorders/" + id.String()
}

// helper to build push data consistently across preorder builders
func pushDataForPreOrder(preorder *model.PreOrder) map[string]string {
	return map[string]string{"data": preorderLink(preorder.ID)}
}

// Builders
func buildPreorderPendingNotification(_ context.Context, po *model.PreOrder, _ *model.User) (requests.PublishNotificationRequest, error) {
	emailSubject := "Pre-order Request Received"
	template := "preorder_pending"
	emailPayload := EmailNotificationPayload{
		EmailSubject:      &emailSubject,
		EmailTemplateName: &template,
		EmailTemplateData: map[string]interface{}{
			"PreOrderCode": po.ID.String(),
			"ProductName":  po.ProductName,
			"Quantity":     po.Quantity,
			"CreatedAt":    po.CreatedAt.Format("02 Jan 2006 15:04"),
			"PreOrderLink": preorderLink(po.ID),
			"Year":         time.Now().Year(),
		},
	}
	pushPayload := PushNotificationPayload{
		Title: "Pre-order Received",
		Body:  fmt.Sprintf("Your pre-order for %s has been received.", po.ProductName),
		Data:  pushDataForPreOrder(po),
	}
	return buildNotificationRequest(po.UserID, channelEmailPush, emailPayload, pushPayload), nil
}

func buildPreorderPaidNotification(_ context.Context, po *model.PreOrder, _ *model.User) ([]requests.PublishNotificationRequest, error) {
	emailSubject := "Pre-order Payment Confirmed"
	template := "preorder_paid"
	emailPayload := EmailNotificationPayload{
		CustomReceiver:    &po.Email,
		EmailSubject:      &emailSubject,
		EmailTemplateName: &template,
		EmailTemplateData: map[string]interface{}{
			"PreOrderCode": po.ID.String(),
			"ProductName":  po.ProductName,
			"TotalAmount":  fmt.Sprintf("%d VND", int(po.TotalAmount)),
			"CreatedAt":    po.UpdatedAt.Format("02 Jan 2006 15:04"),
			"PreOrderLink": preorderLink(po.ID),
			"Year":         time.Now().Year(),
		},
	}
	pushPayload := PushNotificationPayload{
		Title: "Payment Successful",
		Body:  "Thank you — your pre-order payment has been received.",
		Data:  pushDataForPreOrder(po),
	}
	return []requests.PublishNotificationRequest{buildNotificationRequest(po.UserID, channelEmailPush, emailPayload, pushPayload)}, nil
}

func buildPreorderPreOrderedNotification(_ context.Context, po *model.PreOrder, _ *model.User) ([]requests.PublishNotificationRequest, error) {
	pushPayload := PushNotificationPayload{
		Title: "Your Item Has Been Pre-ordered",
		Body:  fmt.Sprintf("%s has been reserved for you.", po.ProductName),
		Data:  pushDataForPreOrder(po),
	}
	return []requests.PublishNotificationRequest{buildNotificationRequest(po.UserID, channelPush, EmailNotificationPayload{}, pushPayload)}, nil
}

func buildPreorderAwaitingPickupNotification(_ context.Context, po *model.PreOrder, _ *model.User) ([]requests.PublishNotificationRequest, error) {
	pushPayload := PushNotificationPayload{
		Title: "Your Pre-order is Ready for Pickup",
		Body:  "Please visit the pickup point to collect your item.",
		Data:  pushDataForPreOrder(po),
	}
	return []requests.PublishNotificationRequest{buildNotificationRequest(po.UserID, channelPush, EmailNotificationPayload{}, pushPayload)}, nil
}

func buildPreorderInTransitNotification(_ context.Context, po *model.PreOrder, _ *model.User) ([]requests.PublishNotificationRequest, error) {
	pushPayload := PushNotificationPayload{
		Title: "Your Pre-order is In Transit",
		Body:  "Your item is on the way.",
		Data:  pushDataForPreOrder(po),
	}
	return []requests.PublishNotificationRequest{buildNotificationRequest(po.UserID, channelPush, EmailNotificationPayload{}, pushPayload)}, nil
}

func buildPreorderDeliveredNotification(_ context.Context, po *model.PreOrder, _ *model.User) ([]requests.PublishNotificationRequest, error) {
	pushPayload := PushNotificationPayload{
		Title: "Your Pre-order Has Been Delivered",
		Body:  "The delivery has been completed.",
		Data:  pushDataForPreOrder(po),
	}
	return []requests.PublishNotificationRequest{buildNotificationRequest(po.UserID, channelPush, EmailNotificationPayload{}, pushPayload)}, nil
}

func buildPreorderReceivedNotification(_ context.Context, po *model.PreOrder, _ *model.User) ([]requests.PublishNotificationRequest, error) {
	pushPayload := PushNotificationPayload{
		Title: "Thanks — Pre-order Received",
		Body:  "We hope you enjoy your purchase.",
		Data:  pushDataForPreOrder(po),
	}
	return []requests.PublishNotificationRequest{buildNotificationRequest(po.UserID, channelPush, EmailNotificationPayload{}, pushPayload)}, nil
}

func buildPreorderCompensateRequestedNotification(_ context.Context, po *model.PreOrder, _ *model.User) ([]requests.PublishNotificationRequest, error) {
	emailSubject := "Compensation Request Received"
	template := "preorder_compensation_received"
	// guard latest action note
	var reason string
	if note := po.GetLatestActionNote(); note != nil {
		reason = note.Reason
	}
	var imageURL any
	if po.UserResource != nil && *po.UserResource != "" {
		imageURL = *po.UserResource
	} else {
		imageURL = nil
	}
	emailPayload := EmailNotificationPayload{
		EmailSubject:      &emailSubject,
		EmailTemplateName: &template,
		EmailTemplateData: map[string]interface{}{
			"PreOrderCode": po.ID.String(),
			"CustomerName": po.FullName,
			"Reason":       reason,
			"ImageURL":     imageURL,
			"PreOrderLink": preorderLink(po.ID),
			"Year":         time.Now().Year(),
		},
	}
	pushPayload := PushNotificationPayload{
		Title: "Compensation Request Received",
		Body:  "We have received your request and will process it.",
		Data:  pushDataForPreOrder(po),
	}
	return []requests.PublishNotificationRequest{buildNotificationRequest(po.UserID, channelEmailPush, emailPayload, pushPayload)}, nil
}

func buildPreorderCompensatedNotification(_ context.Context, po *model.PreOrder, _ *model.User) ([]requests.PublishNotificationRequest, error) {
	emailSubject := "Your Compensation Has Been Approved"
	template := "preorder_compensation_approved"
	var imageURL any
	if po.StaffResource != nil && *po.StaffResource != "" {
		imageURL = *po.StaffResource
	} else {
		imageURL = nil
	}
	emailPayload := EmailNotificationPayload{
		EmailSubject:      &emailSubject,
		EmailTemplateName: &template,
		EmailTemplateData: map[string]interface{}{
			"PreOrderCode": po.ID.String(),
			"ImageURL":     imageURL,
			"PreOrderLink": preorderLink(po.ID),
			"Year":         time.Now().Year(),
		},
	}
	pushPayload := PushNotificationPayload{
		Title: "Compensation Approved",
		Body:  "Your compensation request has been approved.",
		Data:  pushDataForPreOrder(po),
	}
	return []requests.PublishNotificationRequest{buildNotificationRequest(po.UserID, channelEmailPush, emailPayload, pushPayload)}, nil
}

func buildPreorderCancelledNotification(_ context.Context, po *model.PreOrder, _ *model.User) ([]requests.PublishNotificationRequest, error) {
	emailSubject := "Your Pre-order Has Been Cancelled"
	template := "preorder_cancelled"
	emailPayload := EmailNotificationPayload{
		EmailSubject:      &emailSubject,
		EmailTemplateName: &template,
		EmailTemplateData: map[string]interface{}{
			"PreOrderCode": po.ID.String(),
			"ProductName":  po.ProductName,
			"PreOrderLink": preorderLink(po.ID),
			"Year":         time.Now().Year(),
		},
	}
	pushPayload := PushNotificationPayload{
		Title: "Pre-order Cancelled",
		Body:  "Your pre-order has been cancelled.",
		Data:  pushDataForPreOrder(po),
	}
	return []requests.PublishNotificationRequest{buildNotificationRequest(po.UserID, channelEmailPush, emailPayload, pushPayload)}, nil
}
