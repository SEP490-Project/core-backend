package model

import (
	"core-backend/internal/domain/enum"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Product struct {
	ID          uuid.UUID  `json:"id" gorm:"column:id;primaryKey;autoIncrement"`
	BrandID     *uuid.UUID `json:"brand_id" gorm:"column:brand_id;"`
	CategoryID  uuid.UUID  `json:"category_id" gorm:"column:category_id;not null"`
	TaskID      *uuid.UUID `json:"task_id" gorm:"column:task_id"`
	Name        string     `json:"name" gorm:"column:name;not null"`
	Description *string    `json:"description" gorm:"column:description"`
	//CurrentStock *int               `json:"current_stock" gorm:"column:current_stock"`
	Type        enum.ProductType   `json:"type" gorm:"column:type;not null;check:type in ('STANDARD', 'LIMITED')"`
	CreatedAt   time.Time          `json:"created_at" gorm:"column:created_at"`
	UpdatedAt   time.Time          `json:"updated_at" gorm:"column:updated_at"`
	DeletedAt   gorm.DeletedAt     `json:"deleted_at" gorm:"column:deleted_at;index" swaggerignore:"true"`
	Status      enum.ProductStatus `json:"status" gorm:"column:status;not null;check:status in ('DRAFT','SUBMITTED','REVISION','APPROVED','ACTIVED','INACTIVED')"`
	CreatedByID uuid.UUID          `json:"created_by" gorm:"column:created_by;not null"`
	UpdatedByID *uuid.UUID         `json:"updated_by" gorm:"column:updated_by"`
	CreatedBy   *User              `json:"created_by_user" gorm:"foreignKey:CreatedByID" swaggerignore:"true"`
	UpdatedBy   *User              `json:"updated_by_user" gorm:"foreignKey:UpdatedByID" swaggerignore:"true"`
	IsActive    bool               `json:"is_active" gorm:"column:is_active;default:false;not null"`
	// Relationships
	Brand    *Brand           `json:"brand" gorm:"foreignKey:BrandID" swaggerignore:"true"`
	Category *ProductCategory `json:"category" gorm:"foreignKey:CategoryID"`
	Variants []ProductVariant `json:"product_variants" gorm:"foreignKey:ProductID"`
	Task     *Task            `json:"task" gorm:"foreignKey:TaskID" swaggerignore:"true"`
	Limited  *LimitedProduct  `json:"limited" gorm:"foreignKey:Id;references:ID"`
}

func (Product) TableName() string { return "products" }

func (p *Product) BeforeCreate(tx *gorm.DB) (err error) {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}

	return nil
}
