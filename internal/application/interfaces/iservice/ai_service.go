package iservice

import (
	"context"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
)

type AIService interface {
	Generate(ctx context.Context, req *requests.GenerateRequest) (*responses.ChatResponse, error)
	Stream(ctx context.Context, req *requests.GenerateRequest) (<-chan *responses.ChatResponse, error)

	GenerateContent(ctx context.Context, req *requests.GenerateContentRequest) (*responses.ChatResponse, error)
	StreamContent(ctx context.Context, req *requests.GenerateContentRequest) (<-chan *responses.ChatResponse, error)

	GetSupportedModels(ctx context.Context) ([]responses.ModelProviderResponse, error)
}
