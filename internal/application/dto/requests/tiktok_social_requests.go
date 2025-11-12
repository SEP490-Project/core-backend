package requests

type TikTokOAuthRequest struct {
	RedirectURL string `json:"redirect_url,omitempty" form:"redirect_url" validate:"omitempty,url" example:"https://yourapp.com/oauth/callback"`
	CancelURL   string `json:"cancel_url,omitempty" form:"cancel_url" validate:"omitempty,url" example:"https://yourapp.com/oauth/cancel"`
	IsInternal  bool   `json:"is_internal,omitempty" form:"is_internal" validate:"omitempty,boolean" example:"true"`
}

// TikTokOAuthSuccessRequest represents the structure of a successful response from TikTok OAuth
type TikTokOAuthSuccessRequest struct {
	CancelURL   string `json:"cancel_url" form:"cancel_url"`
	RedirectURL string `json:"redirect_url" form:"redirect_url"`
	IsInternal  bool   `json:"is_internal,omitempty" form:"is_internal"`
	State       string `json:"state" form:"state"`
	Code        string `json:"code" form:"code"`

	BackendCallbackURL string `json:"-" form:"-"`
}

// TikTokOAuthErrorRequest represents the structure of an error response from TikTok OAuth
type TikTokOAuthErrorRequest struct {
	CancelURL        string `json:"cancel_url" form:"cancel_url"`
	RedirectURL      string `json:"redirect_url" form:"redirect_url"`
	IsInternal       bool   `json:"is_internal,omitempty" form:"is_internal"`
	Error            string `json:"error" form:"error"`
	ErrorDescription string `json:"error_description" form:"error_description"`
}
