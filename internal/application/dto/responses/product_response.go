package responses

import (
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
	"core-backend/pkg/utils"
	"github.com/google/uuid"
)

/*===========================PRODUCTS Overview=====================================*/
// ProductResponse represents the response structure for a product.
type ProductResponse struct {
	ID           uuid.UUID                  `json:"id"`
	BrandID      uuid.UUID                  `json:"brand_id"`
	BrandLogoURL *string                    `json:"brand_logo_url,omitempty"`
	BrandName    string                     `json:"brand_name,omitempty"`    // optional
	ThumbnailURL *[]string                  `json:"thumbnail_url,omitempty"` // optional
	IsActive     bool                       `json:"is_active"`
	CategoryLv1  string                     `json:"category"`
	CategoryLv2  string                     `json:"category_lv2"`
	Description  string                     `json:"description"`
	Name         string                     `json:"name"`
	Price        float64                    `json:"price"`
	Type         enum.ProductType           `json:"type"`
	Variants     *[]*ProductVariantResponse `json:"variants,omitempty"`
	CreatedAt    string                     `json:"created_at"` // FE parse về Date
}

// ToProductResponse converts a Product model to a ProductResponse DTO.
func (pr *ProductResponse) ToProductResponse(m *model.Product) *ProductResponse {
	if pr == nil {
		pr = &ProductResponse{}
	}

	// IDs & Brand
	pr.ID = m.ID
	pr.BrandID = m.BrandID
	if m.Brand != nil {
		pr.BrandName = m.Brand.Name
		pr.BrandLogoURL = m.Brand.LogoURL // *string
	}

	// Basic
	pr.Name = m.Name
	if m.Description != nil {
		pr.Description = *m.Description
	}
	pr.Type = m.Type

	// Category level 1/2
	if m.Category != nil {
		pr.CategoryLv1 = m.Category.Name
		if m.Category.ParentCategory != nil {
			pr.CategoryLv2 = m.Category.ParentCategory.Name
		}
	}

	// Status & time
	pr.IsActive = m.Status == enum.ProductStatusActived
	pr.CreatedAt = utils.FormatLocalTime(&m.CreatedAt, "")

	// Thumbnail
	pr.ThumbnailURL = primaryProductImageURL(m)

	// Variants
	if len(m.Variants) > 0 {
		variants := make([]*ProductVariantResponse, 0, len(m.Variants))
		for i := range m.Variants {
			variants = append(variants, ProductVariantResponse{}.ToProductVariantResponse(&m.Variants[i]))
		}
		pr.Variants = &variants
	}

	return pr
}

/*===========================PRODUCTS DETAIL=====================================*/
type ProductDetailResponse struct {
	ID           uuid.UUID                  `json:"id"`
	BrandID      uuid.UUID                  `json:"brand_id"`
	BrandLogoURL *string                    `json:"brand_logo_url,omitempty"`
	BrandName    string                     `json:"brand_name,omitempty"`    // optional
	ThumbnailURL *[]string                  `json:"thumbnail_url,omitempty"` // optional
	IsActive     bool                       `json:"is_active"`
	CategoryLv1  string                     `json:"category"`
	CategoryLv2  string                     `json:"category_lv2"`
	Description  string                     `json:"description"`
	Name         string                     `json:"name"`
	Price        float64                    `json:"price"`
	Type         enum.ProductType           `json:"type"`
	Variants     *[]*ProductVariantResponse `json:"variants,omitempty"`
	CreatedAt    string                     `json:"created_at"` // FE parse về Date
}

func primaryProductImageURL(p *model.Product) *[]string {
	if p == nil || len(p.Variants) == 0 {
		return nil
	}

	thumnail := []string{}
	fallback := []string{}

	for i := range p.Variants {
		for j := range p.Variants[i].Images {
			if p.Variants[i].Images[j].IsPrimary {
				thumnail = append(thumnail, p.Variants[i].Images[j].ImageURL)
			}
			// fallback first image
			if len(fallback) == 0 {
				fallback = append(fallback, p.Variants[i].Images[j].ImageURL)
			}
		}
	}

	if len(thumnail) > 0 {
		return &thumnail
	} else {
		return &fallback
	}
}

