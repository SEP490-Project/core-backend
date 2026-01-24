package responses

import (
	"core-backend/internal/domain/model"
	"time"

	"gorm.io/datatypes"

	"github.com/google/uuid"
)

// OrderResponse is a sanitized DTO for returning orders to API clients
type OrderResponse struct {
	ID     uuid.UUID `json:"id"`
	UserID uuid.UUID `json:"user_id"`
	Status string    `json:"status"`

	BankAccount       string `json:"bank_account"`
	BankName          string `json:"bank_name"`
	BankAccountHolder string `json:"bank_account_holder"`

	TotalAmount       float64                     `json:"total_amount"`
	CompanyRevenue    *float64                    `json:"company_revenue,omitempty"`
	KOLRevenue        *float64                    `json:"kol_revenue,omitempty"`
	FullName          string                      `json:"full_name"`
	PhoneNumber       string                      `json:"phone_number"`
	Email             string                      `json:"email"`
	Street            string                      `json:"street"`
	AddressLine2      string                      `json:"address_line2"`
	City              string                      `json:"city"`
	GhnProvinceID     int                         `json:"ghn_province_id" gorm:"column:ghn_province_id"`
	GhnDistrictID     int                         `json:"ghn_district_id" gorm:"column:ghn_district_id"`
	GhnWardCode       string                      `json:"ghn_ward_code" gorm:"column:ghn_ward_code"`
	ProvinceName      string                      `json:"province_name" gorm:"column:province_name"`
	DistrictName      string                      `json:"district_name" gorm:"column:district_name"`
	WardName          string                      `json:"ward_name" gorm:"column:ward_name"`
	ShippingFee       int                         `json:"shipping_fee"`
	CreatedAt         time.Time                   `json:"created_at"2025ba`
	UpdatedAt         time.Time                   `json:"updated_at"`
	IsSelfPickedUp    bool                        `json:"is_self_picked_up"`
	ConfirmationImage *string                     `json:"confirmation_image"`
	UserResource      *string                     `json:"user_resource,omitempty"`
	StaffResource     *string                     `json:"staff_resource,omitempty"`
	OrderType         string                      `json:"order_type"`
	GHNOrderCode      *string                     `json:"ghn_order_code"`
	ActionNotes       *model.OrderActionNotes     `json:"action_notes"`
	UserNote          *string                     `json:"user_note"`
	Payment           *PaymentTransactionResponse `json:"payment_transaction"`
	OrderItems        []OrderItemResponse         `json:"order_items"`
}

func (OrderResponse) ToResponse(o *model.Order, pt *model.PaymentTransaction) *OrderResponse {
	if o == nil {
		return nil
	}

	var orderItems []OrderItemResponse
	if len(o.OrderItems) == 0 {
		orderItems = nil
	} else {
		orderItems = OrderItemResponse{}.ToResponseList(o.OrderItems)
	}

	resp := &OrderResponse{
		ID:          o.ID,
		UserID:      o.UserID,
		Status:      string(o.Status),
		TotalAmount: o.TotalAmount,

		BankAccount:       o.BankAccount,
		BankName:          o.BankName,
		BankAccountHolder: o.BankAccountHolder,

		FullName:          o.FullName,
		PhoneNumber:       o.PhoneNumber,
		Email:             o.Email,
		Street:            o.Street,
		AddressLine2:      o.AddressLine2,
		City:              o.City,
		GhnProvinceID:     o.GhnProvinceID,
		GhnDistrictID:     o.GhnDistrictID,
		GhnWardCode:       o.GhnWardCode,
		ProvinceName:      o.ProvinceName,
		DistrictName:      o.DistrictName,
		WardName:          o.WardName,
		ShippingFee:       o.ShippingFee,
		CreatedAt:         o.CreatedAt,
		UpdatedAt:         o.UpdatedAt,
		IsSelfPickedUp:    o.IsSelfPickedUp,
		ConfirmationImage: o.ConfirmationImage,
		UserResource:      o.UserResource,
		StaffResource:     o.StaffResource,
		ActionNotes:       o.ActionNotes,
		UserNote:          o.UserNote,
		OrderType:         o.OrderType,
		Payment:           nil,
		GHNOrderCode:      o.GHNOrderCode,
		OrderItems:        orderItems,
	}
	if pt != nil {
		resp.Payment = PaymentTransactionResponse{}.ToResponse(pt, nil)
	}
	return resp
}

