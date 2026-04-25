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
	width := 216
	fmt.Println(strings.Repeat("─", width))
	fmt.Printf("  inice - PassWall2 Node Health Report\n")
	fmt.Printf("  Router: %s | Nodes: %d | Shadow Core: %s | Duration: %s\n", routerHost, nodeCount, coreType, duration)
	fmt.Println(strings.Repeat("─", width))
	fmt.Println()
}

func (r *StaticRenderer) RenderTableHeader() {
	fmt.Println(strings.Repeat("─", 216))
	// widths: NAME(32) TYPE(10) PROTO(10) ADDRESS(20) PORT(6) LATENCY(8) EXIT IP(16) GEO(4) GOOGLE(8) NETFLIX(8) CHATGPT(8) GITHUB(8) YOUTUBE(8) TWITTER(8) TELEGRAM(9) INSTAGRAM(10) REDDIT(8) TWITCH(8) IP TYPE(9)
	fmt.Printf("%s %s %s %s %s %s %s %s %s %s %s %s %s %s %s %s %s %s %s\n",
		PadVisual("NAME", 32, true),
		PadVisual("TYPE", 10, true),
		PadVisual("PROTO", 10, true),
		PadVisual("ADDRESS", 20, true),
		PadVisual("PORT", 6, true),
		PadVisual("LATENCY", 8, true),
		PadVisual("EXIT IP", 16, true),
		PadVisual("GEO", 4, true),
		PadVisual("GOOGLE", 8, true),
		PadVisual("NETFLIX", 8, true),
		PadVisual("CHATGPT", 8, true),
		PadVisual("GITHUB", 8, true),
		PadVisual("YOUTUBE", 8, true),
		PadVisual("TWITTER", 8, true),
		PadVisual("TELEGRAM", 9, true),
		PadVisual("INSTAGRAM", 10, true),
		PadVisual("REDDIT", 8, true),
		PadVisual("TWITCH", 8, true),
		PadVisual("IP TYPE", 9, false),
	)
	fmt.Println(strings.Repeat("─", 216))
}

func (r *StaticRenderer) RenderRow(res model.TestResult) {
	latencyText := fmt.Sprintf("%.0fms", float64(res.Latency.Avg)/float64(time.Millisecond))
	if res.Latency.Class == model.LatencyPoor && res.Latency.Avg == 0 {
		latencyText = "ERR"
	}
	latencyStr := LatencyColor(res.Latency.Class) + PadVisual(latencyText, 8, true) + "\033[0m"

	exitIPStr := Truncate(res.ExitIP.IP, 16)
	if exitIPStr == "" {
		exitIPStr = "-"
	}
	
	geoStr := "-"
	if res.ExitIP.Country != "" {
		geoStr = CountryToEmoji(res.ExitIP.Country)
	}

	ipType := "RESIDENT"
	if res.ExitIP.Hosting {
		ipType = "HOSTING"
	}
	if res.ExitIP.IP == "" {
		ipType = "-"
	}

	fmt.Printf("%s %s %s %s %s %s %s %s %s %s %s %s %s %s %s %s %s %s %s\n",
		PadVisual(Truncate(res.Node.Name, 32), 32, true),
		PadVisual(Truncate(string(res.Node.Type), 10), 10, true),
		PadVisual(Truncate(string(res.Node.Protocol), 10), 10, true),
		PadVisual(Truncate(res.Node.Address, 20), 20, true),
		PadVisual(fmt.Sprintf("%d", res.Node.Port), 6, true),
		latencyStr,
		PadVisual(exitIPStr, 16, true),
		PadVisual(geoStr, 4, true),
		StreamingColorStr(res.Streaming.Google, 8),
		StreamingColorStr(res.Streaming.Netflix, 8),
		StreamingColorStr(res.Streaming.ChatGPT, 8),
		StreamingColorStr(res.Streaming.GitHub, 8),
		StreamingColorStr(res.Streaming.YouTube, 8),
		StreamingColorStr(res.Streaming.Twitter, 8),
		StreamingColorStr(res.Streaming.Telegram, 9),
		StreamingColorStr(res.Streaming.Instagram, 10),
		StreamingColorStr(res.Streaming.Reddit, 8),
		StreamingColorStr(res.Streaming.Twitch, 8),
		PadVisual(ipType, 9, false),
	)
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
