package service

import (
	"context"
	"core-backend/internal/application/dto/consumers"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/application/interfaces/iservice"
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
	"core-backend/internal/domain/state/campaignsm"
	"core-backend/internal/domain/state/contentsm"
	"core-backend/internal/domain/state/contractsm"
	"core-backend/internal/domain/state/milestonesm"
	"core-backend/internal/domain/state/paymenttransactionsm"
	"core-backend/internal/domain/state/productsm"
	"core-backend/internal/domain/state/tasksm"
	gormrepository "core-backend/internal/infrastructure/gorm_repository"
	"core-backend/internal/infrastructure/rabbitmq"
	"errors"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm" // added for UpdateByCondition filter closure
)

type stateTransferService struct {
	contractRepository      irepository.GenericRepository[model.Contract]
	campaignRepository      irepository.GenericRepository[model.Campaign]
	milestoneRepository     irepository.GenericRepository[model.Milestone]
	taskRepository          irepository.GenericRepository[model.Task]
	productRepository       irepository.GenericRepository[model.Product]
	affiliateLinkRepository irepository.AffiliateLinkRepository
	uow                     irepository.UnitOfWork
	rabbitMQ                *rabbitmq.RabbitMQ
}

func (t stateTransferService) MoveTaskToState(ctx context.Context, taskID uuid.UUID, targetState enum.TaskStatus, updatedBy uuid.UUID) error {
	//1. Load current task from DB
	// Preload nested product -> task to have back-reference available ("Products.Task")
	task, err := t.taskRepository.GetByID(ctx, taskID, []string{"Products", "Products.Task", "Contents"})
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
		Products: task.Products,
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
	if err := t.taskRepository.Update(ctx, task); err != nil {
		zap.L().Error("Failed to update task state in DB",
			zap.String("user_id", taskID.String()),
			zap.String("new_state", targetState.String()),
			zap.Error(err))
		return errors.New("Failed to update task state in DB: " + err.Error())
	}

	//6. Cascade UpdatedByID (and any status changes applied by state machine) to products, if any
	for _, p := range taskCtx.Products {
		if p == nil {
			continue
		}
		// Ensure task back-reference present (if not, assign for safety)
		if p.Task == nil {
			p.Task = task
		}
		p.UpdatedByID = &updatedBy
		if err := t.productRepository.Update(ctx, p); err != nil {
			// Log and continue; do not fail whole operation after task updated
			zap.L().Error("Failed to cascade product update_by", zap.String("task_id", taskID.String()), zap.String("product_id", p.ID.String()), zap.Error(err))
		}
	}

	return nil
}

