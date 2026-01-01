package responses

import (
	"core-backend/internal/domain/model"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

type PreOrderResponse struct {
	PreOrdersProps
	PaymentTx PaymentTransactionResponse
}

type PreOrdersProps struct {
	ID          uuid.UUID `json:"id"`
	UserID      uuid.UUID `json:"user_id"`
	VariantID   uuid.UUID `json:"variant_id"`
	Quantity    int       `json:"quantity"`
	UnitPrice   float64   `json:"unit_price"`
	TotalAmount float64   `json:"total_amount"`
	Status      string    `json:"status"`

	// Bank Info
	BankAccount       string `json:"user_bank_account"`
	BankName          string `json:"user_bank_name"`
	BankAccountHolder string `json:"user_bank_account_holder"`

	// Shipping Info
	FullName      string `json:"full_name"`
	PhoneNumber   string `json:"phone_number"`
	Email         string `json:"email"`
	Street        string `json:"street"`
	AddressLine2  string `json:"address_line2"`
	City          string `json:"city"`
	GhnProvinceID int    `json:"ghn_province_id"`
	GhnDistrictID int    `json:"ghn_district_id"`
	GhnWardCode   string `json:"ghn_ward_code"`
	ProvinceName  string `json:"province_name"`
	DistrictName  string `json:"district_name"`
	WardName      string `json:"ward_name"`

	IsSelfPickedUp    bool    `json:"is_self_picked_up"`
	ConfirmationImage *string `json:"confirmation_image,omitempty"`
	UserResource      *string `json:"user_resource,omitempty"`
	StaffResource     *string `json:"staff_resource,omitempty"`

	ActionNotes *model.PreOrderActionNotes `json:"action_notes,omitempty"`
	UserNote    *string                    `json:"user_note,omitempty"`

	// Variant Info
	Capacity              *float64        `json:"capacity"`
	CapacityUnit          *string         `json:"capacity_unit"`
	ContainerType         *string         `json:"container_type"`
	DispenserType         *string         `json:"dispenser_type"`
	Uses                  *string         `json:"uses"`
	ManufactureDate       *time.Time      `json:"manufacturing_date"`
	ExpiryDate            *time.Time      `json:"expiry_date"`
	Instructions          *string         `json:"instructions"`
	AttributesDescription *datatypes.JSON `json:"attributes_description" swaggerignore:"true"`

	Weight     int  `json:"weight"`
	Height     int  `json:"height"`
	Length     int  `json:"length"`
	Width      int  `json:"width"`
	IsReviewed bool `json:"is_reviewed"`

	// Product Info
	ProductName       string                  `json:"product_name"`
	Description       *string                 `json:"description"`
	Type              string                  `json:"product_type"`
	LimitedProperties *OrderLimitedProperties `json:"limited_properties"`

	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"` // convert gorm.DeletedAt to *time.Time

	ItemImages []OrderItemImage           `json:"images"`
	Brand      *OrderItemBrandResponse    `json:"brand"`
	Category   *OrderItemCategoryResponse `json:"category"`
}

func (p PreOrderResponse) ToPreOrderResponse(po model.PreOrder, pm *model.PaymentTransaction) PreOrderResponse {
	var brandResp *OrderItemBrandResponse
	var categoryResp *OrderItemCategoryResponse
	var itemImages []OrderItemImage
	var limitedProps *OrderLimitedProperties

	if po.Brand != nil {
		brandResp = OrderItemBrandResponse{}.ToResponse(po.Brand)
	}
	if po.Category != nil {
		categoryResp = OrderItemCategoryResponse{}.ToResponse(po.Category)
	}
	if po.ProductVariant.ID != uuid.Nil && len(po.ProductVariant.Images) > 0 {
		itemImages = OrderItemImage{}.ToResponseList(po.ProductVariant.Images)
	}

	var pmResp PaymentTransactionResponse
	if pm != nil {
		pmResp = *PaymentTransactionResponse{}.ToResponse(pm, nil)
	}
	if po.ProductVariant.Product.Limited != nil {
		limitedProps = OrderLimitedProperties{}.ToResponse(po.ProductVariant.Product.Limited)
	}

	return PreOrderResponse{
		PreOrdersProps: PreOrdersProps{
			ID:                    po.ID,
			UserID:                po.UserID,
			VariantID:             po.VariantID,
			Quantity:              po.Quantity,
			UnitPrice:             po.UnitPrice,
			TotalAmount:           po.TotalAmount,
			Status:                string(po.Status),
			BankAccount:           po.BankAccount,
			BankName:              po.BankName,
			BankAccountHolder:     po.BankAccountHolder,
			FullName:              po.FullName,
			PhoneNumber:           po.PhoneNumber,
			Email:                 po.Email,
			Street:                po.Street,
			AddressLine2:          po.AddressLine2,
			City:                  po.City,
			GhnProvinceID:         po.GhnProvinceID,
			GhnDistrictID:         po.GhnDistrictID,
			GhnWardCode:           po.GhnWardCode,
			ProvinceName:          po.ProvinceName,
			DistrictName:          po.DistrictName,
			WardName:              po.WardName,
			IsSelfPickedUp:        po.IsSelfPickedUp,
			ConfirmationImage:     po.ConfirmationImage,
			UserResource:          po.UserResource,
			StaffResource:         po.StaffResource,
			ActionNotes:           po.ActionNotes,
			UserNote:              po.UserNote,
			Capacity:              po.Capacity,
			CapacityUnit:          po.CapacityUnit,
			ContainerType:         po.ContainerType,
			DispenserType:         po.DispenserType,
			Uses:                  po.Uses,
			ManufactureDate:       po.ManufactureDate,
			ExpiryDate:            po.ExpiryDate,
			Instructions:          po.Instructions,
			AttributesDescription: po.AttributesDescription,
			Weight:                po.Weight,
			Height:                po.Height,
			Length:                po.Length,
			Width:                 po.Width,
			IsReviewed:            po.IsReviewed,
			ProductName:           po.ProductName,
			Description:           po.Description,
			Type:                  po.Type,
			LimitedProperties:     limitedProps,
			CreatedAt:             po.CreatedAt,
			UpdatedAt:             po.UpdatedAt,
			DeletedAt: func() *time.Time {
				if po.DeletedAt.Valid {
					return &po.DeletedAt.Time
				} else {
					return nil
				}
			}(),
			ItemImages: itemImages,
			Brand:      brandResp,
			Category:   categoryResp,
		},
		PaymentTx: pmResp,
	}
}
