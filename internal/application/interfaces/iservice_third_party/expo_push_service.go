package iservice_third_party

import "context"

// ExpoReceipt represents the delivery status of a push notification
type ExpoReceipt struct {
	Status  string         `json:"status"` // "ok" or "error"
	Message string         `json:"message,omitempty"`
	Details map[string]any `json:"details,omitempty"`
}

// ExpoPushService defines the contract for Expo push notification operations
type ExpoPushService interface {
	// IsExpoToken checks if a token is an Expo push token
	IsExpoToken(token string) bool

	// ValidateExpoToken validates the format of an Expo push token
	ValidateExpoToken(token string) error

	// SendToToken sends a push notification to a specific Expo token
	SendToToken(ctx context.Context, token, title, body string, data map[string]string) error

	// SendMulticast sends a push notification to multiple Expo tokens
	// Returns successCount, failureCount, and a list of invalid tokens
	SendMulticast(ctx context.Context, tokens []string, title, body string, data map[string]string) (successCount int, failureCount int, invalidTokens []string, err error)

	// SendWithPriority sends a notification with specific priority
	SendWithPriority(ctx context.Context, token, title, body string, data map[string]string, priority string) error

	// GetReceipts fetches delivery receipts for sent notifications
	GetReceipts(ctx context.Context, ticketIDs []string) (map[string]ExpoReceipt, error)

	// Health returns the health status of the Expo service
	Health(ctx context.Context) ServiceHealth
}
