//go:build linux

package telemetry

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"syscall"
)

func processMetrics(w io.Writer) {
	var usage syscall.Rusage
	if syscall.Getrusage(syscall.RUSAGE_SELF, &usage) == nil {
		cpu := float64(usage.Utime.Sec+usage.Stime.Sec) + float64(usage.Utime.Usec+usage.Stime.Usec)/1e6
		fmt.Fprintf(w, "process_cpu_seconds_total %g\n", cpu)
	}
	if data, err := os.ReadFile("/proc/self/statm"); err == nil {
		fields := strings.Fields(string(data))
		if len(fields) > 1 {
			pages, e := strconv.ParseUint(fields[1], 10, 64)
			if e == nil {
				fmt.Fprintf(w, "process_resident_memory_bytes %d\n", pages*uint64(os.Getpagesize()))
			}
		}
	}
	if data, err := os.ReadFile("/proc/self/status"); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			f := strings.Fields(line)
			if len(f) >= 2 && f[0] == "VmSwap:" {
				if kb, e := strconv.ParseUint(f[1], 10, 64); e == nil {
					fmt.Fprintf(w, "process_swap_bytes %d\n", kb*1024)
				}
			}
		}
	}
	if data, err := os.ReadFile("/sys/fs/cgroup/memory.swap.current"); err == nil {
		if bytes, e := strconv.ParseUint(strings.TrimSpace(string(data)), 10, 64); e == nil {
			fmt.Fprintf(w, "container_swap_current_bytes %d\n", bytes)
		}
	}
}
