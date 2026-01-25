package responses

import (
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
	"core-backend/pkg/utils"

	"github.com/google/uuid"
)

type PaymentTransactionResponse struct {
	ID              uuid.UUID  `json:"id" example:"b3e1f9d2-8c4e-4f5a-9f1e-2d3c4b5a6e7f"`
	ReferenceID     string     `json:"reference_id" example:"a1b2c3d4-e5f6-7g8h-9i0j-k1l2m3n4o5p6"`
	ReferenceType   string     `json:"reference_type" example:"ORDER"`
	ReferenceInfo   any        `json:"reference_info,omitempty"`
	Amount          string     `json:"amount" example:"99.99"`
	Method          string     `json:"method" example:"CREDIT_CARD"`
	Status          string     `json:"status" example:"COMPLETED"`
	TransactionDate string     `json:"transaction_date" example:"2024-01-01T12:00:00Z"`
	GatewayRef      string     `json:"gateway_ref,omitempty" example:"PAYOS123456789"`
	GatewayID       string     `json:"gateway_id,omitempty" example:"GATEWAY987654321"`
	UpdatedAt       string     `json:"updated_at" example:"2024-01-02T12:00:00Z"`
	PayerID         *uuid.UUID `json:"payer_id,omitempty" example:"d4e5f6a7-b8c9-0a1b-2c3d-4e5f6a7b8c9d"`
	ReceivedByID    *uuid.UUID `json:"received_by_id,omitempty" example:"e5f6a7b8-c9d0-1a2b-3c4d-5e6f7a8b9c0d"`
}

// region: ========== Specific Reference Info Structures ==========

type PaymentTransactionReferenceOrder struct {
	ID         uuid.UUID                             `json:"id" example:"d4e5f6a7-b8c9-0a1b-2c3d-4e5f6a7b8c9d"`
	UserInfo   PaymentTransactionReferenceUserInfo   `json:"user_info"`
	BankInfo   PaymentTransactionReferenceBankInfo   `json:"bank_info"`
	OrderItems []PaymentTransactionRefereceOrderItem `json:"order_items"`
}

func (PaymentTransactionReferenceOrder) FromOrderModel(source *model.Order) *PaymentTransactionReferenceOrder {
	if source == nil {
		return nil
	}

	var reference = &PaymentTransactionReferenceOrder{ID: source.ID}

	for _, item := range source.OrderItems {
		reference.OrderItems = append(reference.OrderItems, PaymentTransactionRefereceOrderItem{
			ID:          item.ID,
			ProductName: item.ProductName,
			Quantity:    item.Quantity,
			UnitPrice:   item.UnitPrice,
			Subtotal:    item.Subtotal,
		})
	}
	reference.UserInfo = PaymentTransactionReferenceUserInfo{
		ID:          source.UserID,
		FullName:    source.FullName,
		PhoneNumber: source.PhoneNumber,
		Email:       source.Email,
	}
	reference.BankInfo = PaymentTransactionReferenceBankInfo{
		BankAccount:       source.BankAccount,
		BankName:          source.BankName,
		BankAccountHolder: source.BankAccountHolder,
	}
	return reference
}

type PaymentTransactionReferencePreOrder struct {
	ID                 uuid.UUID                           `json:"id" example:"f1e2d3c4-b5a6-7g8h-9i0j-k1l2m3n4o5p6"`
	UserInfo           PaymentTransactionReferenceUserInfo `json:"user_info"`
	BankInfo           PaymentTransactionReferenceBankInfo `json:"bank_info"`
	ProductVariantInfo struct {
		ID          uuid.UUID `json:"id" example:"a1b2c3d4-e5f6-7g8h-9i0j-k1l2m3n4o5p6"`
		ProductName string    `json:"product_name" example:"Example Product"`
		Quantity    int       `json:"quantity" example:"2"`
		UnitPrice   float64   `json:"unit_price" example:"49.99"`
		TotalAmount float64   `json:"total_amount" example:"99.98"`
	} `json:"product_variant_info"`
}

