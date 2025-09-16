package request

type LoginRequest struct {
	LoginIdentifier   string `json:"login_identifier" binding:"required"`
	Password          string `json:"password" binding:"required,min=8"`
	DeviceFingerprint string `json:"device_fingerprint"`
	RememberMe        bool   `json:"remember_me"`
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type SignUpRequest struct {
	Username        string `json:"username" binding:"required,min=3,max=50,alphanum"`
	Email          string `json:"email" binding:"required,email"`
	Password       string `json:"password" binding:"required,min=8"`
	ConfirmPassword string `json:"confirm_password" binding:"required,eqfield=Password"`
}

type LogoutRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}
