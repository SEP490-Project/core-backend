package iservice

import (
	"context"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/irepository"

	"github.com/google/uuid"
)

type UserService interface {
	// GetUserByID retrieves a user by their ID.
	GetUserByID(ctx context.Context, userID uuid.UUID) (*responses.UserResponse, error)
	// GetUsers retrieves a paginated list of users with optional filters.
	GetUsers(ctx context.Context, filterRequest *requests.UserFilterRequest) ([]*responses.UserListResponse, int64, error)
	// UpdateUserStatus updates the active status of a user.
	UpdateUserStatus(ctx context.Context, userID uuid.UUID, isActive bool) error
	// UpdateUserRole updates the role of a user.
	UpdateUserRole(ctx context.Context, userID uuid.UUID, role string) error
	// DeleteUser soft deletes a user.
	DeleteUser(ctx context.Context, userID uuid.UUID) error
	// UpdateProfile updates the current user's profile.
	UpdateProfile(ctx context.Context, userID uuid.UUID, updateRequset *requests.UpdateProfileRequest, uow irepository.UnitOfWork) (*responses.UserResponse, error)
	// ActivateBrandUser activates a user associated with a brand.
	ActivateBrandUser(ctx context.Context, userID uuid.UUID, unitOfWork irepository.UnitOfWork) error

	// GetPreferences retrieves notification preferences for a user
	// Returns default enabled preferences if none exist
	GetPreferences(ctx context.Context, userID uuid.UUID) (*responses.UserNotificationPreferenceResponse, error)

	// UpdatePreferences updates notification preferences for a user
	// Creates preferences if they don't exist
	UpdatePreferences(ctx context.Context, userID uuid.UUID, req *requests.UserNotificationPreferenceRequest) (*responses.UserNotificationPreferenceResponse, error)

	// GetOrCreateDefault gets existing preferences or creates default ones
	// Used internally by notification consumers
	GetOrCreateDefault(ctx context.Context, userID uuid.UUID) (emailEnabled bool, pushEnabled bool, err error)
}
