package jobs

import (
	"context"
	"core-backend/config"
	asynqtask "core-backend/internal/application/dto/asynq_tasks"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/application/interfaces/iservice"
	"core-backend/internal/application/service/helper"
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
	asynqClient "core-backend/internal/infrastructure/asynq"
	"core-backend/pkg/gorountine"
	stringsbuilder "core-backend/pkg/strings_builder"
	"core-backend/pkg/utils"
	"fmt"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type DailyJob struct {
	contractRepo             irepository.GenericRepository[model.Contract]
	contractPaymentRepo      irepository.GenericRepository[model.ContractPayment]
	contractViolationRepo    irepository.ContractViolationRepository
	notificationService      iservice.NotificationService
	alertService             iservice.AlertManagerService
	stateTransferService     iservice.StateTransferService
	violationService         iservice.ViolationService
	coProducingRefundService iservice.CoProducingRefundService
	unitOfWork               irepository.UnitOfWork
	asynqClient              *asynqClient.AsynqClient
	appConfig                *config.AppConfig
	db                       *gorm.DB
	cronScheduler            *cron.Cron
	cronExpr                 string
	enabled                  bool
	entryID                  cron.EntryID
	lastRunTime              *time.Time
	workerPool               *gorountine.WorkerPool
}

func NewDailyJob(
	cronScheduler *cron.Cron,
	appConfig *config.AppConfig,
	db *gorm.DB,
	contractRepo irepository.GenericRepository[model.Contract],
	contractPaymentRepo irepository.GenericRepository[model.ContractPayment],
	contractViolationRepo irepository.ContractViolationRepository,
	notificationService iservice.NotificationService,
	alertService iservice.AlertManagerService,
	stateTransferService iservice.StateTransferService,
	violationService iservice.ViolationService,
	coProducingRefundService iservice.CoProducingRefundService,
	unitOfWork irepository.UnitOfWork,
	asynqClient *asynqClient.AsynqClient,
) CronJob {
	cronExpr := appConfig.AdminConfig.DailyCronJobCronExpr
	if cronExpr == "" {
		cronExpr = "0 0 0 * * *" // Default to midnight daily
	}

	return &DailyJob{
		cronScheduler:            cronScheduler,
		cronExpr:                 cronExpr,
		enabled:                  appConfig.AdminConfig.DailyCronJobEnabled,
		contractRepo:             contractRepo,
		contractPaymentRepo:      contractPaymentRepo,
		contractViolationRepo:    contractViolationRepo,
		db:                       db,
		appConfig:                appConfig,
		asynqClient:              asynqClient,
		notificationService:      notificationService,
		alertService:             alertService,
		stateTransferService:     stateTransferService,
		violationService:         violationService,
		coProducingRefundService: coProducingRefundService,
		unitOfWork:               unitOfWork,
		workerPool:               gorountine.NewWorkerPool(context.Background(), appConfig.AdminConfig.DailyCronJobWorkerCount),
	}
}

// Initialize implements [CronJob].
func (j *DailyJob) Initialize() error {
	if !j.enabled {
		zap.L().Info("Daily Job is disabled via admin config")
	}
	zap.L().Debug("Initializing Daily Job...", zap.String("cron_expression", j.cronExpr))

	entryID, err := j.cronScheduler.AddFunc(j.cronExpr, func() {
		if j.enabled {
			j.Run()
		}
	})
	if err != nil {
		zap.L().Error("Failed to schedule Daily Job", zap.Error(err))
		return fmt.Errorf("failed to schedule Daily Job: %w", err)
	}

	j.entryID = entryID
	return nil
}

// Run implements [CronJob].
func (j *DailyJob) Run() {
	startTime := time.Now()
	j.lastRunTime = &startTime
	zap.L().Info("Starting Daily Job...", zap.Time("start_time", startTime))
	j.workerPool.Start()

	j.workerPool.Submit(j.registerContractPaymentOverdue)
	j.workerPool.Submit(j.checkViolationProofAutoApproval)
	j.workerPool.Submit(j.checkViolationProofReviewReminder)
	j.workerPool.Submit(j.checkCoProducingRefundAutoApproval)
	j.workerPool.Submit(j.checkZeroAmountPaymentsToPaid)

	j.workerPool.Close()
	j.workerPool.Wait()
	if j.workerPool.HasErrors() {
		zap.L().Error("Daily Job completed with errors",
			zap.Errors("errors", j.workerPool.Errors()))
	}

	zap.L().Info("Daily Job completed",
		zap.Time("end_time", time.Now()),
		zap.Duration("duration", time.Since(startTime)))
}

// Restart implements [CronJob].
func (j *DailyJob) Restart(adminConfig *config.AdminConfig) error {
	zap.L().Info("Restarting Daily Job with updated configuration...")

	j.enabled = adminConfig.DailyCronJobEnabled
	j.cronExpr = adminConfig.DailyCronJobCronExpr

	if j.entryID != 0 {
		j.cronScheduler.Remove(j.entryID)
		j.entryID = 0
	}
	return j.Initialize()
}

// GetLastRunTime implements [CronJob].
func (j *DailyJob) GetLastRunTime() time.Time {
	if j.lastRunTime == nil {
		return time.Time{} // Return zero time if never run
	}
	return *j.lastRunTime
}

// IsEnabled implements [CronJob].
func (j *DailyJob) IsEnabled() bool {
	return j.enabled
}

// SetEnabled implements [CronJob].
func (j *DailyJob) SetEnabled(enabled bool) {
	j.enabled = enabled
	j.appConfig.AdminConfig.DailyCronJobEnabled = enabled
	if err := j.Restart(&j.appConfig.AdminConfig); err != nil {
		zap.L().Error("Failed to restart Daily Job", zap.Error(err))
	}
}

// region: 1. ======== Registered Jobs Methods ========

// region: 2. ======== Contract Payment Overdue Notification ========

func (j *DailyJob) registerContractPaymentOverdue(ctx context.Context) error {
	zap.L().Info("DailyJob - Checking for overdue contract payments...")
	var (
		currentTime               = time.Now()
		scheduledNotificationHour = j.appConfig.AdminConfig.ContractPaymentNotificationHour
		notificationTime          = time.Date(currentTime.Year(), currentTime.Month(), currentTime.Day(),
			scheduledNotificationHour, 0, 0, 0, currentTime.Location())
	)
	// If the notification time has already passed today, set it to current time
	if currentTime.After(notificationTime) {
		notificationTime = currentTime
	}

	// Find contract payments that are overdued
	filterQuery := func(db *gorm.DB) *gorm.DB {
		return db.Joins("Contract").Joins("Contract.Brand").
			Where("contract_payments.status = ?", enum.ContractPaymentStatusPending).
			Where("contract_payments.due_date < ?", currentTime).
			Where("contracts.status IN ?", []enum.ContractStatus{enum.ContractStatusActive})
	}
	overduedContractPayment := make([]model.ContractPayment, 0)
	contractPayments, contractPaymentCount, err := j.contractPaymentRepo.GetAll(ctx, filterQuery, nil, 0, 0)
	if err != nil {
		zap.L().Error("Failed to find overdue contract payments", zap.Error(err))
		return err
	} else if contractPaymentCount == 0 {
		zap.L().Info("No overdue contract payments found, task completed...")
		return nil
	}
	overduedContractPayment = append(overduedContractPayment, contractPayments...)

	// if more records exist, paginate through them and collect all records
	if int64(len(overduedContractPayment)) < contractPaymentCount {
		for page := 2; int64(len(overduedContractPayment)) < contractPaymentCount; page++ {
			limit := int(min(contractPaymentCount-int64(len(overduedContractPayment)), 100))
			var contractPayments []model.ContractPayment
			contractPayments, _, err = j.contractPaymentRepo.GetAll(ctx, filterQuery, nil, page, limit)
			if err != nil {
				zap.L().Error("Failed to find overdue contract payments",
					zap.Int("page", page), zap.Int("limit", limit), zap.Error(err))
				break
			}
			overduedContractPayment = append(overduedContractPayment, contractPayments...)
		}
		if err != nil {
			return err
		}
	}

	// Group overdue contract payments by ContractID
	overduedContractPaymentByContractID := make(map[uuid.UUID][]*model.ContractPayment)
	for _, cp := range overduedContractPayment {
		if _, exists := overduedContractPaymentByContractID[cp.ContractID]; !exists {
			overduedContractPaymentByContractID[cp.ContractID] = make([]*model.ContractPayment, 0)
		}
		overduedContractPaymentByContractID[cp.ContractID] = append(overduedContractPaymentByContractID[cp.ContractID], &cp)
	}

	for _, payments := range overduedContractPaymentByContractID {
		nearestOverduePayment := slices.MinFunc(payments, func(a, b *model.ContractPayment) int {
			return a.DueDate.Compare(b.DueDate)
		})
		j.scheduleOverdueNotification(ctx, nearestOverduePayment, payments, notificationTime, currentTime)
	}

	return nil
}

func (j *DailyJob) scheduleOverdueNotification(
	ctx context.Context,
	nearestOverduePayment *model.ContractPayment,
	payments []*model.ContractPayment,
	notificationTime time.Time,
	currentTime time.Time,
) error {
	if j.asynqClient == nil {
		zap.L().Warn("Asynq client is not initialized, cannot schedule overdue notification")
		return fmt.Errorf("asynq client is not initialized")
	}

	var (
		contract       = nearestOverduePayment.Contract
		contractNumber = utils.DerefPtr(contract.ContractNumber, "N/A")
		daysOverdue    = int(currentTime.Sub(nearestOverduePayment.DueDate).Hours() / 24)
		title          = fmt.Sprintf("Payment Overdue - %d day(s)", daysOverdue)
		userID         = contract.Brand.UserID
		willTerminate  = daysOverdue >= j.appConfig.AdminConfig.ContractPaymentAllowedOverdueDays
		channels       = []enum.NotificationType{enum.NotificationTypeInApp, enum.NotificationTypeEmail}
	)
	if userID == nil {
		zap.L().Warn("Contract brand has no associated user, cannot send overdue notification",
			zap.String("contract_id", contract.ID.String()))
		return fmt.Errorf("contract brand has no associated user")
	}

	// Create payload for Asynq task
	var taskInfo *asynq.TaskInfo
	var err error
	if willTerminate {
		// ============= Contract will be terminated, send termination warning notification =============
		notiBodyBuilder := stringsbuilder.NewStringBuilder(5).
			AppendLineFormat("Your contract %s is overdue by %d day(s) exceeding the allowed overdue days of %d.",
				contractNumber, daysOverdue, j.appConfig.AdminConfig.ContractPaymentAllowedOverdueDays).
			Append("Because of your failure to make the required payments, we regret to inform you that your contract is scheduled for termination.")

		data := map[string]any{
			"BrandName":          "Valued Customer", // Ideally we'd fetch this or include in payload
			"ContractNumber":     contractNumber,
			"DaysOverdue":        daysOverdue,
			"AllowedOverdueDays": j.appConfig.AdminConfig.ContractPaymentAllowedOverdueDays,
			"SupportLink":        j.appConfig.Server.BaseFrontendURL + "/support",
			"CurrentYear":        currentTime.Year(),
		}

		stringData := make(map[string]string, len(data))
		for key, value := range data {
			stringData[key] = utils.ToString(value)
		}

		// Initiate brand violation for non-payment (BEFORE termination)
		violationReason := fmt.Sprintf(
			"Contract terminated due to non-payment. Payments were overdue for %d days (max allowed: %d days).",
			daysOverdue,
			j.appConfig.AdminConfig.ContractPaymentAllowedOverdueDays,
		)
		// Use uuid.Nil as reportedBy since this is a system-initiated violation
		if _, err = j.violationService.InitiateBrandViolation(ctx, contract.ID, uuid.Nil, violationReason); err != nil {
			zap.L().Error("Failed to initiate brand violation before contract termination",
				zap.String("contract_id", contract.ID.String()),
				zap.Error(err))

			j.raiseAlert(ctx, &AlertInfoRequest{
				Type:          enum.AlertTypeError,
				Category:      enum.AlertCategoryViolationDetected,
				Severity:      enum.AlertSeverityHigh,
				Title:         "Brand Violation Creation Failed",
				Description:   fmt.Sprintf("Failed to create violation record for overdue contract %s", contractNumber),
				ReferenceID:   contract.ID,
				ReferenceType: enum.ReferenceTypeContract,
				TargetRoles:   []enum.UserRole{enum.UserRoleSalesStaff, enum.UserRoleAdmin},
			}, nil, []enum.UserRole{enum.UserRoleSalesStaff, enum.UserRoleAdmin})
			// Proceed to termination anyway
		} else {
			zap.L().Info("Brand violation initiated for overdue contract",
				zap.String("contract_id", contract.ID.String()),
				zap.String("violation_reason", violationReason))
		}

		// Transition to BRAND_VIOLATED state instead of TERMINATED
		if err = helper.WithTransaction(ctx, j.unitOfWork, func(ctx context.Context, uow irepository.UnitOfWork) error {
			return j.stateTransferService.MoveContractToState(ctx, uow, contract.ID, enum.ContractStatusBrandViolated, uuid.Nil)
		}); err != nil {
			zap.L().Error("Failed to transition contract to brand violation state",
				zap.String("contract_id", contract.ID.String()),
				zap.Error(err))

			j.raiseAlert(ctx, &AlertInfoRequest{
				Type:          enum.AlertTypeError,
				Category:      enum.AlertCategoryContractTerminateFailed,
				Severity:      enum.AlertSeverityHigh,
				Title:         "Contract Violation Transition Failed",
				Description:   "Failed to transition contract to brand violation state after overdue notification",
				ReferenceID:   contract.ID,
				ReferenceType: enum.ReferenceTypeContract,
				TargetRoles:   []enum.UserRole{enum.UserRoleContentStaff, enum.UserRoleAdmin},
			}, nil, []enum.UserRole{enum.UserRoleContentStaff, enum.UserRoleAdmin})
			return err
		}

		scheduledNotificationPayload := &asynqtask.ScheduledNotificationPayload{
			UserID:       *userID,
			Title:        title,
			Body:         notiBodyBuilder.String(),
			Data:         stringData,
			Types:        channels,
			TemplateName: "contract_terminated",
			Subject:      fmt.Sprintf("URGENT: Contract Termination Notice - %s", contractNumber),
			TemplateData: data,
			ScheduleIDs:  nil,
			ScheduleType: utils.PtrOrNil(enum.ScheduleTypeContractNotification),
		}
		taskType := j.appConfig.Asynq.TaskTypes.NotificationSchedule
		uniqueKey := fmt.Sprintf("contract:terminatation:%s:%s",
			contract.ID.String(), utils.FormatLocalTime(&currentTime, utils.TimestampStringFormat))

		taskInfo, err = j.asynqClient.ScheduleTaskWithUniqueKey(ctx, taskType, scheduledNotificationPayload, notificationTime, uniqueKey)
		if err != nil {
			zap.L().Error("Failed to schedule contract termination notification task",
				zap.String("contract_id", contract.ID.String()),
				zap.Error(err))
			return err
		}

	} else {
		// ============= Regular overdue payment notification =============

		notiBodyBuilder := stringsbuilder.NewStringBuilder(5).
			AppendFormat("Your contract #%s has ", contractNumber).
			If(len(payments) > 1, fmt.Sprintf("%d overdue payments", len(payments))).
			Else("an overdue payment").End().
			AppendFormat(" of %.2f", nearestOverduePayment.Amount).
			AppendLine("").
			AppendLineFormat("The nearest overdue payment was due on %s.", utils.FormatLocalTime(&nearestOverduePayment.DueDate, utils.DateFormat)).
			AppendFormat("Please make the payment as soon as possible to avoid contract termination in the next %d days.",
				j.appConfig.AdminConfig.ContractPaymentAllowedOverdueDays-daysOverdue)

		brandName := "Valued Customer"
		if contract.Brand != nil && contract.Brand.Name != "" {
			brandName = contract.Brand.Name
		}

		data := map[string]any{
			"BrandName":            brandName,
			"ContractNumber":       contractNumber,
			"OverduePaymentCount":  len(payments),
			"TotalAmount":          fmt.Sprintf("%.2f", nearestOverduePayment.Amount),
			"DueDate":              utils.FormatLocalTime(&nearestOverduePayment.DueDate, utils.DateFormat),
			"DaysUntilTermination": j.appConfig.AdminConfig.ContractPaymentAllowedOverdueDays - daysOverdue,
			"PaymentLink":          fmt.Sprintf("%s/manage/brand/contract-payment", j.appConfig.Server.BaseFrontendURL),
			"CurrentYear":          time.Now().Year(),
		}

		stringData := make(map[string]string, len(data))
		for key, value := range data {
			stringData[key] = utils.ToString(value)
		}

		scheduledNotificationPayload := &asynqtask.ScheduledNotificationPayload{
			UserID:       *userID,
			Title:        title,
			Body:         notiBodyBuilder.String(),
			Data:         stringData,
			Types:        channels,
			TemplateName: "contract_payment_overdue",
			Subject:      fmt.Sprintf("Payment Overdue Notice - %s", contractNumber),
			TemplateData: data,
			ScheduleIDs:  nil,
			ScheduleType: utils.PtrOrNil(enum.ScheduleTypeContractNotification),
		}
		taskType := j.appConfig.Asynq.TaskTypes.NotificationSchedule
		uniqueKey := fmt.Sprintf("contract:overdue:%s:%s",
			contract.ID.String(), utils.FormatLocalTime(&currentTime, utils.TimestampStringFormat))

		taskInfo, err = j.asynqClient.ScheduleTaskWithUniqueKey(ctx, taskType, scheduledNotificationPayload, notificationTime, uniqueKey)
		if err != nil {
			zap.L().Error("Failed to schedule contract termination notification task",
				zap.String("contract_id", contract.ID.String()),
				zap.Error(err))
			return err
		}
	}

	zap.L().Info("Scheduled overdue notification task",
		zap.String("contract_id", contract.ID.String()),
		zap.String("task_id", taskInfo.ID),
		zap.Time("notification_time", notificationTime))

	return nil
}

// endregion 2.

// region: 3. ======== Violation Proof Auto-Approval ========

// checkViolationProofAutoApproval automatically approves KOL violation proofs
// that have been pending for longer than the configured review period
func (j *DailyJob) checkViolationProofAutoApproval(ctx context.Context) error {
	requestID := uuid.New().String()
	zap.L().Info("DailyJob - Checking for violation proofs to auto-approve...",
		zap.String("request_id", requestID))

	// Calculate cutoff date: proofs submitted before this date will be auto-approved
	reviewDays := j.appConfig.AdminConfig.ViolationProofReviewDays
	if reviewDays <= 0 {
		reviewDays = 7 // Default to 7 days
	}
	cutoffDate := time.Now().AddDate(0, 0, -reviewDays)

	zap.L().Debug("DailyJob - Auto-approval cutoff date",
		zap.String("request_id", requestID),
		zap.Time("cutoff_date", cutoffDate),
		zap.Int("review_days", reviewDays))

	// Find violations with pending proofs that are overdue
	violations, err := j.contractViolationRepo.FindProofsOverdueForAutoApproval(ctx, cutoffDate)
	if err != nil {
		zap.L().Error("DailyJob - Failed to find violations for auto-approval",
			zap.String("request_id", requestID),
			zap.Error(err))
		return err
	}

	if len(violations) == 0 {
		zap.L().Info("DailyJob - No violation proofs found for auto-approval",
			zap.String("request_id", requestID))
		return nil
	}

	zap.L().Info("DailyJob - Found violations for auto-approval",
		zap.String("request_id", requestID),
		zap.Int("count", len(violations)))

	// Process each violation
	var autoApprovedCount int
	for _, violation := range violations {
		if err := j.processViolationAutoApproval(ctx, requestID, violation); err != nil {
			zap.L().Error("DailyJob - Failed to auto-approve violation proof",
				zap.String("request_id", requestID),
				zap.String("violation_id", violation.ID.String()),
				zap.Error(err))
			// Continue processing other violations
			continue
		}
		autoApprovedCount++
	}

	zap.L().Info("DailyJob - Violation proof auto-approval completed",
		zap.String("request_id", requestID),
		zap.Int("total_processed", len(violations)),
		zap.Int("auto_approved", autoApprovedCount))

	return nil
}

// processViolationAutoApproval handles the auto-approval of a single violation
func (j *DailyJob) processViolationAutoApproval(ctx context.Context, requestID string, violation *model.ContractViolation) error {
	zap.L().Info("DailyJob - Auto-approving violation proof",
		zap.String("request_id", requestID),
		zap.String("violation_id", violation.ID.String()),
		zap.String("contract_id", violation.ContractID.String()))

	// Auto-approve the proof using ViolationService
	if err := j.violationService.AutoApproveProof(ctx, violation.ID); err != nil {
		return fmt.Errorf("failed to auto-approve proof: %w", err)
	}

	// Move contract to KOL_REFUND_APPROVED state using UnitOfWork
	if err := helper.WithTransaction(ctx, j.unitOfWork, func(ctx context.Context, uow irepository.UnitOfWork) error {
		return j.stateTransferService.MoveContractToState(ctx, uow, violation.ContractID, enum.ContractStatusKOLRefundApproved, uuid.Nil)
	}); err != nil {
		zap.L().Error("DailyJob - Failed to move contract to KOL_REFUND_APPROVED",
			zap.String("request_id", requestID),
			zap.String("violation_id", violation.ID.String()),
			zap.String("contract_id", violation.ContractID.String()),
			zap.Error(err))

		// Raise an alert for manual intervention
		j.raiseAlert(ctx, &AlertInfoRequest{
			Type:          enum.AlertTypeError,
			Category:      enum.AlertCategoryViolationDetected,
			Severity:      enum.AlertSeverityHigh,
			Title:         "Violation Auto-Approval State Transition Failed",
			Description:   fmt.Sprintf("Auto-approved violation %s but failed to transition contract %s to KOL_REFUND_APPROVED", violation.ID.String(), violation.ContractID.String()),
			ReferenceID:   violation.ID,
			ReferenceType: enum.ReferenceTypeContractViolation,
			TargetRoles:   []enum.UserRole{enum.UserRoleSalesStaff, enum.UserRoleAdmin},
		}, nil, []enum.UserRole{enum.UserRoleSalesStaff, enum.UserRoleAdmin})

		return err
	}

	// Send notifications to both parties
	j.sendViolationProofAutoApprovedNotification(ctx, requestID, violation)

	zap.L().Info("DailyJob - Violation proof auto-approved and contract transitioned",
		zap.String("request_id", requestID),
		zap.String("violation_id", violation.ID.String()),
		zap.String("contract_id", violation.ContractID.String()))

	return nil
}

// sendViolationProofAutoApprovedNotification sends notification about auto-approved proof
func (j *DailyJob) sendViolationProofAutoApprovedNotification(ctx context.Context, requestID string, violation *model.ContractViolation) {
	// Get contract with brand and KOL details
	contract, err := j.contractRepo.GetByID(ctx, violation.ContractID, []string{"Brand", "Brand.User", "KOL", "KOL.User"})
	if err != nil {
		zap.L().Error("DailyJob - Failed to get contract for notification",
			zap.String("request_id", requestID),
			zap.String("violation_id", violation.ID.String()),
			zap.Error(err))
		return
	}

	contractNumber := "N/A"
	if contract.ContractNumber != nil {
		contractNumber = *contract.ContractNumber
	}

	reviewDays := j.appConfig.AdminConfig.ViolationProofReviewDays
	if reviewDays <= 0 {
		reviewDays = 7
	}

	// Build notification content
	title := "Refund Proof Auto-Approved"
	body := fmt.Sprintf("The refund proof for contract %s has been automatically approved after %d days without review.",
		contractNumber, reviewDays)

	data := map[string]string{
		"violation_id":    violation.ID.String(),
		"contract_id":     contract.ID.String(),
		"contract_number": contractNumber,
		"reference_type":  enum.ReferenceTypeContractViolation.String(),
		"reference_id":    violation.ID.String(),
	}

	channels := []enum.NotificationType{enum.NotificationTypeInApp, enum.NotificationTypeEmail}

	// Notify brand owner
	if contract.Brand != nil && contract.Brand.UserID != nil {
		templateData := map[string]any{
			"BrandName":      contract.Brand.Name,
			"ContractNumber": contractNumber,
			"ReviewDays":     reviewDays,
			"RefundAmount":   fmt.Sprintf("%.2f", violation.RefundAmount),
			"SupportLink":    j.appConfig.Server.BaseFrontendURL + "/support",
			"CurrentYear":    time.Now().Year(),
		}

		notificationPayload := &asynqtask.ScheduledNotificationPayload{
			UserID:       *contract.Brand.UserID,
			Title:        title,
			Body:         body,
			Data:         data,
			Types:        channels,
			TemplateName: "proof_auto_approved",
			Subject:      fmt.Sprintf("Refund Proof Auto-Approved - Contract %s", contractNumber),
			TemplateData: templateData,
		}

		taskType := j.appConfig.Asynq.TaskTypes.NotificationSchedule
		uniqueKey := fmt.Sprintf("violation:auto_approved:brand:%s:%s",
			violation.ID.String(), time.Now().Format(utils.TimestampStringFormat))

		if _, err := j.asynqClient.ScheduleTaskWithUniqueKey(ctx, taskType, notificationPayload, time.Now(), uniqueKey); err != nil {
			zap.L().Error("DailyJob - Failed to schedule auto-approval notification to brand",
				zap.String("request_id", requestID),
				zap.Error(err))
		}
	}

	// Notify KOL (marketing staff)
	staffRoles := []enum.UserRole{enum.UserRoleSalesStaff, enum.UserRoleContentStaff}
	staffNotification := requests.PublishNotificationRequest{
		Title: title,
		Body:  body,
		Data:  data,
		Types: []enum.NotificationType{enum.NotificationTypeInApp},
	}
	if err := j.notificationService.BroadcastToRoleWithRequest(ctx, staffRoles, &staffNotification); err != nil {
		zap.L().Error("DailyJob - Failed to broadcast auto-approval notification to staff",
			zap.String("request_id", requestID),
			zap.Error(err))
	}
}

// endregion 3.

// endregion 1.

// checkViolationProofReviewReminder sends reminders for proofs approaching auto-approval
func (j *DailyJob) checkViolationProofReviewReminder(ctx context.Context) error {
	requestID := uuid.New().String()
	zap.L().Info("DailyJob - Checking for violation proof review reminders...",
		zap.String("request_id", requestID))

	reviewDays := j.appConfig.AdminConfig.ViolationProofReviewDays
	if reviewDays <= 0 {
		reviewDays = 7
	}

	// Reminder window: submitted 1 day before auto-approval cutoff
	// Cutoff is (Now - ReviewDays). Approaching cutoff means submitted slightly *after* cutoff.
	// Specifically: SubmittedAt = Now - (ReviewDays - 1).
	daysUntilAutoApprove := 1
	targetSubmissionDate := time.Now().AddDate(0, 0, -(reviewDays - daysUntilAutoApprove))

	filterFunc := func(db *gorm.DB) *gorm.DB {
		return db.Where("proof_status = ?", enum.ViolationProofStatusPending).
			Where("proof_submitted_at::date = ?::date", targetSubmissionDate)
	}

	violations, _, err := j.contractViolationRepo.GetAll(ctx, filterFunc, nil, 1000, 1)
	if err != nil {
		zap.L().Error("DailyJob - Failed to find violations for reminder", zap.Error(err))
		return err
	}

	if len(violations) == 0 {
		return nil
	}

	zap.L().Info("DailyJob - Found violations for reminder", zap.Int("count", len(violations)))

	for _, v := range violations {
		j.sendViolationProofReviewReminder(ctx, requestID, &v, daysUntilAutoApprove)
	}

	return nil
}

// sendViolationProofReviewReminder sends notification to staff about pending proof
func (j *DailyJob) sendViolationProofReviewReminder(ctx context.Context, requestID string, violation *model.ContractViolation, daysLeft int) {
	contract, err := j.contractRepo.GetByID(ctx, violation.ContractID, []string{"Brand"})
	if err != nil {
		zap.L().Error("DailyJob - Failed to get contract for reminder", zap.Error(err))
		return
	}

	contractNumber := "N/A"
	if contract.ContractNumber != nil {
		contractNumber = *contract.ContractNumber
	}

	zap.L().Info("DailyJob - Sending review reminder",
		zap.String("request_id", requestID),
		zap.String("violation_id", violation.ID.String()))

	title := "Reminder: Refund Proof Auto-Approval Imminent"
	body := fmt.Sprintf("Refund proof for contract %s will be auto-approved in %d day(s). Please review it.", contractNumber, daysLeft)

	data := map[string]string{
		"violation_id":    violation.ID.String(),
		"contract_id":     contract.ID.String(),
		"contract_number": contractNumber,
		"days_left":       fmt.Sprintf("%d", daysLeft),
	}

	templateData := map[string]any{
		"ContractNumber": contractNumber,
		"DaysLeft":       daysLeft,
		"BrandName":      contract.Brand.Name,
		"SubmittedAt":    utils.FormatLocalTime(violation.ProofSubmittedAt, utils.DateFormat),
		"SupportLink":    j.appConfig.Server.BaseFrontendURL + "/admin/dashboard",
		"CurrentYear":    time.Now().Year(),
	}

	notificationReq := requests.PublishNotificationRequest{
		Title:             title,
		Body:              body,
		Data:              data,
		Types:             []enum.NotificationType{enum.NotificationTypeInApp, enum.NotificationTypeEmail},
		EmailTemplateName: utils.PtrOrNil("proof_review_reminder"),
		EmailTemplateData: templateData,
	}

	roles := []enum.UserRole{enum.UserRoleSalesStaff, enum.UserRoleContentStaff}
	if err := j.notificationService.BroadcastToRoleWithRequest(ctx, roles, &notificationReq); err != nil {
		zap.L().Error("DailyJob - Failed to broadcast review reminder", zap.Error(err))
	}
}

// region: 1 ======= Helper Methods ========

type AlertInfoRequest struct {
	Type          enum.AlertType
	Category      enum.AlertCategory
	Severity      enum.AlertSeverity
	Title         string
	Description   string
	ReferenceID   uuid.UUID
	ReferenceType enum.ReferenceType
	TargetRoles   []enum.UserRole
}

func (j *DailyJob) raiseAlert(ctx context.Context, req *AlertInfoRequest, notifyUserID *uuid.UUID, notifyUserRoles []enum.UserRole) {
	if notifyUserID == nil && len(req.TargetRoles) == 0 {
		zap.L().Warn("No notify user ID or role provided, cannot raise alert")
	}

	alertRequest := &requests.RaiseAlertRequest{
		Type:           req.Type,
		Category:       req.Category,
		Severity:       req.Severity,
		Title:          req.Title,
		Description:    req.Description,
		ReferenceID:    &req.ReferenceID,
		ReferenceType:  &req.ReferenceType,
		ActionURL:      nil,
		ExpiresInHours: nil,
		TargetRoles:    req.TargetRoles,
	}
	if _, err := j.alertService.RaiseAlert(ctx, alertRequest); err != nil {
		zap.L().Error("Failed to raise alert",
			zap.Any("request", req),
			zap.Error(err))
		return
	}

	if notifyUserID != nil {
		notificationRequest := requests.PublishInAppRequest{
			UserID: *notifyUserID,
			Title:  req.Title,
			Body:   req.Description,
			Data: map[string]string{
				"reference_type": req.ReferenceType.String(),
				"reference_id":   req.ReferenceID.String(),
			},
		}
		if _, err := j.notificationService.CreateAndPublishInApp(ctx, &notificationRequest); err != nil {
			zap.L().Error("Failed to publish in-app notification to specified user",
				zap.Any("request", req),
				zap.Error(err))
			return
		}
	} else if notifyUserRoles != nil {
		notificationRequest := requests.PublishNotificationRequest{
			Title: req.Title,
			Body:  req.Description,
			Data: map[string]string{
				"reference_type": req.ReferenceType.String(),
				"reference_id":   req.ReferenceID.String(),
			},
			Types: []enum.NotificationType{enum.NotificationTypeInApp},
		}
		if err := j.notificationService.BroadcastToRoleWithRequest(ctx, notifyUserRoles, &notificationRequest); err != nil {
			zap.L().Error("Failed to broadcast in-app notification to specified role",
				zap.Any("request", req),
				zap.Error(err))
			return
		}
	}
}

// endregion

// region: 4. ======== CO_PRODUCING Refund Auto-Approval ========

// checkCoProducingRefundAutoApproval automatically approves CO_PRODUCING refund proofs
// that have been pending for longer than the configured review period
func (j *DailyJob) checkCoProducingRefundAutoApproval(ctx context.Context) error {
	requestID := uuid.New().String()
	zap.L().Info("DailyJob - Checking for CO_PRODUCING refund proofs to auto-approve...",
		zap.String("request_id", requestID))

	if j.coProducingRefundService == nil {
		zap.L().Warn("DailyJob - CoProducingRefundService is not initialized, skipping auto-approval check")
		return nil
	}

	// Calculate cutoff date: proofs submitted before this date will be auto-approved
	reviewDays := j.appConfig.AdminConfig.CoProducingRefundReviewDays
	if reviewDays <= 0 {
		reviewDays = 7 // Default to 7 days
	}
	cutoffDate := time.Now().AddDate(0, 0, -reviewDays)

	zap.L().Debug("DailyJob - CO_PRODUCING refund auto-approval cutoff date",
		zap.String("request_id", requestID),
		zap.Time("cutoff_date", cutoffDate),
		zap.Int("review_days", reviewDays))

	// Find payments with pending refund proofs that are overdue
	payments, err := j.coProducingRefundService.GetPendingRefundProofs(ctx, &cutoffDate)
	if err != nil {
		zap.L().Error("DailyJob - Failed to find CO_PRODUCING refund proofs for auto-approval",
			zap.String("request_id", requestID),
			zap.Error(err))
		return err
	}

	if len(payments) == 0 {
		zap.L().Info("DailyJob - No CO_PRODUCING refund proofs found for auto-approval",
			zap.String("request_id", requestID))
		return nil
	}

	zap.L().Info("DailyJob - Found CO_PRODUCING refund proofs for auto-approval",
		zap.String("request_id", requestID),
		zap.Int("count", len(payments)))

	// Process each payment
	var autoApprovedCount int
	for _, payment := range payments {
		if err := j.coProducingRefundService.AutoApproveRefundProof(ctx, payment.ID); err != nil {
			zap.L().Error("DailyJob - Failed to auto-approve CO_PRODUCING refund proof",
				zap.String("request_id", requestID),
				zap.String("payment_id", payment.ID.String()),
				zap.Error(err))
			continue
		}
		autoApprovedCount++
	}

	zap.L().Info("DailyJob - CO_PRODUCING refund proof auto-approval completed",
		zap.String("request_id", requestID),
		zap.Int("total_processed", len(payments)),
		zap.Int("auto_approved", autoApprovedCount))

	return nil
}

// endregion

// region: 5. ======== Zero-Amount Payments to PAID ========

// checkZeroAmountPaymentsToPaid marks CO_PRODUCING payments with zero net amount as PAID
// after their due date has passed
func (j *DailyJob) checkZeroAmountPaymentsToPaid(ctx context.Context) error {
	requestID := uuid.New().String()
	zap.L().Info("DailyJob - Checking for zero-amount payments to mark as PAID...",
		zap.String("request_id", requestID))

	currentTime := time.Now()

	// Find CO_PRODUCING payments where:
	// - Status is PENDING
	// - Amount is 0 (zero net amount)
	// - Due date has passed
	// - Contract type is CO_PRODUCING
	var payments []model.ContractPayment
	err := j.db.WithContext(ctx).
		Model(&model.ContractPayment{}).
		Joins("Contract").
		Where("contract_payments.status = ?", enum.ContractPaymentStatusPending).
		Where("contract_payments.amount = 0").
		Where("contract_payments.due_date < ?", currentTime).
		Where("\"Contract\".type = ?", enum.ContractTypeCoProduce).
		Find(&payments).Error

	if err != nil {
		zap.L().Error("DailyJob - Failed to find zero-amount payments",
			zap.String("request_id", requestID),
			zap.Error(err))
		return err
	}

	if len(payments) == 0 {
		zap.L().Info("DailyJob - No zero-amount payments found to mark as PAID",
			zap.String("request_id", requestID))
		return nil
	}

	zap.L().Info("DailyJob - Found zero-amount payments to mark as PAID",
		zap.String("request_id", requestID),
		zap.Int("count", len(payments)))

	// Mark each payment as PAID
	var markedCount int
	for _, payment := range payments {
		err := helper.WithTransaction(ctx, j.unitOfWork, func(ctx context.Context, uow irepository.UnitOfWork) error {
			return uow.ContractPayments().UpdateByCondition(ctx, func(db *gorm.DB) *gorm.DB {
				return db.Where("id = ?", payment.ID).Where("status = ?", enum.ContractPaymentStatusPending)
			}, map[string]any{
				"status": enum.ContractPaymentStatusPaid,
			})
		})

		if err != nil {
			zap.L().Error("DailyJob - Failed to mark zero-amount payment as PAID",
				zap.String("request_id", requestID),
				zap.String("payment_id", payment.ID.String()),
				zap.Error(err))
			continue
		}

		zap.L().Info("DailyJob - Marked zero-amount payment as PAID",
			zap.String("request_id", requestID),
			zap.String("payment_id", payment.ID.String()),
			zap.String("contract_id", payment.ContractID.String()))
		markedCount++
	}

	zap.L().Info("DailyJob - Zero-amount payment processing completed",
		zap.String("request_id", requestID),
		zap.Int("total_processed", len(payments)),
		zap.Int("marked_paid", markedCount))

	return nil
}

// endregion
