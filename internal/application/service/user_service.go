package service

import (
	"context"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/application/interfaces/iservice"
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
	"core-backend/pkg/utils"
	"errors"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type UserService struct {
	// userRepository irepository.UserRepository
	userRepository irepository.GenericRepository[model.User]
}

// ActivateBrandUser implements iservice.UserService.
func (s *UserService) ActivateBrandUser(ctx context.Context, userID uuid.UUID, unitOfWork irepository.UnitOfWork) error {
	zap.L().Info("Activating brand user",
		zap.String("user_id", userID.String()),
	)

	userRepo := unitOfWork.Users()
	brandRepo := unitOfWork.Brands()

	filters := func(db *gorm.DB) *gorm.DB {
		return db.Joins("inner join brands on brands.user_id = users.id").
			Where("users.id = ? AND users.role = ? AND users.is_active = ? AND brands.status = ?",
				userID, enum.UserRoleBrandPartner, false, enum.BrandStatusInactive,
			)
	}
	brandUsers, err := userRepo.GetByCondition(ctx, filters, []string{"Brand"})
	if err != nil || brandUsers == nil {
		zap.L().Error("Failed to find brand user for activation",
			zap.String("user_id", userID.String()),
			zap.Error(err))
		return errors.New("brand user not found or already active")
	}

	var generatedPassword string
	generatedPassword, err = utils.GenerateRandomPassword(16)
	if err != nil {
		zap.L().Error("Failed to generate password for brand user activation",
			zap.String("user_id", userID.String()),
			zap.Error(err))
		return errors.New("failed to generate password for brand user activation")
	}
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(generatedPassword), bcrypt.DefaultCost)
	if err != nil {
		zap.L().Error("Failed to hash password for brand user activation",
			zap.String("user_id", userID.String()),
			zap.Error(err))
		return errors.New("failed to hash password for brand user activation")
	}
	brandUsers.PasswordHash = string(hashedPassword)
	brandUsers.IsActive = true
	brandUsers.Brand.Status = enum.BrandStatusActive

	if err := userRepo.Update(ctx, brandUsers); err != nil {
		zap.L().Error("Failed to activate brand user",
			zap.String("user_id", userID.String()),
			zap.Error(err))
		return errors.New("failed to activate brand user")
	}
	if err := brandRepo.Update(ctx, brandUsers.Brand); err != nil {
		zap.L().Error("Failed to update brand status during user activation",
			zap.String("user_id", userID.String()),
			zap.String("brand_id", brandUsers.Brand.ID.String()),
			zap.Error(err))
		return errors.New("failed to update brand status during user activation")
	}

	zap.L().Info("Brand user activated successfully",
		zap.String("user_id", userID.String()))
	zap.L().Debug("Generated password for brand user activation",
		zap.String("user_id", userID.String()),
		zap.String("username", brandUsers.Username),
		zap.String("password", generatedPassword))
	return nil
}

// GetUsers retrieves users with pagination and filters
func (s *UserService) GetUsers(ctx context.Context, page, limit int, search, role string, isActive *bool) ([]*responses.UserResponse, int64, error) {
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
	filters := func(db *gorm.DB) *gorm.DB {
		if search != "" {
			db = db.Where("username ILIKE ? OR email ILIKE ?", "%"+search+"%", "%"+search+"%")
		}
		if role != "" {
			db = db.Where("role = ?", role)
		}
		if isActive != nil {
			db = db.Where("is_active = ?", *isActive)
		}
		return db
	}
	users, total, err := s.userRepository.GetAll(ctx, filters, nil, limit, page)
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
		userResponses[i] = response.ToUserResponse(&user)
	}

	zap.L().Info("Users retrieved successfully",
		zap.Int64("total_users", total),
		zap.Int("returned_users", len(users)),
		zap.Int("page", page),
		zap.Int("limit", limit))

	return userResponses, total, nil
}

