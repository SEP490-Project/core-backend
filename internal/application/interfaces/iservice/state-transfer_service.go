package iservice

import (
	"core-backend/internal/domain/enum"
	"github.com/google/uuid"
)

type StateTransferService interface {
	MoveTaskToState(taskID uuid.UUID, targetState enum.TaskStatus) error
	MoveProductToState(productID uuid.UUID, targetState enum.ProductStatus) error
	MoveMileStoneToState(mileStoneID uuid.UUID, targetState enum.MilestoneStatus) error
	MoveCampaignToState(campaignID uuid.UUID, targetState enum.CampaignStatus) error
	MoveContractToState(contractID uuid.UUID, targetState enum.ContractStatus) error
}
