// Package mlsources prepares external inputs for offline ML inference.
package mlsources

import (
	"encoding/csv"
	"fmt"
	"io"
	"math"
	"strconv"
	"time"
)

var Moscow = time.FixedZone("Europe/Moscow", 3*60*60)

// OccupancyProfile repeats source-year values by Moscow month, day and hour.
// Values retain their original scale (the supplied CSV uses percentages).
type OccupancyProfile struct {
	Year    int
	Sources map[string]bool
	values  map[string]float64
}

func LoadOccupancy(reader io.Reader, sourceYear int) (*OccupancyProfile, error) {
	r := csv.NewReader(reader)
	header, err := r.Read()
	if err != nil {
		return nil, fmt.Errorf("occupancy header: %w", err)
	}
	columns := map[string]int{}
	for i, name := range header {
		columns[name] = i
	}
	timeColumn, hasTime := columns["time"]
	valueColumn, hasValue := columns["avg_occupancy_rate"]
	if !hasTime || !hasValue {
		return nil, fmt.Errorf("occupancy requires time and avg_occupancy_rate")
	}
	p := &OccupancyProfile{Year: sourceYear, Sources: map[string]bool{}, values: map[string]float64{}}
	for line := 2; ; line++ {
		row, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("occupancy row %d: %w", line, err)
		}
		timestamp, err := time.ParseInLocation("2006-01-02 15:04:05", row[timeColumn], Moscow)
		if err != nil {
			return nil, fmt.Errorf("occupancy row %d timestamp: %w", line, err)
		}
		if timestamp.Year() != sourceYear || timestamp.Minute() != 0 || timestamp.Second() != 0 {
			return nil, fmt.Errorf("occupancy row %d must be a whole hour in %d", line, sourceYear)
		}
		value, err := strconv.ParseFloat(row[valueColumn], 64)
		if err != nil || math.IsNaN(value) || math.IsInf(value, 0) || value < 0 {
			return nil, fmt.Errorf("occupancy row %d has invalid value", line)
		}
		key := timestamp.Format("01-02-15")
		if _, exists := p.values[key]; exists {
			return nil, fmt.Errorf("duplicate occupancy hour %s", timestamp)
		}
		p.values[key] = value
		if sourceColumn, ok := columns["source"]; ok {
			p.Sources[row[sourceColumn]] = true
		}
	}
	if len(p.values) == 0 {
		return nil, fmt.Errorf("empty occupancy profile")
	}
	return p, nil
}

func (p *OccupancyProfile) Value(timestamp time.Time) (float64, error) {
	timestamp = timestamp.In(Moscow)
	if timestamp.Minute() != 0 || timestamp.Second() != 0 || timestamp.Nanosecond() != 0 {
		return 0, fmt.Errorf("occupancy timestamp must be a whole hour")
	}
	// A non-leap reference year has no February 29: use February 28 at that hour.
	if timestamp.Month() == time.February && timestamp.Day() == 29 {
		if _, ok := p.values[timestamp.Format("01-02-15")]; !ok {
			timestamp = timestamp.AddDate(0, 0, -1)
		}
	}
	value, ok := p.values[timestamp.Format("01-02-15")]
	if !ok {
		return 0, fmt.Errorf("missing occupancy profile hour %s", timestamp.Format("01-02-15"))
	}
	return value, nil
}

// Features maps each lag/window hour separately, including across New Year.
func (p *OccupancyProfile) Features(timestamp time.Time) (map[string]float64, error) {
	current, err := p.Value(timestamp)
	if err != nil {
		return nil, err
	}
	features := map[string]float64{"occupancy_rate": current}
	for _, lag := range []int{1, 24, 168} {
		value, err := p.Value(timestamp.Add(-time.Duration(lag) * time.Hour))
		if err != nil {
			return nil, err
		}
		features[fmt.Sprintf("occupancy_lag_%d", lag)] = value
	}
	var sum float64
	for lag := 1; lag <= 168; lag++ {
		value, err := p.Value(timestamp.Add(-time.Duration(lag) * time.Hour))
		if err != nil {
			return nil, err
		}
		sum += value
		if lag == 24 {
			features["occupancy_roll_24"] = sum / 24
		}
	}
	features["occupancy_roll_168"] = sum / 168
	return features, nil
}
