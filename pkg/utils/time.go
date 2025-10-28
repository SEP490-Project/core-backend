package utils

import "time"

const (
	TimeFormat string = "2006-01-02 15:04:05"
	DateFormat string = "2006-01-02"
	Timezone   string = "Asia/Ho_Chi_Minh"
)

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
	loc, _ := time.LoadLocation(timezone)

	return data.In(loc).Format(layout)
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
	loc, _ := time.LoadLocation(time.Local.String())
	parsedTime, err := time.ParseInLocation(layout, dateStr, loc)
	if err != nil {
		return nil, err
	}
	return &parsedTime, nil
}
