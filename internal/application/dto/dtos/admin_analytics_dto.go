package dtos

import "time"

// UserGrowthResult represents user growth query result
type UserGrowthResult struct {
	Date     time.Time
	NewUsers int64
	Total    int64
}

// GrowthTrendResult represents growth trend query result
type GrowthTrendResult struct {
	Date         time.Time
	NewUsers     int64
	NewOrders    int64
	NewContracts int64
	Revenue      float64
}
