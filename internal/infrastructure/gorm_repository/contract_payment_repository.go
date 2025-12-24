package gormrepository

import (
	"context"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type contractPaymentRepository struct {
	*genericRepository[model.ContractPayment]
}

// GetNextContractPaymentFromCurrentPaymentID implements [irepository.ContractPaymentRepository].
func (c *contractPaymentRepository) GetNextUnpaidContractPaymentFromCurrentPaymentID(
	ctx context.Context, currentPaymentID uuid.UUID,
) (*model.ContractPayment, error) {
	var payment model.ContractPayment
	query := `
	SELECT next_p.*
	FROM contract_payments curr_p
	JOIN contract_payments next_p ON next_p.contract_id = curr_p.contract_id
	WHERE curr_p.id = ? 
	  AND next_p.id <> curr_p.id
	  AND next_p.status <> ?
	  AND next_p.deleted_at IS NULL
	  AND (next_p.due_date > curr_p.due_date OR (next_p.due_date = curr_p.due_date AND next_p.created_at > curr_p.created_at))
	ORDER BY next_p.due_date ASC, next_p.created_at ASC
	LIMIT 1
	`

	if err := c.db.
		WithContext(ctx).
		Raw(query, currentPaymentID, enum.ContractPaymentStatusPaid).
		Find(&payment).Error; err != nil {
		return nil, err
	}
	return &payment, nil
}

func NewContractPaymentRepository(db *gorm.DB) irepository.ContractPaymentRepository {
	return &contractPaymentRepository{
		genericRepository: &genericRepository[model.ContractPayment]{db: db},
	}
}
