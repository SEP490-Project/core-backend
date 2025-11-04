package iservice_third_party

import (
	"context"

	"firebase.google.com/go/v4/messaging"
)

// FCMService defines the contract for Firebase Cloud Messaging operations
type FCMService interface {
	// SendToToken sends a push notification to a specific device token
	SendToToken(ctx context.Context, token, title, body string, data map[string]string) error

	// SendMulticast sends a push notification to multiple device tokens
	SendMulticast(ctx context.Context, tokens []string, title, body string, data map[string]string) (*messaging.BatchResponse, error)

	// SendWithPlatformConfig sends a notification with platform-specific configuration
	SendWithPlatformConfig(ctx context.Context, token, title, body string, data map[string]string, apnsConfig *messaging.APNSConfig, androidConfig *messaging.AndroidConfig) error

	// SendMulticastWithPlatformConfig sends to multiple tokens with platform-specific configuration
	SendMulticastWithPlatformConfig(ctx context.Context, tokens []string, title, body string, data map[string]string, apnsConfig *messaging.APNSConfig, androidConfig *messaging.AndroidConfig) (*messaging.BatchResponse, error)

	// Health returns the health status of the FCM service
	Health(ctx context.Context) ServiceHealth
}
