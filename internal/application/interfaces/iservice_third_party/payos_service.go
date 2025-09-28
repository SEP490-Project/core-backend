package iservice_third_party

import (
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
)

type PayOSService interface {
	GeneratePaymentLink(req requests.PaymentRequest) (*responses.PaymentResponse, error)
	VerifyPayment(paymentId string) (bool, error)
}
