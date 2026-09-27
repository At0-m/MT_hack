//go:build !linux

package telemetry

import "io"

// Linux process/cgroup counters are unavailable on other operating systems.
// Runtime metrics are still exported; missing counters are not fabricated.
func processMetrics(_ io.Writer) {}
