package workflow

import "github.com/github/gh-aw/pkg/logger"

var antigravityLogsLog = logger.New("workflow:antigravity_logs")

// ParseLogMetrics parses Antigravity CLI log output and extracts metrics.
// Antigravity CLI outputs a single JSON response when using --output-format json.
// We parse the last valid JSON line (most complete response) and aggregate stats.
func (e *AntigravityEngine) ParseLogMetrics(logContent string, verbose bool) LogMetrics {
	return parseSingleJSONResponseLogMetrics(logContent, verbose, "Antigravity", antigravityLogsLog)
}

// GetLogParserScriptId returns the script ID for parsing Antigravity logs
func (e *AntigravityEngine) GetLogParserScriptId() string {
	return "parse_antigravity_log"
}

// GetLogFileForParsing returns the log file path for parsing
func (e *AntigravityEngine) GetLogFileForParsing() string {
	return "/tmp/gh-aw/agent-stdio.log"
}

// GetDefaultDetectionModel returns the default model for threat detection
// Antigravity does not specify a default detection model yet
func (e *AntigravityEngine) GetDefaultDetectionModel() string {
	return ""
}
