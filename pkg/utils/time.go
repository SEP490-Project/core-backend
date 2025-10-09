package utils

import "time"

const (
	TimeFormat string = "2006-01-02 15:04:05"
	DateFormat string = "2006-01-02"
	Timezone   string = "Asia/Bangkok"
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
