package helper

import (
	"core-backend/internal/application/dto/dtos"
	"core-backend/internal/domain/enum"
	"core-backend/pkg/utils"
	"fmt"
	"slices"
	"strconv"
	"time"

	"go.uber.org/zap"
)

// PaymentDateResult represents a calculated payment date with metadata
type PaymentDateResult struct {
	DueDate time.Time
	Note    string
}

// CalculateScheduleBasedPaymentDates calculates payment dates from schedule array
// Used by ADVERTISING and BRAND_AMBASSADOR contract types
// Note: All schedules are treated as regular payments (deposit is handled separately in contract)
func CalculateScheduleBasedPaymentDates(
	schedules []dtos.Schedule,
) ([]PaymentDateResult, error) {
	var results []PaymentDateResult

	for _, schedule := range schedules {
		dueDate, err := time.Parse(utils.DateFormat, schedule.DueDate)
		if err != nil {
			zap.L().Error("Failed to parse schedule due date",
				zap.String("due_date", schedule.DueDate),
				zap.Error(err))
			return nil, fmt.Errorf("failed to parse schedule due date: %w", err)
		}

		note := fmt.Sprintf("Payment for milestone: %s", utils.ToString(schedule.ID))
		results = append(results, PaymentDateResult{
			DueDate: dueDate,
			Note:    note,
		})
	}

	return results, nil
}

// CalculateMonthlyPaymentDates calculates monthly payment dates within contract period
// Used by AFFILIATE and CO_PRODUCING contract types with MONTHLY cycle
func CalculateMonthlyPaymentDates(
	startDate time.Time,
	endDate time.Time,
	paymentDay int,
	minimumDayBeforeDueDate int,
	skipFirstMonthIfNotEnoughLeadTime bool,
) ([]PaymentDateResult, error) {
	if paymentDay < 1 || paymentDay > 31 {
		return nil, fmt.Errorf("invalid payment day: %d (must be 1-31)", paymentDay)
	}

	var results []PaymentDateResult
	isFirstPayment := true

	for currentDate := startDate; currentDate.Before(endDate) || currentDate.Equal(endDate); currentDate = currentDate.AddDate(0, 1, 0) {
		// Skip first month if not enough lead time
		if skipFirstMonthIfNotEnoughLeadTime && isFirstPayment && ((currentDate.Day() + minimumDayBeforeDueDate) > paymentDay) {
			isFirstPayment = false
			zap.L().Debug("Skipping first month payment - not enough lead time",
				zap.Int("current_day", currentDate.Day()),
				zap.Int("payment_day", paymentDay),
				zap.Int("minimum_lead_days", minimumDayBeforeDueDate))
			continue
		}

		// Create due date for this month
		dueDate := time.Date(currentDate.Year(), currentDate.Month(), paymentDay, 0, 0, 0, 0, currentDate.Location())

		// Only include if due date is within contract period
		if dueDate.After(endDate) {
			break
		}

		note := fmt.Sprintf("Monthly payment for period: %s", currentDate.Format(utils.DateFormat))
		results = append(results, PaymentDateResult{
			DueDate: dueDate,
			Note:    note,
		})

		isFirstPayment = false
	}

	// Add final payment if last payment date is before contract end
	if len(results) > 0 {
		lastPaymentDate := results[len(results)-1].DueDate
		if lastPaymentDate.Before(endDate) && !lastPaymentDate.Equal(endDate) {
			note := fmt.Sprintf("Final payment for contract end: %s", endDate.Format(utils.DateFormat))
			results = append(results, PaymentDateResult{
				DueDate: endDate,
				Note:    note,
			})
		}
	}

	return results, nil
}

// CalculateQuarterlyPaymentDates calculates quarterly payment dates
// Used by AFFILIATE and CO_PRODUCING contract types with QUARTERLY cycle
func CalculateQuarterlyPaymentDates(
	contractStartDate time.Time,
	contractEndDate time.Time,
	quarterDates []dtos.PaymentDate,
) ([]PaymentDateResult, error) {
	if len(quarterDates) == 0 {
		return nil, fmt.Errorf("quarterly payment dates array is empty")
	}

	// Sort quarter dates chronologically
	sortedQuarters := make([]dtos.PaymentDate, len(quarterDates))
	copy(sortedQuarters, quarterDates)

	slices.SortFunc(sortedQuarters, func(a, b dtos.PaymentDate) int {
		dateA := time.Date(int(a.Year), time.Month(a.Month), int(a.Day), 0, 0, 0, 0, time.Local)
		dateB := time.Date(int(b.Year), time.Month(b.Month), int(b.Day), 0, 0, 0, 0, time.Local)
		return dateA.Compare(dateB)
	})

	var results []PaymentDateResult

	for _, quarter := range sortedQuarters {
		dueDate := time.Date(int(quarter.Year), time.Month(quarter.Month), int(quarter.Day), 0, 0, 0, 0, time.Local)

		// Only include if due date is within contract period
		if dueDate.Before(contractStartDate) || dueDate.After(contractEndDate) {
			continue
		}

		note := fmt.Sprintf("Quarterly payment due: %s", dueDate.Format(utils.DateFormat))
		results = append(results, PaymentDateResult{
			DueDate: dueDate,
			Note:    note,
		})
	}

	// Add final payment if last quarterly date is before contract end
	if len(results) > 0 {
		lastPaymentDate := results[len(results)-1].DueDate
		if lastPaymentDate.Before(contractEndDate) && !lastPaymentDate.Equal(contractEndDate) {
			note := fmt.Sprintf("Final payment for contract end: %s", contractEndDate.Format(utils.DateFormat))
			results = append(results, PaymentDateResult{
				DueDate: contractEndDate,
				Note:    note,
			})
		}
	}

	return results, nil
}

