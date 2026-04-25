package report

import (
	"fmt"
	"sort"
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
	width := 242
	fmt.Println(strings.Repeat("─", width))
	fmt.Printf("  inice - PassWall2 Node Health Report\n")
	fmt.Printf("  Router: %s | Nodes: %d | Shadow Core: %s | Duration: %s\n", routerHost, nodeCount, coreType, duration)
	fmt.Println(strings.Repeat("─", width))
	fmt.Println()
}

func (r *StaticRenderer) RenderTableHeader() {
	fmt.Println(strings.Repeat("─", 242))
	// widths: NAME(32) TYPE(10) PROTO(10) ADDRESS(20) PORT(6) LATENCY(8) EXIT IP(16) GEO(4) SCORE(20) GOOGLE(8) NETFLIX(8) CHATGPT(8) GITHUB(8) YOUTUBE(8) TWITTER(8) TELEGRAM(9) INSTAGRAM(10) REDDIT(8) TWITCH(8) IP TYPE(9)
	fmt.Printf("%s %s %s %s %s %s %s %s %s %s %s %s %s %s %s %s %s %s %s %s\n",
		PadVisual("NAME", 32, true),
		PadVisual("TYPE", 10, true),
		PadVisual("PROTO", 10, true),
		PadVisual("ADDRESS", 20, true),
		PadVisual("PORT", 6, true),
		PadVisual("LATENCY", 8, true),
		PadVisual("EXIT IP", 16, true),
		PadVisual("GEO", 4, true),
		PadVisual("PRACTICAL SCORE", 20, true),
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
	fmt.Println(strings.Repeat("─", 242))
}

func calculateScore(res model.TestResult) (int, string) {
	if res.Latency.Avg == 0 && res.Latency.Loss >= 1.0 {
		return 0, ""
	}

	// 100 Points Practical Algorithm:
	// 80% - Multi-Site Performance (8 pts max per site x 10 sites)
	// 10% - IP quality (Resident Bonus)
	// 10% - Basic Stability (Packet loss penalty)
	
	score := 0.0
	
	// 1. Multi-Site Performance (80 points max)
	probes := []string{
		res.Streaming.Google, res.Streaming.Netflix, res.Streaming.ChatGPT, 
		res.Streaming.GitHub, res.Streaming.YouTube, res.Streaming.Twitter,
		res.Streaming.Telegram, res.Streaming.Instagram, res.Streaming.Reddit, res.Streaming.Twitch,
	}
	
	for _, p := range probes {
		if strings.HasSuffix(p, "ms") {
			var ms int
			fmt.Sscanf(p, "%dms", &ms)
			
			// Base points for working
			siteScore := 5.0
			
			// Speed bonus for this specific site
			switch {
			case ms < 600:
				siteScore += 3.0 // Fast
			case ms < 1200:
				siteScore += 2.0 // Normal
			case ms < 2000:
				siteScore += 1.0 // Acceptable
			}
			score += siteScore
		} else if strings.HasPrefix(p, "MAYBE") {
			score += 2.0 // Pingable but not fully functional
		}
	}
	
	// 2. IP Quality (10 points max)
	if res.ExitIP.IP != "" && !res.ExitIP.Hosting {
		score += 10.0 // Resident IP
	}

	// 3. Stability Bonus (10 points max)
	if res.Latency.Loss < 0.1 {
		score += 10.0
	} else if res.Latency.Loss < 0.3 {
		score += 5.0
	}

	finalScore := int(score)
	if finalScore > 100 {
		finalScore = 100
	}

	trophies := 0
	switch {
	case finalScore >= 90:
		trophies = 5
	case finalScore >= 75:
		trophies = 4
	case finalScore >= 60:
		trophies = 3
	case finalScore >= 35:
		trophies = 2
	case finalScore >= 10:
		trophies = 1
	}

	scoreText := fmt.Sprintf("%d/100 %s", finalScore, strings.Repeat("🏆", trophies))
	return finalScore, scoreText
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

	_, scoreDisplay := calculateScore(res)

	fmt.Printf("%s %s %s %s %s %s %s %s %s %s %s %s %s %s %s %s %s %s %s %s\n",
		PadVisual(Truncate(res.Node.Name, 32), 32, true),
		PadVisual(Truncate(string(res.Node.Type), 10), 10, true),
		PadVisual(Truncate(string(res.Node.Protocol), 10), 10, true),
		PadVisual(Truncate(res.Node.Address, 20), 20, true),
		PadVisual(fmt.Sprintf("%d", res.Node.Port), 6, true),
		latencyStr,
		PadVisual(exitIPStr, 16, true),
		PadVisual(geoStr, 4, true),
		PadVisual(scoreDisplay, 20, true),
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
	fmt.Printf("🟢 Excellent (<90ms):   %d nodes\n", excellent)
	fmt.Printf("🟡 Good (90-150ms):     %d nodes\n", good)
	fmt.Printf("🟠 Moderate (150-250ms): %d nodes\n", moderate)
	fmt.Printf("🔴 Poor (>250ms/error):   %d nodes\n", poor)
	fmt.Printf("\nDNS Leaks: %d/%d | UDP Support: %d/%d\n", dnsLeaks, len(results), udpOK, len(results))

	// Top 3 Nodes Selection
	validResults := make([]model.TestResult, 0)
	for _, res := range results {
		if res.Latency.Avg > 0 || res.ExitIP.IP != "" {
			validResults = append(validResults, res)
		}
	}

	if len(validResults) > 0 {
		// Sort by our new "Practical & Multi-site Performance" algorithm
		sort.Slice(validResults, func(i, j int) bool {
			si, _ := calculateScore(validResults[i])
			sj, _ := calculateScore(validResults[j])
			return si > sj // Higher score first
		})

		fmt.Println("\n🏆 TOP 3 PRIMARY NODES (Practical Comprehensive Ranking)")
		fmt.Println("────────────────────────────────────────────────────────────────────────")
		limit := 3
		if len(validResults) < limit {
			limit = len(validResults)
		}
		for i := 0; i < limit; i++ {
			res := validResults[i]
			medal := []string{"🥇", "🥈", "🥉"}[i]
			score, trophies := calculateScore(res)
			
			// Just use trophies without the score prefix for the summary trophies column
			justTrophies := ""
			if parts := strings.Split(trophies, " "); len(parts) > 1 {
				justTrophies = parts[1]
			} else {
				justTrophies = trophies
			}

			fmt.Printf("%s %s | %s | 评分: %s | Avg: %s | %s %s\n", 
				medal, 
				PadVisual(Truncate(res.Node.Name, 32), 32, true),
				PadVisual(justTrophies, 11, true),
				PadVisual(fmt.Sprintf("%d/100", score), 7, false),
				PadVisual(fmt.Sprintf("%.0fms", float64(res.Latency.Avg)/float64(time.Millisecond)), 6, false),
				CountryToEmoji(res.ExitIP.Country),
				Truncate(res.ExitIP.ISP, 30),
			)
		}
	}

	return nil
}
