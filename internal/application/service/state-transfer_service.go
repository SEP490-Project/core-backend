package service

import (
	"context"
	"core-backend/config"
	asynqtask "core-backend/internal/application/dto/asynq_tasks"
	"core-backend/internal/application/dto/consumers"
	"core-backend/internal/application/dto/dtos"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/interfaces/iproxies"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/application/interfaces/iservice"
	"core-backend/internal/application/service/helper"
	"core-backend/internal/application/service/notification_builder"
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
	"core-backend/internal/domain/state/campaignsm"
	"core-backend/internal/domain/state/contentsm"
	"core-backend/internal/domain/state/contractsm"
	"core-backend/internal/domain/state/milestonesm"
	"core-backend/internal/domain/state/ordersm"
	"core-backend/internal/domain/state/paymenttransactionsm"
	"core-backend/internal/domain/state/productsm"
	"core-backend/internal/domain/state/tasksm"
	"core-backend/internal/infrastructure/asynq"
	gormrepository "core-backend/internal/infrastructure/gorm_repository"
	"core-backend/internal/infrastructure/rabbitmq"
	"core-backend/pkg/utils"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	gAsynq "github.com/hibiken/asynq"
	"go.uber.org/zap"
	"gorm.io/gorm" // added for UpdateByCondition filter closure
)

type stateTransferService struct {
	contractRepository          irepository.GenericRepository[model.Contract]
	contractPaymentRepository   irepository.ContractPaymentRepository
	contractViolationRepository irepository.ContractViolationRepository
	campaignRepository          irepository.GenericRepository[model.Campaign]
	milestoneRepository         irepository.GenericRepository[model.Milestone]
	taskRepository              irepository.GenericRepository[model.Task]
	productRepository           irepository.GenericRepository[model.Product]
	orderRepository             irepository.GenericRepository[model.Order]
	preOrderRepository          irepository.PreOrderRepository
	variantRepository           irepository.GenericRepository[model.ProductVariant]
	affiliateLinkRepository     irepository.AffiliateLinkRepository
	userRepository              irepository.GenericRepository[model.User]
	contentRepository           irepository.ContentRepository
	notificationService         iservice.NotificationService
	scheduleRepo                irepository.ScheduleRepository
	scheduleService             iservice.ScheduleService
	uow                         irepository.UnitOfWork
	rabbitMQ                    *rabbitmq.RabbitMQ
	ghnProxy                    iproxies.GHNProxy
	adminConfig                 config.AdminConfig
	config                      *config.AppConfig
	taskScheduler               *asynq.AsynqClient
	asynqConfig                 *config.AsynqConfig
}

func (t stateTransferService) MoveOrderToStateByGHNWebhook(ctx context.Context, ghnCode string, ghnStatus enum.GHNDeliveryStatus, orderType string) error {
	filter := func(db *gorm.DB) *gorm.DB {
		return db.Where("ghn_order_code = ?", ghnCode)
	}

	// Handle PreOrder
	if orderType == "PREORDER" {
		preOrder, err := t.preOrderRepository.GetByCondition(ctx, filter, []string{"ProductVariant", "ProductVariant.Product", "ProductVariant.Product.Limited"})
		if err != nil {
			zap.L().Error("Failed to load preorder from DB by GHN code",
				zap.String("ghn_order_code", ghnCode),
				zap.Error(err))
			return errors.New("Unable to find preorder by GHN code: " + err.Error())
		}

		// Map GHN status to PreOrder status
		var newStatus enum.PreOrderStatus
		switch ghnStatus {
		case enum.GHNDeliveryStatusStoring:
			newStatus = enum.PreOrderStatusShipped
		case enum.GHNDeliveryStatusDelivering:
			newStatus = enum.PreOrderStatusInTransit
		case enum.GHNDeliveryStatusDelivered:
			newStatus = enum.PreOrderStatusDelivered
		default:
			zap.L().Info("GHN status does not trigger side effect for PreOrder", zap.String("status", string(ghnStatus)))
			return nil
		}

		// Use system user ID for webhook-triggered state changes
		systemUserID := uuid.Nil
		err = t.MovePreOrderToState(ctx, preOrder.ID, newStatus, systemUserID, nil, nil)
		if err != nil {
			return err
		}
		return nil
	}

	// Handle Order (default)
	order, err := t.orderRepository.GetByCondition(ctx, filter, []string{})
	if err != nil {
		zap.L().Error("Failed to load order from DB by GHN code",
			zap.String("ghn_order_code", ghnCode),
			zap.Error(err))
		return errors.New("Unable to find order by GHN code: " + err.Error())
	}
	//map GHN status to Order status
	var newStatus enum.OrderStatus
	switch ghnStatus {
	case enum.GHNDeliveryStatusStoring:
		newStatus = enum.OrderStatusShipped
	case enum.GHNDeliveryStatusDelivering:
		newStatus = enum.OrderStatusInTransit
	case enum.GHNDeliveryStatusDelivered:
		newStatus = enum.OrderStatusDelivered
	default:
		zap.L().Info("GHN status does not trigger side effect", zap.String("status", string(ghnStatus)))
		return nil
	}

	err = t.MoveOrderToState(ctx, order.ID, newStatus, nil, nil)
	if err != nil {
		return err
	}
	return nil
}

func (t stateTransferService) MovePreOrderToState(ctx context.Context, preOrderID uuid.UUID, targetState enum.PreOrderStatus, updatedBy uuid.UUID, reason, fileURL *string) error {
	preOrder, limitedProduct, actionUser, err := t.lookupPreOrderWithLimitedProductAndUser(ctx, preOrderID, updatedBy)
	if err != nil {
		return err
	}

	// check condition and move preOrder to next state
	err = MovePreOrderStateUsingFSM(preOrder, limitedProduct, actionUser, targetState, reason)
	if err != nil {
		return err
	}

	// proof of delivery file is required when moving to Delivered/Received
	err = preOrderFileAssurance(preOrder, targetState, fileURL)
	if err != nil {
		return err
	}

	// If moving to PreOrdered status and not self-pickup, create GHN order first
	// so we can persist GHNOrderCode together with status in a single DB update
	var ghnOrderCode string
	if targetState == enum.PreOrderStatusPreOrdered && !preOrder.IsSelfPickedUp {
		ghnOrder, ghnErr := t.ghnProxy.CreatePreOrder(ctx, preOrderID)
		if ghnErr != nil {
			zap.L().Error("Failed to create GHN order for pre-order",
				zap.String("preorder_id", preOrderID.String()),
				zap.Error(ghnErr))
			return fmt.Errorf("failed to create GHN order for pre-order: %w", ghnErr)
		}
		if ghnOrder != nil && ghnOrder.OrderCode != "" {
			ghnOrderCode = ghnOrder.OrderCode
			zap.L().Info("Created GHN order for pre-order",
				zap.String("preorder_id", preOrderID.String()),
				zap.String("ghn_order_code", ghnOrderCode))
		}
	}

	err = helper.WithTransaction(ctx, t.uow, func(ctx context.Context, uow irepository.UnitOfWork) error {
		// Update GHN order code if available
		if targetState == enum.PreOrderStatusPreOrdered && ghnOrderCode != "" {
			preOrder.GHNOrderCode = &ghnOrderCode
		}

		if err := uow.PreOrder().Update(ctx, preOrder); err != nil {
			zap.L().Error("Failed to update PreOrder state", zap.String("preorder_id", preOrderID.String()), zap.Error(err))
			return errors.New("failed to update PreOrder state: " + err.Error())
		}

		// schedule to auto update status after opening day
		if targetState == enum.PreOrderStatusPreOrdered {
			// OLD LOGIC
			//openDay := limitedProduct.AvailabilityStartDate
			////isSelfPickedUp := preOrder.IsSelfPickedUp
			////limitedProduct := preOrder.ProductVariant.Product.Limited
			//
			//preOrderOpeningSchedule := &model.Schedule{
			//	ReferenceID:   &preOrder.ID,
			//	Type:          enum.ScheduleTypeOther,
			//	ReferenceType: utils.PtrOrNil(enum.ReferenceTypePreOrderOpening),
			//	ScheduledAt:   openDay,
			//	Status:        enum.ScheduleStatusPending,
			//	RetryCount:    0,
			//	CreatedBy:     utils.DerefPtr(&preOrder.UserID, uuid.Nil),
			//}
			//
			//if err := uow.Schedules().Add(ctx, preOrderOpeningSchedule); err != nil {
			//	zap.L().Error("Failed to create PreOrder opening schedule", zap.String("preorder_id", preOrder.ID.String()), zap.Error(err))
			//	return errors.New("failed to create PreOrder opening schedule: " + err.Error())
			//}
			//
			//// publish to asynq
			//t.publishPreOrderOpeningDelayMessage(ctx, preOrderOpeningSchedule, uow)

			// NEW LOGIC

		}

		// Schedule auto-receive when preorder is delivered (not self-pickup)
		// For self-pickup, we use AwaitingPickup -> Received flow
		if targetState == enum.PreOrderStatusDelivered {
			// Get auto-receive interval from config (default 30 days = 2592000000ms)
			autoReceiveIntervalMs := t.adminConfig.AutoReceivePreOrderIntervalMs
			if autoReceiveIntervalMs <= 0 {
				autoReceiveIntervalMs = 2592000000 // Default to 30 days in milliseconds
			}
			scheduledAt := time.Now().Add(time.Duration(autoReceiveIntervalMs) * time.Millisecond)

			preOrderAutoReceiveSchedule := &model.Schedule{
				ReferenceID:   &preOrder.ID,
				Type:          enum.ScheduleTypeOther,
				ReferenceType: utils.PtrOrNil(enum.ReferenceTypePreOrderAutoReceive),
				ScheduledAt:   scheduledAt,
				Status:        enum.ScheduleStatusPending,
				RetryCount:    0,
				CreatedBy:     utils.DerefPtr(&preOrder.UserID, uuid.Nil),
			}

			if err := uow.Schedules().Add(ctx, preOrderAutoReceiveSchedule); err != nil {
				zap.L().Error("Failed to create PreOrder auto-receive schedule", zap.String("preorder_id", preOrder.ID.String()), zap.Error(err))
				return errors.New("failed to create PreOrder auto-receive schedule: " + err.Error())
			}

			// publish to asynq
			t.publishPreOrderAutoReceiveDelayMessage(ctx, preOrderAutoReceiveSchedule, uow)
		}

		// notification
		go func() {
			ctxBg := context.Background()

			//notifcation
			preorderNotiStatus, err := ConvertPreOrderToNotificationType(preOrder)
			if err != nil {
				zap.L().Error("error when convert preOrder to NotificationType", zap.Error(err))
			}
			payloads, err := notification_builder.BuildPreOrderNotifications(ctxBg, *t.config, t.uow.DB(), preorderNotiStatus, preOrder, actionUser)
			if err != nil {
				zap.L().Error("no notification builder for preorder status", zap.Error(err))
			}
			for _, p := range payloads {
				_, err = t.notificationService.CreateAndPublishNotification(ctxBg, &p)
				if err != nil {
					zap.L().Error("Failed to send notification", zap.Error(err))
				}
			}
		}()

		return nil
	})

	return err
}

