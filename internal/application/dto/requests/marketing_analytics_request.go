package requests

import (
	"errors"
	"fmt"
	"time"

	"github.com/go-playground/validator/v10"
)

// TimeFilter represents time period filter for analytics queries
type TimeFilter struct {
	FilterType string `form:"filter_type" json:"filter_type" validate:"required,oneof=MONTH QUARTER YEAR" example:"MONTH"`
	Year       int    `form:"year" json:"year" validate:"required,min=2000,max=2100" example:"2024"`
	Month      *int   `form:"month" json:"month" validate:"omitempty,min=1,max=12" example:"11"`   // Required for MONTH
	Quarter    *int   `form:"quarter" json:"quarter" validate:"omitempty,min=1,max=4" example:"4"` // Required for QUARTER
}

// GetDateRange returns start and end dates for the filter
func (f *TimeFilter) GetDateRange() (startDate, endDate time.Time, err error) {
	switch f.FilterType {
	case "MONTH":
		if f.Month == nil {
			return time.Time{}, time.Time{}, errors.New("month is required when filter_type is MONTH")
		}
		startDate = time.Date(f.Year, time.Month(*f.Month), 1, 0, 0, 0, 0, time.UTC)
		endDate = startDate.AddDate(0, 1, 0).Add(-time.Second) // Last second of the month

	case "QUARTER":
		if f.Quarter == nil {
			return time.Time{}, time.Time{}, errors.New("quarter is required when filter_type is QUARTER")
		}
		startMonth := (*f.Quarter-1)*3 + 1
		startDate = time.Date(f.Year, time.Month(startMonth), 1, 0, 0, 0, 0, time.UTC)
		endDate = startDate.AddDate(0, 3, 0).Add(-time.Second) // Last second of the quarter

	case "YEAR":
		startDate = time.Date(f.Year, 1, 1, 0, 0, 0, 0, time.UTC)
		endDate = time.Date(f.Year, 12, 31, 23, 59, 59, 0, time.UTC)

	default:
		return time.Time{}, time.Time{}, fmt.Errorf("invalid filter_type: %s", f.FilterType)
	}

	return startDate, endDate, nil
}

// DashboardFilter represents optional time filter for dashboard
type DashboardFilter struct {
	Year  *int `form:"year" json:"year" validate:"omitempty,min=2000,max=2100" example:"2024"`
	Month *int `form:"month" json:"month" validate:"omitempty,min=1,max=12" example:"11"`
}

// GetYearMonth returns year and month (defaults to current if not provided)
func (f *DashboardFilter) GetYearMonth() (year int, month int) {
	now := time.Now()

	if f.Year != nil {
		year = *f.Year
	} else {
		year = now.Year()
	}

	if f.Month != nil {
		month = *f.Month
	} else {
		month = int(now.Month())
	}

	return year, month
}

// MonthlyRevenueRequest represents request for monthly revenue
type MonthlyRevenueRequest struct {
	Year  int `form:"year" json:"year" validate:"required,min=2000,max=2100" example:"2024"`
	Month int `form:"month" json:"month" validate:"required,min=1,max=12" example:"11"`
}

// UpcomingDeadlineFilter represents filter for upcoming campaign deadlines
type UpcomingDeadlineFilter struct {
	Days int `form:"days" json:"days" validate:"omitempty,min=1,max=365" example:"10"` // Default: 10
}

// GetDays returns the days value (defaults to 10 if not provided)
func (f *UpcomingDeadlineFilter) GetDays() int {
	if f.Days <= 0 {
		return 10 // Default value
	}
	return f.Days
}

// ValidateTimeFilter validates TimeFilter struct with custom rules
func ValidateTimeFilter(sl validator.StructLevel) {
	filter := sl.Current().Interface().(TimeFilter)

	switch filter.FilterType {
	case "MONTH":
		if filter.Month == nil {
			sl.ReportError(filter.Month, "month", "Month", "required_for_month", "")
		}
	case "QUARTER":
		if filter.Quarter == nil {
			sl.ReportError(filter.Quarter, "quarter", "Quarter", "required_for_quarter", "")
		}
	}
}
