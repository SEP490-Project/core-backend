package responses

import (
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"

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
	CategoryLv1  string           `json:"category_lv1"`
	CategoryLv2  string           `json:"category_lv2"`

	Variants []*ProductVariantResponse `json:"variants,omitempty"`
}

// ToProductResponse converts a Product model to a ProductResponse DTO.
func (pr *ProductResponse) ToProductResponse(model *model.Product) *ProductResponse {
	pr.ID = model.ID
	pr.BrandID = model.BrandID
	pr.BrandName = model.Brand.Name
	pr.BrandLogoURL = model.Brand.LogoURL
	pr.Name = model.Name
	pr.Description = *model.Description
	pr.Price = model.Price
	pr.Type = model.Type
	if model.Category != nil {
		categoryModel := model.Category
		pr.CategoryLv1 = categoryModel.Name
		if categoryModel.ParentCategory != nil {
			pr.CategoryLv2 = categoryModel.ParentCategory.Name
		}
	}
	if len(model.Variants) > 0 {
		variantResponse := make([]*ProductVariantResponse, len(model.Variants))
		for _, v := range model.Variants {
			variantResponse = append(variantResponse, ProductVariantResponse{}.ToProductVariantResponse(&v))
		}
		pr.Variants = variantResponse
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
	response := &ProductVariantResponse{
		ID:              variant.ID,
		Price:           variant.Price,
		CurrentStock:    variant.CurrentStock,
		Capacity:        variant.Capacity,
		CapacityUnit:    variant.CapacityUnit,
		ContainerType:   variant.ContainerType,
		DispenserType:   variant.DispenserType,
		Uses:            variant.Uses,
		ManufactureDate: variant.ManufactureDate.Format(TimeFormat),
		ExpiryDate:      variant.ExpiryDate.Format(TimeFormat),
		Instructions:    variant.Instructions,
		IsDefault:       variant.IsDefault,
		CreatedAt:       variant.CreatedAt.Format(TimeFormat),
		UpdatedAt:       variant.UpdatedAt.Format(TimeFormat),
	}
	if variant.Product != nil {
		response.Name = variant.Product.Name
		response.Description = variant.Product.Description
		response.Price = variant.Product.Price
		response.Type = variant.Product.Type
	}
	if variant.Story != nil {
		response.Story = variant.Story.Content
	}
	if len(variant.AttributeValues) > 0 {
		attributes := make([]ProductAttributesResponse, 0, len(variant.AttributeValues))
		for _, attr := range variant.AttributeValues {
			attrResp := ProductAttributesResponse{
				Ingredient:  attr.Attribute.Ingredient,
				Description: attr.Attribute.Description,
				Value:       attr.Value,
				Unit:        attr.Unit,
			}
			attributes = append(attributes, attrResp)
		}
		response.Attributes = attributes
	}

	return response
}
