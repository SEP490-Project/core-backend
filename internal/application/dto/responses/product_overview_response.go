package responses

import (
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"

	"github.com/google/uuid"
)

// Regenerated ProductOverviewResponse provides a flattened, list-optimized snapshot
// of a product including minimal brand/category context and lightweight variants.
// Suitable for pagination & task-based listing.
type ProductOverviewResponse struct {
	ID                 uuid.UUID          `json:"id"`
	TaskID             *uuid.UUID         `json:"task_id,omitempty"`
	Name               string             `json:"name"`
	Description        *string            `json:"description,omitempty"`
	Price              float64            `json:"price"`
	Type               enum.ProductType   `json:"type"`
	Status             enum.ProductStatus `json:"status"`
	BrandID            uuid.UUID          `json:"brand_id"`
	BrandName          string             `json:"brand_name,omitempty"`
	BrandLogoURL       *string            `json:"brand_logo_url,omitempty"`
	CategoryID         uuid.UUID          `json:"category_id"`
	CategoryName       string             `json:"category_name,omitempty"`
	ParentCategoryID   *uuid.UUID         `json:"parent_category_id,omitempty"`
	ParentCategoryName *string            `json:"parent_category_name,omitempty"`
	VariantCount       int                `json:"variant_count"`
	Variants           []VariantMini      `json:"variants,omitempty"`
}

// VariantMini lightweight projection of a variant.
type VariantMini struct {
	ID           uuid.UUID         `json:"id"`
	Price        float64           `json:"price"`
	IsDefault    bool              `json:"is_default"`
	CurrentStock *int              `json:"current_stock"`
	Capacity     float64           `json:"capacity"`
	CapacityUnit enum.CapacityUnit `json:"capacity_unit"`
	Weight       int               `json:"weight"` // in grams
	Height       int               `json:"height"` // in centimeters
	Length       int               `json:"length"` // in centimeters
	Width        int               `json:"width"`  //

}

// ToOverview maps a Product domain model to ProductOverviewResponse.
func ToOverview(p *model.Product) *ProductOverviewResponse {
	if p == nil {
		return nil
	}
	ov := &ProductOverviewResponse{
		ID:          p.ID,
		TaskID:      p.TaskID,
		Name:        p.Name,
		Description: p.Description,
		Type:        p.Type,
		Status:      p.Status,
		BrandID:     p.BrandID,
		CategoryID:  p.CategoryID,
	}
	// Brand
	if p.Brand != nil {
		ov.BrandName = p.Brand.Name
		ov.BrandLogoURL = p.Brand.LogoURL
	}
	// Category + parent
	if p.Category != nil {
		ov.CategoryName = p.Category.Name
		if p.Category.ParentCategory != nil {
			parent := p.Category.ParentCategory
			ov.ParentCategoryID = &parent.ID
			ov.ParentCategoryName = &parent.Name
		}
	}
	// Variants
	if len(p.Variants) > 0 {
		ov.VariantCount = len(p.Variants)
		ov.Variants = make([]VariantMini, 0, len(p.Variants))
		for _, v := range p.Variants {
			ov.Variants = append(ov.Variants, VariantMini{
				ID:           v.ID,
				Price:        v.Price,
				IsDefault:    v.IsDefault,
				CurrentStock: v.CurrentStock,
				Capacity:     v.Capacity,
				CapacityUnit: v.CapacityUnit,
				Weight:       v.Weight,
				Height:       v.Height,
				Length:       v.Length,
				Width:        v.Width,
			})
		}
	}
	return ov
}

// ToOverviewList converts slice of *model.Product to slice of *ProductOverviewResponse.
func ToOverviewList(products []*model.Product) []*ProductOverviewResponse {
	res := make([]*ProductOverviewResponse, 0, len(products))
	for _, p := range products {
		if ov := ToOverview(p); ov != nil {
			res = append(res, ov)
		}
	}
	return res
}
