package service

import (
	"context"
	"core-backend/config"
	"core-backend/internal/application/dto/dtos"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/application/interfaces/iservice"
	"core-backend/internal/application/service/helper"
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
	gormrepository "core-backend/internal/infrastructure/gorm_repository"
	"core-backend/pkg/logging"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"core-backend/pkg/utils"
)

type violationService struct {
	contractViolationRepo     irepository.ContractViolationRepository
	contractRepo              irepository.ContractRepository
	contractPaymentRepo       irepository.ContractPaymentRepository
	campaignRepo              irepository.GenericRepository[model.Campaign]
	milestoneRepo             irepository.GenericRepository[model.Milestone]
	brandRepo                 irepository.GenericRepository[model.Brand]
	userRepo                  irepository.UserRepository
	db                        *gorm.DB
	config                    *config.AppConfig
	paymentTransactionService iservice.PaymentTransactionService
	unitOfWork                irepository.UnitOfWork
	notificationService       iservice.NotificationService
	stateTransferService      iservice.StateTransferService
}

// NewViolationService creates a new ViolationService
func NewViolationService(
	dbReg *gormrepository.DatabaseRegistry,
	db *gorm.DB,
	config *config.AppConfig,
	paymentTransactionService iservice.PaymentTransactionService,
	unitOfWork irepository.UnitOfWork,
	notificationService iservice.NotificationService,
	stateTransferService iservice.StateTransferService,
) iservice.ViolationService {
	return &violationService{
		contractViolationRepo:     dbReg.ContractViolationRepository,
		contractRepo:              dbReg.ContractRepository,
		contractPaymentRepo:       dbReg.ContractPaymentRepository,
		campaignRepo:              dbReg.CampaignRepository,
		milestoneRepo:             dbReg.MilestoneRepository,
		brandRepo:                 dbReg.BrandRepository,
		userRepo:                  dbReg.UserRepository,
		db:                        db,
		config:                    config,
		paymentTransactionService: paymentTransactionService,
		unitOfWork:                unitOfWork,
		notificationService:       notificationService,
		stateTransferService:      stateTransferService,
	}
}

// InitiateBrandViolation creates a brand violation record with calculated penalty amounts
func (s *violationService) InitiateBrandViolation(
	ctx context.Context,
	contractID uuid.UUID,
	reportedBy uuid.UUID,
	reason string,
) (*model.ContractViolation, error) {
	requestID := logging.GetRequestID()
	zap.L().Info("ViolationService - Initiating brand violation",
		zap.String("request_id", requestID),
		zap.String("contract_id", contractID.String()))

	// Get contract with payments and brand
	contract, err := s.contractRepo.GetByID(ctx, contractID, []string{"ContractPayments", "Brand"})
	if err != nil {
		return nil, fmt.Errorf("failed to get contract: %w", err)
	}

	if contract.Status != enum.ContractStatusActive {
		return nil, errors.New("contract must be ACTIVE to initiate violation")
	}

	// Check for existing active violation
	existing, err := s.contractViolationRepo.FindActiveByContractID(ctx, contractID)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing violation: %w", err)
	}
	if existing != nil {
		return nil, errors.New("contract already has an active violation")
	}

	// Calculate brand penalty
	calculation, err := s.calculateBrandPenaltyInternal(ctx, contract)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate penalty: %w", err)
	}

	// Get campaign ID if exists
	var campaignID *uuid.UUID
	campaigns, _, err := s.campaignRepo.GetAll(ctx, func(db *gorm.DB) *gorm.DB {
		return db.Where("contract_id = ?", contractID)
	}, nil, 1, 1)
	if err == nil && len(campaigns) > 0 {
		campaignID = &campaigns[0].ID
	}

	// Create breakdown data for audit
	breakdownData := model.CalculationBreakdownData{
		ContractTotalValue:     calculation.ContractTotalValue,
		PenaltyPercentage:      0, // Brand violation uses fixed amount
		CompletedMilestoneIDs:  []string{},
		IncompleteMilestoneIDs: []string{},
		PaidPaymentIDs:         []string{},
		PendingPaymentIDs:      []string{},
		CalculationFormula:     calculation.CalculationFormula,
		CalculatedAt:           time.Now(),
	}

	// Populate payment IDs
	for _, p := range calculation.PaymentDetails {
		if p.Status == string(enum.ContractPaymentStatusPaid) {
			breakdownData.PaidPaymentIDs = append(breakdownData.PaidPaymentIDs, p.ID.String())
		} else {
			breakdownData.PendingPaymentIDs = append(breakdownData.PendingPaymentIDs, p.ID.String())
		}
	}

	breakdownJSON, _ := json.Marshal(breakdownData)

	violation := &model.ContractViolation{
		ContractID:           contractID,
		CampaignID:           campaignID,
		Type:                 enum.ViolationTypeBrand,
		Reason:               reason,
		PenaltyAmount:        calculation.PenaltyAmount,
		RefundAmount:         0,
		TotalPaidByBrand:     calculation.TotalPaidByBrand,
		CompletedMilestones:  calculation.CompletedMilestones,
		TotalMilestones:      calculation.TotalMilestones,
		CalculationBreakdown: breakdownJSON,
	}
	if reportedBy != uuid.Nil {
		violation.CreatedBy = &reportedBy
	}

	if err = helper.WithTransaction(ctx, s.unitOfWork, func(ctx context.Context, uow irepository.UnitOfWork) error {
		if err = uow.ContractViolations().Add(ctx, violation); err != nil {
			return fmt.Errorf("failed to create violation record: %w", err)
		}

		// Transition contract state to BRAND_VIOLATED
		if err = s.stateTransferService.MoveContractToState(ctx, uow, contractID, enum.ContractStatusBrandViolated, reportedBy); err != nil {
			return fmt.Errorf("failed to move contract to BRAND_VIOLATED status: %w", err)
		}

		// Transition contract state to BRAND_PENALTY_PENDING
		if err = s.stateTransferService.MoveContractToState(ctx, uow, contractID, enum.ContractStatusBrandPenaltyPending, reportedBy); err != nil {
			return fmt.Errorf("failed to move contract to BRAND_PENALTY_PENDING status: %w", err)
		}
		return nil
	}); err != nil {
		zap.L().Error("ViolationService - Failed to initiate brand violation", zap.Error(err))
		return nil, fmt.Errorf("failed to initiate brand violation: %w", err)
	}

	zap.L().Info("ViolationService - Brand violation created and status updated",
		zap.String("request_id", requestID),
		zap.String("violation_id", violation.ID.String()),
		zap.Float64("penalty_amount", violation.PenaltyAmount))

	// Send notification
	contractNumber := "N/A"
	if contract.ContractNumber != nil {
		contractNumber = *contract.ContractNumber
	}
	brandName := "Brand"
	if contract.Brand != nil {
		brandName = contract.Brand.Name
	}

	s.sendNotification(ctx, contractID, "Contract Violation Notice",
		fmt.Sprintf("A violation has been reported for contract %s. Reason: %s", contractNumber, reason),
		reason,
		utils.PtrOrNil("brand_violation_notice"),
		map[string]any{
			"BrandName":           brandName,
			"ContractNumber":      contractNumber,
			"ViolationReason":     reason,
			"PenaltyAmount":       fmt.Sprintf("%.2f", violation.PenaltyAmount),
			"CalculationFormula":  calculation.CalculationFormula,
			"PaymentDeadlineDays": s.config.AdminConfig.ViolationPaymentDeadlineDays,
			"PaymentLink":         fmt.Sprintf("%s/manage/brand/contracts/%s", s.config.Server.BaseFrontendURL, contractID.String()),
			"SupportLink":         s.config.Server.BaseFrontendURL + "/support",
			"CurrentYear":         time.Now().Year(),
			"Currency":            "VND",
		},
		violation.ID,
		enum.ViolationTypeBrand,
		nil,
	)

	return violation, nil
}

