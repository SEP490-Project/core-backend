package requests

import (
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type CreateModifiedHistoryRequest struct {
	ReferenceID   string `json:"reference_id" validate:"required,uuid" example:"550e8400-e29b-41d4-a716-446655440000"`
	ReferenceType string `json:"reference_type" validate:"required,oneof=CONTRACT CAMPAIGN MILESTONE TASK CONTENT PRODUCT BLOG" example:"TASK"`
	Operation     string `json:"operation" validate:"required,oneof=CREATE UPDATE DELETE" example:"CREATE"`
	Description   string `json:"description" validate:"required" example:"Task description"`
	ChangedByID   string `json:"changed_by" validate:"required,uuid" example:"550e8400-e29b-41d4-a716-446655440000"`
}

type UpdateModifiedHistoryRequest struct {
	ReferenceID   *string `json:"reference_id" validate:"required,uuid" example:"550e8400-e29b-41d4-a716-446655440000"`
	ReferenceType *string `json:"reference_type" validate:"required,oneof=CONTRACT CAMPAIGN MILESTONE TASK CONTENT PRODUCT BLOG" example:"TASK"`
	Operation     *string `json:"operation" validate:"required,oneof=CREATE UPDATE DELETE" example:"CREATE"`
	Status        *string `json:"status" validate:"required,oneof=IN_PROGRESS COMPLETED FAILED" example:"IN_PROGRESS"`
	Description   *string `json:"description" validate:"required" example:"Task description"`
	ChangedByID   *string `json:"changed_by" validate:"required,uuid" example:"550e8400-e29b-41d4-a716-446655440000"`
}

type ModifiedHistoryFilterRequest struct {
	PaginationRequest
	ReferenceID    *string    `json:"reference_id" validate:"omitempty,uuid" example:"550e8400-e29b-41d4-a716-446655440000"`
	ReferenceType  *string    `json:"reference_type" validate:"omitempty,oneof=CAMPAIGN MILESTONE TASK CONTENT PRODUCT BLOG" example:"TASK"`
	ChangedByID    *string    `json:"changed_by" validate:"omitempty,uuid" example:"550e8400-e29b-41d4-a716-446655440000"`
	StartChangedAt *time.Time `json:"start_changed_at" validate:"omitempty,datetime" example:"2023-01-01T00:00:00Z"`
	EndChangedAt   *time.Time `json:"end_changed_at" validate:"omitempty,datetime" example:"2023-12-31T23:59:59Z"`
}

func (cmhr *CreateModifiedHistoryRequest) ToModel() (result *model.ModifiedHistory, err error) {
	var referenceID, changedByID uuid.UUID
	referenceType := enum.ModifiedType(cmhr.ReferenceType)
	operation := enum.ModifiedOperation(cmhr.Operation)
	currentTime := time.Now()

	referenceID, err = uuid.Parse(cmhr.ReferenceID)
	if cmhr.ReferenceID != "" && err != nil {
		zap.L().Error("Failed to parse reference ID", zap.Error(err))
		return nil, err
	}
	if changedByID, err = uuid.Parse(cmhr.ChangedByID); err != nil {
		zap.L().Error("Failed to parse changed by ID", zap.Error(err))
		return nil, err
	}
	if !operation.IsValid() {
		zap.L().Error("Invalid operation", zap.String("operation", cmhr.Operation))
		return nil, errors.New("invalid operation")
	}
	if !referenceType.IsValid() {
		zap.L().Error("Invalid reference type", zap.String("reference_type", cmhr.ReferenceType))
		return nil, errors.New("invalid reference type")
	}
	if cmhr.Description == "" {
		cmhr.Description = fmt.Sprintf("User %s %s %s", cmhr.ChangedByID, cmhr.Operation, cmhr.ReferenceType)
	}

	result = &model.ModifiedHistory{
		ID:            uuid.New(),
		ReferenceID:   referenceID,
		ReferenceType: referenceType,
		Operation:     operation,
		Status:        enum.ModifiedStatusInProgress,
		Description:   cmhr.Description,
		ChangedByID:   &changedByID,
		ChangedAt:     &currentTime,
	}

	return
}

func (umhr *UpdateModifiedHistoryRequest) ToExistingModel(model *model.ModifiedHistory) (*model.ModifiedHistory, error) {
	if model == nil {
		return model, nil
	}

	if umhr.ReferenceID != nil {
		referenceID, err := uuid.Parse(*umhr.ReferenceID)
		if err != nil {
			zap.L().Error("Failed to parse reference ID", zap.Error(err))
			return nil, err
		}
		model.ReferenceID = referenceID
	}
	if umhr.ReferenceType != nil {
		referenceType := enum.ModifiedType(*umhr.ReferenceType)
		if referenceType.IsValid() {
			model.ReferenceType = referenceType
		} else {
			zap.L().Error("Invalid reference type", zap.String("reference_type", *umhr.ReferenceType))
			return nil, errors.New("invalid reference type")
		}
	}
	if umhr.Operation != nil {
		operation := enum.ModifiedOperation(*umhr.Operation)
		if operation.IsValid() {
			model.Operation = operation
		} else {
			zap.L().Error("Invalid operation", zap.String("operation", *umhr.Operation))
			return nil, errors.New("invalid operation")
		}
	}
	if umhr.Status != nil {
		status := enum.ModifiedStatus(*umhr.Status)
		if status.IsValid() {
			model.Status = status
		} else {
			zap.L().Error("Invalid status", zap.String("status", *umhr.Status))
			return nil, errors.New("invalid status")
		}
	}
	if umhr.Description != nil {
		model.Description = *umhr.Description
	}
	if umhr.ChangedByID != nil {
		changedByID, err := uuid.Parse(*umhr.ChangedByID)
		if err != nil {
			zap.L().Error("Failed to parse changed by ID", zap.Error(err))
			return nil, err
		}
		model.ChangedByID = &changedByID
	}

	return model, nil
}
