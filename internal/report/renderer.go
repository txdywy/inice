package report

import (
	"fmt"
	"strings"

	"github.com/txdywy/inice/internal/model"
)

// Renderer defines how test results are displayed.
type Renderer interface {
	RenderHeader(routerHost string, nodeCount int, coreType string, duration string)
	RenderTableHeader()
	RenderRow(result model.TestResult)
	RenderSummary(results []model.TestResult) error
}

// VisualLength calculates the display width of a string (CJK = 2)
func VisualLength(s string) int {
	w := 0
	for _, r := range s {
		if r >= 0x1100 && (r <= 0x115f || r == 0x2329 || r == 0x232a ||
			(r >= 0x2e80 && r <= 0xa4cf && r != 0x303f) ||
			(r >= 0xac00 && r <= 0xd7a3) ||
			(r >= 0xf900 && r <= 0xfaff) ||
			(r >= 0xfe10 && r <= 0xfe19) ||
			(r >= 0xfe30 && r <= 0xfe6f) ||
			(r >= 0xff00 && r <= 0xff60) ||
			(r >= 0xffe0 && r <= 0xffe6) ||
			(r >= 0x20000 && r <= 0x2fffd) ||
			(r >= 0x30000 && r <= 0x3fffd)) {
			w += 2
		} else {
			w += 1
		}
	}
	return w
}

// PadVisual pads a string to a specific visual width
func PadVisual(s string, width int, left bool) string {
	visLen := VisualLength(s)
	if visLen >= width {
		return s
	}
	pad := strings.Repeat(" ", width-visLen)
	if left {
		return s + pad
	}
	return pad + s
}

// Truncate cuts a string to max visual width.
func Truncate(s string, max int) string {
	if VisualLength(s) <= max {
		return s
	}
	var sb strings.Builder
	w := 0
	for _, r := range s {
		rw := 1
		if VisualLength(string(r)) == 2 {
			rw = 2
		}
		if w+rw > max-3 {
			break
		}
		sb.WriteRune(r)
		w += rw
	}
	sb.WriteString("...")
	return sb.String()
}

// StreamingColorStr formats streaming results (latency or error state) and pads it with color-coding.
func StreamingColorStr(status string, width int) string {
	var color, text string
	if status == "ERROR" {
		color = "\033[31m" // red
		text = "ERR"
	} else if status == "NO" {
		color = "\033[31m" // red
		text = "NO"
	} else if strings.HasPrefix(status, "MAYBE") {
		color = "\033[33m" // yellow
		text = status
	} else if strings.HasSuffix(status, "ms") {
		text = status
		// Extract number to decide color
		var ms int
		fmt.Sscanf(status, "%dms", &ms)
		if ms < 400 {
			color = "\033[32m" // green (fast)
		} else if ms < 800 {
			color = "\033[33m" // yellow (moderate)
		} else {
			color = "\033[38;5;214m" // orange/red (slow)
		}
	} else {
		color = "\033[90m" // gray
		text = status
	}
	
	if VisualLength(text) > width {
		text = Truncate(text, width)
	}
	text = PadVisual(text, width, true)
	
	return color + text + "\033[0m"
}

// LatencyColor returns an ANSI color code for a latency class.
func LatencyColor(class model.LatencyClass) string {
	switch class {
	case model.LatencyExcellent:
		return "\033[32m" // green
	case model.LatencyGood:
		return "\033[33m" // yellow
	case model.LatencyModerate:
		return "\033[38;5;214m" // orange
	case model.LatencyPoor:
		return "\033[31m" // red
	default:
		return "\033[0m"
	}
}

// CountryToEmoji converts a 2-letter ISO country code to its corresponding emoji flag.
func CountryToEmoji(countryCode string) string {
	if len(countryCode) != 2 {
		return countryCode
	}
	code := strings.ToUpper(countryCode)
	
	// Check if they are valid A-Z characters
	if code[0] < 'A' || code[0] > 'Z' || code[1] < 'A' || code[1] > 'Z' {
		return countryCode
	}

	// Convert A-Z to regional indicator symbol letters
	// Regional Indicator Symbol Letter A is U+1F1E6
	// 'A' is U+0041
	const regionalIndicatorBase = 0x1F1E6 - 0x0041
	
	return string(rune(code[0])+regionalIndicatorBase) + string(rune(code[1])+regionalIndicatorBase)
}

// NewRenderer creates a renderer based on the format string.
func NewRenderer(format string) (Renderer, error) {
	switch format {
	case "table", "":
		return NewStaticRenderer(), nil
	case "json":
		return NewJSONRenderer(), nil
	case "csv":
		return NewCSVRenderer(), nil
	default:
		return nil, fmt.Errorf("unsupported format: %s (valid: table, json, csv)", format)
	}
}
