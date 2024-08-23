package utils

import (
	"fmt"
	"time"
)

// Generates an age from a duration.
func RoundedAge(duration time.Duration) string {
	seconds := duration.Seconds()

	minutes := 60.0
	hour := 60 * minutes
	day := 24 * hour

	week := 7 * day
	month := 30 * day
	year := 12 * month

	if seconds/year >= 1 {
		return text(seconds/year, "yr", "yrs")
	} else if seconds/month >= 1 {
		return text(seconds/month, "mo", "mo")
	} else if seconds/week >= 1 {
		return text(seconds/week, "wk", "wks")
	} else if seconds/day >= 1 {
		return text(seconds/day, "d", "d")
	} else if seconds/hour >= 1 {
		return text(seconds/hour, "h", "h")
	} else if seconds/minutes >= 1 {
		return text(seconds/minutes, "m", "m")
	}

	return text(seconds, "s", "s")
}

func text(value float64, singular string, plural string) string {
	suffix := singular
	if value >= 2 {
		suffix = plural
	}
	return fmt.Sprintf("%d %s", int(value), suffix)
}
