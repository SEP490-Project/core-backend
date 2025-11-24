package ai

import (
	"bufio"
	"bytes"
	"context"
	"core-backend/config"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"go.uber.org/zap"
)

// OpenAIClient implements AIStrategy for OpenAI-compatible APIs (OpenRouter, Moonshot, etc.)
type OpenAIClient struct {
	httpClient *http.Client
	config     config.AIProviderConfig
}

func NewOpenAIClient(httpClient *http.Client, config config.AIProviderConfig) *OpenAIClient {
	return &OpenAIClient{
		httpClient: httpClient,
		config:     config,
	}
}

type openAIChatRequest struct {
	Model          string             `json:"model"`
	Messages       []requests.Message `json:"messages"`
	Temperature    float64            `json:"temperature,omitempty"`
	MaxTokens      int                `json:"max_tokens,omitempty"`
	Stream         bool               `json:"stream,omitempty"`
	ResponseFormat *responseFormat    `json:"response_format,omitempty"`
}

type responseFormat struct {
	Type string `json:"type"`
}

type openAIChatResponse struct {
	Choices []struct {
		Message requests.Message `json:"message"`
	} `json:"choices"`
	Usage responses.Usage `json:"usage"`
	Error *struct {
		Message string      `json:"message"`
		Type    string      `json:"type"`
		Code    interface{} `json:"code"`
	} `json:"error,omitempty"`
}

func (c *OpenAIClient) Send(ctx context.Context, req *requests.ChatRequest) (*responses.ChatResponse, error) {
	url := fmt.Sprintf("%s/chat/completions", c.config.BaseURL)

	openAIReq := openAIChatRequest{
		Model:       req.Model,
		Messages:    req.Messages,
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
	}

	if req.JSONMode {
		openAIReq.ResponseFormat = &responseFormat{Type: "json_object"}
	}

	bodyBytes, err := json.Marshal(openAIReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.config.APIKey))

	// OpenRouter specific headers (optional but good practice)
	httpReq.Header.Set("HTTP-Referer", "https://core-backend.sep490") // Replace with actual site
	httpReq.Header.Set("X-Title", "Core Backend")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		zap.L().Error("OpenAI API error",
			zap.Int("status", resp.StatusCode),
			zap.String("body", string(respBody)))
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var openAIResp openAIChatResponse
	if err := json.Unmarshal(respBody, &openAIResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if openAIResp.Error != nil {
		return nil, fmt.Errorf("API error: %s", openAIResp.Error.Message)
	}

	if len(openAIResp.Choices) == 0 {
		return nil, fmt.Errorf("no choices returned from API")
	}

	return &responses.ChatResponse{
		Content: openAIResp.Choices[0].Message.Content,
		Usage:   openAIResp.Usage,
	}, nil
}

type openAIStreamResponse struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
}

func (c *OpenAIClient) Stream(ctx context.Context, req *requests.ChatRequest) (<-chan *responses.ChatResponse, error) {
	url := fmt.Sprintf("%s/chat/completions", c.config.BaseURL)

	openAIReq := openAIChatRequest{
		Model:       req.Model,
		Messages:    req.Messages,
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
		Stream:      true,
	}

	if req.JSONMode {
		openAIReq.ResponseFormat = &responseFormat{Type: "json_object"}
	}

	bodyBytes, err := json.Marshal(openAIReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.config.APIKey))
	httpReq.Header.Set("HTTP-Referer", "https://core-backend.sep490")
	httpReq.Header.Set("X-Title", "Core Backend")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	streamChan := make(chan *responses.ChatResponse)

	go func() {
		defer resp.Body.Close()
		defer close(streamChan)

		reader := bufio.NewReader(resp.Body)
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				if err != io.EOF {
					zap.L().Error("Error reading stream", zap.Error(err))
				}
				return
			}

			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}

			if !strings.HasPrefix(line, "data: ") {
				continue
			}

			data := strings.TrimPrefix(line, "data: ")
			if data == "[DONE]" {
				return
			}

			var streamResp openAIStreamResponse
			if err := json.Unmarshal([]byte(data), &streamResp); err != nil {
				zap.L().Error("Failed to unmarshal stream data", zap.Error(err), zap.String("data", data))
				continue
			}

			if len(streamResp.Choices) > 0 {
				content := streamResp.Choices[0].Delta.Content
				if content != "" {
					streamChan <- &responses.ChatResponse{
						Content: content,
					}
				}
			}
		}
	}()

	return streamChan, nil
}
