package service

import (
	"context"
	"core-backend/config"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/application/interfaces/iservice"
	"core-backend/internal/application/service/helper"
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
	gormrepository "core-backend/internal/infrastructure/gorm_repository"
	"core-backend/pkg/utils"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type coProducingRefundService struct {
	contractPaymentRepo irepository.GenericRepository[model.ContractPayment]
	contractRepo        irepository.GenericRepository[model.Contract]
	userRepo            irepository.UserRepository
	notificationService iservice.NotificationService
	unitOfWork          irepository.UnitOfWork
	appConfig           *config.AppConfig
	adminConfig         *config.AdminConfig
	db                  *gorm.DB
}

// NewCoProducingRefundService creates a new CoProducingRefundService
func NewCoProducingRefundService(
	dbReg *gormrepository.DatabaseRegistry,
	unitOfWork irepository.UnitOfWork,
	notificationService iservice.NotificationService,
	appConfig *config.AppConfig,
	adminConfig *config.AdminConfig,
	db *gorm.DB,
) iservice.CoProducingRefundService {
	return &coProducingRefundService{
		contractPaymentRepo: dbReg.ContractPaymentRepository,
		contractRepo:        dbReg.ContractRepository,
		userRepo:            dbReg.UserRepository,
		notificationService: notificationService,
		unitOfWork:          unitOfWork,
		appConfig:           appConfig,
		adminConfig:         adminConfig,
		db:                  db,
	}
}

// SubmitRefundProof allows Marketing Staff to submit proof of refund to brand
func (s *coProducingRefundService) SubmitRefundProof(
	ctx context.Context,
	req *requests.SubmitCoProducingRefundProofRequest,
	submittedBy uuid.UUID,
) (*model.ContractPayment, error) {
	zap.L().Info("CoProducingRefundService - SubmitRefundProof",
		zap.String("payment_id", req.ContractPaymentID.String()),
		zap.String("submitted_by", submittedBy.String()))

	// Get payment with contract and brand
	payment, err := s.contractPaymentRepo.GetByID(ctx, req.ContractPaymentID, []string{"Contract", "Contract.Brand"})
	if err != nil {
		return nil, fmt.Errorf("failed to get contract payment: %w", err)
	}
	if payment == nil {
		return nil, errors.New("contract payment not found")
	}

	// Validate contract type
	if payment.Contract.Type != enum.ContractTypeCoProduce {
		return nil, errors.New("refund proof submission is only for CO_PRODUCING contracts")
	}

	// Validate status
	if !payment.CanSubmitRefundProof() {
		return nil, fmt.Errorf("cannot submit refund proof in current status: %s", payment.Status)
	}

	// Check max attempts
	maxAttempts := s.adminConfig.CoProducingRefundProofMaxAttempts
	if maxAttempts == 0 {
		maxAttempts = 3
	}
	if payment.RefundAttempts >= maxAttempts {
		return nil, fmt.Errorf("maximum refund proof attempts (%d) exceeded", maxAttempts)
	}

	// Update payment with refund proof
	now := time.Now()
	err = helper.WithTransaction(ctx, s.unitOfWork, func(ctx context.Context, uow irepository.UnitOfWork) error {
		return uow.ContractPayments().UpdateByCondition(ctx, func(db *gorm.DB) *gorm.DB {
			return db.Where("id = ?", payment.ID)
		}, map[string]any{
			"status":               enum.ContractPaymentStatusKOLProofSubmitted,
			"refund_proof_url":     req.RefundProofURL,
			"refund_proof_note":    req.RefundProofNote,
			"refund_submitted_at":  &now,
			"refund_submitted_by":  submittedBy,
			"refund_attempts":      payment.RefundAttempts + 1,
			"refund_reject_reason": nil, // Clear previous rejection reason
		})
	})
	if err != nil {
		return nil, fmt.Errorf("failed to submit refund proof: %w", err)
	}

	// Update in-memory for response
	payment.Status = enum.ContractPaymentStatusKOLProofSubmitted
	payment.RefundProofURL = &req.RefundProofURL
	payment.RefundProofNote = req.RefundProofNote
	payment.RefundSubmittedAt = &now
	payment.RefundSubmittedBy = &submittedBy
	payment.RefundAttempts++
	payment.RefundRejectReason = nil

	// Send notification to brand
	s.sendRefundProofSubmittedNotification(ctx, payment)

	zap.L().Info("CoProducingRefundService - Refund proof submitted successfully",
		zap.String("payment_id", payment.ID.String()),
		zap.Int("attempt", payment.RefundAttempts))

	return payment, nil
}

// ReviewRefundProof allows Brand to approve or reject the submitted refund proof
func (s *coProducingRefundService) ReviewRefundProof(
	ctx context.Context,
	req *requests.ReviewCoProducingRefundProofRequest,
	reviewedBy uuid.UUID,
) (*model.ContractPayment, error) {
	zap.L().Info("CoProducingRefundService - ReviewRefundProof",
		zap.String("payment_id", req.ContractPaymentID.String()),
		zap.Bool("approved", req.Approved),
		zap.String("reviewed_by", reviewedBy.String()))

	// Get payment with contract and brand
	payment, err := s.contractPaymentRepo.GetByID(ctx, req.ContractPaymentID, []string{"Contract", "Contract.Brand", "Contract.Brand.User"})
	if err != nil {
		return nil, fmt.Errorf("failed to get contract payment: %w", err)
	}
	if payment == nil {
		return nil, errors.New("contract payment not found")
	}

	// Validate contract type
	if payment.Contract.Type != enum.ContractTypeCoProduce {
		return nil, errors.New("refund proof review is only for CO_PRODUCING contracts")
	}

	// Validate status
	if !payment.CanReviewRefundProof() {
		return nil, fmt.Errorf("cannot review refund proof in current status: %s", payment.Status)
	}

	// Validate reviewer is brand owner
	if payment.Contract.Brand.UserID == nil || *payment.Contract.Brand.UserID != reviewedBy {
		return nil, errors.New("only the brand owner can review refund proof")
	}

	// Determine new status
	now := time.Now()
	var newStatus enum.ContractPaymentStatus
	var rejectReason *string

	if req.Approved {
		newStatus = enum.ContractPaymentStatusKOLRefundApproved
	} else {
		// Check if max attempts reached
		maxAttempts := s.adminConfig.CoProducingRefundProofMaxAttempts
		if maxAttempts == 0 {
			maxAttempts = 3
		}
		if payment.RefundAttempts >= maxAttempts {
			// Max attempts reached, auto-approve with warning
			newStatus = enum.ContractPaymentStatusKOLRefundApproved
			zap.L().Warn("CoProducingRefundService - Max attempts reached, auto-approving",
				zap.String("payment_id", payment.ID.String()))
		} else {
			newStatus = enum.ContractPaymentStatusKOLProofRejected
			rejectReason = req.RejectReason
		}
	}

	// Update payment
	updateMap := map[string]any{
		"status":             newStatus,
		"refund_reviewed_at": &now,
		"refund_reviewed_by": reviewedBy,
	}
	if newStatus == enum.ContractPaymentStatusKOLRefundApproved {
		updateMap["paid_at"] = &now
	}
	if rejectReason != nil {
		updateMap["refund_reject_reason"] = rejectReason
	}

	err = helper.WithTransaction(ctx, s.unitOfWork, func(ctx context.Context, uow irepository.UnitOfWork) error {
		if err = uow.ContractPayments().UpdateByCondition(ctx, func(db *gorm.DB) *gorm.DB {
			return db.Where("id = ?", payment.ID)
		}, updateMap); err != nil {
			zap.L().Error("Failed to update contract payment during refund proof review", zap.Error(err))
			return err
		}
		if !req.Approved {
			return nil
		}
		// If approved, create refund transaction here (omitted for brevity)
		negativePayment := &model.PaymentTransaction{
			ReferenceID:     payment.ID,
			ReferenceType:   enum.PaymentTransactionReferenceTypeContractPayment,
			Amount:          utils.PtrOrNil(-payment.RefundAmount),
			Status:          enum.PaymentTransactionStatusRefunded,
			Method:          enum.ContractPaymentMethodBankTransfer.String(),
			TransactionDate: time.Now(),
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
			PayerID:         payment.RefundSubmittedBy,
			ReceivedByID:    utils.PtrOrNil(reviewedBy),
		}
		if err = uow.PaymentTransaction().Add(ctx, negativePayment); err != nil {
			zap.L().Error("Failed to create negative payment transaction during refund proof review", zap.Error(err))
			return err
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to review refund proof: %w", err)
	}

	// Update in-memory for response
	payment.Status = newStatus
	payment.RefundReviewedAt = &now
	payment.RefundReviewedBy = &reviewedBy
	if newStatus == enum.ContractPaymentStatusKOLRefundApproved {
		payment.PaidAt = &now
	}
	if rejectReason != nil {
		payment.RefundRejectReason = rejectReason
	}

	// Send notification
	if req.Approved || newStatus == enum.ContractPaymentStatusKOLRefundApproved {
		s.sendRefundProofApprovedNotification(ctx, payment)
	} else {
		s.sendRefundProofRejectedNotification(ctx, payment)
	}

	zap.L().Info("CoProducingRefundService - Refund proof reviewed",
		zap.String("payment_id", payment.ID.String()),
		zap.String("new_status", string(newStatus)))

	return payment, nil
}

// AutoApproveRefundProof auto-approves refund proof after review deadline
func (s *coProducingRefundService) AutoApproveRefundProof(ctx context.Context, paymentID uuid.UUID) error {
	zap.L().Info("CoProducingRefundService - AutoApproveRefundProof",
		zap.String("payment_id", paymentID.String()))

	now := time.Now()
	err := helper.WithTransaction(ctx, s.unitOfWork, func(ctx context.Context, uow irepository.UnitOfWork) error {
		return uow.ContractPayments().UpdateByCondition(ctx, func(db *gorm.DB) *gorm.DB {
			return db.Where("id = ?", paymentID).
				Where("status = ?", enum.ContractPaymentStatusKOLProofSubmitted)
		}, map[string]any{
			"status":             enum.ContractPaymentStatusKOLRefundApproved,
			"refund_reviewed_at": &now,
			"paid_at":            &now,
			// refund_reviewed_by remains nil (auto-approved)
		})
	})
	if err != nil {
		return fmt.Errorf("failed to auto-approve refund proof: %w", err)
	}

	// Get payment for notification
	payment, _ := s.contractPaymentRepo.GetByID(ctx, paymentID, []string{"Contract", "Contract.Brand"})
	if payment != nil {
		s.sendRefundProofAutoApprovedNotification(ctx, payment)
	}

	zap.L().Info("CoProducingRefundService - Refund proof auto-approved",
		zap.String("payment_id", paymentID.String()))

	return nil
}

// GetRefundPayments returns all contract payments in refund workflow for a brand
func (s *coProducingRefundService) GetRefundPayments(ctx context.Context, brandUserID uuid.UUID) ([]responses.ContractPaymentResponse, error) {
	zap.L().Info("CoProducingRefundService - GetRefundPayments",
		zap.String("brand_user_id", brandUserID.String()))

	refundStatuses := []enum.ContractPaymentStatus{
		enum.ContractPaymentStatusKOLPending,
		enum.ContractPaymentStatusKOLProofSubmitted,
		enum.ContractPaymentStatusKOLProofRejected,
		enum.ContractPaymentStatusKOLRefundApproved,
	}

	payments, _, err := s.contractPaymentRepo.GetAll(ctx, func(db *gorm.DB) *gorm.DB {
		return db.Joins("Contract").
			Joins("Contract.Brand").
			Where("Contract__Brand.user_id = ?", brandUserID).
			Where("contract_payments.status IN ?", refundStatuses)
	}, []string{"Contract", "Contract.Brand", "RefundSubmitter", "RefundReviewer"}, 100, 1)
	if err != nil {
		return nil, fmt.Errorf("failed to get refund payments: %w", err)
	}

	result := make([]responses.ContractPaymentResponse, 0, len(payments))
	mapper := responses.ContractPaymentResponse{}
	for _, p := range payments {
		if resp := mapper.ToResponse(&p); resp != nil {
			result = append(result, *resp)
		}
	}

	return result, nil
}

// GetPendingRefundProofs returns payments awaiting brand review
func (s *coProducingRefundService) GetPendingRefundProofs(ctx context.Context, submittedBefore *time.Time) ([]*model.ContractPayment, error) {
	zap.L().Info("CoProducingRefundService - GetPendingRefundProofs")

	var payments []model.ContractPayment
	query := s.db.WithContext(ctx).
		Model(&model.ContractPayment{}).
		Preload("Contract").
		Preload("Contract.Brand").
		Preload("Contract.Brand.User").
		Where("status = ?", enum.ContractPaymentStatusKOLProofSubmitted)

	if submittedBefore != nil {
		query = query.Where("refund_submitted_at < ?", submittedBefore)
	}

	if err := query.Find(&payments).Error; err != nil {
		return nil, fmt.Errorf("failed to get pending refund proofs: %w", err)
	}

	result := make([]*model.ContractPayment, len(payments))
	for i := range payments {
		result[i] = &payments[i]
	}

	return result, nil
}

// region: ============== Notification Helpers ==============

func (s *coProducingRefundService) sendRefundProofSubmittedNotification(ctx context.Context, payment *model.ContractPayment) {
	if payment.Contract == nil || payment.Contract.Brand == nil || payment.Contract.Brand.UserID == nil {
		zap.L().Warn("CoProducingRefundService - Cannot send notification: missing brand user ID")
		return
	}

	contractNumber := "N/A"
	if payment.Contract.ContractNumber != nil {
		contractNumber = *payment.Contract.ContractNumber
	}

	reviewDays := s.adminConfig.CoProducingRefundReviewDays
	if reviewDays == 0 {
		reviewDays = 7
	}

	brandName := payment.Contract.Brand.Name
	proofNote := ""
	if payment.RefundProofNote != nil {
		proofNote = *payment.RefundProofNote
	}
	proofURL := ""
	if payment.RefundProofURL != nil {
		proofURL = *payment.RefundProofURL
	}

	// Notify Brand - Email + In-App
	if _, err := s.notificationService.CreateAndPublishNotification(ctx, &requests.PublishNotificationRequest{
		UserID: *payment.Contract.Brand.UserID,
		Types:  []enum.NotificationType{enum.NotificationTypeInApp, enum.NotificationTypeEmail},
		Title:  "Refund Proof Submitted - Review Required",
		Body:   fmt.Sprintf("A refund proof has been submitted for contract %s. Please review within %d days.", contractNumber, reviewDays),
		Data: map[string]string{
			"contract_payment_id": payment.ID.String(),
			"contract_id":         payment.ContractID.String(),
			"refund_amount":       fmt.Sprintf("%.2f", payment.RefundAmount),
			"type":                "co_producing_refund_proof_submitted",
		},
		EmailTemplateName: utils.PtrOrNil("co_producing_refund_proof_submitted"),
		EmailTemplateData: map[string]any{
			"BrandName":          brandName,
			"ContractNumber":     contractNumber,
			"Currency":           "VND",
			"RefundAmount":       fmt.Sprintf("%.0f", payment.RefundAmount),
			"SubmissionDate":     time.Now().Format("02/01/2006 15:04"),
			"AttemptNumber":      payment.RefundAttempts,
			"ProofNote":          proofNote,
			"ProofURL":           proofURL,
			"ReviewDeadlineDays": reviewDays,
			"ReviewLink":         fmt.Sprintf("%s/brand/contract-payments", s.appConfig.Server.BaseFrontendURL),
			"CurrentYear":        time.Now().Year(),
		},
	}); err != nil {
		zap.L().Error("Failed to send submitted notification to brand", zap.Error(err))
	}

	zap.L().Info("CoProducingRefundService - Refund proof submitted notification sent to brand",
		zap.String("payment_id", payment.ID.String()),
		zap.String("brand_user_id", payment.Contract.Brand.UserID.String()))
}

func (s *coProducingRefundService) sendRefundProofApprovedNotification(ctx context.Context, payment *model.ContractPayment) {
	contractNumber := "N/A"
	if payment.Contract != nil && payment.Contract.ContractNumber != nil {
		contractNumber = *payment.Contract.ContractNumber
	}

	brandName := "N/A"
	if payment.Contract != nil && payment.Contract.Brand != nil {
		brandName = payment.Contract.Brand.Name
	}

	reviewedBy := ""
	if payment.RefundReviewedBy != nil {
		reviewer, err := s.userRepo.GetByID(ctx, *payment.RefundReviewedBy, nil)
		if err == nil && reviewer != nil {
			reviewedBy = reviewer.FullName
		}
	}

	// Notify Marketing Staff - Email + In-App
	if payment.RefundSubmittedBy == nil {
		zap.L().Warn("CoProducingRefundService - Cannot send notification: missing refund submitted by user ID")
		return
	}

	if _, err := s.notificationService.CreateAndPublishNotification(ctx, &requests.PublishNotificationRequest{
		UserID: *payment.RefundSubmittedBy,
		Title:  "Refund Proof Approved",
		Body:   fmt.Sprintf("The refund proof for contract %s has been approved by the brand.", contractNumber),
		Data: map[string]string{
			"contract_payment_id": payment.ID.String(),
			"contract_id":         payment.ContractID.String(),
			"type":                "co_producing_refund_proof_approved",
		},
		Types:             []enum.NotificationType{enum.NotificationTypeInApp, enum.NotificationTypeEmail},
		EmailTemplateName: utils.PtrOrNil("co_producing_refund_proof_approved"),
		EmailTemplateData: map[string]any{
			"ContractNumber": contractNumber,
			"BrandName":      brandName,
			"Currency":       "VND",
			"RefundAmount":   fmt.Sprintf("%.0f", payment.RefundAmount),
			"ApprovedDate":   time.Now().Format("02/01/2006 15:04"),
			"ReviewedBy":     reviewedBy,
			"CurrentYear":    time.Now().Year(),
		},
	}); err != nil {
		zap.L().Error("Failed to send approved notification to marketing staff", zap.Error(err))
	}

	zap.L().Info("CoProducingRefundService - Refund proof approved notification sent to marketing staff",
		zap.String("payment_id", payment.ID.String()))
}

func (s *coProducingRefundService) sendRefundProofRejectedNotification(ctx context.Context, payment *model.ContractPayment) {
	contractNumber := "N/A"
	if payment.Contract != nil && payment.Contract.ContractNumber != nil {
		contractNumber = *payment.Contract.ContractNumber
	}

	brandName := "N/A"
	if payment.Contract != nil && payment.Contract.Brand != nil {
		brandName = payment.Contract.Brand.Name
	}

	rejectReason := "No reason provided"
	if payment.RefundRejectReason != nil && *payment.RefundRejectReason != "" {
		rejectReason = *payment.RefundRejectReason
	}

	maxAttempts := s.adminConfig.CoProducingRefundProofMaxAttempts
	if maxAttempts == 0 {
		maxAttempts = 3
	}
	remainingAttempts := maxAttempts - payment.RefundAttempts

	// Notify Marketing Staff - Email + In-App
	if payment.RefundSubmittedBy == nil {
		zap.L().Warn("CoProducingRefundService - Cannot send notification: missing refund submitted by user ID")
		return
	}
	s.notificationService.CreateAndPublishNotification(ctx, &requests.PublishNotificationRequest{
		UserID: *payment.RefundSubmittedBy,
		Title:  "Refund Proof Rejected",
		Body:   fmt.Sprintf("The refund proof for contract %s has been rejected. Reason: %s. Remaining attempts: %d", contractNumber, rejectReason, remainingAttempts),
		Data: map[string]string{
			"contract_payment_id": payment.ID.String(),
			"contract_id":         payment.ContractID.String(),
			"type":                "co_producing_refund_proof_rejected",
		},
		Types:             []enum.NotificationType{enum.NotificationTypeInApp, enum.NotificationTypeEmail},
		EmailTemplateName: utils.PtrOrNil("co_producing_refund_proof_rejected"),
		EmailTemplateData: map[string]any{
			"ContractNumber":    contractNumber,
			"BrandName":         brandName,
			"Currency":          "VND",
			"RefundAmount":      fmt.Sprintf("%.0f", payment.RefundAmount),
			"RejectReason":      rejectReason,
			"RejectedDate":      time.Now().Format("02/01/2006 15:04"),
			"RemainingAttempts": remainingAttempts,
			"ResubmitLink":      fmt.Sprintf("%s/marketing/contract-payments", s.appConfig.Server.BaseFrontendURL),
			"CurrentYear":       time.Now().Year(),
		},
	})

	zap.L().Info("CoProducingRefundService - Refund proof rejected notification sent to marketing staff",
		zap.String("payment_id", payment.ID.String()),
		zap.String("reason", rejectReason),
		zap.Int("remaining_attempts", remainingAttempts))
}

func (s *coProducingRefundService) sendRefundProofAutoApprovedNotification(ctx context.Context, payment *model.ContractPayment) {
	contractNumber := "N/A"
	if payment.Contract != nil && payment.Contract.ContractNumber != nil {
		contractNumber = *payment.Contract.ContractNumber
	}

	brandName := "N/A"
	if payment.Contract != nil && payment.Contract.Brand != nil {
		brandName = payment.Contract.Brand.Name
	}

	reviewDays := s.adminConfig.CoProducingRefundReviewDays
	if reviewDays == 0 {
		reviewDays = 7
	}

	submissionDate := "N/A"
	if payment.RefundSubmittedAt != nil {
		submissionDate = payment.RefundSubmittedAt.Format("02/01/2006 15:04")
	}

	// Notify Brand - In-App + Email
	if payment.Contract != nil && payment.Contract.Brand != nil && payment.Contract.Brand.UserID != nil {
		_, err := s.notificationService.CreateAndPublishNotification(ctx, &requests.PublishNotificationRequest{
			UserID: *payment.Contract.Brand.UserID,
			Types:  []enum.NotificationType{enum.NotificationTypeInApp, enum.NotificationTypeEmail},
			Title:  "Refund Proof Auto-Approved",
			Body:   fmt.Sprintf("Refund proof for contract %s was auto-approved after %d-day review deadline.", contractNumber, reviewDays),
			Data: map[string]string{
				"contract_payment_id": payment.ID.String(),
				"contract_id":         payment.ContractID.String(),
				"type":                "co_producing_refund_auto_approved",
			},
			EmailTemplateName: utils.PtrOrNil("co_producing_refund_auto_approved"),
			EmailTemplateData: map[string]any{
				"RecipientName":      brandName,
				"ContractNumber":     contractNumber,
				"BrandName":          brandName,
				"Currency":           "VND",
				"RefundAmount":       fmt.Sprintf("%.0f", payment.RefundAmount),
				"ReviewDeadlineDays": reviewDays,
				"SubmissionDate":     submissionDate,
				"ApprovedDate":       time.Now().Format("02/01/2006 15:04"),
				"CurrentYear":        time.Now().Year(),
			},
		})
		if err != nil {
			zap.L().Error("Failed to send auto approved notification to brand", zap.Error(err))
		}
	}

	// Also notify Marketing Staff
	if _, err := s.notificationService.CreateAndPublishNotification(ctx, &requests.PublishNotificationRequest{
		UserID: *payment.RefundSubmittedBy,
		Title:  "Refund Proof Auto-Approved",
		Body:   fmt.Sprintf("Refund proof for contract %s with brand %s was auto-approved after review deadline.", contractNumber, brandName),
		Data: map[string]string{
			"contract_payment_id": payment.ID.String(),
			"contract_id":         payment.ContractID.String(),
			"type":                "co_producing_refund_auto_approved",
		},
		Types:             []enum.NotificationType{enum.NotificationTypeInApp, enum.NotificationTypeEmail},
		EmailTemplateName: utils.PtrOrNil("co_producing_refund_auto_approved"),
		EmailTemplateData: map[string]any{
			"RecipientName":      "Marketing Staff",
			"ContractNumber":     contractNumber,
			"BrandName":          brandName,
			"Currency":           "VND",
			"RefundAmount":       fmt.Sprintf("%.0f", payment.RefundAmount),
			"ReviewDeadlineDays": reviewDays,
			"SubmissionDate":     submissionDate,
			"ApprovedDate":       time.Now().Format("02/01/2006 15:04"),
			"CurrentYear":        time.Now().Year(),
		},
	}); err != nil {
		zap.L().Error("Failed to send auto approved notification to marketing staff", zap.Error(err))
	}

	zap.L().Info("CoProducingRefundService - Refund proof auto-approved notification sent",
		zap.String("payment_id", payment.ID.String()))
}

// endregion
