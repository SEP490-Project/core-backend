package responses

// ChatResponse represents the response from the AI chat interface
type ChatResponse struct {
	Content string `json:"content" example:"Hello! How can I assist you today?"` // The generated response content
	Usage   Usage  `json:"usage"`
}

// Usage represents token usage statistics
type Usage struct {
	PromptTokens     int `json:"prompt_tokens" example:"50"`
	CompletionTokens int `json:"completion_tokens" example:"20"`
	TotalTokens      int `json:"total_tokens" example:"70"`
}

// ModelProviderResponse represents the response containing available models from a provider
type ModelProviderResponse struct {
	Provider string   `json:"provider" example:"openai"`                    // e.g., "openai", "gemini"
	BaseURL  string   `json:"base_url" example:"https://api.openai.com/v1"` // Base URL of the AI provider
	Enable   bool     `json:"enable" example:"true"`                        // Whether the provider based on the APIKey is provided
	Models   []string `json:"models"`                                       // List of model IDs provided by the provider
}