// preOrderFileAssurance (Special case): proof of delivery file is required when moving to Delivered/Received
func preOrderFileAssurance(preOrder *model.PreOrder, targetState enum.PreOrderStatus, fileURL *string) error {
	isSelfPick := preOrder.IsSelfPickedUp
	//isStatusDelivered := targetState.String() == enum.PreOrderStatusDelivered.String()
	isStatusReceived := targetState.String() == enum.PreOrderStatusReceived.String()
	if isSelfPick && isStatusReceived {
		if fileURL == nil || *fileURL == "" {
			errMsg := fmt.Sprintf("proof of delivery file is required when transitioning to %s", targetState.String())
			zap.L().Error(errMsg, zap.String("preorder_id", preOrder.ID.String()))
			return errors.New(errMsg)
		}
		preOrder.ConfirmationImage = fileURL
	}
	return nil
}

func (t stateTransferService) MoveTaskToState(ctx context.Context, uow irepository.UnitOfWork, taskID uuid.UUID, targetState enum.TaskStatus, updatedBy uuid.UUID) error {
	if uow == nil {
		uow = t.uow
	}
	return helper.WithTransaction(ctx, uow, func(ctx context.Context, uow irepository.UnitOfWork) error {
		// Use transactional repositories
		taskRepo := uow.Tasks()

		//1. Load current task from DB
		// Preload nested product -> task to have back-reference available ("Products.Task")
		task, err := taskRepo.GetByID(ctx, taskID, []string{"Product", "Product.Task", "Contents", "Contents.Task", "Milestone"})
		if err != nil {
			zap.L().Error("Failed to load task from DB",
				zap.String("task ID", taskID.String()),
				zap.Error(err))
			return errors.New("Unable to find task: " + err.Error())
		}
		// Set updatedBy AFTER successful fetch
		task.UpdatedByID = &updatedBy

		//2. Load task context
		taskCtx := &tasksm.TaskContext{
			State:    tasksm.NewTaskState(task.Status),
			Task:     task,
			Products: task.Product,
			Contents: task.Contents,
		}

		//3. Init target State
		nextState := tasksm.NewTaskState(targetState)
		if nextState == nil {
			zap.L().Error("Invalid target state",
				zap.String("user_id", taskID.String()),
				zap.String("target_state", targetState.String()))
			return errors.New("invalid target state")
		}

		//4. Forward state
		if err := taskCtx.State.Next(taskCtx, nextState); err != nil {
			zap.L().Error("State transition failed",
				zap.String("user_id", taskID.String()),
				zap.String("from", taskCtx.State.Name().String()),
				zap.String("to", targetState.String()),
				zap.Error(err))
			return errors.New("State transition failed: " + err.Error())
		}

		//5. Persist task new state
		task.Status = targetState
		if err := taskRepo.Update(ctx, task); err != nil {
			zap.L().Error("Failed to update task state in DB",
				zap.String("user_id", taskID.String()),
				zap.String("new_state", targetState.String()),
				zap.Error(err))
			return errors.New("Failed to update task state in DB: " + err.Error())
		}
		taskCtx.Task = task // reflect any in-memory changes made by state machine

		// 6. Handle side-effects
		if err := t.handleTaskSideEffects(ctx, uow, taskCtx, task, targetState, updatedBy); err != nil {
			return err
		}

		return nil
	})
}

func (t *stateTransferService) handleTaskSideEffects(
	ctx context.Context, uow irepository.UnitOfWork, taskCtx *tasksm.TaskContext, task *model.Task, targetState enum.TaskStatus, updatedBy uuid.UUID,
) error {
	//1. Cascade UpdatedByID (and any status changes applied by state machine) to product, if any
	// Ensure task back-reference present (if not, assign for safety)
	if taskCtx.Products != nil {
		if taskCtx.Products.Task == nil {
			taskCtx.Products.Task = task
		}
		taskCtx.Products.UpdatedByID = &updatedBy
		if err := uow.Products().Update(ctx, taskCtx.Products); err != nil {
			// Log and continue; do not fail whole operation after task updated
			zap.L().Error("Failed to cascade product update_by",
				zap.String("task_id", task.ID.String()),
				zap.String("product_id", taskCtx.Products.ID.String()),
				zap.Error(err))
		}
	}

	// 2. If the task is moved to "IN_PROGRESS", if milestones are not yet "ON_GOING", move them too
	if targetState == enum.TaskStatusInProgress || targetState == enum.TaskStatusDone {
		tasksInMilestone, totalTasksCount, err := uow.Tasks().GetAll(ctx, func(db *gorm.DB) *gorm.DB {
			return db.Where("milestone_id = ?", task.Milestone.ID).Where("deleted_at IS NULL")
		}, []string{}, 0, 0)
		if err != nil {
			return err
		}
		var (
			totalCompletedTasks                   = 0
			milestoneCompletionPercentage float64 = 0
			milestoneUpdatingStatus               = enum.MilestoneStatusOnGoing
			completedAt                   time.Time
		)
		for _, t := range tasksInMilestone {
			if t.Status == enum.TaskStatusDone {
				totalCompletedTasks++
			}
		}
		if totalTasksCount > 0 {
			milestoneCompletionPercentage = float64(float64(totalCompletedTasks)/float64(totalTasksCount)) * 100
		}
		if int64(totalCompletedTasks) == totalTasksCount && totalTasksCount > 0 {
			milestoneUpdatingStatus = enum.MilestoneStatusCompleted
			completedAt = time.Now()
		}
		updatingFields := map[string]any{
			"status":                milestoneUpdatingStatus,
			"completion_percentage": milestoneCompletionPercentage,
		}
		if !completedAt.IsZero() {
			updatingFields["completed_at"] = completedAt
		}
		uow.Milestones().DB().Model(new(model.Milestone)).
			Where("milestones.id = ?", task.Milestone.ID).
			Updates(updatingFields)
	}
	return nil
}

func (t stateTransferService) MoveProductToState(ctx context.Context, productID uuid.UUID, targetState enum.ProductStatus, updatedBy uuid.UUID) error {
	// Use unit-of-work transaction so we can persist both product and its task atomically
	trx := t.uow.Begin(ctx)
	defer func() {
		if r := recover(); r != nil {
			_ = trx.Rollback()
			zap.L().Error("panic recovered in MoveProductToState", zap.Any("recover", r))
		}
	}()

	productRepo := trx.Products()
	// Make sure we load the Task relation so we can update it
	includes := []string{"Variants", "Limited", "Task"}

	product, err := productRepo.GetByID(ctx, productID, includes)
	if err != nil {
		_ = trx.Rollback()
		zap.L().Error("Failed to load Product from DB",
			zap.String("product ID", productID.String()),
			zap.Error(err))
		return errors.New("Unable to find Product: " + err.Error())
	}
	// Set updatedBy AFTER successful fetch
	product.UpdatedByID = &updatedBy

	// load current product context
	productCtx := &productsm.ProductContext{
		State:   productsm.NewProductState(product.Status),
		Product: *product,
	}

	// init next product State
	nextProductState := productsm.NewProductState(targetState)
	if nextProductState == nil {
		_ = trx.Rollback()
		zap.L().Error("Invalid target state",
			zap.String("target_state", targetState.String()))
		return errors.New("invalid target state")
	}

	//4. Forward state
	if err := productCtx.State.Next(productCtx, nextProductState); err != nil {
		_ = trx.Rollback()
		zap.L().Error("State transition failed",
			zap.String("user_id", productID.String()),
			zap.String("from", productCtx.State.Name().String()),
			zap.String("to", targetState.String()),
			zap.Error(err))
		return errors.New("State transition failed: " + err.Error())
	}

	//5. Save to DB
	product.Status = targetState
	if targetState == enum.ProductStatusActived {
		product.IsActive = true
		if product.Task != nil {
			// Capture local reference to avoid races/mutations and avoid deref in logger
			task := product.Task
			task.Status = enum.TaskStatusDone
			task.UpdatedByID = &updatedBy
			// Use raw SQL update to persist task changes within the same transaction
			res := trx.DB().Exec(`UPDATE tasks SET status = ?, updated_by = ?, updated_at = now() WHERE id = ?`, task.Status, updatedBy.String(), task.ID)
			if res.Error != nil {
				_ = trx.Rollback()
				// Prepare safe task id string for logging
				taskID := task.ID.String()
				zap.L().Error("Failed to update related task state in DB (raw sql)",
					zap.String("product_id", productID.String()),
					zap.String("task_id", taskID),
					zap.Error(res.Error))
				return errors.New("Failed to update related task state in DB: " + res.Error.Error())
			}
			// Ensure a row was actually updated
			if res.RowsAffected == 0 {
				_ = trx.Rollback()
				taskID := task.ID.String()
				zap.L().Error("No task row updated when attempting to set task to DONE",
					zap.String("product_id", productID.String()),
					zap.String("task_id", taskID))
				return errors.New("failed to update task: no rows affected")
			}
		}
	}

	if err := productRepo.Update(ctx, product); err != nil {
		_ = trx.Rollback()
		zap.L().Error("Failed to update product state in DB",
			zap.String("user_id", productID.String()),
			zap.String("new_state", targetState.String()),
			zap.Error(err))
		return errors.New("Failed to update product state in DB: " + err.Error())
	}

	if err := trx.Commit(); err != nil {
		zap.L().Error("Failed to commit transaction in MoveProductToState", zap.Error(err))
		return errors.New("transaction commit failed: " + err.Error())
	}

	// Schedule limited product announcements after successful commit (for LIMITED products only)
	if targetState == enum.ProductStatusActived && product.Type == enum.ProductTypeLimited && product.Limited != nil {
		go t.scheduleLimitedProductAnnouncements(context.Background(), product)
	}

	return nil
}

func (t stateTransferService) MoveMileStoneToState(ctx context.Context, mileStoneID uuid.UUID, targetState enum.MilestoneStatus, updatedBy uuid.UUID) error {
	trx := t.uow.Begin(ctx)
	defer func() {
		if r := recover(); r != nil {
			_ = trx.Rollback()
			zap.L().Error("panic recovered in MoveMileStoneToState", zap.Any("recover", r))
		}
	}()

	milestoneRepo := trx.Milestones()
	// We want tasks (and optionally their products/contents if later cascades rely on them)
	milestone, err := milestoneRepo.GetByID(ctx, mileStoneID, []string{"Tasks", "Tasks.Milestone", "Tasks.Product", "Tasks.Contents"})
	if err != nil {
		_ = trx.Rollback()
		zap.L().Error("Failed to load milestone from DB", zap.String("milestone_id", mileStoneID.String()), zap.Error(err))
		return errors.New("failed to load milestone from DB: " + err.Error())
	}
	//TODO: Set updatedBy AFTER successful fetch -> incase for cascade
	milestone.UpdatedByID = &updatedBy

	// Build milestone context
	var tasks []*model.Task
	if milestone.Tasks != nil {
		tasks = milestone.Tasks
	}
	mCtx := &milestonesm.MilestoneContext{
		State: milestonesm.NewMilestoneState(milestone.Status),
		Tasks: tasks,
	}

	//3. Init target State
	nextState := milestonesm.NewMilestoneState(targetState)
	if nextState == nil {
		_ = trx.Rollback()
		zap.L().Error("Invalid target milestone state", zap.String("milestone_id", mileStoneID.String()), zap.String("target_state", targetState.String()))
		return errors.New("invalid target milestone state")
	}

	// Transition
	//4. Forward state
	if err := mCtx.State.Next(mCtx, nextState); err != nil {
		_ = trx.Rollback()
		zap.L().Error("Milestone state transition failed", zap.String("milestone_id", mileStoneID.String()), zap.String("from", mCtx.State.Name().String()), zap.String("to", targetState.String()), zap.Error(err))
		return errors.New("milestone state transition failed: " + err.Error())
	}

	// Persist milestone
	milestone.Status = targetState
	if err := milestoneRepo.Update(ctx, milestone); err != nil {
		_ = trx.Rollback()
		zap.L().Error("Failed to update milestone state", zap.String("milestone_id", mileStoneID.String()), zap.Error(err))
		return errors.New("failed to update milestone state: " + err.Error())
	}

	//if err := trx.Commit(); err != nil {
	//	zap.L().Error("Milestone transaction commit failed", zap.Error(err))
	//	return errors.New("transaction commit failed: " + err.Error())
	//}
	return nil
}

