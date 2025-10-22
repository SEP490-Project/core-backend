package iservice

import "core-backend/internal/application/dto/requests"

type OrderService interface {
	PlaceOrder(request requests.OrderRequest)
}