func (PaymentTransactionReferencePreOrder) FromPreOrderModel(source *model.PreOrder) *PaymentTransactionReferencePreOrder {
	if source == nil {
		return nil
	}
	var reference = &PaymentTransactionReferencePreOrder{ID: source.ID}
	reference.ProductVariantInfo.ID = source.VariantID
	reference.ProductVariantInfo.ProductName = source.ProductName
	reference.ProductVariantInfo.Quantity = source.Quantity
	reference.ProductVariantInfo.UnitPrice = source.UnitPrice
	reference.ProductVariantInfo.TotalAmount = source.TotalAmount

	reference.UserInfo = PaymentTransactionReferenceUserInfo{
		ID:          source.UserID,
		FullName:    source.FullName,
		PhoneNumber: source.PhoneNumber,
		Email:       source.Email,
	}
	reference.BankInfo = PaymentTransactionReferenceBankInfo{
		BankAccount:       source.BankAccount,
		BankName:          source.BankName,
		BankAccountHolder: source.BankAccountHolder,
	}

	return reference
}

type PaymentTransactionReferenceContractPayment struct {
	ID             uuid.UUID `json:"id" example:"c3d4e5f6-a7b8-9c0d-1e2f-3g4h5i6j7k8l"`
	ContractID     uuid.UUID `json:"contract_id" example:"h1i2j3k4-l5m6-n7o8-p9q0-r1s2t3u4v5w6"`
	ContractNumber string    `json:"contract_number" example:"CONTRACT-2024-0001"`
	IsDeposit      bool      `json:"is_deposit" example:"false"`
	BrandInfo      struct {
		ID                  uuid.UUID `json:"id" gorm:"column:id" example:"b2c3d4e5-f6a7-8b9c-0d1e-2f3g4h5i6j7k"`
		UserID              uuid.UUID `json:"user_id" gorm:"column:user_id" example:"b2c3d4e5-f6a7-8b9c-0d1e-2f3g4h5i6j7k"`
		Name                string    `json:"name" gorm:"column:name" example:"Acme Corp"`
		ContactEmail        string    `json:"contact_email" gorm:"column:contact_email" example:"johndoe@example.com"`
		ContactPhone        string    `json:"contact_phone" gorm:"column:contact_phone" example:"+1234567890"`
		RepresentativeName  *string   `json:"representative_name" gorm:"column:representative_name" example:"Jane Smith"`
		RepresentativeEmail *string   `json:"representative_email" gorm:"column:representative_email" example:"janesmith@example.com"`
		RepresentativePhone *string   `json:"representative_phone" gorm:"column:representative_phone" example:"+1234567890"`
	} `json:"brand_info" gorm:"type:jsonb;column:brand_info"`
	BankInfo PaymentTransactionReferenceBankInfo `json:"bank_info" gorm:"type:jsonb;column:bank_info"`
}

func (PaymentTransactionReferenceContractPayment) FromContractPaymentModel(source *model.ContractPayment) *PaymentTransactionReferenceContractPayment {
	if source == nil {
		return nil
	}
	var reference = &PaymentTransactionReferenceContractPayment{
		ID:         source.ID,
		ContractID: source.ContractID,
		IsDeposit:  source.IsDeposit,
	}
	if source.Contract != nil {
		reference.ContractNumber = utils.DerefPtr(source.Contract.ContractNumber, "N\\A")

		brand := source.Contract.Brand
		if brand != nil {
			reference.BrandInfo.ID = brand.ID
			reference.BrandInfo.UserID = utils.DerefPtr(brand.UserID, uuid.Nil)
			reference.BrandInfo.Name = brand.Name
			reference.BrandInfo.ContactEmail = brand.ContactEmail
			reference.BrandInfo.ContactPhone = brand.ContactPhone
			reference.BrandInfo.RepresentativeName = brand.RepresentativeName
			reference.BrandInfo.RepresentativeEmail = brand.RepresentativeEmail
			reference.BrandInfo.RepresentativePhone = brand.RepresentativePhone
		}
		reference.BankInfo = PaymentTransactionReferenceBankInfo{
			BankAccount:       utils.DerefPtr(source.Contract.BrandBankAccountNumber, "N\\A"),
			BankName:          utils.DerefPtr(source.Contract.BrandBankName, "N\\A"),
			BankAccountHolder: utils.DerefPtr(source.Contract.BrandBankAccountHolder, "N\\A"),
		}
	}

	return reference
}

