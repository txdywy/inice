package probes

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/txdywy/inice/internal/model"
)

const (
	latencyURL = "http://cp.cloudflare.com/generate_204"
)

// Latency runs HTTP HEAD latency probes through the SOCKS5 HTTP client.
func Latency(ctx context.Context, client *http.Client, warmup, measure int) (model.LatencyResult, error) {
	result := model.LatencyResult{}

	// Warmup probes
	for i := 0; i < warmup; i++ {
		doHead(ctx, client, latencyURL)
	}

	// Measurement probes
	var latencies []time.Duration
	var mu sync.Mutex
	var failures int

	for i := 0; i < measure; i++ {
		d, err := doHead(ctx, client, latencyURL)
		if err != nil {
			failures++
			continue
		}
		mu.Lock()
		latencies = append(latencies, d)
		mu.Unlock()
	}

	if len(latencies) == 0 {
		return result, fmt.Errorf("all %d measurements failed (%d warmup skipped)", measure, warmup)
	}

	// Compute statistics
	sort.Slice(latencies, func(i, j int) bool { return latencies[i] < latencies[j] })

	n := len(latencies)
	result.Min = latencies[0]
	result.Max = latencies[n-1]
	result.Loss = float64(failures) / float64(measure+failures)

	var sum time.Duration
	for _, d := range latencies {
		sum += d
	}
	result.Avg = time.Duration(sum.Nanoseconds() / int64(n))

	if n%2 == 0 {
		result.Median = time.Duration((latencies[n/2-1] + latencies[n/2]).Nanoseconds() / 2)
	} else {
		result.Median = latencies[n/2]
	}

	result.Class = model.ClassifyLatency(result.Avg)
	return result, nil
}

// doHead performs a HEAD request and returns the round-trip duration.
func doHead(ctx context.Context, client *http.Client, url string) (time.Duration, error) {
	req, err := http.NewRequestWithContext(ctx, "HEAD", url, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Cache-Control", "no-store")

	start := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	resp.Body.Close()
	return time.Since(start), nil
}
