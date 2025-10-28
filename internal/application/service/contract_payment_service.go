package service

import (
	"context"
	"core-backend/internal/application/dto/dtos"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/application/interfaces/iservice"
	"core-backend/internal/application/service/helper"
	"core-backend/internal/domain/constant"
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
	gormrepository "core-backend/internal/infrastructure/gorm_repository"
	"core-backend/pkg/utils"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type ContractPaymentService struct {
	contractPaymentRepo irepository.GenericRepository[model.ContractPayment]
}

// CreateContractPaymentsFromContract implements iservice.ContractPaymentService.
func (c *ContractPaymentService) CreateContractPaymentsFromContract(
	ctx context.Context, userID uuid.UUID, contractID uuid.UUID, uow irepository.UnitOfWork,
) error {
	zap.L().Info("Creating contract payments from contract",
		zap.String("contract_id", contractID.String()))

	contractRepo := uow.Contracts()
	configRepo := uow.AdminConfigs()
	contractPaymentRepo := uow.ContractPayments()

	contract, err := contractRepo.GetByID(ctx, contractID, []string{"Brand"})
	if err != nil {
		zap.L().Error("Failed to fetch contract", zap.Error(err))
		return err
	} else if contract == nil {
		zap.L().Warn("Contract not found", zap.String("contract_id", contractID.String()))
		return fmt.Errorf("contract with ID %s not found", contractID)
	}

	minimumDayBeforeDueDateConfig, err := configRepo.GetByCondition(ctx, func(db *gorm.DB) *gorm.DB {
		return db.Where("key = ?", constant.ConfigKeyMinimumDayBeforeContracPaymentDue)
	}, nil)
	var minimumDayBeforeDueDate int
	if err == nil && minimumDayBeforeDueDateConfig != nil && minimumDayBeforeDueDateConfig.ValueType == enum.ConfigValueTypeNumber {
		minimumDayBeforeDueDate, err = strconv.Atoi(strings.TrimSpace(minimumDayBeforeDueDateConfig.Value))
		if err != nil {
			zap.L().Error("Failed to parse minimum day before contract payment due date config value",
				zap.String("value", minimumDayBeforeDueDateConfig.Value),
				zap.Error(err))
			return err
		}
	} else if minimumDayBeforeDueDateConfig == nil {
		zap.L().Warn("Failed to fetch minimum day before contract payment due date config, default to 5", zap.Error(err))
		minimumDayBeforeDueDate = 5
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
) iservice.ContractPaymentService {
	return &ContractPaymentService{
		contractPaymentRepo: databaseRegistry.ContractPaymentRepository,
	}

}

// region: ================ Helper functions ================

func (c *ContractPaymentService) processPaymentDateFromContract(
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

		remainingPercentRatio := 1.0
		if *contract.DepositPercent != 0 {
			remainingPercentRatio = float64((100 - *contract.DepositPercent) / 100)
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
				InstallmentPercentage: float64(schedule.Percent) * remainingPercentRatio,
				Amount:                float64(schedule.Amount) * remainingPercentRatio,
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

		for _, paymentResult := range paymentResults {
			contractPayment := &model.ContractPayment{
				ContractID:            contract.ID,
				InstallmentPercentage: 0, // Will be calculated later based on actual performance
				Amount:                0,
				DueDate:               paymentResult.DueDate,
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

		for _, paymentResult := range paymentResults {
			contractPayment := &model.ContractPayment{
				ContractID:            contract.ID,
				InstallmentPercentage: 0, // Will be calculated later based on profit distribution
				Amount:                0,
				DueDate:               paymentResult.DueDate,
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