// InitiateKOLViolation creates a KOL violation record with calculated refund amounts
func (s *violationService) InitiateKOLViolation(
	ctx context.Context,
	contractID uuid.UUID,
	reportedBy uuid.UUID,
	reason string,
) (*model.ContractViolation, error) {
	requestID := logging.GetRequestID()
	zap.L().Info("ViolationService - Initiating KOL violation",
		zap.String("request_id", requestID),
		zap.String("contract_id", contractID.String()))

	// Get contract along with Brand
	contract, err := s.contractRepo.GetByID(ctx, contractID, []string{"ContractPayments", "Brand"})
	if err != nil {
		return nil, fmt.Errorf("failed to get contract: %w", err)
	}

	if contract.Status != enum.ContractStatusActive {
		return nil, errors.New("contract must be ACTIVE to initiate violation")
	}

	// Check for existing active violation
	existing, err := s.contractViolationRepo.FindActiveByContractID(ctx, contractID)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing violation: %w", err)
	}
	if existing != nil {
		return nil, errors.New("contract already has an active violation")
	}

	// Calculate KOL refund
	calculation, err := s.calculateKOLRefundInternal(ctx, contract)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate refund: %w", err)
	}

	// Get campaign ID and Name if exists
	var campaignID *uuid.UUID
	var campaignName string
	campaigns, _, err := s.campaignRepo.GetAll(ctx, func(db *gorm.DB) *gorm.DB {
		return db.Where("contract_id = ?", contractID)
	}, nil, 1, 1)
	if err == nil && len(campaigns) > 0 {
		campaignID = &campaigns[0].ID
		campaignName = campaigns[0].Name
	}

	// Create breakdown data for audit
	breakdownData := model.CalculationBreakdownData{
		ContractTotalValue:     calculation.ContractTotalValue,
		PenaltyPercentage:      calculation.PenaltyPercentage,
		CompletedMilestoneIDs:  []string{},
		IncompleteMilestoneIDs: []string{},
		PaidPaymentIDs:         []string{},
		PendingPaymentIDs:      []string{},
		CalculationFormula:     calculation.CalculationFormula,
		CalculatedAt:           time.Now(),
	}

	breakdownJSON, _ := json.Marshal(breakdownData)

	// Initialize proof status as PENDING
	proofStatus := enum.ViolationProofStatusPending

	violation := &model.ContractViolation{
		ContractID:           contractID,
		CampaignID:           campaignID,
		Type:                 enum.ViolationTypeKOL,
		Reason:               reason,
		PenaltyAmount:        calculation.ContractTotalValue * (calculation.PenaltyPercentage / 100),
		RefundAmount:         calculation.RefundAmount,
		TotalPaidByBrand:     calculation.TotalPaidByBrand,
		CompletedMilestones:  calculation.CompletedMilestones,
		TotalMilestones:      calculation.TotalMilestones,
		CalculationBreakdown: breakdownJSON,
		ProofStatus:          &proofStatus,
		CreatedBy:            &reportedBy,
	}

	if err = helper.WithTransaction(ctx, s.unitOfWork, func(ctx context.Context, uow irepository.UnitOfWork) error {
		if err = uow.ContractViolations().Add(ctx, violation); err != nil {
			return fmt.Errorf("failed to create violation record: %w", err)
		}

		// Transition contract state to KOL_VIOLATED
		if err = s.stateTransferService.MoveContractToState(ctx, uow, contractID, enum.ContractStatusKOLViolated, reportedBy); err != nil {
			return fmt.Errorf("failed to move contract to KOL_VIOLATED status: %w", err)
		}

		// Transition contract state to KOL_REFUND_PENDING
		if err = s.stateTransferService.MoveContractToState(ctx, uow, contractID, enum.ContractStatusKOLRefundPending, reportedBy); err != nil {
			return fmt.Errorf("failed to move contract to KOL_REFUND_PENDING status: %w", err)
		}

		return nil
	}); err != nil {
		zap.L().Error("ViolationService - Failed to initiate KOL violation", zap.Error(err))
		return nil, fmt.Errorf("failed to initiate KOL violation: %w", err)
	}

	zap.L().Info("ViolationService - KOL violation created and status updated",
		zap.String("request_id", requestID),
		zap.String("violation_id", violation.ID.String()),
		zap.Float64("refund_amount", violation.RefundAmount))

	// Get Reporter Name
	var reportedByName string
	reporter, err := s.userRepo.GetByID(ctx, reportedBy, nil)
	if err == nil && reporter != nil {
		reportedByName = reporter.FullName
	}

	contractNumber := "N/A"
	if contract.ContractNumber != nil {
		contractNumber = *contract.ContractNumber
	}
	brandName := "Brand"
	if contract.Brand != nil {
		brandName = contract.Brand.Name
	}
	kolName := utils.DerefPtr(contract.RepresentativeName, "KOL Partner")

	// Send notification to all Marketing Staff
	s.notificationService.BroadcastToRoleWithRequest(ctx, []enum.UserRole{enum.UserRoleMarketingStaff}, &requests.PublishNotificationRequest{
		Title: "Contract Violation Reported",
		Body:  fmt.Sprintf("A violation has been reported for contract. Reason: %s", reason),
		Data: map[string]string{
			"violation_id": violation.ID.String(),
			"reference_id": violation.ID.String(),
		},
		Types:             []enum.NotificationType{enum.NotificationTypeInApp},
		EmailTemplateName: utils.PtrOrNil("kol_violation_reported"),
		EmailTemplateData: map[string]any{
			"ContractNumber":  contractNumber,
			"BrandName":       brandName,
			"KOLName":         kolName,
			"CampaignName":    campaignName,
			"ReportedBy":      reportedByName,
			"ReportDate":      time.Now().Format("2006-01-02"),
			"ViolationReason": reason,
			"Currency":        "VND",
			"RefundAmount":    fmt.Sprintf("%.2f", violation.RefundAmount),
		},
	})

	// Send notification to Brand
	s.sendNotification(ctx, contractID, "KOL Violation Notice",
		fmt.Sprintf("A violation by the KOL has been reported. Reason: %s", reason),
		reason,
		utils.PtrOrNil("kol_violation_notice"),
		map[string]any{
			"Reason":       reason,
			"RefundAmount": fmt.Sprintf("%.2f", violation.RefundAmount),
			"TotalAmount":  fmt.Sprintf("%.2f", violation.PenaltyAmount+violation.RefundAmount),
		},
		violation.ID,
		enum.ViolationTypeBrand,
		nil,
	)

	return violation, nil
}