func (OrderResponse) ToResponseList(source []model.Order, payments map[uuid.UUID]model.PaymentTransaction) []OrderResponse {
	if len(source) == 0 {
		return []OrderResponse{}
	}
	res := make([]OrderResponse, 0, len(source))
	for _, o := range source {
		var pt *model.PaymentTransaction
		if p, ok := payments[o.ID]; ok {
			pt = &p
		}
		res = append(res, *OrderResponse{}.ToResponse(&o, pt))
	}
	return res
}

func (OrderResponse) ToResponseListWithRevenue(source []model.Order, payments map[uuid.UUID]model.PaymentTransaction, companyRevenues map[uuid.UUID]float64, kolRevenues map[uuid.UUID]float64) []OrderResponse {
	if len(source) == 0 {
		return []OrderResponse{}
	}
	res := make([]OrderResponse, 0, len(source))
	for _, o := range source {
		var pt *model.PaymentTransaction
		if p, ok := payments[o.ID]; ok {
			pt = &p
		}
		resp := OrderResponse{}.ToResponse(&o, pt)

		// Set revenue fields if available
		if companyRev, ok := companyRevenues[o.ID]; ok {
			resp.CompanyRevenue = &companyRev
		}
		if kolRev, ok := kolRevenues[o.ID]; ok {
			resp.KOLRevenue = &kolRev
		}

		res = append(res, *resp)
	}
	return res
}

// ======================================== Order.Response (END) ========================================

// * OrderItemResponse ============================== OrderItem.Response (Start) ==============================
type OrderItemResponse struct {
	ID              uuid.UUID  `json:"id"`
	Quantity        int        `json:"quantity"`
	Subtotal        float64    `json:"subtotal"`
	UnitPrice       float64    `json:"unit_price"`
	Capacity        *float64   `json:"capacity"`
	CapacityUnit    string     `json:"capacity_unit"`
	ContainerType   string     `json:"container_type"`
	DispenserType   string     `json:"dispenser_type"`
	Uses            *string    `json:"uses"`
	ManufactureDate *time.Time `json:"manufacturing_date"`
	ExpiryDate      *time.Time `json:"expiry_date"`
	Instructions    *string    `json:"instructions"`
	Weight          int        `json:"weight"`
	Height          int        `json:"height"`
	Length          int        `json:"length"`
	Width           int        `json:"width"`
	IsReviewed      bool       `json:"is_reviewed"`
	BrandPlaceHolder *string     `json:"brand_place_holder"`

	//product fields
	ProductName       string                  `json:"product_name"`
	Description       *string                 `json:"description"`
	Type              string                  `json:"product_type"`
	LimitedProperties *OrderLimitedProperties `json:"limited_properties"`

	AttributesDescription *datatypes.JSON `json:"attributes_description" swaggerignore:"true"` //JSON

	//Relationships
	Review     OrderItemReview            `json:"review"`
	ItemImages []OrderItemImage           `json:"images"`
	Brand      *OrderItemBrandResponse    `json:"brand"`
	Category   *OrderItemCategoryResponse `json:"category"`
}

// OrderItemBrandResponse ============================== OrderItem.Brand.Response ==============================
type OrderItemBrandResponse struct {
	ID                  uuid.UUID `json:"id"`
	Name                string    `json:"name"`
	Description         *string   `json:"description,omitempty"`
	ContactEmail        *string   `json:"contact_email,omitempty"`
	ContactPhone        *string   `json:"contact_phone,omitempty"`
	Address             *string   `json:"address,omitempty"`
	Website             *string   `json:"website,omitempty"`
	LogoURL             *string   `json:"logo_url,omitempty"`
	TaxNumber           *string   `json:"tax_number,omitempty"`
	RepresentativeName  *string   `json:"representative_name,omitempty"`
	RepresentativeRole  *string   `json:"representative_role,omitempty"`
	RepresentativeEmail *string   `json:"representative_email,omitempty"`
}

