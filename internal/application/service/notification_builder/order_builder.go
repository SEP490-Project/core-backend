package notification_builder

import (
	"context"
	"core-backend/config"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
	"fmt"
	"github.com/aws/smithy-go/ptr"
	"gorm.io/gorm"
	"time"

	"github.com/google/uuid"
)

// notificationBuilders maps order status to a builder function that returns
// a PublishNotificationRequest for that status. Builders live in the service
// package so they can call the shared buildNotificationRequest helper.
var notificationOrderBuilders = map[OrderNotificationType]func(context.Context, config.AppConfig, *gorm.DB, *model.Order, *model.User) ([]requests.PublishNotificationRequest, error){
	OrderNotifyPending:             nil,
	OrderNotifyPaid:                buildOrderPaidNotification,
	OrderNotifyRefundRequested:     buildRefundRequestedNotification,
	OrderNotifyRefunded:            buildRefundedNotification,
	OrderNotifyConfirmed:           buildOrderConfirmedNotification,
	OrderNotifyCancelled:           nil,
	OrderNotifyShipped:             buildOrderShippedNotification,
	OrderNotifyInTransit:           buildOrderInTransitNotification,
	OrderNotifyDelivered:           buildOrderDeliveredNotification,
	OrderNotifyReceived:            buildOrderReceivedNotification,
	OrderNotifyCompensateRequested: buildCompensationRequestedNotification,
	OrderNotifyCompensated:         buildCompensatedRequestNotification,
	OrderNotifyAwaitingPickUp:      buildAwaitingPickupNotification,

	//CompensateRequest -> Delivered
	OrderNotifyCompensationDenied: buildCompensationDenied,
	OrderNotifyObligateRefund:     buildObligateRefund,
}

// helper to build a user-facing order link
func orderLink(id uuid.UUID) string {
	return "https://yourdomain.com/user/orders/" + id.String()
}

// helper to build push data consistently across builders
func pushDataForOrder(order *model.Order) map[string]string {
	return map[string]string{"data": orderLink(order.ID)}
}

func buildOrderPaidNotification(ctx context.Context, cfg config.AppConfig, db *gorm.DB, order *model.Order, actionBy *model.User) ([]requests.PublishNotificationRequest, error) {
	var resp []requests.PublishNotificationRequest
	totalFeeString := fmt.Sprintf(
		"TOTAL %d (Product Fee: %d VND + Shipping Fee: %d VND)",
		int(order.TotalAmount)+order.ShippingFee,
		int(order.TotalAmount),
		order.ShippingFee,
	)

	var shippingAddr string
	if !order.IsSelfPickedUp {
		shippingAddr = order.Street + order.WardName + order.ProvinceName + order.DistrictName + order.City
	} else {
		shippingAddr = cfg.AdminConfig.RepresentativeCompanyAddress
	}

	_ = ctx
	_ = actionBy
	emailSubject := "Thanks you for choosing us"
	selectedTemplate := "order_created"
	emailPayload := EmailNotificationPayload{
		EmailSubject:      ptr.String(emailSubject),
		EmailTemplateName: ptr.String(selectedTemplate),
		EmailTemplateData: map[string]interface{}{
			"OrderCode":       order.ID.String(),
			"OrderDate":       order.CreatedAt.Format("02 Jan 2006 15:04"),
			"TotalAmount":     totalFeeString,
			"ShippingAddress": shippingAddr,
			"PaymentMethod":   "PAYOS",
		},
		EmailHTMLBody: nil,
	}
	pushPayload := PushNotificationPayload{
		Title: "Payment Successful",
		Body:  "Thank you for your payment. We will process your order as soon as possible.",
		Data:  pushDataForOrder(order),
	}
	resp = append(resp, buildNotificationRequest(order.UserID, channelEmailPush, emailPayload, pushPayload))
	return resp, nil
}

