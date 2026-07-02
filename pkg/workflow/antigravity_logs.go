package workflow

import (
	"github.com/github/gh-aw/pkg/logger"
)

var antigravityLogsLog = logger.New("workflow:antigravity_logs")

// ParseLogMetrics parses Antigravity CLI log output and extracts metrics.
// Antigravity CLI outputs a single JSON response when using --output-format json.
// We parse the last valid JSON line (most complete response) and aggregate stats.
func (e *AntigravityEngine) ParseLogMetrics(logContent string, verbose bool) LogMetrics {
	return parseStatsJSONLMetrics(logContent, verbose, "Antigravity", antigravityLogsLog)
}

// GetLogParserScriptId returns the script ID for parsing Antigravity logs
func (e *AntigravityEngine) GetLogParserScriptId() string {
	return "parse_antigravity_log"
}