func (t stateTransferService) MoveCampaignToState(
	ctx context.Context,
	uow irepository.UnitOfWork,
	campaignID uuid.UUID,
	targetState enum.CampaignStatus,
	updatedBy uuid.UUID,
) error {
	//1. Load current task from DB
	campaignRepo := uow.Campaigns()
	campaign, err := campaignRepo.GetByID(ctx, campaignID, []string{"Milestones", "Milestones.Campaign", "Milestones.Tasks", "Milestones.Tasks.Product", "Milestones.Tasks.Contents"})
	if err != nil {
		zap.L().Error("Failed to load campaign", zap.String("campaign_id", campaignID.String()), zap.Error(err))
		return errors.New("failed to load campaign: " + err.Error())
	}
	campaign.UpdatedByID = &updatedBy

	//2. Load task context
	cCtx := &campaignsm.CampaignContext{
		State:      campaignsm.NewCampaignState(campaign.Status),
		Campaign:   campaign,
		MileStones: campaign.Milestones,
	}

	//3. Init target State
	nextState := campaignsm.NewCampaignState(targetState)
	if nextState == nil {
		zap.L().Error("Invalid target campaign state", zap.String("campaign_id", campaignID.String()), zap.String("target_state", targetState.String()))
		return errors.New("invalid target campaign state")
	}

	//4. Forward state
	if err := cCtx.State.Next(cCtx, nextState); err != nil {
		zap.L().Error("Campaign state transition failed", zap.String("campaign_id", campaignID.String()), zap.String("from", cCtx.State.Name().String()), zap.String("to", targetState.String()), zap.Error(err))
		return errors.New("campaign state transition failed: " + err.Error())
	}

	//5. Persist task new state
	campaign = cCtx.Campaign // reflect any in-memory changes made by state machine
	campaign.Status = targetState
	if err := campaignRepo.Update(ctx, campaign); err != nil {
		zap.L().Error("Failed to persist campaign state", zap.String("campaign_id", campaignID.String()), zap.Error(err))
		return errors.New("failed to persist campaign state: " + err.Error())
	}

	return nil
}

func (t stateTransferService) MoveContractToState(ctx context.Context, trx irepository.UnitOfWork, contractID uuid.UUID, targetState enum.ContractStatus, updatedBy uuid.UUID) error {
	contractRepo := trx.Contracts()
	campaignRepo := trx.Campaigns()
	milestoneRepo := trx.Milestones()
	taskRepo := trx.Tasks()
	productRepo := trx.Products()

	contract, err := contractRepo.GetByID(ctx, contractID, []string{"Brand", "Campaign"})
	if err != nil {
		zap.L().Error("Failed to load contract", zap.String("contract_id", contractID.String()), zap.Error(err))
		return errors.New("failed to load contract: " + err.Error())
	} else if contract == nil {
		zap.L().Error("Contract not found", zap.String("contract_id", contractID.String()))
		return errors.New("contract not found")
	}
	oldStatus := contract.Status

	if oldStatus == targetState {
		zap.L().Info("Contract already in target state, no action taken", zap.String("contract_id", contractID.String()), zap.String("state", targetState.String()))
		return nil
	}

	// Preload deeper campaign tree if contract has a campaign
	if contract.Campaign != nil {
		camp, err2 := campaignRepo.GetByID(ctx, contract.Campaign.ID, []string{"Milestones", "Milestones.Tasks", "Milestones.Tasks.Product", "Milestones.Tasks.Contents"})
		if err2 == nil {
			contract.Campaign = camp
		}
	}

	cCtx := &contractsm.ContractContext{State: contractsm.NewContractState(contract.Status), Campaign: contract.Campaign}
	nextState := contractsm.NewContractState(targetState)
	if nextState == nil {
		zap.L().Error("Invalid contract target state", zap.String("contract_id", contractID.String()), zap.String("target_state", targetState.String()))
		return errors.New("invalid target contract state")
	}

	if err := cCtx.State.Next(cCtx, nextState); err != nil {
		zap.L().Error("Contract state transition failed", zap.String("contract_id", contractID.String()), zap.String("from", cCtx.State.Name().String()), zap.String("to", targetState.String()), zap.Error(err))
		return errors.New("contract state transition failed: " + err.Error())
	}
	// Override in case of any adjustments to the targetState
	targetState = cCtx.State.Name()

	contract.Status = targetState
	filterQuery := func(db *gorm.DB) *gorm.DB {
		return db.Where("id = ?", contractID)
	}
	if err := contractRepo.UpdateByCondition(ctx, filterQuery, map[string]any{"status": targetState}); err != nil {
		zap.L().Error("Failed updating contract", zap.String("contract_id", contractID.String()), zap.Error(err))
		return errors.New("failed to update contract: " + err.Error())
	}

	// Side-effects after state transitioning
	switch targetState {
	case enum.ContractStatusBrandPenaltyPaid, enum.ContractStatusKOLRefundApproved:
		if targetState == enum.ContractStatusBrandPenaltyPaid {
			zap.L().Info("Brand penalty paid - preparing for contract termination",
				zap.String("contract_id", contractID.String()))
		} else {
			zap.L().Info("KOL refund approved - preparing for contract termination",
				zap.String("contract_id", contractID.String()))
		}

		// Auto-transition to TERMINATED
		zap.L().Info("Auto-transitioning contract to TERMINATED from "+targetState.String(), zap.String("contract_id", contractID.String()))
		if err := t.MoveContractToState(ctx, trx, contractID, enum.ContractStatusTerminated, updatedBy); err != nil {
			return err
		}

	// Terminate contract -> cascade cancel related campaign, milestones, tasks, contents, and products
	case enum.ContractStatusTerminated, enum.ContractStatusKOLViolated, enum.ContractStatusBrandViolated:
		zap.L().Info("Contract terminated or violated - cascading cancellations",
			zap.String("status", targetState.String()),
			zap.String("contract_id", contractID.String()))

		if contract.Campaign == nil {
			break
		}
		camp := contract.Campaign
		if err := milestoneRepo.UpdateByCondition(ctx, func(db *gorm.DB) *gorm.DB {
			return db.Where("campaign_id = ? AND status <> ?", camp.ID, enum.MilestoneStatusCancelled)
		}, map[string]any{"status": enum.MilestoneStatusCancelled}); err != nil {
			zap.L().Error("Failed cancel milestones (contract)", zap.String("contract_id", contractID.String()), zap.Error(err))
			return errors.New("cascade milestone cancel failed: " + err.Error())
		}

		// Batch cancel milestones
		if err := milestoneRepo.UpdateByCondition(ctx, func(db *gorm.DB) *gorm.DB {
			return db.Where("campaign_id = ? AND status <> ?", camp.ID, enum.MilestoneStatusCancelled)
		}, map[string]any{"status": enum.MilestoneStatusCancelled}); err != nil {
			zap.L().Error("Failed cancel milestones (contract)", zap.String("contract_id", contractID.String()), zap.Error(err))
			return errors.New("cascade milestone cancel failed: " + err.Error())
		}
		// Batch cancel tasks
		if err := taskRepo.UpdateByCondition(ctx, func(db *gorm.DB) *gorm.DB {
			return db.Where("milestone_id = ? AND status <> ?", camp.ID, enum.TaskStatusCancelled)
		}, map[string]any{"status": enum.TaskStatusCancelled}); err != nil {
			zap.L().Error("Failed cancel tasks (contract)", zap.String("contract_id", contractID.String()), zap.Error(err))
			return errors.New("cascade task cancel failed: " + err.Error())
		}
		// Batch inactivate products
		if err := productRepo.UpdateByCondition(ctx, func(db *gorm.DB) *gorm.DB {
			var taskIDs []any
			_ = trx.DB().Model(new(model.Task)).Where("milestone_id = ?", camp.ID).Pluck("id", &taskIDs).Error
			return db.
				Where("products.task_id IN (?) AND products.status <> ?", taskIDs, enum.ProductStatusInactived)
		}, map[string]any{"status": enum.ProductStatusInactived}); err != nil {
			zap.L().Error("Failed inactivate products (contract)", zap.String("contract_id", contractID.String()), zap.Error(err))
			return errors.New("cascade product inactivate failed: " + err.Error())
		}

		// Cancel contents associated with tasks in this campaign
		// Use custom repository method to get content IDs (bypasses 100-record limit)
		contentIDs, err := t.contentRepository.GetContentIDsByCampaignID(ctx, camp.ID, enum.ContentStatusCancelled)
		if err != nil {
			zap.L().Error("Failed to fetch content IDs for cancellation", zap.String("contract_id", contractID.String()), zap.Error(err))
			// Don't fail the entire transaction - log warning and continue
			zap.L().Warn("Continuing contract termination despite content fetch failure")
		} else {
			for _, contentID := range contentIDs {
				if err := t.MoveContentToState(ctx, trx, contentID, enum.ContentStatusCancelled, updatedBy); err != nil {
					zap.L().Warn("Failed to cancel content",
						zap.String("content_id", contentID.String()),
						zap.Error(err))
					// Continue with other contents, don't fail entire operation
				}
			}
			zap.L().Info("Cancelled contents due to contract termination",
				zap.String("contract_id", contractID.String()),
				zap.Int("content_count", len(contentIDs)))
		}

		// Batch expire affiliate links associated with this contract
		if err := trx.AffiliateLinks().UpdateByCondition(ctx, func(db *gorm.DB) *gorm.DB {
			return db.Where("contract_id = ? AND status = ?", contractID, enum.AffiliateLinkStatusActive)
		}, map[string]any{"status": enum.AffiliateLinkStatusExpired}); err != nil {
			zap.L().Error("Failed to expire affiliate links (contract)", zap.String("contract_id", contractID.String()), zap.Error(err))
			// Don't fail the entire transaction - log warning and continue
			zap.L().Warn("Continuing contract termination despite affiliate link update failure")
		} else {
			zap.L().Info("Expired affiliate links due to contract termination", zap.String("contract_id", contractID.String()))
		}

		// Batch terminate Contract Payment Transactions associated with this contract
		if err := trx.ContractPayments().UpdateByCondition(ctx, func(db *gorm.DB) *gorm.DB {
			return db.Where("contract_id = ? AND status <> ?", contractID, enum.ContractPaymentStatusTerminated)
		}, map[string]any{"status": enum.ContractPaymentStatusTerminated}); err != nil {
			zap.L().Error("Failed to terminate contract payment transactions (contract)", zap.String("contract_id", contractID.String()), zap.Error(err))
			// Don't fail the entire transaction - log warning and continue
			zap.L().Warn("Continuing contract termination despite contract payment transaction update failure")
		} else {
			zap.L().Info("Terminated contract payment transactions due to contract termination", zap.String("contract_id", contractID.String()))
		}

		// Reflect memory
		camp.Status = enum.CampaignCancelled
		for _, ms := range camp.Milestones {
			if ms == nil {
				continue
			}
			ms.Status = enum.MilestoneStatusCancelled

			if ms.Tasks == nil {
				continue
			}
			for _, tk := range ms.Tasks {
				if tk == nil {
					continue
				}
				tk.Status = enum.TaskStatusCancelled
				//for _, p := range tk.Products {
				if tk.Product != nil {
					tk.Product.Status = enum.ProductStatusInactived
				}
				//}
			}
		}

	case enum.ContractStatusKOLProofRejected:
		zap.L().Info("KOL proof rejected - KOL may resubmit",
			zap.String("contract_id", contractID.String()))

		// Send notification to KOL about proof rejection
		go func() {
			ctxBg := context.Background()
			contract, err := t.contractRepository.GetByID(ctxBg, contractID, nil)
			if err != nil {
				zap.L().Error("Failed to get contract for proof rejection notification", zap.Error(err))
				return
			}

			if contract.RepresentativeEmail != nil {
				// Find User by email
				user, err := t.userRepository.GetByCondition(ctxBg, func(db *gorm.DB) *gorm.DB {
					return db.Where("email = ?", *contract.RepresentativeEmail)
				}, nil)

				if err == nil && user != nil {
					contractNumber := "N/A"
					if contract.ContractNumber != nil {
						contractNumber = *contract.ContractNumber
					}

					templateData := map[string]any{
						"KOLName":        user.FullName,
						"ContractNumber": contractNumber,
						"SupportLink":    t.config.Server.BaseFrontendURL + "/support",
						"CurrentYear":    time.Now().Year(),
					}

					req := requests.PublishNotificationRequest{
						UserID:            user.ID,
						Title:             "Refund Proof Rejected",
						Body:              fmt.Sprintf("Your refund proof for contract %s has been rejected. Please review and resubmit.", contractNumber),
						Data:              map[string]string{"contract_id": contractID.String()},
						Types:             []enum.NotificationType{enum.NotificationTypeEmail, enum.NotificationTypeInApp},
						EmailTemplateName: utils.PtrOrNil("kol_proof_rejected"),
						EmailTemplateData: templateData,
					}
					if _, err = t.notificationService.CreateAndPublishNotification(ctxBg, &req); err != nil {
						zap.L().Error("Failed to send proof rejection notification", zap.Error(err))
					}
				} else {
					zap.L().Warn("Could not find KOL user by representative email",
						zap.String("email", *contract.RepresentativeEmail),
						zap.Error(err))
				}
			}
		}()

	case enum.ContractStatusApproved:
		// Create contract payment based on the contract by publishing to RabbitMQ exchange
		contractCreatePaymentProducer, err := t.rabbitMQ.GetProducer("contract-create-payment-producer")
		if err != nil {
			zap.L().Error("Failed to get contract create payment producer", zap.Error(err))
			return errors.New("failed to get contract create payment producer: " + err.Error())
		}
		message := &consumers.ContractCreatePaymentMessage{
			UserID:     updatedBy.String(),
			ContractID: contractID.String(),
		}
		err = contractCreatePaymentProducer.PublishJSON(ctx, message)
		if err != nil {
			zap.L().Error("Failed to publish contract create payment message", zap.Error(err))
			return errors.New("failed to publish contract create payment message: " + err.Error())
		}

		zap.L().Info("Successfully published contract create payment message",
			zap.String("contract_id", contractID.String()),
			zap.String("user_id", updatedBy.String()))

	default:
		zap.L().Debug("There are no side-effects to be applied to the contract after transitioning",
			zap.String("contract_id", contractID.String()),
			zap.String("old_status", oldStatus.String()),
			zap.String("new_status", nextState.Name().String()),
		)
	}

	return nil
}