func (t stateTransferService) MoveProductToState(ctx context.Context, productID uuid.UUID, targetState enum.ProductStatus, updatedBy uuid.UUID) error {
	product, err := t.productRepository.GetByID(ctx, productID, []string{})
	product.UpdatedByID = &updatedBy

	if err != nil {
		zap.L().Error("Failed to load Product from DB",
			zap.String("product ID", productID.String()),
			zap.Error(err))
		return errors.New("Unable to find Product: " + err.Error())
	}

	// load current product context
	productCtx := &productsm.ProductContext{
		State: productsm.NewProductState(product.Status),
	}

	// init next product State
	nextProductState := productsm.NewProductState(targetState)
	if nextProductState == nil {
		zap.L().Error("Invalid target state",
			zap.String("target_state", targetState.String()))
		return errors.New("invalid target state")
	}

	//4. Forward state
	if err := productCtx.State.Next(productCtx, nextProductState); err != nil {
		zap.L().Error("State transition failed",
			zap.String("user_id", productID.String()),
			zap.String("from", productCtx.State.Name().String()),
			zap.String("to", targetState.String()),
			zap.Error(err))
		return errors.New("State transition failed: " + err.Error())
	}

	//5. Save to DB
	product.Status = targetState
	if err := t.productRepository.Update(ctx, product); err != nil {
		zap.L().Error("Failed to update product state in DB",
			zap.String("user_id", productID.String()),
			zap.String("new_state", targetState.String()),
			zap.Error(err))
		return errors.New("Failed to update product state in DB: " + err.Error())
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
	milestone, err := milestoneRepo.GetByID(ctx, mileStoneID, []string{"Tasks", "Tasks.Milestone", "Tasks.Products", "Tasks.Contents"})
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

func (t stateTransferService) MoveCampaignToState(ctx context.Context, campaignID uuid.UUID, targetState enum.CampaignStatus, updatedBy uuid.UUID) error {
	//1. Load current task from DB
	trx := t.uow.Begin(ctx)
	defer func() {
		if r := recover(); r != nil {
			_ = trx.Rollback()
			zap.L().Error("panic recovered in MoveCampaignToState", zap.Any("recover", r))
		}
	}()

	campaignRepo := trx.Campaigns()
	campaign, err := campaignRepo.GetByID(ctx, campaignID, []string{"Milestones", "Milestones.Campaign", "Milestones.Tasks", "Milestones.Tasks.Products", "Milestones.Tasks.Contents"})
	if err != nil {
		_ = trx.Rollback()
		zap.L().Error("Failed to load campaign", zap.String("campaign_id", campaignID.String()), zap.Error(err))
		return errors.New("failed to load campaign: " + err.Error())
	}
	//TODO: Set updatedBy AFTER successful fetch -> incase for cascade
	campaign.UpdatedByID = &updatedBy

	//2. Load task context
	cCtx := &campaignsm.CampaignContext{State: campaignsm.NewCampaignState(campaign.Status), MileStones: campaign.Milestones}

	//3. Init target State
	nextState := campaignsm.NewCampaignState(targetState)
	if nextState == nil {
		_ = trx.Rollback()
		zap.L().Error("Invalid target campaign state", zap.String("campaign_id", campaignID.String()), zap.String("target_state", targetState.String()))
		return errors.New("invalid target campaign state")
	}

	//4. Forward state
	if err := cCtx.State.Next(cCtx, nextState); err != nil {
		_ = trx.Rollback()
		zap.L().Error("Campaign state transition failed", zap.String("campaign_id", campaignID.String()), zap.String("from", cCtx.State.Name().String()), zap.String("to", targetState.String()), zap.Error(err))
		return errors.New("campaign state transition failed: " + err.Error())
	}

	//5. Persist task new state
	campaign.Status = targetState
	if err := campaignRepo.Update(ctx, campaign); err != nil {
		_ = trx.Rollback()
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

	// Preload deeper campaign tree if contract has a campaign
	if contract.Campaign != nil {
		camp, err2 := campaignRepo.GetByID(ctx, contract.Campaign.ID, []string{"Milestones", "Milestones.Tasks", "Milestones.Tasks.Products", "Milestones.Tasks.Contents"})
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
		if err := t.affiliateLinkRepository.UpdateByCondition(ctx, func(db *gorm.DB) *gorm.DB {
			return db.Where("contract_id = ? AND status = ?", contractID, enum.AffiliateLinkStatusActive)
		}, map[string]any{"status": enum.AffiliateLinkStatusExpired}); err != nil {
			zap.L().Error("Failed to expire affiliate links (contract)", zap.String("contract_id", contractID.String()), zap.Error(err))
			// Don't fail the entire transaction - log warning and continue
			zap.L().Warn("Continuing contract termination despite affiliate link update failure")
		} else {
			zap.L().Info("Expired affiliate links due to contract termination", zap.String("contract_id", contractID.String()))
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
				for _, p := range tk.Products {
					if p != nil {
						p.Status = enum.ProductStatusInactived
					}
				}
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

	if err := trx.Commit(); err != nil {
		zap.L().Error("Contract transaction commit failed", zap.Error(err))
		return errors.New("transaction commit failed: " + err.Error())
	}
	return nil
}

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

	// 6. Side-effects: Expire affiliate links if content is unpublished
	// If content is moved away from POSTED status, expire associated affiliate links
	if targetState != enum.ContentStatusPosted {
		affiliateLinkRepo := uow.AffiliateLinks()
		if err := affiliateLinkRepo.UpdateByCondition(ctx, func(db *gorm.DB) *gorm.DB {
			return db.Where("content_id = ? AND status = ?", contentID, enum.AffiliateLinkStatusActive)
		}, map[string]any{"status": enum.AffiliateLinkStatusExpired}); err != nil {
			zap.L().Error("Failed to expire affiliate links (content unpublish)",
				zap.String("content_id", contentID.String()),
				zap.Error(err))
			// Don't fail the entire transaction - log warning and continue
			zap.L().Warn("Continuing content state change despite affiliate link update failure")
		} else {
			zap.L().Info("Expired affiliate links due to content unpublish",
				zap.String("content_id", contentID.String()),
				zap.String("new_status", string(targetState)))
		}
	}

	zap.L().Info("Content state transition successful",
		zap.String("content_id", contentID.String()),
		zap.String("new_state", string(targetState)),
		zap.String("updated_by", updatedBy.String()))

	return nil
}

func (t stateTransferService) MovePaymentTransactionToState(ctx context.Context, uow irepository.UnitOfWork, transactionID uuid.UUID, targetState enum.PaymentTransactionStatus) error {
	// Use transactional repository from UnitOfWork
	transactionRepo := uow.PaymentTransaction()
	contractPaymentRepo := uow.ContractPayments()
	orderRepo := uow.Order()

	// 1. Load payment transaction with reference entity
	transaction, err := transactionRepo.GetByID(ctx, transactionID, nil)
	if err != nil {
		zap.L().Error("Failed to load payment transaction from DB",
			zap.String("transaction_id", transactionID.String()),
			zap.Error(err))
		return errors.New("unable to find payment transaction: " + err.Error())
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
		contractPayment, err := contractPaymentRepo.GetByID(ctx, transaction.ReferenceID, nil)
		if err != nil {
			zap.L().Error("Failed to load contract payment",
				zap.String("contract_payment_id", transaction.ReferenceID.String()),
				zap.Error(err))
			return errors.New("unable to find contract payment: " + err.Error())
		}
		transactionCtx.ContractPayment = contractPayment

	case enum.PaymentTransactionReferenceTypeOrder:
		order, err := orderRepo.GetByID(ctx, transaction.ReferenceID, nil)
		if err != nil {
			zap.L().Error("Failed to load order",
				zap.String("order_id", transaction.ReferenceID.String()),
				zap.Error(err))
			return errors.New("unable to find order: " + err.Error())
		}
		transactionCtx.Order = order
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
	if err := t.handlePaymentTransactionSideEffects(ctx, uow, transactionCtx, targetState); err != nil {
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

// handlePaymentTransactionSideEffects handles cascading updates based on payment status
func (t stateTransferService) handlePaymentTransactionSideEffects(
	ctx context.Context,
	uow irepository.UnitOfWork,
	transactionCtx *paymenttransactionsm.PaymentTransactionContext,
	targetState enum.PaymentTransactionStatus,
) error {
	switch transactionCtx.ReferenceType {
	case enum.PaymentTransactionReferenceTypeContractPayment:
		return t.handleContractPaymentSideEffect(ctx, uow, transactionCtx.ContractPayment, targetState)

	case enum.PaymentTransactionReferenceTypeOrder:
		return t.handleOrderSideEffect(ctx, uow, transactionCtx.Order, targetState)

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

	case enum.PaymentTransactionStatusFailed,
		enum.PaymentTransactionStatusCancelled,
		enum.PaymentTransactionStatusExpired:
		newStatus = enum.ContractPaymentStatusPending
		zap.L().Info("Reverting contract payment to PENDING",
			zap.String("contract_payment_id", contractPayment.ID.String()),
			zap.String("transaction_status", string(transactionStatus)))

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
func (t stateTransferService) handleOrderSideEffect(
	ctx context.Context,
	uow irepository.UnitOfWork,
	order *model.Order,
	transactionStatus enum.PaymentTransactionStatus,
) error {
	if order == nil {
		return errors.New("order is nil")
	}

	orderRepo := uow.Order()
	var newStatus enum.OrderStatus

	switch transactionStatus {
	case enum.PaymentTransactionStatusCompleted:
		newStatus = enum.OrderStatusPending
		zap.L().Info("Updating order to PENDING (payment completed)",
			zap.String("order_id", order.ID.String()))

	case enum.PaymentTransactionStatusFailed,
		enum.PaymentTransactionStatusCancelled,
		enum.PaymentTransactionStatusExpired:
		// Revert order to PENDING state if payment failed
		newStatus = enum.OrderStatusPending
		zap.L().Info("Keeping/reverting order to PENDING",
			zap.String("order_id", order.ID.String()),
			zap.String("transaction_status", string(transactionStatus)))

	default:
		// PENDING or other statuses - no change needed
		zap.L().Debug("No order status change needed",
			zap.String("transaction_status", string(transactionStatus)))
		return nil
	}

	// Update order status
	order.Status = newStatus
	if err := orderRepo.Update(ctx, order); err != nil {
		zap.L().Error("Failed to update order status",
			zap.String("order_id", order.ID.String()),
			zap.String("new_status", string(newStatus)),
			zap.Error(err))
		return errors.New("failed to update order status: " + err.Error())
	}

	zap.L().Info("Order status updated successfully",
		zap.String("order_id", order.ID.String()),
		zap.String("new_status", string(newStatus)))

	return nil
}

func NewStateTransferService(
	dbReg *gormrepository.DatabaseRegistry,
	uow irepository.UnitOfWork,
	rabbitmq *rabbitmq.RabbitMQ,
) iservice.StateTransferService {
	return &stateTransferService{
		contractRepository:      dbReg.ContractRepository,
		campaignRepository:      dbReg.CampaignRepository,
		milestoneRepository:     dbReg.MilestoneRepository,
		taskRepository:          dbReg.TaskRepository,
		productRepository:       dbReg.ProductRepository,
		affiliateLinkRepository: dbReg.AffiliateLinkRepository,
		uow:                     uow,
		rabbitMQ:                rabbitmq,
	}
}
