package responses

import (
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
	"core-backend/pkg/utils"

	"github.com/google/uuid"
)

// ProductResponse represents the response structure for a product.
type ProductResponse struct {
	ID           uuid.UUID        `json:"id"`
	BrandID      uuid.UUID        `json:"brand_id"`
	BrandName    string           `json:"brand_name"`
	BrandLogoURL *string          `json:"brand_logo_url"`
	Name         string           `json:"name"`
	Description  string           `json:"description"`
	Price        float64          `json:"price"`
	Type         enum.ProductType `json:"type"`
	CategoryLv1  string           `json:"category"`
	CategoryLv2  string           `json:"category_lv2"`

	Variants []*ProductVariantResponse `json:"variants,omitempty"`
}

// ToProductResponse converts a Product model to a ProductResponse DTO.
func (pr *ProductResponse) ToProductResponse(m *model.Product) *ProductResponse {
	if pr == nil {
		pr = &ProductResponse{}
	}
	pr.ID = m.ID
	pr.BrandID = m.BrandID
	if m.Brand != nil {
		pr.BrandName = m.Brand.Name
		pr.BrandLogoURL = m.Brand.LogoURL
	}
	pr.Name = m.Name
	if m.Description != nil {
		pr.Description = *m.Description
	}
	pr.Price = m.Price
	pr.Type = m.Type
	if m.Category != nil {
		pr.CategoryLv1 = m.Category.Name
		if m.Category.ParentCategory != nil {
			pr.CategoryLv2 = m.Category.ParentCategory.Name
		}
	}
	if len(m.Variants) > 0 {
		variants := make([]*ProductVariantResponse, 0, len(m.Variants))
		for i := range m.Variants {
			variants = append(variants, ProductVariantResponse{}.ToProductVariantResponse(&m.Variants[i]))
		}
		pr.Variants = variants
	}
	return pr
}

// ProductVariantResponse represents the response structure for a product variant.
type ProductVariantResponse struct {
	ID              uuid.UUID                   `json:"id,omitempty"`
	Name            string                      `json:"name,omitempty"`
	Description     *string                     `json:"description,omitempty"`
	Price           float64                     `json:"price,omitempty"`
	Type            enum.ProductType            `json:"type,omitempty"`
	CurrentStock    int                         `json:"current_stock,omitempty"`
	Capacity        float64                     `json:"capacity,omitempty"`
	CapacityUnit    enum.CapacityUnit           `json:"capacity_unit,omitempty"`
	ContainerType   enum.ContainerType          `json:"container_type,omitempty"`
	DispenserType   enum.DispenserType          `json:"dispenser_type,omitempty"`
	Uses            string                      `json:"uses,omitempty"`
	ManufactureDate string                      `json:"manufactring_date,omitempty"`
	ExpiryDate      string                      `json:"expiry_date,omitempty"`
	Instructions    string                      `json:"instructions,omitempty"`
	IsDefault       bool                        `json:"is_default,omitempty"`
	CreatedAt       string                      `json:"created_at"`
	UpdatedAt       string                      `json:"updated_at"`
	Story           []byte                      `json:"story,omitempty"`
	Attributes      []ProductAttributesResponse `json:"attributes,omitempty"`
}

// ProductAttributesResponse represents the attributes of a product variant.
type ProductAttributesResponse struct {
	Ingredient  string             `json:"ingredient,omitempty"`
	Description *string            `json:"description,omitempty"`
	Value       float64            `json:"value,omitempty"`
	Unit        enum.AttributeUnit `json:"unit,omitempty"`
}

// ToProductVariantResponse converts a ProductVariant model to a ProductVariantResponse DTO.
func (pvr ProductVariantResponse) ToProductVariantResponse(variant *model.ProductVariant) *ProductVariantResponse {
	resp := &ProductVariantResponse{
		ID:              variant.ID,
		Price:           variant.Price,
		CurrentStock:    variant.CurrentStock,
		Capacity:        variant.Capacity,
		CapacityUnit:    variant.CapacityUnit,
		ContainerType:   variant.ContainerType,
		DispenserType:   variant.DispenserType,
		Uses:            variant.Uses,
		ManufactureDate: utils.FormatLocalTime(variant.ManufactureDate, ""),
		ExpiryDate:      utils.FormatLocalTime(variant.ExpiryDate, ""),
		Instructions:    variant.Instructions,
		IsDefault:       variant.IsDefault,
		CreatedAt:       utils.FormatLocalTime(&variant.CreatedAt, ""),
		UpdatedAt:       utils.FormatLocalTime(&variant.UpdatedAt, ""),
	}
	if variant.Product != nil {
		resp.Name = variant.Product.Name
		resp.Description = variant.Product.Description
		resp.Price = variant.Product.Price // override with product base price if desired
		resp.Type = variant.Product.Type
	}
	if variant.Story != nil {
		resp.Story = variant.Story.Content
	}
	if len(variant.AttributeValues) > 0 {
		attrs := make([]ProductAttributesResponse, 0, len(variant.AttributeValues))
		for i := range variant.AttributeValues {
			av := variant.AttributeValues[i]
			if av.Attribute != nil { // ensure preloaded
				attrs = append(attrs, ProductAttributesResponse{
					Ingredient:  av.Attribute.Ingredient,
					Description: av.Attribute.Description,
					Value:       av.Value,
					Unit:        av.Unit,
				})
			}
		}
		resp.Attributes = attrs
	}
	return resp
}