// region: ============== Content State Transfer ==============

func (t stateTransferService) MoveContentToState(ctx context.Context, uow irepository.UnitOfWork, contentID uuid.UUID, targetState enum.ContentStatus, updatedBy uuid.UUID) error {
	// Use transactional repository from UnitOfWork
	contentRepo := uow.Contents()

	// 1. Load current content from DB with relationships
	content, err := contentRepo.GetByID(ctx, contentID, []string{"ContentChannels", "ContentChannels.Channel"})
	if err != nil {
		zap.L().Error("Failed to load content from DB",
			zap.String("content_id", contentID.String()),
			zap.Error(err))
		return errors.New("unable to find content: " + err.Error())
	}

	// 2. Load content context for FSM
	currentState, err := contentsm.NewContentState(content.Status)
	if err != nil {
		zap.L().Error("Failed to create current state",
			zap.String("content_id", contentID.String()),
			zap.String("current_status", string(content.Status)),
			zap.Error(err))
		return errors.New("failed to create current state: " + err.Error())
	}

	contentCtx := &contentsm.ContentContext{
		State:           currentState,
		ContentChannels: content.ContentChannels,
	}

	// 3. Initialize target state
	nextState, err := contentsm.NewContentState(targetState)
	if err != nil {
		zap.L().Error("Invalid target state",
			zap.String("content_id", contentID.String()),
			zap.String("target_state", string(targetState)),
			zap.Error(err))
		return errors.New("invalid target state: " + err.Error())
	}

	// 4. Validate and forward state through FSM
	if err := contentCtx.State.Next(contentCtx, nextState); err != nil {
		zap.L().Error("State transition failed",
			zap.String("content_id", contentID.String()),
			zap.String("from", string(contentCtx.State.Name())),
			zap.String("to", string(targetState)),
			zap.Error(err))
		return errors.New("state transition failed: " + err.Error())
	}

	// 5. Persist new state to database using transactional repository
	content.Status = targetState
	if err := contentRepo.Update(ctx, content); err != nil {
		zap.L().Error("Failed to update content state in DB",
			zap.String("content_id", contentID.String()),
			zap.String("new_state", string(targetState)),
			zap.Error(err))
		return errors.New("failed to update content state in DB: " + err.Error())
	}

	// 6. Side-effects:
	if err := t.handleContentSideEffects(ctx, uow, content, currentState, contentCtx, targetState, updatedBy); err != nil {
		zap.L().Error("Failed to handle content side-effects", zap.Error(err))
		return err
	}

	zap.L().Info("Content state transition successful",
		zap.String("content_id", contentID.String()),
		zap.String("new_state", string(targetState)),
		zap.String("updated_by", updatedBy.String()))

	return nil
}

func (t *stateTransferService) handleContentSideEffects(
	ctx context.Context, uow irepository.UnitOfWork, content *model.Content, currentState contentsm.ContentState,
	_ *contentsm.ContentContext, targetState enum.ContentStatus, updatedBy uuid.UUID,
) error {
	// 1. Expire affiliate links if content is unpublished
	// If content is moved away from POSTED status, expire associated affiliate links
	if currentState.Name() == enum.ContentStatusPosted && targetState != enum.ContentStatusPosted {
		affiliateLinkRepo := uow.AffiliateLinks()
		if err := affiliateLinkRepo.UpdateByCondition(ctx, func(db *gorm.DB) *gorm.DB {
			return db.Where("content_id = ? AND status = ?", content.ID, enum.AffiliateLinkStatusActive)
		}, map[string]any{"status": enum.AffiliateLinkStatusExpired}); err != nil {
			zap.L().Error("Failed to expire affiliate links (content unpublish)",
				zap.String("content_id", content.ID.String()),
				zap.Error(err))
			// Don't fail the entire transaction - log warning and continue
			zap.L().Warn("Continuing content state change despite affiliate link update failure")
		} else {
			zap.L().Info("Expired affiliate links due to content unpublish",
				zap.String("content_id", content.ID.String()),
				zap.String("new_status", string(targetState)))
		}
	}

	// 2. Move Task state to DONE if content is POSTED and has associated Task
	if targetState == enum.ContentStatusPosted && content.TaskID != nil {
		if err := t.MoveTaskToState(ctx, uow, *content.TaskID, enum.TaskStatusDone, updatedBy); err != nil {
			zap.L().Error("Failed to move associated Task to DONE (content posted)",
				zap.String("content_id", content.ID.String()),
				zap.Error(err))
			return err
		}
	}

	// 3. Handle CANCELLED state side effects
	if targetState == enum.ContentStatusCancelled {
		// 3a. Cancel any pending schedules for this content's channels
		for _, cc := range content.ContentChannels {
			if cc == nil {
				continue
			}
			if err := t.scheduleService.CancelByReferenceID(ctx, cc.ID, updatedBy); err != nil {
				zap.L().Warn("Failed to cancel schedule for content channel",
					zap.String("content_channel_id", cc.ID.String()),
					zap.Error(err))
				// Continue with other channels, don't fail entire operation
			}
		}

		// 3b. Trigger social media unpublish for POSTED content (placeholder)
		if currentState.Name() == enum.ContentStatusPosted {
			if err := t.triggerSocialMediaUnpublish(ctx, uow, content); err != nil {
				zap.L().Warn("Failed to trigger social media unpublish",
					zap.String("content_id", content.ID.String()),
					zap.Error(err))
				// Continue - don't fail the cancellation
			}
		}
	}

	return nil
}

// triggerSocialMediaUnpublish handles unpublishing content from social media platforms.
// This is a placeholder for future implementation.
//
// TODO: Implement actual unpublish logic for each platform:
// - Facebook: Use Graph API to delete/unpublish post
// - TikTok: Use Content Posting API to delete video
// - Instagram: Use Graph API to delete post
//
// Implementation considerations:
// - Each platform has different unpublish endpoints
// - Some platforms may not support programmatic unpublish
// - May need to queue async unpublish jobs via RabbitMQ
// - Should update content_channel.external_post_id and related fields
func (t *stateTransferService) triggerSocialMediaUnpublish(
	_ context.Context,
	_ irepository.UnitOfWork,
	content *model.Content,
) error {
	// Placeholder: Log the unpublish request for now
	zap.L().Info("Social media unpublish triggered (not implemented)",
		zap.String("content_id", content.ID.String()),
		zap.String("content_type", string(content.Type)),
		zap.Int("channel_count", len(content.ContentChannels)))

	// Future implementation:
	// for _, cc := range content.ContentChannels {
	//     if cc.ExternalPostID != nil && *cc.ExternalPostID != "" {
	//         switch strings.ToLower(cc.Channel.Code) {
	//         case "facebook":
	//             err := t.facebookProxy.UnpublishPost(ctx, *cc.ExternalPostID)
	//         case "tiktok":
	//             err := t.tiktokProxy.DeleteVideo(ctx, *cc.ExternalPostID)
	//         }
	//     }
	// }

	return nil // No-op for now
}

// endregion

// region: ============== Payment Transaction State Transfer ==============

