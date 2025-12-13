package responses

import (
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
	"core-backend/pkg/utils"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/datatypes"
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
	pr.BrandID = *m.BrandID
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
	ID           uuid.UUID               `json:"id"`
	BrandID      uuid.UUID               `json:"brand_id"`
	BrandLogoURL *string                 `json:"brand_logo_url,omitempty"`
	BrandName    string                  `json:"brand_name,omitempty"`    // optional
	ThumbnailURL *[]string               `json:"thumbnail_url,omitempty"` // optional
	IsActive     bool                    `json:"is_active"`
	Category     ProductCategoryResponse `json:"category"`
	Description  string                  `json:"description"`
	Name         string                  `json:"name"`
	//Price            float64                  `json:"price"`
	Type             enum.ProductType         `json:"type"`
	LimitedAttribute *LimitedProductResponse  `json:"limited_product"`
	Variants         []ProductVariantResponse `json:"variants,omitempty"`
	CreatedAt        string                   `json:"created_at"` // FE parse về Date
	Concept          *model.Concept           `json:"concept,omitempty"`
	AverageRating    float64                  `json:"average_rating"`
	//Reviews          []ProductReviewResponse  `json:"reviews,omitempty"`
}

type LimitedProductResponse struct {
	AchievableQuantity    int    `json:"achievable_quantity"`
	PremiereDate          string `json:"premiere_date"`
	AvailabilityStartDate string `json:"availability_start_date"`
	AvailabilityEndDate   string `json:"availability_end_date"`
}

func (l LimitedProductResponse) ToLimitedProductResponse(m *model.LimitedProduct) *LimitedProductResponse {
	if m == nil {
		return nil
	}
	return &LimitedProductResponse{
		AchievableQuantity:    m.AchievableQuantity,
		PremiereDate:          utils.FormatLocalTime(&m.PremiereDate, ""),
		AvailabilityStartDate: utils.FormatLocalTime(&m.AvailabilityStartDate, ""),
		AvailabilityEndDate:   utils.FormatLocalTime(&m.AvailabilityEndDate, ""),
	}
}

func (d ProductDetailResponse) ToProductDetailResponse(m *model.Product) *ProductDetailResponse {
	// IDs & Brand
	d.ID = m.ID
	d.BrandID = *m.BrandID
	if m.Brand != nil {
		d.BrandName = m.Brand.Name
		d.BrandLogoURL = m.Brand.LogoURL // *string
	}

	// Basic
	d.Name = m.Name
	if m.Description != nil {
		d.Description = *m.Description
	}
	d.Type = m.Type

	// Category
	if m.Category != nil {
		response := ProductCategoryResponse{}
		catPtr := response.ToModelResponse(m.Category)
		if catPtr != nil {
			d.Category = *catPtr
		}
	}

	// Status & time
	d.IsActive = m.Status == enum.ProductStatusActived
	d.CreatedAt = utils.FormatLocalTime(&m.CreatedAt, "")

	// Thumbnail
	d.ThumbnailURL = primaryProductImageURL(m)

	// Variants
	if len(m.Variants) > 0 {
		variants := make([]ProductVariantResponse, 0, len(m.Variants))
		for i := range m.Variants {
			variants = append(variants, *ProductVariantResponse{}.ToProductVariantResponse(&m.Variants[i]))
		}
		d.Variants = variants
	}

	if m.Limited != nil {
		d.LimitedAttribute = LimitedProductResponse{}.ToLimitedProductResponse(m.Limited)
		d.Concept = m.Limited.Concept
	}

	// Map reviews if preloaded
	d.AverageRating = m.AverageRating
	return &d
}

type ProductReviewResponse struct {
	ID                 uuid.UUID  `json:"id"`
	ProductID          uuid.UUID  `json:"product_id"`
	ProductVariantName string     `json:"product_variant_name"`
	UserID             *uuid.UUID `json:"user_id,omitempty"`
	UserName           *string    `json:"user_name,omitempty"`
	RatingStars        int        `json:"rating_stars"`
	Comment            *string    `json:"comment,omitempty"`
	AssetsURL          *string    `json:"assets_url,omitempty"`
	CreatedAt          string     `json:"created_at"`
	OrderAt            string     `json:"order_at"`
	Type               string     `json:"type"`
}

