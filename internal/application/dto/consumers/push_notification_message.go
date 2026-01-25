package consumers

import "github.com/google/uuid"

// PushNotificationMessage represents a push notification message consumed from RabbitMQ
type PushNotificationMessage struct {
	NotificationID uuid.UUID         `json:"notification_id"` // Reference to notifications table
	UserID         uuid.UUID         `json:"user_id"`         // Target user
	Title          string            `json:"title"`
	Body           string            `json:"body"`
	Data           map[string]string `json:"data,omitempty"`            // Custom data payload
	PlatformConfig *PlatformConfig   `json:"platform_config,omitempty"` // Platform-specific configurations

	// Optional ScheduleID if the notification is related to a scheduled task
	ScheduleID *uuid.UUID `json:"schedule_id,omitempty"`
}

// PlatformConfig contains platform-specific notification configurations
type PlatformConfig struct {
	IOS     *IOSConfig     `json:"ios,omitempty"`
	Android *AndroidConfig `json:"android,omitempty"`
}

// IOSConfig contains iOS-specific notification configuration (APNs)
type IOSConfig struct {
	Badge            *int           `json:"badge,omitempty"`             // Badge count to display
	Sound            string         `json:"sound,omitempty"`             // Sound to play (e.g., "default")
	Category         string         `json:"category,omitempty"`          // Notification category
	ThreadID         string         `json:"thread_id,omitempty"`         // Thread identifier for grouping
	ContentAvailable bool           `json:"content_available,omitempty"` // Background content-available flag
	MutableContent   bool           `json:"mutable_content,omitempty"`   // Enable notification service extension
	CustomData       map[string]any `json:"custom_data,omitempty"`       // Custom APNs payload
}

// AndroidConfig contains Android-specific notification configuration
type AndroidConfig struct {
	Priority    string            `json:"priority,omitempty"`     // Message priority: "high" or "normal"
	CollapseKey string            `json:"collapse_key,omitempty"` // Collapse key for message grouping
	TTL         string            `json:"ttl,omitempty"`          // Time-to-live duration (e.g., "3600s")
	ChannelID   string            `json:"channel_id,omitempty"`   // Android notification channel ID
	Sound       string            `json:"sound,omitempty"`        // Sound to play
	Color       string            `json:"color,omitempty"`        // Notification color (hex format)
	Icon        string            `json:"icon,omitempty"`         // Notification icon resource name
	Tag         string            `json:"tag,omitempty"`          // Notification tag for replacement
	ClickAction string            `json:"click_action,omitempty"` // Action on click (activity name)
	CustomData  map[string]string `json:"custom_data,omitempty"`  // Custom FCM data
}
