package requests

import "github.com/google/uuid"

// BroadcastToUserRequest represents a request to broadcast a notification to a specific user
type BroadcastToUserRequest struct {
	UserID   uuid.UUID         `json:"user_id" binding:"required"`
	Title    string            `json:"title" binding:"required"`
	Body     string            `json:"body" binding:"required"`
	Data     map[string]string `json:"data"`
	Channels []string          `json:"channels"` // Optional: "EMAIL", "PUSH", "IN_APP"
}

// BroadcastToAllRequest represents a request to broadcast a notification to all users
type BroadcastToAllRequest struct {
	Title string            `json:"title" binding:"required"`
	Body  string            `json:"body" binding:"required"`
	Data  map[string]string `json:"data"`
	Role  *string           `json:"role"` // Optional: Filter by role
}
