package engine

import (
	"fmt"
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
	if weather.Precipitation != nil && *weather.Precipitation > 0 {
		indicator.Variant = "rain"
		if weather.Temperature != nil && *weather.Temperature <= 0 {
			indicator.Variant = "snow"
		}
		indicator.Text = "В закреплённых погодных данных есть осадки."
	}
	return indicator
}

func fleetIndicator(metrics d.Metrics) d.Indicator {
	indicator := neutralIndicator("fleet", "Выпуск", "Выпуск неизвестен.")
	if metrics.MeanFleet.Value == nil {
		return indicator
	}
	indicator.Tone = "info"
	indicator.Icon = "fleet"
	indicator.Variant = "unknown"
	indicator.Text = fmt.Sprintf("Средний эффективный выпуск: %.2f. Это не число отправлений.", *metrics.MeanFleet.Value)
	if metrics.Required.Value != nil {
		switch {
		case *metrics.MeanFleet.Value < *metrics.Required.Value:
			indicator.Variant = "deficit"
			indicator.Tone = "warning"
		case *metrics.MeanFleet.Value > *metrics.Required.Value:
			indicator.Variant = "surplus"
		default:
			indicator.Variant = "balanced"
		}
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

func Indicators(reading d.RouteReading, desc d.Descriptor) []d.Indicator {
	indicators := []d.Indicator{
		weatherIndicator(reading.Weather),
		fleetIndicator(reading.Evaluated),
		peakIndicator(reading.Evaluated),
		neutralIndicator("event", "События", "Сведения о событиях не подключены."),
		neutralIndicator("calendar", "Календарь", "Календарные признаки относятся к закреплённому периоду."),
	}
	if desc.Overrides == nil {
		return indicators
	}
	factors := FactorMillis(desc.Overrides)
	for factor, position := range []int{0, 3, 4} {
		if factors[factor] == 1000 {
			continue
		}
		indicator := &indicators[position]
		indicator.Tone = "info"
		indicator.Icon = indicator.Key
		indicator.Variant = "unknown"
		indicator.Text = fmt.Sprintf("Ручной коэффициент %.3f применяется только внутри окна сценария.", float64(factors[factor])/1000)
	}
	return indicators
}

// Route-only results never duplicate route values as stop predictions.
func Enrich(response *d.Response) {
	decorate := func(reading *d.RouteReading) {
		reading.StopReadings = []d.StopReading{}
		reading.Indicators = Indicators(*reading, response.Descriptor)
	}
	for i := range response.Frames {
		for j := range response.Frames[i].Routes {
			decorate(&response.Frames[i].Routes[j])
		}
	}
	for i := range response.Totals {
		decorate(&response.Totals[i])
	}
}
