package report

import (
	"encoding/csv"
	"fmt"
	"os"
	"time"

	"github.com/txdywy/inice/internal/model"
)

// CSVRenderer outputs test results as CSV.
type CSVRenderer struct {
	w *csv.Writer
}

// NewCSVRenderer creates a CSV renderer.
func NewCSVRenderer() *CSVRenderer {
	return &CSVRenderer{w: csv.NewWriter(os.Stdout)}
}

func (r *CSVRenderer) RenderHeader(routerHost string, nodeCount int, coreType string, duration string) {
	// No header in CSV
}

func (r *CSVRenderer) RenderProgress(current, total int, nodeName, status string) {
	// No progress in CSV
}

func (r *CSVRenderer) RenderResults(results []model.TestResult) error {
	header := []string{
		"name", "type", "protocol", "address", "port",
		"latency_ms", "latency_class", "loss_pct",
		"exit_ip", "country", "isp", "hosting",
		"netflix", "chatgpt", "youtube", "disney", "bilibili",
		"udp_ok", "errors",
	}
	if err := r.w.Write(header); err != nil {
		return fmt.Errorf("write CSV header: %w", err)
	}

	for _, res := range results {
		latencyMs := ""
		if res.Latency.Avg > 0 {
			latencyMs = fmt.Sprintf("%.1f", float64(res.Latency.Avg)/float64(time.Millisecond))
		}
		lossPct := fmt.Sprintf("%.1f", res.Latency.Loss*100)
		hosting := ""
		if res.ExitIP.IP != "" {
			if res.ExitIP.Hosting {
				hosting = "yes"
			} else {
				hosting = "no"
			}
		}
		udpOk := "no"
		if res.UDPOK {
			udpOk = "yes"
		}

		row := []string{
			res.Node.Name,
			string(res.Node.Type),
			string(res.Node.Protocol),
			res.Node.Address,
			fmt.Sprintf("%d", res.Node.Port),
			latencyMs,
			string(res.Latency.Class),
			lossPct,
			res.ExitIP.IP,
			res.ExitIP.Country,
			res.ExitIP.ISP,
			hosting,
			res.Streaming.Netflix,
			res.Streaming.ChatGPT,
			res.Streaming.YouTube,
			res.Streaming.Disney,
			res.Streaming.Bilibili,
			udpOk,
			joinErrors(res.Errors),
		}
		if err := r.w.Write(row); err != nil {
			return fmt.Errorf("write CSV row: %w", err)
		}
	}

	r.w.Flush()
	return r.w.Error()
}

func (r *CSVRenderer) RenderSummary(results []model.TestResult) error {
	// Summary not rendered in CSV format
	return nil
}

func joinErrors(errs []string) string {
	if len(errs) == 0 {
		return ""
	}
	result := errs[0]
	for i := 1; i < len(errs); i++ {
		result += "; " + errs[i]
	}
	return result
}
