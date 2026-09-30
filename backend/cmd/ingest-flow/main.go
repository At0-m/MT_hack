package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"tramflow/internal/ingest"
)

func run() error {
	position := flag.String("timestamp-position", "", "required: start or end of 15-minute interval")
	output := flag.String("output", "flow-import", "new output directory")
	allowPartial := flag.Bool("allow-partial", false, "retain partial hourly sums and explicitly report them")
	flag.Parse()
	if len(flag.Args()) == 0 {
		return fmt.Errorf("provide one or more route_flow CSV paths")
	}
	aggregator, err := ingest.New(*position)
	if err != nil {
		return err
	}
	inputs := []map[string]string{}
	for _, path := range flag.Args() {
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		hash := sha256.New()
		parseErr := aggregator.Add(io.TeeReader(file, hash))
		closeErr := file.Close()
		if parseErr != nil {
			return fmt.Errorf("%s: %w", filepath.Base(path), parseErr)
		}
		if closeErr != nil {
			return closeErr
		}
		inputs = append(inputs, map[string]string{"file": filepath.Base(path), "sha256": hex.EncodeToString(hash.Sum(nil))})
	}
	hours := aggregator.Hours()
	if len(hours) == 0 {
		return fmt.Errorf("empty inputs")
	}
	partial := []ingest.Hour{}
	var total int64
	for _, hour := range hours {
		total += hour.Boardings
		if hour.ObservedQuarters != 4 {
			partial = append(partial, hour)
		}
	}
	if len(partial) > 0 && !*allowPartial {
		return fmt.Errorf("%d partial hours; inspect source coverage or explicitly use --allow-partial", len(partial))
	}
	var labels bytes.Buffer
	if err := ingest.Write(&labels, hours); err != nil {
		return err
	}
	digest := sha256.Sum256(labels.Bytes())
	report := map[string]any{
		"pipeline_version":     "route-flow-hourly-v1",
		"timezone":             "Europe/Moscow",
		"timestamp_position":   *position,
		"target":               "sum of total boardings, not total plus ticket categories",
		"inputs":               inputs,
		"rows":                 aggregator.Rows,
		"hours":                len(hours),
		"total_boardings":      total,
		"partial_hours":        partial,
		"missing_hours_policy": "absent, never silently zero-filled",
		"labels_sha256":        hex.EncodeToString(digest[:]),
	}
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	// New directory only: previous import remains intact on rerun.
	if err := os.Mkdir(*output, 0755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(*output, "labels.csv"), labels.Bytes(), 0644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(*output, "report.json"), append(data, '\n'), 0644); err != nil {
		return err
	}
	fmt.Printf("Imported %d quarters -> %d hours, %d boardings; partial hours: %d\n", aggregator.Rows, len(hours), total, len(partial))
	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
