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
	contractRepo        irepository.GenericRepository[model.Contract]
	contractPaymentRepo irepository.GenericRepository[model.ContractPayment]
	notificationService iservice.NotificationService
	alertService        iservice.AlertManagerService
	unitOfWork          irepository.UnitOfWork
	asynqClient         *asynqClient.AsynqClient
	appConfig           *config.AppConfig
	db                  *gorm.DB
	cronScheduler       *cron.Cron
	cronExpr            string
	enabled             bool
	entryID             cron.EntryID
	lastRunTime         *time.Time
	workerPool          *gorountine.WorkerPool
}

func NewDailyJob(
	cronScheduler *cron.Cron,
	appConfig *config.AppConfig,
	db *gorm.DB,
	contractRepo irepository.GenericRepository[model.Contract],
	contractPaymentRepo irepository.GenericRepository[model.ContractPayment],
	notificationService iservice.NotificationService,
	alertService iservice.AlertManagerService,
	unitOfWork irepository.UnitOfWork,
	asynqClient *asynqClient.AsynqClient,
) CronJob {
	cronExpr := appConfig.AdminConfig.DailyCronJobCronExpr
	if cronExpr == "" {
		cronExpr = "0 0 0 * * *" // Default to midnight daily
	}

	return &DailyJob{
		cronScheduler:       cronScheduler,
		cronExpr:            cronExpr,
		enabled:             appConfig.AdminConfig.DailyCronJobEnabled,
		contractRepo:        contractRepo,
		contractPaymentRepo: contractPaymentRepo,
		db:                  db,
		appConfig:           appConfig,
		asynqClient:         asynqClient,
		notificationService: notificationService,
		alertService:        alertService,
		unitOfWork:          unitOfWork,
		workerPool:          gorountine.NewWorkerPool(context.Background(), appConfig.AdminConfig.DailyCronJobWorkerCount),
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

// region: 1. ======== Helper Methods ========

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
			Where("contract_payments.due_date < ?", currentTime)
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

		// Terminate the contract immediately after sending the notification
		if err = helper.WithTransaction(ctx, j.unitOfWork, func(ctx context.Context, uow irepository.UnitOfWork) error {
			return uow.Contracts().UpdateByCondition(ctx, func(db *gorm.DB) *gorm.DB {
				return db.Where("id = ?", contract.ID).Where("status = ?", enum.ContractStatusActive)
			}, map[string]any{"status": enum.ContractStatusTerminated})
		}); err != nil {
			zap.L().Error("Failed to terminate contract after overdue notification",
				zap.String("contract_id", contract.ID.String()),
				zap.Error(err))

			j.raiseAlert(ctx, &AlertInfoRequest{
				Type:          enum.AlertTypeError,
				Category:      enum.AlertCategoryContractTerminateFailed,
				Severity:      enum.AlertSeverityHigh,
				Title:         "Contract Termination Failed",
				Description:   "Failed to terminate contract after overdue notification",
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
			TemplateName: "contract_termination",
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

// endregion 1.

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
			Channels: []string{enum.NotificationTypeInApp.String()},
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
