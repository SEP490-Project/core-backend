package requests

// CreateChannelRequest represents the request data for creating a channel
type CreateChannelRequest struct {
	Code        string  `json:"code" validate:"required,min=3,max=100" example:"FACEBOOK"`
	Name        string  `json:"name" validate:"required,min=3,max=100" example:"Facebook"`
	Description *string `json:"description" validate:"omitempty,min=3,max=100" example:"This is a social media channel."`
	HomePageURL *string `json:"home_page_url" validate:"omitempty,url" example:"https://www.facebook.com"`
	IsActive    bool    `json:"is_active,omitempty" validate:"omitempty" example:"true"`
}

// UpdateChannelRequest represents the request data for updating a channel
type UpdateChannelRequest struct {
	Code        *string `json:"code,omitempty" validate:"required,min=3,max=100" example:"FACEBOOK"`
	Name        *string `json:"name,omitempty" validate:"omitempty,min=3,max=100" example:"Facebook"`
	Description *string `json:"description,omitempty" validate:"omitempty,min=3,max=100" example:"This is a social media channel."`
	HomePageURL *string `json:"home_page_url,omitempty" validate:"omitempty,url" example:"https://www.facebook.com"`
	IsActive    *bool   `json:"is_active,omitempty" validate:"omitempty" example:"true"`
}
