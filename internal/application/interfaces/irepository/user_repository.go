package irepository

import (
	"core-backend/internal/domain/model"

	"github.com/google/uuid"
)

type UserRepository interface {
	Create(user *model.User) error
	GetByID(id uuid.UUID) (*model.User, error)
	GetByUsername(username string) (*model.User, error)
	GetByEmail(email string) (*model.User, error)
	GetByUsernameOrEmail(identifier string) (*model.User, error)
	GetByFilters(limit, offset int, search, role string, isActive *bool) ([]*model.User, int, error)
	Update(user *model.User) error
	Delete(id uuid.UUID) error
	List(limit, offset int) ([]*model.User, error)
	Count() (int64, error)
	IsUsernameExists(username string) (bool, error)
	IsEmailExists(email string) (bool, error)
}
