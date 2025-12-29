package requests

import (
	"core-backend/internal/domain/enum"

	"github.com/google/uuid"
)

// BroadcastToUserRequest represents a request to broadcast a notification to a specific user
type BroadcastToUserRequest struct {
	UserID   uuid.UUID                  `json:"user_id" binding:"required" example:"123e4567-e89b-12d3-a456-426614174000"`
	Title    string                     `json:"title" binding:"required" example:"Title"`
	Body     string                     `json:"body" binding:"required" example:"Notification body content"`
	Data     map[string]string          `json:"data"`
	Types    []enum.NotificationType    `json:"types" example:"EMAIL"`
	Severity *enum.NotificationSeverity `json:"severity" example:"INFO"`
}

// BroadcastToAllRequest represents a request to broadcast a notification to all users
type BroadcastToAllRequest struct {
	Title string            `json:"title" binding:"required"`
	Body  string            `json:"body" binding:"required"`
	Data  map[string]string `json:"data"`
	Role  *enum.UserRole    `json:"role"` // Optional: Filter by role
}