func buildRefundRequestedNotification(ctx context.Context, _ config.AppConfig, db *gorm.DB, order *model.Order, actionBy *model.User) ([]requests.PublishNotificationRequest, error) {
	var resp []requests.PublishNotificationRequest
	emailSubject := "📩 Refund Request Received"
	selectedTemplate := "refund_request_received"
	emailPayload := EmailNotificationPayload{
		CustomReceiver:    &order.Email,
		EmailSubject:      &(emailSubject),
		EmailTemplateName: &(selectedTemplate),
		EmailTemplateData: map[string]interface{}{
			"RefundCode":   order.ID.String(),
			"RequestDate":  order.UpdatedAt.Format("02 Jan 2006 15:04"),
			"RefundAmount": fmt.Sprintf("%d VND", int(order.TotalAmount)),
			"Reason":       order.GetLatestActionNote().Reason,
			"Year":         time.Now().Year(),
		},
		EmailHTMLBody: nil,
	}
	pushPayload := PushNotificationPayload{
		Title: "Refund Request Received",
		Body:  "We have received your refund request and will process it shortly.",
		Data:  pushDataForOrder(order),
	}
	resp = append(resp, buildNotificationRequest(order.UserID, channelEmailPush, emailPayload, pushPayload))

	// Announce PUSH to all SaleStaff
	var saleStaffs []model.User
	if err := db.Where("role = ?", enum.UserRoleSalesStaff.String()).Find(&saleStaffs).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch sale staffs: %w", err)
	}
	for _, staff := range saleStaffs {
		staffPushPayload := PushNotificationPayload{
			Title: "New Refund Request",
			Body:  fmt.Sprintf("A new refund request has been made for Order %s.", order.ID.String()),
			Data:  pushDataForOrder(order),
		}
		emailSubject := "You have new refund request need to be done"
		emailTemplateName := "refund_request_announcement"
		staffEmailPayload := EmailNotificationPayload{
			EmailSubject:      &emailSubject,
			EmailTemplateName: &emailTemplateName,
			EmailTemplateData: map[string]interface{}{
				"StaffName":       staff.FullName,
				"CustomerName":    order.FullName,
				"OrderCode":       order.ID,
				"Reason":          order.GetLatestActionNote().Reason,
				"RequestedAmount": fmt.Sprintf("%d VND", int(order.TotalAmount)),
				"CreatedAt":       order.GetLatestActionNote().CreatedAt,
				"ReviewURL":       "https://bshowsell.site",
				"Year":            time.Now().Year(),
			},
			EmailHTMLBody: nil,
		}
		resp = append(resp, buildNotificationRequest(staff.ID, channelEmailPush, staffEmailPayload, staffPushPayload))
	}
	return resp, nil
}

func buildRefundedNotification(ctx context.Context, _ config.AppConfig, _ *gorm.DB, order *model.Order, _ *model.User) ([]requests.PublishNotificationRequest, error) {
	var resp []requests.PublishNotificationRequest
	emailSubject := "💰 Your Refund Has Been Approved!"
	selectedTemplate := "refund_approved"
	emailPayload := EmailNotificationPayload{
		CustomReceiver:    &order.Email,
		EmailSubject:      &emailSubject,
		EmailTemplateName: &selectedTemplate,
		EmailTemplateData: map[string]interface{}{
			"CustomerName":  order.FullName,
			"OrderCode":     order.ID.String(),
			"RefundAmount":  fmt.Sprintf("%d VND", int(order.TotalAmount)),
			"RefundDate":    order.UpdatedAt.Format("02 Jan 2006 15:04"),
			"PaymentMethod": "Bank Transfer",
			"ImageURL":      order.StaffResource,
			"OrderLink":     "https://bshowsell.site",
			"Year":          time.Now().Year(),
		},
		EmailHTMLBody: nil,
	}
	pushPayload := PushNotificationPayload{
		Title: "Ding Ding Ding 💰... Your Refund Has Been Approved!",
		Body:  "",
		Data:  pushDataForOrder(order),
	}
	resp = append(resp, buildNotificationRequest(order.UserID, channelEmailPush, emailPayload, pushPayload))
	return resp, nil
}

func buildOrderConfirmedNotification(ctx context.Context, _ config.AppConfig, _ *gorm.DB, order *model.Order, _ *model.User) ([]requests.PublishNotificationRequest, error) {
	var resp []requests.PublishNotificationRequest
	pushPayload := PushNotificationPayload{
		Title: "Your Order Has Been Confirm!",
		Body:  "Happy Happy Happy",
		Data:  pushDataForOrder(order),
	}
	resp = append(resp, buildNotificationRequest(order.UserID, channelPush, EmailNotificationPayload{}, pushPayload))
	return resp, nil
}

func buildOrderShippedNotification(_ context.Context, _ config.AppConfig, _ *gorm.DB, order *model.Order, _ *model.User) ([]requests.PublishNotificationRequest, error) {
	var resp []requests.PublishNotificationRequest
	pushPayload := PushNotificationPayload{
		Title: "Your order is on the way!",
		Body:  "Your order had delivered to transportation Unit!",
		Data:  pushDataForOrder(order),
	}
	resp = append(resp, buildNotificationRequest(order.UserID, channelPush, EmailNotificationPayload{}, pushPayload))
	return resp, nil
}

