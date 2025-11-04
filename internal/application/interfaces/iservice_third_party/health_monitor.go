package iservice_third_party

import (
	"context"
	"time"
)

// ServiceHealth represents health status of a service
type ServiceHealth struct {
	Name          string
	IsHealthy     bool
	LastCheckTime time.Time
	LastError     error
	Details       map[string]any
}

// HealthMonitor defines the interface for infrastructure health monitoring
type HealthMonitor interface {
	// CheckAllServices checks the health of all infrastructure services
	CheckAllServices(ctx context.Context) map[string]ServiceHealth

	// CheckTimescaleDB checks if TimescaleDB extension and hypertables are working correctly
	CheckTimescaleDB(ctx context.Context) ServiceHealth

	// IsEmailHealthy returns true if the email service is healthy
	IsEmailHealthy() bool

	// IsFCMHealthy returns true if the FCM service is healthy
	IsFCMHealthy() bool

	// GetEmailHealth returns the current health status of the email service
	GetEmailHealth() ServiceHealth

	// GetFCMHealth returns the current health status of the FCM service
	GetFCMHealth() ServiceHealth
}
