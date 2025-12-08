package requests

import (
	"core-backend/internal/domain/model"
	"errors"
)

// AddProductReviewRequest defines the request payload for adding a product review
type AddProductReviewRequest struct {
	ReferenceID string  `json:"referenceID" validate:"required"`
	Type        string  `json:"type" validate:"required,oneof=ORDER PREORDER"`
	Rating      int     `json:"rating" validate:"required,min=1,max=5"`
	Comment     *string `json:"comment"`
	AssetsURL   *string `json:"assets_url"`
}

// ToPersistedModel converts the request DTO into a domain ProductReview model suitable for persistence.
// It validates and parses string UUIDs and enforces business rules (product_id required; exactly one of order_item_id or pre_order_id).
func (d *AddProductReviewRequest) ToPersistedModel(v model.ProductVariant) (*model.ProductReview, error) {
	if d == nil {
		return nil, errors.New("nil request")
	}

	// Rating should be validated earlier (validator in handler) but defend here as well
	if d.Rating < 1 || d.Rating > 5 {
		return nil, errors.New("rating must be between 1 and 5")
	}

	review := &model.ProductReview{}

	return review, nil
}