func buildOrderInTransitNotification(_ context.Context, _ config.AppConfig, _ *gorm.DB, order *model.Order, _ *model.User) ([]requests.PublishNotificationRequest, error) {
	var resp []requests.PublishNotificationRequest

	pushPayload := PushNotificationPayload{
		Title: "Your Order is on the way!",
		Body:  "Your Order will reach you soon!",
		Data:  pushDataForOrder(order),
	}
	resp = append(resp, buildNotificationRequest(order.UserID, channelPush, EmailNotificationPayload{}, pushPayload))
	return resp, nil
}

func buildOrderDeliveredNotification(_ context.Context, _ config.AppConfig, _ *gorm.DB, order *model.Order, _ *model.User) ([]requests.PublishNotificationRequest, error) {
	var resp []requests.PublishNotificationRequest

	pushPayload := PushNotificationPayload{
		Title: "Your order has arrived!",
		Body:  "The delivery person will contact you soon!",
		Data:  pushDataForOrder(order),
	}
	resp = append(resp, buildNotificationRequest(order.UserID, channelPush, EmailNotificationPayload{}, pushPayload))
	return resp, nil
}

func buildOrderReceivedNotification(_ context.Context, _ config.AppConfig, _ *gorm.DB, order *model.Order, _ *model.User) ([]requests.PublishNotificationRequest, error) {
	var resp []requests.PublishNotificationRequest

	pushPayload := PushNotificationPayload{
		Title: "Thanks for using our service!",
		Body:  "We hope to see you again soon.",
		Data:  pushDataForOrder(order),
	}

	resp = append(resp, buildNotificationRequest(order.UserID, channelPush, EmailNotificationPayload{}, pushPayload))
	return resp, nil
}

func buildCompensationRequestedNotification(_ context.Context, _ config.AppConfig, db *gorm.DB, order *model.Order, _ *model.User) ([]requests.PublishNotificationRequest, error) {
	var resp []requests.PublishNotificationRequest
	emailSubject := "📝 Compensation Request Received"
	selectedTemplate := "compensation_received"
	emailPayload := EmailNotificationPayload{
		CustomReceiver:    &order.Email,
		EmailSubject:      &(emailSubject),
		EmailTemplateName: &(selectedTemplate),
		EmailTemplateData: map[string]interface{}{
			"CustomerName":    order.FullName,
			"OrderCode":       order.ID.String(),
			"Reason":          order.GetLatestActionNote().Reason,
			"RequestedAmount": fmt.Sprintf("%d VND", int(order.TotalAmount)),
			"ImageURL":        order.UserResource,
			"Year":            time.Now().Year(),
		},
		EmailHTMLBody: nil,
	}
	pushPayload := PushNotificationPayload{
		Title: "Your Compensation Request Received",
		Body:  "",
		Data:  pushDataForOrder(order),
	}
	resp = append(resp, buildNotificationRequest(order.UserID, channelEmailPush, emailPayload, pushPayload))

	//Staff Notification
	var saleStaffs []model.User
	if err := db.Where("role = ?", enum.UserRoleSalesStaff.String()).Find(&saleStaffs).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch sale staffs: %w", err)
	}
	for _, staff := range saleStaffs {
		staffPushPayload := PushNotificationPayload{
			Title: "There are new Compensation Request need to be resolve!",
			Body:  fmt.Sprintf("A new compensation request has been made for Order %s.", order.ID.String()),
			Data:  pushDataForOrder(order),
		}
		emailSubject := "New Compensation Request"
		emailTemplateName := "compensation_staff_announcement"
		staffEmailPayload := EmailNotificationPayload{
			EmailSubject:      &(emailSubject),
			EmailTemplateName: &(emailTemplateName),
			EmailTemplateData: map[string]interface{}{
				"StaffName":       staff.FullName,
				"CustomerName":    order.FullName,
				"OrderCode":       order.ID,
				"Reason":          order.GetLatestActionNote().Reason,
				"RequestedAmount": fmt.Sprintf("%d VND", int(order.TotalAmount)),
				"CreatedAt":       order.GetLatestActionNote().CreatedAt,
				"ImageURL":        order.UserResource,
				"ReviewURL":       "https://bshowsell.site",
				"Year":            time.Now().Year(),
			},
		}
		resp = append(resp, buildNotificationRequest(staff.ID, channelEmailPush, staffEmailPayload, staffPushPayload))
	}

	return resp, nil
}