// sendNotification sends notification to brand or KOL based on violation type
func (s *violationService) sendNotification(
	ctx context.Context,
	contractID uuid.UUID,
	title, body string,
	reason string,
	templateName *string,
	templateData map[string]any,
	violationID uuid.UUID,
	violationType enum.ViolationType,
	toUserID *uuid.UUID,
) {
	if err := utils.RunWithRetry(ctx, utils.DefaultRetryOptions, func(ctx context.Context) error {
		contractWithDetails, err := s.contractRepo.GetByID(ctx, contractID, []string{"Brand.User"})
		if err != nil {
			zap.L().Error("Failed to fetch contract for notification", zap.Error(err))
			return err
		}

		var targetUserID uuid.UUID
		var recipientName string

		if violationType == enum.ViolationTypeBrand {
			// Notify Brand
			if contractWithDetails.Brand != nil && contractWithDetails.Brand.UserID != nil {
				targetUserID = *contractWithDetails.Brand.UserID
				recipientName = contractWithDetails.Brand.Name
			}
		} else if violationType == enum.ViolationTypeKOL && toUserID != nil && *toUserID != uuid.Nil {
			// Notify KOL
			targetUserID = *toUserID
			user, err := s.userRepo.GetByID(ctx, targetUserID, nil)
			if err == nil && user != nil {
				recipientName = user.FullName
			} else {
				zap.L().Warn("Could not find KOL user by representative email",
					zap.String("email", *contractWithDetails.RepresentativeEmail),
					zap.Error(err))
			}
		}

		if targetUserID == uuid.Nil {
			zap.L().Warn("Target user for violation notification not found", zap.String("contract_id", contractID.String()))
			return nil
		}

		contractNumber := "N/A"
		if contractWithDetails.ContractNumber != nil {
			contractNumber = *contractWithDetails.ContractNumber
		}

		// Update template data with common fields
		if templateData == nil {
			templateData = make(map[string]any)
		}
		templateData["BrandName"] = recipientName // Or generic Name
		templateData["RecipientName"] = recipientName
		templateData["ContractNumber"] = contractNumber
		templateData["ViolationType"] = string(violationType)
		templateData["SupportLink"] = s.config.Server.BaseFrontendURL + "/support"
		templateData["CurrentYear"] = time.Now().Year()
		templateData["Reason"] = reason

		req := requests.PublishNotificationRequest{
			UserID:            targetUserID,
			Title:             title,
			Body:              fmt.Sprintf("%s. Contract: %s. Reason: %s", body, contractNumber, reason),
			Data:              map[string]string{"violation_id": violationID.String()},
			Types:             []enum.NotificationType{enum.NotificationTypeEmail, enum.NotificationTypeInApp},
			EmailTemplateName: templateName,
			EmailTemplateData: templateData,
		}
		s.notificationService.CreateAndPublishNotification(ctx, &req)
		return nil
	}); err != nil {
		zap.L().Error("ViolationService - Failed to send notification", zap.Error(err))
	}
}