type PaymentTransactionReferenceUserInfo struct {
	ID          uuid.UUID `json:"id" example:"e7f8g9h0-a1b2-c3d4-e5f6-7g8h9i0j1k2l"`
	FullName    string    `json:"full_name" example:"John Doe"`
	PhoneNumber string    `json:"phone_number" example:"+1234567890"`
	Email       string    `json:"email" example:"johndoe@example.com"`
}

type PaymentTransactionRefereceOrderItem struct {
	ID          uuid.UUID `json:"id" example:"a1b2c3d4-e5f6-7g8h-9i0j-k1l2m3n4o5p6"`
	ProductName string    `json:"product_name" example:"Example Product"`
	Quantity    int       `json:"quantity" example:"2"`
	UnitPrice   float64   `json:"unit_price" example:"24.99"`
	Subtotal    float64   `json:"subtotal" example:"49.99"`
}

type PaymentTransactionReferenceBankInfo struct {
	BankAccount       string `json:"bank_account" gorm:"column:bank_account" example:"1234567890"`
	BankName          string `json:"bank_name" gorm:"column:bank_name" example:"Bank of Examples"`
	BankAccountHolder string `json:"bank_account_holder" gorm:"column:bank_account_holder" example:"John Doe"`
}

type PaymentTransactionReferenceContractViolation struct {
	ID             uuid.UUID          `json:"id" example:"z1x2c3v4-b5n6-m7a8-s9d0-f1g2h3j4k5l6"`
	ContractID     uuid.UUID          `json:"contract_id" example:"m1n2b3v4-c5x6-z7a8-q9w0-e1r2t3y4u5i6"`
	ContractNumber string             `json:"contract_number,omitempty" example:"CONTRACT-2024-001"`
	ViolationType  enum.ViolationType `json:"violation_type" example:"KOL"`
	Reason         string             `json:"reason" example:"Violation of contract terms"`
	Amount         float64            `json:"amount" example:"100.00"` // PenaltyAmount or RefundAmount
	ResolvedAt     *string            `json:"resolved_at,omitempty" example:"2024-01-01T12:00:00Z"`

	// If KOL Violation
	ProofURL         *string `json:"proof_url,omitempty" example:"https://www.google.com"`
	ProofSubmittedAt *string `json:"proof_submitted_at,omitempty" example:"2024-01-02T12:00:00Z"`
	ProofSubmittedBy *struct {
		ID       uuid.UUID `json:"id" example:"a1b2c3d4-e5f6-7g8h-9i0j-k1l2m3n4o5p6"`
		FullName string    `json:"full_name" example:"John Doe"`
		Email    string    `json:"email" example:"johndoe@example.com"`
	} `json:"proof_submitted_by,omitempty"`
	ProofReviewNote *string `json:"proof_review_note,omitempty" example:"The proof is insufficient"`

	BrandInfo struct {
		ID                  uuid.UUID `json:"id" gorm:"column:id"`
		UserID              uuid.UUID `json:"user_id" gorm:"column:user_id"`
		Name                string    `json:"name" gorm:"column:name"`
		ContactEmail        string    `json:"contact_email" gorm:"column:contact_email"`
		ContactPhone        string    `json:"contact_phone" gorm:"column:contact_phone"`
		RepresentativeName  *string   `json:"representative_name" gorm:"column:representative_name"`
		RepresentativeEmail *string   `json:"representative_email" gorm:"column:representative_email"`
		RepresentativePhone *string   `json:"representative_phone" gorm:"column:representative_phone"`
	} `json:"brand_info" gorm:"type:jsonb;column:brand_info"`

	CampaignInfo *struct {
		ID   uuid.UUID `json:"id"`
		Name string    `json:"name"`
	} `json:"campaign_info,omitempty"`
}

