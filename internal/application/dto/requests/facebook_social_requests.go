package requests

type FacebookOAuthRequest struct {
	RedirectURL string `json:"redirect_url,omitempty" form:"redirect_url" validate:"omitempty,url" example:"https://yourapp.com/oauth/callback"`
	CancelURL   string `json:"cancel_url,omitempty" form:"cancel_url" validate:"omitempty,url" example:"https://yourapp.com/oauth/cancel"`
	IsInternal  bool   `json:"is_internal,omitempty" form:"is_internal" validate:"omitempty,boolean" example:"true"`
}

// FacebookOAuthSuccessRequest represents the structure of a successful response from Facebook OAuth
type FacebookOAuthSuccessRequest struct {
	CancelURL   string `json:"cancel_url" form:"cancel_url"`
	RedirectURL string `json:"redirect_url" form:"redirect_url"`
	IsInternal  bool   `json:"is_internal,omitempty" form:"is_internal"`
	State       string `json:"state" form:"state"`
	Code        string `json:"code" form:"code"`

	BackendCallbackURL string `json:"-" form:"-"`
}

// FacebookOAuthErrorRequest represents the structure of an error response from Facebook OAuth
type FacebookOAuthErrorRequest struct {
	CancelURL        string `json:"cancel_url" form:"cancel_url"`
	RedirectURL      string `json:"redirect_url" form:"redirect_url"`
	IsInternal       bool   `json:"is_internal,omitempty" form:"is_internal"`
	ErrorReason      string `json:"error_reason" form:"error_reason"`
	Error            string `json:"error" form:"error"`
	ErrorDescription string `json:"error_description" form:"error_description"`
}
