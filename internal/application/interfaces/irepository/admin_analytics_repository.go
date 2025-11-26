package irepository

import (
	"context"
	"core-backend/internal/application/dto/dtos"
	"time"
)

// AdminAnalyticsRepository defines the interface for admin analytics data access
type AdminAnalyticsRepository interface {
	// Users
	GetTotalUsersCount(ctx context.Context) (int64, error)
	GetActiveUsersCount(ctx context.Context, activeDays int) (int64, error)
	GetUserCountByRole(ctx context.Context, role string) (int64, error)
	GetNewUsersCount(ctx context.Context, startDate, endDate time.Time) (int64, error)
	GetUserGrowthTrend(ctx context.Context, granularity string, startDate, endDate *time.Time) ([]dtos.UserGrowthResult, error)

	// Brands
	GetTotalBrandsCount(ctx context.Context) (int64, error)
	GetActiveBrandsCount(ctx context.Context) (int64, error)

	// Contracts
	GetTotalContractsCount(ctx context.Context) (int64, error)
	GetContractCountByStatus(ctx context.Context, status string) (int64, error)
	GetTotalContractValue(ctx context.Context) (float64, error)
	GetCollectedContractAmount(ctx context.Context) (float64, error)
	GetPendingContractAmount(ctx context.Context) (float64, error)

	// Campaigns
	GetTotalCampaignsCount(ctx context.Context) (int64, error)
	GetCampaignCountByStatus(ctx context.Context, status string) (int64, error)
	GetTotalContentCount(ctx context.Context) (int64, error)
	GetPostedContentCount(ctx context.Context) (int64, error)

	// Revenue
	GetTotalPlatformRevenue(ctx context.Context, startDate, endDate *time.Time) (float64, error)
	GetPlatformRevenueByContractType(ctx context.Context, contractType string, startDate, endDate *time.Time) (float64, error)
	GetPlatformProductRevenue(ctx context.Context, productType string, startDate, endDate *time.Time) (float64, error)
	GetPlatformRevenueTrend(ctx context.Context, granularity string, startDate, endDate *time.Time) ([]dtos.RevenueTrendResult, error)

	// Orders
	GetTotalOrdersCount(ctx context.Context, startDate, endDate *time.Time) (int64, error)

	// Growth Trend
	GetGrowthTrend(ctx context.Context, granularity string, startDate, endDate *time.Time) ([]dtos.GrowthTrendResult, error)
}
