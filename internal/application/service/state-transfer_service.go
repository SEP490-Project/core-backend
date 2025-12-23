package service

import (
	"context"
	"core-backend/config"
	asynqtask "core-backend/internal/application/dto/asynq_tasks"
	"core-backend/internal/application/dto/consumers"
	"core-backend/internal/application/dto/dtos"
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
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	githubAsynq "github.com/hibiken/asynq"
	"go.uber.org/zap"
	"gorm.io/gorm" // added for UpdateByCondition filter closure
)

type stateTransferService struct {
	contractRepository      irepository.GenericRepository[model.Contract]
	campaignRepository      irepository.GenericRepository[model.Campaign]
	milestoneRepository     irepository.GenericRepository[model.Milestone]
	taskRepository          irepository.GenericRepository[model.Task]
	productRepository       irepository.GenericRepository[model.Product]
	orderRepository         irepository.GenericRepository[model.Order]
	preOrderRepository      irepository.PreOrderRepository
	variantRepository       irepository.GenericRepository[model.ProductVariant]
	affiliateLinkRepository irepository.AffiliateLinkRepository
	userRepository          irepository.GenericRepository[model.User]
	notificationService     iservice.NotificationService
	uow                     irepository.UnitOfWork
	rabbitMQ                *rabbitmq.RabbitMQ
	ghnProxy                iproxies.GHNProxy
	adminConfig             config.AdminConfig
	config                  *config.AppConfig
	taskScheduler           *asynq.AsynqClient
	asynqConfig             *config.AsynqConfig
}

func (t stateTransferService) MoveOrderToStateByGHNWebhook(ctx context.Context, ghnCode string, ghnStatus enum.GHNDeliveryStatus) error {
	//find order By GHN code
	filter := func(db *gorm.DB) *gorm.DB {
		return db.Where("ghn_order_code = ?", ghnCode)
	}
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

	if err := t.preOrderRepository.Update(ctx, preOrder); err != nil {
		zap.L().Error("Failed to update PreOrder state", zap.String("preorder_id", preOrderID.String()), zap.Error(err))
		return errors.New("failed to update PreOrder state: " + err.Error())
	}

	// notification
	go func() {
		ctxBg := context.Background()
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
}

// preOrderFileAssurance (Special case): proof of delivery file is required when moving to Delivered/Received
func preOrderFileAssurance(preOrder *model.PreOrder, targetState enum.PreOrderStatus, fileURL *string) error {
	isSelfPick := preOrder.IsSelfPickedUp
	isStatusDelivered := targetState.String() == enum.PreOrderStatusDelivered.String()
	isStatusReceived := targetState.String() == enum.PreOrderStatusReceived.String()
	if (isSelfPick && isStatusReceived) || (!isSelfPick && isStatusDelivered) {
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
	// Terminate contract -> cascade cancel related campaign, milestones, tasks, contents, and products
	case enum.ContractStatusTerminated:
		if contract.Campaign == nil {
			break
		}
		camp := contract.Campaign
		// Batch cancel milestones
		if err := milestoneRepo.UpdateByCondition(ctx, func(db *gorm.DB) *gorm.DB {
			return db.Where("campaign_id = ? AND status <> ?", camp.ID, enum.MilestoneStatusCancelled)
		}, map[string]any{"status": enum.MilestoneStatusCancelled}); err != nil {
			zap.L().Error("Failed cancel milestones (contract)", zap.String("contract_id", contractID.String()), zap.Error(err))
			return errors.New("cascade milestone cancel failed: " + err.Error())
		}
		// Batch cancel tasks
		if err := taskRepo.UpdateByCondition(ctx, func(db *gorm.DB) *gorm.DB {
			return db.Where("milestone_id IN (?) AND status <> ?", db.Select("id").Model(&model.Milestone{}).Where("campaign_id = ?", camp.ID), enum.TaskStatusCancelled)
		}, map[string]any{"status": enum.TaskStatusCancelled}); err != nil {
			zap.L().Error("Failed cancel tasks (contract)", zap.String("contract_id", contractID.String()), zap.Error(err))
			return errors.New("cascade task cancel failed: " + err.Error())
		}
		// Batch inactivate products
		if err := productRepo.UpdateByCondition(ctx, func(db *gorm.DB) *gorm.DB {
			return db.Where("task_id IN (?) AND status <> ?", db.Select("t.id").Table("tasks as t").Where("t.milestone_id IN (?)", db.Select("id").Model(&model.Milestone{}).Where("campaign_id = ?", camp.ID)), enum.ProductStatusInactived)
		}, map[string]any{"status": enum.ProductStatusInactived}); err != nil {
			zap.L().Error("Failed inactivate products (contract)", zap.String("contract_id", contractID.String()), zap.Error(err))
			return errors.New("cascade product inactivate failed: " + err.Error())
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

	return nil
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

		// send notification
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

		delay := time.Duration(t.config.AdminConfig.AutoReceiveOrderIntervalHours) * time.Hour
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
			githubAsynq.Queue("default"),
			githubAsynq.MaxRetry(10),
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
	notificationService *NotificationService,
	uow irepository.UnitOfWork,
	rabbitmq *rabbitmq.RabbitMQ,
	ghnProxy iproxies.GHNProxy,
	taskScheduler *asynq.AsynqClient,
	configs *config.AppConfig,
) iservice.StateTransferService {
	return &stateTransferService{
		contractRepository:      dbReg.ContractRepository,
		campaignRepository:      dbReg.CampaignRepository,
		milestoneRepository:     dbReg.MilestoneRepository,
		taskRepository:          dbReg.TaskRepository,
		productRepository:       dbReg.ProductRepository,
		affiliateLinkRepository: dbReg.AffiliateLinkRepository,
		orderRepository:         dbReg.OrderRepository,
		preOrderRepository:      dbReg.PreOrderRepository,
		variantRepository:       dbReg.ProductVariantRepository,
		userRepository:          dbReg.UserRepository,
		notificationService:     notificationService,
		uow:                     uow,
		rabbitMQ:                rabbitmq,
		ghnProxy:                ghnProxy,
		config:                  configs,
		adminConfig:             configs.AdminConfig,
		taskScheduler:           taskScheduler,
		asynqConfig:             &configs.Asynq,
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
	default:
		return "", fmt.Errorf("unrecognized preorder status: %s", status)
	}
}
