package domain

type Position struct {
	Longitude float64 `json:"longitude"`
	Latitude  float64 `json:"latitude"`
}
type Indicator struct {
	Variant string `json:"variant"`
	Key     string `json:"key"`
	Tone    string `json:"tone"`
	Icon    string `json:"icon"`
	Title   string `json:"title"`
	Text    string `json:"text"`
}

type StopReading struct {
	RouteID   string   `json:"route_id"`
	Pattern   string   `json:"route_pattern_id"`
	RouteStop string   `json:"route_stop_id"`
	StopID    string   `json:"stop_id"`
	Sequence  int      `json:"sequence"`
	Method    string   `json:"prediction_method"`
	Typical   Reading  `json:"typical_boardings"`
	Relative  *float64 `json:"relative_to_typical"`
	Baseline  Metrics  `json:"baseline"`
	Evaluated Metrics  `json:"evaluated"`
	Delta     Delta    `json:"delta"`
}

type StopColumns struct {
	DataAnchor     string  `json:"data_anchor"`
	ValueMetric    string  `json:"value_metric"`
	ApproachEffect string  `json:"approach_effect"`
	ApproachLength float64 `json:"approach_length_m"`
	Profile        string  `json:"profile"`
}
type Focus struct {
	Kind    string `json:"kind"`
	Route   string `json:"route_id,omitempty"`
	Pattern string `json:"route_pattern_id,omitempty"`
	Stop    string `json:"route_stop_id,omitempty"`
}
type SummaryQuery struct {
	Calculation Descriptor `json:"calculation"`
	Expected    string     `json:"expected_calculation_id"`
	Focus       Focus      `json:"focus"`
}
type Fact struct {
	Key  string `json:"key"`
	Text string `json:"text"`
}
type Summary struct {
	ID     string `json:"calculation_id"`
	Status string `json:"status"`
	Focus  Focus  `json:"focus"`
	Window Window `json:"window"`
	Text   string `json:"text,omitempty"`
	Facts  []Fact `json:"facts,omitempty"`
	Reason string `json:"reason,omitempty"`
}
