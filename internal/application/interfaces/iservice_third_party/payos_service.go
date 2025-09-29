package iservice_third_party

import (
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
)

type PayOSService interface {
	GeneratePayOSLink(req requests.PaymentRequest) (*responses.PayOSWrapperResponse[responses.PayOSLinkResponse], error)
	CancelPayOSLink(paymentId string, cancellationReason string) (bool, error)
	GetPayOSOrderInfo(orderId string) (*responses.PayOSWrapperResponse[responses.PayOSOrderInfoResponse], error)
}