func (t stateTransferService) MovePaymentTransactionToState(ctx context.Context, uow irepository.UnitOfWork, transactionID uuid.UUID, targetState enum.PaymentTransactionStatus, updatedBy uuid.UUID) error {
	// Use transactional repository from UnitOfWork
	transactionRepo := uow.PaymentTransaction()
	contractPaymentRepo := uow.ContractPayments()
	orderRepo := uow.Order()
	preorderRepo := uow.PreOrder()

	// 1. Load payment transaction with reference entity
	transaction, err := transactionRepo.GetByID(ctx, transactionID, nil)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			zap.L().Error("Payment transaction not found",
				zap.String("transaction_id", transactionID.String()))
			return errors.New("payment transaction not found")
		}
		zap.L().Error("Failed to load payment transaction from DB",
			zap.String("transaction_id", transactionID.String()),
			zap.Error(err))
		return errors.New("unable to find payment transaction: " + err.Error())
	} else if transaction == nil {
		zap.L().Error("Payment transaction not found",
			zap.String("transaction_id", transactionID.String()))
		return errors.New("payment transaction not found")
	}

	if transaction.Status == targetState {
		zap.L().Info("Payment transaction already in target state, no action taken",
			zap.String("transaction_id", transactionID.String()),
			zap.String("state", targetState.String()))
		return nil
	}

	// 2. Load current state for FSM
	currentState, err := paymenttransactionsm.NewPaymentTransactionState(transaction.Status)
	if err != nil {
		zap.L().Error("Failed to create current state",
			zap.String("transaction_id", transactionID.String()),
			zap.String("current_status", string(transaction.Status)),
			zap.Error(err))
		return errors.New("failed to create current state: " + err.Error())
	}

	// 3. Load reference entities based on type
	transactionCtx := &paymenttransactionsm.PaymentTransactionContext{
		State:         currentState,
		ReferenceType: transaction.ReferenceType,
	}

	switch transaction.ReferenceType {
	case enum.PaymentTransactionReferenceTypeContractPayment:
		var contractPayment *model.ContractPayment
		contractPayment, err = contractPaymentRepo.GetByID(ctx, transaction.ReferenceID, nil)
		if err != nil {
			zap.L().Error("Failed to load contract payment",
				zap.String("contract_payment_id", transaction.ReferenceID.String()),
				zap.Error(err))
			return errors.New("unable to find contract payment: " + err.Error())
		}
		transactionCtx.ContractPayment = contractPayment

	case enum.PaymentTransactionReferenceTypeOrder:
		var order *model.Order
		includes := []string{"OrderItems", "OrderItems.Variant", "OrderItems.Variant.Product"}
		order, err = orderRepo.GetByID(ctx, transaction.ReferenceID, includes)
		if err != nil {
			zap.L().Error("Failed to load order",
				zap.String("order_id", transaction.ReferenceID.String()),
				zap.Error(err))
			return errors.New("unable to find order: " + err.Error())
		}
		transactionCtx.Order = order

	case enum.PaymentTransactionReferenceTypePreOrder:
		var preorder *model.PreOrder
		includes := []string{"ProductVariant", "ProductVariant.Product", "ProductVariant.Product.Limited"}
		preorder, err = preorderRepo.GetByID(ctx, transaction.ReferenceID, includes)
		if err != nil {
			zap.L().Error("Failed to load pre-order",
				zap.String("preorder_id", transaction.ReferenceID.String()),
				zap.Error(err))
			return errors.New("unable to find pre-order: " + err.Error())
		}
		transactionCtx.PreOrder = preorder

	case enum.PaymentTransactionReferenceTypeContractViolation:
		var violation *model.ContractViolation
		violation, err = t.contractViolationRepository.GetByID(ctx, transaction.ReferenceID, []string{"Contract"})
		if err != nil {
			zap.L().Error("Failed to load contract violation",
				zap.String("violation_id", transaction.ReferenceID.String()),
				zap.Error(err))
			return errors.New("unable to find contract violation: " + err.Error())
		}
		transactionCtx.ContractViolation = violation
	}

	// 4. Initialize target state
	nextState, err := paymenttransactionsm.NewPaymentTransactionState(targetState)
	if err != nil {
		zap.L().Error("Invalid target state",
			zap.String("transaction_id", transactionID.String()),
			zap.String("target_state", string(targetState)),
			zap.Error(err))
		return errors.New("invalid target state: " + err.Error())
	}

	// 5. Validate and forward state through FSM
	if err := transactionCtx.State.Next(transactionCtx, nextState); err != nil {
		zap.L().Error("State transition failed",
			zap.String("transaction_id", transactionID.String()),
			zap.String("from", string(transactionCtx.State.Name())),
			zap.String("to", string(targetState)),
			zap.Error(err))
		return errors.New("state transition failed: " + err.Error())
	}

	// 6. Persist new state to database
	transaction.Status = targetState
	if err := transactionRepo.Update(ctx, transaction); err != nil {
		zap.L().Error("Failed to update payment transaction state in DB",
			zap.String("transaction_id", transactionID.String()),
			zap.String("new_state", string(targetState)),
			zap.Error(err))
		return errors.New("failed to update payment transaction state in DB: " + err.Error())
	}

	// 7. Handle side effects based on reference type and target state
	if err := t.handlePaymentTransactionSideEffects(ctx, uow, transactionCtx, targetState, updatedBy); err != nil {
		zap.L().Error("Failed to handle payment transaction side effects",
			zap.String("transaction_id", transactionID.String()),
			zap.String("target_state", string(targetState)),
			zap.Error(err))
		return errors.New("failed to handle side effects: " + err.Error())
	}

	zap.L().Info("Payment transaction state transition successful",
		zap.String("transaction_id", transactionID.String()),
		zap.String("new_state", string(targetState)),
		zap.String("reference_type", string(transaction.ReferenceType)))

	return nil
}

func (t stateTransferService) MoveOrderToState(ctx context.Context, orderID uuid.UUID, targetState enum.OrderStatus, updatedUserID *uuid.UUID, note *string) error {

	err := helper.WithTransaction(ctx, t.uow, func(ctx context.Context, uow irepository.UnitOfWork) error {
		// 1) Load order with items and variants + product
		includes := []string{"OrderItems", "OrderItems.Variant", "OrderItems.Variant.Product"}
		order, err := uow.Order().GetByID(ctx, orderID, includes)
		if err != nil {
			zap.L().Error("Failed to load order for confirm", zap.Error(err))
			return fmt.Errorf("failed to load order: %w", err)
		}

		if order == nil {
			return errors.New("order not found")
		}

		// 1) check if staff's censor time pass initialTime?
		isCurrentStatePerfomedByCustomer := order.Status.String() == enum.OrderStatusPaid.String()
		standByMinutes := t.adminConfig.CensorshipIntervalMinutes
		isAllow := order.UpdatedAt.Add(time.Duration(standByMinutes) * time.Minute).After(time.Now())
		if isCurrentStatePerfomedByCustomer && isAllow {
			msg := fmt.Sprintf("You can only allow to do this action after %d mins after user action, remaining time: %s", standByMinutes, time.Until(order.UpdatedAt.Add(time.Duration(standByMinutes)*time.Minute)).String())
			return errors.New(msg)
		}

		var updatedBy *model.User
		if updatedUserID == nil {
			updatedBy = &model.User{
				ID:       uuid.UUID{},
				FullName: t.adminConfig.SystemName,
				Email:    t.adminConfig.SystemEmail,
			}
		} else {
			updatedBy, err = uow.Users().GetByID(ctx, updatedUserID, []string{})
			if err != nil {
				return err
			}
		}

		// 2) Handle order side effects
		err = t.handleOrderStatusSideEffect(ctx, uow, nil, targetState, order, updatedBy, note)
		if err != nil {
			return err
		}

		// 3) Validate state transition using state machine
		err = MoveOrderStateUsingFSM(order, updatedBy, targetState, note)
		if err != nil {
			return err
		}

		// 4) If order Status is Confirmed, create GHN Order first so we can persist GHNOrderCode together with status in a single DB update
		// But if its mark as SELF PICK UP we skip GHN order creation
		var ghnOrderCode string
		if targetState == enum.OrderStatusConfirmed && !order.IsSelfPickedUp {
			var ghnOrder *dtos.CreatedGHNOrderResponse
			ghnOrder, err = t.ghnProxy.CreateOrder(ctx, order.ID)
			if err != nil {
				zap.L().Error("Failed to create GHN order", zap.Error(err))
				return fmt.Errorf("failed to create GHN order: %w", err)
			}
			ghnOrderCode = ghnOrder.OrderCode
		}

		// 5) Persist new state (and GHNOrderCode if any) to database in one update
		order.Status = targetState
		if targetState == enum.OrderStatusConfirmed && ghnOrderCode != "" {
			order.GHNOrderCode = &ghnOrderCode
		}

		if err = uow.Order().Update(ctx, order); err != nil {
			zap.L().Error("Failed to update order state", zap.Error(err))
			return fmt.Errorf("failed to update order state: %w", err)
		}

		// 6) send notification
		orderNotiStatus, err := ConvertToNotificationType(order)
		if err != nil {
			return err
		}

		payloads, err := notification_builder.BuildOrderNotifications(ctx, *t.config, t.uow.DB(), orderNotiStatus, order, updatedBy)
		if err != nil {
			zap.L().Debug("no notification builder for order status", zap.Error(err))
		}

		for _, p := range payloads {
			_, err = t.notificationService.CreateAndPublishNotification(ctx, &p)
			if err != nil {
				zap.L().Error("Failed to send notification", zap.Error(err))
			}
		}
		return nil
	})

	return err

}

// handlePaymentTransactionSideEffects handles cascading updates based on payment status
func (t stateTransferService) handlePaymentTransactionSideEffects(
	ctx context.Context,
	uow irepository.UnitOfWork,
	transactionCtx *paymenttransactionsm.PaymentTransactionContext,
	targetState enum.PaymentTransactionStatus,
	updatedBy uuid.UUID,
) error {
	switch transactionCtx.ReferenceType {
	case enum.PaymentTransactionReferenceTypeContractPayment:
		return t.handleContractPaymentSideEffect(ctx, uow, transactionCtx.ContractPayment, targetState, updatedBy)
	case enum.PaymentTransactionReferenceTypeOrder:
		return t.handleOrderSideEffect(ctx, uow, transactionCtx.Order, targetState, updatedBy)
	case enum.PaymentTransactionReferenceTypePreOrder:
		return t.handlePreOrderSideEffect(ctx, uow, transactionCtx.PreOrder, targetState, updatedBy)
	case enum.PaymentTransactionReferenceTypeContractViolation:
		return t.handleContractViolationSideEffect(ctx, uow, transactionCtx.ContractViolation, targetState, updatedBy)
	default:
		zap.L().Warn("Unknown reference type, skipping side effects",
			zap.String("reference_type", string(transactionCtx.ReferenceType)))
		return nil
	}
}

