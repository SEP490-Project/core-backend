package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ProductReview represents a customer review for a product
type ProductReview struct {
	ID          uuid.UUID  `json:"id" gorm:"type:uuid;column:id;primaryKey;default"`
	ProductID   uuid.UUID  `json:"product_id" gorm:"column:product_id;not null"`
	UserID      *uuid.UUID `json:"user_id,omitempty" gorm:"column:user_id;type:uuid"`
	OrderItemID *uuid.UUID `json:"order_item_id,omitempty" gorm:"column:order_item_id;type:uuid"`
	PreOrderID  *uuid.UUID `json:"pre_order_id,omitempty" gorm:"column:pre_order_id;type:uuid"`
	RatingStars int        `json:"rating_stars" gorm:"column:rating_stars;not null;check:rating_stars >= 1 AND rating_stars <= 5"`
	Comment     *string    `json:"comment,omitempty" gorm:"column:comment;type:text"`
	AssetsURL   *string    `json:"assets_url,omitempty" gorm:"column:assets_url;type:text"`

	CreatedAt time.Time      `json:"created_at" gorm:"column:created_at;autoCreateTime"`
	UpdatedAt time.Time      `json:"updated_at" gorm:"column:updated_at;autoUpdateTime"`
	DeletedAt gorm.DeletedAt `json:"deleted_at" gorm:"column:deleted_at;index" swaggerignore:"true"`

	// Relations
	Product   Product    `json:"product" gorm:"foreignKey:ProductID"`
	User      User       `json:"user,omitempty" gorm:"foreignKey:UserID" swaggerignore:"true"`
	OrderItem *OrderItem `json:"order_item" gorm:"foreignKey:OrderItemID" swaggerignore:"true"`
	PreOrder  *PreOrder  `json:"preorder" gorm:"foreignKey:PreOrderID" swaggerignore:"true"`
}

func (ProductReview) TableName() string { return "product_reviews" }

func (r *ProductReview) BeforeCreate(tx *gorm.DB) (err error) {
	_ = tx
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	return nil
}
