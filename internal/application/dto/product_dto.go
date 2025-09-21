package dto

type ProductResponse struct {
	ID          int     `json:"id"`
	BrandID     int     `json:"brand_id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Price       float64 `json:"price"`
	//CurrentStock int              `gorm:"column:current_stock;not null"`
	Type string `json:"type"`

	//Relationship
	Variants []*ProductVariantResponse `json:"variants,omitempty"`
}
