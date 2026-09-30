package engine

import (
	"time"
	d "tramflow/internal/domain"
)

const MaxCustomDays = 31

var Moscow = time.FixedZone("Europe/Moscow", 3*60*60)

// ValidateCalendar enforces local calendar boundaries before coverage checks.
// Moscow has no daylight-saving transitions in the supported forecast period.
func ValidateCalendar(q d.Selection) error {
	from := q.Window.From.In(Moscow)
	to := q.Window.To.In(Moscow)
	if from.Hour() != 0 || to.Hour() != 0 {
		return d.Fail(422, "DAY_ALIGNMENT", "Границы периода должны быть в полночь Москвы.")
	}
	var expected time.Time
	switch q.View {
	case "day":
		if q.Resolution != "hour" {
			return calendarError()
		}
		expected = from.AddDate(0, 0, 1)
	case "week":
		if q.Resolution != "day" {
			return calendarError()
		}
		expected = from.AddDate(0, 0, 7)
	case "month":
		if q.Resolution != "day" || from.Day() != 1 {
			return calendarError()
		}
		expected = from.AddDate(0, 1, 0)
	case "custom":
		if q.Resolution != "day" {
			return d.Fail(422, "CUSTOM_HOURLY_UNSUPPORTED", "Произвольный период поддерживает только суточную агрегацию.")
		}
		if to.Sub(from) > MaxCustomDays*24*time.Hour {
			return d.Fail(422, "WINDOW_TOO_LARGE", "Произвольный период превышает 31 день.")
		}
		return nil
	default:
		return calendarError()
	}
	if !to.Equal(expected) {
		return calendarError()
	}
	return nil
}

func calendarError() error {
	return d.Fail(422, "INVALID_CALENDAR_WINDOW", "День — 1 сутки/hour, неделя — 7 суток/day, месяц — календарный месяц/day.")
}

// ValidateWeatherWindow uses hourly coverage, independent of forecast view modes.
func ValidateWeatherWindow(route string, w d.Window, snap d.Snapshot) error {
	if err := ValidateWindow(w); err != nil {
		return err
	}
	if w.To.Sub(w.From) > 744*time.Hour {
		return d.Fail(422, "WINDOW_TOO_LARGE", "Погодное окно превышает 744 часа.")
	}
	for _, coverage := range snap.Coverage {
		if coverage.RouteID != route {
			continue
		}
		outside := w.From.Before(coverage.Window.From) || w.To.After(coverage.Window.To)
		outside = outside || w.From.Before(snap.Provenance.Origin)
		outside = outside || w.To.Sub(snap.Provenance.Origin) > time.Duration(coverage.MaxLead)*time.Hour
		if outside || !contains(coverage.Resolutions, "hour") {
			return d.Fail(422, "OUTSIDE_COVERAGE", "Погодное окно выходит за опубликованное покрытие.")
		}
		return nil
	}
	return d.Fail(422, "ROUTE_UNSUPPORTED", "Маршрут отсутствует в закреплённом прогнозе.")
}
