package gormrepository

import (
	"context"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/domain/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type TagRepository struct {
	*genericRepository[model.Tag]
}

func NewTagRepository(db *gorm.DB) irepository.TagRepository {
	return &TagRepository{genericRepository: &genericRepository[model.Tag]{db: db}}
}

// CreateIfNotExists creates tags if they do not already exist based on their names.
func (r *TagRepository) CreateIfNotExists(ctx context.Context, tags []model.Tag) ([]model.Tag, error) {
	var createdTags []model.Tag
	if len(tags) == 0 {
		return createdTags, nil
	}

	insertQuery := r.db.
		WithContext(ctx).
		Model(new(model.Tag)).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "name"}}, // key column
			DoNothing: true,
		}).
		Create(&tags)
	if insertQuery.Error != nil {
		return nil, insertQuery.Error
	}

	// Fetch all tags to return, including existing ones
	var tagNames []string
	for _, tag := range tags {
		tagNames = append(tagNames, tag.Name)
	}
	query := r.db.WithContext(ctx).Model(new(model.Tag)).Where("name IN (?)", tagNames)
	if err := query.Find(&createdTags).Error; err != nil {
		return nil, err
	}
	return createdTags, nil
}
