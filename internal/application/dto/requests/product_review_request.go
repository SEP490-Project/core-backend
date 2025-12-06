package requests

import (
	"errors"
	"fmt"
	"strings"

	"core-backend/internal/domain/model"

	"github.com/google/uuid"
)

// AddProductReviewRequest defines the request payload for adding a product review
type AddProductReviewRequest struct {
	ProductID   *string `json:"productID"`
	VariantID   *string `json:"variant_id"`
	OrderItemID *string `json:"order_item_id"`
	PreOrderID  *string `json:"pre_order_id"`
	Rating      int     `json:"rating" validate:"required,min=1,max=5"`
	Comment     *string `json:"comment"`
	AssetsURL   *string `json:"assets_url"`
}

// ToPersistedModel converts the request DTO into a domain ProductReview model suitable for persistence.
// It validates and parses string UUIDs and enforces business rules (product_id required; exactly one of order_item_id or pre_order_id).
func (d *AddProductReviewRequest) ToPersistedModel() (*model.ProductReview, error) {
	if d == nil {
		return nil, errors.New("nil request")
	}

	// product_id is required
	if d.ProductID == nil || strings.TrimSpace(*d.ProductID) == "" {
		return nil, errors.New("product_id is required")
	}
	productID, err := uuid.Parse(strings.TrimSpace(*d.ProductID))
	if err != nil {
		return nil, fmt.Errorf("invalid product_id: %w", err)
	}

	// optional variant_id
	var variantID *uuid.UUID
	if d.VariantID != nil && strings.TrimSpace(*d.VariantID) != "" {
		v, err := uuid.Parse(strings.TrimSpace(*d.VariantID))
		if err != nil {
			return nil, fmt.Errorf("invalid variant_id: %w", err)
		}
		variantID = &v
	}

	// optional order_item_id
	var orderItemID *uuid.UUID
	if d.OrderItemID != nil && strings.TrimSpace(*d.OrderItemID) != "" {
		o, err := uuid.Parse(strings.TrimSpace(*d.OrderItemID))
		if err != nil {
			return nil, fmt.Errorf("invalid order_item_id: %w", err)
		}
		orderItemID = &o
	}

	// optional pre_order_id
	var preOrderID *uuid.UUID
	if d.PreOrderID != nil && strings.TrimSpace(*d.PreOrderID) != "" {
		p, err := uuid.Parse(strings.TrimSpace(*d.PreOrderID))
		if err != nil {
			return nil, fmt.Errorf("invalid pre_order_id: %w", err)
		}
		preOrderID = &p
	}

	// Business rule: exactly one of order_item_id or pre_order_id must be provided
	if (orderItemID == nil && preOrderID == nil) || (orderItemID != nil && preOrderID != nil) {
		return nil, errors.New("either order_item_id or pre_order_id must be provided (but not both)")
	}

	// Rating should be validated earlier (validator in handler) but defend here as well
	if d.Rating < 1 || d.Rating > 5 {
		return nil, errors.New("rating must be between 1 and 5")
	}

	review := &model.ProductReview{
		ProductID:   productID,
		VariantID:   variantID,
		OrderItemID: orderItemID,
		PreOrderID:  preOrderID,
		RatingStars: d.Rating,
		Comment:     d.Comment,
		AssetsURL:   d.AssetsURL,
	}

	return review, nil
}
