// Package telemetry exposes bounded, dependency-free Prometheus text metrics.
// Metric names/labels are supplied only by internal call sites, never raw URLs,
// usernames, snapshot IDs or request IDs.
package telemetry

import (
	"fmt"
	"io"
	"net/http"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Labels map[string]string
type histogram struct {
	sum     float64
	count   uint64
	buckets []uint64
}
type Registry struct {
	mu       sync.Mutex
	counters map[string]float64
	hist     map[string]*histogram
	gauges   map[string]func() float64
}

var bounds = []float64{.001, .005, .01, .025, .05, .1, .2, .3, .5, 1, 2, 5, 10, 30, 60}
var Default = New()

func New() *Registry {
	return &Registry{counters: map[string]float64{}, hist: map[string]*histogram{}, gauges: map[string]func() float64{}}
}
func labelString(labels Labels) string {
	if len(labels) == 0 {
		return ""
	}
	keys := make([]string, 0, len(labels))
	for key := range labels {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	values := make([]string, 0, len(keys))
	for _, key := range keys {
		values = append(values, key+"="+strconv.Quote(labels[key]))
	}
	return "{" + strings.Join(values, ",") + "}"
}
func (r *Registry) Add(name string, labels Labels, value float64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.counters[name+labelString(labels)] += value
}
func (r *Registry) Gauge(name string, get func() float64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.gauges[name] = get
}
func (r *Registry) Observe(name string, labels Labels, duration time.Duration) {
	value := duration.Seconds()
	r.mu.Lock()
	defer r.mu.Unlock()
	key := name + labelString(labels)
	h := r.hist[key]
	if h == nil {
		h = &histogram{buckets: make([]uint64, len(bounds))}
		r.hist[key] = h
	}
	h.sum += value
	h.count++
	for i, b := range bounds {
		if value <= b {
			h.buckets[i]++
		}
	}
}
func Timer(name string, labels Labels) func() {
	start := time.Now()
	return func() { Default.Observe(name, labels, time.Since(start)) }
}
func (r *Registry) HTTP(method, endpoint string, status int, elapsed time.Duration) {
	switch method {
	case "GET", "POST", "OPTIONS", "HEAD", "PUT", "DELETE", "PATCH":
	default:
		method = "OTHER"
	}
	if endpoint == "" {
		endpoint = "unmatched"
	}
	if status < 100 || status > 599 {
		status = 500
	}
	labels := Labels{"method": method, "endpoint": endpoint, "status": strconv.Itoa(status)}
	r.Add("http_requests_total", labels, 1)
	r.Observe("http_request_duration_seconds", labels, elapsed)
}
func sampleKey(key, suffix string) string {
	if i := strings.IndexByte(key, '{'); i >= 0 {
		return key[:i] + suffix + key[i:]
	}
	return key + suffix
}
func bucketKey(key, le string) string {
	if strings.HasSuffix(key, "}") {
		return key[:len(key)-1] + ",le=" + strconv.Quote(le) + "}"
	}
	return key + "{le=" + strconv.Quote(le) + "}"
}
func (r *Registry) Write(w io.Writer) {
	// Copy state before I/O or callbacks; a slow scrape must not hold the hot lock.
	r.mu.Lock()
	c := make(map[string]float64, len(r.counters))
	for k, v := range r.counters {
		c[k] = v
	}
	h := map[string]histogram{}
	for k, v := range r.hist {
		copy := *v
		copy.buckets = append([]uint64(nil), v.buckets...)
		h[k] = copy
	}
	g := map[string]func() float64{}
	for k, v := range r.gauges {
		g[k] = v
	}
	r.mu.Unlock()
	keys := make([]string, 0, len(c))
	for k := range c {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Fprintf(w, "%s %g\n", k, c[k])
	}
	keys = keys[:0]
	for k := range h {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		v := h[k]
		for i, b := range bounds {
			fmt.Fprintf(w, "%s %d\n", bucketKey(sampleKey(k, "_bucket"), strconv.FormatFloat(b, 'g', -1, 64)), v.buckets[i])
		}
		fmt.Fprintf(w, "%s %d\n%s %g\n%s %d\n", bucketKey(sampleKey(k, "_bucket"), "+Inf"), v.count, sampleKey(k, "_sum"), v.sum, sampleKey(k, "_count"), v.count)
	}
	keys = keys[:0]
	for k := range g {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Fprintf(w, "%s %g\n", k, g[k]())
	}
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	fmt.Fprintf(w, "go_goroutines %d\ngo_memstats_heap_alloc_bytes %d\ngo_memstats_alloc_bytes %d\ngo_gc_pause_seconds_total %g\n", runtime.NumGoroutine(), mem.HeapAlloc, mem.Alloc, float64(mem.PauseTotalNs)/1e9)
	processMetrics(w)
}
func (r *Registry) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		r.Write(w)
	})
}
