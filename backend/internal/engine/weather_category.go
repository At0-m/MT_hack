package engine

import d "tramflow/internal/domain"

// Categories use each provider WMO code, never averaged temperature/precipitation.
func weatherCategory(p d.WeatherPoint) string {
	if p.Mode == "missing" || p.Code == nil {
		return "unknown"
	}
	if (*p.Code >= 0 && *p.Code <= 3) && p.Temperature != nil {
		if *p.Temperature >= 30 {
			return "heat"
		}
		if *p.Temperature <= -15 {
			return "cold"
		}
	}
	switch *p.Code {
	case 0:
		return "clear"
	case 1, 2, 3:
		return "cloudy"
	case 51, 53, 55, 56, 57, 61, 63, 65, 66, 67, 80, 81, 82:
		return "rain"
	case 71, 73, 75, 77, 85, 86:
		return "snow"
	case 45, 48, 95, 96, 99:
		return "other"
	default:
		return "unknown"
	}
}
