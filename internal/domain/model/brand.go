package model

import (
	"core-backend/internal/domain/enum"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Brand struct {
	ID           uuid.UUID        `json:"id" gorm:"primaryKey"`
	Name         string           `json:"name" gorm:"not null;unique"`
	Description  string           `json:"description"`
	ContactEmail string           `json:"contact_email"`
	ContactPhone string           `json:"contact_phone"`
	Website      string           `json:"website"`
	Status       enum.BrandStatus `json:"status" gorm:"not null"`
	LogoURL      string           `json:"logo_url"`
}

func (b *Brand) BeforeCreate(tx *gorm.DB) error {
	if b.Status == "" {
		b.Status = enum.BrandStatusActive
	}
	if b.ID == uuid.Nil {
		b.ID = uuid.New()
	}

	return nil
}
