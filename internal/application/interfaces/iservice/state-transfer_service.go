package iservice

import (
	"core-backend/internal/domain/enum"
	"github.com/google/uuid"
)

type StateTransferService interface {
	MoveTaskToState(taskID uuid.UUID, targetState enum.TaskStatus, updatedBy uuid.UUID) error
	MoveProductToState(productID uuid.UUID, targetState enum.ProductStatus, updatedBy uuid.UUID) error
	MoveMileStoneToState(mileStoneID uuid.UUID, targetState enum.MilestoneStatus, updatedBy uuid.UUID) error
	MoveCampaignToState(campaignID uuid.UUID, targetState enum.CampaignStatus, updatedBy uuid.UUID) error
	MoveContractToState(contractID uuid.UUID, targetState enum.ContractStatus, updatedBy uuid.UUID) error
}