// ProductVariantResponse represents the response structure for a product variant.
type ProductVariantResponse struct {
	ID              uuid.UUID                   `json:"id,omitempty"`
	Name            string                      `json:"name,omitempty"`
	Description     *string                     `json:"description,omitempty"`
	Price           float64                     `json:"price,omitempty"`
	Type            enum.ProductType            `json:"type,omitempty"`
	CurrentStock    *int                        `json:"current_stock,omitempty"`
	Capacity        float64                     `json:"capacity,omitempty"`
	CapacityUnit    enum.CapacityUnit           `json:"capacity_unit,omitempty"`
	ContainerType   enum.ContainerType          `json:"container_type,omitempty"`
	DispenserType   enum.DispenserType          `json:"dispenser_type,omitempty"`
	Uses            string                      `json:"uses,omitempty"`
	ManufactureDate string                      `json:"manufacturing_date,omitempty"`
	ExpiryDate      string                      `json:"expiry_date,omitempty"`
	Instructions    string                      `json:"instructions,omitempty"`
	IsDefault       bool                        `json:"is_default,omitempty"`
	CreatedAt       string                      `json:"created_at"`
	UpdatedAt       string                      `json:"updated_at"`
	Story           []byte                      `json:"story,omitempty"`
	Attributes      []ProductAttributesResponse `json:"attributes,omitempty"`
	Images          []VariantImageResponse      `json:"images,omitempty"`
}

// ProductAttributesResponse represents the attributes of a product variant.
type ProductAttributesResponse struct {
	//Attribute
	Ingredient  string  `json:"ingredient,omitempty"`
	Description *string `json:"description,omitempty"`
	//Value
	Value float64            `json:"value,omitempty"`
	Unit  enum.AttributeUnit `json:"unit,omitempty"`
}