// UpdateUserStatus updates a user's active status
func (s *UserService) UpdateUserStatus(ctx context.Context, userID uuid.UUID, isActive bool) error {
	zap.L().Info("Updating user status",
		zap.String("user_id", userID.String()),
		zap.Bool("is_active", isActive))

	// user, err := s.userRepository.GetByID(userID)
	user, err := s.userRepository.GetByID(ctx, userID, nil)
	if err != nil || user == nil {
		zap.L().Error("Failed to find user for status update",
			zap.String("user_id", userID.String()),
			zap.Error(err))
		return errors.New("user not found")
	}

	oldStatus := user.IsActive
	user.IsActive = isActive

	if err := s.userRepository.Update(ctx, user); err != nil {
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
func (s *UserService) UpdateUserRole(ctx context.Context, userID uuid.UUID, role string) error {
	zap.L().Info("Updating user role",
		zap.String("user_id", userID.String()),
		zap.String("new_role", role))

	user, err := s.userRepository.GetByID(ctx, userID, nil)
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

	if err := s.userRepository.Update(ctx, user); err != nil {
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
func (s *UserService) DeleteUser(ctx context.Context, userID uuid.UUID) error {
	zap.L().Info("Deleting user",
		zap.String("user_id", userID.String()))

	// user, err := s.userRepository.GetByID(userID)
	userExisted, err := s.userRepository.Exists(ctx, func(db *gorm.DB) *gorm.DB { return db.Where("id = ?", userID) })
	if err != nil || userExisted {
		zap.L().Error("Failed to find user for deletion",
			zap.String("user_id", userID.String()),
			zap.Error(err))
		return errors.New("user not found")
	}

	zap.L().Debug("Found user for deletion",
		zap.String("user_id", userID.String()),
	)

	if err := s.userRepository.DeleteByID(ctx, userID); err != nil {
		zap.L().Error("Failed to delete user",
			zap.String("user_id", userID.String()),
			zap.Error(err))
		return errors.New("failed to delete user")
	}

	zap.L().Info("User deleted successfully",
		zap.String("user_id", userID.String()),
	)

	return nil
}

// GetUserByID implements iservice.UserService.
func (s *UserService) GetUserByID(ctx context.Context, userID uuid.UUID) (*responses.UserResponse, error) {
	zap.L().Debug("Retrieving user by ID",
		zap.String("user_id", userID.String()))

	// user, err := s.userRepository.GetByID(userID)
	user, err := s.userRepository.GetByID(ctx, userID, []string{"ShippingAddress", "Sessions"})
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

// UpdateProfile updates the current user's profile
func (s *UserService) UpdateProfile(
	ctx context.Context,
	userID uuid.UUID,
	updateRequest *requests.UpdateProfileRequest,
	uow irepository.UnitOfWork,
) (*responses.UserResponse, error) {
	zap.L().Info("Updating user profile",
		zap.String("user_id", userID.String()),
		zap.Any("request", *updateRequest))

	userRepo := uow.Users()
	addressRepo := uow.ShippingAddresses()

	user, err := userRepo.GetByID(ctx, userID, []string{"ShippingAddress"})
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

	updatingUserModel, modifyingAddresses := updateRequest.ToExistingProfile(user)

	// Update fields if provided
	if updateRequest.Username != nil && *updateRequest.Username != user.Username {
		zap.L().Debug("Checking username availability",
			zap.String("user_id", userID.String()),
			zap.String("new_username", *updateRequest.FullName))

		// Check if username already exists
		filters := func(db *gorm.DB) *gorm.DB {
			return db.Where("username = ?", *updateRequest.FullName)
		}
		if exists, err := userRepo.Exists(ctx, filters); err != nil {
			zap.L().Error("Failed to check username availability",
				zap.String("user_id", userID.String()),
				zap.String("username", *updateRequest.FullName),
				zap.Error(err))
			return nil, errors.New("failed to check username availability")
		} else if exists {
			zap.L().Debug("Username already exists",
				zap.String("user_id", userID.String()),
				zap.String("username", *updateRequest.FullName))
			return nil, errors.New("username already exists")
		}
		updatingUserModel.Username = *updateRequest.FullName
	}

	if err := userRepo.Update(ctx, user); err != nil {
		zap.L().Error("Failed to update user profile",
			zap.String("user_id", userID.String()),
			zap.Error(err))
		return nil, errors.New("failed to update profile")
	}

	err = addressRepo.DB().Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{"type", "full_name", "phone_number", "email", "street", "address_line_2", "city", "state", "postal_code", "country", "company", "is_default"}),
		UpdateAll: false,
		DoNothing: false,
	}).CreateInBatches(modifyingAddresses, len(modifyingAddresses)).Error
	if err != nil {
		zap.L().Error("Failed to update shipping addresses during profile update",
			zap.String("user_id", userID.String()),
			zap.Int("original_addresses_count", len(user.ShippingAddress)),
			zap.Int("modified_addresses_count", len(modifyingAddresses)),
			zap.Error(err))
		return nil, errors.New("failed to update shipping addresses during profile update")
	}

	zap.L().Info("User profile updated successfully with addresses",
		zap.String("user_id", userID.String()),
		zap.Int("modified_addresses_count", len(modifyingAddresses)))

	response := &responses.UserResponse{}
	return response.ToUserResponse(user), nil
}

func NewUserService(userRepository irepository.GenericRepository[model.User]) iservice.UserService {
	return &UserService{
		userRepository: userRepository,
	}
}
