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
func (r *Runner) RunTests(ctx context.Context, nodes []model.ProxyNode, onComplete func(idx int, total int, res model.TestResult)) []model.TestResult {
	results := make([]model.TestResult, len(nodes))
	sem := make(chan struct{}, r.cfg.Concurrency)
	var wg sync.WaitGroup
	var completed int
	var mu sync.Mutex

	for i, node := range nodes {
		wg.Add(1)
		go func(idx int, n model.ProxyNode) {
			defer wg.Done()

			// Acquire semaphore
			select {
			case sem <- struct{}{}:
			case <-ctx.Done():
				res := model.TestResult{
					Node:   n,
					Errors: []string{"cancelled"},
				}
				results[idx] = res
				mu.Lock()
				completed++
				if onComplete != nil {
					onComplete(idx, len(nodes), res)
				}
				mu.Unlock()
				return
			}

			res := r.testNode(ctx, n)
			results[idx] = res
			
			<-sem

			mu.Lock()
			completed++
			if onComplete != nil {
				onComplete(idx, len(nodes), res)
			}
			mu.Unlock()
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

	// 2 & 4. Exit IP and Streaming probes run concurrently to save time
	var exitIP model.IPInfo
	var streaming model.StreamingResult
	var wgProbe sync.WaitGroup
	wgProbe.Add(2)

	go func() {
		defer wgProbe.Done()
		exitIP = probes.ExitIP(ctx, httpClient)
	}()

	go func() {
		defer wgProbe.Done()
		streaming = probes.Streaming(ctx, httpClient)
	}()

	wgProbe.Wait()
	result.ExitIP = exitIP
	result.Streaming = streaming

	// 3. DNS leak probe (skipped for MVP)

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
