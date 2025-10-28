package service

import (
	"context"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/application/interfaces/iservice"
	"core-backend/internal/application/service/helper"
	"core-backend/internal/domain/model"
	"errors"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type TagService struct {
	tagRepo irepository.GenericRepository[model.Tag]
}

// Create implements iservice.TagService.
func (t *TagService) Create(ctx context.Context, uow irepository.UnitOfWork, request *requests.CreateTagRequest) (*responses.TagResponse, error) {
	zap.L().Info("TagService - Create called")
	var (
		creatingTag, err = request.ToModel()
		tagRepo          = uow.Tags()
	)
	if err != nil {
		zap.L().Error("TagService - Create - request.ToModel error", zap.Error(err))
		return nil, err
	}

	if err = tagRepo.Add(ctx, creatingTag); err != nil {
		zap.L().Error("TagService - Create - tagRepo.Create error", zap.Error(err))
		return nil, err
	}

	var createdTag *model.Tag
	createdTag, err = tagRepo.GetByID(ctx, creatingTag.ID, nil)
	if err != nil {
		zap.L().Error("TagService - Create - tagRepo.GetByID error", zap.Error(err))
		return nil, err
	}

	zap.L().Info("TagService - Create - successfully created tag", zap.String("tagID", createdTag.ID.String()))
	return responses.TagResponse{}.ToResponse(createdTag), nil
}

// DeleteByID implements iservice.TagService.
func (t *TagService) DeleteByID(ctx context.Context, uow irepository.UnitOfWork, id uuid.UUID) error {
	zap.L().Info("TagService - DeleteByID called")
	tagRepo := uow.Tags()

	var tag *model.Tag
	tag, err := tagRepo.GetByID(ctx, id, nil)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			zap.L().Warn("TagService - DeleteByID - tag not found", zap.String("tagID", id.String()))
			return errors.New("tag not found")
		}
		zap.L().Error("TagService - DeleteByID - tagRepo.GetByID error", zap.Error(err))
		return err
	} else if tag == nil {
		zap.L().Warn("TagService - DeleteByID - tag not found", zap.String("tagID", id.String()))
		return err
	}

	if err := tagRepo.DeleteByID(ctx, tag.ID); err != nil {
		zap.L().Error("TagService - DeleteByID - tagRepo.DeleteByID error", zap.Error(err))
		return err
	}

	zap.L().Info("TagService - DeleteByID - successfully deleted tag", zap.String("tagID", id.String()))
	return nil
}

// GetByFilter implements iservice.TagService.
func (t *TagService) GetByFilter(ctx context.Context, filterRequest *requests.TagFilterRequest) ([]responses.TagResponse, int64, error) {
	zap.L().Info("TagService - GetByFilter called", zap.Any("request", filterRequest))

	filterQuery := func(db *gorm.DB) *gorm.DB {
		if filterRequest.Keyword != nil {
			keyword := "%" + *filterRequest.Keyword + "%"
			db = db.Where("name ILIKE ? or description ILIKE ?", keyword, keyword)
		}

		db = db.Order(helper.ConvertToSortString(filterRequest.PaginationRequest))

		return db
	}
	tags, totalCounts, err := t.tagRepo.GetAll(ctx, filterQuery, nil, filterRequest.Limit, filterRequest.Page)
	if err != nil {
		zap.L().Error("TagService - GetByFilter - tagRepo.GetAll error", zap.Error(err))
		return nil, 0, err
	}

	zap.L().Info("TagService - GetByFilter - successfully retrieved tags", zap.Int("count", len(tags)))
	return responses.TagResponse{}.ToListResponse(tags), totalCounts, nil
}

// GetByID implements iservice.TagService.
func (t *TagService) GetByID(ctx context.Context, id uuid.UUID) (*responses.TagResponse, error) {
	zap.L().Info("TagService - GetByID called", zap.String("tagID", id.String()))

	var tag *model.Tag
	tag, err := t.tagRepo.GetByID(ctx, id, nil)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			zap.L().Warn("TagService - GetByID - tag not found", zap.String("tagID", id.String()))
			return nil, errors.New("tag not found")
		}
		zap.L().Error("TagService - GetByID - tagRepo.GetByID error", zap.Error(err))
		return nil, err
	} else if tag == nil {
		zap.L().Warn("TagService - GetByID - tag not found", zap.String("tagID", id.String()))
		return nil, errors.New("tag not found")
	}

	zap.L().Info("TagService - GetByID - successfully retrieved tag", zap.String("tagID", id.String()))
	return responses.TagResponse{}.ToResponse(tag), nil
}

