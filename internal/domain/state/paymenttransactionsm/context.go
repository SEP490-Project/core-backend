package paymenttransactionsm

import (
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
)

// PaymentTransactionContext holds the current state and related data for FSM
type PaymentTransactionContext struct {
	State           PaymentTransactionState
	ReferenceType   enum.PaymentTransactionReferenceType
	ContractPayment *model.ContractPayment // Set if ReferenceType is CONTRACT_PAYMENT
	Order           *model.Order           // Set if ReferenceType is ORDER
}
