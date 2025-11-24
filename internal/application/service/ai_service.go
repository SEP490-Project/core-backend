package service

import (
	"bytes"
	"context"
	"core-backend/config"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/iproxies"
	"core-backend/internal/application/interfaces/iservice"
	"slices"
	"text/template"

	"go.uber.org/zap"
)

type AIService struct {
	config   *config.AppConfig
	aiClient iproxies.AIClientManager
}

func NewAIService(config *config.AppConfig, aiClient iproxies.AIClientManager) iservice.AIService {
	return &AIService{
		aiClient: aiClient,
		config:   config,
	}
}

func (s *AIService) getPromptTemplate() string {
	tmpl := s.config.AdminConfig.ContentGenerationPromptTemplate
	if tmpl == "" {
		return `You are a professional social media content creator.
Create a post based on the following context:
{{.Context}}

Requirements:
- Tone: {{.Tone}}
- Platform: {{.Platform}}

**CRITICAL INSTRUCTION:**
Your output must be ONLY the Tiptap JSON object. Do not include any other text. Follow the structure precisely.

**JSON Example with Formatting:**
{
  "type": "doc",
  "content": [
    {
      "type": "paragraph",
      "content": [
        {
          "type": "text",
          "text": "Example text content. "
        },
        {
          "type": "text",
          "marks": [
            {
              "type": "bold"
            }
          ],
          "text": "Bold text example"
        }
      ]
    }
  ]
}`
	}
	return tmpl
}

func (s *AIService) constructPrompt(req *requests.GenerateContentRequest) (string, error) {
	tmplStr := s.getPromptTemplate()
	tmpl, err := template.New("prompt").Parse(tmplStr)
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, req); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func (s *AIService) Generate(ctx context.Context, req *requests.GenerateRequest) (*responses.ChatResponse, error) {
	chatReq := &requests.ChatRequest{
		Model: req.Model,
		Messages: []requests.Message{
			{Role: "user", Content: req.Prompt},
		},
		JSONMode: req.JSONMode,
	}

	return s.aiClient.GenerateChatCompletion(ctx, chatReq)
}

func (s *AIService) Stream(ctx context.Context, req *requests.GenerateRequest) (<-chan *responses.ChatResponse, error) {
	chatReq := &requests.ChatRequest{
		Model: req.Model,
		Messages: []requests.Message{
			{Role: "user", Content: req.Prompt},
		},
		JSONMode: req.JSONMode,
	}

	return s.aiClient.StreamChatCompletion(ctx, chatReq)
}

func (s *AIService) GenerateContent(ctx context.Context, req *requests.GenerateContentRequest) (*responses.ChatResponse, error) {
	prompt, err := s.constructPrompt(req)
	if err != nil {
		return nil, err
	}

	chatReq := &requests.ChatRequest{
		Model: req.Model,
		Messages: []requests.Message{
			{Role: "user", Content: prompt},
		},
		JSONMode: true, // Enforce JSON for TipTap format
	}

	return s.aiClient.GenerateChatCompletion(ctx, chatReq)
}

func (s *AIService) StreamContent(ctx context.Context, req *requests.GenerateContentRequest) (<-chan *responses.ChatResponse, error) {
	prompt, err := s.constructPrompt(req)
	if err != nil {
		return nil, err
	}

	chatReq := &requests.ChatRequest{
		Model: req.Model,
		Messages: []requests.Message{
			{Role: "user", Content: prompt},
		},
		JSONMode: true,
	}

	return s.aiClient.StreamChatCompletion(ctx, chatReq)
}

func (s *AIService) GetSupportedModels(ctx context.Context) ([]responses.ModelProviderResponse, error) {
	zap.L().Info("AIService - GetSupportedModels called")

	models := s.config.AI.Models
	providers := s.config.AI.Providers

	// 1. Build a map of provider type to model IDs
	modelMap := make(map[string][]string)
	for _, m := range models {
		if _, ok := modelMap[m.Provider]; !ok {
			modelMap[m.Provider] = []string{}
		}
		modelMap[m.Provider] = append(modelMap[m.Provider], m.ID)
	}

	// 2. Construct the response
	resp := make([]responses.ModelProviderResponse, len(providers))
	index := 0
	for name, p := range providers {
		slices.Sort(modelMap[name])
		resp[index] = responses.ModelProviderResponse{
			Provider: name,
			BaseURL:  p.BaseURL,
			Enable:   p.APIKey != "",
			Models:   modelMap[name],
		}
		index++
	}

	return resp, nil
}