func (ProductReviewResponse) ToResponse(m *model.ProductReview) *ProductReviewResponse {
	timelayout := "2006-01-02 15:04:05"
	var (
		orderAt            string
		userName           *string
		orderType          string
		productVariantName string
	)
	if m.OrderItemID != nil {
		orderType = "ORDER"
	} else {
		orderType = "PREORDER"
	}

	if o := m.OrderItem; o != nil && o.Order != nil {
		orderAt = o.Order.CreatedAt.Format(timelayout)
		userName = &o.Order.FullName
		if v := o.Variant; v.Product != nil {

			nameFmt := "%s - (Ingredient: %s)"
			var ingredientConcat string

			rawJson := m.OrderItem.AttributesDescription
			if rawJson != nil {
				var attrs []map[string]interface{}
				if err := json.Unmarshal(*rawJson, &attrs); err != nil {
					ingredientConcat = ""
				}

				for _, item := range attrs {
					var tmp string
					if ingredient, ok := item["ingredient"].(string); ok {
						if ingredient != "" {
							tmp = ingredient
						}
					}
					ingredientValue, okValue := item["value"].(int)
					ingredientUnit, okUnit := item["unit"].(string)
					if okValue && okUnit {
						tmp += fmt.Sprintf(" :%d %s", ingredientValue, ingredientUnit)
					}
					ingredientConcat += tmp + ", "
				}
			}
			// Remove trailing comma and space
			if len(ingredientConcat) >= 2 {
				ingredientConcat = ingredientConcat[:len(ingredientConcat)-2]
			}
			productVariantName = fmt.Sprintf(nameFmt, v.Product.Name, ingredientConcat)
		}

	}
	return &ProductReviewResponse{
		ID:                 m.ID,
		ProductID:          m.ProductID,
		ProductVariantName: productVariantName,
		UserID:             m.UserID,
		UserName:           userName,
		RatingStars:        m.RatingStars,
		Comment:            m.Comment,
		AssetsURL:          m.AssetsURL,
		CreatedAt:          m.CreatedAt.Format(timelayout),
		OrderAt:            orderAt,
		Type:               orderType,
	}
}

