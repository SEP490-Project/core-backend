package responses

import (
	"time"

	"github.com/google/uuid"
)

// BlogResponse DTO for blog-specific data
type BlogResponse struct {
	ContentID uuid.UUID  `json:"content_id"`
	AuthorID  uuid.UUID  `json:"author_id"`
	Author    *UserBrief `json:"author,omitempty"`
	Tags      []string   `json:"tags,omitempty"`
	Excerpt   *string    `json:"excerpt,omitempty"`
	ReadTime  *int       `json:"read_time,omitempty"`
	CreatedAt *time.Time `json:"created_at"`
	UpdatedAt *time.Time `json:"updated_at"`
}

// UserBrief for nested user info in blog response
type UserBrief struct {
	ID       uuid.UUID `json:"id"`
	Username string    `json:"username"`
	Email    string    `json:"email"`
}