// ToProductVariantResponse converts a ProductVariant model to a ProductVariantResponse DTO.
func (pvr ProductVariantResponse) ToProductVariantResponse(variant *model.ProductVariant) *ProductVariantResponse {
	resp := ProductVariantResponse{
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

	if len(variant.Images) > 0 {
		images := make([]VariantImageResponse, 0, len(variant.Images))
		for i := range variant.Images {
			vir := VariantImageResponse{}.ToVariantImageResponse(&variant.Images[i])
			if vir != nil {
				images = append(images, *vir)
			}
		}
		resp.Images = images
	}
	return &resp
}

// TODO:====================================== VERSION 2======================================================
type ProductResponseV2 struct {
	ID           uuid.UUID                 `json:"id"`
	BrandID      uuid.UUID                 `json:"brand_id"`
	BrandLogoURL *string                   `json:"brand_logo_url,omitempty"`
	BrandName    string                    `json:"brand_name,omitempty"`    // optional
	ThumbnailURL *[]string                 `json:"thumbnail_url,omitempty"` // optional
	IsActive     bool                      `json:"is_active"`
	Category     ProductCategoryResponse   `json:"category"`
	Description  string                    `json:"description"`
	Name         string                    `json:"name"`
	Price        float64                   `json:"price"`
	Type         enum.ProductType          `json:"type"`
	Variants     []*ProductVariantResponse `json:"variants,omitempty"`
	CreatedAt    string                    `json:"created_at"` // FE parse về Date
	UpdatedAt    string                    `json:"updated_at"`
	Status       enum.ProductStatus        `json:"status"`
	CreatedBy    *UserListResponse         `json:"created_by"`
	UpdatedBy    *UserListResponse         `json:"updated_by"`
}

// ToProductResponse converts a Product model to a ProductResponse DTO.
func (pr *ProductResponseV2) ToProductResponseV2(m *model.Product) *ProductResponseV2 {
	if pr == nil {
		pr = &ProductResponseV2{}
	}

	// IDs & Brand
	pr.ID = m.ID
	pr.BrandID = m.BrandID
	if m.Brand != nil {
		pr.BrandName = m.Brand.Name
		pr.BrandLogoURL = m.Brand.LogoURL // *string
	}

	// Basic
	pr.Name = m.Name
	if m.Description != nil {
		pr.Description = *m.Description
	}
	pr.Type = m.Type

	// Category
	if m.Category != nil {
		response := ProductCategoryResponse{}
		catPtr := response.ToModelResponse(m.Category)
		if catPtr != nil {
			pr.Category = *catPtr
		}
	}

	// Status & time
	pr.IsActive = m.Status == enum.ProductStatusActived
	pr.CreatedAt = utils.FormatLocalTime(&m.CreatedAt, "")

	// Thumbnail
	pr.ThumbnailURL = primaryProductImageURL(m)

	if m.CreatedBy != nil {
		pr.CreatedBy = UserListResponse{}.ToSingleUserListResponse(*m.CreatedBy)
	}
	if m.UpdatedBy != nil {
		pr.UpdatedBy = UserListResponse{}.ToSingleUserListResponse(*m.UpdatedBy)
	}

	// Variants
	if len(m.Variants) > 0 {
		variants := make([]*ProductVariantResponse, 0, len(m.Variants))
		for i := range m.Variants {
			variants = append(variants, ProductVariantResponse{}.ToProductVariantResponse(&m.Variants[i]))
		}
		pr.Variants = variants
	}

	return pr
}

// ProductV2ForCustomer
type ProductResponseV2Partial struct {
	ID           uuid.UUID                  `json:"id"`
	BrandID      uuid.UUID                  `json:"brand_id"`
	BrandLogoURL *string                    `json:"brand_logo_url,omitempty"`
	BrandName    string                     `json:"brand_name,omitempty"`    // optional
	ThumbnailURL *[]string                  `json:"thumbnail_url,omitempty"` // optional
	Category     ProductCategoryResponse    `json:"category"`
	Description  string                     `json:"description"`
	Name         string                     `json:"name"`
	Price        float64                    `json:"price"`
	Type         enum.ProductType           `json:"type"`
	Variants     *[]*ProductVariantResponse `json:"variants,omitempty"`
}

func (pr *ProductResponseV2Partial) ToProductResponseV2(m *model.Product) *ProductResponseV2Partial {
	if pr == nil {
		pr = &ProductResponseV2Partial{}
	}

	// IDs & Brand
	pr.ID = m.ID
	pr.BrandID = m.BrandID
	if m.Brand != nil {
		pr.BrandName = m.Brand.Name
		pr.BrandLogoURL = m.Brand.LogoURL // *string
	}

	// Basic
	pr.Name = m.Name
	if m.Description != nil {
		pr.Description = *m.Description
	}
	pr.Type = m.Type

	// Category
	if m.Category != nil {
		response := ProductCategoryResponse{}
		catPtr := response.ToModelResponse(m.Category)
		if catPtr != nil {
			pr.Category = *catPtr
		}
	}

	// Thumbnail
	pr.ThumbnailURL = primaryProductImageURL(m)

	// Variants
	if len(m.Variants) > 0 {
		variants := make([]*ProductVariantResponse, 0, len(m.Variants))
		for i := range m.Variants {
			variants = append(variants, ProductVariantResponse{}.ToProductVariantResponse(&m.Variants[i]))
		}
		pr.Variants = &variants
	}

	return pr
}

// ToProductVariantResponseV2 converts a ProductVariant model to a ProductVariantResponse DTO.
func (pvr ProductVariantResponse) ToProductVariantResponseV2(variant *model.ProductVariant) *ProductVariantResponse {
	resp := ProductVariantResponse{
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

	if len(variant.Images) > 0 {
		images := make([]VariantImageResponse, 0, len(variant.Images))
		for i := range variant.Images {
			vir := VariantImageResponse{}.ToVariantImageResponse(&variant.Images[i])
			if vir != nil {
				images = append(images, *vir)
			}
		}
		resp.Images = images
	}
	return &resp
}

type ProductResponseTop5Newest struct {
	Standard []ProductResponseV2Partial `json:"standard"`
	Limited  []ProductResponseV2Partial `json:"limited"`
}
