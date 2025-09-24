package model

import (
	"core-backend/internal/domain/enum"
	"errors"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type RefundRequest struct {
	ID         uuid.UUID         `json:"id" gorm:"type:uuid;column:id;primaryKey;default"`
	OrderID    *uuid.UUID        `json:"order_id" gorm:"type:uuid;column:order_id"`
	PreOrderID *uuid.UUID        `json:"pre_order_id" gorm:"type:uuid;column:pre_order_id"`
	Reason     string            `json:"reason" gorm:"type:text;column:reason;not null"`
	Amount     *float64          `json:"amount" gorm:"column:amount"`
	Status     enum.RefundStatus `json:"status" gorm:"column:status;not null;check:status in ('PENDING', 'APPROVED', 'REJECTED', 'COMPLETED')"`
	CreatedAt  time.Time         `json:"created_at" gorm:"column:created_at;autoCreateTime"`
	UpdatedAt  time.Time         `json:"updated_at" gorm:"column:updated_at;autoUpdateTime"`
	DeletedAt  gorm.DeletedAt    `json:"deleted_at" gorm:"column:deleted_at;index"`
}

func (RefundRequest) TableName() string { return "refund_request" }

func (rr *RefundRequest) BeforeCreate(tx *gorm.DB) (err error) {
	if rr.ID == uuid.Nil {
		rr.ID = uuid.New()
	}
	if rr.Amount != nil && *rr.Amount < 0 {
		zap.L().Warn("Amount is less than 0, setting to 0")
		*rr.Amount = 0
	}
	if err := validateOnlyOneOrderType(rr); err != nil {
		return err
	}

	return nil
}

func (rr *RefundRequest) BeforeUpdate(tx *gorm.DB) (err error) {
	if rr.Amount != nil && *rr.Amount < 0 {
		zap.L().Warn("Amount is less than 0, setting to 0")
	}
	if err := validateOnlyOneOrderType(rr); err != nil {
		return err
	}

	return nil
}

func validateOnlyOneOrderType(rr *RefundRequest) error {
	if (rr.OrderID != nil && rr.PreOrderID != nil) || (rr.OrderID == nil && rr.PreOrderID == nil) {
		return errors.New("exactly one of OrderID or PreOrderID must be set")
	}
	return nil
}