// handleContractPaymentSideEffect updates contract payment status based on payment transaction status
func (t stateTransferService) handleContractPaymentSideEffect(
	ctx context.Context,
	uow irepository.UnitOfWork,
	contractPayment *model.ContractPayment,
	transactionStatus enum.PaymentTransactionStatus,
	updatedBy uuid.UUID,
) error {
	if contractPayment == nil {
		return errors.New("contract payment is nil")
	}

	contractPaymentRepo := uow.ContractPayments()
	var newStatus enum.ContractPaymentStatus

	switch transactionStatus {
	case enum.PaymentTransactionStatusCompleted:
		newStatus = enum.ContractPaymentStatusPaid
		zap.L().Info("Updating contract payment to PAID",
			zap.String("contract_payment_id", contractPayment.ID.String()))

		// Update contract to ACTIVE if the contract payment is deposit
		if contractPayment.IsDeposit {
			err := t.MoveContractToState(ctx, uow, contractPayment.ContractID, enum.ContractStatusActive, updatedBy)
			if err != nil {
				zap.L().Error("Failed to update contract to ACTIVE after deposit payment",
					zap.String("contract_id", contractPayment.ContractID.String()),
					zap.Error(err))
				return errors.New("failed to update contract to ACTIVE: " + err.Error())
			}
		}
		// Update the next contract payment of a contract periodStart date to the current date
		// trigger recalculation of payment amount based on actual performance from the current time
		nextPayment, err := t.contractPaymentRepository.GetNextUnpaidContractPaymentFromCurrentPaymentID(ctx, contractPayment.ID)
		if err == nil && nextPayment != nil {
			curr := time.Now()
			nextPayment.PeriodStart = utils.PtrOrNil(time.Date(curr.Year(), curr.Month(), curr.Day(), 0, 0, 0, 0, curr.Location()))
			if err = contractPaymentRepo.Update(ctx, nextPayment); err != nil {
				zap.L().Error("Failed to update next contract payment period start date to current date",
					zap.String("current_payment_id", contractPayment.ID.String()),
					zap.String("next_payment_id", nextPayment.ID.String()),
					zap.Error(err))
				// Not returning error for side effect failure
			}
		}

	case enum.PaymentTransactionStatusFailed,
		enum.PaymentTransactionStatusCancelled,
		enum.PaymentTransactionStatusExpired:
		newStatus = enum.ContractPaymentStatusPending
		zap.L().Info("Reverting contract payment to PENDING",
			zap.String("contract_payment_id", contractPayment.ID.String()),
			zap.String("transaction_status", string(transactionStatus)))

		// Unlock payment amount for AFFILIATE/CO_PRODUCING contracts
		// This allows the amount to be recalculated on the next GET request
		if contractPayment.LockedAmount != nil {
			contractPayment.LockedAmount = nil
			contractPayment.LockedAt = nil
			contractPayment.LockedClicks = nil
			contractPayment.LockedRevenue = nil
			zap.L().Info("Unlocked payment amount after failure",
				zap.String("contract_payment_id", contractPayment.ID.String()))
		}

	default:
		// PENDING or other statuses - no change needed
		zap.L().Debug("No contract payment status change needed",
			zap.String("transaction_status", string(transactionStatus)))
		return nil
	}

	// Update contract payment status
	contractPayment.Status = newStatus
	if err := contractPaymentRepo.Update(ctx, contractPayment); err != nil {
		zap.L().Error("Failed to update contract payment status",
			zap.String("contract_payment_id", contractPayment.ID.String()),
			zap.String("new_status", string(newStatus)),
			zap.Error(err))
		return errors.New("failed to update contract payment status: " + err.Error())
	}

	zap.L().Info("Contract payment status updated successfully",
		zap.String("contract_payment_id", contractPayment.ID.String()),
		zap.String("new_status", string(newStatus)))

	return nil
}

// handleOrderSideEffect updates order status based on payment transaction status
// This handle side effect of Transactions onto Order
func (t stateTransferService) handleOrderSideEffect(
	ctx context.Context,
	uow irepository.UnitOfWork,
	order *model.Order,
	transactionStatus enum.PaymentTransactionStatus,
	userID uuid.UUID,
) error {
	if order == nil {
		return errors.New("order is nil")
	}

	orderRepo := uow.Order()
	var newStatus enum.OrderStatus

	switch transactionStatus {
	// Payment paid -> mark order as PAID
	case enum.PaymentTransactionStatusCompleted:
		//Update Order to Confirm and handle the
		newStatus = enum.OrderStatusPaid
		zap.L().Info("Updating order to OrderStatusPaid (payment completed)",
			zap.String("order_id", order.ID.String()))

	case enum.PaymentTransactionStatusFailed,
		enum.PaymentTransactionStatusCancelled,
		enum.PaymentTransactionStatusExpired:

		newStatus = enum.OrderStatusCancelled

		zap.L().Info("Keeping/reverting order to CANCELLED",
			zap.String("order_id", order.ID.String()),
			zap.String("transaction_status", string(transactionStatus)))

		// Regain stock for LIMITED orders and persist per-variant
		if order.OrderType == enum.ProductTypeLimited.String() {
			variantRepo := uow.ProductVariant()
			for _, item := range order.OrderItems {
				oldStock := 0
				if item.Variant.CurrentStock != nil {
					oldStock = *item.Variant.CurrentStock
				}
				regainStock := item.Quantity
				newStock := oldStock + regainStock
				item.Variant.CurrentStock = &newStock

				if err := variantRepo.Update(ctx, &item.Variant); err != nil {
					zap.L().Error("Failed to persist regained stock for variant",
						zap.String("variant_id", item.Variant.ID.String()),
						zap.Error(err))
					return errors.New("failed to persist variant stock: " + err.Error())
				}

				zap.L().Info("Regained stock for LIMITED variant",
					zap.String("variant_id", item.Variant.ID.String()),
					zap.Int("old_stock", oldStock),
					zap.Int("regain", regainStock),
					zap.Int("new_stock", newStock))
			}
		}

	default:
		// PENDING or other statuses - no change needed
		zap.L().Debug("No order status change needed",
			zap.String("transaction_status", string(transactionStatus)))
		return nil
	}
	// Build SystemUser
	user := &model.User{
		ID:       uuid.UUID{},
		FullName: t.adminConfig.SystemName,
		Email:    t.adminConfig.SystemEmail,
	}
	// Update order status
	err := MoveOrderStateUsingFSM(order, user, newStatus, nil)
	if err != nil {
		zap.L().Error("Order state transition validation failed",
			zap.String("order_id", order.ID.String()),
			zap.String("from", string(order.Status)),
			zap.String("to", string(newStatus)),
			zap.Error(err))
		return err
	}
	//order.Status = newStatus
	if err = orderRepo.Update(ctx, order); err != nil {
		zap.L().Error("Failed to update order status",
			zap.String("order_id", order.ID.String()),
			zap.String("new_status", string(newStatus)),
			zap.Error(err))
		return errors.New("failed to update order status: " + err.Error())
	}

	zap.L().Info("Order status updated successfully",
		zap.String("order_id", order.ID.String()),
		zap.String("new_status", string(newStatus)))

	//Send notifications asynchronously
	go func() {
		ctxBg := context.Background()

		actionBy, _ := t.userRepository.GetByID(ctxBg, userID, nil)

		orderNotiStatus, err := ConvertToNotificationType(order)
		if err != nil {
			zap.L().Warn("ConvertToNotificationType failed", zap.Error(err))
			return
		}

		payloads, err := notification_builder.BuildOrderNotifications(
			ctxBg,
			*t.config,
			t.uow.DB(),
			orderNotiStatus,
			order,
			actionBy,
		)
		if err != nil {
			zap.L().Debug("no notification builder for order status", zap.Error(err))
			return
		}

		for _, p := range payloads {
			_, err = t.notificationService.CreateAndPublishNotification(ctxBg, &p)
			if err != nil {
				zap.L().Error("Failed to send notification async", zap.Error(err))
			}
		}
	}()

	return nil
}

func (t stateTransferService) handlePreOrderSideEffect(
	ctx context.Context,
	uow irepository.UnitOfWork,
	preorder *model.PreOrder,
	transactionStatus enum.PaymentTransactionStatus,
	updatedBy uuid.UUID,
) error {
	if preorder == nil {
		return errors.New("pre-order is nil")
	}

	preorderRepo := uow.PreOrder()
	var newStatus enum.PreOrderStatus

	switch transactionStatus {
	case enum.PaymentTransactionStatusCompleted:
		// mark preorder as pre-ordered (payment succeeded)
		newStatus = enum.PreOrderStatusPaid
		zap.L().Info("Updating PreOrder STATUS to Paid (payment completed)",
			zap.String("preorder_id", preorder.ID.String()))

	case enum.PaymentTransactionStatusFailed, enum.PaymentTransactionStatusCancelled, enum.PaymentTransactionStatusExpired:
		// revert stock (restore) and mark preorder cancelled
		// attempt to load variant and restore stock
		newStatus = enum.PreOrderStatusCancelled
		variantRepo := uow.ProductVariant()
		variant, err := variantRepo.GetByID(ctx, preorder.VariantID, nil)
		if err != nil {
			zap.L().Error("Failed to load variant to restore stock", zap.String("variant_id", preorder.VariantID.String()), zap.Error(err))
		} else if variant != nil {
			if variant.CurrentStock == nil {
				v := preorder.Quantity
				variant.CurrentStock = &v
			} else {
				*variant.CurrentStock += preorder.Quantity
				*variant.PreOrderCount -= preorder.Quantity
			}
			if err := variantRepo.Update(ctx, variant); err != nil {
				zap.L().Error("Failed to restore variant stock after preorder payment failure",
					zap.String("variant_id", variant.ID.String()), zap.Error(err))
			} else {
				zap.L().Info("Restored variant stock due to preorder payment failure", zap.String("variant_id", variant.ID.String()), zap.Int("restored", preorder.Quantity))
			}
		}

	default:
		zap.L().Debug("No preorder side-effect for transaction status", zap.String("transaction_status", string(transactionStatus)))
		return nil
	}

	// Build SystemUser
	user := &model.User{
		ID:       uuid.UUID{},
		FullName: t.adminConfig.SystemName,
		Email:    t.adminConfig.SystemEmail,
	}

	limitedProduct := preorder.ProductVariant.Product.Limited
	err := MovePreOrderStateUsingFSM(preorder, limitedProduct, user, newStatus, nil)
	if err != nil {
		zap.L().Error("Order state transition validation failed",
			zap.String("order_id", preorder.ID.String()),
			zap.String("from", string(preorder.Status)),
			zap.String("to", string(newStatus)),
			zap.Error(err))
		return err
	}

	if err := preorderRepo.Update(ctx, preorder); err != nil {
		zap.L().Error("Failed to update order status",
			zap.String("order_id", preorder.ID.String()),
			zap.String("new_status", string(newStatus)),
			zap.Error(err))
		return errors.New("failed to update order status: " + err.Error())
	}

	zap.L().Info("Order status updated successfully",
		zap.String("order_id", preorder.ID.String()),
		zap.String("new_status", string(newStatus)))

	//Send notifications asynchronously
	go func() {
		ctxBg := context.Background()

		actionBy, _ := t.userRepository.GetByID(ctxBg, updatedBy, nil)

		preorderNotiStatus, err := ConvertPreOrderToNotificationType(preorder)
		if err != nil {
			zap.L().Warn("ConvertToNotificationType failed", zap.Error(err))
			return
		}

		payloads, err := notification_builder.BuildPreOrderNotifications(
			ctxBg,
			*t.config,
			t.uow.DB(),
			preorderNotiStatus,
			preorder,
			actionBy,
		)
		if err != nil {
			zap.L().Debug("no notification builder for order status", zap.Error(err))
			return
		}

		for _, p := range payloads {
			_, err = t.notificationService.CreateAndPublishNotification(ctxBg, &p)
			if err != nil {
				zap.L().Error("Failed to send notification async", zap.Error(err))
			}
		}
	}()

	return nil

}

