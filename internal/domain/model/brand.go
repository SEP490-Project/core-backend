package model

import (
	"core-backend/internal/domain/enum"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Brand struct {
	ID                      uuid.UUID        `json:"id" gorm:"type:uuid;column:id;primaryKey;default"`
	UserID                  *uuid.UUID       `json:"user_id" gorm:"type:uuid;column:user_id"`
	Name                    string           `json:"name" gorm:"type:varchar(255);column:name;not null"`
	Description             *string          `json:"description" gorm:"type:text;column:description"`
	ContactEmail            string           `json:"contact_email" gorm:"type:varchar(255);column:contact_email;not null"`
	ContactPhone            string           `json:"contact_phone" gorm:"type:varchar(20);column:contact_phone"`
	Address                 *string          `json:"address" gorm:"type:varchar(255);column:address"`
	Website                 *string          `json:"website" gorm:"type:varchar(255);column:website"`
	Status                  enum.BrandStatus `json:"status" gorm:"type:varchar(50);column:status;not null;check:status IN ('ACTIVE','INACTIVE')"`
	LogoURL                 *string          `json:"logo_url" gorm:"type:text;column:logo_url"`
	TaxNumber               *string          `json:"tax_number" gorm:"type:varchar(100);column:tax_number"`
	RepresentativeName      *string          `json:"representative_name" gorm:"type:varchar(255);column:representative_name"`
	RepresentativeRole      *string          `json:"representative_role" gorm:"type:varchar(100);column:representative_position"`
	RepresentativeEmail     *string          `json:"representative_email" gorm:"type:varchar(255);column:representative_email"`
	RepresentativePhone     *string          `json:"representative_phone" gorm:"type:varchar(25);column:representative_phone"`
	RepresentativeCitizenID *string          `json:"representative_citizen_id" gorm:"type:varchar(100);column:representative_citizen_id"`
	CreatedAt               time.Time        `json:"created_at" gorm:"column:created_at;autoCreateTime"`
	UpdatedAt               time.Time        `json:"updated_at" gorm:"column:updated_at;autoUpdateTime"`
	DeletedAt               gorm.DeletedAt   `json:"deleted_at" gorm:"column:deleted_at;index"`

	// Relationships
	Products []Product `json:"-" gorm:"foreignKey:BrandID"`
	User     *User     `json:"-" gorm:"foreignKey:UserID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
}

func (Brand) TableName() string { return "brands" }

func (b *Brand) BeforeCreate(tx *gorm.DB) error {
	if b.ID == uuid.Nil {
		b.ID = uuid.New()
	}
	if b.Status == "" {
		b.Status = enum.BrandStatusActive
	}

	return nil
}
