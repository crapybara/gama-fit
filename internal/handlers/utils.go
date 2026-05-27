package handlers

import (
	"time"
)

func todayDate() string {
	return time.Now().Format("2006-01-02")
}

func formatTime12h(value string) string {
	layouts := []string{"15:04", "15:04:05"}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, value); err == nil {
			return t.Format("03:04 PM")
		}
	}
	return value
}

func sqliteDayLabel(dateStr string) string {
	if len(dateStr) >= 10 {
		if t, err := time.Parse("2006-01-02", dateStr[:10]); err == nil {
			return t.Format("Mon")
		}
	}
	return dateStr
}
