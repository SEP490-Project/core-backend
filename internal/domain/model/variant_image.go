package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type VariantImage struct {
	ID        uuid.UUID      `json:"id" gorm:"type:uuid;column:id;primaryKey;default:gen_random_uuid()"`
	VariantID uuid.UUID      `json:"variant_id" gorm:"type:uuid;column:variant_id;not null"`
	ImageURL  string         `json:"image_url" gorm:"type:text;column:image_url;not null"`
	AltText   *string        `json:"alt_text" gorm:"type:varchar(255);column:alt_text"`
	IsPrimary bool           `json:"is_primary" gorm:"column:is_primary;not null;default:false"`
	CreatedAt time.Time      `json:"created_at" gorm:"column:created_at"`
	UpdatedAt time.Time      `json:"updated_at" gorm:"column:updated_at"`
	DeletedAt gorm.DeletedAt `json:"deleted_at" gorm:"column:deleted_at;index" swaggerignore:"true"`
}

func (VariantImage) TableName() string { return "variant_images" }

//func (vi *VariantImage) BeforeCreate(tx any) (err error) {
//	if vi.ID == uuid.Nil {
//		vi.ID = uuid.New()
//	}
//
//	return nil
//}
