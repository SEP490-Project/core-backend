package requests

import "core-backend/internal/domain/enum"

// StaffOrdersQuery represents query parameters for staff order list endpoint.
// Used by swaggo to generate a single object for query parameters.
// swagger:model StaffOrdersQuery
type StaffOrdersQuery struct {
	// Page number (default: 1)
	// in: query
	// example: 1
	Page int `form:"page" example:"1"`

	// Number of items per page (default: 10, max: 100)
	// in: query
	// example: 10
	Limit int `form:"limit" example:"10"`

	// Search term to filter by orderID/paymentID/paymentBin
	// in: query
	// example: "ORDER123"
	Search string `form:"search" example:"ORDER123"`

	// Order status filter
	// in: query
	// example: "PAID"
	Status []enum.OrderStatus `form:"status" example:"PAID"`

	// Customer full name to filter
	// in: query
	// example: "John Doe"
	FullName string `form:"full_name" example:"John Doe"`

	// Customer phone number to filter
	// in: query
	// example: "0912345678"
	Phone string `form:"phone" example:"0912345678"`

	// GHN province id
	// in: query
	// example: 1
	ProvinceID string `form:"province_id" example:"1"`

	// GHN district id
	// in: query
	// example: 10
	DistrictID string `form:"district_id" example:"10"`

	// GHN ward code
	// in: query
	// example: "01234"
	WardCode string `form:"ward_code" example:"01234"`

	// Order type filter
	// in: query
	// example: "STANDARD"
	OrderType enum.ProductType `form:"order_type" example:"STANDARD"`
}

// SelfDeliveringQuery represents query parameters for staff order list endpoint.
// Used by swaggo to generate a single object for query parameters.
// swagger:model SelfDeliveringQuery
type SelfDeliveringQuery struct {
	// Page number (default: 1)
	// in: query
	// example: 1
	Page int `form:"page" example:"1"`

	// Number of items per page (default: 10, max: 100)
	// in: query
	// example: 10
	Limit int `form:"limit" example:"10"`

	// Search term to filter by orderID/paymentID/paymentBin
	// in: query
	// example: "ORDER123"
	Search string `form:"search" example:"ORDER123"`

	// Order status filter
	// in: query
	// example: "PAID"
	Status enum.OrderStatus `form:"status" example:"PAID"`

	// Customer full name to filter
	// in: query
	// example: "John Doe"
	FullName string `form:"full_name" example:"John Doe"`

	// Customer phone number to filter
	// in: query
	// example: "0912345678"
	Phone string `form:"phone" example:"0912345678"`

	// GHN province id
	// in: query
	// example: 1
	ProvinceID string `form:"province_id" example:"1"`

	// GHN district id
	// in: query
	// example: 10
	DistrictID string `form:"district_id" example:"10"`

	// GHN ward code
	// in: query
	// example: "01234"
	WardCode string `form:"ward_code" example:"01234"`
}

type StaffPreOrdersQuery struct {
	// Page number (default: 1)
	// in: query
	// example: 1
	Page int `form:"page" example:"1"`

	// Number of items per page (default: 10, max: 100)
	// in: query
	// example: 10
	Limit int `form:"limit" example:"10"`

	// Search term to filter by orderID/paymentID/paymentBin
	// in: query
	// example: "ORDER123"
	Search string `form:"search" example:"ORDER123"`

	// Order status filter
	// in: query
	// example: "PENDING PAID REFUNDED CONFIRMED CANCELLED SHIPPED IN_TRANSIT DELIVERED RECEIVED AWAITING_PICKUP"
	Status []enum.PreOrderStatus `form:"status" example:"PAID"`

	// Customer full name to filter
	// in: query
	// example: "John Doe"
	FullName string `form:"full_name" example:"John Doe"`

	// Customer phone number to filter
	// in: query
	// example: "0912345678"
	Phone string `form:"phone" example:"0912345678"`

	// GHN province id
	// in: query
	// example: 1
	ProvinceID string `form:"province_id" example:"1"`

	// GHN district id
	// in: query
	// example: 10
	DistrictID string `form:"district_id" example:"10"`

	// GHN ward code
	// in: query
	// example: "01234"
	WardCode string `form:"ward_code" example:"01234"`

	// Order type filter
	// in: query
	// example: "STANDARD"
	OrderType enum.ProductType `form:"order_type" example:"STANDARD"`
}
