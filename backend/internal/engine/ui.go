package engine

import (
	"fmt"
	"time"
	d "tramflow/internal/domain"
)

func neutralIndicator(key, title, text string) d.Indicator {
	return d.Indicator{
		Key:     key,
		Variant: "none",
		Tone:    "neutral",
		Icon:    "none",
		Title:   title,
		Text:    text,
	}
}

func weatherIndicator(weather d.WeatherSummary) d.Indicator {
	indicator := neutralIndicator("weather", "Погода", "Погодные данные отсутствуют.")
	if weather.Mode == "missing" {
		return indicator
	}
	indicator.Tone = "info"
	indicator.Icon = "weather"
	indicator.Variant = "unknown"
	indicator.Text = "Использована погода из закреплённой версии."
	indicator.Variant = weather.Category
	if indicator.Variant == "" {
		indicator.Variant = "unknown"
	}

	return indicator
}

func fleetIndicator(reading d.RouteReading) d.Indicator {
	metrics := reading.Evaluated
	indicator := neutralIndicator("fleet", "Выпуск", "Выпуск неизвестен.")
	if metrics.MeanFleet.Value == nil {
		indicator.Variant = "unknown"
		indicator.Icon = "fleet"
		indicator.Tone = "info"
		return indicator
	}
	indicator.Tone = "info"
	indicator.Icon = "fleet"
	indicator.Variant = "unknown"
	indicator.Text = fmt.Sprintf("Средний эффективный выпуск: %.2f. Это не число отправлений.", *metrics.MeanFleet.Value)
	indicator.Variant = reading.FleetAssessment.Variant
	if indicator.Variant == "" {
		indicator.Variant = "unknown"
	}
	if indicator.Variant == "deficit" {
		indicator.Tone = "warning"
		indicator.Text = fmt.Sprintf("Недостаток выпуска в %d ч.; максимальный дефицит %.0f вагонов.", reading.FleetAssessment.DeficitHours, reading.FleetAssessment.MaxDeficit)
	} else if indicator.Variant == "unknown" {
		indicator.Text = "Недостаточно почасовых данных для проверки выпуска."
	}

	return indicator
}

func peakIndicator(metrics d.Metrics) d.Indicator {
	indicator := neutralIndicator("peak", "Интенсивность", "Исторический эталон недоступен.")
	if metrics.Index.Value == nil {
		return indicator
	}
	indicator.Icon = "peak"
	indicator.Variant = "off_peak"
	indicator.Tone = "positive"
	switch metrics.Level {
	case "elevated":
		indicator.Tone = "info"
	case "high":
		indicator.Tone = "warning"
		indicator.Variant = "peak"
	case "very_high":
		indicator.Tone = "critical"
		indicator.Variant = "peak"
	}
	indicator.Text = fmt.Sprintf("Индекс %.2f относительно исторического эталона; не заполненность салона.", *metrics.Index.Value)
	return indicator
}

// Trend uses a 5% band around the pinned typical profile.
func trendIndicator(reading d.RouteReading) d.Indicator {
	indicator := neutralIndicator("trend", "К типовому профилю", "Типовой профиль недоступен.")
	if reading.Relative == nil {
		return indicator
	}
	indicator.Icon, indicator.Tone, indicator.Variant = "trend", "info", "flat"
	if *reading.Relative > 0.05 {
		indicator.Variant = "up"
	}
	if *reading.Relative < -0.05 {
		indicator.Variant = "down"
	}
	indicator.Text = fmt.Sprintf("Отклонение от типового профиля: %.1f%%; полоса стабильности ±5%%.", *reading.Relative*100)
	return indicator
}

func calendarIndicator(window d.Window) d.Indicator {
	indicator := neutralIndicator("calendar", "Календарь", "Смешанный календарный период.")
	weekday, weekend := false, false
	for t := window.From; t.Before(window.To); t = t.Add(time.Hour) {
		day := t.In(Moscow).Weekday()
		if day == time.Saturday || day == time.Sunday {
			weekend = true
		} else {
			weekday = true
		}
	}
	indicator.Icon, indicator.Tone, indicator.Variant = "calendar", "info", "unknown"
	if weekday && !weekend {
		indicator.Variant = "weekday"
		indicator.Text = "Будние дни; государственные праздники отдельно не подтверждены."
	}
	if weekend && !weekday {
		indicator.Variant = "weekend"
		indicator.Text = "Суббота или воскресенье; переносы рабочих дней отдельно не подтверждены."
	}
	return indicator
}

func Indicators(reading d.RouteReading, desc d.Descriptor, window d.Window) []d.Indicator {
	indicators := []d.Indicator{
		weatherIndicator(reading.Weather), fleetIndicator(reading), trendIndicator(reading),
		peakIndicator(reading.Evaluated), calendarIndicator(window),
	}
	if reading.CalendarCategory == "holiday" {
		indicators[4].Variant = "holiday"
		indicators[4].Text = "Праздничный день по закреплённым календарным признакам."
	} else if reading.CalendarCategory == "unknown" {
		indicators[4].Variant = "unknown"
		indicators[4].Text = "Смешанный календарный период."
	} else if reading.CalendarCategory != "" {
		indicators[4].Variant = reading.CalendarCategory
	}
	if desc.Overrides == nil {
		return indicators
	}
	effective := desc.Overrides.Window
	if !window.From.Before(effective.To) || !effective.From.Before(window.To) {
		return indicators
	}
	factors := FactorMillis(desc.Overrides)
	partial := window.From.Before(effective.From) || window.To.After(effective.To)
	for factor, position := range []int{0, 3, 4} {
		if factors[factor] == 1000 {
			continue
		}
		if factor == 1 {
			// Contract v1.2 has five slots. An active event has priority over peak;
			// numeric peak metrics remain present in frames and summary.
			indicators[position] = neutralIndicator("event", "События", "")
			indicators[position].Variant = "active"
		}
		indicator := &indicators[position]
		indicator.Tone = "info"
		indicator.Icon = indicator.Key
		scope := "в этом кадре"
		if partial {
			scope = "в части периода, пересекающей окно сценария"
		}
		indicator.Text += fmt.Sprintf(" Ручной коэффициент %.3f действует %s.", float64(factors[factor])/1000, scope)
	}
	return indicators
}

// Preserve supplied stop readings; route-only results remain explicitly empty.
func Enrich(response *d.Response) {
	decorate := func(reading *d.RouteReading, window d.Window) {
		if reading.StopReadings == nil {
			reading.StopReadings = []d.StopReading{}
		}
		reading.Indicators = Indicators(*reading, response.Descriptor, window)
	}
	for i := range response.Frames {
		for j := range response.Frames[i].Routes {
			decorate(&response.Frames[i].Routes[j], response.Frames[i].Window)
		}
	}
	for i := range response.Totals {
		decorate(&response.Totals[i], response.Descriptor.Selection.Window)
	}
}
