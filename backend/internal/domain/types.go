package domain

import (
	"encoding/json"
	"fmt"
	"time"
)

type Window struct {
	From time.Time `json:"from"`
	To   time.Time `json:"to"`
}
type Selection struct {
	SnapshotID    string   `json:"forecast_snapshot_id"`
	RouteIDs      []string `json:"route_ids"`
	View          string   `json:"view_mode"`
	Window        Window   `json:"window"`
	Resolution    string   `json:"resolution"`
	SpatialDetail string   `json:"spatial_detail"`
}
type Factors struct {
	Weather *float64 `json:"weather,omitempty"`
	Event   *float64 `json:"event,omitempty"`
	Season  *float64 `json:"season,omitempty"`
}
type Fleet struct {
	Kind     string `json:"kind"`
	Absolute *int   `json:"vehicle_count,omitempty"`
	Delta    *int   `json:"vehicle_count_delta,omitempty"`
}
type Overrides struct {
	Window  Window   `json:"effective_window"`
	Fleet   *Fleet   `json:"fleet,omitempty"`
	Factors *Factors `json:"factors,omitempty"`
}
type Descriptor struct {
	Kind      string     `json:"kind"`
	Selection Selection  `json:"selection"`
	Overrides *Overrides `json:"overrides,omitempty"`
}
type Query struct {
	Selection Selection  `json:"selection"`
	Overrides *Overrides `json:"overrides,omitempty"`
}
type ExportRequest struct {
	Calculation Descriptor `json:"calculation"`
	Expected    string     `json:"expected_calculation_id"`
	Format      string     `json:"format"`
}
type Provenance struct {
	StopModel       string    `json:"stop_model_version,omitempty"`
	StopFeatures    string    `json:"stop_feature_schema_version,omitempty"`
	SnapshotID      string    `json:"forecast_snapshot_id"`
	Origin          time.Time `json:"forecast_origin_at"`
	CompleteThrough time.Time `json:"observations_complete_through"`
	Model           string    `json:"model_version"`
	Features        string    `json:"feature_schema_version"`
	History         string    `json:"history_version"`
	Weather         string    `json:"weather_snapshot_id"`
	Supply          string    `json:"supply_profile_version"`
	Reference       string    `json:"reference_version"`
	Policy          string    `json:"scenario_policy_version"`
	Network         string    `json:"network_version"`
	Mode            string    `json:"runtime_mode"`
}
type Coverage struct {
	RouteID     string   `json:"route_id"`
	Window      Window   `json:"window"`
	Resolutions []string `json:"resolutions"`
	Scopes      []string `json:"prediction_scopes"`
	StopStatus  string   `json:"stop_forecast_status"`
	MaxLead     int      `json:"max_lead_hours"`
	Status      string   `json:"status"`
}
type Notice struct {
	Code     string `json:"code"`
	Message  string `json:"message"`
	Severity string `json:"severity"`
}
type Snapshot struct {
	Provenance  Provenance `json:"provenance"`
	Created     time.Time  `json:"created_at"`
	Coverage    []Coverage `json:"coverage"`
	Views       []string   `json:"view_modes"`
	Quantile    float64    `json:"reference_quantile"`
	Target      float64    `json:"target_load_index"`
	Retention   int        `json:"retention_hours_after_deactivation"`
	Limitations []Notice   `json:"limitations"`
}
type Reading struct {
	Status string   `json:"status"`
	Value  *float64 `json:"value"`
	Reason string   `json:"reason,omitempty"`
}

func Available(v float64) Reading   { return Reading{Status: "available", Value: &v} }
func Missing(reason string) Reading { return Reading{Status: "unavailable", Reason: reason} }

type Metrics struct {
	Boardings    float64 `json:"boardings"`
	VehicleHours Reading `json:"vehicle_hours"`
	MeanFleet    Reading `json:"mean_vehicle_count"`
	PerVehicle   Reading `json:"boardings_per_vehicle_hour"`
	Reference    Reading `json:"reference_boardings"`
	Index        Reading `json:"load_index"`
	Level        string  `json:"load_level"`
	Required     Reading `json:"required_vehicle_count"`
	Source       string  `json:"fleet_source"`
	Proxy        bool    `json:"fleet_is_proxy"`
}
type WeatherPoint struct {
	Window        Window     `json:"window"`
	Mode          string     `json:"mode"`
	Temperature   *float64   `json:"temperature_c"`
	Precipitation *float64   `json:"precipitation_mm"`
	Code          *int       `json:"weather_code"`
	RunAt         *time.Time `json:"provider_run_at"`
	AvailableAt   *time.Time `json:"available_at"`
	Verified      bool       `json:"availability_verified"`
}
type WeatherSummary struct {
	Category      string   `json:"-"`
	Mode          string   `json:"mode"`
	Temperature   *float64 `json:"temperature_mean_c"`
	Precipitation *float64 `json:"precipitation_sum_mm"`
	Forecast      int      `json:"forecast_hours"`
	Climatology   int      `json:"climatology_hours"`
	Missing       int      `json:"missing_hours"`
}
type Delta struct {
	Boardings    float64  `json:"boardings"`
	Index        *float64 `json:"load_index"`
	VehicleHours *float64 `json:"vehicle_hours"`
}
type FleetAssessment struct {
	Variant      string
	DeficitHours int
	MaxDeficit   float64
}

