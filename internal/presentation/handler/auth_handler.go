package handler

import (
	"net/http"
	"core-backend/internal/application/dto"
	"core-backend/internal/application/service"
	"core-backend/internal/presentation/dto/request"
	"core-backend/internal/presentation/dto/response"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

type AuthHandler struct {
	authService *service.AuthService
	validator   *validator.Validate
}

func NewAuthHandler(authService *service.AuthService) *AuthHandler {
	return &AuthHandler{
		authService: authService,
		validator:   validator.New(),
	}
}

// Login godoc
// @Summary      User Login
// @Description  Authenticate user with credentials
// @Tags         Authentication
// @Accept       json
// @Produce      json
// @Param        request body request.LoginRequest true "Login credentials"
// @Success      200 {object} response.APIResponse{data=dto.LoginResponse} "Login successful"
// @Failure      400 {object} response.APIResponse "Invalid request"
// @Failure      401 {object} response.APIResponse "Invalid credentials"
// @Router       /api/v1/auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var loginRequest request.LoginRequest
	if err := c.ShouldBindJSON(&loginRequest); err != nil {
		response := response.ErrorResponse("Invalid request format: "+err.Error(), http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	// Validate request
	if err := h.validator.Struct(&loginRequest); err != nil {
		response := response.ErrorResponse("Validation failed: "+err.Error(), http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	// Convert to application DTO
	appLoginRequest := &dto.LoginRequest{
		LoginIdentifier:   loginRequest.LoginIdentifier,
		Password:          loginRequest.Password,
		DeviceFingerprint: loginRequest.DeviceFingerprint,
	}

	// Call auth service
	loginResponse, err := h.authService.Login(appLoginRequest)
	if err != nil {
		response := response.ErrorResponse("Login failed: "+err.Error(), http.StatusUnauthorized)
		c.JSON(http.StatusUnauthorized, response)
		return
	}

	response := response.SuccessResponse("Login successful", http.StatusOK, loginResponse)
	c.JSON(http.StatusOK, response)
}

// SignUp godoc
// @Summary      User Registration
// @Description  Register a new user account
// @Tags         Authentication
// @Accept       json
// @Produce      json
// @Param        request body request.SignUpRequest true "User registration data"
// @Success      201 {object} response.APIResponse{data=dto.SignUpResponse} "User created successfully"
// @Failure      400 {object} response.APIResponse "Invalid request"
// @Failure      409 {object} response.APIResponse "User already exists"
// @Router       /api/v1/auth/signup [post]
func (h *AuthHandler) SignUp(c *gin.Context) {
	var signUpRequest request.SignUpRequest
	if err := c.ShouldBindJSON(&signUpRequest); err != nil {
		response := response.ErrorResponse("Invalid request format: "+err.Error(), http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	// Validate request
	if err := h.validator.Struct(&signUpRequest); err != nil {
		response := response.ErrorResponse("Validation failed: "+err.Error(), http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	// Convert to application DTO
	appSignUpRequest := &dto.SignUpRequest{
		Username: signUpRequest.Username,
		Email:    signUpRequest.Email,
		Password: signUpRequest.Password,
	}

	// Call auth service
	signUpResponse, err := h.authService.SignUp(appSignUpRequest)
	if err != nil {
		response := response.ErrorResponse("Sign up failed: "+err.Error(), http.StatusConflict)
		c.JSON(http.StatusConflict, response)
		return
	}

	response := response.SuccessResponse("User created successfully", http.StatusCreated, signUpResponse)
	c.JSON(http.StatusCreated, response)
}

// RefreshToken godoc
// @Summary      Refresh Access Token
// @Description  Generate new access token using refresh token
// @Tags         Authentication
// @Accept       json
// @Produce      json
// @Param        request body request.RefreshTokenRequest true "Refresh token"
// @Success      200 {object} response.APIResponse{data=dto.LoginResponse} "Token refreshed successfully"
// @Failure      400 {object} response.APIResponse "Invalid request"
// @Failure      401 {object} response.APIResponse "Invalid or expired refresh token"
// @Router       /api/v1/auth/refresh [post]
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var refreshRequest request.RefreshTokenRequest
	if err := c.ShouldBindJSON(&refreshRequest); err != nil {
		response := response.ErrorResponse("Invalid request format: "+err.Error(), http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	// Validate request
	if err := h.validator.Struct(&refreshRequest); err != nil {
		response := response.ErrorResponse("Validation failed: "+err.Error(), http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	// Convert to application DTO
	appRefreshRequest := &dto.RefreshTokenRequest{
		RefreshToken: refreshRequest.RefreshToken,
	}

	// Call auth service
	loginResponse, err := h.authService.RefreshToken(appRefreshRequest)
	if err != nil {
		response := response.ErrorResponse("Token refresh failed: "+err.Error(), http.StatusUnauthorized)
		c.JSON(http.StatusUnauthorized, response)
		return
	}

	response := response.SuccessResponse("Token refreshed successfully", http.StatusOK, loginResponse)
	c.JSON(http.StatusOK, response)
}

// Logout godoc
// @Summary      User Logout
// @Description  Logout user and invalidate refresh token
// @Tags         Authentication
// @Accept       json
// @Produce      json
// @Param        request body dto.LogoutRequest true "Logout request"
// @Success      200 {object} response.APIResponse{data=dto.LogoutResponse} "Logout successful"
// @Failure      400 {object} response.APIResponse "Invalid request"
// @Security     BearerAuth
// @Router       /api/v1/auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	var request dto.LogoutRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		response := response.ErrorResponse("Invalid request format: "+err.Error(), http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	// Call auth service
	logoutResponse, err := h.authService.Logout(&request)
	if err != nil {
		response := response.ErrorResponse("Logout failed: "+err.Error(), http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	response := response.SuccessResponse("Logout successful", http.StatusOK, logoutResponse)
	c.JSON(http.StatusOK, response)
}

// LogoutAll godoc
// @Summary      Logout All Sessions
// @Description  Logout user from all active sessions
// @Tags         Authentication
// @Accept       json
// @Produce      json
// @Success      200 {object} response.APIResponse{data=dto.LogoutResponse} "All sessions logged out successfully"
// @Failure      401 {object} response.APIResponse "Unauthorized"
// @Failure      500 {object} response.APIResponse "Internal server error"
// @Security     BearerAuth
// @Router       /api/v1/auth/logout-all [post]
func (h *AuthHandler) LogoutAll(c *gin.Context) {
	userIDStr, exists := c.Get("user_id")
	if !exists {
		response := response.ErrorResponse("Unauthorized: User ID not found in context", http.StatusUnauthorized)
		c.JSON(http.StatusUnauthorized, response)
		return
	}

	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		response := response.ErrorResponse("Invalid user ID: "+err.Error(), http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	// Call auth service
	logoutResponse, err := h.authService.LogoutAll(userID)
	if err != nil {
		response := response.ErrorResponse("Logout all failed: "+err.Error(), http.StatusInternalServerError)
		c.JSON(http.StatusInternalServerError, response)
		return
	}

	response := response.SuccessResponse("All sessions logged out successfully", http.StatusOK, logoutResponse)
	c.JSON(http.StatusOK, response)
}

// GetActiveSessions godoc
// @Summary      Get Active Sessions
// @Description  Retrieve user's active sessions
// @Tags         Authentication
// @Accept       json
// @Produce      json
// @Success      200 {object} response.APIResponse{data=[]dto.SessionInfo} "Active sessions retrieved successfully"
// @Failure      401 {object} response.APIResponse "Unauthorized"
// @Failure      500 {object} response.APIResponse "Internal server error"
// @Security     BearerAuth
// @Router       /api/v1/auth/sessions [get]
func (h *AuthHandler) GetActiveSessions(c *gin.Context) {
	userIDStr, exists := c.Get("user_id")
	if !exists {
		response := response.ErrorResponse("Unauthorized: User ID not found in context", http.StatusUnauthorized)
		c.JSON(http.StatusUnauthorized, response)
		return
	}

	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		response := response.ErrorResponse("Invalid user ID: "+err.Error(), http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	// Call auth service
	sessions, err := h.authService.GetActiveSessions(userID)
	if err != nil {
		response := response.ErrorResponse("Failed to get sessions: "+err.Error(), http.StatusInternalServerError)
		c.JSON(http.StatusInternalServerError, response)
		return
	}

	response := response.SuccessResponse("Active sessions retrieved successfully", http.StatusOK, sessions)
	c.JSON(http.StatusOK, response)
}

// RevokeSession godoc
// @Summary      Revoke Session
// @Description  Revoke a specific user session
// @Tags         Authentication
// @Accept       json
// @Produce      json
// @Param        sessionId path string true "Session ID"
// @Success      200 {object} response.APIResponse{data=dto.LogoutResponse} "Session revoked successfully"
// @Failure      400 {object} response.APIResponse "Invalid session ID"
// @Failure      500 {object} response.APIResponse "Internal server error"
// @Security     BearerAuth
// @Router       /api/v1/auth/sessions/{sessionId} [delete]
func (h *AuthHandler) RevokeSession(c *gin.Context) {
	sessionIDParam := c.Param("sessionId")
	sessionID, err := uuid.Parse(sessionIDParam)
	if err != nil {
		response := response.ErrorResponse("Invalid session ID: "+err.Error(), http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	// Call auth service
	logoutResponse, err := h.authService.RevokeSession(sessionID)
	if err != nil {
		response := response.ErrorResponse("Failed to revoke session: "+err.Error(), http.StatusInternalServerError)
		c.JSON(http.StatusInternalServerError, response)
		return
	}

	response := response.SuccessResponse("Session revoked successfully", http.StatusOK, logoutResponse)
	c.JSON(http.StatusOK, response)
}
