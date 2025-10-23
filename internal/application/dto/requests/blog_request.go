package requests

import "github.com/google/uuid"

// BlogFieldsDTO for blog-specific fields when creating POST type content
type BlogFieldsDTO struct {
	AuthorID uuid.UUID `json:"author_id" validate:"required,uuid"`
	Tags     []string  `json:"tags,omitempty"`
	Excerpt  *string   `json:"excerpt,omitempty" validate:"omitempty,max=500"`
	ReadTime *int      `json:"read_time,omitempty" validate:"omitempty,gt=0"`
}

// UpdateBlogRequest DTO for updating blog-specific details
type UpdateBlogRequest struct {
	Tags     []string `json:"tags,omitempty"`
	Excerpt  *string  `json:"excerpt,omitempty" validate:"omitempty,max=500"`
	ReadTime *int     `json:"read_time,omitempty" validate:"omitempty,gt=0"`
}
