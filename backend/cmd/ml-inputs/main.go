// ml-inputs pins Open-Meteo weather and the repeating occupancy profile.
package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"tramflow/internal/mlsources"
)

func run() error {
	occupancy := flag.String("occupancy", "occupancy_2025_popular_only.csv", "source-year hourly occupancy CSV")
	sourceYear := flag.Int("occupancy-year", 2025, "reference year inside CSV")
	days := flag.Int("days", 7, "Open-Meteo forecast days, 1..16")
	latitude := flag.Float64("latitude", 55.78, "latitude")
	longitude := flag.Float64("longitude", 37.58, "longitude")
	output := flag.String("output", "ml/generated/external-inputs.json", "new output snapshot file")
	flag.Parse()
	source, err := os.Open(*occupancy)
	if err != nil {
		return err
	}
	defer source.Close()
	hash := sha256.New()
	profile, err := mlsources.LoadOccupancy(io.TeeReader(source, hash), *sourceYear)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	weather, err := (mlsources.MeteoClient{}).Fetch(ctx, *latitude, *longitude, *days)
	if err != nil {
		return err
	}
	rows := make([]map[string]any, 0, len(weather.Hours))
	for _, hour := range weather.Hours {
		features, err := profile.Features(hour.Time)
		if err != nil {
			return err
		}
		for name, value := range hour.Features {
			features[name] = value
		}
		rows = append(rows, map[string]any{
			"time":     hour.Time,
			"features": features,
		})
	}
	bundle := map[string]any{
		"status":  "external_inputs_only_not_forecast_bundle",
		"weather": weather,
		"occupancy": map[string]any{
			"reference_year": profile.Year,
			"sources":        profile.Sources,
			"sha256":         hex.EncodeToString(hash.Sum(nil)),
			"mapping":        "same Moscow month/day/hour; Feb29 uses Feb28 when missing",
			"scale":          "unchanged",
		},
		"hours": rows,
	}
	data, err := json.MarshalIndent(bundle, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(*output), 0755); err != nil {
		return err
	}
	file, err := os.OpenFile(*output, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return err
	}
	_, writeErr := file.Write(append(data, '\n'))
	closeErr := file.Close()
	if writeErr != nil {
		return writeErr
	}
	if closeErr != nil {
		return closeErr
	}
	fmt.Printf("Saved %d hourly external inputs to %s\n", len(rows), *output)
	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
