package service

import (
	"context"
	"core-backend/internal/application/dto/dtos"
	"core-backend/internal/application/interfaces/irepository"
	"core-backend/internal/application/interfaces/iservice"
	"core-backend/internal/domain/constant"
	"core-backend/internal/domain/enum"
	"core-backend/internal/domain/model"
	gormrepository "core-backend/internal/infrastructure/gorm_repository"
	"core-backend/pkg/utils"
	"encoding/json"
	"fmt"
	"slices"
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
	configRepo := uow.Configs()
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

		for _, schedule := range advertisingFinancialTerms.Schedules {
			note := fmt.Sprintf("Payment for milestone: %s - contract number: %s", utils.ToString(schedule.ID), *contract.ContractNumber)

			var dueDate time.Time
			dueDate, err = time.Parse(utils.DateFormat, schedule.DueDate)
			if err != nil {
				zap.L().Error("Failed to parse due date", zap.Error(err))
				return
			}

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

		contractStartDate := contract.StartDate
		contractEndDate := contract.EndDate
		paymentDateStr := utils.ToString(affiliateFinancialTerms.PaymentDate)
		switch paymentCycle {
		case enum.PaymentCycleMonthly:
			var paymentDay int
			paymentDay, err = strconv.Atoi(paymentDateStr)
			if err != nil {
				zap.L().Error("Failed to parse payment date",
					zap.String("payment_date", paymentDateStr),
					zap.String("payment_date", paymentDateStr),
					zap.Error(err))
				return
			}

			isFirstPayment := true
			var currentDate time.Time
			for currentDate = contractStartDate; currentDate.Before(contractEndDate) || currentDate.Equal(contractEndDate); currentDate = currentDate.AddDate(0, 1, 0) {
				if isFirstPayment && ((currentDate.Day() + minimumDayBeforeDueDate) > paymentDay) {
					isFirstPayment = false
					zap.L().Debug("isPaymentDaySkipped for first month",
						zap.Int("current_day", currentDate.Day()),
						zap.Int("payment_day", paymentDay),
						zap.Bool("is_first_payment", isFirstPayment))

					continue
				}
				dueDate := time.Date(currentDate.Year(), currentDate.Month(), paymentDay, 0, 0, 0, 0, time.Local)

				paymentNote := fmt.Sprintf("(NOT YET CALCULATED UTILS %d DAYS BEFORE DUE DATE) Monthly affiliate payment for contract number %s, date %s",
					minimumDayBeforeDueDate,
					*contract.ContractNumber,
					currentDate.Format(utils.DateFormat))
				contractPayment := &model.ContractPayment{
					ContractID:            contract.ID,
					InstallmentPercentage: float64(*contract.DepositPercent),
					Amount:                float64(*contract.DepositAmount),
					DueDate:               dueDate,
					PaymentMethod:         enum.ContractPaymentMethodBankTransfer,
					Note:                  &paymentNote,
					CreatedBy:             &userID,
					UpdatedBy:             &userID,
				}
				contractPaymentsSlice = append(contractPaymentsSlice, contractPayment)
			}
			if currentDate.Before(contractEndDate) && !currentDate.Equal(contractEndDate) {
				paymentNote := fmt.Sprintf("(NOT YET CALCULATED UTILS %d DAYS BEFORE DUE DATE) Final affiliate payment for contract number %s, date %s",
					minimumDayBeforeDueDate,
					*contract.ContractNumber,
					contractEndDate.Format(utils.DateFormat))
				contractPayment := &model.ContractPayment{
					ContractID:            contract.ID,
					InstallmentPercentage: 0,
					Amount:                0,
					DueDate:               contractEndDate,
					PaymentMethod:         enum.ContractPaymentMethodBankTransfer,
					Note:                  &paymentNote,
					CreatedBy:             &userID,
					UpdatedBy:             &userID,
				}
				contractPaymentsSlice = append(contractPaymentsSlice, contractPayment)
			}

		case enum.PaymentCycleQuarterly:
			paymentQuarterData, ok := affiliateFinancialTerms.PaymentDate.([]any)
			if !ok || len(paymentQuarterData) != 4 {
				zap.L().Error("Invalid payment date format for quarterly payment cycle", zap.Any("payment_date", affiliateFinancialTerms.PaymentDate))
				err = fmt.Errorf("invalid payment date format for quarterly payment cycle")
				return
			}
			paymentQuarter := make([]dtos.PaymentDate, 4)
			for i, item := range paymentQuarterData {
				temp, ok := item.(dtos.PaymentDate)
				if !ok {
					zap.L().Error("Invalid payment date item format for quarterly payment cycle", zap.Any("item", item))
					err = fmt.Errorf("invalid payment date item format for quarterly payment cycle")
					return
				}
				paymentQuarter[i] = temp
			}

			slices.SortFunc(paymentQuarter, func(a, b dtos.PaymentDate) int {
				dateA := time.Date(int(a.Year), time.Month(a.Month), int(a.Day), 0, 0, 0, 0, time.Local)
				dateB := time.Date(int(b.Year), time.Month(b.Month), int(b.Day), 0, 0, 0, 0, time.Local)
				return dateA.Compare(dateB)
			})

			for _, quarter := range paymentQuarter {
				dueDate := time.Date(int(quarter.Year), time.Month(quarter.Month), int(quarter.Day), 0, 0, 0, 0, time.Local)
				note := fmt.Sprintf("(NOT YET CALCULATED UTILS %d DAYS BEFORE DUE DATE) Quarterly affiliate payment for contract number %s, due date %s",
					minimumDayBeforeDueDate,
					*contract.ContractNumber,
					dueDate.Format(utils.DateFormat),
				)
				contractPayment := &model.ContractPayment{
					ContractID:            contract.ID,
					InstallmentPercentage: 0,
					Amount:                0,
					DueDate:               dueDate,
					PaymentMethod:         enum.ContractPaymentMethodBankTransfer,
					Note:                  &note,
					CreatedBy:             &userID,
					UpdatedBy:             &userID,
				}
				contractPaymentsSlice = append(contractPaymentsSlice, contractPayment)
			}

			lastPaymentQuarter := time.Date(int(paymentQuarter[3].Year), time.Month(paymentQuarter[3].Month), int(paymentQuarter[3].Day), 0, 0, 0, 0, time.Local)
			if lastPaymentQuarter.Before(contractEndDate) && !lastPaymentQuarter.Equal(contractEndDate) {
				note := fmt.Sprintf("(NOT YET CALCULATED UNTILS %d BEFORE DUE DATE) Final affiliate payment for contract number %s, due date %s",
					minimumDayBeforeDueDate,
					*contract.ContractNumber,
					contractEndDate.Format(utils.DateFormat))
				contractPayment := &model.ContractPayment{
					ContractID:            contract.ID,
					InstallmentPercentage: 0,
					Amount:                0,
					DueDate:               contractEndDate,
					PaymentMethod:         enum.ContractPaymentMethodBankTransfer,
					Note:                  &note,
					CreatedBy:             &userID,
					UpdatedBy:             &userID,
				}
				contractPaymentsSlice = append(contractPaymentsSlice, contractPayment)
			}

		case enum.PaymentCycleAnnually:
			var paymentDate time.Time
			paymentDate, err = time.Parse(utils.DateFormat, paymentDateStr)
			if err != nil {
				zap.L().Error("Failed to parse payment date",
					zap.String("payment_date", paymentDateStr),
					zap.String("payment_date", paymentDateStr),
					zap.Error(err))
				return
			}

			for currentDate := contractStartDate; currentDate.Before(contractEndDate) || currentDate.Equal(contractEndDate); currentDate = currentDate.AddDate(1, 0, 0) {
				dueDate := time.Date(currentDate.Year(), paymentDate.Month(), paymentDate.Day(), 0, 0, 0, 0, currentDate.Location())
				note := fmt.Sprintf("(NOT YET CALCULATED UTILS %d DAYS BEFORE DUE DATE) Annual affiliate payment for contract number %s, due date %s",
					minimumDayBeforeDueDate,
					*contract.ContractNumber,
					dueDate.Format(utils.DateFormat),
				)
				contractPayment := &model.ContractPayment{
					ContractID:            contract.ID,
					InstallmentPercentage: 0,
					Amount:                0,
					DueDate:               dueDate,
					PaymentMethod:         enum.ContractPaymentMethodBankTransfer,
					Note:                  &note,
					CreatedBy:             &userID,
					UpdatedBy:             &userID,
				}
				contractPaymentsSlice = append(contractPaymentsSlice, contractPayment)
			}
			if paymentDate.Before(contractEndDate) && !paymentDate.Equal(contractEndDate) {
				note := fmt.Sprintf("(NOT YET CALCULATED UNTILS %d BEFORE DUE DATE) Final affiliate payment for contract number %s, due date %s",
					minimumDayBeforeDueDate,
					*contract.ContractNumber,
					contractEndDate.Format(utils.DateFormat))
				contractPayment := &model.ContractPayment{
					ContractID:            contract.ID,
					InstallmentPercentage: 0,
					Amount:                0,
					DueDate:               contractEndDate,
					PaymentMethod:         enum.ContractPaymentMethodBankTransfer,
					Note:                  &note,
					CreatedBy:             &userID,
					UpdatedBy:             &userID,
				}
				contractPaymentsSlice = append(contractPaymentsSlice, contractPayment)
			}
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

		contractStartDate := contract.StartDate
		contractEndDate := contract.EndDate
		profitDistributionDateStr := utils.ToString(coProducingFinancialTerms.ProfitDistributionDate)
		switch paymentCycle {
		case enum.PaymentCycleMonthly:
			var paymentDay int
			paymentDay, err = strconv.Atoi(profitDistributionDateStr)
			if err != nil {
				zap.L().Error("Failed to parse profit distribution date date",
					zap.String("profit_distribution_date", profitDistributionDateStr),
					zap.Error(err))
				return
			}

			var currentDate time.Time
			for currentDate = contractStartDate; currentDate.Before(contractEndDate) || currentDate.Equal(contractEndDate); currentDate = currentDate.AddDate(0, 1, 0) {
				if (currentDate.Day() + minimumDayBeforeDueDate) > paymentDay {
					continue
				}
				dueDate := time.Date(currentDate.Year(), currentDate.Month(), paymentDay, 0, 0, 0, 0, currentDate.Location())

				paymentNote := fmt.Sprintf("(NOT YET CALCULATED UTILS %d DAYS BEFORE DUE DATE) Monthly co-producing payment for contract number %s, date %s",
					minimumDayBeforeDueDate,
					*contract.ContractNumber,
					currentDate.Format(utils.DateFormat))
				contractPayment := &model.ContractPayment{
					ContractID:            contract.ID,
					InstallmentPercentage: float64(*contract.DepositPercent),
					Amount:                float64(*contract.DepositAmount),
					DueDate:               dueDate,
					PaymentMethod:         enum.ContractPaymentMethodBankTransfer,
					Note:                  &paymentNote,
					CreatedBy:             &userID,
					UpdatedBy:             &userID,
				}
				contractPaymentsSlice = append(contractPaymentsSlice, contractPayment)
			}
			if currentDate.Before(contractEndDate) && !currentDate.Equal(contractEndDate) {
				paymentNote := fmt.Sprintf("(NOT YET CALCULATED UTILS %d DAYS BEFORE DUE DATE) Final co-producing payment for contract number %s, date %s",
					minimumDayBeforeDueDate,
					*contract.ContractNumber,
					contractEndDate.Format(utils.DateFormat))
				contractPayment := &model.ContractPayment{
					ContractID:            contract.ID,
					InstallmentPercentage: 0,
					Amount:                0,
					DueDate:               contractEndDate,
					PaymentMethod:         enum.ContractPaymentMethodBankTransfer,
					Note:                  &paymentNote,
					CreatedBy:             &userID,
					UpdatedBy:             &userID,
				}
				contractPaymentsSlice = append(contractPaymentsSlice, contractPayment)
			}

		case enum.PaymentCycleQuarterly:
			paymentQuarterData, ok := coProducingFinancialTerms.ProfitDistributionDate.([]any)
			if !ok || len(paymentQuarterData) != 4 {
				zap.L().Error("Invalid payment date format for quarterly profit distribution cycle", zap.Any("payment_date", coProducingFinancialTerms.ProfitDistributionDate))
				err = fmt.Errorf("invalid payment date format for quarterly profit distribution cycle")
				return
			}
			paymentQuarter := make([]dtos.PaymentDate, 4)
			for i, item := range paymentQuarterData {
				var rawBytes []byte
				rawBytes, err = json.Marshal(item)
				if err != nil {
					zap.L().Error("Failed to marshal payment date item for quarterly profit distribution cycle", zap.Any("item", item), zap.Error(err))
					return nil, err
				}
				var paymentDateObj dtos.PaymentDate
				if err = json.Unmarshal(rawBytes, &paymentDateObj); err != nil {
					zap.L().Error("Failed to unmarshal payment date item for quarterly profit distribution cycle", zap.Any("item", item), zap.Error(err))
					return nil, err
				}
				paymentQuarter[i] = paymentDateObj
			}

			slices.SortFunc(paymentQuarter, func(a, b dtos.PaymentDate) int {
				dateA := time.Date(int(a.Year), time.Month(a.Month), int(a.Day), 0, 0, 0, 0, time.Local)
				dateB := time.Date(int(b.Year), time.Month(b.Month), int(b.Day), 0, 0, 0, 0, time.Local)
				return dateA.Compare(dateB)
			})

			for _, quarter := range paymentQuarter {
				dueDate := time.Date(int(quarter.Year), time.Month(quarter.Month), int(quarter.Day), 0, 0, 0, 0, time.Local)
				note := fmt.Sprintf("(NOT YET CALCULATED UTILS %d DAYS BEFORE DUE DATE) Quarterly co-producing payment for contract number %s, due date %s",
					minimumDayBeforeDueDate,
					*contract.ContractNumber,
					dueDate.Format(utils.DateFormat),
				)
				contractPayment := &model.ContractPayment{
					ContractID:            contract.ID,
					InstallmentPercentage: 0,
					Amount:                0,
					DueDate:               dueDate,
					PaymentMethod:         enum.ContractPaymentMethodBankTransfer,
					Note:                  &note,
					CreatedBy:             &userID,
					UpdatedBy:             &userID,
				}
				contractPaymentsSlice = append(contractPaymentsSlice, contractPayment)
			}

			lastPaymentQuarter := time.Date(int(paymentQuarter[3].Year), time.Month(paymentQuarter[3].Month), int(paymentQuarter[3].Day), 0, 0, 0, 0, time.Local)
			if lastPaymentQuarter.Before(contractEndDate) && !lastPaymentQuarter.Equal(contractEndDate) {
				note := fmt.Sprintf("(NOT YET CALCULATED UNTILS %d BEFORE DUE DATE) Final co-producing payment for contract number %s, due date %s",
					minimumDayBeforeDueDate,
					*contract.ContractNumber,
					contractEndDate.Format(utils.DateFormat))
				contractPayment := &model.ContractPayment{
					ContractID:            contract.ID,
					InstallmentPercentage: 0,
					Amount:                0,
					DueDate:               contractEndDate,
					PaymentMethod:         enum.ContractPaymentMethodBankTransfer,
					Note:                  &note,
					CreatedBy:             &userID,
					UpdatedBy:             &userID,
				}
				contractPaymentsSlice = append(contractPaymentsSlice, contractPayment)
			}

		case enum.PaymentCycleAnnually:
			var paymentDate time.Time
			paymentDate, err = time.Parse(utils.DateFormat, profitDistributionDateStr)
			if err != nil {
				zap.L().Error("Failed to parse payment date",
					zap.String("payment_date", profitDistributionDateStr),
					zap.String("payment_date", profitDistributionDateStr),
					zap.Error(err))
				return
			}

			for currentDate := contractStartDate; currentDate.Before(contractEndDate) || currentDate.Equal(contractEndDate); currentDate = currentDate.AddDate(1, 0, 0) {
				dueDate := time.Date(currentDate.Year(), paymentDate.Month(), paymentDate.Day(), 0, 0, 0, 0, currentDate.Location())
				note := fmt.Sprintf("(NOT YET CALCULATED UTILS %d DAYS BEFORE DUE DATE) Annually co-producing payment for contract number %s, due date %s",
					minimumDayBeforeDueDate,
					*contract.ContractNumber,
					dueDate.Format(utils.DateFormat),
				)
				contractPayment := &model.ContractPayment{
					ContractID:            contract.ID,
					InstallmentPercentage: 0,
					Amount:                0,
					DueDate:               dueDate,
					PaymentMethod:         enum.ContractPaymentMethodBankTransfer,
					Note:                  &note,
					CreatedBy:             &userID,
					UpdatedBy:             &userID,
				}
				contractPaymentsSlice = append(contractPaymentsSlice, contractPayment)
			}
			if paymentDate.Before(contractEndDate) && !paymentDate.Equal(contractEndDate) {
				note := fmt.Sprintf("(NOT YET CALCULATED UNTILS %d BEFORE DUE DATE) Final co-producing payment for contract number %s, due date %s",
					minimumDayBeforeDueDate,
					*contract.ContractNumber,
					contractEndDate.Format(utils.DateFormat))
				contractPayment := &model.ContractPayment{
					ContractID:            contract.ID,
					InstallmentPercentage: 0,
					Amount:                0,
					DueDate:               contractEndDate,
					PaymentMethod:         enum.ContractPaymentMethodBankTransfer,
					Note:                  &note,
					CreatedBy:             &userID,
					UpdatedBy:             &userID,
				}
				contractPaymentsSlice = append(contractPaymentsSlice, contractPayment)
			}
		}
	}

	return
}

// endregion
