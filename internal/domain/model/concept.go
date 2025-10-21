package model

import (
	"time"

	"github.com/google/uuid"

	"core-backend/internal/domain/enum"
)

// Concept represents a marketing/product concept persisted in DB (table: concepts)
type Concept struct {
	ID             uuid.UUID          `json:"id" gorm:"type:uuid;column:id;primaryKey;default:gen_random_uuid()"`
	Name           string             `json:"name" gorm:"column:name;not null"`
	Description    *string            `json:"description" gorm:"column:description"`
	Status         enum.ConceptStatus `json:"status" gorm:"column:status;type:concept_status;not null;default:'DRAFT'"`
	StartDate      *time.Time         `json:"start_date" gorm:"column:start_date"`
	EndDate        *time.Time         `json:"end_date" gorm:"column:end_date"`
	CreatedAt      time.Time          `json:"created_at" gorm:"column:created_at;autoCreateTime"`
	UpdatedAt      time.Time          `json:"updated_at" gorm:"column:updated_at;autoUpdateTime"`
	BannerURL      *string            `json:"banner_url" gorm:"column:banner_url"`
	VideoThumbnail *string            `json:"video_thumbnail" gorm:"column:video_thumbnail"`
}

func (Concept) TableName() string { return "concepts" }
