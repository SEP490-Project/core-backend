package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ProductCategory struct {
	ID               uuid.UUID      `json:"id" gorm:"type:uuid;column:id;primaryKey;default"`
	Name             string         `json:"name" gorm:"type:varchar(100);column:name;not null;unique"`
	Description      *string        `json:"description" gorm:"type:text;column:description"`
	ParentCategoryID *uuid.UUID     `json:"parent_category_id" gorm:"type:uuid;column:parent_category_id"`
	CreatedAt        time.Time      `json:"created_at" gorm:"column:created_at;autoCreateTime"`
	UpdatedAt        time.Time      `json:"updated_at" gorm:"column:updated_at;autoUpdateTime"`
	DeletedAt        gorm.DeletedAt `json:"deleted_at" gorm:"column:deleted_at;index" swaggerignore:"true"`

	// Relationships
	ParentCategory  *ProductCategory  `json:"-" gorm:"foreignKey:ParentCategoryID"`
	ChildCategories []ProductCategory `json:"-" gorm:"foreignKey:ParentCategoryID"`
	Products        []Product         `json:"-" gorm:"foreignKey:CategoryID"`
}

func (ProductCategory) TableName() string { return "product_categories" }

func (pc *ProductCategory) BeforeCreate(tx *gorm.DB) (err error) {
	if pc.ID == uuid.Nil {
		pc.ID = uuid.New()
	}

	return nil
}
