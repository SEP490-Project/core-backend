package iservice

import (
	"github.com/google/uuid"
)

// SSEMessage represents a Server-Sent Event message
type SSEMessage struct {
	Event string `json:"event"`
	Data  any    `json:"data"`
}

// SSEService defines the interface for Server-Sent Events operations
type SSEService interface {
	// SendUnreadCount sends the unread notification count to the user
	SendUnreadCount(userID uuid.UUID, count int64) error

	// SendEvent sends a general event to the user
	SendEvent(userID uuid.UUID, event string, data any) error

	// Subscribe subscribes a user to real-time updates
	Subscribe(userID uuid.UUID) (<-chan SSEMessage, func())
}
