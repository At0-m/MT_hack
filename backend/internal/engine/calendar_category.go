package engine

import (
	"time"
	d "tramflow/internal/domain"
)

func calendarCategory(hours []d.Hour) string {
	category := ""
	for _, hour := range hours {
		current := "weekday"
		day := hour.Time.In(Moscow).Weekday()
		weekend := day == time.Saturday || day == time.Sunday
		if flag, ok := hour.Features["is_weekend"]; ok {
			weekend = flag == 1
		}
		if weekend {
			current = "weekend"
		}
		if hour.Features["is_holiday"] == 1 {
			current = "holiday"
		}
		if category == "" {
			category = current
		} else if category != current {
			return "unknown"
		}
	}
	return category
}