// handleContractViolationSideEffect handles contract violation penalty payment completion
func (t stateTransferService) handleContractViolationSideEffect(
	ctx context.Context,
	uow irepository.UnitOfWork,
	violation *model.ContractViolation,
	transactionStatus enum.PaymentTransactionStatus,
	updatedBy uuid.UUID,
) error {
	if violation == nil {
		return errors.New("contract violation is nil")
	}

	contractViolationRepo := uow.ContractViolations()

	switch transactionStatus {
	case enum.PaymentTransactionStatusCompleted:
		zap.L().Info("Processing brand penalty payment completion",
			zap.String("violation_id", violation.ID.String()),
			zap.String("contract_id", violation.ContractID.String()))

		// Mark violation as resolved
		now := time.Now()
		violation.ResolvedAt = &now
		if updatedBy != uuid.Nil {
			violation.ResolvedBy = &updatedBy
			violation.UpdatedBy = &updatedBy
		}

		if err := contractViolationRepo.Update(ctx, violation); err != nil {
			zap.L().Error("Failed to mark violation as resolved",
				zap.String("violation_id", violation.ID.String()),
				zap.Error(err))
			return errors.New("failed to mark violation as resolved: " + err.Error())
		}

		// Transition contract to BRAND_PENALTY_PAID state
		// This should trigger auto-transition to TERMINATED via FSM side effects
		if err := t.MoveContractToState(ctx, uow, violation.ContractID, enum.ContractStatusBrandPenaltyPaid, updatedBy); err != nil {
			zap.L().Error("Failed to move contract to BRAND_PENALTY_PAID",
				zap.String("contract_id", violation.ContractID.String()),
				zap.Error(err))
			return errors.New("failed to update contract status: " + err.Error())
		}

		zap.L().Info("Brand penalty payment processed, contract moved to BRAND_PENALTY_PAID",
			zap.String("violation_id", violation.ID.String()),
			zap.String("contract_id", violation.ContractID.String()))

		// Send notification asynchronously
		go func() {
			ctxBg := context.Background()

			// Get contract with brand details for notification
			contract, err := t.contractRepository.GetByID(ctxBg, violation.ContractID, []string{"Brand", "Brand.User"})
			if err != nil {
				zap.L().Error("Failed to get contract for penalty payment notification", zap.Error(err))
				return
			}

			if contract.Brand != nil && contract.Brand.UserID != nil {
				contractNumber := "N/A"
				if contract.ContractNumber != nil {
					contractNumber = *contract.ContractNumber
				}

				templateData := map[string]any{
					"BrandName":      contract.Brand.Name,
					"ContractNumber": contractNumber,
					"SupportLink":    t.config.Server.BaseFrontendURL + "/support",
					"CurrentYear":    time.Now().Year(),
				}

				notificationReq := requests.PublishNotificationRequest{
					UserID: *contract.Brand.UserID,
					Title:  "Penalty Payment Received",
					Body:   fmt.Sprintf("Your penalty payment for contract %s has been received. The violation has been resolved.", contractNumber),
					Data: map[string]string{
						"violation_id":    violation.ID.String(),
						"contract_id":     contract.ID.String(),
						"contract_number": contractNumber,
						"reference_type":  enum.ReferenceTypeContractViolation.String(),
						"reference_id":    violation.ID.String(),
					},
					Types:             []enum.NotificationType{enum.NotificationTypeInApp, enum.NotificationTypeEmail},
					EmailTemplateName: utils.PtrOrNil("brand_penalty_paid"),
					EmailTemplateData: templateData,
				}

				if _, err := t.notificationService.CreateAndPublishNotification(ctxBg, &notificationReq); err != nil {
					zap.L().Error("Failed to send penalty payment confirmation notification", zap.Error(err))
				}
			}
		}()

	case enum.PaymentTransactionStatusFailed, enum.PaymentTransactionStatusCancelled, enum.PaymentTransactionStatusExpired:
		zap.L().Warn("Brand penalty payment failed/cancelled/expired",
			zap.String("violation_id", violation.ID.String()),
			zap.String("transaction_status", string(transactionStatus)))
		// Payment not completed - remove payment_transaction_id reference in contract violation
		if err := uow.ContractViolations().UpdateByCondition(ctx, func(db *gorm.DB) *gorm.DB {
			return db.Where("id = ?", violation.ID)
		}, map[string]any{"payment_transaction_id": nil}); err != nil {
			zap.L().Error("Failed to clear payment transaction reference from violation",
				zap.String("violation_id", violation.ID.String()),
				zap.Error(err))
			return errors.New("failed to clear payment transaction reference: " + err.Error())
		}

		zap.L().Info("Cleared payment transaction reference from contract violation due to payment failure",
			zap.String("violation_id", violation.ID.String()),
			zap.String("transaction_status", string(transactionStatus)))

	default:
		zap.L().Debug("No contract violation side-effect for transaction status",
			zap.String("transaction_status", string(transactionStatus)))
	}

	zap.L().Info("Contract violation side-effect processing completed",
		zap.String("violation_id", violation.ID.String()),
		zap.String("transaction_status", string(transactionStatus)))

	return nil
}

// handleOrderStatusSideEffect: handler side effect of OrderStatus itself
func (t stateTransferService) handleOrderStatusSideEffect(
	ctx context.Context,
	uow irepository.UnitOfWork,
	transactionCtx *ordersm.OrderContext,
	nextStatus enum.OrderStatus,
	order *model.Order,
	updatedBy *model.User,
	reason *string, //optional
) error {
	var err error
	// Silence unused parameter warnings in some branches
	_ = transactionCtx
	_ = updatedBy
	_ = reason

	if order == nil {
		return errors.New("order is nil")
	}
	orderRepo := uow.Order()

	switch nextStatus {
	case enum.OrderStatusConfirmed:
		zap.L().Info("Order confirmation")
		if order.Status != enum.OrderStatusPaid {
			return fmt.Errorf("order must be PAID before confirmation action")
		}

		// For each order item, if product type is LIMITED, check and decrement variant stock
		for _, it := range order.OrderItems {
			if it.VariantID == uuid.Nil || it.Variant.ProductID == uuid.Nil {
				return fmt.Errorf("invalid order item: variant not found")
			}
			if it.Variant.Product.Type == enum.ProductTypeLimited {
				if it.Variant.CurrentStock == nil {
					return fmt.Errorf("variant %s stock is nil", it.VariantID.String())
				}
				if *it.Variant.CurrentStock < it.Quantity {
					return fmt.Errorf("insufficient stock for variant %s: have %d, need %d", it.VariantID.String(), *it.Variant.CurrentStock, it.Quantity)
				}
				*it.Variant.CurrentStock = *it.Variant.CurrentStock - it.Quantity
			}
		}
	case enum.OrderStatusDelivered:
		// Schedule auto-receive after a delay (idempotent by unique key)
		if t.taskScheduler == nil || t.asynqConfig == nil {
			zap.L().Warn("Asynq not configured; skip scheduling auto receive",
				zap.String("order_id", order.ID.String()))
			break
		}

		delay := time.Duration(t.config.AdminConfig.AutoReceiveOrderIntervalMs) * time.Millisecond
		processAt := time.Now().Add(delay)

		payload := asynqtask.AutoReceiveOrderTaskPayload{OrderID: order.ID}
		taskType := t.asynqConfig.TaskTypes.AutoReceiveOrder
		if taskType == "" {
			taskType = "task:order:auto-receive"
		}

		uniqueKey := fmt.Sprintf("order:auto-receive:%s", order.ID.String())
		if _, schErr := t.taskScheduler.ScheduleTaskWithUniqueKey(
			ctx,
			taskType,
			payload,
			processAt,
			uniqueKey,
			gAsynq.Queue("default"),
			gAsynq.MaxRetry(10),
		); schErr != nil {
			zap.L().Error("Failed to schedule auto receive order task",
				zap.Error(schErr),
				zap.String("order_id", order.ID.String()))
			// Do not fail the whole state transition because of scheduling.
		} else {
			zap.L().Info("Auto receive order task scheduled",
				zap.String("order_id", order.ID.String()),
				zap.Time("process_at", processAt),
				zap.String("unique_key", uniqueKey))
		}
	case enum.OrderStatusCancelled:
		// Regain stock for LIMITED orders and persist per-variant
		if order.OrderType == enum.ProductTypeLimited.String() {
			variantRepo := uow.ProductVariant()
			for _, it := range order.OrderItems {
				// only LIMITED products affect stock
				if it.Variant.Product.Type != enum.ProductTypeLimited {
					continue
				}
				old := 0
				if it.Variant.CurrentStock != nil {
					old = *it.Variant.CurrentStock
				}
				newStock := old + it.Quantity
				it.Variant.CurrentStock = &newStock
				if err = variantRepo.Update(ctx, &it.Variant); err != nil {
					zap.L().Error("Failed to persist regained stock for variant (manual cancel)",
						zap.String("variant_id", it.Variant.ID.String()),
						zap.Error(err))
					return err
				}
				zap.L().Info("Regained stock for LIMITED variant (manual cancel)",
					zap.String("variant_id", it.Variant.ID.String()),
					zap.Int("old_stock", old),
					zap.Int("regain", it.Quantity),
					zap.Int("new_stock", newStock))
			}
		}
	case enum.OrderStatusRefunded:
		zap.L().Info("Order refunded, sending rejected email")
		err = orderRepo.Update(ctx, order)
	}
	return err
}

func NewStateTransferService(
	dbReg *gormrepository.DatabaseRegistry,
	notificationService iservice.NotificationService,
	scheduleService iservice.ScheduleService,
	uow irepository.UnitOfWork,
	rabbitmq *rabbitmq.RabbitMQ,
	ghnProxy iproxies.GHNProxy,
	taskScheduler *asynq.AsynqClient,
	configs *config.AppConfig,
) iservice.StateTransferService {
	return &stateTransferService{
		contractRepository:          dbReg.ContractRepository,
		contractPaymentRepository:   dbReg.ContractPaymentRepository,
		contractViolationRepository: dbReg.ContractViolationRepository,
		campaignRepository:          dbReg.CampaignRepository,
		milestoneRepository:         dbReg.MilestoneRepository,
		taskRepository:              dbReg.TaskRepository,
		productRepository:           dbReg.ProductRepository,
		affiliateLinkRepository:     dbReg.AffiliateLinkRepository,
		orderRepository:             dbReg.OrderRepository,
		preOrderRepository:          dbReg.PreOrderRepository,
		variantRepository:           dbReg.ProductVariantRepository,
		userRepository:              dbReg.UserRepository,
		contentRepository:           dbReg.ContentRepository,
		notificationService:         notificationService,
		scheduleRepo:                dbReg.ScheduleRepository,
		scheduleService:             scheduleService,
		uow:                         uow,
		rabbitMQ:                    rabbitmq,
		ghnProxy:                    ghnProxy,
		config:                      configs,
		adminConfig:                 configs.AdminConfig,
		taskScheduler:               taskScheduler,
		asynqConfig:                 &configs.Asynq,
	}
}

func (t *stateTransferService) lookupPreOrderWithLimitedProductAndUser(ctx context.Context, preorderID, actionBy uuid.UUID) (*model.PreOrder, *model.LimitedProduct, *model.User, error) {
	// 1) Load PreOrder
	preOrder, err := t.preOrderRepository.GetByID(ctx, preorderID, nil)
	if err != nil {
		return nil, nil, nil, err
	}

	variantIncludes := []string{"Product", "Product.Limited"}
	variant, err := t.variantRepository.GetByID(ctx, preOrder.VariantID, variantIncludes)
	if err != nil {
		return nil, nil, nil, err
	}

	// If actionBy is zero value, treat as System user
	var user *model.User
	if actionBy == uuid.Nil {
		user = &model.User{
			ID:       uuid.UUID{},
			FullName: t.adminConfig.SystemName,
			Email:    t.adminConfig.SystemEmail,
		}
	} else {
		user, err = t.userRepository.GetByID(ctx, actionBy, nil)
		if err != nil {
			return nil, nil, nil, err
		}
	}

	return preOrder, variant.Product.Limited, user, nil
}

