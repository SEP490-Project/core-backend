// Package ai provides a manager for handling multiple AI service providers
// via different strategies.
package ai

import (
	"context"
	"core-backend/config"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/iproxies"
	"fmt"
	"net/http"
)

type aiClientManager struct {
	strategies map[string]AIStrategy
	modelMap   map[string]string
}

func NewAIClientManager(httpClient *http.Client, aiConfig *config.AIConfig) iproxies.AIClientManager {
	strategies := make(map[string]AIStrategy)

	for key, providerConfig := range aiConfig.Providers {
		switch providerConfig.Type {
		case "gemini":
			strategies[key] = NewGeminiClient(httpClient, providerConfig)
		case "openai":
			strategies[key] = NewOpenAIClient(httpClient, providerConfig)
		default:
			// Log warning or error for unknown provider type
			fmt.Printf("Unknown AI provider type: %s\n", providerConfig.Type)
		}
	}

	modelMap := make(map[string]string)
	for _, modelConfig := range aiConfig.Models {
		modelMap[modelConfig.ID] = modelConfig.Provider
	}

	return &aiClientManager{
		strategies: strategies,
		modelMap:   modelMap,
	}
}

func (m *aiClientManager) GenerateChatCompletion(ctx context.Context, req *requests.ChatRequest) (*responses.ChatResponse, error) {
	// 1. Resolve provider key from model ID
	providerKey, ok := m.modelMap[req.Model]
	if !ok {
		// Fallback: try to find a provider that supports this model directly if we had that info,
		// or just default to a specific one?
		// For now, return error if mapping not found.
		return nil, fmt.Errorf("no provider configured for model: %s", req.Model)
	}

	// 2. Get strategy
	strategy, ok := m.strategies[providerKey]
	if !ok {
		return nil, fmt.Errorf("strategy not found for provider key: %s", providerKey)
	}

	// 3. Execute
	return strategy.Send(ctx, req)
}

func (m *aiClientManager) StreamChatCompletion(ctx context.Context, req *requests.ChatRequest) (<-chan *responses.ChatResponse, error) {
	// 1. Resolve provider key from model ID
	providerKey, ok := m.modelMap[req.Model]
	if !ok {
		return nil, fmt.Errorf("no provider configured for model: %s", req.Model)
	}

	// 2. Get strategy
	strategy, ok := m.strategies[providerKey]
	if !ok {
		return nil, fmt.Errorf("strategy not found for provider key: %s", providerKey)
	}

	// 3. Execute
	return strategy.Stream(ctx, req)
}
