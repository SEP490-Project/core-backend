package constant

import "time"

// DashboardPeriodPreset defines time period presets for dashboard filtering
type DashboardPeriodPreset string

const (
	PeriodPresetToday       DashboardPeriodPreset = "TODAY"
	PeriodPresetYesterday   DashboardPeriodPreset = "YESTERDAY"
	PeriodPresetThisWeek    DashboardPeriodPreset = "THIS_WEEK"
	PeriodPresetLastWeek    DashboardPeriodPreset = "LAST_WEEK"
	PeriodPresetThisMonth   DashboardPeriodPreset = "THIS_MONTH"
	PeriodPresetLastMonth   DashboardPeriodPreset = "LAST_MONTH"
	PeriodPresetThisQuarter DashboardPeriodPreset = "THIS_QUARTER"
	PeriodPresetLastQuarter DashboardPeriodPreset = "LAST_QUARTER"
	PeriodPresetThisYear    DashboardPeriodPreset = "THIS_YEAR"
	PeriodPresetLastYear    DashboardPeriodPreset = "LAST_YEAR"
	PeriodPresetLast7Days   DashboardPeriodPreset = "LAST_7_DAYS"
	PeriodPresetLast30Days  DashboardPeriodPreset = "LAST_30_DAYS"
	PeriodPresetCustom      DashboardPeriodPreset = "CUSTOM"
)

// IsValid checks if the preset is a valid value
func (p DashboardPeriodPreset) IsValid() bool {
	switch p {
	case PeriodPresetToday, PeriodPresetYesterday,
		PeriodPresetThisWeek, PeriodPresetLastWeek,
		PeriodPresetThisMonth, PeriodPresetLastMonth,
		PeriodPresetThisQuarter, PeriodPresetLastQuarter,
		PeriodPresetThisYear, PeriodPresetLastYear,
		PeriodPresetLast7Days, PeriodPresetLast30Days,
		PeriodPresetCustom:
		return true
	}
	return false
}

// GetLabel returns a human-readable label for the preset
func (p DashboardPeriodPreset) GetLabel() string {
	labels := map[DashboardPeriodPreset]string{
		PeriodPresetToday:       "Today",
		PeriodPresetYesterday:   "Yesterday",
		PeriodPresetThisWeek:    "This Week",
		PeriodPresetLastWeek:    "Last Week",
		PeriodPresetThisMonth:   "This Month",
		PeriodPresetLastMonth:   "Last Month",
		PeriodPresetThisQuarter: "This Quarter",
		PeriodPresetLastQuarter: "Last Quarter",
		PeriodPresetThisYear:    "This Year",
		PeriodPresetLastYear:    "Last Year",
		PeriodPresetLast7Days:   "Last 7 Days",
		PeriodPresetLast30Days:  "Last 30 Days",
		PeriodPresetCustom:      "Custom Range",
	}
	return labels[p]
}

// GetCompareLabel returns a label describing what the current period is compared against
func (p DashboardPeriodPreset) GetCompareLabel() string {
	labels := map[DashboardPeriodPreset]string{
		PeriodPresetToday:       "vs Yesterday",
		PeriodPresetYesterday:   "vs Day Before",
		PeriodPresetThisWeek:    "vs Last Week",
		PeriodPresetLastWeek:    "vs Week Before",
		PeriodPresetThisMonth:   "vs Last Month",
		PeriodPresetLastMonth:   "vs Month Before",
		PeriodPresetThisQuarter: "vs Last Quarter",
		PeriodPresetLastQuarter: "vs Quarter Before",
		PeriodPresetThisYear:    "vs Last Year",
		PeriodPresetLastYear:    "vs Year Before",
		PeriodPresetLast7Days:   "vs Previous 7 Days",
		PeriodPresetLast30Days:  "vs Previous 30 Days",
		PeriodPresetCustom:      "vs Previous Period",
	}
	return labels[p]
}

// DateRange represents a date range with start and end times
type DateRange struct {
	Start time.Time
	End   time.Time
}

