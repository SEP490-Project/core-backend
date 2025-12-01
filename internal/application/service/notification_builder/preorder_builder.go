package notification_builder

import (
	"context"
	"core-backend/config"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
	"fmt"
	"gorm.io/gorm"
	"time"
)

var notificationPreOrderBuilders = map[PreOrderNotificationType]func(ctx context.Context, cfg config.AppConfig, db *gorm.DB, status PreOrderNotificationType, preorder *model.PreOrder, user *model.User) ([]requests.PublishNotificationRequest, error){
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

	PreOrderNotifyRefund:           buildPreorderRefundedNotification,
	PreOrderNotifyRefundRequest:    buildPreorderRefundRequestNotification,
	PreOrderNotifyObligateRefund:   buildPreorderObligateRefundedNotification,
	PreOrderNotifyCompensateDenied: buildPreorderCompensateRejectedNotification,
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

func buildPreorderPaidNotification(ctx context.Context, cfg config.AppConfig, db *gorm.DB, status PreOrderNotificationType, po *model.PreOrder, user *model.User) ([]requests.PublishNotificationRequest, error) {
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

func buildPreorderPreOrderedNotification(ctx context.Context, cfg config.AppConfig, db *gorm.DB, status PreOrderNotificationType, po *model.PreOrder, user *model.User) ([]requests.PublishNotificationRequest, error) {
	pushPayload := PushNotificationPayload{
		Title: "Your Item Has Been Pre-ordered",
		Body:  fmt.Sprintf("%s has been reserved for you.", po.ProductName),
		Data:  pushDataForPreOrder(po),
	}
	return []requests.PublishNotificationRequest{buildNotificationRequest(po.UserID, channelPush, EmailNotificationPayload{}, pushPayload)}, nil
}

func buildPreorderAwaitingPickupNotification(ctx context.Context, cfg config.AppConfig, db *gorm.DB, status PreOrderNotificationType, po *model.PreOrder, user *model.User) ([]requests.PublishNotificationRequest, error) {
	pushPayload := PushNotificationPayload{
		Title: "Your Pre-order is Ready for Pickup",
		Body:  "Please visit the pickup point to collect your item.",
		Data:  pushDataForPreOrder(po),
	}
	return []requests.PublishNotificationRequest{buildNotificationRequest(po.UserID, channelPush, EmailNotificationPayload{}, pushPayload)}, nil
}

func buildPreorderInTransitNotification(ctx context.Context, cfg config.AppConfig, db *gorm.DB, status PreOrderNotificationType, po *model.PreOrder, user *model.User) ([]requests.PublishNotificationRequest, error) {
	pushPayload := PushNotificationPayload{
		Title: "Your Pre-order is In Transit",
		Body:  "Your item is on the way.",
		Data:  pushDataForPreOrder(po),
	}
	return []requests.PublishNotificationRequest{buildNotificationRequest(po.UserID, channelPush, EmailNotificationPayload{}, pushPayload)}, nil
}

func buildPreorderDeliveredNotification(ctx context.Context, cfg config.AppConfig, db *gorm.DB, status PreOrderNotificationType, po *model.PreOrder, user *model.User) ([]requests.PublishNotificationRequest, error) {
	pushPayload := PushNotificationPayload{
		Title: "Your Pre-order Has Been Delivered",
		Body:  "The delivery has been completed.",
		Data:  pushDataForPreOrder(po),
	}
	return []requests.PublishNotificationRequest{buildNotificationRequest(po.UserID, channelPush, EmailNotificationPayload{}, pushPayload)}, nil
}

func buildPreorderReceivedNotification(ctx context.Context, cfg config.AppConfig, db *gorm.DB, status PreOrderNotificationType, po *model.PreOrder, user *model.User) ([]requests.PublishNotificationRequest, error) {
	pushPayload := PushNotificationPayload{
		Title: "Thanks — Pre-order Received",
		Body:  "We hope you enjoy your purchase.",
		Data:  pushDataForPreOrder(po),
	}
	return []requests.PublishNotificationRequest{buildNotificationRequest(po.UserID, channelPush, EmailNotificationPayload{}, pushPayload)}, nil
}

func buildPreorderCompensateRequestedNotification(ctx context.Context, cfg config.AppConfig, db *gorm.DB, status PreOrderNotificationType, po *model.PreOrder, user *model.User) ([]requests.PublishNotificationRequest, error) {
	var resp []requests.PublishNotificationRequest
	emailSubject := "Compensation Request Received"
	template := "preorder_compensation_received"
	emailPayload := EmailNotificationPayload{
		CustomReceiver:    &po.Email,
		EmailSubject:      &emailSubject,
		EmailTemplateName: &template,
		EmailTemplateData: map[string]interface{}{
			"CustomerName": po.FullName,
			"PreOrderCode": po.ID.String(),
			"Reason":       po.GetLatestActionNote().Reason,
			"ImageURL":     po.UserResource,
			"PreOrderLink": preorderLink(po.ID),
			"Year":         time.Now().Year(),
		},
	}
	pushPayload := PushNotificationPayload{
		Title: "Compensation Request Received",
		Body:  "We have received your request and will process it.",
		Data:  pushDataForPreOrder(po),
	}
	resp = append(resp, buildNotificationRequest(po.UserID, channelEmailPush, emailPayload, pushPayload))

	// Announce PUSH to all SaleStaff
	var saleStaffs []model.User
	if err := db.Where("role = ?", enum.UserRoleSalesStaff.String()).Find(&saleStaffs).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch sale staffs: %w", err)
	}
	for _, staff := range saleStaffs {
		staffPushPayload := PushNotificationPayload{
			Title: "There are new Compensation Request need to be resolve!",
			Body:  fmt.Sprintf("A new compensation request has been made for PreOrder %s.", po.ID.String()),
			Data:  pushDataForPreOrder(po),
		}
		emailSubject := "New Compensation Request"
		emailTemplateName := "compensation_staff_announcement"
		staffEmailPayload := EmailNotificationPayload{
			EmailSubject:      &(emailSubject),
			EmailTemplateName: &(emailTemplateName),
			EmailTemplateData: map[string]interface{}{
				"StaffName":       staff.FullName,
				"CustomerName":    po.FullName,
				"RequestTitle":    "PreOrder Code:",
				"RequestCode":     po.ID,
				"Reason":          po.GetLatestActionNote().Reason,
				"RequestedAmount": fmt.Sprintf("%d VND", int(po.TotalAmount)),
				"CreatedAt":       po.GetLatestActionNote().CreatedAt,
				"ImageURL":        po.UserResource,
				"ReviewURL":       "https://bshowsell.site",
				"Year":            time.Now().Year(),
			},
		}
		resp = append(resp, buildNotificationRequest(staff.ID, channelEmailPush, staffEmailPayload, staffPushPayload))
	}
	return resp, nil
}

func buildPreorderCompensateRejectedNotification(ctx context.Context, cfg config.AppConfig, db *gorm.DB, status PreOrderNotificationType, po *model.PreOrder, user *model.User) ([]requests.PublishNotificationRequest, error) {
	emailSubject := "Compensation Request Denied"
	template := "preorder_compensation_denied"
	emailPayload := EmailNotificationPayload{
		CustomReceiver:    &po.Email,
		EmailSubject:      &emailSubject,
		EmailTemplateName: &template,
		EmailTemplateData: map[string]interface{}{
			"CustomerName":    po.FullName,
			"PreOrderCode":    po.ID.String(),
			"DenialReason":    po.GetLatestActionNote().Reason,
			"RequestedAmount": po.TotalAmount,
			"Year":            time.Now().Year(),
		},
	}
	pushPayload := PushNotificationPayload{
		Title: "Compensation Request Denied",
		Body:  "Your compensation request has been denied.",
		Data:  pushDataForPreOrder(po),
	}
	return []requests.PublishNotificationRequest{buildNotificationRequest(po.UserID, channelEmailPush, emailPayload, pushPayload)}, nil
}

func buildPreorderCompensatedNotification(ctx context.Context, cfg config.AppConfig, db *gorm.DB, status PreOrderNotificationType, po *model.PreOrder, user *model.User) ([]requests.PublishNotificationRequest, error) {
	emailSubject := "Your Compensation Has Been Approved"
	template := "preorder_compensation_approved"
	var imageURL any
	if po.StaffResource != nil && *po.StaffResource != "" {
		imageURL = *po.StaffResource
	} else {
		imageURL = nil
	}
	emailPayload := EmailNotificationPayload{
		CustomReceiver:    &po.Email,
		EmailSubject:      &emailSubject,
		EmailTemplateName: &template,
		EmailTemplateData: map[string]interface{}{
			"PreOrderCode":      po.ID.String(),
			"ImageURL":          imageURL,
			"PreOrderLink":      preorderLink(po.ID),
			"BankAccount":       po.BankAccount,
			"BankName":          po.BankName,
			"BankAccountHolder": po.BankAccountHolder,
			"ApprovedAmount":    po.TotalAmount,
			"Reason":            po.GetLatestActionNote().Reason,
			"Year":              time.Now().Year(),
		},
	}
	pushPayload := PushNotificationPayload{
		Title: "Compensation Approved",
		Body:  "Your compensation request has been approved.",
		Data:  pushDataForPreOrder(po),
	}
	return []requests.PublishNotificationRequest{buildNotificationRequest(po.UserID, channelEmailPush, emailPayload, pushPayload)}, nil
}

func buildPreorderCancelledNotification(ctx context.Context, cfg config.AppConfig, db *gorm.DB, status PreOrderNotificationType, po *model.PreOrder, user *model.User) ([]requests.PublishNotificationRequest, error) {
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

func buildPreorderRefundRequestNotification(ctx context.Context, cfg config.AppConfig, db *gorm.DB, status PreOrderNotificationType, po *model.PreOrder, user *model.User) ([]requests.PublishNotificationRequest, error) {
	var resp []requests.PublishNotificationRequest
	emailSubject := "📩 Refund Request Received"
	template := "refund_preorder_request_received"
	emailPayload := EmailNotificationPayload{
		CustomReceiver:    &po.Email,
		EmailSubject:      &emailSubject,
		EmailTemplateName: &template,
		EmailTemplateData: map[string]interface{}{
			"CustomerName": po.FullName,
			"RefundCode":   po.ID.String(),
			"ProductName":  po.ProductName,
			"RequestDate":  po.UpdatedAt.Format("02 Jan 2006 15:04"),
			"RefundAmount": po.TotalAmount,
			"Reason":       po.GetLatestActionNote().Reason,
		},
	}
	pushPayload := PushNotificationPayload{
		Title: "Pre-order Received",
		Body:  fmt.Sprintf("Your pre-order for %s has been received.", po.ProductName),
		Data:  pushDataForPreOrder(po),
	}
	resp = append(resp, buildNotificationRequest(po.UserID, channelEmailPush, emailPayload, pushPayload))

	// Announce PUSH to all SaleStaff
	var saleStaffs []model.User
	if err := db.Where("role = ?", enum.UserRoleSalesStaff.String()).Find(&saleStaffs).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch sale staffs: %w", err)
	}
	for _, staff := range saleStaffs {
		staffPushPayload := PushNotificationPayload{
			Title: "New Refund Request",
			Body:  fmt.Sprintf("A new refund request has been made for Order %s.", po.ID.String()),
			Data:  pushDataForPreOrder(po),
		}
		emailSubject := "You have new refund request need to be done"
		emailTemplateName := "refund_request_announcement"
		staffEmailPayload := EmailNotificationPayload{
			CustomReceiver:    &staff.Email,
			EmailSubject:      &emailSubject,
			EmailTemplateName: &emailTemplateName,
			EmailTemplateData: map[string]interface{}{
				"StaffName":       staff.FullName,
				"CustomerName":    po.FullName,
				"OrderCode":       po.ID,
				"Reason":          po.GetLatestActionNote().Reason,
				"RequestedAmount": fmt.Sprintf("%d VND", int(po.TotalAmount)),
				"CreatedAt":       po.GetLatestActionNote().CreatedAt,
				"ReviewURL":       "https://bshowsell.site",
				"Year":            time.Now().Year(),
			},
			EmailHTMLBody: nil,
		}
		resp = append(resp, buildNotificationRequest(staff.ID, channelEmailPush, staffEmailPayload, staffPushPayload))
	}
	return resp, nil
}

func buildPreorderRefundedNotification(ctx context.Context, cfg config.AppConfig, db *gorm.DB, status PreOrderNotificationType, po *model.PreOrder, user *model.User) ([]requests.PublishNotificationRequest, error) {
	var resp []requests.PublishNotificationRequest
	emailSubject := "💰 Your Refund Has Been Approved!"
	selectedTemplate := "refund_approved"
	emailPayload := EmailNotificationPayload{
		CustomReceiver:    &po.Email,
		EmailSubject:      &emailSubject,
		EmailTemplateName: &selectedTemplate,
		EmailTemplateData: map[string]interface{}{
			"CustomerName":  po.FullName,
			"OrderCode":     po.ID.String(),
			"RefundAmount":  fmt.Sprintf("%d VND", int(po.TotalAmount)),
			"RefundDate":    po.UpdatedAt.Format("02 Jan 2006 15:04"),
			"PaymentMethod": "Bank Transfer",
			"ImageURL":      po.StaffResource,
			"OrderLink":     "https://bshowsell.site",
			"Year":          time.Now().Year(),
		},
		EmailHTMLBody: nil,
	}
	pushPayload := PushNotificationPayload{
		Title: "Ding Ding Ding 💰... Your Refund Has Been Approved!",
		Body:  "",
		Data:  pushDataForPreOrder(po),
	}
	resp = append(resp, buildNotificationRequest(po.UserID, channelEmailPush, emailPayload, pushPayload))
	return resp, nil
}

func buildPreorderObligateRefundedNotification(ctx context.Context, cfg config.AppConfig, db *gorm.DB, status PreOrderNotificationType, po *model.PreOrder, user *model.User) ([]requests.PublishNotificationRequest, error) {
	var resp []requests.PublishNotificationRequest
	emailSubject := "💰 Your PreOrder has been Refunded!"
	selectedTemplate := "preorder_refund_obligation"
	emailPayload := EmailNotificationPayload{
		CustomReceiver:    &po.Email,
		EmailSubject:      &emailSubject,
		EmailTemplateName: &selectedTemplate,
		EmailTemplateData: map[string]interface{}{
			"CustomerName":     po.FullName,
			"PreOrderCode":     po.ID.String(),
			"RefundAmount":     fmt.Sprintf("%d VND", int(po.TotalAmount)),
			"RefundMethod":     "Bank Transfer",
			"RefundedAt":       po.UpdatedAt.Format("02 Jan 2006 15:04"),
			"Reason":           po.GetLatestActionNote().Reason,
			"EvidenceImageURL": po.StaffResource,
			"Year":             time.Now().Year(),
		},
		EmailHTMLBody: nil,
	}
	pushPayload := PushNotificationPayload{
		Title: "Unfortunate, Your PreOrder has been refunded!",
		Body:  "",
		Data:  pushDataForPreOrder(po),
	}
	resp = append(resp, buildNotificationRequest(po.UserID, channelEmailPush, emailPayload, pushPayload))
	return resp, nil

}
