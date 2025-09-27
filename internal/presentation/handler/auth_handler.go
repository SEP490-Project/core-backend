package handler

import (
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/iservice"
	"core-backend/pkg/utils"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

type AuthHandler struct {
	authService iservice.AuthService
	validator   *validator.Validate
}

func NewAuthHandler(authService iservice.AuthService) *AuthHandler {
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
// @Param        request body requests.LoginRequest true "Login credentials"
// @Success      200 {object} responses.APIResponse{data=responses.LoginResponse} "Login successful"
// @Failure      400 {object} responses.APIResponse "Invalid request"
// @Failure      401 {object} responses.APIResponse "Invalid credentials"
// @Router       /api/v1/auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var loginRequest requests.LoginRequest
	if err := c.ShouldBindJSON(&loginRequest); err != nil {
		response := responses.ErrorResponse("Invalid request format: "+err.Error(), http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	// Validate request
	if err := h.validator.Struct(&loginRequest); err != nil {
		response := responses.ErrorResponse("Validation failed: "+err.Error(), http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	// Call auth service
	loginResponse, err := h.authService.Login(c.Request.Context(), &loginRequest)
	if err != nil {
		response := responses.ErrorResponse("Login failed: "+err.Error(), http.StatusUnauthorized)
		c.JSON(http.StatusUnauthorized, response)
		return
	}

	response := responses.SuccessResponse("Login successful", nil, loginResponse)
	c.JSON(http.StatusOK, response)
}

// SignUp godoc
// @Summary      User Registration
// @Description  Register a new user account
// @Tags         Authentication
// @Accept       json
// @Produce      json
// @Param        request body requests.SignUpRequest true "User registration data"
// @Success      201 {object} responses.APIResponse{data=responses.SignUpResponse} "User created successfully"
// @Failure      400 {object} responses.APIResponse "Invalid request"
// @Failure      409 {object} responses.APIResponse "User already exists"
// @Router       /api/v1/auth/signup [post]
func (h *AuthHandler) SignUp(c *gin.Context) {
	var signUpRequest requests.SignUpRequest
	if err := c.ShouldBindJSON(&signUpRequest); err != nil {
		response := responses.ErrorResponse("Invalid request format: "+err.Error(), http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	// Validate request
	if err := h.validator.Struct(&signUpRequest); err != nil {
		response := responses.ErrorResponse("Validation failed: "+err.Error(), http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	// Convert to application DTO
	appSignUpRequest := &requests.SignUpRequest{
		Username: signUpRequest.Username,
		Email:    signUpRequest.Email,
		Password: signUpRequest.Password,
	}

	// Call auth service
	signUpResponse, err := h.authService.SignUp(c.Request.Context(), appSignUpRequest)
	if err != nil {
		response := responses.ErrorResponse("Sign up failed: "+err.Error(), http.StatusConflict)
		c.JSON(http.StatusConflict, response)
		return
	}

	// response := responses.SuccessResponse("User created successfully", *(http.StatusCreated, signUpResponse)
	// StatusCreated
	response := responses.SuccessResponse("User created successfully", utils.IntPtr(http.StatusCreated), signUpResponse)
	c.JSON(http.StatusCreated, response)
}

// RefreshToken godoc
// @Summary      Refresh Access Token
// @Description  Generate new access token using refresh token
// @Tags         Authentication
// @Accept       json
// @Produce      json
// @Param        request body requests.RefreshTokenRequest true "Refresh token"
// @Success      200 {object} responses.APIResponse{data=responses.LoginResponse} "Token refreshed successfully"
// @Failure      400 {object} responses.APIResponse "Invalid requests"
// @Failure      401 {object} responses.APIResponse "Invalid or expired refresh token"
// @Router       /api/v1/auth/refresh [post]
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var refreshRequest requests.RefreshTokenRequest
	if err := c.ShouldBindJSON(&refreshRequest); err != nil {
		response := responses.ErrorResponse("Invalid requests format: "+err.Error(), http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	// Validate requests
	if err := h.validator.Struct(&refreshRequest); err != nil {
		response := responses.ErrorResponse("Validation failed: "+err.Error(), http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	// Convert to application DTO
	appRefreshRequest := &requests.RefreshTokenRequest{
		RefreshToken: refreshRequest.RefreshToken,
	}

	// Call auth service
	loginResponse, err := h.authService.RefreshToken(c.Request.Context(), appRefreshRequest)
	if err != nil {
		response := responses.ErrorResponse("Token refresh failed: "+err.Error(), http.StatusUnauthorized)
		c.JSON(http.StatusUnauthorized, response)
		return
	}

	response := responses.SuccessResponse("Token refreshed successfully", nil, loginResponse)
	c.JSON(http.StatusOK, response)
}

// Logout godoc
// @Summary      User Logout
// @Description  Logout user and invalidate refresh token
// @Tags         Authentication
// @Accept       json
// @Produce      json
// @Param        requests body requests.LogoutRequest true "Logout requests"
// @Success      200 {object} responses.APIResponse{data=responses.LogoutResponse} "Logout successful"
// @Failure      400 {object} responses.APIResponse "Invalid requests"
// @Security     BearerAuth
// @Router       /api/v1/auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	var requests requests.LogoutRequest
	if err := c.ShouldBindJSON(&requests); err != nil {
		response := responses.ErrorResponse("Invalid requests format: "+err.Error(), http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	// Call auth service
	logoutResponse, err := h.authService.Logout(c.Request.Context(), &requests)
	if err != nil {
		response := responses.ErrorResponse("Logout failed: "+err.Error(), http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	response := responses.SuccessResponse("Logout successful", nil, logoutResponse)
	c.JSON(http.StatusOK, response)
}

// LogoutAll godoc
// @Summary      Logout All Sessions
// @Description  Logout user from all active sessions
// @Tags         Authentication
// @Accept       json
// @Produce      json
// @Success      200 {object} responses.APIResponse{data=responses.LogoutResponse} "All sessions logged out successfully"
// @Failure      401 {object} responses.APIResponse "Unauthorized"
// @Failure      500 {object} responses.APIResponse "Internal server error"
// @Security     BearerAuth
// @Router       /api/v1/auth/logout-all [post]
func (h *AuthHandler) LogoutAll(c *gin.Context) {
	userIDStr, exists := c.Get("user_id")
	if !exists {
		response := responses.ErrorResponse("Unauthorized: User ID not found in context", http.StatusUnauthorized)
		c.JSON(http.StatusUnauthorized, response)
		return
	}

	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		response := responses.ErrorResponse("Invalid user ID: "+err.Error(), http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	// Call auth service
	logoutResponse, err := h.authService.LogoutAll(c.Request.Context(), userID)
	if err != nil {
		response := responses.ErrorResponse("Logout all failed: "+err.Error(), http.StatusInternalServerError)
		c.JSON(http.StatusInternalServerError, response)
		return
	}

	response := responses.SuccessResponse("All sessions logged out successfully", nil, logoutResponse)
	c.JSON(http.StatusOK, response)
}

// GetActiveSessions godoc
// @Summary      Get Active Sessions
// @Description  Retrieve user's active sessions
// @Tags         Authentication
// @Accept       json
// @Produce      json
// @Success      200 {object} responses.APIResponse{data=[]responses.SessionInfo} "Active sessions retrieved successfully"
// @Failure      401 {object} responses.APIResponse "Unauthorized"
// @Failure      500 {object} responses.APIResponse "Internal server error"
// @Security     BearerAuth
// @Router       /api/v1/auth/sessions [get]
func (h *AuthHandler) GetActiveSessions(c *gin.Context) {
	userIDStr, exists := c.Get("user_id")
	if !exists {
		response := responses.ErrorResponse("Unauthorized: User ID not found in context", http.StatusUnauthorized)
		c.JSON(http.StatusUnauthorized, response)
		return
	}

	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		response := responses.ErrorResponse("Invalid user ID: "+err.Error(), http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	// Call auth service
	sessions, err := h.authService.GetActiveSessions(c.Request.Context(), userID)
	if err != nil {
		response := responses.ErrorResponse("Failed to get sessions: "+err.Error(), http.StatusInternalServerError)
		c.JSON(http.StatusInternalServerError, response)
		return
	}

	response := responses.SuccessResponse("Active sessions retrieved successfully", nil, sessions)
	c.JSON(http.StatusOK, response)
}

// RevokeSession godoc
// @Summary      Revoke Session
// @Description  Revoke a specific user session
// @Tags         Authentication
// @Accept       json
// @Produce      json
// @Param        sessionId path string true "Session ID"
// @Success      200 {object} responses.APIResponse{data=responses.LogoutResponse} "Session revoked successfully"
// @Failure      400 {object} responses.APIResponse "Invalid session ID"
// @Failure      500 {object} responses.APIResponse "Internal server error"
// @Security     BearerAuth
// @Router       /api/v1/auth/sessions/{sessionId} [delete]
func (h *AuthHandler) RevokeSession(c *gin.Context) {
	sessionIDParam := c.Param("sessionId")
	sessionID, err := uuid.Parse(sessionIDParam)
	if err != nil {
		response := responses.ErrorResponse("Invalid session ID: "+err.Error(), http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	// Call auth service
	logoutResponse, err := h.authService.RevokeSession(c.Request.Context(), sessionID)
	if err != nil {
		response := responses.ErrorResponse("Failed to revoke session: "+err.Error(), http.StatusInternalServerError)
		c.JSON(http.StatusInternalServerError, response)
		return
	}

	response := responses.SuccessResponse("Session revoked successfully", nil, logoutResponse)
	c.JSON(http.StatusOK, response)
}
