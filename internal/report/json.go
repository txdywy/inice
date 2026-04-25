package report

import (
	"encoding/json"
	"fmt"

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

func (r *JSONRenderer) RenderTableHeader() {
	// JSONL doesn't need a table header, but we can print nothing
}

func (r *JSONRenderer) RenderRow(res model.TestResult, rank int) {
	// Map to jsonResult for clean JSON output
	out := jsonResult{
		Rank:          rank,
		Node:          res.Node,
		Latency:       res.Latency,
		ExitIP:        res.ExitIP,
		DNSLeak:       res.DNSLeak,
		Streaming:     res.Streaming,
		UDPOK:         res.UDPOK,
		UDPError:      res.UDPError,
		Errors:        res.Errors,
		TotalDuration: res.TotalDuration.String(),
	}

	data, err := json.Marshal(out)
	if err == nil {
		fmt.Println(string(data))
	}
}

func (r *JSONRenderer) RenderSummary(results []model.TestResult) error {
	// Summary is included in RenderResults for JSON
	return nil
}

// jsonResult is a serializable view of TestResult.
type jsonResult struct {
	Rank          int                   `json:"rank"`
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