// GetDateRangeForPreset calculates start and end dates based on preset
// Returns (current period, previous period for comparison)
func GetDateRangeForPreset(preset DashboardPeriodPreset, customStart, customEnd *time.Time) (current DateRange, previous DateRange) {
	now := time.Now()
	loc := now.Location()

	switch preset {
	case PeriodPresetToday:
		current = DateRange{
			Start: startOfDay(now),
			End:   endOfDay(now),
		}
		previous = DateRange{
			Start: startOfDay(now.AddDate(0, 0, -1)),
			End:   endOfDay(now.AddDate(0, 0, -1)),
		}

	case PeriodPresetYesterday:
		current = DateRange{
			Start: startOfDay(now.AddDate(0, 0, -1)),
			End:   endOfDay(now.AddDate(0, 0, -1)),
		}
		previous = DateRange{
			Start: startOfDay(now.AddDate(0, 0, -2)),
			End:   endOfDay(now.AddDate(0, 0, -2)),
		}

	case PeriodPresetThisWeek:
		current = DateRange{
			Start: startOfWeek(now, loc),
			End:   endOfDay(now),
		}
		previous = DateRange{
			Start: startOfWeek(now.AddDate(0, 0, -7), loc),
			End:   endOfWeek(now.AddDate(0, 0, -7), loc),
		}

	case PeriodPresetLastWeek:
		lastWeek := now.AddDate(0, 0, -7)
		current = DateRange{
			Start: startOfWeek(lastWeek, loc),
			End:   endOfWeek(lastWeek, loc),
		}
		previous = DateRange{
			Start: startOfWeek(now.AddDate(0, 0, -14), loc),
			End:   endOfWeek(now.AddDate(0, 0, -14), loc),
		}

	case PeriodPresetThisMonth:
		current = DateRange{
			Start: startOfMonth(now),
			End:   endOfDay(now),
		}
		previous = DateRange{
			Start: startOfMonth(now.AddDate(0, -1, 0)),
			End:   endOfMonth(now.AddDate(0, -1, 0)),
		}

	case PeriodPresetLastMonth:
		lastMonth := now.AddDate(0, -1, 0)
		current = DateRange{
			Start: startOfMonth(lastMonth),
			End:   endOfMonth(lastMonth),
		}
		previous = DateRange{
			Start: startOfMonth(now.AddDate(0, -2, 0)),
			End:   endOfMonth(now.AddDate(0, -2, 0)),
		}

	case PeriodPresetThisQuarter:
		current = DateRange{
			Start: startOfQuarter(now),
			End:   endOfDay(now),
		}
		previous = DateRange{
			Start: startOfQuarter(now.AddDate(0, -3, 0)),
			End:   endOfQuarter(now.AddDate(0, -3, 0)),
		}

	case PeriodPresetLastQuarter:
		lastQuarter := now.AddDate(0, -3, 0)
		current = DateRange{
			Start: startOfQuarter(lastQuarter),
			End:   endOfQuarter(lastQuarter),
		}
		previous = DateRange{
			Start: startOfQuarter(now.AddDate(0, -6, 0)),
			End:   endOfQuarter(now.AddDate(0, -6, 0)),
		}

	case PeriodPresetThisYear:
		current = DateRange{
			Start: startOfYear(now),
			End:   endOfDay(now),
		}
		previous = DateRange{
			Start: startOfYear(now.AddDate(-1, 0, 0)),
			End:   endOfYear(now.AddDate(-1, 0, 0)),
		}

	case PeriodPresetLastYear:
		lastYear := now.AddDate(-1, 0, 0)
		current = DateRange{
			Start: startOfYear(lastYear),
			End:   endOfYear(lastYear),
		}
		previous = DateRange{
			Start: startOfYear(now.AddDate(-2, 0, 0)),
			End:   endOfYear(now.AddDate(-2, 0, 0)),
		}

	case PeriodPresetLast7Days:
		current = DateRange{
			Start: startOfDay(now.AddDate(0, 0, -6)), // 7 days including today
			End:   endOfDay(now),
		}
		previous = DateRange{
			Start: startOfDay(now.AddDate(0, 0, -13)), // Previous 7 days
			End:   endOfDay(now.AddDate(0, 0, -7)),
		}

	case PeriodPresetLast30Days:
		current = DateRange{
			Start: startOfDay(now.AddDate(0, 0, -29)), // 30 days including today
			End:   endOfDay(now),
		}
		previous = DateRange{
			Start: startOfDay(now.AddDate(0, 0, -59)), // Previous 30 days
			End:   endOfDay(now.AddDate(0, 0, -30)),
		}

	case PeriodPresetCustom:
		if customStart != nil && customEnd != nil {
			current = DateRange{
				Start: startOfDay(*customStart),
				End:   endOfDay(*customEnd),
			}
			// Calculate previous period with same duration
			duration := customEnd.Sub(*customStart)
			previous = DateRange{
				Start: startOfDay(customStart.Add(-duration - 24*time.Hour)),
				End:   endOfDay(customStart.Add(-24 * time.Hour)),
			}
		} else {
			// Default to this month if custom dates not provided
			current = DateRange{
				Start: startOfMonth(now),
				End:   endOfDay(now),
			}
			previous = DateRange{
				Start: startOfMonth(now.AddDate(0, -1, 0)),
				End:   endOfMonth(now.AddDate(0, -1, 0)),
			}
		}

	default:
		// Default to this month
		current = DateRange{
			Start: startOfMonth(now),
			End:   endOfDay(now),
		}
		previous = DateRange{
			Start: startOfMonth(now.AddDate(0, -1, 0)),
			End:   endOfMonth(now.AddDate(0, -1, 0)),
		}
	}

	return current, previous
}

// Helper functions for date calculations

func startOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

func endOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 999999999, t.Location())
}

func startOfWeek(t time.Time, loc *time.Location) time.Time {
	weekday := int(t.Weekday())
	if weekday == 0 {
		weekday = 7 // Sunday = 7 (ISO week starts on Monday)
	}
	return startOfDay(t.AddDate(0, 0, -(weekday - 1)))
}

func endOfWeek(t time.Time, loc *time.Location) time.Time {
	return endOfDay(startOfWeek(t, loc).AddDate(0, 0, 6))
}

func startOfMonth(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
}

func endOfMonth(t time.Time) time.Time {
	return endOfDay(startOfMonth(t).AddDate(0, 1, -1))
}

func startOfQuarter(t time.Time) time.Time {
	quarter := (int(t.Month()) - 1) / 3
	return time.Date(t.Year(), time.Month(quarter*3+1), 1, 0, 0, 0, 0, t.Location())
}

func endOfQuarter(t time.Time) time.Time {
	return endOfDay(startOfQuarter(t).AddDate(0, 3, -1))
}

func startOfYear(t time.Time) time.Time {
	return time.Date(t.Year(), 1, 1, 0, 0, 0, 0, t.Location())
}

func endOfYear(t time.Time) time.Time {
	return time.Date(t.Year(), 12, 31, 23, 59, 59, 999999999, t.Location())
}