// GetByName implements iservice.TagService.
func (t *TagService) GetByName(ctx context.Context, uow irepository.UnitOfWork, name string, userID uuid.UUID) (*responses.TagResponse, error) {
	zap.L().Info("TagService - GetByName called", zap.String("tagName", name))
	var (
		isNotExist = false
		tagRepo    = uow.Tags()
	)

	filterQuery := func(db *gorm.DB) *gorm.DB {
		db = db.Where("name = ?", name)
		return db
	}
	tag, err := tagRepo.GetByCondition(ctx, filterQuery, nil)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			isNotExist = true
		} else {
			zap.L().Error("TagService - GetByName - tagRepo.GetByName error", zap.Error(err))
			return nil, err
		}
	} else if tag == nil {
		isNotExist = true
	}

	if isNotExist {
		zap.L().Info("TagService - GetByName - tag not found, creating new tag", zap.String("tagName", name))
		creatingTag := &model.Tag{
			ID:          uuid.New(),
			Name:        name,
			Description: &name,
			CreatedByID: &userID,
			UpdatedByID: &userID,
		}
		if err = tagRepo.Add(ctx, creatingTag); err != nil {
			zap.L().Error("TagService - GetByName - tagRepo.Add error", zap.Error(err))
			return nil, err
		}

		tag, err = tagRepo.GetByID(ctx, creatingTag.ID, nil)
		if err != nil {
			zap.L().Error("TagService - GetByName - tagRepo.GetByID error", zap.Error(err))
			return nil, err
		}
	}

	zap.L().Info("TagService - GetByName - successfully retrieved tag", zap.String("tagName", name))
	return responses.TagResponse{}.ToResponse(tag), nil
}

// UpdateByID implements iservice.TagService.
func (t *TagService) UpdateByID(ctx context.Context, uow irepository.UnitOfWork, request *requests.UpdateTagRequest) (*responses.TagResponse, error) {
	zap.L().Info("TagService - UpdateByID called", zap.String("tagID", *request.ID))
	var (
		tagRepo = uow.Tags()
		tag     *model.Tag
		err     error
	)
	tag, err = tagRepo.GetByID(ctx, *request.ID, nil)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			zap.L().Warn("TagService - UpdateByID - tag not found", zap.String("tagID", *request.ID))
			return nil, errors.New("tag not found")
		}
		zap.L().Error("TagService - UpdateByID - tagRepo.GetByID error", zap.Error(err))
		return nil, err
	} else if tag == nil {
		zap.L().Warn("TagService - UpdateByID - tag not found", zap.String("tagID", *request.ID))
		return nil, errors.New("tag not found")
	}

	var updatingTag *model.Tag
	updatingTag, err = request.ToExistingModel(tag)
	if err != nil {
		zap.L().Error("TagService - UpdateByID - request.ToExistingModel error", zap.Error(err))
		return nil, err
	}

	if err = tagRepo.Update(ctx, updatingTag); err != nil {
		zap.L().Error("TagService - UpdateByID - tagRepo.Update error", zap.Error(err))
		return nil, err
	}

	updatingTag, err = tagRepo.GetByID(ctx, updatingTag.ID, nil)
	if err != nil {
		zap.L().Error("TagService - UpdateByID - tagRepo.GetByID error", zap.Error(err))
		return nil, err
	}

	zap.L().Info("TagService - UpdateByID - successfully updated tag", zap.String("tagID", *request.ID))
	return responses.TagResponse{}.ToResponse(updatingTag), nil
}

func NewTagService(tagRepo irepository.GenericRepository[model.Tag]) iservice.TagService {
	return &TagService{tagRepo: tagRepo}
}
