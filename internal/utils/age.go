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
		return text(seconds/year, "year")
	} else if seconds/month >= 1 {
		return text(seconds/month, "month")
	} else if seconds/week >= 1 {
		return text(seconds/week, "week")
	} else if seconds/day >= 1 {
		return text(seconds/day, "day")
	} else if seconds/hour >= 1 {
		return text(seconds/hour, "hour")
	} else if seconds/minutes >= 1 {
		return text(seconds/minutes, "minute")
	}

	return text(seconds, "second")
}

func text(value float64, suffix string) string {
	if value >= 2 {
		suffix += "s"
	}
	return fmt.Sprintf("%d %s", int(value), suffix)
}