// CreatePenaltyPayment creates a PayOS payment link for brand penalty
func (s *violationService) CreatePenaltyPayment(
	ctx context.Context,
	userID uuid.UUID,
	request *requests.CreatePenaltyPaymentRequest,
) (*responses.PayOSLinkResponse, error) {
	requestID := logging.GetRequestID()
	zap.L().Info("ViolationService - Creating penalty payment",
		zap.String("request_id", requestID),
		zap.Any("request", request))

	// Get violation record
	violation, err := s.contractViolationRepo.GetByID(ctx, *request.ViolationID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get violation: %w", err)
	}

	if violation.Type != enum.ViolationTypeBrand {
		return nil, errors.New("penalty payment is only for brand violations")
	}

	if violation.IsResolved() {
		return nil, errors.New("violation is already resolved")
	}

	if violation.PenaltyAmount <= 0 {
		return nil, errors.New("no penalty amount to pay")
	}

	// Get contract for reference
	contract, err := s.contractRepo.GetByID(ctx, violation.ContractID, []string{"Brand", "Brand.User"})
	if err != nil {
		return nil, fmt.Errorf("failed to get contract: %w", err)
	}

	// Get payer ID from brand
	var payerID uuid.UUID
	if contract.Brand != nil && contract.Brand.UserID != nil {
		payerID = *contract.Brand.UserID
	} else {
		payerID = userID
	}

	// Create payment request
	contractNumber := "N/A"
	if contract.ContractNumber != nil {
		contractNumber = *contract.ContractNumber
	}

	paymentReq := &requests.PaymentRequest{
		ReferenceID:   *request.ViolationID,
		ReferenceType: enum.PaymentTransactionReferenceTypeContractViolation,
		PayerID:       &payerID,
		Amount:        int64(violation.PenaltyAmount),
		BuyerName:     contract.Brand.Name,
		Items: []requests.PaymentItemRequest{
			{
				Name:     fmt.Sprintf("Contract Violation Penalty - %s", contractNumber),
				Quantity: 1,
				Price:    int64(violation.PenaltyAmount),
			},
		},
		Description: fmt.Sprintf("Penalty payment for contract violation on contract %s", contractNumber),
		ReturnURL:   request.ReturnURL,
		CancelURL:   request.CancelURL,
	}

	// Generate PayOS payment link using UnitOfWork
	var paymentResp *responses.PayOSLinkResponse
	if err = helper.WithTransaction(ctx, s.unitOfWork, func(ctx context.Context, uow irepository.UnitOfWork) error {
		paymentResp, err = s.paymentTransactionService.GeneratePaymentLink(ctx, uow, paymentReq)
		if err != nil {
			return fmt.Errorf("failed to generate payment link: %w", err)
		}

		var transaction *model.PaymentTransaction
		if transaction, err = uow.PaymentTransaction().GetPaymentTransactionByOrderCode(ctx, strconv.FormatInt(int64(paymentResp.OrderCode), 10)); err != nil {
			return fmt.Errorf("failed to get payment transaction: %w", err)
		} else if transaction == nil || transaction.ID == uuid.Nil {
			return fmt.Errorf("payment transaction not found")
		}

		// Update violation with payment transaction ID
		violation.UpdatedBy = &userID
		violation.PaymentTransactionID = &transaction.ID
		if err = uow.ContractViolations().Update(ctx, violation); err != nil {
			return fmt.Errorf("failed to update violation with payment ID: %w", err)
		}

		return nil
	}); err != nil {
		zap.L().Error("ViolationService - Failed to create penalty payment", zap.Error(err))
		return nil, fmt.Errorf("failed to create penalty payment: %w", err)
	}

	zap.L().Info("ViolationService - Penalty payment created",
		zap.String("request_id", requestID),
		zap.String("violation_id", request.ViolationID.String()),
		zap.String("payment_link", paymentResp.CheckoutURL))

	// Send notification with payment link
	s.sendNotification(
		ctx,
		contract.ID,
		"Action Required: Pay Contract Penalty",
		fmt.Sprintf("Please pay the penalty for contract %s violation. Amount: %.2f", contractNumber, violation.PenaltyAmount),
		"",
		utils.PtrOrNil("brand_penalty_payment_link"),
		map[string]any{
			"BrandName":       contract.Brand.Name,
			"ContractNumber":  contractNumber,
			"ViolationReason": violation.Reason,
			"ViolationDate":   violation.CreatedAt.Format("2006-01-02"),
			"Currency":        paymentResp.Currency,
			"PenaltyAmount":   fmt.Sprintf("%.2f", violation.PenaltyAmount),
			"PaymentLink":     paymentResp.CheckoutURL,
			"ExpiryDate":      time.Unix(paymentResp.ExpiredAt, 0).Format("2006-01-02 15:04 MST"),
			"SupportLink":     s.config.Server.BaseFrontendURL + "/support",
			"CurrentYear":     time.Now().Year(),
		},
		*request.ViolationID,
		enum.ViolationTypeBrand,
		nil,
	)

	return paymentResp, nil
}