type RouteReading struct {
	CalendarCategory string          `json:"-"`
	FleetAssessment  FleetAssessment `json:"-"`
	StopReadings     []StopReading   `json:"stop_readings"`
	Indicators       []Indicator     `json:"indicators"`
	RouteID          string          `json:"route_id"`
	Typical          Reading         `json:"typical_boardings"`
	Relative         *float64        `json:"relative_to_typical"`
	Baseline         Metrics         `json:"baseline"`
	Evaluated        Metrics         `json:"evaluated"`
	Delta            Delta           `json:"delta"`
	Weather          WeatherSummary  `json:"weather"`
}
type Frame struct {
	Window Window         `json:"window"`
	Routes []RouteReading `json:"routes"`
}
type Visualization struct {
	StopColumns StopColumns `json:"stop_columns"`
	Metric      string      `json:"metric"`
	Min         float64     `json:"scale_min"`
	Max         float64     `json:"scale_max"`
	Overflow    string      `json:"overflow"`
	Thresholds  []float64   `json:"thresholds"`
	Reference   string      `json:"reference_version"`
	Label       string      `json:"label"`
}
type Response struct {
	ID            string         `json:"calculation_id"`
	Descriptor    Descriptor     `json:"descriptor"`
	Provenance    Provenance     `json:"provenance"`
	Target        float64        `json:"target_load_index"`
	Frames        []Frame        `json:"frames"`
	Totals        []RouteReading `json:"totals"`
	Visualization Visualization  `json:"visualization"`
	Notices       []Notice       `json:"notices"`
}

// Prepared inputs are private. Availability is verified before publication.
type Hour struct {
	RouteID             string             `json:"route_id"`
	Time                time.Time          `json:"time"`
	Features            map[string]float64 `json:"features"`
	FeaturesAvailableAt time.Time          `json:"features_available_at"`
	Fleet               *float64           `json:"fleet"`
	Reference           *float64           `json:"reference"`
	Typical             *float64           `json:"typical"`
	Source              string             `json:"fleet_source"`
	Proxy               bool               `json:"fleet_is_proxy"`
	Weather             WeatherPoint       `json:"weather"`
	SyntheticBoardings  *float64           `json:"synthetic_boardings,omitempty"`
	Boardings           float64            `json:"-"`
}
type Model struct {
	Version        string    `json:"version"`
	Schema         string    `json:"feature_schema_version"`
	Path           string    `json:"path"`
	SHA256         string    `json:"sha256"`
	SchemaPath     string    `json:"schema_path"`
	SchemaSHA256   string    `json:"schema_sha256"`
	GoldenPath     string    `json:"golden_path"`
	GoldenSHA256   string    `json:"golden_sha256"`
	Input          string    `json:"input"`
	Output         string    `json:"output"`
	Columns        []string  `json:"columns"`
	Postprocessing string    `json:"postprocessing"`
	Release        string    `json:"release_status"`
	TrainCutoff    time.Time `json:"train_cutoff"`
}
type RouteImport struct {
	Detail   json.RawMessage `json:"detail"`
	Geometry json.RawMessage `json:"geometry"`
}
type WeatherSnapshot struct {
	ID          string            `json:"weather_snapshot_id"`
	Provider    string            `json:"provider"`
	SourceID    string            `json:"source_id"`
	RetrievedAt time.Time         `json:"retrieved_at"`
	Locations   map[string]string `json:"route_locations"`
}

type Bundle struct {
	WeatherMetadata  WeatherSnapshot `json:"weather_metadata"`
	ExpectedPrevious string          `json:"expected_previous_snapshot"`
	Snapshot         Snapshot        `json:"snapshot"`
	Model            Model           `json:"model"`
	Routes           []RouteImport   `json:"routes"`
	Hours            []Hour          `json:"hours"`
	Quality          json.RawMessage `json:"quality"`
	Sources          json.RawMessage `json:"sources"`
}
type Error struct {
	Status int
	Code   string
	Detail string
}

func (e *Error) Error() string                   { return fmt.Sprintf("%s: %s", e.Code, e.Detail) }
func Fail(status int, code, detail string) error { return &Error{status, code, detail} }
