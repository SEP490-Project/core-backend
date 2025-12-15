package dtos

import (
	"time"

	"github.com/google/uuid"
)

// BrandProductMetrics represents brand product metrics
type BrandProductMetrics struct {
	ProductID   uuid.UUID
	ProductName string
	ProductType string
	Status      string
	OrderCount  int64
	UnitsSold   int64
	Revenue     float64
}

// BrandCampaignMetrics represents brand campaign metrics
type BrandCampaignMetrics struct {
	CampaignID       uuid.UUID
	CampaignName     string
	Status           string
	StartDate        *time.Time
	EndDate          *time.Time
	MilestoneCount   int64
	TaskCount        int64
	CompletedTasks   int64
	ContentCount     int64
	TotalViews       int64
	TotalEngagements int64
}

// BrandContentMetrics represents aggregated content metrics for a brand
type BrandContentMetrics struct {
	TotalContent   int64
	PostedContent  int64
	TotalViews     int64
	TotalLikes     int64
	TotalComments  int64
	TotalShares    int64
	EngagementRate float64
}

// BrandRevenueTrendResult represents brand revenue trend point
type BrandRevenueTrendResult struct {
	Date       time.Time
	OrderCount int64
	UnitsSold  int64
	Revenue    float64
}

// BrandAffiliateMetrics represents brand affiliate link metrics
type BrandAffiliateMetrics struct {
	TotalLinks  int64
	ActiveLinks int64
	TotalClicks int64
}

// BrandContractDetails represents contract details for a brand
type BrandContractDetails struct {
	ContractID     uuid.UUID
	ContractNumber string
	Type           string
	Status         string
	TotalValue     float64
	StartDate      *time.Time
	EndDate        *time.Time
	PaidAmount     float64
	PendingAmount  float64
	CampaignCount  int64
}

type BrandProductRating struct {
	ProductID     uuid.UUID
	ProductName   string
	Type          string
	AverageRating float64
}

type BrandTopSoldProducts struct {
	ProductID   uuid.UUID
	ProductName string
	TotalSold   int64
}