// SubmitRefundProof allows KOL to submit proof of refund
func (s *violationService) SubmitRefundProof(
	ctx context.Context,
	violationID uuid.UUID,
	proofURL string,
	message *string,
	submittedBy uuid.UUID,
) (*model.ContractViolation, error) {
	requestID := logging.GetRequestID()
	zap.L().Info("ViolationService - Submitting refund proof",
		zap.String("request_id", requestID),
		zap.String("violation_id", violationID.String()))

	violation, err := s.contractViolationRepo.GetByID(ctx, violationID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get violation: %w", err)
	}

	if violation.Type != enum.ViolationTypeKOL {
		zap.L().Warn("Attempt to submit proof for non-KOL violation",
			zap.String("violation_id", violationID.String()))
		return nil, errors.New("proof submission is only for KOL violations")
	}

	if violation.IsResolved() {
		zap.L().Warn("Attempt to submit proof for resolved violation",
			zap.String("violation_id", violationID.String()))
		return nil, errors.New("violation is already resolved")
	}

	if violation.ProofAttempts >= s.config.AdminConfig.ViolationProofMaxAttempts {
		zap.L().Warn("Maximum proof submission attempts reached",
			zap.String("request_id", requestID),
			zap.String("violation_id", violationID.String()),
			zap.Int("attempts", violation.ProofAttempts))
		return nil, errors.New("maximum proof submission attempts reached")
	}

	// Update proof fields
	now := time.Now()
	proofStatus := enum.ViolationProofStatusPending
	violation.ProofURL = &proofURL
	violation.ProofSubmittedAt = &now
	violation.ProofStatus = &proofStatus
	violation.ProofAttempts += 1
	violation.UpdatedBy = &submittedBy
	violation.ProofSubmittedBy = &submittedBy

	// Store message in proof review note temporarily
	if message != nil {
		violation.ProofReviewNote = message
	}

	if err := s.contractViolationRepo.Update(ctx, violation); err != nil {
		return nil, fmt.Errorf("failed to update violation: %w", err)
	}

	zap.L().Info("ViolationService - Refund proof submitted",
		zap.String("request_id", requestID),
		zap.String("violation_id", violationID.String()))

	// Send notification to Brand
	s.sendNotification(ctx, violation.ContractID, "Refund Proof Submitted",
		"KOL has submitted proof of refund payment.",
		"Refund Proof Submitted",
		utils.PtrOrNil("refund_proof_submitted"),
		map[string]any{
			"ProofURL": proofURL,
			"Message":  utils.DerefPtr(message, "No message"),
		},
		violation.ID,
		enum.ViolationTypeBrand,
		nil,
	)

	return violation, nil
}

// ReviewRefundProof allows admin to approve/reject KOL refund proof
func (s *violationService) ReviewRefundProof(
	ctx context.Context,
	violationID uuid.UUID,
	req *requests.ReviewRefundProofRequest,
	reviewedBy uuid.UUID,
) (*model.ContractViolation, error) {
	requestID := logging.GetRequestID()
	zap.L().Info("ViolationService - Reviewing refund proof",
		zap.String("request_id", requestID),
		zap.String("violation_id", violationID.String()),
		zap.String("action", req.Action))

	violation, err := s.contractViolationRepo.GetByID(ctx, violationID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get violation: %w", err)
	}

	if violation.Type != enum.ViolationTypeKOL {
		return nil, errors.New("proof review is only for KOL violations")
	}

	if violation.IsResolved() {
		return nil, errors.New("violation is already resolved")
	}

	if violation.ProofURL == nil || *violation.ProofURL == "" {
		return nil, errors.New("no proof has been submitted")
	}

	now := time.Now()
	violation.ProofReviewedAt = &now
	violation.ProofReviewedBy = &reviewedBy
	violation.UpdatedBy = &reviewedBy

	if req.IsApprove() {
		approvedStatus := enum.ViolationProofStatusApproved
		violation.ProofStatus = &approvedStatus
		violation.ProofReviewNote = nil
	} else {
		rejectedStatus := enum.ViolationProofStatusRejected
		violation.ProofStatus = &rejectedStatus
		if req.RejectReason != nil {
			violation.ProofReviewNote = req.RejectReason
		}
	}

	if err = helper.WithTransaction(ctx, s.unitOfWork, func(ctx context.Context, uow irepository.UnitOfWork) error {
		if err = uow.ContractViolations().Update(ctx, violation); err != nil {
			return fmt.Errorf("failed to update violation: %w", err)
		}

		if !req.IsApprove() {
			return nil
		}
		// Create a negative payment transaction to refund back to brand
		negativePayment := &model.PaymentTransaction{
			ReferenceID:     violation.ID,
			ReferenceType:   enum.PaymentTransactionReferenceTypeKOLViolationRefunding,
			Amount:          utils.PtrOrNil(-violation.RefundAmount),
			Status:          enum.PaymentTransactionStatusRefunded,
			Method:          enum.ContractPaymentMethodBankTransfer.String(),
			TransactionDate: time.Now(),
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
			PayerID:         violation.ProofSubmittedBy,
			ReceivedByID:    utils.PtrOrNil(reviewedBy),
		}
		if err = uow.PaymentTransaction().Add(ctx, negativePayment); err != nil {
			return fmt.Errorf("failed to add negative payment: %w", err)
		}

		return nil
	}); err != nil {
		zap.L().Error("ViolationService - Failed to review refund proof", zap.Error(err))
		return nil, fmt.Errorf("failed to review refund proof: %w", err)
	}

	zap.L().Info("ViolationService - Refund proof reviewed",
		zap.String("request_id", requestID),
		zap.String("violation_id", violationID.String()),
		zap.String("status", string(*violation.ProofStatus)))

	// Send notifications
	if req.IsApprove() {
		// Notify Brand: Violation Resolved
		s.sendNotification(ctx, violation.ContractID, "Violation Resolved",
			"The KOL refund proof has been approved. The violation is resolved.",
			"Refund Approved",
			utils.PtrOrNil("violation_resolved"),
			map[string]any{
				"ReportedDate":   violation.CreatedAt.Format("2006-01-02"),
				"ResolutionDate": violation.ResolvedAt.Format("2006-01-02"),
				"AmountSettled":  fmt.Sprintf("%.2f", violation.RefundAmount),
				"ResolutionNote": utils.DerefPtr(req.RejectReason, "No reason provided"),
				"Currency":       "VND",
			},
			violation.ID,
			enum.ViolationTypeBrand,
			nil,
		)

		var reviewerFullName string
		reviewerFullName, err = s.userRepo.GetUserFullnameByID(ctx, reviewedBy)
		if err != nil {
			zap.L().Error("Failed to get reviewer full name", zap.Error(err))
			reviewerFullName = "Brand Partner"
		}
		// Notify Staff: Proof Approved (Broadcast)
		s.sendNotification(ctx, violation.ContractID, "Refund Proof Approved",
			"Refund proof has been approved by brand.",
			"Proof Approved",
			utils.PtrOrNil("kol_proof_approved"),
			map[string]any{
				"ViolationID":  violation.ID.String(),
				"Currency":     "VND",
				"RefundAmount": fmt.Sprintf("%.2f", violation.RefundAmount),
				"ReviewerName": reviewerFullName,
				"ApprovalDate": time.Now().Format(utils.TimeFormat),
			},
			violation.ID,
			enum.ViolationTypeKOL,
			violation.ProofSubmittedBy,
		)
	} else {
		// Notify marketing staff: Proof Rejected
		s.sendNotification(ctx, violation.ContractID, "Refund Proof Rejected",
			fmt.Sprintf("Your refund proof was rejected. Reason: %s", utils.DerefPtr(req.RejectReason, "N/A")),
			"Proof Rejected",
			utils.PtrOrNil("kol_proof_rejected"),
			map[string]any{
				"RejectReason":      utils.DerefPtr(req.RejectReason, "N/A"),
				"RemainingAttempts": s.config.AdminConfig.ViolationProofMaxAttempts - violation.ProofAttempts,
				"DashboardLink":     fmt.Sprintf("%s/manage/marketing/contracts/%s", s.config.Server.BaseFrontendURL, violation.ContractID.String()),
			},
			violation.ID,
			enum.ViolationTypeKOL,
			violation.ProofSubmittedBy,
		)
	}

	return violation, nil
}

