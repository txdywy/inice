package probes

import (
	"context"
	"fmt"

	"github.com/txdywy/inice/internal/model"
)

// UDP tests SOCKS5 UDP ASSOCIATE connectivity.
// This is a stub for MVP - full implementation requires manual SOCKS5
// UDP ASSOCIATE handshake which is deferred to v2.
func UDP(ctx context.Context, proxyAddr string) (bool, string) {
	// TODO: Implement SOCKS5 UDP ASSOCIATE:
	// 1. Open TCP connection to SOCKS5 server
	// 2. Send UDP ASSOCIATE command (0x03)
	// 3. Receive relay address
	// 4. Send DNS query to 8.8.8.8:53 through relay
	// 5. Check for response
	return false, fmt.Sprintf("UDP test not yet implemented")
}

// ProbeResult holds the result of any probe.
type ProbeResult struct {
	Name   string
	OK     bool
	Error  string
	Detail interface{}
}

// CollectResults gathers probe results for a single node.
func CollectResults(result *model.TestResult) []ProbeResult {
	var results []ProbeResult

	results = append(results, ProbeResult{
		Name: "Latency",
		OK:   result.Latency.Class != model.LatencyPoor,
		Detail: fmt.Sprintf("avg=%s min=%s max=%s loss=%.0f%%",
			result.Latency.Avg, result.Latency.Min,
			result.Latency.Max, result.Latency.Loss*100),
	})

	results = append(results, ProbeResult{
		Name:   "Exit IP",
		OK:     result.ExitIP.IP != "",
		Detail: fmt.Sprintf("%s (%s, %s, %s)", result.ExitIP.IP, result.ExitIP.Country, result.ExitIP.City, result.ExitIP.ISP),
	})

	results = append(results, ProbeResult{
		Name:   "DNS Leak",
		OK:     !result.DNSLeak.LeakDetected,
		Detail: fmt.Sprintf("%d leaks", result.DNSLeak.LeakCount),
	})

	results = append(results, ProbeResult{
		Name: "Streaming",
		OK:   result.Streaming.Netflix == "YES",
		Detail: fmt.Sprintf("Netflix=%s ChatGPT=%s YouTube=%s",
			result.Streaming.Netflix, result.Streaming.ChatGPT, result.Streaming.YouTube),
	})

	results = append(results, ProbeResult{
		Name:   "UDP",
		OK:     result.UDPOK,
		Detail: result.UDPError,
	})

	return results
}