func ConvertToNotificationType(order *model.Order) (notification_builder.OrderNotificationType, error) {
	status := order.Status
	switch status {
	case enum.OrderStatusPending:
		return notification_builder.OrderNotifyPending, nil
	case enum.OrderStatusPaid:
		return notification_builder.OrderNotifyPaid, nil
	case enum.OrderStatusRefundRequested:
		return notification_builder.OrderNotifyRefundRequested, nil
	case enum.OrderStatusRefunded:
		lastActionStatus := order.GetLatestActionNote().ActionType
		if lastActionStatus == enum.OrderStatusPaid {
			return notification_builder.OrderNotifyObligateRefund, nil
		}
		return notification_builder.OrderNotifyRefunded, nil
	case enum.OrderStatusConfirmed:
		return notification_builder.OrderNotifyConfirmed, nil
	case enum.OrderStatusCancelled:
		return notification_builder.OrderNotifyCancelled, nil
	case enum.OrderStatusShipped:
		return notification_builder.OrderNotifyShipped, nil
	case enum.OrderStatusInTransit:
		return notification_builder.OrderNotifyInTransit, nil
	case enum.OrderStatusDelivered:
		lastActionStatus := order.GetLatestActionNote().ActionType
		if lastActionStatus == enum.OrderStatusCompensateRequested {
			return notification_builder.OrderNotifyCompensationDenied, nil
		}
		return notification_builder.OrderNotifyDelivered, nil
	case enum.OrderStatusReceived:
		return notification_builder.OrderNotifyReceived, nil
	case enum.OrderStatusCompensateRequested:
		return notification_builder.OrderNotifyCompensateRequested, nil
	case enum.OrderStatusCompensated:
		return notification_builder.OrderNotifyCompensated, nil
	case enum.OrderStatusAwaitingPickUp:
		return notification_builder.OrderNotifyAwaitingPickUp, nil
	default:
		return "", fmt.Errorf("unrecognized order status: %s", status)
	}
}

func ConvertPreOrderToNotificationType(preorder *model.PreOrder) (notification_builder.PreOrderNotificationType, error) {
	status := preorder.Status
	switch status {
	case enum.PreOrderStatusPending:
		return notification_builder.PreOrderNotifyPending, nil
	case enum.PreOrderStatusPaid:
		return notification_builder.PreOrderNotifyPaid, nil
	case enum.PreOrderStatusPreOrdered:
		return notification_builder.PreOrderNotifyPreOrdered, nil
	case enum.PreOrderStatusAwaitingPickup:
		return notification_builder.PreOrderNotifyAwaitingPickup, nil
	case enum.PreOrderStatusInTransit:
		return notification_builder.PreOrderNotifyInTransit, nil
	case enum.PreOrderStatusDelivered:
		return notification_builder.PreOrderNotifyDelivered, nil
	case enum.PreOrderStatusReceived:
		return notification_builder.PreOrderNotifyReceived, nil
	case enum.PreOrderStatusCompensateRequest:
		return notification_builder.PreOrderNotifyCompensated, nil
	case enum.PreOrderStatusCompensated:
		return notification_builder.PreOrderNotifyCompensated, nil
	case enum.PreOrderStatusCancelled:
		return notification_builder.PreOrderNotifyCancelled, nil
	case enum.PreOrderStatusShipped:
		return notification_builder.PreOrderNotifyShipped, nil
	default:
		return "", fmt.Errorf("unrecognized preorder status: %s", status)
	}
}

func (t stateTransferService) publishPreOrderOpeningDelayMessage(ctx context.Context, schedule *model.Schedule, uow irepository.UnitOfWork) error {
	if t.taskScheduler == nil {
		return errors.New("task scheduler not initialized")
	}
	if t.asynqConfig == nil {
		return errors.New("asynq config not initialized")
	}
	if schedule == nil {
		return errors.New("schedule is nil")
	}
	if schedule.ReferenceID == nil || *schedule.ReferenceID == uuid.Nil {
		return errors.New("schedule reference id is required")
	}

	includes := []string{"ProductVariant", "ProductVariant.Product", "ProductVariant.Product.Limited"}
	preOrder, err := uow.PreOrder().GetByID(ctx, *schedule.ReferenceID, includes)
	if err != nil {
		return fmt.Errorf("pre-order not found for schedule reference id: %s", schedule.ReferenceID.String())
	}

	payload := asynqtask.PreOrderOpeningTaskPayload{
		PreOrderID: preOrder.ID,
	}

	taskType := t.asynqConfig.TaskTypes.PreOrderOpening
	if taskType == "" {
		taskType = "task:order:preorder-opening"
	}

	taskInfo, err := t.taskScheduler.ScheduleTaskWithUniqueKey(
		ctx,
		taskType,
		payload,
		schedule.ScheduledAt,
		fmt.Sprintf("preorder-opening:%s", preOrder.ID.String()),
		gAsynq.Queue("default"),
		gAsynq.MaxRetry(10),
	)

	if err != nil {
		return fmt.Errorf("failed to schedule preorder opening task: %w", err)
	}
	t.updateScheduleMetadataAsync(schedule, taskInfo)
	zap.L().Info("Scheduled PreOrder opening task",
		zap.String("preorder_id", preOrder.ID.String()),
		zap.Time("start_at", schedule.ScheduledAt),
		zap.String("task_type", taskType),
	)

	return nil
}

func (t stateTransferService) updateScheduleMetadataAsync(schedule *model.Schedule, taskInfo *gAsynq.TaskInfo) {
	go func() {
		if err := utils.RunWithRetry(context.Background(), utils.DefaultRetryOptions, func(ctx context.Context) error {
			rawTaskInfo, err := json.Marshal(taskInfo)
			if err != nil {
				return fmt.Errorf("failed to marshal task info: %w", err)
			}

			return t.scheduleRepo.UpdateByCondition(ctx, func(db *gorm.DB) *gorm.DB {
				return db.Where("id = ?", schedule.ID)
			}, map[string]any{"metadata": rawTaskInfo})
		}); err != nil {
			zap.L().Error("Failed to update schedule metadata asynchronously",
				zap.String("schedule_id", schedule.ID.String()),
				zap.Error(err))
		}
	}()
}

func (t stateTransferService) publishPreOrderAutoReceiveDelayMessage(ctx context.Context, schedule *model.Schedule, uow irepository.UnitOfWork) error {
	if t.taskScheduler == nil {
		return errors.New("task scheduler not initialized")
	}
	if t.asynqConfig == nil {
		return errors.New("asynq config not initialized")
	}
	if schedule == nil {
		return errors.New("schedule is nil")
	}
	if schedule.ReferenceID == nil || *schedule.ReferenceID == uuid.Nil {
		return errors.New("schedule reference id is required")
	}

	preOrder, err := uow.PreOrder().GetByID(ctx, *schedule.ReferenceID, nil)
	if err != nil {
		return fmt.Errorf("pre-order not found for schedule reference id: %s", schedule.ReferenceID.String())
	}

	payload := asynqtask.PreOrderAutoReceiveTaskPayload{
		PreOrderID: preOrder.ID,
	}

	taskType := t.asynqConfig.TaskTypes.PreOrderAutoReceive
	if taskType == "" {
		taskType = "task:order:preorder-auto-receive"
	}

	taskInfo, err := t.taskScheduler.ScheduleTaskWithUniqueKey(
		ctx,
		taskType,
		payload,
		schedule.ScheduledAt,
		fmt.Sprintf("preorder-auto-receive:%s", preOrder.ID.String()),
		gAsynq.Queue("default"),
		gAsynq.MaxRetry(10),
	)

	if err != nil {
		return fmt.Errorf("failed to schedule preorder auto-receive task: %w", err)
	}
	t.updateScheduleMetadataAsync(schedule, taskInfo)
	zap.L().Info("Scheduled PreOrder auto-receive task",
		zap.String("preorder_id", preOrder.ID.String()),
		zap.Time("scheduled_at", schedule.ScheduledAt),
		zap.String("task_type", taskType),
	)

	return nil
}

// scheduleLimitedProductAnnouncements schedules notification announcements for LIMITED products
// when they are activated. It schedules notifications at:
// - 3 days before PremiereDate
// - 1 day before PremiereDate
// - 3 days before AvailabilityStartDate
// - 1 day before AvailabilityStartDate
func (t stateTransferService) scheduleLimitedProductAnnouncements(ctx context.Context, product *model.Product) {
	if t.taskScheduler == nil || t.asynqConfig == nil {
		zap.L().Warn("Asynq not configured; skip scheduling limited product announcements",
			zap.String("product_id", product.ID.String()))
		return
	}

	if product.Limited == nil {
		zap.L().Warn("Product does not have limited info; skip scheduling announcements",
			zap.String("product_id", product.ID.String()))
		return
	}

	limited := product.Limited
	now := time.Now()

	taskType := t.asynqConfig.TaskTypes.LimitedProductAnnouncement
	if taskType == "" {
		taskType = "task:product:limited-announcement"
	}

	// Define announcement schedules: (days before, announcement type, target date)
	type announcementSchedule struct {
		daysBefore       int
		announcementType asynqtask.LimitedProductAnnouncementType
		targetDate       time.Time
		dateLabel        string
	}

	schedules := []announcementSchedule{
		{3, asynqtask.AnnouncementTypePremiereDate3Days, limited.PremiereDate, "premiere"},
		{1, asynqtask.AnnouncementTypePremiereDate1Day, limited.PremiereDate, "premiere"},
		{3, asynqtask.AnnouncementTypeAvailability3Days, limited.AvailabilityStartDate, "availability"},
		{1, asynqtask.AnnouncementTypeAvailability1Day, limited.AvailabilityStartDate, "availability"},
	}

	for _, sched := range schedules {
		scheduledAt := sched.targetDate.AddDate(0, 0, -sched.daysBefore)

		// Skip if the scheduled time is in the past
		if scheduledAt.Before(now) {
			zap.L().Info("Skipping limited product announcement (scheduled time in past)",
				zap.String("product_id", product.ID.String()),
				zap.String("announcement_type", string(sched.announcementType)),
				zap.Time("scheduled_at", scheduledAt),
				zap.Time("target_date", sched.targetDate))
			continue
		}

		payload := asynqtask.LimitedProductAnnouncementPayload{
			ProductID:        product.ID,
			ProductName:      product.Name,
			AnnouncementType: sched.announcementType,
			TargetDate:       sched.targetDate,
			ScheduledAt:      scheduledAt,
		}

		uniqueKey := fmt.Sprintf("product:limited-announcement:%s:%s",
			product.ID.String(), string(sched.announcementType))

		_, err := t.taskScheduler.ScheduleTaskWithUniqueKey(
			ctx,
			taskType,
			payload,
			scheduledAt,
			uniqueKey,
			gAsynq.Queue("default"),
			gAsynq.MaxRetry(5),
		)

		if err != nil {
			zap.L().Error("Failed to schedule limited product announcement",
				zap.String("product_id", product.ID.String()),
				zap.String("announcement_type", string(sched.announcementType)),
				zap.Time("scheduled_at", scheduledAt),
				zap.Error(err))
			continue
		}

		zap.L().Info("Scheduled limited product announcement",
			zap.String("product_id", product.ID.String()),
			zap.String("product_name", product.Name),
			zap.String("announcement_type", string(sched.announcementType)),
			zap.Int("days_before", sched.daysBefore),
			zap.String("date_type", sched.dateLabel),
			zap.Time("scheduled_at", scheduledAt),
			zap.Time("target_date", sched.targetDate))
	}
}
