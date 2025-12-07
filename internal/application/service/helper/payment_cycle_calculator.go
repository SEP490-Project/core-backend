package helper

import (
	"core-backend/internal/application/dto/dtos"
	"core-backend/internal/domain/enum"
	"core-backend/pkg/utils"
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"time"

	"go.uber.org/zap"
)

// PaymentDateResult represents a calculated payment date with metadata
type PaymentDateResult struct {
	DueDate     time.Time
	PeriodStart time.Time // Start of the payment period (inclusive)
	PeriodEnd   time.Time // End of the payment period (exclusive)
	Note        string
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
	startDate, endDate time.Time,
	paymentDay, minimumDayBeforeDueDate int,
	skipFirstMonthIfNotEnoughLeadTime bool,
) ([]PaymentDateResult, error) {
	if paymentDay < 1 || paymentDay > 31 {
		return nil, fmt.Errorf("invalid payment day: %d (must be 1-31)", paymentDay)
	}

	var results []PaymentDateResult
	loc := startDate.Location()

	// Start from the 1st of the month instead of using startDate directly
	current := time.Date(startDate.Year(), startDate.Month(), 1, 0, 0, 0, 0, loc)

	calcDueDate := func(base time.Time) time.Time {
		// Compute last day of month to avoid invalid dates like 31 Feb
		lastDay := time.Date(base.Year(), base.Month()+1, 0, 0, 0, 0, 0, loc).Day()
		// Clamp payment day to last day of month
		day := min(paymentDay, lastDay)
		return time.Date(base.Year(), base.Month(), day, 0, 0, 0, 0, loc)
	}

	due := calcDueDate(current)

	// Lead time uses due.Day() instead of startDate.Day()
	if skipFirstMonthIfNotEnoughLeadTime {
		effectivePaymentDay := due.Day()
		if startDate.Day()+minimumDayBeforeDueDate > effectivePaymentDay {
			current = current.AddDate(0, 1, 0)
			due = calcDueDate(current)
		}
	}

	// Ensure due date is never before startDate
	for due.Before(startDate) {
		current = current.AddDate(0, 1, 0)
		due = calcDueDate(current)
	}

	for !due.After(endDate) {
		// Calculate period boundaries for this month
		periodStart := time.Date(current.Year(), current.Month(), 1, 0, 0, 0, 0, loc)
		periodEnd := periodStart.AddDate(0, 1, 0) // First day of next month

		results = append(results, PaymentDateResult{
			DueDate:     due,
			PeriodStart: periodStart,
			PeriodEnd:   periodEnd,
			Note:        fmt.Sprintf("Monthly payment for period: %s", due.Format("02/01/2006")),
		})

		current = current.AddDate(0, 1, 0)
		due = calcDueDate(current)
	}

	if len(results) > 0 {
		lastPeriodEnd := results[len(results)-1].PeriodEnd
		if lastPeriodEnd.Before(endDate) {
			// Final payment period: from last period end to contract end
			results = append(results, PaymentDateResult{
				DueDate:     endDate,
				PeriodStart: lastPeriodEnd,
				PeriodEnd:   endDate.AddDate(0, 0, 1), // Day after contract end
				Note:        fmt.Sprintf("Final payment for contract end: %s", endDate.Format("02/01/2006")),
			})
		}
	} else {
		// If no monthly payments were generated (e.g. short contract or skipped first month),
		// add a single payment at the end date
		results = append(results, PaymentDateResult{
			DueDate:     endDate,
			PeriodStart: startDate,
			PeriodEnd:   endDate.AddDate(0, 0, 1),
			Note:        fmt.Sprintf("Final payment for contract end: %s", endDate.Format("02/01/2006")),
		})
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
	loc := contractStartDate.Location()

	for _, quarter := range sortedQuarters {
		dueDate := time.Date(int(quarter.Year), time.Month(quarter.Month), int(quarter.Day), 0, 0, 0, 0, loc)

		// Only include if due date is within contract period
		if dueDate.Before(contractStartDate) || dueDate.After(contractEndDate) {
			continue
		}

		// Calculate period boundaries for this quarter
		// Period starts from the first day of the quarter
		quarterMonth := ((dueDate.Month()-1)/3)*3 + 1 // Q1=Jan, Q2=Apr, Q3=Jul, Q4=Oct
		periodStart := time.Date(dueDate.Year(), quarterMonth, 1, 0, 0, 0, 0, loc)
		periodEnd := periodStart.AddDate(0, 3, 0) // First day of next quarter

		// Adjust period start for first payment to ensure no gaps from contract start
		if len(results) == 0 {
			periodStart = contractStartDate
		}

		note := fmt.Sprintf("Quarterly payment due: %s", dueDate.Format(utils.DateFormat))
		results = append(results, PaymentDateResult{
			DueDate:     dueDate,
			PeriodStart: periodStart,
			PeriodEnd:   periodEnd,
			Note:        note,
		})
	}

	// Add final payment if last quarterly date is before contract end
	// Frontend already handles this and returns contract end date similar to this function
	if len(results) > 0 {
		lastPeriodEnd := results[len(results)-1].PeriodEnd
		if lastPeriodEnd.Before(contractEndDate) {
			// Final period: from last period end to contract end
			note := fmt.Sprintf("Final payment for contract end: %s", contractEndDate.Format(utils.DateFormat))
			results = append(results, PaymentDateResult{
				DueDate:     contractEndDate,
				PeriodStart: lastPeriodEnd,
				PeriodEnd:   contractEndDate.AddDate(0, 0, 1),
				Note:        note,
			})
		}
	} else {
		// If no quarterly payments were generated, add a single payment at the end date
		results = append(results, PaymentDateResult{
			DueDate:     contractEndDate,
			PeriodStart: contractStartDate,
			PeriodEnd:   contractEndDate.AddDate(0, 0, 1),
			Note:        fmt.Sprintf("Final payment for contract end: %s", contractEndDate.Format(utils.DateFormat)),
		})
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
	loc := contractStartDate.Location()

	// Determine firstPaymentDate explicitly instead of looping from contractStartDate
	year := paymentDate.Year() // Change to paymentDate's year to respect user's input and generate correct first payment date
	month := paymentDate.Month()
	day := paymentDate.Day()

	// Clamp day to last day of month to avoid invalid dates (31 Feb, 30 Feb, etc.)
	maxDay := time.Date(year, month+1, 0, 0, 0, 0, 0, loc).Day()
	if day > maxDay {
		day = maxDay
	}

	firstPaymentDate := time.Date(year, month, day, 0, 0, 0, 0, loc)

	// Shift +1 year if firstPaymentDate is before contractStartDate
	if firstPaymentDate.Before(contractStartDate) {
		year++
		maxDay = time.Date(year, month+1, 0, 0, 0, 0, 0, loc).Day()
		day = min(paymentDate.Day(), maxDay)
		firstPaymentDate = time.Date(year, month, day, 0, 0, 0, 0, loc)
	}

	// Loop from firstPaymentDate
	currentDate := firstPaymentDate
	for !currentDate.After(contractEndDate) {
		// Calculate annual period boundaries
		periodStart := time.Date(currentDate.Year(), 1, 1, 0, 0, 0, 0, loc)
		periodEnd := time.Date(currentDate.Year()+1, 1, 1, 0, 0, 0, 0, loc)

		// Adjust period start for first payment to ensure no gaps from contract start
		if len(results) == 0 {
			periodStart = contractStartDate
		}

		results = append(results, PaymentDateResult{
			DueDate:     currentDate,
			PeriodStart: periodStart,
			PeriodEnd:   periodEnd,
			Note:        fmt.Sprintf("Annual payment for year: %d", currentDate.Year()),
		})

		// Clamp next year's day to last day of month
		nextYear := currentDate.Year() + 1
		maxDayNext := time.Date(nextYear, month+1, 0, 0, 0, 0, 0, loc).Day()
		nextDay := min(paymentDate.Day(), maxDayNext)
		currentDate = time.Date(nextYear, month, nextDay, 0, 0, 0, 0, loc)
	}

	if len(results) > 0 {
		lastPeriodEnd := results[len(results)-1].PeriodEnd
		if lastPeriodEnd.Before(contractEndDate) {
			// Final period: from last period end to contract end
			results = append(results, PaymentDateResult{
				DueDate:     contractEndDate,
				PeriodStart: lastPeriodEnd,
				PeriodEnd:   contractEndDate.AddDate(0, 0, 1),
				Note:        fmt.Sprintf("Final payment for contract end: %s", contractEndDate.Format("2006-01-02")),
			})
		}
	} else {
		// If no annual payments were generated, add a single payment at the end date
		results = append(results, PaymentDateResult{
			DueDate:     contractEndDate,
			PeriodStart: contractStartDate,
			PeriodEnd:   contractEndDate.AddDate(0, 0, 1),
			Note:        fmt.Sprintf("Final payment for contract end: %s", contractEndDate.Format("2006-01-02")),
		})
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
