package workflow

import (
	"github.com/github/gh-aw/pkg/logger"
)

var geminiLogsLog = logger.New("workflow:gemini_logs")

// ParseLogMetrics parses Gemini CLI log output and extracts metrics.
// Gemini CLI outputs a single JSON response when using --output-format json.
// We parse the last valid JSON line (most complete response) and aggregate stats.
func (e *GeminiEngine) ParseLogMetrics(logContent string, verbose bool) LogMetrics {
	return parseStatsJSONLMetrics(logContent, verbose, "Gemini", geminiLogsLog)
}

// GetLogParserScriptId returns the script ID for parsing Gemini logs
func (e *GeminiEngine) GetLogParserScriptId() string {
	return "parse_gemini_log"
}
