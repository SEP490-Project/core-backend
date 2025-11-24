package ai

import (
	"context"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
)

// AIStrategy defines the interface that all AI provider implementations must follow
type AIStrategy interface {
	// Send sends a chat request to the specific AI provider
	Send(ctx context.Context, req *requests.ChatRequest) (*responses.ChatResponse, error)
	// Stream sends a chat request and returns a channel for streaming responses
	Stream(ctx context.Context, req *requests.ChatRequest) (<-chan *responses.ChatResponse, error)
}
