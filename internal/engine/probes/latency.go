package probes

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"time"

	"github.com/txdywy/inice/internal/model"
)

var latencyURLs = []string{
	"http://www.gstatic.com/generate_204",
	"http://cp.cloudflare.com/generate_204",
	"http://connect.rom.miui.com/generate_204",
	"http://connectivitycheck.gstatic.com/generate_204",
	"http://connectivitycheck.platform.hicloud.com/generate_204",
	"http://wifi.vivo.com.cn/generate_204",
	"http://edge-http.microsoft.com/captiveportal/generate_204",
}

// Latency runs HTTP HEAD latency probes through the SOCKS5 HTTP client.
func Latency(ctx context.Context, client *http.Client, warmup, measure int) (model.LatencyResult, error) {
	result := model.LatencyResult{}

	// Warmup probes - also determines the best URL to use
	bestURL := latencyURLs[0]
	var bestDuration time.Duration = 999 * time.Second

	for _, url := range latencyURLs {
		d, err := doHead(ctx, client, url)
		if err == nil && d < bestDuration {
			bestDuration = d
			bestURL = url
		}
	}

	if bestDuration == 999*time.Second {
		return result, fmt.Errorf("all warmup probes failed")
	}

	// Additional warmups with best URL
	for i := 1; i < warmup; i++ {
		doHead(ctx, client, bestURL)
	}

	// Measurement probes
	var latencies []time.Duration
	var failures int

	for i := 0; i < measure; i++ {
		d, err := doHead(ctx, client, bestURL)
		if err != nil {
			failures++
			continue
		}
		latencies = append(latencies, d)
	}

	if len(latencies) == 0 {
		return result, fmt.Errorf("all %d measurements failed", measure)
	}

	// Compute statistics
	sort.Slice(latencies, func(i, j int) bool { return latencies[i] < latencies[j] })

	n := len(latencies)
	result.Min = latencies[0]
	result.Max = latencies[n-1]
	result.Loss = float64(failures) / float64(measure)

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