// AutoApproveProof auto-approves proof after admin review deadline
func (s *violationService) AutoApproveProof(ctx context.Context, violationID uuid.UUID) error {
	requestID := logging.GetRequestID()
	zap.L().Info("ViolationService - Auto-approving proof",
		zap.String("request_id", requestID),
		zap.String("violation_id", violationID.String()))

	violation, err := s.contractViolationRepo.GetByID(ctx, violationID, nil)
	if err != nil {
		return fmt.Errorf("failed to get violation: %w", err)
	}

	if violation.IsResolved() {
		return errors.New("violation is already resolved")
	}

	if violation.ProofStatus == nil || *violation.ProofStatus != enum.ViolationProofStatusPending {
		return errors.New("proof is not in pending status")
	}

	now := time.Now()
	approvedStatus := enum.ViolationProofStatusApproved
	autoApproveNote := "Auto-approved: Review deadline exceeded"

	violation.ProofStatus = &approvedStatus
	violation.ProofReviewedAt = &now
	violation.ProofReviewNote = &autoApproveNote

	if err := s.contractViolationRepo.Update(ctx, violation); err != nil {
		return fmt.Errorf("failed to update violation: %w", err)
	}

	zap.L().Info("ViolationService - Proof auto-approved",
		zap.String("request_id", requestID),
		zap.String("violation_id", violationID.String()))

	return nil
}

// ResolveViolation marks a violation as resolved
func (s *violationService) ResolveViolation(
	ctx context.Context,
	violationID uuid.UUID,
	resolvedBy uuid.UUID,
) error {
	requestID := logging.GetRequestID()
	zap.L().Info("ViolationService - Resolving violation",
		zap.String("request_id", requestID),
		zap.String("violation_id", violationID.String()))

	violation, err := s.contractViolationRepo.GetByID(ctx, violationID, nil)
	if err != nil {
		return fmt.Errorf("failed to get violation: %w", err)
	}

	if violation.IsResolved() {
		return errors.New("violation is already resolved")
	}

	now := time.Now()
	violation.ResolvedAt = &now
	violation.ResolvedBy = &resolvedBy
	violation.UpdatedBy = &resolvedBy

	if err := s.contractViolationRepo.Update(ctx, violation); err != nil {
		return fmt.Errorf("failed to update violation: %w", err)
	}

	zap.L().Info("ViolationService - Violation resolved",
		zap.String("request_id", requestID),
		zap.String("violation_id", violationID.String()))

	return nil
}

// GetByID retrieves a violation by ID
func (s *violationService) GetByID(ctx context.Context, violationID uuid.UUID) (*model.ContractViolation, error) {
	return s.contractViolationRepo.GetByID(ctx, violationID, []string{"Contract", "Campaign", "PaymentTransaction"})
}

// GetByContractID retrieves active violation for a contract
func (s *violationService) GetByContractID(ctx context.Context, contractID uuid.UUID) (*model.ContractViolation, error) {
	return s.contractViolationRepo.FindActiveByContractID(ctx, contractID)
}

