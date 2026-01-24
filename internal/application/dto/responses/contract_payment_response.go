package responses

import (
	"core-backend/config"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
	"core-backend/pkg/utils"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// region: ============== Contract Payment Response ==============
var (
	endStatuses = []enum.ContractPaymentStatus{enum.ContractPaymentStatusPaid, enum.ContractPaymentStatusTerminated}
)

type ContractPaymentResponse struct {
	ID                    string                     `json:"id" example:"b3e1f9d2-8c4e-4f5a-9f1e-2d3c4b5a6e7f"`
	ContractID            string                     `json:"contract_id" example:"a1b2c3d4-e5f6-7a8b-9c0d-e1f2a3b4c5d6"`
	ContractTitle         string                     `json:"contract_title" example:"Website Development Contract"`
	ContractNumber        string                     `json:"contract_number" example:"WD-2024-001"`
	ContractType          enum.ContractType          `json:"contract_type" example:"ADVERTISING"`
	BrandID               string                     `json:"brand_id" example:"d4c3b2a1-0f9e-8d7c-6b5a-4c3b2a1f0e9d"`
	BrandName             string                     `json:"brand_name" example:"Tech Solutions Inc."`
	InstallmentPercentage float64                    `json:"installment_percentage" example:"50.0"`
	Amount                float64                    `json:"amount" example:"5000.00"`
	BaseAmount            float64                    `json:"base_amount" example:"4500.00"`
	PerformanceAmount     float64                    `json:"performance_amount" example:"500.00"`
	Breakdown             any                        `json:"breakdown,omitempty"`
	Status                enum.ContractPaymentStatus `json:"status" example:"PENDING"`
	DueDate               string                     `json:"due_date" example:"2024-07-15T00:00:00Z"`
	PaymentMethod         string                     `json:"payment_method" example:"BANK_TRANSFER"`
	Note                  *string                    `json:"note" example:"First installment payment"`
	IsDeposit             bool                       `json:"is_deposit" example:"true"`
	PayNow                bool                       `json:"pay_now" example:"false"`
	PaidAt                *string                    `json:"paid_at" example:"2006-01-02T15:04:05Z07:00"`
	CreatedAt             string                     `json:"created_at" example:"2006-01-02T15:04:05Z07:00"`
	UpdatedAt             string                     `json:"updated_at" example:"2006-01-02T15:04:05Z07:00"`

	// CO_PRODUCING Refund Fields
	RefundAmount       *float64 `json:"refund_amount,omitempty" example:"150000.00"`
	RefundProofURL     *string  `json:"refund_proof_url,omitempty" example:"https://s3.example.com/refund-proof.pdf"`
	RefundProofNote    *string  `json:"refund_proof_note,omitempty" example:"Bank transfer completed"`
	RefundSubmittedAt  *string  `json:"refund_submitted_at,omitempty" example:"2024-07-15T10:30:00Z"`
	RefundSubmittedBy  *string  `json:"refund_submitted_by,omitempty" example:"John Doe"`
	RefundReviewedAt   *string  `json:"refund_reviewed_at,omitempty" example:"2024-07-16T14:00:00Z"`
	RefundReviewedBy   *string  `json:"refund_reviewed_by,omitempty" example:"Brand Manager"`
	RefundRejectReason *string  `json:"refund_reject_reason,omitempty" example:"Proof image is unclear"`
	RefundAttempts     int      `json:"refund_attempts" example:"1"`

	// Brand Bank Info (for refund proof submission UI)
	BrandBankName          *string `json:"brand_bank_name,omitempty" example:"Vietcombank"`
	BrandBankAccountNumber *string `json:"brand_bank_account_number,omitempty" example:"1234567890"`
	BrandBankAccountHolder *string `json:"brand_bank_account_holder,omitempty" example:"NGUYEN VAN A"`

	// Computed Fields
	CanGeneratePaymentLink bool `json:"can_generate_payment_link" example:"true"`
}

// ToResponse converts a ContractPayment model to a ContractPaymentResponse
func (ContractPaymentResponse) ToResponse(m *model.ContractPayment) *ContractPaymentResponse {
	if m == nil {
		return nil
	}

	response := &ContractPaymentResponse{
		ID:                    m.ID.String(),
		ContractID:            m.ContractID.String(),
		InstallmentPercentage: m.InstallmentPercentage,
		Amount:                m.Amount,
		BaseAmount:            m.BaseAmount,
		PerformanceAmount:     m.PerformanceAmount,
		Breakdown:             utils.PtrOrNil(m.CalculationBreakdown),
		Status:                m.Status,
		DueDate:               utils.FormatLocalTime(&m.DueDate, utils.DateFormat),
		PaymentMethod:         m.PaymentMethod.String(),
		Note:                  m.Note,
		IsDeposit:             m.IsDeposit,
		CreatedAt:             utils.FormatLocalTime(&m.CreatedAt, utils.TimezoneFormat),
		UpdatedAt:             utils.FormatLocalTime(&m.UpdatedAt, utils.TimezoneFormat),
		RefundAttempts:        m.RefundAttempts,
	}

	if m.PaidAt != nil {
		paidAt := utils.FormatLocalTime(m.PaidAt, utils.TimezoneFormat)
		response.PaidAt = &paidAt
	}

	// Populate contract and brand info
	if m.ContractID != uuid.Nil && m.Contract != nil {
		response.ContractID = m.Contract.ID.String()
		response.ContractTitle = *m.Contract.Title
		response.ContractNumber = *m.Contract.ContractNumber
		response.ContractType = m.Contract.Type
		if m.Contract.Brand != nil {
			response.BrandID = m.Contract.Brand.ID.String()
			response.BrandName = m.Contract.Brand.Name
		}

		// Brand bank info (for refund proof submission UI)
		response.BrandBankName = m.Contract.BrandBankName
		response.BrandBankAccountNumber = m.Contract.BrandBankAccountNumber
		response.BrandBankAccountHolder = m.Contract.BrandBankAccountHolder
	}

	// Populate refund fields (only when in refund flow)
	if m.IsInRefundFlow() {
		refundAmount := m.RefundAmount
		response.RefundAmount = &refundAmount
		response.RefundProofURL = m.RefundProofURL
		response.RefundProofNote = m.RefundProofNote
		if m.RefundSubmittedAt != nil {
			formatted := utils.FormatLocalTime(m.RefundSubmittedAt, utils.TimezoneFormat)
			response.RefundSubmittedAt = &formatted
		}
		if m.RefundSubmitter != nil {
			response.RefundSubmittedBy = &m.RefundSubmitter.Username
		}
		if m.RefundReviewedAt != nil {
			formatted := utils.FormatLocalTime(m.RefundReviewedAt, utils.TimezoneFormat)
			response.RefundReviewedAt = &formatted
		}
		if m.RefundReviewer != nil {
			response.RefundReviewedBy = &m.RefundReviewer.Username
		}
		response.RefundRejectReason = m.RefundRejectReason
	}

	// Compute CanGeneratePaymentLink:
	// - Status must be PENDING
	// - Current time must be after due date
	now := time.Now()
	response.CanGeneratePaymentLink = m.Status == enum.ContractPaymentStatusPending && now.After(m.DueDate)
	// Compute PayNow: only available when payment is pending and within allowed overdue window
	allowedDays := config.GetAppConfig().AdminConfig.ContractPaymentAllowedOverdueDays
	response.PayNow = response.Status == enum.ContractPaymentStatusPending && isWithinAllowedOverdue(m.DueDate, allowedDays)

	return response
}

// ToResponseList converts a list of ContractPayment models to a list of ContractPaymentResponses
func (ContractPaymentResponse) ToResponseList(sources []model.ContractPayment, filter *requests.PaginationRequest) []ContractPaymentResponse {
	if len(sources) == 0 {
		return []ContractPaymentResponse{}
	}

	// 1. Setup Sort Configuration
	sortBy := "created_at"
	sortOrder := "desc"
	if filter != nil {
		if filter.SortBy != "" {
			sortBy = strings.ToLower(filter.SortBy)
		}
		if filter.SortOrder != "" {
			sortOrder = strings.ToLower(filter.SortOrder)
		}
	}

	// 2. Group by ContractID (ContractID -> []*model.ContractPayment)
	paymentMap := make(map[uuid.UUID][]*model.ContractPayment)
	for i := range sources {
		p := &sources[i]
		paymentMap[p.ContractID] = append(paymentMap[p.ContractID], p)
	}

	// 3. Parallel Internal List Sort
	var wg sync.WaitGroup

	// Convert map to slice of keys to iterate deterministically later
	uniqueContractIDs := make([]uuid.UUID, 0, len(paymentMap))
	for contractID, payments := range paymentMap {
		uniqueContractIDs = append(uniqueContractIDs, contractID)
		wg.Add(1)
		go func(pList []*model.ContractPayment) {
			defer wg.Done()
			// Sort Internal List: DueDate ASC, then by sortBy & sortOrder
			slices.SortFunc(pList, func(a, b *model.ContractPayment) int {
				// PAID and Terminated always go to the end
				aEnd := utils.ContainsSlice(endStatuses, a.Status)
				bEnd := utils.ContainsSlice(endStatuses, b.Status)

				if aEnd != bEnd {
					if aEnd {
						return 1
					} // a is PAID, move to end
					return -1 // b is PAID, move a to front
				}

				if diff := a.DueDate.Compare(b.DueDate); diff != 0 {
					return diff
				}

				res := utils.CompareByJSONTag(a, b, sortBy)
				if sortOrder == "desc" {
					return -res
				}
				return res
			})
		}(payments)
	}
	wg.Wait()

	// 4. Sort Groups based on Payment Fields
	slices.SortFunc(uniqueContractIDs, func(idA, idB uuid.UUID) int {
		// Compare the FIRST payment of each group (the "Next Due" payment) to decide which Contract comes first.
		pA := paymentMap[idA][0]
		pB := paymentMap[idB][0]

		res := utils.CompareByJSONTag(pA, pB, sortBy)
		if sortOrder == "desc" {
			return -res
		}
		return res
	})

	// 5. Parallel Mapping to Response
	// We preserve order by pre-allocating a slot for each contract group
	tempResults := make([][]ContractPaymentResponse, len(uniqueContractIDs))
	wg.Add(len(uniqueContractIDs))

	for i, cID := range uniqueContractIDs {
		// Capture loop variables
		idx := i
		contractID := cID

		go func() {
			defer wg.Done()
			payments := paymentMap[contractID]
			groupRes := make([]ContractPaymentResponse, 0, len(payments))
			mapper := ContractPaymentResponse{}

			index := 0
			for k, p := range payments {
				resp := mapper.ToResponse(p)
				if resp == nil {
					continue
				}
				// Apply PayNow logic: Only true for the first item in the sorted group
				if utils.ContainsSlice(endStatuses, resp.Status) {
					// End statuses are never payable now
					resp.PayNow = false
					index++
				} else {
					// Only the first non-ended payment in the group is eligible to be paid now.
					if k == index {
						allowedDays := config.GetAppConfig().AdminConfig.ContractPaymentAllowedOverdueDays
						resp.PayNow = resp.Status == enum.ContractPaymentStatusPending && isWithinAllowedOverdue(p.DueDate, allowedDays)
						if !resp.PayNow {
							if !resp.Status.IsRefundStatus() && !resp.Status.IsTerminalStatus() {
								resp.Status = enum.ContractPaymentStatusNotStarted
							}
						}
					} else {
						resp.PayNow = false
						if !resp.Status.IsRefundStatus() && !resp.Status.IsTerminalStatus() {
							resp.Status = enum.ContractPaymentStatusNotStarted
						}
					}
				}
				groupRes = append(groupRes, *resp)
			}
			tempResults[idx] = groupRes
		}()
	}

	wg.Wait()

	// 6. Flatten
	totalSize := 0
	for _, subList := range tempResults {
		totalSize += len(subList)
	}

	finalResponses := make([]ContractPaymentResponse, 0, totalSize)
	for _, subList := range tempResults {
		finalResponses = append(finalResponses, subList...)
	}

	return finalResponses
}

func (ContractPaymentResponse) ToSimpleResponseList(sources []model.ContractPayment) []ContractPaymentResponse {
	if len(sources) == 0 {
		return []ContractPaymentResponse{}
	}

	// var response []ContractPaymentResponse
	list := make([]ContractPaymentResponse, len(sources))
	for i, source := range sources {
		// list = append(list, *ContractPaymentResponse{}.ToResponse(&source))
		list[i] = *ContractPaymentResponse{}.ToResponse(&source)
	}
	return list
}

// endregion

// isWithinAllowedOverdue returns true if now is between due date (inclusive)
// and due date + allowedDays (inclusive) using local date comparison.
func isWithinAllowedOverdue(due time.Time, allowedDays int) bool {
	// if allowedDays < 0 {
	// 	return false
	// }
	// // Normalize to local date (midnight)
	// loc := time.Local
	// now := time.Now().In(loc)
	// dueLocal := due.In(loc)

	// nowDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	// dueDate := time.Date(dueLocal.Year(), dueLocal.Month(), dueLocal.Day(), 0, 0, 0, 0, loc)

	// endDate := dueDate.AddDate(0, 0, allowedDays)

	// return !nowDate.Before(dueDate) && !nowDate.After(endDate)
	return true
}

type ContractPaymentPaginationResponse PaginationResponse[ContractPaymentResponse]
