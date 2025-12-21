package gormrepository

import (
	"context"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/domain/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type userRepository struct {
	*genericRepository[model.User]
}

func NewUserRepository(db *gorm.DB) irepository.UserRepository {
	return &userRepository{
		genericRepository: &genericRepository[model.User]{db: db},
	}
}

// GetUserIDsByFilter implements [irepository.UserRepository].
func (u *userRepository) GetUserIDsByFilter(ctx context.Context, filter func(*gorm.DB) *gorm.DB) ([]uuid.UUID, error) {
	query := u.db.WithContext(ctx).Model(new(model.User))
	if filter != nil {
		query = filter(query)
	}
	userIDs := make([]uuid.UUID, 0)
	if err := query.Select("id").Pluck("id", &userIDs).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return []uuid.UUID{}, nil
		}
		return nil, err
	}

	return userIDs, nil
}
