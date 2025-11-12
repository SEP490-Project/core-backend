package handler

import (
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/application/interfaces/iservice"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserHandler struct {
	userService iservice.UserService
	unitOfWork  irepository.UnitOfWork
	validator   *validator.Validate
}

func NewUserHandler(userService iservice.UserService, unitOfWork irepository.UnitOfWork) *UserHandler {
	return &UserHandler{
		userService: userService,
		unitOfWork:  unitOfWork,
		validator:   validator.New(),
	}
}

// GetProfile godoc
//
//	@Summary		Get User Profile
//	@Description	Get current authenticated user's profile
//	@Tags			Users
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	responses.APIResponse{data=responses.UserResponse}	"Profile retrieved successfully"
//	@Failure		401	{object}	responses.APIResponse								"Unauthorized"
//	@Failure		404	{object}	responses.APIResponse								"User not found"
//	@Security		BearerAuth
//	@Router			/api/v1/users/profile [get]
func (h *UserHandler) GetProfile(c *gin.Context) {
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

	user, err := h.userService.GetUserByID(c.Request.Context(), userID)
	if err != nil {
		response := responses.ErrorResponse("Failed to get user profile: "+err.Error(), http.StatusNotFound)
		c.JSON(http.StatusNotFound, response)
		return
	}

	response := responses.SuccessResponse("Profile retrieved successfully", nil, user)
	c.JSON(http.StatusOK, response)
}

