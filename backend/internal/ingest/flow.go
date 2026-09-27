// Package ingest normalizes already aggregated route flow, not validator events.
package ingest

import (
	"encoding/csv"
	"fmt"
	"io"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

var moscow = time.FixedZone("Europe/Moscow", 10800)
var routeName = regexp.MustCompile(`^([1-9][0-9]*)(?:\s+трамвай)?$`)
var ticketColumns = []string{"ediny", "troyka", "koshelek", "bank_cards", "lgotnye", "vedomstvennye", "tat", "other"}

type Hour struct {
	Route            int       `json:"route"`
	Time             time.Time `json:"time"`
	Boardings        int64     `json:"boardings"`
	ObservedQuarters int       `json:"observed_quarters"`
}

type Aggregator struct {
	position string
	seen     map[string]bool
	hours    map[string]*Hour
	Rows     int `json:"rows"`
}

func New(position string) (*Aggregator, error) {
	if position != "start" && position != "end" {
		return nil, fmt.Errorf("timestamp-position must be start or end")
	}
	return &Aggregator{position: position, seen: map[string]bool{}, hours: map[string]*Hour{}}, nil
}

func count(raw string) (int64, error) {
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil || math.IsNaN(value) || math.IsInf(value, 0) || value < 0 || value > 1e9 || value != math.Trunc(value) {
		return 0, fmt.Errorf("invalid boarding count %q", raw)
	}
	return int64(value), nil
}

func (a *Aggregator) Add(reader io.Reader) error {
	r := csv.NewReader(reader)
	header, err := r.Read()
	if err != nil {
		return err
	}
	columns := map[string]int{}
	for i, name := range header {
		name = strings.TrimPrefix(name, "\ufeff")
		if _, exists := columns[name]; exists {
			return fmt.Errorf("duplicate CSV column %s", name)
		}
		columns[name] = i
	}
	for _, name := range append([]string{"route", "dt_15min", "total"}, ticketColumns...) {
		if _, ok := columns[name]; !ok {
			return fmt.Errorf("missing column %s", name)
		}
	}
	for line := 2; ; line++ {
		row, err := r.Read()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return fmt.Errorf("row %d: %w", line, err)
		}
		match := routeName.FindStringSubmatch(strings.TrimSpace(row[columns["route"]]))
		if match == nil {
			return fmt.Errorf("row %d: invalid route", line)
		}
		route, err := strconv.Atoi(match[1])
		if err != nil {
			return err
		}
		timestamp, err := time.ParseInLocation("2006-01-02 15:04:05", row[columns["dt_15min"]], moscow)
		if err != nil || timestamp.Minute()%15 != 0 || timestamp.Second() != 0 {
			return fmt.Errorf("row %d: invalid quarter-hour timestamp", line)
		}
		key := fmt.Sprintf("%d/%s", route, timestamp.Format(time.RFC3339))
		if a.seen[key] {
			return fmt.Errorf("row %d: duplicate route/quarter %s (including across files)", line, key)
		}
		total, err := count(row[columns["total"]])
		if err != nil {
			return fmt.Errorf("row %d total: %w", line, err)
		}
		var sum int64
		for _, name := range ticketColumns {
			value, err := count(row[columns[name]])
			if err != nil {
				return fmt.Errorf("row %d %s: %w", line, name, err)
			}
			sum += value
		}
		if total != sum {
			return fmt.Errorf("row %d: total %d differs from ticket sum %d", line, total, sum)
		}
		a.seen[key] = true
		if a.position == "end" {
			timestamp = timestamp.Add(-time.Nanosecond)
		}
		hourTime := timestamp.Truncate(time.Hour)
		hourKey := fmt.Sprintf("%d/%s", route, hourTime.Format(time.RFC3339))
		hour := a.hours[hourKey]
		if hour == nil {
			hour = &Hour{Route: route, Time: hourTime}
			a.hours[hourKey] = hour
		}
		hour.Boardings += total
		hour.ObservedQuarters++
		a.Rows++
	}
}

func (a *Aggregator) Hours() []Hour {
	result := make([]Hour, 0, len(a.hours))
	for _, hour := range a.hours {
		result = append(result, *hour)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Route != result[j].Route {
			return result[i].Route < result[j].Route
		}
		return result[i].Time.Before(result[j].Time)
	})
	return result
}

// Write emits observed hourly sums, never fabricated zero-filled hours.
// Coverage stays in the report so partial hours cannot masquerade as complete.
func Write(writer io.Writer, hours []Hour) error {
	w := csv.NewWriter(writer)
	w.Comma = ';'
	if err := w.Write([]string{"route", "date", "hour", "boardings"}); err != nil {
		return err
	}
	for _, h := range hours {
		if err := w.Write([]string{strconv.Itoa(h.Route), h.Time.Format("2006-01-02"), strconv.Itoa(h.Time.Hour()), strconv.FormatInt(h.Boardings, 10)}); err != nil {
			return err
		}
	}
	w.Flush()
	return w.Error()
}
