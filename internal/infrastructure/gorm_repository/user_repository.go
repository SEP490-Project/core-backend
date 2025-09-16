package gorm_repository

import (
	"core-backend/internal/domain/model"
	"core-backend/internal/domain/repository"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type userRepository struct {
	db *gorm.DB
}

// Count implements repository.UserRepository.
func (u *userRepository) Count() (int64, error) {
	var count int64
	err := u.db.Model(&model.User{}).Count(&count).Error
	return count, err
}

// Create implements repository.UserRepository.
func (u *userRepository) Create(user *model.User) error {
	return u.db.Create(user).Error
}

// Delete implements repository.UserRepository.
func (u *userRepository) Delete(id uuid.UUID) error {
	return u.db.Delete(&model.User{}, id).Error
}

// GetByEmail implements repository.UserRepository.
func (u *userRepository) GetByEmail(email string) (*model.User, error) {
	var user model.User
	err := u.db.Model(&model.User{}).Where("email = ?", email).First(&user).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &user, err
}

// GetByID implements repository.UserRepository.
func (u *userRepository) GetByID(id uuid.UUID) (*model.User, error) {
	var user model.User
	err := u.db.Model(&model.User{}).Where("id = ?", id).First(&user).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &user, err
}

// GetByUsername implements repository.UserRepository.
func (u *userRepository) GetByUsername(username string) (*model.User, error) {
	var user model.User
	err := u.db.Model(&model.User{}).Where("username = ?", username).First(&user).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &user, err
}

// GetByUsernameOrEmail implements repository.UserRepository.
func (u *userRepository) GetByUsernameOrEmail(identifier string) (*model.User, error) {
	var user model.User
	err := u.db.Model(&model.User{}).Where("username = ? OR email = ?", identifier, identifier).First(&user).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &user, err
}

// GetByFilters implements repository.UserRepository.
func (u *userRepository) GetByFilters(limit, offset int, search, role string, isActive *bool) ([]*model.User, int, error) {
	var users []*model.User
	var total int64
	
	query := u.db.Model(&model.User{})
	
	// Apply filters
	if search != "" {
		query = query.Where("username ILIKE ? OR email ILIKE ?", "%"+search+"%", "%"+search+"%")
	}
	
	if role != "" {
		query = query.Where("role = ?", role)
	}
	
	if isActive != nil {
		query = query.Where("is_active = ?", *isActive)
	}
	
	// Get total count
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	
	// Apply pagination and get results
	err := query.Limit(limit).Offset(offset).Find(&users).Error
	
	return users, int(total), err
}

// IsEmailExists implements repository.UserRepository.
func (u *userRepository) IsEmailExists(email string) (bool, error) {
	var count int64
	err := u.db.Model(&model.User{}).Where("email = ?", email).Count(&count).Error
	return count > 0, err
}

// IsUsernameExists implements repository.UserRepository.
func (u *userRepository) IsUsernameExists(username string) (bool, error) {
	var count int64
	err := u.db.Model(&model.User{}).Where("username = ?", username).Count(&count).Error
	return count > 0, err
}

// List implements repository.UserRepository.
func (u *userRepository) List(limit int, offset int) ([]*model.User, error) {
	var users []*model.User
	err := u.db.Model(&model.User{}).Limit(limit).Offset(offset).Find(&users).Error
	return users, err
}

// Update implements repository.UserRepository.
func (u *userRepository) Update(user *model.User) error {
	return u.db.Save(user).Error
}

func newUserRepository(db *gorm.DB) repository.UserRepository {
	return &userRepository{db: db}
}
