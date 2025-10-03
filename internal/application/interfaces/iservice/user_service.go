package iservice

import (
	"context"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/irepository"

	"github.com/google/uuid"
)

type UserService interface {
	// GetUserByID retrieves a user by their ID.
	GetUserByID(ctx context.Context, userID uuid.UUID) (*responses.UserResponse, error)
	// GetUsers retrieves a paginated list of users with optional filters.
	GetUsers(ctx context.Context, page, limit int, search, role string, isActive *bool) ([]*responses.UserResponse, int64, error)
	// UpdateUserStatus updates the active status of a user.
	UpdateUserStatus(ctx context.Context, userID uuid.UUID, isActive bool) error
	// UpdateUserRole updates the role of a user.
	UpdateUserRole(ctx context.Context, userID uuid.UUID, role string) error
	// DeleteUser soft deletes a user.
	DeleteUser(ctx context.Context, userID uuid.UUID) error
	// UpdateProfile updates the current user's profile.
	UpdateProfile(ctx context.Context, userID uuid.UUID, username, email string) (*responses.UserResponse, error)
	// ActivateBrandUser activates a user associated with a brand.
	ActivateBrandUser(ctx context.Context, userID uuid.UUID, unitOfWork irepository.UnitOfWork) error
}
