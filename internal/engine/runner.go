package engine

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/txdywy/inice/internal/engine/probes"
	"github.com/txdywy/inice/internal/model"
)

// Runner orchestrates concurrent health testing across all proxy nodes.
type Runner struct {
	routerIP string
	cfg      model.TestConfig
}

// NewRunner creates a test runner.
func NewRunner(routerIP string, cfg model.TestConfig) *Runner {
	return &Runner{
		routerIP: routerIP,
		cfg:      cfg,
	}
}

// RunTests tests all nodes concurrently and returns results in order.
func (r *Runner) RunTests(ctx context.Context, nodes []model.ProxyNode) []model.TestResult {
	results := make([]model.TestResult, len(nodes))
	sem := make(chan struct{}, r.cfg.Concurrency)
	var wg sync.WaitGroup

	for i, node := range nodes {
		wg.Add(1)
		go func(idx int, n model.ProxyNode) {
			defer wg.Done()

			// Acquire semaphore
			select {
			case sem <- struct{}{}:
			case <-ctx.Done():
				results[idx] = model.TestResult{
					Node:   n,
					Errors: []string{"cancelled"},
				}
				return
			}
			defer func() { <-sem }()

			results[idx] = r.testNode(ctx, n)
		}(i, node)
	}

	wg.Wait()
	return results
}

func (r *Runner) testNode(ctx context.Context, node model.ProxyNode) model.TestResult {
	start := time.Now()
	result := model.TestResult{Node: node}
	proxyAddr := fmt.Sprintf("%s:%d", r.routerIP, node.SOCKS5Port)

	// Create HTTP client through SOCKS5 proxy
	httpClient, err := NewHTTPClient(proxyAddr, r.cfg.Timeout)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("SOCKS5 client: %v", err))
		result.Latency.Class = model.LatencyPoor
		result.TotalDuration = time.Since(start)
		return result
	}

	// 1. Latency probe
	latency, err := probes.Latency(ctx, httpClient, r.cfg.WarmupProbes, r.cfg.MeasurementProbes)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("latency: %v", err))
		result.Latency = model.LatencyResult{Class: model.LatencyPoor}
	} else {
		result.Latency = latency
	}

	// 2. Exit IP probe
	result.ExitIP = probes.ExitIP(ctx, httpClient)

	// 3. DNS leak probe (skipped for MVP - requires raw SOCKS5 dialer)

	// 4. Streaming probe
	result.Streaming = probes.Streaming(ctx, httpClient)

	// 5. IP purity probe (use exit IP)
	if result.ExitIP.IP != "" {
		hosting, err := probes.Purity(ctx, httpClient, result.ExitIP.IP)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("purity: %v", err))
		}
		result.ExitIP.Hosting = hosting
	}

	// 6. UDP probe (stub)
	result.UDPOK, result.UDPError = probes.UDP(ctx, proxyAddr)

	result.TotalDuration = time.Since(start)
	return result
}
