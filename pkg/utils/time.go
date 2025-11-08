package utils

import (
	"sync"
	"time"
)

const (
	TimezoneFormat        string = "2006-01-02T15:04:05Z07:00"
	TimestampStringFormat string = "20060102150405"
	TimeFormat            string = "2006-01-02 15:04:05"
	DateFormat            string = "2006-01-02"
	Timezone              string = "Asia/Ho_Chi_Minh"
)

var (
	dateTimeFormats = []string{
		// ISO 8601 / RFC3339 variants
		time.RFC3339,     // "2006-01-02T15:04:05Z07:00"
		time.RFC3339Nano, // "2006-01-02T15:04:05.999999999Z07:00"

		// Common date-time formats
		"2006-01-02 15:04:05",      // MySQL/Postgres default
		"2006-01-02 15:04",         // without seconds
		"2006-01-02",               // just date
		"2006/01/02 15:04:05",      // slash date-time
		"2006/01/02",               // slash date only
		"02/01/2006 15:04:05",      // European style (day/month/year)
		"02/01/2006 15:04",         // European, without seconds
		"02/01/2006",               // European, date only
		"02-01-2006",               // European, dash
		"2006.01.02 15:04:05",      // dotted date-time
		"2006.01.02",               // dotted date
		"Jan 2 2006 15:04:05",      // "Nov 8 2025 15:30:00"
		"Jan 2 2006",               // "Nov 8 2025"
		"02 Jan 2006 15:04:05",     // "08 Nov 2025 15:04:05"
		"02 Jan 2006",              // "08 Nov 2025"
		"January 2, 2006 15:04:05", // long English form
		"January 2, 2006",          // long English date
	}
	formatCache sync.Map
)

// GetFormattedCurrentTime returns the current time formatted according to the specified layout and timezone.
func GetFormattedCurrentTime(layout, timezone string) string {
	if layout == "" {
		layout = TimeFormat
	}
	if timezone == "" {
		timezone = Timezone
	}
	loc, _ := time.LoadLocation(timezone)
	return time.Now().In(loc).Format(layout)
}

func FormatTimeWithTimezone(data *time.Time, layout, timezone string) string {
	if data == nil {
		return ""
	}
	if layout == "" {
		layout = TimeFormat
	}
	if timezone == "" {
		timezone = Timezone
	}

	return data.In(time.Local).Format(layout)
}

func FormatLocalTime(data *time.Time, layout string) string {
	if layout == "" {
		layout = TimeFormat
	}
	locale := time.Local.String()
	return FormatTimeWithTimezone(data, layout, locale)
}

// ParseLocalTime parses a date string into a time.Time object in the local timezone.
func ParseLocalTime(dateStr, layout string) (*time.Time, error) {
	if layout == "" {
		layout = TimeFormat
	}
	parsedTime, err := time.ParseInLocation(layout, dateStr, time.Local)
	if err != nil {
		return nil, err
	}
	return &parsedTime, nil
}

// BestEffortParseLocalTime attempts to parse a date string using multiple known formats.
// It caches the successful formats for future use.
// If parsing fails for all formats, it returns the current local time.
func BestEffortParseLocalTime(dateStr string) *time.Time {
	// Check cache first
	if layout, ok := formatCache.Load(dateStr); ok {
		if t, err := time.ParseInLocation(layout.(string), dateStr, time.Local); err == nil {
			return &t
		}
	}

	// Try all known formats
	for _, layout := range dateTimeFormats {
		if t, err := time.ParseInLocation(layout, dateStr, time.Local); err == nil {
			formatCache.Store(dateStr, layout)
			return &t
		}
	}

	return nil
}

// ParseLocalTimeWithFallback tries to parse the date string with the specified layout first.
// If it fails, it falls back to BestEffortParseLocalTime.
func ParseLocalTimeWithFallback(dateStr, layout string) *time.Time {
	t, err := ParseLocalTime(dateStr, layout)
	if err != nil {
		t = BestEffortParseLocalTime(dateStr)
	}
	return t
}
