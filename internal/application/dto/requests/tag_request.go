package requests

import (
	"core-backend/internal/domain/model"

	"github.com/google/uuid"
)

// CreateTagRequest DTO for creating a new tag
type CreateTagRequest struct {
	Name        string  `json:"name" validate:"required,min=1,max=255" example:"Technology"`
	Description *string `json:"description,omitempty" validate:"omitempty,max=1000" example:"Posts related to the latest technology trends."`

	// Populated internally
	CreatedByID *string `json:"-" validate:"omitempty,uuid"`
}

// ToModel converts CreateTagRequest DTO to Tag model
func (r *CreateTagRequest) ToModel() (*model.Tag, error) {
	tagModel := &model.Tag{
		ID:          uuid.New(),
		Name:        r.Name,
		Description: r.Description,
	}

	if r.CreatedByID != nil {
		createdByID, err := uuid.Parse(*r.CreatedByID)
		if err != nil {
			return nil, err
		}
		tagModel.CreatedByID = &createdByID
		tagModel.UpdatedByID = &createdByID
	}
	return tagModel, nil
}

// UpdateTagRequest DTO for updating tag details
type UpdateTagRequest struct {
	Name        *string `json:"name,omitempty" validate:"omitempty,min=1,max=255" example:"Technology"`
	Description *string `json:"description,omitempty" validate:"omitempty,max=1000" example:"Posts related to the latest technology trends."`

	// Populated internally
	ID          *string `json:"-" validate:"omitempty,uuid"`
	UpdatedByID *string `json:"-" validate:"omitempty,uuid"`
}

// ToExistingModel updates an existing Tag model with fields from UpdateTagRequest DTO
func (r *UpdateTagRequest) ToExistingModel(existingTag *model.Tag) (*model.Tag, error) {
	if existingTag == nil {
		existingTag = &model.Tag{ID: uuid.New()}
	}
	if r.Name != nil {
		existingTag.Name = *r.Name
	}
	if r.Description != nil {
		existingTag.Description = r.Description
	}
	if r.UpdatedByID != nil {
		updatedByID, err := uuid.Parse(*r.UpdatedByID)
		if err == nil {
			return nil, err
		}
		existingTag.UpdatedByID = &updatedByID
	}
	return existingTag, nil
}

// TagFilterRequest DTO for filtering and paginating tags
type TagFilterRequest struct {
	PaginationRequest
	Keyword *string `json:"keyword,omitempty" validate:"omitempty,max=255"`
}
