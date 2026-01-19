package responses

import (
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
	"core-backend/pkg/utils"
	"slices"
	"strings"
	"sync"

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
	CreatedAt             string                     `json:"created_at" example:"2006-01-02T15:04:05Z07:00"`
	UpdatedAt             string                     `json:"updated_at" example:"2006-01-02T15:04:05Z07:00"`
}

// ToResponse converts a ContractPayment model to a ContractPaymentResponse
func (ContractPaymentResponse) ToResponse(model *model.ContractPayment) *ContractPaymentResponse {
	if model == nil {
		return nil
	}

	response := &ContractPaymentResponse{
		ID:                    model.ID.String(),
		ContractID:            model.ContractID.String(),
		InstallmentPercentage: model.InstallmentPercentage,
		Amount:                model.Amount,
		BaseAmount:            model.BaseAmount,
		PerformanceAmount:     model.PerformanceAmount,
		Breakdown:             utils.PtrOrNil(model.CalculationBreakdown),
		Status:                model.Status,
		DueDate:               utils.FormatLocalTime(&model.DueDate, utils.DateFormat),
		PaymentMethod:         model.PaymentMethod.String(),
		Note:                  model.Note,
		IsDeposit:             model.IsDeposit,
		CreatedAt:             utils.FormatLocalTime(&model.CreatedAt, utils.TimezoneFormat),
		UpdatedAt:             utils.FormatLocalTime(&model.UpdatedAt, utils.TimezoneFormat),
	}
	if model.ContractID != uuid.Nil && model.Contract != nil {
		response.ContractID = model.Contract.ID.String()
		response.ContractTitle = *model.Contract.Title
		response.ContractNumber = *model.Contract.ContractNumber
		response.ContractType = model.Contract.Type
		if model.Contract.Brand != nil {
			response.BrandID = model.Contract.Brand.ID.String()
			response.BrandName = model.Contract.Brand.Name
		}
	}

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
					index++
				} else {
					// resp.PayNow = (k == index)
					if k == index {
						resp.PayNow = true
					} else {
						resp.Status = enum.ContractPaymentStatusNotStarted
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

// endregion

type ContractPaymentPaginationResponse PaginationResponse[ContractPaymentResponse]
