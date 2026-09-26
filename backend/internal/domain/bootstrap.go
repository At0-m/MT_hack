package domain

import "encoding/json"

type Bootstrap struct {
	APIVersion   string            `json:"api_version"`
	Timezone     string            `json:"display_timezone"`
	Locale       string            `json:"locale"`
	Snapshot     Snapshot          `json:"active_snapshot"`
	Routes       []json.RawMessage `json:"routes"`
	Selection    Selection         `json:"default_selection"`
	Map          MapConfiguration  `json:"map"`
	Limits       Limits            `json:"limits"`
	Capabilities Capabilities      `json:"capabilities"`
}

type MapConfiguration struct {
	Renderer     string   `json:"renderer"`
	StyleURL     string   `json:"style_url"`
	FallbackPath string   `json:"fallback_style_path"`
	Attribution  string   `json:"attribution"`
	Center       Position `json:"center"`
	Zoom         float64  `json:"zoom"`
}

type Limits struct {
	Routes              int `json:"max_routes"`
	WindowHours         int `json:"max_window_hours"`
	HourlyCells         int `json:"max_hourly_cells"`
	BodyBytes           int `json:"max_body_bytes"`
	ExportRows          int `json:"max_export_rows"`
	ExportBytes         int `json:"max_export_bytes"`
	TimeoutMS           int `json:"forecast_timeout_ms"`
	ConcurrentInference int `json:"max_concurrent_inference"`
	ResponseCells       int `json:"max_response_cells"`
	CustomDays          int `json:"max_custom_days"`
	StopCells           int `json:"max_stop_cells"`
}

type Capabilities struct {
	Scopes             []string `json:"forecast_scopes"`
	ScenarioFleet      bool     `json:"scenario_fleet"`
	ScenarioFactors    []string `json:"scenario_factors"`
	PhysicalWeather    bool     `json:"physical_weather_overrides"`
	CSV                bool     `json:"csv_export"`
	LiveIngestion      bool     `json:"live_ingestion"`
	Summary            bool     `json:"summary"`
	CustomPeriod       bool     `json:"custom_period"`
	SessionAuth        bool     `json:"session_auth"`
	StopForecasts      bool     `json:"stop_forecasts"`
	SegmentForecasts   bool     `json:"segment_forecasts"`
	DecorativeApproach bool     `json:"decorative_approach_effect"`
}
