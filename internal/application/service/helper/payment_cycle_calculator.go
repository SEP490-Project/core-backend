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
	ID          *int8 // Optional ID for reference
	DueDate     time.Time
	PeriodStart time.Time // Start of the payment period (inclusive)
	PeriodEnd   time.Time // End of the payment period (exclusive)
	Note        string
}

// CalculateScheduleBasedPaymentDates calculates payment dates from schedule array
// Used by ADVERTISING and BRAND_AMBASSADOR contract types
// Note: All schedules are treated as regular payments (deposit is handled separately in contract)
func CalculateScheduleBasedPaymentDates(
	startDate, endDate time.Time,
	schedules []dtos.Schedule,
) ([]PaymentDateResult, error) {
	var results []PaymentDateResult
	lastTrackedDate := startDate

	for _, schedule := range schedules {
		dueDate, err := time.Parse(utils.DateFormat, schedule.DueDate)
		if err != nil {
			zap.L().Error("Failed to parse schedule due date",
				zap.String("due_date", schedule.DueDate),
				zap.Error(err))
			return nil, fmt.Errorf("failed to parse schedule due date: %w", err)
		}

		pStart := lastTrackedDate
		pEnd := dueDate
		if pEnd.After(endDate) {
			pEnd = endDate
		}
		results = append(results, PaymentDateResult{
			ID:          schedule.ID,
			DueDate:     dueDate,
			PeriodStart: pStart,
			PeriodEnd:   pEnd,
			Note:        fmt.Sprintf("Payment for milestone: %s", utils.ToString(schedule.ID)),
		})
		lastTrackedDate = pEnd
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
		if startDate.Day()+minimumDayBeforeDueDate > due.Day() {
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
		if periodStart.Before(startDate) || len(results) == 0 {
			periodStart = startDate
		}
		periodEnd := periodStart.AddDate(0, 1, 0) // First day of next month
		if periodEnd.After(endDate) {
			periodEnd = endDate // Day contract end
		}

		results = append(results, PaymentDateResult{
			DueDate:     due,
			PeriodStart: periodStart,
			PeriodEnd:   periodEnd,
			Note:        fmt.Sprintf("Monthly payment for period: %s", due.Format("02/01/2006")),
		})

		current = current.AddDate(0, 1, 0)
		due = calcDueDate(current)
	}

	// Add final residual payment if last period end is before contract end
	var lastPeriodEnd time.Time
	lastPaymentDate := results[len(results)-1]
	if len(results) > 0 {
		lastPeriodEnd = lastPaymentDate.PeriodEnd.AddDate(0, 0, -1) // Convert back to inclusive for comparison
	} else {
		lastPeriodEnd = startDate.AddDate(0, 0, -1)
	}

	if lastPeriodEnd.Before(endDate) {
		results = append(results, PaymentDateResult{
			DueDate:     endDate,
			PeriodStart: lastPeriodEnd.AddDate(0, 0, 1),
			PeriodEnd:   endDate.AddDate(0, 0, 1),
			Note:        fmt.Sprintf("Final residual payment: %s", endDate.Format("02/01/2006")),
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
	lastTrackedEnd := contractStartDate

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
		// Adjust period start for first payment to ensure no gaps from contract start
		if periodStart.Before(contractStartDate) || len(results) == 0 {
			periodStart = contractStartDate
		}
		periodEnd := periodStart.AddDate(0, 3, 0) // First day of next quarter
		if periodEnd.After(contractEndDate) {
			periodEnd = contractEndDate
		}

		results = append(results, PaymentDateResult{
			DueDate:     dueDate,
			PeriodStart: periodStart,
			PeriodEnd:   periodEnd,
			Note:        fmt.Sprintf("Quarterly payment due: %s", dueDate.Format(utils.DateFormat)),
		})
		lastTrackedEnd = periodEnd.AddDate(0, 0, -1)
	}

	if lastTrackedEnd.Before(contractEndDate) {
		results = append(results, PaymentDateResult{
			DueDate:     contractEndDate,
			PeriodStart: lastTrackedEnd.AddDate(0, 0, 1),
			PeriodEnd:   contractEndDate.AddDate(0, 0, 1),
			Note:        fmt.Sprintf("Final residual payment: %s", contractEndDate.Format(utils.DateFormat)),
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

	// 1. Determine the starting year
	// If user's year is before the contract starts, start looking from the contract start year
	startYear := max(paymentDate.Year(), contractStartDate.Year())
	month := paymentDate.Month()
	day := paymentDate.Day()

	// Helper to calculate a safe date for a given year (handles Feb 29/30/31)
	calcDueForYear := func(y int) time.Time {
		maxDay := time.Date(y, month+1, 0, 0, 0, 0, 0, loc).Day()
		return time.Date(y, month, min(day, maxDay), 0, 0, 0, 0, loc)
	}

	// 2. Initialize the first "Due Date"
	due := calcDueForYear(startYear)

	// If the calculated date for that year is still before the contract started,
	// move to the following year to ensure the first payment is valid.
	if due.Before(contractStartDate) {
		startYear++
		due = calcDueForYear(startYear)
	}

	// 3. Generate Annual Cycles
	// This loop will not run if 'due' is already past 'contractEndDate'
	for !due.After(contractEndDate) {
		// Period logic: Usually covers the calendar year of the payment
		pStart := time.Date(due.Year(), 1, 1, 0, 0, 0, 0, loc)
		pEnd := time.Date(due.Year()+1, 1, 1, 0, 0, 0, 0, loc)

		// Boundary Clamping: Ensure periods don't leak outside the contract
		if pStart.Before(contractStartDate) || len(results) == 0 {
			pStart = contractStartDate
		}
		if pEnd.After(contractEndDate) {
			pEnd = contractEndDate // End Day (exclusive)
		}

		results = append(results, PaymentDateResult{
			DueDate:     due,
			PeriodStart: pStart,
			PeriodEnd:   pEnd,
			Note:        fmt.Sprintf("Annual payment for year: %d", due.Year()),
		})

		startYear++
		due = calcDueForYear(startYear)
	}

	// 4. RESIDUAL LOGIC
	// This handles the case where the contract is still active after the last payment
	// OR if the contract was too short to ever trigger an annual payment.
	var lastPeriodEnd time.Time
	if len(results) > 0 {
		// Convert the exclusive PeriodEnd back to inclusive for the comparison check
		lastPeriodEnd = results[len(results)-1].PeriodEnd.AddDate(0, 0, -1)
	} else {
		// No annual payments were generated, treat the "previous end" as the day before start
		lastPeriodEnd = contractStartDate.AddDate(0, 0, -1)
	}

	if lastPeriodEnd.Before(contractEndDate) {
		results = append(results, PaymentDateResult{
			DueDate:     contractEndDate,
			PeriodStart: lastPeriodEnd.AddDate(0, 0, 1), // Start exactly where we left off
			PeriodEnd:   contractEndDate.AddDate(0, 0, 1),
			Note:        fmt.Sprintf("Final residual payment: %s", contractEndDate.Format("2006-01-02")),
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
