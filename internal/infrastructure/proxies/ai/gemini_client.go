package ai

import (
	"context"
	"core-backend/config"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"fmt"
	"net/http"

	"go.uber.org/zap"
	"google.golang.org/genai"
)

// GeminiClient implements AIStrategy for Google's Gemini API using the official Go SDK
type GeminiClient struct {
	client *genai.Client
	config config.AIProviderConfig
}

func NewGeminiClient(httpClient *http.Client, config config.AIProviderConfig) *GeminiClient {
	// Initialize the Gemini client
	// Note: We use context.Background() here as this is initialization.
	// If initialization fails (e.g. invalid API key format), we panic as the service cannot function.
	ctx := context.Background()
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  config.APIKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		panic(fmt.Sprintf("failed to initialize Gemini client: %v", err))
	}

	return &GeminiClient{
		client: client,
		config: config,
	}
}

func (c *GeminiClient) Send(ctx context.Context, req *requests.ChatRequest) (*responses.ChatResponse, error) {
	genaiConfig := c.mapConfig(req)
	contents := c.mapMessages(req.Messages)

	resp, err := c.client.Models.GenerateContent(ctx, req.Model, contents, genaiConfig)
	if err != nil {
		zap.L().Error("Gemini API error", zap.Error(err))
		return nil, fmt.Errorf("failed to generate content: %w", err)
	}

	return c.mapResponse(resp)
}

func (c *GeminiClient) Stream(ctx context.Context, req *requests.ChatRequest) (<-chan *responses.ChatResponse, error) {
	genaiConfig := c.mapConfig(req)
	contents := c.mapMessages(req.Messages)

	// Create a channel to stream responses
	streamChan := make(chan *responses.ChatResponse)

	go func() {
		defer close(streamChan)

		// GenerateContentStream returns an iterator
		iter := c.client.Models.GenerateContentStream(ctx, req.Model, contents, genaiConfig)

		for resp, err := range iter {
			if err != nil {
				zap.L().Error("Gemini stream error", zap.Error(err))
				return
			}

			chatResp, err := c.mapResponse(resp)
			if err != nil {
				zap.L().Error("Failed to map stream response", zap.Error(err))
				continue
			}

			streamChan <- chatResp
		}
	}()

	return streamChan, nil
}

// Helper functions

func (c *GeminiClient) mapConfig(req *requests.ChatRequest) *genai.GenerateContentConfig {
	cfg := &genai.GenerateContentConfig{
		Temperature: genai.Ptr(float32(req.Temperature)),
	}

	if req.MaxTokens > 0 {
		cfg.MaxOutputTokens = int32(req.MaxTokens)
	}

	if req.JSONMode {
		cfg.ResponseMIMEType = "application/json"
	}

	return cfg
}

func (c *GeminiClient) mapMessages(msgs []requests.Message) []*genai.Content {
	var contents []*genai.Content
	for _, msg := range msgs {
		role := genai.RoleUser
		if msg.Role == "model" || msg.Role == "assistant" {
			role = genai.RoleModel
		}

		contents = append(contents, &genai.Content{
			Role: role,
			Parts: []*genai.Part{
				{Text: msg.Content},
			},
		})
	}
	return contents
}

func (c *GeminiClient) mapResponse(resp *genai.GenerateContentResponse) (*responses.ChatResponse, error) {
	if len(resp.Candidates) == 0 {
		return nil, fmt.Errorf("no candidates returned")
	}

	candidate := resp.Candidates[0]
	if candidate.Content == nil || len(candidate.Content.Parts) == 0 {
		// It's possible to get empty content in some stream chunks or safety blocks
		return &responses.ChatResponse{Content: ""}, nil
	}

	// Concatenate all text parts
	var content string
	for _, part := range candidate.Content.Parts {
		content += part.Text
	}

	chatResp := &responses.ChatResponse{
		Content: content,
	}

	if resp.UsageMetadata != nil {
		chatResp.Usage = responses.Usage{
			PromptTokens:     int(resp.UsageMetadata.PromptTokenCount),
			CompletionTokens: int(resp.UsageMetadata.CandidatesTokenCount),
			TotalTokens:      int(resp.UsageMetadata.TotalTokenCount),
		}
	}

	return chatResp, nil
}
