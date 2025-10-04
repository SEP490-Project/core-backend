package requests

// LoginRequest represents login request data
type LoginRequest struct {
	LoginIdentifier string `json:"login_identifier" validate:"required" example:"user@example.com"`
	Password        string `json:"password" validate:"required,min=8" example:"password123"`
}

// RefreshTokenRequest represents refresh token request data
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
}

// SignUpRequest represents sign up request data
type SignUpRequest struct {
	Username string `json:"username" validate:"required,min=3,max=50,ascii" example:"john_doe"`
	Email    string `json:"email" validate:"required,email" example:"john@example.com"`
	Password string `json:"password" validate:"required,min=8" example:"password123"`
	FullName string `json:"full_name" validate:"omitempty,min=2,max=100" example:"John Doe"`
}

// LogoutRequest represents logout request data
type LogoutRequest struct {
	RefreshToken string `json:"refresh_token" validate:"omitempty" example:"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."`
}
