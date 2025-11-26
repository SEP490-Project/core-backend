package dtos

// BrandRevenueResult represents brand revenue query result
type BrandRevenueResult struct {
	BrandID      uuid.UUID
	BrandName    string
	TotalRevenue float64
	OrderCount   int64
	ProductCount int
}

// ProductRevenueResult represents product revenue query result
type ProductRevenueResult struct {
	ProductID    uuid.UUID
	ProductName  string
	BrandName    string
	ProductType  string
	TotalRevenue float64
	UnitsSold    int64
}

// RevenueTrendResult represents revenue trend query result
type RevenueTrendResult struct {
	Date              time.Time
	Revenue           float64
	OrderCount        int64
	AverageOrderValue float64
}

// RecentOrderResult represents recent order query result
type RecentOrderResult struct {
	OrderID      uuid.UUID
	OrderNumber  string
	CustomerName string
	TotalAmount  float64
	Status       string
	OrderType    string
	ItemCount    int
	CreatedAt    time.Time
}

// PaymentStatusResult represents payment status query result
type PaymentStatusResult struct {
	TotalPayments   int64
	PaidPayments    int64
	PendingPayments int64
	OverduePayments int64
	TotalAmount     float64
	PaidAmount      float64
	PendingAmount   float64
	OverdueAmount   float64
}