// List retrieves violations with filtering
func (s *violationService) List(
	ctx context.Context,
	filter *requests.ViolationFilterRequest,
) ([]*responses.ViolationListResponse, int64, error) {
	filterFunc := func(db *gorm.DB) *gorm.DB {
		if filter.ContractID != nil {
			db = db.Where("contract_id = ?", *filter.ContractID)
		}
		if filter.CampaignID != nil {
			db = db.Where("campaign_id = ?", *filter.CampaignID)
		}
		if filter.Type != nil {
			db = db.Where("type = ?", *filter.Type)
		}
		if filter.IsResolved != nil {
			if *filter.IsResolved {
				db = db.Where("resolved_at IS NOT NULL")
			} else {
				db = db.Where("resolved_at IS NULL")
			}
		}
		if filter.ProofStatus != nil {
			db = db.Where("proof_status = ?", *filter.ProofStatus)
		}
		return db
	}

	violations, total, err := s.contractViolationRepo.GetAll(
		ctx, filterFunc, nil,
		filter.GetPageSize(), filter.GetPage(),
	)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list violations: %w", err)
	}

	// Build response with contract and campaign info
	var result []*responses.ViolationListResponse
	for _, v := range violations {
		resp := &responses.ViolationListResponse{
			ID:            v.ID,
			ContractID:    v.ContractID,
			CampaignID:    v.CampaignID,
			Type:          v.Type,
			Reason:        v.Reason,
			PenaltyAmount: v.PenaltyAmount,
			RefundAmount:  v.RefundAmount,
			ProofStatus:   v.ProofStatus,
			IsResolved:    v.IsResolved(),
			CreatedAt:     v.CreatedAt,
		}

		// Get contract info
		contract, err := s.contractRepo.GetByID(ctx, v.ContractID, []string{"Brand"})
		if err == nil && contract != nil {
			if contract.ContractNumber != nil {
				resp.ContractNumber = *contract.ContractNumber
			}
			if contract.Brand != nil {
				resp.BrandID = contract.Brand.ID
				resp.BrandName = contract.Brand.Name
			}
		}

		// Get campaign info
		if v.CampaignID != nil {
			campaign, err := s.campaignRepo.GetByID(ctx, *v.CampaignID, nil)
			if err == nil && campaign != nil {
				resp.CampaignName = &campaign.Name
			}
		}

		result = append(result, resp)
	}

	return result, total, nil
}

// CalculateBrandPenalty calculates penalty amounts for brand violation
func (s *violationService) CalculateBrandPenalty(
	ctx context.Context,
	contractID uuid.UUID,
) (*responses.ViolationCalculationResponse, error) {
	contract, err := s.contractRepo.GetByID(ctx, contractID, []string{"ContractPayments"})
	if err != nil {
		return nil, fmt.Errorf("failed to get contract: %w", err)
	}

	return s.calculateBrandPenaltyInternal(ctx, contract)
}

// CalculateKOLRefund calculates refund amounts for KOL violation
func (s *violationService) CalculateKOLRefund(
	ctx context.Context,
	contractID uuid.UUID,
) (*responses.ViolationCalculationResponse, error) {
	contract, err := s.contractRepo.GetByID(ctx, contractID, []string{"ContractPayments"})
	if err != nil {
		return nil, fmt.Errorf("failed to get contract: %w", err)
	}

	return s.calculateKOLRefundInternal(ctx, contract)
}

// calculateBrandPenaltyInternal calculates penalty amounts for brand violation
func (s *violationService) calculateBrandPenaltyInternal(
	ctx context.Context,
	contract *model.Contract,
) (*responses.ViolationCalculationResponse, error) {
	// 1. Sum all PAID contract_payments (forfeit amount - informational)
	var totalPaidByBrand float64
	var paymentDetails []responses.PaymentBreakdownDTO

	for _, cp := range contract.ContractPayments {
		detail := responses.PaymentBreakdownDTO{
			ID:          cp.ID,
			Amount:      cp.Amount,
			Status:      string(cp.Status),
			DueDate:     cp.DueDate,
			MilestoneID: cp.MilestoneID,
			IsDeposit:   cp.IsDeposit,
		}
		paymentDetails = append(paymentDetails, detail)

		if cp.Status == enum.ContractPaymentStatusPaid {
			totalPaidByBrand += cp.Amount
		}
	}

	// 2. Find pending payment for active milestone
	activeMilestonePayments, err := s.findActiveMilestonePayment(ctx, contract.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to find active milestone payment: %w", err)
	}
	zap.L().Debug("Active milestone payments found",
		zap.Int("count", len(activeMilestonePayments)))

	var penaltyAmount float64
	for _, amp := range activeMilestonePayments {
		zap.L().Debug("Active milestone payment considered for penalty",
			zap.String("payment_id", amp.ID.String()),
			zap.Float64("amount", amp.Amount))
		penaltyAmount += amp.Amount
	}

	// 3. Get milestone counts
	completedMilestones, totalMilestones, _, err := s.getMilestoneStats(ctx, contract.ID)
	if err != nil {
		return nil, err
	}

	// 4. Parse financial terms for total value
	var totalValue float64
	var financialTerms dtos.FinancialTerms
	if contract.FinancialTerms != nil {
		if err := json.Unmarshal(contract.FinancialTerms, &financialTerms); err == nil {
			if financialTerms.TotalCost != nil {
				totalValue = float64(*financialTerms.TotalCost)
			}
		}
	}

	return &responses.ViolationCalculationResponse{
		ContractID:          contract.ID,
		ContractTotalValue:  totalValue,
		TotalPaidByBrand:    totalPaidByBrand,
		CompletedMilestones: completedMilestones,
		TotalMilestones:     totalMilestones,
		PenaltyAmount:       penaltyAmount,
		CalculationFormula:  "PenaltyAmount = Amount of payment linked to active (ongoing) milestone",
		PaymentDetails:      paymentDetails,
	}, nil
}