func (OrderItemBrandResponse) ToResponse(brand *model.Brand) *OrderItemBrandResponse {
	return &OrderItemBrandResponse{
		ID:                  brand.ID,
		Name:                brand.Name,
		Description:         brand.Description,
		ContactEmail:        &brand.ContactEmail,
		ContactPhone:        &brand.ContactPhone,
		Address:             brand.Address,
		Website:             brand.Website,
		LogoURL:             brand.LogoURL,
		TaxNumber:           brand.TaxNumber,
		RepresentativeName:  brand.RepresentativeName,
		RepresentativeRole:  brand.RepresentativeRole,
		RepresentativeEmail: brand.RepresentativeEmail,
	}
}

// OrderItemCategoryResponse ============================== OrderItem.Category.Response ==============================
type OrderItemCategoryResponse struct {
	ID              uuid.UUID                   `json:"id"`
	Name            string                      `json:"name"`
	Description     *string                     `json:"description"`
	ParentCategory  *OrderItemCategoryResponse  `json:"parent_category"`
	ChildCategories []OrderItemCategoryResponse `json:"child_categories"`
}

func (OrderItemCategoryResponse) ToResponse(category *model.ProductCategory) *OrderItemCategoryResponse {
	return &OrderItemCategoryResponse{
		ID:          category.ID,
		Name:        category.Name,
		Description: category.Description,
		ParentCategory: func() *OrderItemCategoryResponse {
			if category.ParentCategory != nil {
				return OrderItemCategoryResponse{}.ToResponse(category.ParentCategory)
			}
			return nil
		}(),
		ChildCategories: func() []OrderItemCategoryResponse {
			if len(category.ChildCategories) == 0 {
				return []OrderItemCategoryResponse{}
			}
			res := make([]OrderItemCategoryResponse, 0, len(category.ChildCategories))
			for _, cc := range category.ChildCategories {
				res = append(res, *OrderItemCategoryResponse{}.ToResponse(&cc))
			}
			return res
		}(),
	}
}

func (OrderItemResponse) ToResponse(oi *model.OrderItem) *OrderItemResponse {
	if oi == nil {
		return nil
	}

	// safe derefs for pointer string fields
	var capacityUnit string
	if oi.CapacityUnit != nil {
		capacityUnit = *oi.CapacityUnit
	}
	var containerType string
	if oi.ContainerType != nil {
		containerType = *oi.ContainerType
	}
	var dispenserType string
	if oi.DispenserType != nil {
		dispenserType = *oi.DispenserType
	}

	// Limited properties only if Variant and Product are present
	var limitedProps *OrderLimitedProperties
	if oi.Variant.ID != uuid.Nil && oi.Variant.Product != nil {
		limitedProps = OrderLimitedProperties{}.ToResponse(oi.Variant.Product.Limited)
	}

	// Images only if Variant present
	var itemImages []OrderItemImage
	if oi.Variant.ID != uuid.Nil {
		itemImages = OrderItemImage{}.ToResponseList(oi.Variant.Images)
	}

	// Brand and Category only if present
	var brandResp *OrderItemBrandResponse
	if oi.Brand != nil {
		brandResp = OrderItemBrandResponse{}.ToResponse(oi.Brand)
	}
	var categoryResp *OrderItemCategoryResponse
	if oi.Category != nil {
		categoryResp = OrderItemCategoryResponse{}.ToResponse(oi.Category)
	}
	var reviewResp OrderItemReview
	if oi.ProductReview != nil {
		reviewResp = *OrderItemReview{}.ToResponse(oi.ProductReview)
	}

	return &OrderItemResponse{
		ID:                    oi.ID,
		Quantity:              oi.Quantity,
		Subtotal:              oi.Subtotal,
		UnitPrice:             oi.UnitPrice,
		Capacity:              oi.Capacity,
		CapacityUnit:          capacityUnit,
		ContainerType:         containerType,
		DispenserType:         dispenserType,
		Uses:                  oi.Uses,
		ManufactureDate:       oi.ManufactureDate,
		ExpiryDate:            oi.ExpiryDate,
		Instructions:          oi.Instructions,
		Weight:                oi.Weight,
		Height:                oi.Height,
		Length:                oi.Length,
		Width:                 oi.Width,
		IsReviewed:            oi.IsReviewed,
		Review:                reviewResp,
		ProductName:           oi.ProductName,
		Description:           oi.Description,
		Type:                  oi.Type,
		LimitedProperties:     limitedProps,
		AttributesDescription: oi.AttributesDescription,
		ItemImages:            itemImages,
		Brand:                 brandResp,
		Category:              categoryResp,
		BrandPlaceHolder:      oi.Variant.Product.BrandPlaceHolder,
	}
}

