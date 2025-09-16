package model

import (
	"core-backend/internal/domain/enum"

	"github.com/google/uuid"
)

// - Attributes: BrandID (SERIAL PK), Name (VARCHAR), Description (TEXT), ContactEmail (VARCHAR), ContactPhone (VARCHAR), Website (VARCHAR), Status (ENUM: Active/Inactive), LogoFileID (INT FK to File, optional).

type Brand struct {
	ID           uuid.UUID         `json:"id" gorm:"primaryKey"`
	Name         string            `json:"name" gorm:"not null;unique"`
	Description  string            `json:"description"`
	ContactEmail string            `json:"contact_email"`
	ContactPhone string            `json:"contact_phone"`
	Website      string            `json:"website"`
	Status       enum.BrandStatus `json:"status" gorm:"type:enum('ACTIVE','INACTIVE');default:'ACTIVE'"`
	LogoURL      string            `json:"logo_url"`
}
