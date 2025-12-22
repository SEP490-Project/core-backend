package service

import (
	"context"
	"core-backend/config"
	"core-backend/internal/application/dto/dtos"
	"core-backend/internal/application/dto/requests"
	"core-backend/internal/application/dto/responses"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/application/interfaces/iservice"
	"core-backend/internal/application/service/helper"
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
	gormrepository "core-backend/internal/infrastructure/gorm_repository"
	"core-backend/pkg/utils"
	"encoding/json"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type contractPaymentService struct {
	contractPaymentRepo    irepository.GenericRepository[model.ContractPayment]
	contractRepo           irepository.GenericRepository[model.Contract]
	paymentCalculationRepo irepository.ContractPaymentCalculationRepository
	config                 *config.AdminConfig
}

// GetContractPaymentByID implements iservice.ContractPaymentService.
// For AFFILIATE and CO_PRODUCING contracts, this method automatically
// recalculates the payment amount if the payment is for the current period.
func (c *contractPaymentService) GetContractPaymentByID(ctx context.Context, contractPaymentID uuid.UUID) (*responses.ContractPaymentResponse, error) {
	zap.L().Info("ContractPaymentService - GetContractPaymentByID called",
		zap.String("contractPaymentID", contractPaymentID.String()))

	contractPayment, err := c.contractPaymentRepo.GetByID(ctx, contractPaymentID, []string{"Contract", "Contract.Brand"})
	if err != nil {
		zap.L().Error("Failed to get contract payment by ID", zap.Error(err))
		return nil, err
	}

	// Trigger recalculation if applicable (AFFILIATE or CO_PRODUCING, current period, not paid)
	if err := c.recalculateIfNeeded(ctx, contractPayment); err != nil {
		zap.L().Warn("Failed to recalculate payment",
			zap.Error(err),
			zap.String("payment_id", contractPaymentID.String()))
		// Don't fail the request, just log and return current data
	}

	return responses.ContractPaymentResponse{}.ToResponse(contractPayment), nil
}

// CreatePaymentLinkFromContractPayment implements iservice.ContractPaymentService.
func (c *contractPaymentService) CreatePaymentLinkFromContractPayment(
	ctx context.Context,
	uow irepository.UnitOfWork,
	request *requests.GenerateContractPaymentLinkRequest,
	paymentTransactionService iservice.PaymentTransactionService,
) (*responses.PayOSLinkResponse, error) {
	zap.L().Info("Creating payment link from contract payment",
		zap.Any("request", request))

	// 1. Fetch contract payment with contract and brand details
	contractPayment, err := uow.ContractPayments().GetByID(ctx, request.ContractPaymentID, []string{"Contract", "Contract.Brand"})
	if err != nil {
		zap.L().Error("Failed to get contract payment", zap.Error(err))
		return nil, fmt.Errorf("failed to get contract payment: %w", err)
	}

	if contractPayment == nil {
		return nil, fmt.Errorf("contract payment not found")
	}

	// 2. Validate payment status - only PENDING payments can generate links
	if contractPayment.Status != enum.ContractPaymentStatusPending {
		return nil, fmt.Errorf("payment link can only be generated for pending payments, current status: %s", contractPayment.Status)
	}

	// 3. For AFFILIATE/CO_PRODUCING contracts, lock the calculated amount
	//    This ensures new clicks/revenue go to the next payment period
	if contractPayment.Contract.Type == enum.ContractTypeAffiliate ||
		contractPayment.Contract.Type == enum.ContractTypeCoProduce {
		if err = c.LockPaymentAmount(ctx, contractPayment); err != nil {
			zap.L().Error("Failed to lock payment amount", zap.Error(err))
			return nil, fmt.Errorf("failed to lock payment amount: %w", err)
		}
	}

	// 4. Build payment request using the (potentially locked) amount
	contractNumber := "Unknown"
	if contractPayment.Contract.ContractNumber != nil {
		contractNumber = *contractPayment.Contract.ContractNumber
	}

	// Use locked amount if available, otherwise use the regular amount
	paymentAmount := contractPayment.Amount
	if contractPayment.LockedAmount != nil {
		paymentAmount = *contractPayment.LockedAmount
	}

	paymentReq := &requests.PaymentRequest{
		ReferenceID:   contractPayment.ID,
		ReferenceType: enum.PaymentTransactionReferenceTypeContractPayment,
		PayerID:       contractPayment.Contract.Brand.UserID,
		Amount:        int64(paymentAmount),
		Description:   fmt.Sprintf("Payment for Contract %s - Installment %.0f%%", contractNumber, contractPayment.InstallmentPercentage),
		ReturnURL:     request.ReturnURL,
		CancelURL:     request.CancelURL,
	}

	// Add buyer information from contract brand if available
	if contractPayment.Contract != nil && contractPayment.Contract.Brand != nil {
		brand := contractPayment.Contract.Brand
		paymentReq.BuyerName = brand.Name
		paymentReq.BuyerEmail = brand.ContactEmail
		paymentReq.BuyerPhone = brand.ContactPhone
	}

	// Add payment item
	paymentReq.Items = []requests.PaymentItemRequest{
		{
			Name:     fmt.Sprintf("Contract Payment - %s", contractNumber),
			Quantity: 1,
			Price:    int64(contractPayment.Amount),
		},
	}

	// 4. Generate payment link using PaymentTransactionService
	payosResp, err := paymentTransactionService.GeneratePaymentLink(ctx, uow, paymentReq)
	if err != nil {
		zap.L().Error("Failed to generate payment link", zap.Error(err))
		return nil, fmt.Errorf("failed to generate payment link: %w", err)
	}

	zap.L().Info("Payment link created successfully for contract payment",
		zap.String("checkout_url", payosResp.CheckoutURL))

	return payosResp, nil
}

// GetContractPaymentsByFilter implements iservice.ContractPaymentService.
// For AFFILIATE and CO_PRODUCING contracts, this method automatically
// recalculates payment amounts for current period payments using parallel processing.
func (c *contractPaymentService) GetContractPaymentsByFilter(ctx context.Context, filter *requests.ContractPaymentFilterRequest) (*[]responses.ContractPaymentResponse, int64, error) {
	zap.L().Info("ContractPaymentService - GetContractPaymentsByFilter called", zap.Any("filter", filter))

	filterQuery := func(db *gorm.DB) *gorm.DB {
		if filter.BrandID != nil {
			db = db.Joins("JOIN contracts c ON c.id = contract_payments.contract_id").
				Where("c.brand_id = ?", *filter.BrandID)
		}
		if filter.BrandUserID != nil {
			db = db.Joins("JOIN contracts c ON c.id = contract_payments.contract_id").
				Joins("JOIN brands b ON b.id = c.brand_id").
				Where("b.user_id = ?", *filter.BrandUserID)
		}
		if filter.ContractID != nil {
			db = db.Where("contract_payments.contract_id = ?", *filter.ContractID)
		}
		if filter.Status != nil {
			db = db.Where("contract_payments.status = ?", *filter.Status)
		}
		if filter.DueDateFrom != nil {
			db = db.Where("contract_payments.due_date >= ?", *filter.DueDateFrom)
		}
		if filter.DueDateTo != nil {
			db = db.Where("contract_payments.due_date <= ?", *filter.DueDateTo)
		}

		db = db.Order(helper.ConvertToSortString(filter.PaginationRequest))
		return db
	}
	payments, total, err := c.contractPaymentRepo.GetAll(ctx, filterQuery, []string{"Contract", "Contract.Brand"}, filter.Limit, filter.Page)
	if err != nil {
		zap.L().Error("Failed to get contract payments by filter", zap.Error(err))
		return nil, 0, err
	}

	// Build list of recalculation tasks for applicable payments only
	var tasks []func(ctx context.Context) error
	var mu sync.Mutex

	for i := range payments {
		payment := &payments[i] // Capture pointer for closure

		// Pre-filter: only add task if payment might need recalculation
		if c.shouldRecalculate(payment) {
			tasks = append(tasks, func(ctx context.Context) error {
				if err := c.recalculateIfNeeded(ctx, payment); err != nil {
					mu.Lock()
					zap.L().Warn("Failed to recalculate payment",
						zap.Error(err),
						zap.String("payment_id", payment.ID.String()))
					mu.Unlock()
				}
				return nil // Don't fail the entire request for individual calculation errors
			})
		}
	}

	// Execute recalculations in parallel with concurrency limit (max 5 workers)
	if len(tasks) > 0 {
		if err := utils.RunParallel(ctx, 5, tasks...); err != nil {
			zap.L().Warn("Some payment recalculations failed", zap.Error(err))
			// Don't fail the request, continue with whatever data we have
		}
	}

	responsesList := responses.ContractPaymentResponse{}.ToResponseList(payments, &filter.PaginationRequest)
	return &responsesList, total, nil
}

// CreateContractPaymentsFromContract implements iservice.ContractPaymentService.
func (c *contractPaymentService) CreateContractPaymentsFromContract(
	ctx context.Context, userID uuid.UUID, contractID uuid.UUID, uow irepository.UnitOfWork,
) error {
	zap.L().Info("Creating contract payments from contract",
		zap.String("contract_id", contractID.String()))

	contractRepo := uow.Contracts()
	contractPaymentRepo := uow.ContractPayments()
	minimumDayBeforeDueDate := c.config.MinimumDayBeforeContractPaymentDue

	contract, err := contractRepo.GetByID(ctx, contractID, []string{"Brand"})
	if err != nil {
		zap.L().Error("Failed to fetch contract", zap.Error(err))
		return err
	} else if contract == nil {
		zap.L().Warn("Contract not found", zap.String("contract_id", contractID.String()))
		return fmt.Errorf("contract with ID %s not found", contractID)
	}

	contractPaymentsSlice, err := c.processPaymentDateFromContract(minimumDayBeforeDueDate, userID, contract)
	if err != nil {
		return err
	}

	if rowsAffected, err := contractPaymentRepo.BulkAdd(ctx, contractPaymentsSlice, 100); err != nil {
		zap.L().Error("Failed to create contract payments from contract", zap.Error(err))
		return err
	} else if rowsAffected != int64(len(contractPaymentsSlice)) {
		zap.L().Warn("Not all contract payments were created",
			zap.Int("expected", len(contractPaymentsSlice)),
			zap.Int64("actual", rowsAffected))
		return fmt.Errorf("only %d out of %d contract payments were created", rowsAffected, len(contractPaymentsSlice))
	}

	return nil
}

func NewContractPaymentService(
	databaseRegistry *gormrepository.DatabaseRegistry,
	config *config.AdminConfig,
) iservice.ContractPaymentService {
	return &contractPaymentService{
		contractPaymentRepo:    databaseRegistry.ContractPaymentRepository,
		contractRepo:           databaseRegistry.ContractRepository,
		paymentCalculationRepo: databaseRegistry.ContractPaymentCalculationRepository,
		config:                 config,
	}

}

// region: ================ Helper functions ================

func (c *contractPaymentService) processPaymentDateFromContract(
	minimumDayBeforeDueDate int,
	userID uuid.UUID,
	contract *model.Contract,
) (contractPaymentsSlice []*model.ContractPayment, err error) {
	// Default ContractPayment entry for deposit
	depositNote := fmt.Sprintf("Deposit payment before starting the contract number %s for brand %s", *contract.ContractNumber, contract.Brand.Name)
	var tempDepositPercent float64
	if contract.DepositPercent != nil {
		tempDepositPercent = float64(*contract.DepositPercent)
	}
	depositContractPayment := &model.ContractPayment{
		ContractID:            contract.ID,
		InstallmentPercentage: tempDepositPercent,
		Amount:                float64(*contract.DepositAmount),
		DueDate:               contract.StartDate,
		PaymentMethod:         enum.ContractPaymentMethodBankTransfer,
		Note:                  &depositNote,
		IsDeposit:             true,
		CreatedBy:             &userID,
		UpdatedBy:             &userID,
	}
	if contract.IsDepositPaid != nil && *contract.IsDepositPaid {
		depositContractPayment.Status = enum.ContractPaymentStatusPaid
	}
	contractPaymentsSlice = append(contractPaymentsSlice, depositContractPayment)

	// Add contract payments entries based on contract type and schedules
	switch contract.Type {
	case enum.ContractTypeAdvertising, enum.ContractTypeAmbassador:
		var advertisingFinancialTerms dtos.AdvertisingFinancialTerms
		if err = json.Unmarshal(contract.FinancialTerms, &advertisingFinancialTerms); err != nil {
			zap.L().Error("Failed to unmarshal advertising financial terms", zap.Error(err))
			return
		}

		// Process each schedule to create payments
		for _, schedule := range advertisingFinancialTerms.Schedules {
			var dueDate time.Time
			dueDate, err = time.Parse(utils.DateFormat, schedule.DueDate)
			if err != nil {
				zap.L().Error("Failed to parse due date", zap.Error(err))
				return nil, err
			}

			note := fmt.Sprintf("Payment for milestone: %s - contract number: %s", utils.ToString(schedule.ID), *contract.ContractNumber)

			contractPayment := &model.ContractPayment{
				ContractID:            contract.ID,
				InstallmentPercentage: float64(schedule.Percent),
				Amount:                float64(schedule.Amount),
				DueDate:               dueDate,
				PaymentMethod:         enum.ContractPaymentMethodBankTransfer,
				Note:                  &note,
				CreatedBy:             &userID,
				UpdatedBy:             &userID,
			}
			contractPaymentsSlice = append(contractPaymentsSlice, contractPayment)
		}

	case enum.ContractTypeAffiliate:
		var affiliateFinancialTerms dtos.AffiliateFinancialTerms
		if err = json.Unmarshal(contract.FinancialTerms, &affiliateFinancialTerms); err != nil {
			zap.L().Error("Failed to unmarshal affiliate financial terms", zap.Error(err))
			return
		}

		paymentCycle := enum.PaymentCycle(affiliateFinancialTerms.PaymentCycle)
		if !paymentCycle.IsValid() {
			zap.L().Error("Invalid payment cycle", zap.String("payment_cycle", string(affiliateFinancialTerms.PaymentCycle)))
			err = fmt.Errorf("invalid payment cycle: %s", affiliateFinancialTerms.PaymentCycle)
			return
		}

		// Use shared payment cycle calculator
		var paymentResults []helper.PaymentDateResult
		paymentResults, err = helper.CalculatePaymentDatesForCycle(
			paymentCycle,
			contract.StartDate,
			contract.EndDate,
			affiliateFinancialTerms.PaymentDate,
			minimumDayBeforeDueDate,
		)
		if err != nil {
			zap.L().Error("Failed to calculate affiliate payment dates", zap.Error(err))
			return nil, err
		}

		depositPercent := float64(0)
		if contract.DepositPercent != nil {
			depositPercent = float64(*contract.DepositPercent)
		}

		basePayment, percent := helper.CalculateBasePaymentPerPeriod(float64(affiliateFinancialTerms.TotalCost), depositPercent, len(paymentResults))

		// Devided equally the payment amount per period based on the total cost
		// The performance cost will be calculated later during the payment link creation phase
		for _, paymentResult := range paymentResults {
			periodStart := paymentResult.PeriodStart
			periodEnd := paymentResult.PeriodEnd
			paymentResult.Note = fmt.Sprintf(`%s
Base Payment: %.2f VND for contract number %s.
Further performance cost will be calculated during the payment link creation phase`,
				paymentResult.Note, basePayment, *contract.ContractNumber)
			contractPayment := &model.ContractPayment{
				ContractID:            contract.ID,
				InstallmentPercentage: percent,
				Amount:                basePayment,
				DueDate:               paymentResult.DueDate,
				PeriodStart:           &periodStart,
				PeriodEnd:             &periodEnd,
				PaymentMethod:         enum.ContractPaymentMethodBankTransfer,
				Note:                  &paymentResult.Note,
				CreatedBy:             &userID,
				UpdatedBy:             &userID,
			}
			contractPaymentsSlice = append(contractPaymentsSlice, contractPayment)
		}

	case enum.ContractTypeCoProduce:
		var coProducingFinancialTerms dtos.CoProducingFinancialTerms
		if err = json.Unmarshal(contract.FinancialTerms, &coProducingFinancialTerms); err != nil {
			zap.L().Error("Failed to unmarshal co-producing financial terms", zap.Error(err))
			return
		}

		paymentCycle := enum.PaymentCycle(coProducingFinancialTerms.ProfitDistributionCycle)
		if !paymentCycle.IsValid() {
			zap.L().Error("Invalid profit distribution cycle", zap.String("profit_distribution_cycle", string(coProducingFinancialTerms.ProfitDistributionCycle)))
			err = fmt.Errorf("invalid profit distribution cycle: %s", coProducingFinancialTerms.ProfitDistributionCycle)
			return
		}

		// Use shared payment cycle calculator
		paymentResults, err := helper.CalculatePaymentDatesForCycle(
			paymentCycle,
			contract.StartDate,
			contract.EndDate,
			coProducingFinancialTerms.ProfitDistributionDate,
			minimumDayBeforeDueDate,
		)
		if err != nil {
			zap.L().Error("Failed to calculate co-producing payment dates", zap.Error(err))
			return nil, err
		}

		depositPercent := float64(0)
		if contract.DepositPercent != nil {
			depositPercent = float64(*contract.DepositPercent)
		}

		basePayment, percent := helper.CalculateBasePaymentPerPeriod(float64(coProducingFinancialTerms.TotalCost), depositPercent, len(paymentResults))

		// Devided equally the payment amount per period based on the total cost
		// The revenue distribution will be calculated later during the payment link creation phase
		for _, paymentResult := range paymentResults {
			periodStart := paymentResult.PeriodStart
			periodEnd := paymentResult.PeriodEnd
			paymentResult.Note = fmt.Sprintf(`%s
Base Payment: %.2f VND for contract number %s.
Further revenue distribution will be calculated during the payment link creation phase.`,
				paymentResult.Note, basePayment, *contract.ContractNumber)
			contractPayment := &model.ContractPayment{
				ContractID:            contract.ID,
				InstallmentPercentage: percent,
				Amount:                basePayment,
				DueDate:               paymentResult.DueDate,
				PeriodStart:           &periodStart,
				PeriodEnd:             &periodEnd,
				PaymentMethod:         enum.ContractPaymentMethodBankTransfer,
				Note:                  &paymentResult.Note,
				CreatedBy:             &userID,
				UpdatedBy:             &userID,
			}
			contractPaymentsSlice = append(contractPaymentsSlice, contractPayment)
		}
	}

	return
}

// endregion

// region: ================ Payment Calculation Methods ================

// shouldRecalculate is a fast pre-filter check (no DB calls) to determine
// if a payment needs recalculation based on contract type and payment status.
func (c *contractPaymentService) shouldRecalculate(payment *model.ContractPayment) bool {
	// Check 1: Contract must be loaded
	if payment.Contract == nil {
		return false
	}

	// Check 2: Contract type must be AFFILIATE or CO_PRODUCING
	if payment.Contract.Type != enum.ContractTypeAffiliate &&
		payment.Contract.Type != enum.ContractTypeCoProduce {
		return false
	}

	// Check 3: Payment must not be paid yet
	if payment.Status == enum.ContractPaymentStatusPaid {
		return false
	}

	// Check 4: Payment must not be locked (pending payment link)
	if payment.LockedAt != nil {
		return false
	}

	// Check 5: Must be the CURRENT payment period
	if !c.isCurrentPeriod(payment) {
		return false
	}

	return true
}

// isCurrentPeriod checks if the payment period is the current active period.
// Returns true if now is between PeriodStart (inclusive) and PeriodEnd (exclusive).
func (c *contractPaymentService) isCurrentPeriod(payment *model.ContractPayment) bool {
	now := time.Now()

	// Payment period must be defined
	if payment.PeriodStart == nil || payment.PeriodEnd == nil {
		return false
	}

	// Check: PeriodStart <= now < PeriodEnd
	return !now.Before(*payment.PeriodStart) && now.Before(*payment.PeriodEnd)
}

// recalculateIfNeeded performs recalculation for AFFILIATE or CO_PRODUCING payments
// if conditions are met. Updates the payment in the database if amount changed.
func (c *contractPaymentService) recalculateIfNeeded(ctx context.Context, payment *model.ContractPayment) error {
	// Double-check conditions (for thread safety)
	if !c.shouldRecalculate(payment) {
		return nil
	}

	var newAmount float64
	var breakdownJSON []byte
	var err error

	switch payment.Contract.Type {
	case enum.ContractTypeAffiliate:
		calculation, calcErr := c.calculateAffiliatePayment(ctx, payment)
		if calcErr != nil {
			return fmt.Errorf("failed to calculate affiliate payment: %w", calcErr)
		}
		newAmount = float64(calculation.NetPayment)
		breakdownJSON, err = json.Marshal(calculation)
		if err != nil {
			return fmt.Errorf("failed to marshal affiliate calculation breakdown: %w", err)
		}

	case enum.ContractTypeCoProduce:
		calculation, calcErr := c.calculateCoProducingPayment(ctx, payment)
		if calcErr != nil {
			return fmt.Errorf("failed to calculate co-producing payment: %w", calcErr)
		}
		newAmount = calculation.BrandShare
		breakdownJSON, err = json.Marshal(calculation)
		if err != nil {
			return fmt.Errorf("failed to marshal co-producing calculation breakdown: %w", err)
		}

	default:
		return nil
	}

	// Only update if amount changed (avoid unnecessary DB writes)
	if payment.Amount == newAmount && payment.CalculatedAt != nil {
		return nil
	}

	// Update payment
	now := time.Now()
	payment.Amount = newAmount
	payment.CalculatedAt = &now
	payment.CalculationBreakdown = breakdownJSON

	if err := c.contractPaymentRepo.Update(ctx, payment); err != nil {
		return fmt.Errorf("failed to update payment with new calculation: %w", err)
	}

	zap.L().Info("Payment amount recalculated",
		zap.String("payment_id", payment.ID.String()),
		zap.Float64("new_amount", newAmount),
		zap.String("contract_type", string(payment.Contract.Type)))

	return nil
}

// calculateAffiliatePayment calculates the payment for an AFFILIATE contract
// using tiered/level-based pricing (similar to electricity billing).
func (c *contractPaymentService) calculateAffiliatePayment(
	ctx context.Context,
	payment *model.ContractPayment,
) (*dtos.AffiliatePaymentCalculation, error) {
	// Parse financial terms from contract
	var terms dtos.AffiliateFinancialTerms
	if err := json.Unmarshal(payment.Contract.FinancialTerms, &terms); err != nil {
		return nil, fmt.Errorf("failed to parse affiliate financial terms: %w", err)
	}

	// Get total clicks for the period
	totalClicks, err := c.paymentCalculationRepo.GetTotalClicksForContract(
		ctx,
		payment.ContractID,
		*payment.PeriodStart,
		*payment.PeriodEnd,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get total clicks: %w", err)
	}

	// Calculate tiered payment
	grossPayment, breakdown := c.calculateTieredPayment(totalClicks, terms.BasePerClick, terms.Levels)

	// Apply tax withholding if applicable
	var taxAmount int64
	if grossPayment > int64(terms.TaxWithholding.Threshold) {
		taxableAmount := grossPayment - int64(terms.TaxWithholding.Threshold)
		taxAmount = taxableAmount * int64(terms.TaxWithholding.RatePercent) / 100
	}

	return &dtos.AffiliatePaymentCalculation{
		ContractID:   payment.ContractID,
		PeriodStart:  *payment.PeriodStart,
		PeriodEnd:    *payment.PeriodEnd,
		TotalClicks:  totalClicks,
		GrossPayment: grossPayment,
		TaxAmount:    taxAmount,
		NetPayment:   grossPayment - taxAmount,
		Breakdown:    breakdown,
		CalculatedAt: time.Now(),
	}, nil
}

// calculateTieredPayment implements electricity-style tiered billing.
// Returns gross payment and detailed breakdown per level.
func (c *contractPaymentService) calculateTieredPayment(
	totalClicks int64,
	baseRate int,
	levels []dtos.Level,
) (int64, []dtos.LevelPaymentBreakdown) {
	// Sort levels by Level number
	sortedLevels := make([]dtos.Level, len(levels))
	copy(sortedLevels, levels)
	sort.Slice(sortedLevels, func(i, j int) bool {
		return sortedLevels[i].Level < sortedLevels[j].Level
	})

	var payment int64
	var breakdown []dtos.LevelPaymentBreakdown
	remainingClicks := totalClicks
	previousMax := int64(0)

	for _, level := range sortedLevels {
		if remainingClicks <= 0 {
			break
		}

		// Calculate clicks in this tier
		tierCapacity := level.MaxClicks - previousMax
		clicksInTier := min(remainingClicks, tierCapacity)

		// Calculate payment for this tier
		ratePerClick := int(float32(baseRate) * level.Multiplier)
		tierPayment := clicksInTier * int64(ratePerClick)
		payment += tierPayment

		breakdown = append(breakdown, dtos.LevelPaymentBreakdown{
			Level:        level.Level,
			ClicksInTier: clicksInTier,
			Multiplier:   level.Multiplier,
			RatePerClick: ratePerClick,
			TierPayment:  tierPayment,
		})

		remainingClicks -= clicksInTier
		previousMax = level.MaxClicks
	}

	// If clicks exceed highest level, charge at highest multiplier
	if remainingClicks > 0 && len(sortedLevels) > 0 {
		highestLevel := sortedLevels[len(sortedLevels)-1]
		ratePerClick := int(float32(baseRate) * highestLevel.Multiplier)
		tierPayment := remainingClicks * int64(ratePerClick)
		payment += tierPayment

		breakdown = append(breakdown, dtos.LevelPaymentBreakdown{
			Level:        highestLevel.Level + 1, // Overflow tier
			ClicksInTier: remainingClicks,
			Multiplier:   highestLevel.Multiplier,
			RatePerClick: ratePerClick,
			TierPayment:  tierPayment,
		})
	}

	return payment, breakdown
}

// calculateCoProducingPayment calculates the revenue distribution for a CO_PRODUCING contract.
func (c *contractPaymentService) calculateCoProducingPayment(
	ctx context.Context,
	payment *model.ContractPayment,
) (*dtos.CoProducingPaymentCalculation, error) {
	// Parse financial terms from contract
	var terms dtos.CoProducingFinancialTerms
	if err := json.Unmarshal(payment.Contract.FinancialTerms, &terms); err != nil {
		return nil, fmt.Errorf("failed to parse co-producing financial terms: %w", err)
	}

	// Get limited product revenue for the period
	revenueResult, err := c.paymentCalculationRepo.GetLimitedProductRevenue(
		ctx,
		payment.ContractID,
		*payment.PeriodStart,
		*payment.PeriodEnd,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get limited product revenue: %w", err)
	}

	// Calculate distribution
	companyShare := revenueResult.TotalRevenue * float64(terms.CompanyPercent) / 100
	brandShare := revenueResult.TotalRevenue * float64(terms.KolPercent) / 100

	return &dtos.CoProducingPaymentCalculation{
		ContractID:     payment.ContractID,
		PeriodStart:    *payment.PeriodStart,
		PeriodEnd:      *payment.PeriodEnd,
		TotalRevenue:   revenueResult.TotalRevenue,
		CompanyPercent: terms.CompanyPercent,
		BrandPercent:   terms.KolPercent,
		CompanyShare:   companyShare,
		BrandShare:     brandShare,
		RevenueBreakdown: &dtos.LimitedProductRevenueBreakdown{
			PreOrderRevenue: revenueResult.PreOrderRevenue,
			OrderRevenue:    revenueResult.OrderRevenue,
			TotalRevenue:    revenueResult.TotalRevenue,
		},
		CalculatedAt: time.Now(),
	}, nil
}

// endregion

// region: ================ Payment Locking Methods ================

// LockPaymentAmount locks the current calculated amount when creating a payment link.
// This prevents the amount from changing while payment is in progress.
// New clicks/revenue after locking will be attributed to the next payment period.
func (c *contractPaymentService) LockPaymentAmount(
	ctx context.Context,
	payment *model.ContractPayment,
) error {
	// Validate contract type
	if payment.Contract == nil {
		return fmt.Errorf("contract must be loaded")
	}

	if payment.Contract.Type != enum.ContractTypeAffiliate &&
		payment.Contract.Type != enum.ContractTypeCoProduce {
		// Nothing to lock for non-variable payment contracts
		return nil
	}

	// Calculate current amount
	if err := c.recalculateIfNeeded(ctx, payment); err != nil {
		return fmt.Errorf("failed to calculate before locking: %w", err)
	}

	now := time.Now()
	payment.LockedAmount = &payment.Amount
	payment.LockedAt = &now

	// Store type-specific locked values
	switch payment.Contract.Type {
	case enum.ContractTypeAffiliate:
		clicks, _ := c.paymentCalculationRepo.GetTotalClicksForContract(
			ctx, payment.ContractID, *payment.PeriodStart, *payment.PeriodEnd)
		payment.LockedClicks = &clicks

	case enum.ContractTypeCoProduce:
		revenue, _ := c.paymentCalculationRepo.GetLimitedProductRevenue(
			ctx, payment.ContractID, *payment.PeriodStart, *payment.PeriodEnd)
		if revenue != nil {
			payment.LockedRevenue = &revenue.TotalRevenue
		}
	}

	if err := c.contractPaymentRepo.Update(ctx, payment); err != nil {
		return fmt.Errorf("failed to lock payment amount: %w", err)
	}

	zap.L().Info("Payment amount locked",
		zap.String("payment_id", payment.ID.String()),
		zap.Float64("locked_amount", payment.Amount))

	return nil
}

// UnlockPaymentOnFailure clears the locked state when payment fails.
// This allows the amount to be recalculated on the next GET request.
func (c *contractPaymentService) UnlockPaymentOnFailure(
	ctx context.Context,
	payment *model.ContractPayment,
) error {
	payment.LockedAmount = nil
	payment.LockedAt = nil
	payment.LockedClicks = nil
	payment.LockedRevenue = nil

	if err := c.contractPaymentRepo.Update(ctx, payment); err != nil {
		return fmt.Errorf("failed to unlock payment: %w", err)
	}

	zap.L().Info("Payment unlocked after failure",
		zap.String("payment_id", payment.ID.String()))

	return nil
}

// endregion
