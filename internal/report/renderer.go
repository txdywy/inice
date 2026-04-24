package report

import (
	"fmt"

	"github.com/txdywy/inice/internal/model"
)

// Renderer defines how test results are displayed.
type Renderer interface {
	RenderHeader(routerHost string, nodeCount int, coreType string, duration string)
	RenderTableHeader()
	RenderRow(result model.TestResult)
	RenderSummary(results []model.TestResult) error
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

// LatencyIcon returns a status icon for a latency class.
func LatencyIcon(class model.LatencyClass) string {
	switch class {
	case model.LatencyExcellent:
		return "🟢"
	case model.LatencyGood:
		return "🟡"
	case model.LatencyModerate:
		return "🟠"
	case model.LatencyPoor:
		return "🔴"
	default:
		return "⚪"
	}
}

// BoolIcon returns a checkmark or cross for a boolean value.
func BoolIcon(ok bool) string {
	if ok {
		return "✅"
	}
	return "❌"
}

// YesNoIcon returns the appropriate icon for a YES/NO/MAYBE/ERROR string.
func YesNoIcon(status string) string {
	switch status {
	case "YES":
		return "✅"
	case "NO":
		return "❌"
	case "MAYBE":
		return "⚠️"
	default:
		return "❓"
	}
}

// Truncate cuts a string to max runes.
func Truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
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
