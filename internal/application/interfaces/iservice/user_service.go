package iservice

import (
	"core-backend/internal/application/dto/responses"

	"github.com/google/uuid"
)

type UserService interface {
	GetUserByID(userID uuid.UUID) (*responses.UserResponse, error)
	GetUsers(page, limit int, search, role string, isActive *bool) ([]*responses.UserResponse, int, error)
	UpdateUserStatus(userID uuid.UUID, isActive bool) error
	UpdateUserRole(userID uuid.UUID, role string) error
	DeleteUser(userID uuid.UUID) error
	UpdateProfile(userID uuid.UUID, username, email string) (*responses.UserResponse, error)
}
