package report

import (
	"fmt"
	"strings"
	"time"

	"github.com/txdywy/inice/internal/model"
)

// StaticRenderer prints results as a styled table using plain terminal output.
type StaticRenderer struct{}

// NewStaticRenderer creates a static table renderer.
func NewStaticRenderer() *StaticRenderer {
	return &StaticRenderer{}
}

func (r *StaticRenderer) RenderHeader(routerHost string, nodeCount int, coreType string, duration string) {
	width := 110
	fmt.Println(strings.Repeat("─", width))
	fmt.Printf("  inice - PassWall2 Node Health Report\n")
	fmt.Printf("  Router: %s | Nodes: %d | Shadow Core: %s | Duration: %s\n", routerHost, nodeCount, coreType, duration)
	fmt.Println(strings.Repeat("─", width))
	fmt.Println()
}

func (r *StaticRenderer) RenderProgress(current, total int, nodeName, status string) {
	bar := strings.Repeat("█", current) + strings.Repeat("░", total-current)
	fmt.Printf("\r[%s] %d/%d %s: %s", bar, current, total, nodeName, status)
}

func (r *StaticRenderer) RenderResults(results []model.TestResult) error {
	// Header
	fmt.Println(strings.Repeat("─", 120))
	fmt.Printf("%-16s %-10s %-10s %-20s %8s %-18s %-8s %-8s %9s\n",
		"NAME", "TYPE", "PROTO", "ADDRESS", "LATENCY", "EXIT IP", "NETFLIX", "CHATGPT", "IP TYPE")
	fmt.Println(strings.Repeat("─", 120))

	for _, r := range results {
		latencyStr := fmt.Sprintf("%s%.0fms\033[0m", LatencyColor(r.Latency.Class), float64(r.Latency.Avg)/float64(time.Millisecond))
		if r.Latency.Class == model.LatencyPoor && r.Latency.Avg == 0 {
			latencyStr = "\033[31mERROR\033[0m"
		}

		exitIPStr := Truncate(r.ExitIP.IP, 15)
		if exitIPStr == "" {
			exitIPStr = "-"
		}
		if r.ExitIP.Country != "" {
			exitIPStr += " " + r.ExitIP.Country
		}

		ipType := "RESIDENT"
		if r.ExitIP.Hosting {
			ipType = "HOSTING"
		}
		if r.ExitIP.IP == "" {
			ipType = "-"
		}

		fmt.Printf("%-16s %-10s %-10s %-20s %s %-18s %-8s %-8s %9s\n",
			Truncate(r.Node.Name, 16),
			Truncate(string(r.Node.Type), 10),
			Truncate(string(r.Node.Protocol), 10),
			Truncate(r.Node.Address, 20),
			latencyStr,
			Truncate(exitIPStr, 18),
			YesNoIcon(r.Streaming.Netflix),
			YesNoIcon(r.Streaming.ChatGPT),
			ipType,
		)
	}

	fmt.Println(strings.Repeat("─", 120))
	return nil
}

func (r *StaticRenderer) RenderSummary(results []model.TestResult) error {
	var excellent, good, moderate, poor int
	var dnsLeaks, udpOK int

	for _, result := range results {
		switch result.Latency.Class {
		case model.LatencyExcellent:
			excellent++
		case model.LatencyGood:
			good++
		case model.LatencyModerate:
			moderate++
		case model.LatencyPoor:
			poor++
		}
		if result.DNSLeak.LeakDetected {
			dnsLeaks++
		}
		if result.UDPOK {
			udpOK++
		}
	}

	fmt.Println("\nSUMMARY")
	fmt.Println("───────")
	fmt.Printf("🟢 Excellent (<90ms):  %d nodes\n", excellent)
	fmt.Printf("🟡 Good (90-150ms):    %d nodes\n", good)
	fmt.Printf("🟠 Moderate (150-250ms): %d nodes\n", moderate)
	fmt.Printf("🔴 Poor (>250ms/error):  %d nodes\n", poor)
	fmt.Printf("\nDNS Leaks: %d/%d | Streaming Loss: check above\n", dnsLeaks, len(results))

	return nil
}
