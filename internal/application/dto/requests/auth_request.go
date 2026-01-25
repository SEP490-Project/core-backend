package requests

import (
	"core-backend/internal/domain/enum"

	"github.com/google/uuid"
)

// LoginRequest represents login request data
type LoginRequest struct {
	LoginIdentifier string             `json:"login_identifier" validate:"required" example:"abc@gmail.com"`
	Password        string             `json:"password" validate:"required,min=8" example:"12345678"`
	DeviceToken     *string            `json:"device_token,omitempty" validate:"omitempty,min=10"`
	Platform        *enum.PlatformType `json:"platform,omitempty" validate:"omitempty,oneof=IOS ANDROID"`
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

// ForgotPasswordRequest is the request DTO for initiating password reset
type ForgotPasswordRequest struct {
	Email       string `json:"email" validate:"required,email"`
	FrontendURL string `json:"frontend_url" validate:"required,url"`
}

// ResetPasswordRequest is the request DTO for completing password reset via email link
type ResetPasswordRequest struct {
	Email       string `json:"email" validate:"required,email"`
	Token       string `json:"token" validate:"required"`
	NewPassword string `json:"new_password" validate:"required,min=8"`
}

// ChangePasswordRequest is the request DTO for authenticated users changing their password
type ChangePasswordRequest struct {
	UserID          uuid.UUID `json:"-" validate:"required,uuid"`
	CurrentPassword string    `json:"current_password" validate:"required"`
	NewPassword     string    `json:"new_password" validate:"required,min=8"`
}
