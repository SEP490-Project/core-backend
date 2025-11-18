package responses

import "core-backend/internal/domain/model"

type PreOrderResponse struct {
	model.PreOrder
	PaymentTx PaymentTransactionResponse
}
