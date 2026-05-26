package workflow

import "github.com/github/gh-aw/pkg/logger"

var geminiLogsLog = logger.New("workflow:gemini_logs")

// ParseLogMetrics parses Gemini CLI log output and extracts metrics.
// Gemini CLI outputs a single JSON response when using --output-format json.
// We parse the last valid JSON line (most complete response) and aggregate stats.
func (e *GeminiEngine) ParseLogMetrics(logContent string, verbose bool) LogMetrics {
	return parseSingleJSONResponseLogMetrics(logContent, verbose, "Gemini", geminiLogsLog)
}

// GetLogParserScriptId returns the script ID for parsing Gemini logs
func (e *GeminiEngine) GetLogParserScriptId() string {
	return "parse_gemini_log"
}

// GetLogFileForParsing returns the log file path for parsing
func (e *GeminiEngine) GetLogFileForParsing() string {
	return "/tmp/gh-aw/agent-stdio.log"
}

// GetDefaultDetectionModel returns the default model for threat detection
// Gemini does not specify a default detection model yet
func (e *GeminiEngine) GetDefaultDetectionModel() string {
	return ""
}