// CalculateAnnualPaymentDates calculates annual payment dates
// Used by AFFILIATE and CO_PRODUCING contract types with ANNUALLY cycle
func CalculateAnnualPaymentDates(
	contractStartDate time.Time,
	contractEndDate time.Time,
	paymentDate time.Time, // The month/day to pay each year
) ([]PaymentDateResult, error) {
	var results []PaymentDateResult

	for currentDate := contractStartDate; currentDate.Before(contractEndDate) || currentDate.Equal(contractEndDate); currentDate = currentDate.AddDate(1, 0, 0) {
		dueDate := time.Date(currentDate.Year(), paymentDate.Month(), paymentDate.Day(), 0, 0, 0, 0, currentDate.Location())

		// Only include if due date is within contract period
		if dueDate.After(contractEndDate) {
			break
		}

		note := fmt.Sprintf("Annual payment for year: %d", currentDate.Year())
		results = append(results, PaymentDateResult{
			DueDate: dueDate,
			Note:    note,
		})
	}

	// Add final payment if last annual date is before contract end
	if len(results) > 0 {
		lastPaymentDate := results[len(results)-1].DueDate
		if lastPaymentDate.Before(contractEndDate) && !lastPaymentDate.Equal(contractEndDate) {
			note := fmt.Sprintf("Final payment for contract end: %s", contractEndDate.Format(utils.DateFormat))
			results = append(results, PaymentDateResult{
				DueDate: contractEndDate,
				Note:    note,
			})
		}
	}

	return results, nil
}

// CalculatePaymentDatesForCycle is a convenience function that routes to the appropriate calculator
// based on payment cycle type
func CalculatePaymentDatesForCycle(
	paymentCycle enum.PaymentCycle,
	contractStartDate time.Time,
	contractEndDate time.Time,
	paymentDateData any, // Can be: int (monthly day), []PaymentDate (quarterly), or string (annually)
	minimumDayBeforeDueDate int,
) ([]PaymentDateResult, error) {
	switch paymentCycle {
	case enum.PaymentCycleMonthly:
		var paymentDay int
		switch v := paymentDateData.(type) {
		case int:
			paymentDay = v
		case string:
			var err error
			paymentDay, err = strconv.Atoi(v)
			if err != nil {
				return nil, fmt.Errorf("failed to parse monthly payment day: %w", err)
			}
		default:
			return nil, fmt.Errorf("invalid payment date data type for monthly cycle: %T", paymentDateData)
		}

		return CalculateMonthlyPaymentDates(
			contractStartDate,
			contractEndDate,
			paymentDay,
			minimumDayBeforeDueDate,
			true, // Skip first month if not enough lead time
		)

	case enum.PaymentCycleQuarterly:
		quarterDatesRaw, err := json.Marshal(paymentDateData)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal quarterly payment dates: %w", err)
		}
		var quarterDates []dtos.PaymentDate
		if err = json.Unmarshal(quarterDatesRaw, &quarterDates); err != nil {
			return nil, fmt.Errorf("failed to unmarshal quarterly payment dates: %w", err)
		}

		return CalculateQuarterlyPaymentDates(
			contractStartDate,
			contractEndDate,
			quarterDates,
		)

	case enum.PaymentCycleAnnually:
		var paymentDate time.Time
		switch v := paymentDateData.(type) {
		case time.Time:
			paymentDate = v
		case string:
			var err error
			paymentDate, err = time.Parse(utils.DateFormat, v)
			if err != nil {
				return nil, fmt.Errorf("failed to parse annual payment date: %w", err)
			}
		default:
			return nil, fmt.Errorf("invalid payment date data type for annual cycle: %T", paymentDateData)
		}

		return CalculateAnnualPaymentDates(
			contractStartDate,
			contractEndDate,
			paymentDate,
		)

	default:
		return nil, fmt.Errorf("unsupported payment cycle: %s", paymentCycle)
	}
}
