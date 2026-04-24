package report

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/txdywy/inice/internal/model"
)

// JSONRenderer outputs test results as JSON.
type JSONRenderer struct {
	headerPrinted bool
}

// NewJSONRenderer creates a JSON renderer.
func NewJSONRenderer() *JSONRenderer {
	return &JSONRenderer{}
}

func (r *JSONRenderer) RenderHeader(routerHost string, nodeCount int, coreType string, duration string) {
	// JSON output doesn't use header - everything goes in one object
}

func (r *JSONRenderer) RenderProgress(current, total int, nodeName, status string) {
	// JSON output doesn't show progress
}

func (r *JSONRenderer) RenderResults(results []model.TestResult) error {
	output := map[string]interface{}{
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"results":   results,
	}
	data, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal results: %w", err)
	}
	fmt.Println(string(data))
	return nil
}

func (r *JSONRenderer) RenderSummary(results []model.TestResult) error {
	// Summary is included in RenderResults for JSON
	return nil
}

// jsonResult is a serializable view of TestResult.
type jsonResult struct {
	Node          model.ProxyNode       `json:"node"`
	Latency       model.LatencyResult   `json:"latency"`
	ExitIP        model.IPInfo          `json:"exit_ip"`
	DNSLeak       model.DNSLeakResult   `json:"dns_leak"`
	Streaming     model.StreamingResult `json:"streaming"`
	UDPOK         bool                  `json:"udp_ok"`
	UDPError      string                `json:"udp_error,omitempty"`
	Errors        []string              `json:"errors,omitempty"`
	TotalDuration string                `json:"total_duration"`
}
