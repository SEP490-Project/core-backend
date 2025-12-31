package requests

// Message represents a single message in a chat conversation
type Message struct {
	Role    string `json:"role" example:"user"`      // "system", "user", "assistant"
	Content string `json:"content" example:"Hello!"` // The text content of the message
}

// ChatRequest represents a request to the AI chat interface
type ChatRequest struct {
	Model       string    `json:"model" validate:"required" example:"gemini-2.5-flash-lite"`
	Messages    []Message `json:"messages" validate:"required"`
	Temperature float64   `json:"temperature,omitempty" validate:"omitempty,min=0,max=1" example:"1.0"` // Controls randomness
	MaxTokens   int       `json:"max_tokens,omitempty" validate:"omitempty,min=1" example:"1024"`       // Maximum tokens in response
	JSONMode    bool      `json:"json_mode,omitempty" example:"false"`                                  // If true, enforces JSON output
}

type GenerateRequest struct {
	Prompt   string `json:"prompt" validate:"required" example:"Create a social media post about the benefits of AI."`
	Model    string `json:"model" validate:"required" example:"gemini-2.5-flash-lite"`
	Stream   bool   `json:"stream" example:"false"`
	JSONMode bool   `json:"json_mode" example:"false"`
}

type GenerateContentRequest struct {
	Context  string  `json:"context" validate:"required" example:"Promote our new AI-powered product that enhances productivity."`
	Current  *string `json:"current,omitempty"  example:"Discover the future of work with our AI solutions."`
	Tone     string  `json:"tone" validate:"required" example:"Professional and engaging"`
	Platform string  `json:"platform" validate:"required" example:"Facebook"`
	Model    string  `json:"model" validate:"required" example:"gemini-2.5-flash-lite"`
	Stream   bool    `json:"stream" example:"false"`
}
