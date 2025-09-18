package service

import (
	"errors"
	"core-backend/internal/application/dto"
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
	"core-backend/internal/domain/repository"

	"github.com/google/uuid"
)

type UserService struct {
	userRepository repository.UserRepository
}

func NewUserService(userRepository repository.UserRepository) *UserService {
	return &UserService{
		userRepository: userRepository,
	}
}

// GetUserByID retrieves a user by ID
func (s *UserService) GetUserByID(userID uuid.UUID) (*dto.UserResponse, error) {
	user, err := s.userRepository.GetByID(userID)
	if err != nil {
		return nil, errors.New("user not found")
	}
	if user == nil {
		return nil, errors.New("user not found")
	}

	return s.mapUserToResponse(user), nil
}

// GetUsers retrieves users with pagination and filters
func (s *UserService) GetUsers(page, limit int, search, role string, isActive *bool) ([]*dto.UserResponse, int, error) {
	// Set default pagination values
	if page <= 0 {
		page = 1
	}
	if limit <= 0 || limit > 100 {
		limit = 10
	}

	offset := (page - 1) * limit

	// Get users with filters
	users, total, err := s.userRepository.GetByFilters(limit, offset, search, role, isActive)
	if err != nil {
		return nil, 0, errors.New("failed to retrieve users")
	}

	// Map to response DTOs
	userResponses := make([]*dto.UserResponse, len(users))
	for i, user := range users {
		userResponses[i] = s.mapUserToResponse(user)
	}

	return userResponses, total, nil
}

// UpdateUserStatus updates a user's active status
func (s *UserService) UpdateUserStatus(userID uuid.UUID, isActive bool) error {
	user, err := s.userRepository.GetByID(userID)
	if err != nil || user == nil {
		return errors.New("user not found")
	}

	user.IsActive = isActive
	if err := s.userRepository.Update(user); err != nil {
		return errors.New("failed to update user status")
	}

	return nil
}

// UpdateUserRole updates a user's role
func (s *UserService) UpdateUserRole(userID uuid.UUID, role string) error {
	user, err := s.userRepository.GetByID(userID)
	if err != nil || user == nil {
		return errors.New("user not found")
	}

	// Validate role
	userRole := enum.UserRole(role)
	if !s.isValidRole(userRole) {
		return errors.New("invalid role")
	}

	user.Role = userRole
	if err := s.userRepository.Update(user); err != nil {
		return errors.New("failed to update user role")
	}

	return nil
}

// DeleteUser soft deletes a user
func (s *UserService) DeleteUser(userID uuid.UUID) error {
	user, err := s.userRepository.GetByID(userID)
	if err != nil || user == nil {
		return errors.New("user not found")
	}

	if err := s.userRepository.Delete(userID); err != nil {
		return errors.New("failed to delete user")
	}

	return nil
}

// UpdateProfile updates the current user's profile
func (s *UserService) UpdateProfile(userID uuid.UUID, username, email string) (*dto.UserResponse, error) {
	user, err := s.userRepository.GetByID(userID)
	if err != nil || user == nil {
		return nil, errors.New("user not found")
	}

	// Update fields if provided
	if username != "" && username != user.Username {
		// Check if username already exists
		if exists, err := s.userRepository.IsUsernameExists(username); err != nil {
			return nil, errors.New("failed to check username availability")
		} else if exists {
			return nil, errors.New("username already exists")
		}
		user.Username = username
	}

	if email != "" && email != user.Email {
		// Check if email already exists
		if exists, err := s.userRepository.IsEmailExists(email); err != nil {
			return nil, errors.New("failed to check email availability")
		} else if exists {
			return nil, errors.New("email already exists")
		}
		user.Email = email
	}

	if err := s.userRepository.Update(user); err != nil {
		return nil, errors.New("failed to update profile")
	}

	return s.mapUserToResponse(user), nil
}

// Helper methods
func (s *UserService) mapUserToResponse(user *model.User) *dto.UserResponse {
	return &dto.UserResponse{
		ID:        user.ID,
		Username:  user.Username,
		Email:     user.Email,
		Role:      string(user.Role),
		IsActive:  user.IsActive,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
		LastLogin: user.LastLogin,
	}
}

func (s *UserService) isValidRole(role enum.UserRole) bool {
	switch role {
	case enum.RoleAdmin, enum.RoleCustomer, enum.RoleBrandPartner, enum.RoleContentStaff, enum.RoleMarketingStaff, enum.RoleSalesStaff:
		return true
	default:
		return false
	}
}