// UpdateProfile godoc
//
//	@Summary		Update User Profile
//	@Description	Update current authenticated user's profile
//	@Tags			Users
//	@Accept			json
//	@Produce		json
//	@Param			request	body		requests.UpdateProfileRequest						true	"Profile update data"
//	@Success		200		{object}	responses.APIResponse{data=responses.UserResponse}	"Profile updated successfully"
//	@Failure		400		{object}	responses.APIResponse								"Invalid request"
//	@Failure		401		{object}	responses.APIResponse								"Unauthorized"
//	@Failure		409		{object}	responses.APIResponse								"Username or email already exists"
//	@Security		BearerAuth
//	@Router			/api/v1/users/profile [put]
func (h *UserHandler) UpdateProfile(c *gin.Context) {
	userID, err := extractUserID(c)
	if err != nil {
		responses := responses.ErrorResponse("Unauthorized: "+err.Error(), http.StatusUnauthorized)
		c.JSON(http.StatusUnauthorized, responses)
		return
	}

	var request requests.UpdateProfileRequest
	if err = c.ShouldBindJSON(&request); err != nil {
		response := responses.ErrorResponse("Invalid request format: "+err.Error(), http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	if err = h.validator.Struct(&request); err != nil {
		response := responses.ErrorResponse("Validation failed: "+err.Error(), http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	uow := h.unitOfWork.Begin(c.Request.Context())

	var updatedUser *responses.UserResponse
	updatedUser, err = h.userService.UpdateProfile(c.Request.Context(), userID, &request, uow)
	if err != nil {
		uow.Rollback()
		response := responses.ErrorResponse("Failed to update profile: "+err.Error(), http.StatusConflict)
		c.JSON(http.StatusConflict, response)
		return
	}

	uow.Commit()

	response := responses.SuccessResponse("Profile updated successfully", nil, updatedUser)
	c.JSON(http.StatusOK, response)
}

// GetUsers godoc
//
//	@Summary		Get Users List
//	@Description	Get paginated list of users (admin and marketing only)
//	@Tags			Users
//	@Accept			json
//	@Produce		json
//	@Param			page				query		int									false	"Page number"				default(1)
//	@Param			limit				query		int									false	"Items per page"			default(10)
//	@Param			sort_by				query		string								false	"Field to sort by"			default(created_at)
//	@Param			sort_order			query		string								false	"Sort order (asc or desc)"	default(asc)
//	@Param			search				query		string								false	"Search term for username or email"
//	@Param			role				query		[]string							false	"Filter by user role" collectionFormat(multi)
//	@Param			is_active			query		boolean								false	"Filter by active status"
//	@Param			is_brand_account	query		boolean								false	"Filter by brand account status"
//	@Success		200					{object}	responses.UserPaginationResponse	"Users retrieved successfully"
//	@Failure		401					{object}	responses.APIResponse				"Unauthorized"
//	@Failure		403					{object}	responses.APIResponse				"Forbidden - Admin access required"
//	@Failure		500					{object}	responses.APIResponse				"Internal server error"
//	@Security		BearerAuth
//	@Router			/api/v1/users [get]
func (h *UserHandler) GetUsers(c *gin.Context) {
	var userFilterRequest *requests.UserFilterRequest
	if err := c.ShouldBindQuery(&userFilterRequest); err != nil {
		response := responses.ErrorResponse("Invalid query parameters: "+err.Error(), http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	users, total, err := h.userService.GetUsers(c.Request.Context(), userFilterRequest)
	if err != nil {
		var response *responses.APIResponse
		var statusCode int
		switch err.Error() {
		case gorm.ErrRecordNotFound.Error():
			response = responses.ErrorResponse("No users found", http.StatusNotFound)
			statusCode = http.StatusNotFound
		default:
			response = responses.ErrorResponse("Failed to get users: "+err.Error(), http.StatusInternalServerError)
			statusCode = http.StatusInternalServerError
		}
		c.JSON(statusCode, response)
		return
	}

	paginationData := responses.NewPaginationResponse(
		"Users retrieved successfully",
		http.StatusOK,
		users,
		responses.Pagination{
			Total: total,
			Page:  userFilterRequest.Page,
			Limit: userFilterRequest.Limit,
		},
	)
	c.JSON(http.StatusOK, paginationData)
}

// GetUserByID godoc
//
//	@Summary		Get User by ID
//	@Description	Get user details by ID (admin only)
//	@Tags			Users
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string												true	"User ID"
//	@Success		200	{object}	responses.APIResponse{data=responses.UserResponse}	"User retrieved successfully"
//	@Failure		400	{object}	responses.APIResponse								"Invalid user ID"
//	@Failure		401	{object}	responses.APIResponse								"Unauthorized"
//	@Failure		403	{object}	responses.APIResponse								"Forbidden - Admin access required"
//	@Failure		404	{object}	responses.APIResponse								"User not found"
//	@Security		BearerAuth
//	@Router			/api/v1/users/{id} [get]
func (h *UserHandler) GetUserByID(c *gin.Context) {
	userIDParam := c.Param("id")
	userID, err := uuid.Parse(userIDParam)
	if err != nil {
		response := responses.ErrorResponse("Invalid user ID: "+err.Error(), http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	user, err := h.userService.GetUserByID(c.Request.Context(), userID)
	if err != nil {
		response := responses.ErrorResponse("User not found: "+err.Error(), http.StatusNotFound)
		c.JSON(http.StatusNotFound, response)
		return
	}

	response := responses.SuccessResponse("User retrieved successfully", nil, user)
	c.JSON(http.StatusOK, response)
}

// UpdateUserStatus godoc
//
//	@Summary		Update User Status
//	@Description	Update user active status (admin only)
//	@Tags			Users
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string								true	"User ID"
//	@Param			request	body		requests.UpdateUserStatusRequest	true	"Status update data"
//	@Success		200		{object}	responses.APIResponse				"User status updated successfully"
//	@Failure		400		{object}	responses.APIResponse				"Invalid request"
//	@Failure		401		{object}	responses.APIResponse				"Unauthorized"
//	@Failure		403		{object}	responses.APIResponse				"Forbidden - Admin access required"
//	@Failure		500		{object}	responses.APIResponse				"Internal server error"
//	@Security		BearerAuth
//	@Router			/api/v1/users/{id}/status [put]
func (h *UserHandler) UpdateUserStatus(c *gin.Context) {
	userIDParam := c.Param("id")
	userID, err := uuid.Parse(userIDParam)
	if err != nil {
		response := responses.ErrorResponse("Invalid user ID: "+err.Error(), http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	var request requests.UpdateUserStatusRequest
	if err = c.ShouldBindJSON(&request); err != nil {
		response := responses.ErrorResponse("Invalid request format: "+err.Error(), http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	if err = h.validator.Struct(&request); err != nil {
		response := responses.ErrorResponse("Validation failed: "+err.Error(), http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	err = h.userService.UpdateUserStatus(c.Request.Context(), userID, *request.IsActive)
	if err != nil {
		response := responses.ErrorResponse("Failed to update user status: "+err.Error(), http.StatusInternalServerError)
		c.JSON(http.StatusInternalServerError, response)
		return
	}

	message := "User activated successfully"
	if !*request.IsActive {
		message = "User deactivated successfully"
	}

	response := responses.SuccessResponse(message, nil, nil)
	c.JSON(http.StatusOK, response)
}

// ActivateBrandUser godoc
//
//	@Summary		Activate Brand User
//	@Description	Activate a user associated with a brand (admin only)
//	@Tags			Users
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string					true	"User ID"
//	@Success		200	{object}	responses.APIResponse	"Brand user activated successfully"
//	@Failure		400	{object}	responses.APIResponse	"Invalid user ID"
//	@Failure		401	{object}	responses.APIResponse	"Unauthorized"
//	@Failure		403	{object}	responses.APIResponse	"Forbidden - Admin access required"
//	@Failure		500	{object}	responses.APIResponse	"Internal server error"
//	@Security		BearerAuth
//	@Router			/api/v1/users/{id}/activate-brand [patch]
func (h *UserHandler) ActivateBrandUser(c *gin.Context) {
	userIDParam := c.Param("id")
	userID, err := uuid.Parse(userIDParam)
	if err != nil {
		response := responses.ErrorResponse("Invalid user ID: "+err.Error(), http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}
	uow := h.unitOfWork.Begin(c.Request.Context())

	err = h.userService.ActivateBrandUser(c.Request.Context(), userID, uow)
	if err != nil {
		uow.Rollback()
		response := responses.ErrorResponse("Failed to activate brand user: "+err.Error(), http.StatusInternalServerError)
		c.JSON(http.StatusInternalServerError, response)
		return
	}

	uow.Commit()
	response := responses.SuccessResponse("Brand user activated successfully", nil, nil)
	c.JSON(http.StatusOK, response)
}

// UpdateUserRole godoc
//
//	@Summary		Update User Role
//	@Description	Update user role (admin only)
//	@Tags			Users
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string							true	"User ID"
//	@Param			request	body		requests.UpdateUserRoleRequest	true	"Role update data"
//	@Success		200		{object}	responses.APIResponse			"User role updated successfully"
//	@Failure		400		{object}	responses.APIResponse			"Invalid request"
//	@Failure		401		{object}	responses.APIResponse			"Unauthorized"
//	@Failure		403		{object}	responses.APIResponse			"Forbidden - Admin access required"
//	@Failure		500		{object}	responses.APIResponse			"Internal server error"
//	@Security		BearerAuth
//	@Router			/api/v1/users/{id}/role [put]
func (h *UserHandler) UpdateUserRole(c *gin.Context) {
	userIDParam := c.Param("id")
	userID, err := uuid.Parse(userIDParam)
	if err != nil {
		response := responses.ErrorResponse("Invalid user ID: "+err.Error(), http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	var request requests.UpdateUserRoleRequest
	if err = c.ShouldBindJSON(&request); err != nil {
		response := responses.ErrorResponse("Invalid request format: "+err.Error(), http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	if err = h.validator.Struct(&request); err != nil {
		response := responses.ErrorResponse("Validation failed: "+err.Error(), http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	err = h.userService.UpdateUserRole(c.Request.Context(), userID, request.Role)
	if err != nil {
		response := responses.ErrorResponse("Failed to update user role: "+err.Error(), http.StatusInternalServerError)
		c.JSON(http.StatusInternalServerError, response)
		return
	}

	response := responses.SuccessResponse("User role updated successfully", nil, nil)
	c.JSON(http.StatusOK, response)
}

// DeleteUser godoc
//
//	@Summary		Delete User
//	@Description	Soft delete a user (admin only)
//	@Tags			Users
//	@Accept			json
//	@Produce		json
//	@Param			id	path		string					true	"User ID"
//	@Success		200	{object}	responses.APIResponse	"User deleted successfully"
//	@Failure		400	{object}	responses.APIResponse	"Invalid user ID"
//	@Failure		401	{object}	responses.APIResponse	"Unauthorized"
//	@Failure		403	{object}	responses.APIResponse	"Forbidden - Admin access required"
//	@Failure		500	{object}	responses.APIResponse	"Internal server error"
//	@Security		BearerAuth
//	@Router			/api/v1/users/{id} [delete]
func (h *UserHandler) DeleteUser(c *gin.Context) {
	userIDParam := c.Param("id")
	userID, err := uuid.Parse(userIDParam)
	if err != nil {
		response := responses.ErrorResponse("Invalid user ID: "+err.Error(), http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	err = h.userService.DeleteUser(c.Request.Context(), userID)
	if err != nil {
		response := responses.ErrorResponse("Failed to delete user: "+err.Error(), http.StatusInternalServerError)
		c.JSON(http.StatusInternalServerError, response)
		return
	}

	response := responses.SuccessResponse("User deleted successfully", nil, nil)
	c.JSON(http.StatusOK, response)
}

// GetUserPreference godoc
//
//	@Summary		Get User notification preferences
//	@Description	Retrieves notification preference settings for the authenticated user
//	@Tags			Users
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	responses.APIResponse{data=responses.UserNotificationPreferenceResponse}
//	@Failure		401	{object}	responses.APIResponse
//	@Failure		500	{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/users/notification-preferences [get]
func (h *UserHandler) GetUserPreference(c *gin.Context) {
	// Get user ID from context
	userID, err := extractUserID(c)
	if err != nil {
		responses := responses.ErrorResponse("Unauthorized: "+err.Error(), http.StatusUnauthorized)
		c.JSON(http.StatusUnauthorized, responses)
	}

	// Get preferences
	var prefs *responses.UserNotificationPreferenceResponse
	prefs, err = h.userService.GetPreferences(c.Request.Context(), userID)
	if err != nil {
		response := responses.ErrorResponse("Failed to retrieve notification preferences", http.StatusInternalServerError)
		c.JSON(http.StatusInternalServerError, response)
		return
	}

	statusCode := http.StatusOK
	response := responses.SuccessResponse("Notification preferences retrieved successfully", &statusCode, prefs)
	c.JSON(statusCode, response)
}

// UpdateUserPreferences godoc
//
//	@Summary		Update notification preferences
//	@Description	Updates notification preference settings for the authenticated user
//	@Tags			Users
//	@Accept			json
//	@Produce		json
//	@Param			request	body		requests.UserNotificationPreferenceRequest	true	"Notification preferences"
//	@Success		200		{object}	responses.APIResponse{data=responses.UserNotificationPreferenceResponse}
//	@Failure		400		{object}	responses.APIResponse
//	@Failure		401		{object}	responses.APIResponse
//	@Failure		500		{object}	responses.APIResponse
//	@Security		BearerAuth
//	@Router			/api/v1/users/notification-preferences [put]
func (h *UserHandler) UpdateUserPreferences(c *gin.Context) {
	// Get user ID from context
	userID, err := extractUserID(c)
	if err != nil {
		responses := responses.ErrorResponse("Unauthorized: "+err.Error(), http.StatusUnauthorized)
		c.JSON(http.StatusUnauthorized, responses)
	}

	// Parse request body
	var req requests.UserNotificationPreferenceRequest
	if err = c.ShouldBindJSON(&req); err != nil {
		response := responses.ErrorResponse("Invalid request body", http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	// Validate request
	if err = h.validator.Struct(&req); err != nil {
		response := responses.ErrorResponse("Validation failed: "+err.Error(), http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	// At least one field must be provided
	if req.EmailEnabled == nil && req.PushEnabled == nil {
		response := responses.ErrorResponse("At least one preference field must be provided", http.StatusBadRequest)
		c.JSON(http.StatusBadRequest, response)
		return
	}

	// Update preferences
	var prefs *responses.UserNotificationPreferenceResponse
	prefs, err = h.userService.UpdatePreferences(c.Request.Context(), userID, &req)
	if err != nil {
		response := responses.ErrorResponse("Failed to update notification preferences", http.StatusInternalServerError)
		c.JSON(http.StatusInternalServerError, response)
		return
	}

	statusCode := http.StatusOK
	response := responses.SuccessResponse("Notification preferences updated successfully", &statusCode, prefs)
	c.JSON(http.StatusOK, response)
}