func (ProductReviewResponse) ToResponseList(m []model.ProductReview) []ProductReviewResponse {
	reviews := make([]ProductReviewResponse, 0, len(m))
	for i := range m {
		review := ProductReviewResponse{}.ToResponse(&m[i])
		reviews = append(reviews, *review)
	}
	return reviews
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
	MaxStock        *int                        `json:"max_stock,omitempty"`
	CurrentStock    *int                        `json:"current_stock,omitempty"`
	PreOrderLimit   *int                        `json:"pre_order_limit,omitempty"`
	PreOrderCount   *int                        `json:"pre_order_count,omitempty"`
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
	Weight          int                         `json:"weight"` // in grams
	Height          int                         `json:"height"` // in centimeters
	Length          int                         `json:"length"` // in centimeters
	Width           int                         `json:"width"`  //
	Story           datatypes.JSON              `json:"story" swaggerignore:"true"`
	Attributes      []ProductAttributesResponse `json:"attributes"`
	Images          []VariantImageResponse      `json:"images"`
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
		Name:            "",
		Description:     nil,
		Price:           variant.Price,
		Type:            "",
		MaxStock:        variant.MaxStock,
		CurrentStock:    variant.CurrentStock,
		PreOrderLimit:   variant.PreOrderLimit,
		PreOrderCount:   variant.PreOrderCount,
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
		Weight:          variant.Weight,
		Height:          variant.Height,
		Length:          variant.Length,
		Width:           variant.Width,
		Story:           nil,
		Attributes:      nil,
		Images:          nil,
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

// ToProductVariantResponse converts a ProductVariant model to a ProductVariantResponse DTO.

func (pvr ProductVariantResponse) ToFullProductVariantResponse(variant *model.ProductVariant, story *model.ProductStory, attributeValueList []ProductAttributesResponse) *ProductVariantResponse {
	if variant == nil {
		return nil
	}

	resp := ProductVariantResponse{
		ID:              variant.ID,
		Price:           variant.Price,
		MaxStock:        variant.MaxStock,
		CurrentStock:    variant.CurrentStock,
		PreOrderLimit:   variant.PreOrderLimit,
		PreOrderCount:   variant.PreOrderCount,
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
		Weight:          variant.Weight,
		Height:          variant.Height,
		Length:          variant.Length,
		Width:           variant.Width,
		Story:           nil,
		Attributes:      nil,
		Images:          nil,
	}

	// Include product-level fields if product relation is present
	if variant.Product != nil {
		resp.Name = variant.Product.Name
		resp.Description = variant.Product.Description
		resp.Type = variant.Product.Type
	}

	// Prefer provided story param if present, otherwise fall back to preloaded variant.Story
	if story != nil {
		resp.Story = story.Content
	} else if variant.Story != nil {
		resp.Story = variant.Story.Content
	}

	if len(attributeValueList) > 0 {
		resp.Attributes = attributeValueList
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
	ID               uuid.UUID                 `json:"id"`
	BrandID          uuid.UUID                 `json:"brand_id"`
	BrandLogoURL     *string                   `json:"brand_logo_url,omitempty"`
	BrandName        string                    `json:"brand_name,omitempty"`    // optional
	ThumbnailURL     *[]string                 `json:"thumbnail_url,omitempty"` // optional
	IsActive         bool                      `json:"is_active"`
	CreatedAt        string                    `json:"created_at"` // FE parse về Date
	UpdatedAt        string                    `json:"updated_at"`
	Status           enum.ProductStatus        `json:"status"`
	Description      string                    `json:"description"`
	Name             string                    `json:"name"`
	Type             enum.ProductType          `json:"type"`
	AverageRating    float64                   `json:"average_rating"`
	LimitedAttribute *LimitedProductResponse   `json:"limited_product"`
	Category         ProductCategoryResponse   `json:"category"`
	Variants         []*ProductVariantResponse `json:"variants,omitempty"`
	CreatedBy        *UserListResponse         `json:"created_by"`
	UpdatedBy        *UserListResponse         `json:"updated_by"`
}

// ToProductResponseV2 converts a Product model to a ProductResponse DTO.
func (pr *ProductResponseV2) ToProductResponseV2(m *model.Product) *ProductResponseV2 {
	var limitedResp *LimitedProductResponse
	if m.Limited != nil {
		limitedResp = LimitedProductResponse{}.ToLimitedProductResponse(m.Limited)
	}

	return &ProductResponseV2{
		ID:      m.ID,
		BrandID: *m.BrandID,

		// Brand
		BrandName:    utils.IfNotNil(m.Brand, func(b *model.Brand) string { return b.Name }),
		BrandLogoURL: utils.IfNotNil(m.Brand, func(b *model.Brand) *string { return b.LogoURL }),

		// Basic
		Name:             m.Name,
		Description:      utils.IfNotNil(m.Description, func(s *string) string { return *s }),
		Type:             m.Type,
		AverageRating:    m.AverageRating,
		LimitedAttribute: limitedResp,
		IsActive:         m.IsActive,

		CreatedAt: utils.FormatLocalTime(&m.CreatedAt, ""),
		UpdatedAt: utils.FormatLocalTime(&m.UpdatedAt, ""),

		Status: m.Status,

		// Category
		Category: utils.IfNotNil(m.Category, func(c *model.ProductCategory) ProductCategoryResponse {
			resp := ProductCategoryResponse{}
			return *resp.ToModelResponse(c)
		}),

		// Thumbnail
		ThumbnailURL: primaryProductImageURL(m),

		// CreatedBy
		CreatedBy: utils.IfNotNil(m.CreatedBy,
			func(u *model.User) *UserListResponse {
				return UserListResponse{}.ToSingleUserListResponse(*u)
			},
		),

		// UpdatedBy
		UpdatedBy: utils.IfNotNil(m.UpdatedBy,
			func(u *model.User) *UserListResponse {
				return UserListResponse{}.ToSingleUserListResponse(*u)
			},
		),

		// Variants
		Variants: func() []*ProductVariantResponse {
			if len(m.Variants) == 0 {
				return nil
			}
			v := make([]*ProductVariantResponse, len(m.Variants))
			for i := range m.Variants {
				v[i] = ProductVariantResponse{}.ToProductVariantResponse(&m.Variants[i])
			}
			return v
		}(),
	}
}

// ProductResponseV2Partial represents a partial response structure for a product.
type ProductResponseV2Partial struct {
	ID               uuid.UUID                  `json:"id"`
	BrandID          uuid.UUID                  `json:"brand_id"`
	BrandLogoURL     *string                    `json:"brand_logo_url,omitempty"`
	BrandName        string                     `json:"brand_name,omitempty"`    // optional
	ThumbnailURL     *[]string                  `json:"thumbnail_url,omitempty"` // optional
	Category         ProductCategoryResponse    `json:"category"`
	Description      string                     `json:"description"`
	Name             string                     `json:"name"`
	Type             enum.ProductType           `json:"type"`
	AverageRating    float64                    `json:"average_rating"`
	LimitedAttribute *LimitedProductResponse    `json:"limited_product"`
	Variants         *[]*ProductVariantResponse `json:"variants,omitempty"`
}

func (pr *ProductResponseV2Partial) ToProductResponseV2(m *model.Product) *ProductResponseV2Partial {
	if pr == nil {
		pr = &ProductResponseV2Partial{}
	}

	var limitedResp *LimitedProductResponse
	if m.Limited != nil {
		limitedResp = LimitedProductResponse{}.ToLimitedProductResponse(m.Limited)
	}
	pr.LimitedAttribute = limitedResp

	// IDs & Brand
	pr.ID = m.ID
	pr.BrandID = *m.BrandID
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
	pr.AverageRating = m.AverageRating

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

// =======================ProductReviewResponseStaff (Start)=======================
type ProductReviewResponseStaff struct {
	UserInfo      *ProductReviewUserInfo    `json:"user_info"`
	Product       *ProductReviewProductInfo `json:"product"`
	ReviewContent *ProductReviewContent     `json:"review_content"`
}
type ProductReviewUserInfo struct {
	ID        uuid.UUID  `json:"id"`
	UserID    *uuid.UUID `json:"user_id,omitempty"`
	UserName  string     `json:"user_name,omitempty"`
	FullName  string     `json:"full_name,omitempty"`
	AvatarURL *string    `json:"avatar_url,omitempty"`
}

func (ProductReviewUserInfo) ToProductReviewUserInfo(m *model.User) *ProductReviewUserInfo {
	if m == nil {
		return nil
	}
	return &ProductReviewUserInfo{
		ID:        m.ID,
		UserID:    &m.ID,
		UserName:  m.Username,
		FullName:  m.FullName,
		AvatarURL: m.AvatarURL,
	}
}

type ProductReviewProductInfo struct {
	ID            uuid.UUID          `json:"id"`
	CapacityUnit  enum.CapacityUnit  `json:"capacity_unit"`
	ContainerType enum.ContainerType `json:"container_type"`
	DispenserType enum.DispenserType `json:"dispenser_type"`
	Weight        int                `json:"weight"`
	Height        int                `json:"height"`
	Length        int                `json:"length"`
	Width         int                `json:"width"`

	//Product stats
	ProductID uuid.UUID `json:"product_id"`
	Name      string    `json:"name"`

	Type        enum.ProductType        `json:"type"`
	Category    ProductCategoryResponse `json:"category"`
	LimitedInfo *LimitedProductResponse `json:"limited_info"`
	Brand       *BrandResponse          `json:"brand"`
}

func (pr ProductReviewProductInfo) ToProductReviewProductInfo(m *model.ProductVariant) *ProductReviewProductInfo {
	if m == nil {
		return nil
	}

	var prdName string
	var prdType enum.ProductType
	var prdCategory ProductCategoryResponse
	var limitedInfo *LimitedProductResponse
	var brandResp *BrandResponse
	if m.Product != nil {
		prdName = m.Product.Name
		prdType = m.Product.Type
		if m.Product.Category != nil {
			catResp := ProductCategoryResponse{}
			prdCategory = *catResp.ToModelResponse(m.Product.Category)
		}
		if m.Product.Limited != nil {
			limitedInfo = LimitedProductResponse{}.ToLimitedProductResponse(m.Product.Limited)
		}
		if m.Product.Brand != nil {
			brandResp = BrandResponse{}.ToBrandResponse(m.Product.Brand)
		}
	}

	return &ProductReviewProductInfo{
		ID:            m.ID,
		CapacityUnit:  m.CapacityUnit,
		ContainerType: m.ContainerType,
		DispenserType: m.DispenserType,
		Weight:        m.Weight,
		Height:        m.Height,
		Length:        m.Length,
		Width:         m.Width,
		ProductID:     m.ProductID,
		Name:          prdName,
		Type:          prdType,
		Category:      prdCategory,
		LimitedInfo:   limitedInfo,
		Brand:         brandResp,
	}
}

type ProductReviewContent struct {
	RatingStars int     `json:"rating_stars"`
	Comment     *string `json:"comment,omitempty"`
	AssetsURL   *string `json:"assets_url,omitempty"`
	CreatedAt   string  `json:"created_at"`
}

func (ProductReviewContent) ToProductReviewContent(m *model.ProductReview) *ProductReviewContent {
	if m == nil {
		return nil
	}
	return &ProductReviewContent{
		RatingStars: m.RatingStars,
		Comment:     m.Comment,
		AssetsURL:   m.AssetsURL,
		CreatedAt:   utils.FormatLocalTime(&m.CreatedAt, "2006-01-02 15:04:05"),
	}
}

func (ProductReviewResponseStaff) ToProductReviewResponseStaff(m *model.ProductReview) *ProductReviewResponseStaff {
	var userInfo *ProductReviewUserInfo
	var productInfo *ProductReviewProductInfo
	var reviewContent *ProductReviewContent

	if m == nil {
		return nil
	} else {
		reviewContent = ProductReviewContent{}.ToProductReviewContent(m)
	}

	if m.User.ID != uuid.Nil {
		userInfo = ProductReviewUserInfo{}.ToProductReviewUserInfo(&m.User)
	}
	if m.OrderItem != nil && m.OrderItem.Variant.ID != uuid.Nil {
		productInfo = ProductReviewProductInfo{}.ToProductReviewProductInfo(&m.OrderItem.Variant)
	} else if m.PreOrder != nil && m.PreOrder.ProductVariant != nil {
		productInfo = ProductReviewProductInfo{}.ToProductReviewProductInfo(m.PreOrder.ProductVariant)
	}

	return &ProductReviewResponseStaff{
		UserInfo:      userInfo,
		Product:       productInfo,
		ReviewContent: reviewContent,
	}
}

func (ProductReviewResponseStaff) ToProductReviewResponseStaffList(m []model.ProductReview) []ProductReviewResponseStaff {
	reviews := make([]ProductReviewResponseStaff, 0, len(m))
	for i := range m {
		review := ProductReviewResponseStaff{}.ToProductReviewResponseStaff(&m[i])
		reviews = append(reviews, *review)
	}
	return reviews
}

//=======================ProductReviewResponseStaff (END)=======================
