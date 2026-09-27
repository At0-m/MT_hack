package mlsources

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

var WeatherColumns = []string{"temperature_2m", "rain", "snowfall", "snow_depth", "wind_speed_10m", "wind_speed_100m"}

type WeatherHour struct {
	Time     time.Time          `json:"time"`
	Features map[string]float64 `json:"features"`
}

type WeatherSnapshot struct {
	Provider    string            `json:"provider"`
	Endpoint    string            `json:"endpoint"`
	Latitude    float64           `json:"latitude"`
	Longitude   float64           `json:"longitude"`
	RetrievedAt time.Time         `json:"retrieved_at"`
	Units       map[string]string `json:"units"`
	Hours       []WeatherHour     `json:"hours"`
}

type MeteoClient struct {
	HTTP     *http.Client
	Endpoint string
}

// Fetch returns one pinned hourly forecast; it never substitutes old-year CSV.
// RetrievedAt is our reception time, not the provider's model issue time.
func (c MeteoClient) Fetch(ctx context.Context, latitude, longitude float64, days int) (WeatherSnapshot, error) {
	var result WeatherSnapshot
	if math.IsNaN(latitude) || math.IsInf(latitude, 0) || latitude < -90 || latitude > 90 ||
		math.IsNaN(longitude) || math.IsInf(longitude, 0) || longitude < -180 || longitude > 180 || days < 1 || days > 16 {
		return result, fmt.Errorf("invalid location or forecast days (1..16)")
	}
	endpoint := c.Endpoint
	if endpoint == "" {
		endpoint = "https://api.open-meteo.com/v1/gfs"
	}
	u, err := url.Parse(endpoint)
	if err != nil {
		return result, err
	}
	query := u.Query()
	query.Set("latitude", strconv.FormatFloat(latitude, 'f', -1, 64))
	query.Set("longitude", strconv.FormatFloat(longitude, 'f', -1, 64))
	query.Set("hourly", strings.Join(WeatherColumns, ","))
	query.Set("timezone", "Europe/Moscow")
	query.Set("temperature_unit", "celsius")
	query.Set("wind_speed_unit", "kmh")
	query.Set("precipitation_unit", "mm")
	query.Set("forecast_days", strconv.Itoa(days))
	u.RawQuery = query.Encode()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return result, err
	}
	client := c.HTTP
	if client == nil {
		client = &http.Client{Timeout: 20 * time.Second}
	}
	response, err := client.Do(request)
	if err != nil {
		return result, fmt.Errorf("Open-Meteo request: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return result, fmt.Errorf("Open-Meteo HTTP %d", response.StatusCode)
	}
	const maxBytes = 4 * 1024 * 1024
	body, err := io.ReadAll(io.LimitReader(response.Body, maxBytes+1))
	if err != nil {
		return result, err
	}
	if len(body) > maxBytes {
		return result, fmt.Errorf("Open-Meteo response too large")
	}
	var payload struct {
		Error    bool                       `json:"error"`
		Reason   string                     `json:"reason"`
		Timezone string                     `json:"timezone"`
		Offset   int                        `json:"utc_offset_seconds"`
		Units    map[string]string          `json:"hourly_units"`
		Hourly   map[string]json.RawMessage `json:"hourly"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return result, fmt.Errorf("Open-Meteo JSON: %w", err)
	}
	if payload.Error {
		return result, fmt.Errorf("Open-Meteo: %s", payload.Reason)
	}
	if payload.Timezone != "Europe/Moscow" || payload.Offset != 10800 {
		return result, fmt.Errorf("unexpected weather timezone")
	}
	expectedUnits := map[string]string{"temperature_2m": "°C", "rain": "mm", "snowfall": "cm", "snow_depth": "m", "wind_speed_10m": "km/h", "wind_speed_100m": "km/h"}
	var timestamps []string
	if err := json.Unmarshal(payload.Hourly["time"], &timestamps); err != nil || len(timestamps) == 0 {
		return result, fmt.Errorf("missing weather timestamps")
	}
	columns := map[string][]*float64{}
	for _, column := range WeatherColumns {
		if payload.Units[column] != expectedUnits[column] {
			return result, fmt.Errorf("unexpected weather unit for %s", column)
		}
		var values []*float64
		if err := json.Unmarshal(payload.Hourly[column], &values); err != nil || len(values) != len(timestamps) {
			return result, fmt.Errorf("missing/misaligned weather column %s", column)
		}
		columns[column] = values
	}
	result = WeatherSnapshot{Provider: "open-meteo", Endpoint: endpoint, Latitude: latitude, Longitude: longitude, RetrievedAt: time.Now().UTC(), Units: payload.Units}
	for i, rawTime := range timestamps {
		timestamp, err := time.ParseInLocation("2006-01-02T15:04", rawTime, Moscow)
		if err != nil || timestamp.Minute() != 0 {
			return WeatherSnapshot{}, fmt.Errorf("invalid weather hour %s", rawTime)
		}
		if i > 0 && !timestamp.Equal(result.Hours[i-1].Time.Add(time.Hour)) {
			return WeatherSnapshot{}, fmt.Errorf("weather hours are not contiguous")
		}
		features := map[string]float64{}
		for _, column := range WeatherColumns {
			value := columns[column][i]
			if value == nil || math.IsNaN(*value) || math.IsInf(*value, 0) {
				return WeatherSnapshot{}, fmt.Errorf("missing weather %s at %s", column, rawTime)
			}
			features[column] = *value
		}
		features["weather_is_real"] = 1 // Complete hourly API values, including forecasts.
		features["weather_is_partial"] = 0
		features["weather_is_climatology"] = 0
		features["is_rain"] = flag(features["rain"] > 0)
		features["is_snow"] = flag(features["snowfall"] > 0)
		features["temp_below_zero"] = flag(features["temperature_2m"] < 0)
		result.Hours = append(result.Hours, WeatherHour{Time: timestamp, Features: features})
	}
	return result, nil
}

func flag(value bool) float64 {
	if value {
		return 1
	}
	return 0
}
