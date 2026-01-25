package model

import (
	"time"

	"github.com/google/uuid"
)

type LimitedProduct struct {
	Id                    uuid.UUID `json:"id" gorm:"type:uuid;column:id;primaryKey;not null"`
	PremiereDate          time.Time `json:"premiere_date" gorm:"column:premiere_date;not null"`
	AvailabilityStartDate time.Time `json:"availability_start_date" gorm:"column:availability_start_date;not null"`
	AvailabilityEndDate   time.Time `json:"availability_end_date" gorm:"column:availability_end_date;not null"`
	AchievableQuantity    int       `json:"achievable_quantity" gorm:"column:achievable_quantity;not null"`
	// Relationships
	Product Product `json:"-" gorm:"foreignKey:Id;references:ID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`

	// Concept relation (nullable)
	ConceptID *uuid.UUID `json:"concept_id" gorm:"column:concept_id;type:uuid"`
	Concept   *Concept   `json:"concept" gorm:"foreignKey:ConceptID;references:ID"`
}

func (LimitedProduct) TableName() string { return "limited_products" }