func buildCompensatedRequestNotification(_ context.Context, _ config.AppConfig, db *gorm.DB, order *model.Order, _ *model.User) ([]requests.PublishNotificationRequest, error) {
	var resp []requests.PublishNotificationRequest
	emailSubject := "💰 Your Compensation Has Been Approved!"
	selectedTemplate := "compensation_approved"
	emailPayload := EmailNotificationPayload{
		CustomReceiver:    &order.Email,
		EmailSubject:      &emailSubject,
		EmailTemplateName: &selectedTemplate,
		EmailTemplateData: map[string]interface{}{
			"CustomerName":   order.FullName,
			"OrderCode":      order.ID.String(),
			"ApprovedAmount": fmt.Sprintf("%d VND", int(order.TotalAmount)),
			"Reason":         order.GetLatestActionNote().Reason,
			"ImageURL":       order.StaffResource,
			"Year":           time.Now().Year(),
		},
		EmailHTMLBody: nil,
	}
	pushPayload := PushNotificationPayload{
		Title: "💰 Good news! Your Compensation Has Been Approved!",
		Body:  "",
		Data:  pushDataForOrder(order),
	}
	resp = append(resp, buildNotificationRequest(order.UserID, channelEmailPush, emailPayload, pushPayload))
	return resp, nil
}

func buildAwaitingPickupNotification(_ context.Context, _ config.AppConfig, _ *gorm.DB, order *model.Order, _ *model.User) ([]requests.PublishNotificationRequest, error) {
	var resp []requests.PublishNotificationRequest

	pushPayload := PushNotificationPayload{
		Title: "Your order is ready for pick-up!",
		Body:  "Please visit our store to collect your order.",
		Data:  pushDataForOrder(order),
	}
	resp = append(resp, buildNotificationRequest(order.UserID, channelPush, EmailNotificationPayload{}, pushPayload))
	return resp, nil
}

func buildCompensationDenied(_ context.Context, _ config.AppConfig, _ *gorm.DB, order *model.Order, _ *model.User) ([]requests.PublishNotificationRequest, error) {
	var resp []requests.PublishNotificationRequest

	pushPayload := PushNotificationPayload{
		Title: "Unfortunately, Your compensation request has been denied",
		Body:  "Please contact our support for more details.",
		Data:  pushDataForOrder(order),
	}

	emailSubject := "❌ Your compensation request has been denied"
	selectedTemplate := "compensation_denied"
	emailPayload := EmailNotificationPayload{
		CustomReceiver:    &order.Email,
		EmailSubject:      &emailSubject,
		EmailTemplateName: &selectedTemplate,
		EmailTemplateData: map[string]interface{}{
			"CustomerName":  order.FullName,
			"OrderCode":     order.ID.String(),
			"ApprovedAmout": fmt.Sprintf("%d VND", int(order.TotalAmount)),
			"Reason":        order.GetLatestActionNote().Reason,
			"ImageURL":      order.StaffResource,
			"Year":          time.Now().Year(),
		},
		EmailHTMLBody: nil,
	}
	resp = append(resp, buildNotificationRequest(order.UserID, channelEmailPush, emailPayload, pushPayload))
	return resp, nil
}

func buildObligateRefund(_ context.Context, _ config.AppConfig, _ *gorm.DB, order *model.Order, _ *model.User) ([]requests.PublishNotificationRequest, error) {
	var resp []requests.PublishNotificationRequest
	emailSubject := "Order Cancellation - Refund Processed"
	selectedTemplate := "refund_obligation"
	emailPayload := EmailNotificationPayload{
		CustomReceiver:    &order.Email,
		EmailSubject:      &emailSubject,
		EmailTemplateName: &selectedTemplate,
		EmailTemplateData: map[string]interface{}{
			"OrderCode":    order.ID.String(),
			"RefundAmount": fmt.Sprintf("%d VND", int(order.TotalAmount)),
			"RefundMethod": "Bank Transfer",
			"RefundedAt":   order.UpdatedAt.Format("02 Jan 2006 15:04"),
			"ImageURL":     order.StaffResource,
			"Year":         time.Now().Year(),
		},
		EmailHTMLBody: nil,
	}
	pushPayload := PushNotificationPayload{
		Title: "Your order has been cancelled",
		Body:  "We sincerely apologize for the inconvenience. We understand how frustrating this situation can be, and we are committed to making things right.",
		Data:  pushDataForOrder(order),
	}
	resp = append(resp, buildNotificationRequest(order.UserID, channelEmailPush, emailPayload, pushPayload))
	return resp, nil
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
		CustomReceiver:    emailPayload.CustomReceiver,
		EmailSubject:      emailPayload.EmailSubject,
		EmailTemplateName: emailPayload.EmailTemplateName,
		EmailTemplateData: emailPayload.EmailTemplateData,
		EmailHTMLBody:     emailPayload.EmailHTMLBody,
	}
}