func (PaymentTransactionReferenceContractViolation) FromContractViolationModel(source *model.ContractViolation) *PaymentTransactionReferenceContractViolation {
	if source == nil {
		return nil
	}

	response := &PaymentTransactionReferenceContractViolation{
		ID:            source.ID,
		ContractID:    source.ContractID,
		ViolationType: source.Type,
		Reason:        source.Reason,
	}

	if source.Contract != nil {
		if source.Contract.ContractNumber != nil {
			response.ContractNumber = *source.Contract.ContractNumber
		}
		if source.Contract.Brand != nil {
			brand := source.Contract.Brand
			response.BrandInfo.ID = brand.ID
			if brand.UserID != nil {
				response.BrandInfo.UserID = *brand.UserID
			}
			response.BrandInfo.Name = brand.Name
			response.BrandInfo.ContactEmail = brand.ContactEmail
			response.BrandInfo.ContactPhone = brand.ContactPhone
			response.BrandInfo.RepresentativeName = brand.RepresentativeName
			response.BrandInfo.RepresentativeEmail = brand.RepresentativeEmail
			response.BrandInfo.RepresentativePhone = brand.RepresentativePhone
		}
	}

	if source.Campaign != nil {
		response.CampaignInfo = &struct {
			ID   uuid.UUID `json:"id"`
			Name string    `json:"name"`
		}{
			ID:   source.Campaign.ID,
			Name: source.Campaign.Name,
		}
	}

	if source.ResolvedAt != nil {
		response.ResolvedAt = utils.PtrOrNil(utils.FormatLocalTime(source.ResolvedAt, utils.TimeFormat))
	}

	switch source.Type {
	case enum.ViolationTypeBrand:
		response.Amount = source.PenaltyAmount

	case enum.ViolationTypeKOL:
		response.Amount = source.RefundAmount
		response.ProofURL = source.ProofURL
		response.ProofReviewNote = source.ProofReviewNote
		if source.ProofSubmittedAt != nil {
			response.ProofSubmittedAt = utils.PtrOrNil(utils.FormatLocalTime(source.ProofSubmittedAt, utils.TimeFormat))
		}
		if source.ProofSubmittedBy != nil && source.ProofSubmitter != nil {
			response.ProofSubmittedBy = &struct {
				ID       uuid.UUID `json:"id" example:"a1b2c3d4-e5f6-7g8h-9i0j-k1l2m3n4o5p6"`
				FullName string    `json:"full_name" example:"John Doe"`
				Email    string    `json:"email" example:"johndoe@example.com"`
			}{
				ID:       source.ProofSubmitter.ID,
				FullName: source.ProofSubmitter.FullName,
				Email:    source.ProofSubmitter.Email,
			}
		}

	}

	return response
}

// endregion

func (PaymentTransactionResponse) ToResponse(source *model.PaymentTransaction, additionalInfo any) *PaymentTransactionResponse {
	if source == nil {
		return nil
	}

	return &PaymentTransactionResponse{
		ID:              source.ID,
		ReferenceID:     source.ReferenceID.String(),
		ReferenceType:   source.ReferenceType.String(),
		ReferenceInfo:   additionalInfo,
		Amount:          utils.ToString(source.Amount),
		Method:          source.Method,
		Status:          string(source.Status),
		TransactionDate: utils.FormatLocalTime(&source.TransactionDate, utils.TimeFormat),
		GatewayRef:      source.GatewayRef,
		GatewayID:       source.GatewayID,
		UpdatedAt:       utils.FormatLocalTime(&source.UpdatedAt, utils.TimeFormat),
		PayerID:         source.PayerID,
		ReceivedByID:    source.ReceivedByID,
	}
}

func (PaymentTransactionResponse) ToResponseList(source []model.PaymentTransaction) []PaymentTransactionResponse {
	if len(source) == 0 {
		return []PaymentTransactionResponse{}
	}

	responses := make([]PaymentTransactionResponse, len(source))
	for i, v := range source {
		responses[i] = *PaymentTransactionResponse{}.ToResponse(&v, nil)
	}
	return responses
}

// PaymentTransactionPaginationResponse represents a paginated response for PaymentTransactionResponse
// Only used for Swaggo swagger docs generation
type PaymentTransactionPaginationResponse PaginationResponse[PaymentTransactionResponse]
