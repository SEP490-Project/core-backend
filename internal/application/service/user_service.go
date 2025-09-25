package service

import (
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/application/interfaces/iservice"
	"core-backend/internal/domain/enum"
	"errors"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type UserService struct {
	userRepository irepository.UserRepository
}

func NewUserService(userRepository irepository.UserRepository) iservice.UserService {
	return &UserService{
		userRepository: userRepository,
	}
}

// GetUserByID retrieves a user by ID
func (s *UserService) GetUserByID(userID uuid.UUID) (*responses.UserResponse, error) {
	zap.L().Debug("Retrieving user by ID", 
		zap.String("user_id", userID.String()))

	user, err := s.userRepository.GetByID(userID)
	if err != nil {
		zap.L().Error("Failed to retrieve user by ID", 
			zap.String("user_id", userID.String()),
			zap.Error(err))
		return nil, errors.New("user not found")
	}
	if user == nil {
		zap.L().Debug("User not found by ID", 
			zap.String("user_id", userID.String()))
		return nil, errors.New("user not found")
	}

	zap.L().Debug("User retrieved successfully", 
		zap.String("user_id", user.ID.String()),
		zap.String("username", user.Username))

	response := &responses.UserResponse{}
	return response.ToUserResponse(user), nil
}

// GetUsers retrieves users with pagination and filters
func (s *UserService) GetUsers(page, limit int, search, role string, isActive *bool) ([]*responses.UserResponse, int, error) {
	zap.L().Debug("Retrieving users with filters", 
		zap.Int("page", page),
		zap.Int("limit", limit),
		zap.String("search", search),
		zap.String("role", role),
		zap.Any("is_active", isActive))

	// Set default pagination values
	if page <= 0 {
		page = 1
	}
	if limit <= 0 || limit > 100 {
		limit = 10
	}

	offset := (page - 1) * limit

	zap.L().Debug("Pagination parameters set", 
		zap.Int("page", page),
		zap.Int("limit", limit),
		zap.Int("offset", offset))

	// Get users with filters
	users, total, err := s.userRepository.GetByFilters(limit, offset, search, role, isActive)
	if err != nil {
		zap.L().Error("Failed to retrieve users with filters", 
			zap.Int("limit", limit),
			zap.Int("offset", offset),
			zap.String("search", search),
			zap.String("role", role),
			zap.Error(err))
		return nil, 0, errors.New("failed to retrieve users")
	}

	// Map to response DTOs
	userResponses := make([]*responses.UserResponse, len(users))
	for i, user := range users {
		response := &responses.UserResponse{}
		userResponses[i] = response.ToUserResponse(user)
	}

	zap.L().Info("Users retrieved successfully", 
		zap.Int("total_users", total),
		zap.Int("returned_users", len(users)),
		zap.Int("page", page),
		zap.Int("limit", limit))

	return userResponses, total, nil
}

// UpdateUserStatus updates a user's active status
func (s *UserService) UpdateUserStatus(userID uuid.UUID, isActive bool) error {
	zap.L().Info("Updating user status", 
		zap.String("user_id", userID.String()),
		zap.Bool("is_active", isActive))

	user, err := s.userRepository.GetByID(userID)
	if err != nil || user == nil {
		zap.L().Error("Failed to find user for status update", 
			zap.String("user_id", userID.String()),
			zap.Error(err))
		return errors.New("user not found")
	}

	oldStatus := user.IsActive
	user.IsActive = isActive

	if err := s.userRepository.Update(user); err != nil {
		zap.L().Error("Failed to update user status", 
			zap.String("user_id", userID.String()),
			zap.Bool("old_status", oldStatus),
			zap.Bool("new_status", isActive),
			zap.Error(err))
		return errors.New("failed to update user status")
	}

	zap.L().Info("User status updated successfully", 
		zap.String("user_id", userID.String()),
		zap.String("username", user.Username),
		zap.Bool("old_status", oldStatus),
		zap.Bool("new_status", isActive))

	return nil
}

// UpdateUserRole updates a user's role
func (s *UserService) UpdateUserRole(userID uuid.UUID, role string) error {
	zap.L().Info("Updating user role", 
		zap.String("user_id", userID.String()),
		zap.String("new_role", role))

	user, err := s.userRepository.GetByID(userID)
	if err != nil || user == nil {
		zap.L().Error("Failed to find user for role update", 
			zap.String("user_id", userID.String()),
			zap.Error(err))
		return errors.New("user not found")
	}

	// Validate role
	userRole := enum.UserRole(role)
	if !userRole.IsValid() {
		zap.L().Debug("Invalid role provided for user role update", 
			zap.String("user_id", userID.String()),
			zap.String("invalid_role", role))
		return errors.New("invalid role")
	}

	oldRole := user.Role
	user.Role = userRole

	if err := s.userRepository.Update(user); err != nil {
		zap.L().Error("Failed to update user role", 
			zap.String("user_id", userID.String()),
			zap.String("old_role", string(oldRole)),
			zap.String("new_role", role),
			zap.Error(err))
		return errors.New("failed to update user role")
	}

	zap.L().Info("User role updated successfully", 
		zap.String("user_id", userID.String()),
		zap.String("username", user.Username),
		zap.String("old_role", string(oldRole)),
		zap.String("new_role", string(userRole)))

	return nil
}

// DeleteUser soft deletes a user
func (s *UserService) DeleteUser(userID uuid.UUID) error {
	zap.L().Info("Deleting user", 
		zap.String("user_id", userID.String()))

	user, err := s.userRepository.GetByID(userID)
	if err != nil || user == nil {
		zap.L().Error("Failed to find user for deletion", 
			zap.String("user_id", userID.String()),
			zap.Error(err))
		return errors.New("user not found")
	}

	zap.L().Debug("Found user for deletion", 
		zap.String("user_id", user.ID.String()),
		zap.String("username", user.Username))

	if err := s.userRepository.Delete(userID); err != nil {
		zap.L().Error("Failed to delete user", 
			zap.String("user_id", userID.String()),
			zap.String("username", user.Username),
			zap.Error(err))
		return errors.New("failed to delete user")
	}

	zap.L().Info("User deleted successfully", 
		zap.String("user_id", userID.String()),
		zap.String("username", user.Username))

	return nil
}

// UpdateProfile updates the current user's profile
func (s *UserService) UpdateProfile(userID uuid.UUID, username, email string) (*responses.UserResponse, error) {
	zap.L().Info("Updating user profile", 
		zap.String("user_id", userID.String()),
		zap.String("new_username", username),
		zap.String("new_email", email))

	user, err := s.userRepository.GetByID(userID)
	if err != nil || user == nil {
		zap.L().Error("Failed to find user for profile update", 
			zap.String("user_id", userID.String()),
			zap.Error(err))
		return nil, errors.New("user not found")
	}

	zap.L().Debug("Found user for profile update", 
		zap.String("user_id", user.ID.String()),
		zap.String("current_username", user.Username),
		zap.String("current_email", user.Email))

	// Update fields if provided
	if username != "" && username != user.Username {
		zap.L().Debug("Checking username availability", 
			zap.String("user_id", userID.String()),
			zap.String("new_username", username))

		// Check if username already exists
		if exists, err := s.userRepository.IsUsernameExists(username); err != nil {
			zap.L().Error("Failed to check username availability", 
				zap.String("user_id", userID.String()),
				zap.String("username", username),
				zap.Error(err))
			return nil, errors.New("failed to check username availability")
		} else if exists {
			zap.L().Debug("Username already exists", 
				zap.String("user_id", userID.String()),
				zap.String("username", username))
			return nil, errors.New("username already exists")
		}
		user.Username = username
		zap.L().Debug("Username updated", 
			zap.String("user_id", userID.String()),
			zap.String("new_username", username))
	}

	if email != "" && email != user.Email {
		zap.L().Debug("Checking email availability", 
			zap.String("user_id", userID.String()),
			zap.String("new_email", email))

		// Check if email already exists
		if exists, err := s.userRepository.IsEmailExists(email); err != nil {
			zap.L().Error("Failed to check email availability", 
				zap.String("user_id", userID.String()),
				zap.String("email", email),
				zap.Error(err))
			return nil, errors.New("failed to check email availability")
		} else if exists {
			zap.L().Debug("Email already exists", 
				zap.String("user_id", userID.String()),
				zap.String("email", email))
			return nil, errors.New("email already exists")
		}
		user.Email = email
		zap.L().Debug("Email updated", 
			zap.String("user_id", userID.String()),
			zap.String("new_email", email))
	}

	if err := s.userRepository.Update(user); err != nil {
		zap.L().Error("Failed to update user profile", 
			zap.String("user_id", userID.String()),
			zap.String("username", user.Username),
			zap.String("email", user.Email),
			zap.Error(err))
		return nil, errors.New("failed to update profile")
	}

	zap.L().Info("User profile updated successfully", 
		zap.String("user_id", userID.String()),
		zap.String("username", user.Username),
		zap.String("email", user.Email))

	response := &responses.UserResponse{}
	return response.ToUserResponse(user), nil
}
