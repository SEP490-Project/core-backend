package service

import (
	"context"
	"core-backend/internal/application/dto/consumers"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/application/interfaces/iservice"
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
	"core-backend/internal/domain/state/campaignsm"
	"core-backend/internal/domain/state/contractsm"
	"core-backend/internal/domain/state/milestonesm"
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
	contractRepository  irepository.GenericRepository[model.Contract]
	campaignRepository  irepository.GenericRepository[model.Campaign]
	milestoneRepository irepository.GenericRepository[model.Milestone]
	taskRepository      irepository.GenericRepository[model.Task]
	productRepository   irepository.GenericRepository[model.Product]
	uow                 irepository.UnitOfWork
	rabbitMQ            *rabbitmq.RabbitMQ
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
	trx := t.uow.Begin()
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
	trx := t.uow.Begin()
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

func (t stateTransferService) MoveContractToState(ctx context.Context, contractID uuid.UUID, targetState enum.ContractStatus, updatedBy uuid.UUID) error {
	trx := t.uow.Begin()
	defer func() {
		if r := recover(); r != nil {
			_ = trx.Rollback()
			zap.L().Error("panic recovered in MoveContractToState", zap.Any("recover", r))
		}
	}()

	contractRepo := trx.Contracts()
	campaignRepo := trx.Campaigns()
	milestoneRepo := trx.Milestones()
	taskRepo := trx.Tasks()
	productRepo := trx.Products()

	contract, err := contractRepo.GetByID(ctx, contractID, []string{"Brand", "Campaign"})
	if err != nil {
		_ = trx.Rollback()
		zap.L().Error("Failed to load contract", zap.String("contract_id", contractID.String()), zap.Error(err))
		return errors.New("failed to load contract: " + err.Error())
	} else if contract == nil {
		_ = trx.Rollback()
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
		_ = trx.Rollback()
		zap.L().Error("Invalid contract target state", zap.String("contract_id", contractID.String()), zap.String("target_state", targetState.String()))
		return errors.New("invalid target contract state")
	}

	if err := cCtx.State.Next(cCtx, nextState); err != nil {
		_ = trx.Rollback()
		zap.L().Error("Contract state transition failed", zap.String("contract_id", contractID.String()), zap.String("from", cCtx.State.Name().String()), zap.String("to", targetState.String()), zap.Error(err))
		return errors.New("contract state transition failed: " + err.Error())
	}

	contract.Status = targetState
	if err := contractRepo.Update(ctx, contract); err != nil {
		_ = trx.Rollback()
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
			_ = trx.Rollback()
			zap.L().Error("Failed cancel milestones (contract)", zap.String("contract_id", contractID.String()), zap.Error(err))
			return errors.New("cascade milestone cancel failed: " + err.Error())
		}
		// Batch cancel tasks
		if err := taskRepo.UpdateByCondition(ctx, func(db *gorm.DB) *gorm.DB {
			return db.Where("milestone_id IN (?) AND status <> ?", db.Select("id").Model(&model.Milestone{}).Where("campaign_id = ?", camp.ID), enum.TaskStatusCancelled)
		}, map[string]any{"status": enum.TaskStatusCancelled}); err != nil {
			_ = trx.Rollback()
			zap.L().Error("Failed cancel tasks (contract)", zap.String("contract_id", contractID.String()), zap.Error(err))
			return errors.New("cascade task cancel failed: " + err.Error())
		}
		// Batch inactivate products
		if err := productRepo.UpdateByCondition(ctx, func(db *gorm.DB) *gorm.DB {
			return db.Where("task_id IN (?) AND status <> ?", db.Select("t.id").Table("tasks as t").Where("t.milestone_id IN (?)", db.Select("id").Model(&model.Milestone{}).Where("campaign_id = ?", camp.ID)), enum.ProductStatusInactived)
		}, map[string]any{"status": enum.ProductStatusInactived}); err != nil {
			_ = trx.Rollback()
			zap.L().Error("Failed inactivate products (contract)", zap.String("contract_id", contractID.String()), zap.Error(err))
			return errors.New("cascade product inactivate failed: " + err.Error())
		}

		// Reflect memory
		camp.Status = enum.CampaignCanceled
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
		contractCreatePaymentProducer, err := t.rabbitMQ.GetProducer("")
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

func NewStateTransferService(
	dbReg *gormrepository.DatabaseRegistry,
	uow irepository.UnitOfWork,
) iservice.StateTransferService {
	return &stateTransferService{
		contractRepository:  dbReg.ContractRepository,
		campaignRepository:  dbReg.CampaignRepository,
		milestoneRepository: dbReg.MilestoneRepository,
		taskRepository:      dbReg.TaskRepository,
		productRepository:   dbReg.ProductRepository,
		uow:                 uow,
	}
}