// calculateKOLRefundInternal calculates refund amounts for KOL violation
func (s *violationService) calculateKOLRefundInternal(
	ctx context.Context,
	contract *model.Contract,
) (*responses.ViolationCalculationResponse, error) {
	// 1. Sum all PAID contract_payments (refund amount)
	var totalPaidByBrand float64
	var paymentDetails []responses.PaymentBreakdownDTO

	for _, cp := range contract.ContractPayments {
		detail := responses.PaymentBreakdownDTO{
			ID:          cp.ID,
			Amount:      cp.Amount,
			Status:      string(cp.Status),
			DueDate:     cp.DueDate,
			MilestoneID: cp.MilestoneID,
			IsDeposit:   cp.IsDeposit,
		}
		paymentDetails = append(paymentDetails, detail)

		if cp.Status == enum.ContractPaymentStatusPaid {
			totalPaidByBrand += cp.Amount
		}
	}

	// 2. Parse financial terms for total cost
	var totalValue float64
	var financialTerms dtos.FinancialTerms
	if contract.FinancialTerms != nil {
		if err := json.Unmarshal(contract.FinancialTerms, &financialTerms); err == nil {
			if financialTerms.TotalCost != nil {
				totalValue = float64(*financialTerms.TotalCost)
			}
		}
	}

	// 3. Parse legal terms for penalty percentage
	var penaltyPercent float64
	var legalTerms dtos.LegalTerms
	if contract.LegalTerms != nil {
		if err := json.Unmarshal(contract.LegalTerms, &legalTerms); err == nil {
			for _, item := range legalTerms.BreachOfContract.Items {
				if item.CompensationPercent != nil {
					penaltyPercent = float64(*item.CompensationPercent)
					break
				}
			}
		}
	}

	// 4. Calculate refund amount = total paid by brand
	penaltyAmount := totalValue * (penaltyPercent / 100)
	refundAmount := totalPaidByBrand + penaltyAmount

	// 5. Get milestone stats
	completedMilestones, totalMilestones, milestoneDetails, err := s.getMilestoneStats(ctx, contract.ID)
	if err != nil {
		return nil, err
	}

	return &responses.ViolationCalculationResponse{
		ContractID:          contract.ID,
		ContractTotalValue:  totalValue,
		TotalPaidByBrand:    totalPaidByBrand,
		CompletedMilestones: completedMilestones,
		TotalMilestones:     totalMilestones,
		PenaltyPercentage:   penaltyPercent,
		PenaltyAmount:       penaltyAmount,
		RefundAmount:        refundAmount,
		CalculationFormula:  "RefundAmount = Sum of all PAID contract_payments (total amount brand has paid to KOL)",
		PaymentDetails:      paymentDetails,
		MilestoneDetails:    milestoneDetails,
	}, nil
}

// findActiveMilestonePayment finds the payment linked to the active milestone
func (s *violationService) findActiveMilestonePayment(
	ctx context.Context,
	contractID uuid.UUID,
) ([]model.ContractPayment, error) {
	// var payment model.ContractPayment
	var payments []model.ContractPayment

	// Query: Find payment linked to ONGOING milestone

	err := s.db.WithContext(ctx).
		Table("contract_payments cp").
		Joins("JOIN contracts c ON c.id = cp.contract_id").
		Where("c.id = ?", contractID).
		Where("? between cp.period_start and cp.period_end", time.Now()).
		Where("cp.status = ?", enum.ContractPaymentStatusPending).
		Where("cp.deleted_at IS NULL").
		Order("cp.due_date ASC").
		Find(&payments).Error

	/* SELECT "cp"."id",
	          "cp"."contract_id",
	          "cp"."milestone_id",
	          "cp"."installment_percentage",
	          "cp"."amount",
	          "cp"."base_amount",
	          "cp"."performance_amount",
	          "cp"."status",
	          "cp"."due_date",
	          "cp"."payment_method",
	          "cp"."note",
	          "cp"."is_deposit",
	          "cp"."created_at",
	          "cp"."updated_at",
	          "cp"."created_by",
	          "cp"."updated_by",
	          "cp"."deleted_at",
	          "cp"."period_start",
	          "cp"."period_end",
	          "cp"."calculated_at",
	          "cp"."calculation_breakdown",
	          "cp"."locked_amount",
	          "cp"."locked_at",
	          "cp"."locked_clicks",
	          "cp"."locked_revenue"
	   FROM contract_payments cp
	   JOIN contracts c ON c.id = cp.contract_id
	   WHERE c.id = 'ba2f2b35-5b38-4a6f-808b-32fd2d3db344'
	     AND current_timestamp between cp.period_start and cp.period_end
	     AND cp.status = 'PENDING'
	     AND cp.deleted_at IS NULL
	     AND "cp"."deleted_at" IS NULL
	   ORDER BY cp.due_date ASC, "cp"."id" */

	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to find active milestone payment: %w", err)
	}

	return payments, nil
}

// getMilestoneStats gets milestone completion statistics
func (s *violationService) getMilestoneStats(
	ctx context.Context,
	contractID uuid.UUID,
) (completed int, total int, details []responses.MilestoneBreakdownDTO, err error) {
	// Get campaigns for this contract
	campaigns, _, err := s.campaignRepo.GetAll(ctx, func(db *gorm.DB) *gorm.DB {
		return db.Where("contract_id = ?", contractID)
	}, nil, 100, 1)
	if err != nil {
		return 0, 0, nil, fmt.Errorf("failed to get campaigns: %w", err)
	}

	if len(campaigns) == 0 {
		return 0, 0, nil, nil
	}

	// Get milestones for the campaign
	campaignID := campaigns[0].ID
	milestones, _, err := s.milestoneRepo.GetAll(ctx, func(db *gorm.DB) *gorm.DB {
		return db.Where("campaign_id = ?", campaignID)
	}, nil, 100, 1)
	if err != nil {
		return 0, 0, nil, fmt.Errorf("failed to get milestones: %w", err)
	}

	for _, m := range milestones {
		total++
		if m.Status == enum.MilestoneStatusCompleted {
			completed++
		}

		milestoneName := ""
		if m.Description != nil {
			milestoneName = *m.Description
		}

		detail := responses.MilestoneBreakdownDTO{
			ID:         m.ID,
			Name:       milestoneName,
			Percentage: m.CompletionPercentage,
			Status:     string(m.Status),
		}
		details = append(details, detail)
	}

	return completed, total, details, nil
}
