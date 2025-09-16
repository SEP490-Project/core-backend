package handler

import (
	"net/http"
	"strconv"
	"core-backend/internal/application/dto"
	"core-backend/internal/application/service"
	"core-backend/internal/presentation/dto/response"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

type UserHandler struct {
	userService *service.UserService
	validator   *validator.Validate
}

func NewUserHandler(userService *service.UserService) *UserHandler {
	return &UserHandler{
		userService: userService,
		validator:   validator.New(),
	}
}

// GetProfile godoc
// @Summary      Get User Profile
// @Description  Get current authenticated user's profile
// @Tags         Users
// @Accept       json
// @Produce      json
// @Success      200 {object} response.APIResponse{data=dto.UserResponse} "Profile retrieved successfully"
// @Failure      401 {object} response.APIResponse "Unauthorized"
// @Failure      404 {object} response.APIResponse "User not found"
// @Security     BearerAuth
// @Router       /api/v1/users/profile [get]
func (h *UserHandler) GetProfile(c *gin.Context) {
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

	user, err := h.userService.GetUserByID(userID)
	if err != nil {
		response := response.ErrorResponse("Failed to get user profile: "+err.Error(), http.StatusNotFound)
		c.JSON(http.StatusNotFound, response)
		return
	}

	response := response.SuccessResponse("Profile retrieved successfully", http.StatusOK, user)
	c.JSON(http.StatusOK, response)
}

// UpdateProfile godoc
// @Summary      Update User Profile
// @Description  Update current authenticated user's profile
// @Tags         Users
// @Accept       json
// @Produce      json
// @Param        request body dto.UpdateProfileRequest true "Profile update data"
// @Success      200 {object} response.APIResponse{data=dto.UserResponse} "Profile updated successfully"
// @Failure      400 {object} response.APIResponse "Invalid request"
// @Failure      401 {object} response.APIResponse "Unauthorized"
// @Failure      409 {object} response.APIResponse "Username or email already exists"
// @Security     BearerAuth
// @Router       /api/v1/users/profile [put]
func (h *UserHandler) UpdateProfile(c *gin.Context) {
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

	var request dto.UpdateProfileRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		response := response.ErrorResponse("Invalid request format: "+err.Error(), http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	if err := h.validator.Struct(&request); err != nil {
		response := response.ErrorResponse("Validation failed: "+err.Error(), http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	updatedUser, err := h.userService.UpdateProfile(userID, request.Username, request.Email)
	if err != nil {
		response := response.ErrorResponse("Failed to update profile: "+err.Error(), http.StatusConflict)
		c.JSON(http.StatusConflict, response)
		return
	}

	response := response.SuccessResponse("Profile updated successfully", http.StatusOK, updatedUser)
	c.JSON(http.StatusOK, response)
}

// GetUsers godoc
// @Summary      Get Users List
// @Description  Get paginated list of users (admin only)
// @Tags         Users
// @Accept       json
// @Produce      json
// @Param        page query int false "Page number" default(1)
// @Param        limit query int false "Items per page" default(10)
// @Param        search query string false "Search term for username or email"
// @Param        role query string false "Filter by user role"
// @Param        is_active query boolean false "Filter by active status"
// @Success      200 {object} response.APIResponse{data=dto.UserListResponse} "Users retrieved successfully"
// @Failure      401 {object} response.APIResponse "Unauthorized"
// @Failure      403 {object} response.APIResponse "Forbidden - Admin access required"
// @Failure      500 {object} response.APIResponse "Internal server error"
// @Security     BearerAuth
// @Router       /api/v1/users [get]
func (h *UserHandler) GetUsers(c *gin.Context) {
	// Parse pagination parameters
	page := 1
	limit := 10

	if pageParam := c.Query("page"); pageParam != "" {
		if p, err := strconv.Atoi(pageParam); err == nil && p > 0 {
			page = p
		}
	}

	if limitParam := c.Query("limit"); limitParam != "" {
		if l, err := strconv.Atoi(limitParam); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	// Parse search parameters
	search := c.Query("search")
	role := c.Query("role")
	isActiveParam := c.Query("is_active")

	var isActive *bool
	if isActiveParam != "" {
		if active, err := strconv.ParseBool(isActiveParam); err == nil {
			isActive = &active
		}
	}

	users, total, err := h.userService.GetUsers(page, limit, search, role, isActive)
	if err != nil {
		response := response.ErrorResponse("Failed to get users: "+err.Error(), http.StatusInternalServerError)
		c.JSON(http.StatusInternalServerError, response)
		return
	}

	// Calculate pagination info
	totalPages := (total + limit - 1) / limit
	hasNext := page < totalPages
	hasPrev := page > 1

	paginationData := dto.UserListResponse{
		Users:      users,
		Total:      total,
		Page:       page,
		Limit:      limit,
		TotalPages: totalPages,
		HasNext:    hasNext,
		HasPrev:    hasPrev,
	}

	response := response.SuccessResponse("Users retrieved successfully", http.StatusOK, paginationData)
	c.JSON(http.StatusOK, response)
}

// GetUserByID godoc
// @Summary      Get User by ID
// @Description  Get user details by ID (admin only)
// @Tags         Users
// @Accept       json
// @Produce      json
// @Param        id path string true "User ID"
// @Success      200 {object} response.APIResponse{data=dto.UserResponse} "User retrieved successfully"
// @Failure      400 {object} response.APIResponse "Invalid user ID"
// @Failure      401 {object} response.APIResponse "Unauthorized"
// @Failure      403 {object} response.APIResponse "Forbidden - Admin access required"
// @Failure      404 {object} response.APIResponse "User not found"
// @Security     BearerAuth
// @Router       /api/v1/users/{id} [get]
func (h *UserHandler) GetUserByID(c *gin.Context) {
	userIDParam := c.Param("id")
	userID, err := uuid.Parse(userIDParam)
	if err != nil {
		response := response.ErrorResponse("Invalid user ID: "+err.Error(), http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	user, err := h.userService.GetUserByID(userID)
	if err != nil {
		response := response.ErrorResponse("User not found: "+err.Error(), http.StatusNotFound)
		c.JSON(http.StatusNotFound, response)
		return
	}

	response := response.SuccessResponse("User retrieved successfully", http.StatusOK, user)
	c.JSON(http.StatusOK, response)
}

// UpdateUserStatus godoc
// @Summary      Update User Status
// @Description  Update user active status (admin only)
// @Tags         Users
// @Accept       json
// @Produce      json
// @Param        id path string true "User ID"
// @Param        request body dto.UpdateUserStatusRequest true "Status update data"
// @Success      200 {object} response.APIResponse "User status updated successfully"
// @Failure      400 {object} response.APIResponse "Invalid request"
// @Failure      401 {object} response.APIResponse "Unauthorized"
// @Failure      403 {object} response.APIResponse "Forbidden - Admin access required"
// @Failure      500 {object} response.APIResponse "Internal server error"
// @Security     BearerAuth
// @Router       /api/v1/users/{id}/status [put]
func (h *UserHandler) UpdateUserStatus(c *gin.Context) {
	userIDParam := c.Param("id")
	userID, err := uuid.Parse(userIDParam)
	if err != nil {
		response := response.ErrorResponse("Invalid user ID: "+err.Error(), http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	var request dto.UpdateUserStatusRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		response := response.ErrorResponse("Invalid request format: "+err.Error(), http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	if err := h.validator.Struct(&request); err != nil {
		response := response.ErrorResponse("Validation failed: "+err.Error(), http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	err = h.userService.UpdateUserStatus(userID, request.IsActive)
	if err != nil {
		response := response.ErrorResponse("Failed to update user status: "+err.Error(), http.StatusInternalServerError)
		c.JSON(http.StatusInternalServerError, response)
		return
	}

	message := "User activated successfully"
	if !request.IsActive {
		message = "User deactivated successfully"
	}

	response := response.SuccessResponse(message, http.StatusOK, nil)
	c.JSON(http.StatusOK, response)
}

// UpdateUserRole godoc
// @Summary      Update User Role
// @Description  Update user role (admin only)
// @Tags         Users
// @Accept       json
// @Produce      json
// @Param        id path string true "User ID"
// @Param        request body dto.UpdateUserRoleRequest true "Role update data"
// @Success      200 {object} response.APIResponse "User role updated successfully"
// @Failure      400 {object} response.APIResponse "Invalid request"
// @Failure      401 {object} response.APIResponse "Unauthorized"
// @Failure      403 {object} response.APIResponse "Forbidden - Admin access required"
// @Failure      500 {object} response.APIResponse "Internal server error"
// @Security     BearerAuth
// @Router       /api/v1/users/{id}/role [put]
func (h *UserHandler) UpdateUserRole(c *gin.Context) {
	userIDParam := c.Param("id")
	userID, err := uuid.Parse(userIDParam)
	if err != nil {
		response := response.ErrorResponse("Invalid user ID: "+err.Error(), http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	var request dto.UpdateUserRoleRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		response := response.ErrorResponse("Invalid request format: "+err.Error(), http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	if err := h.validator.Struct(&request); err != nil {
		response := response.ErrorResponse("Validation failed: "+err.Error(), http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	err = h.userService.UpdateUserRole(userID, request.Role)
	if err != nil {
		response := response.ErrorResponse("Failed to update user role: "+err.Error(), http.StatusInternalServerError)
		c.JSON(http.StatusInternalServerError, response)
		return
	}

	response := response.SuccessResponse("User role updated successfully", http.StatusOK, nil)
	c.JSON(http.StatusOK, response)
}

// DeleteUser godoc
// @Summary      Delete User
// @Description  Soft delete a user (admin only)
// @Tags         Users
// @Accept       json
// @Produce      json
// @Param        id path string true "User ID"
// @Success      200 {object} response.APIResponse "User deleted successfully"
// @Failure      400 {object} response.APIResponse "Invalid user ID"
// @Failure      401 {object} response.APIResponse "Unauthorized"
// @Failure      403 {object} response.APIResponse "Forbidden - Admin access required"
// @Failure      500 {object} response.APIResponse "Internal server error"
// @Security     BearerAuth
// @Router       /api/v1/users/{id} [delete]
func (h *UserHandler) DeleteUser(c *gin.Context) {
	userIDParam := c.Param("id")
	userID, err := uuid.Parse(userIDParam)
	if err != nil {
		response := response.ErrorResponse("Invalid user ID: "+err.Error(), http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	err = h.userService.DeleteUser(userID)
	if err != nil {
		response := response.ErrorResponse("Failed to delete user: "+err.Error(), http.StatusInternalServerError)
		c.JSON(http.StatusInternalServerError, response)
		return
	}

	response := response.SuccessResponse("User deleted successfully", http.StatusOK, nil)
	c.JSON(http.StatusOK, response)
}
