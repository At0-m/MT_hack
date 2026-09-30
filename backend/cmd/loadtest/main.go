package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"os"
	"runtime"
	"sort"
	"sync"
	"time"
)

func main() {
	base := flag.String("url", "http://localhost:8080", "API origin")
	n := flag.Int("n", 3000, "Request count")
	concurrency := flag.Int("c", 16, "Workers (1..32)")
	output := flag.String("output", "loadtest-results.json", "JSON report path")
	flag.Parse()
	if *n < 1 || *concurrency < 1 || *concurrency > 32 {
		panic("invalid n/c")
	}
	user, password := os.Getenv("API_USER"), os.Getenv("API_PASSWORD")
	if user == "" || password == "" {
		panic("API_USER/API_PASSWORD are required")
	}
	client := &http.Client{Timeout: 15 * time.Second, Transport: &http.Transport{MaxIdleConns: 64, MaxIdleConnsPerHost: 32}}
	jar, _ := cookiejar.New(nil)
	client.Jar = jar
	loginPayload, _ := json.Marshal(map[string]string{"username": user, "password": password})
	login, _ := http.NewRequest("POST", *base+"/api/v1/auth/login", bytes.NewReader(loginPayload))
	login.Header.Set("Content-Type", "application/json")
	loginResponse, err := client.Do(login)
	if err != nil {
		panic(err)
	}
	_, _ = io.Copy(io.Discard, loginResponse.Body)
	loginResponse.Body.Close()
	if loginResponse.StatusCode != 200 {
		panic(fmt.Sprintf("login failed: status %d", loginResponse.StatusCode))
	}
	get, _ := http.NewRequest("GET", *base+"/api/v1/bootstrap", nil)
	response, err := client.Do(get)
	if err != nil {
		panic(err)
	}
	b, err := io.ReadAll(io.LimitReader(response.Body, 2*1024*1024))
	response.Body.Close()
	if err != nil || response.StatusCode != 200 {
		panic(fmt.Sprintf("bootstrap failed: status %d", response.StatusCode))
	}
	var bootstrap map[string]json.RawMessage
	if err = json.Unmarshal(b, &bootstrap); err != nil {
		panic(err)
	}
	payload, _ := json.Marshal(map[string]json.RawMessage{"selection": bootstrap["default_selection"]})
	latencies := make([]float64, *n)
	failures := make([]bool, *n)
	jobs := make(chan int)
	var wg sync.WaitGroup
	start := time.Now()
	for w := 0; w < *concurrency; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range jobs {
				request, _ := http.NewRequest("POST", *base+"/api/v1/forecasts/query", bytes.NewReader(payload))
				request.Header.Set("Content-Type", "application/json")
				begin := time.Now()
				res, err := client.Do(request)
				if err != nil {
					failures[i] = true
				} else {
					_, err = io.Copy(io.Discard, res.Body)
					res.Body.Close()
					failures[i] = err != nil || res.StatusCode != 200
				}
				latencies[i] = float64(time.Since(begin).Microseconds()) / 1000
			}
		}()
	}
	for i := 0; i < *n; i++ {
		jobs <- i
	}
	close(jobs)
	wg.Wait()
	elapsed := time.Since(start).Seconds()
	sort.Float64s(latencies)
	failed := 0
	for _, v := range failures {
		if v {
			failed++
		}
	}
	percentile := func(p float64) float64 { return latencies[int(float64(*n-1)*p)] }
	report := map[string]any{
		"timestamp":       time.Now().UTC(),
		"target":          *base,
		"requests":        *n,
		"concurrency":     *concurrency,
		"errors":          failed,
		"elapsed_seconds": elapsed,
		"rps":             float64(*n) / elapsed,
		"p50_ms":          percentile(.5),
		"p95_ms":          percentile(.95),
		"p99_ms":          percentile(.99),
		"client_go":       runtime.Version(),
		"client_os":       runtime.GOOS,
		"client_arch":     runtime.GOARCH,
		"client_cpus":     runtime.NumCPU(),
		"selection":       json.RawMessage(payload),
		"notes":           "Full HTTP endpoint; record server/DB CPU and total process/native RSS separately. Synthetic fixture results do not establish ML quality.",
	}
	b, _ = json.MarshalIndent(report, "", "  ")
	if err = os.WriteFile(*output, b, 0644); err != nil {
		panic(err)
	}
	fmt.Printf("requests=%d errors=%d RPS=%.1f p95=%.3fms\n", *n, failed, float64(*n)/elapsed, percentile(.95))
	if failed > 0 {
		os.Exit(1)
	}
}
