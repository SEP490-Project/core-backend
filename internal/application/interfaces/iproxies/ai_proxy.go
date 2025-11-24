package iproxies

import (
	"context"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
)

// AIClientManager defines the interface for managing AI model interactions
type AIClientManager interface {
	// GenerateChatCompletion generates a response for a chat conversation
	// It automatically selects the appropriate provider based on the model ID in the request
	GenerateChatCompletion(ctx context.Context, req *requests.ChatRequest) (*responses.ChatResponse, error)

	// StreamChatCompletion generates a streaming response for a chat conversation
	StreamChatCompletion(ctx context.Context, req *requests.ChatRequest) (<-chan *responses.ChatResponse, error)
}