func (OrderItemResponse) ToResponseList(source []model.OrderItem) []OrderItemResponse {
	if len(source) == 0 {
		return []OrderItemResponse{}
	}
	res := make([]OrderItemResponse, 0, len(source))
	for _, oi := range source {
		res = append(res, *OrderItemResponse{}.ToResponse(&oi))
	}
	return res
}

// ======================================== OrderItem.OrderLimitedProperties (START) ========================================
type OrderLimitedProperties struct {
	PremiereDate          time.Time `json:"premiere_date,omitempty"`
	AvailabilityStartDate time.Time `json:"availability_start_date,omitempty"`
	AvailabilityEndDate   time.Time `json:"availability_end_date,omitempty"`
}

func (OrderLimitedProperties) ToResponse(lp *model.LimitedProduct) *OrderLimitedProperties {
	if lp == nil {
		return nil
	}

	return &OrderLimitedProperties{
		PremiereDate:          lp.PremiereDate,
		AvailabilityStartDate: lp.AvailabilityStartDate,
		AvailabilityEndDate:   lp.AvailabilityEndDate,
	}
}

// ======================================== OrderItem.OrderLimitedProperties (END) ========================================

// ======================================== OrderItem.OrderItemImage (START) ========================================
type OrderItemImage struct {
	ImageURL  string `json:"image_url"`
	AltText   string `json:"alt_text"`
	IsPrimary bool   `json:"is_primary"`
}

func (OrderItemImage) ToResponse(image *model.VariantImage) *OrderItemImage {
	return &OrderItemImage{
		ImageURL:  image.ImageURL,
		AltText:   *image.AltText,
		IsPrimary: image.IsPrimary,
	}
}

func (OrderItemImage) ToResponseList(source []model.VariantImage) []OrderItemImage {
	if len(source) == 0 {
		return []OrderItemImage{}
	}
	res := make([]OrderItemImage, 0, len(source))
	for _, img := range source {
		res = append(res, *OrderItemImage{}.ToResponse(&img))
	}
	return res
}

// ======================================== OrderItem.OrderItemImage (END) ========================================

// ======================================== OrderItem.Review (START) ========================================
type OrderItemReview struct {
	ID          uuid.UUID `json:"id"`
	RatingStars int       `json:"rating_stars"`
	Comment     *string   `json:"comment"`
	AssetsURL   *string   `json:"assets_url"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (OrderItemReview) ToResponse(review *model.ProductReview) *OrderItemReview {
	if review == nil {
		return nil
	}
	return &OrderItemReview{
		ID:          review.ID,
		RatingStars: review.RatingStars,
		Comment:     review.Comment,
		AssetsURL:   review.AssetsURL,
		CreatedAt:   review.CreatedAt,
		UpdatedAt:   review.UpdatedAt,
	}
}

// ======================================== OrderItem.Review (END) ========================================

//* ============================== OrderItem.Response (END) ==============================

type PriceBreakdown struct {
	ItemID            uuid.UUID `json:"item_id"`
	CompanyPercentage int       `json:"company_percentage"`
	KOLPercentage     int       `json:"kol_percentage"`
	CompanyAmount     float64   `json:"company_amount"`
	KOLAmount         float64   `json:"kol_amount"`
}
