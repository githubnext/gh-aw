package workflow

import (
	"strings"
	"testing"
)

// BenchmarkCountErrorsAndWarningsWithPatterns benchmarks the optimized pattern matching
func BenchmarkCountErrorsAndWarningsWithPatterns(b *testing.B) {
	// Create a realistic log content with 1000 lines
	logLines := []string{
		"2024-01-01T10:00:00.123Z [ERROR] Connection failed to server",
		"2024-01-01T10:00:01.456Z [WARN] Retrying connection attempt",
		"npm ERR! Authentication failed with registry",
		"npm WARN deprecated package@1.0.0: Use @latest instead",
		"[2024-01-01T10:00:02] stream error: network timeout occurred",
		"Some random log line without errors",
		"Another info line for context",
		"copilot: error: Invalid token provided",
		"Warning: Feature will be deprecated in next version",
		"INFO: Processing started successfully",
	}

	// Create 1000 lines of log content
	var logBuilder strings.Builder
	for i := 0; i < 100; i++ {
		for _, line := range logLines {
			logBuilder.WriteString(line)
			logBuilder.WriteString("\n")
		}
	}
	logContent := logBuilder.String()

	// Define realistic error patterns
	patterns := []ErrorPattern{
		{
			ID:           "codex-stream-error",
			Pattern:      `\[(\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2})\]\s+stream\s+(error):\s+(.+)`,
			LevelGroup:   2,
			MessageGroup: 3,
			Description:  "Codex stream errors",
		},
		{
			ID:           "copilot-timestamped-error",
			Pattern:      `(\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{3}Z)\s+\[(ERROR)\]\s+(.+)`,
			LevelGroup:   2,
			MessageGroup: 3,
			Description:  "Copilot timestamped ERROR messages",
		},
		{
			ID:           "copilot-timestamped-warning",
			Pattern:      `(\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{3}Z)\s+\[(WARN|WARNING)\]\s+(.+)`,
			LevelGroup:   2,
			MessageGroup: 3,
			Description:  "Copilot timestamped WARNING messages",
		},
		{
			ID:           "npm-error",
			Pattern:      `npm (ERR!)(.+)`,
			LevelGroup:   1,
			MessageGroup: 2,
			Description:  "NPM errors",
		},
		{
			ID:           "npm-warning",
			Pattern:      `npm (WARN)(.+)`,
			LevelGroup:   1,
			MessageGroup: 2,
			Description:  "NPM warnings",
		},
		{
			ID:           "copilot-cli-error",
			Pattern:      `copilot:\s+(error):\s+(.+)`,
			LevelGroup:   1,
			MessageGroup: 2,
			Description:  "Copilot CLI errors",
		},
		{
			ID:           "generic-warning",
			Pattern:      `(Warning):\s+(.+)`,
			LevelGroup:   1,
			MessageGroup: 2,
			Description:  "Generic warnings",
		},
		{
			ID:           "generic-error",
			Pattern:      `(.*)error(.*)`,
			LevelGroup:   0,
			MessageGroup: 0,
			Description:  "Generic error pattern",
		},
		{
			ID:           "generic-warning-pattern",
			Pattern:      `(.*)warning(.*)`,
			LevelGroup:   0,
			MessageGroup: 0,
			Description:  "Generic warning pattern",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = CountErrorsAndWarningsWithPatterns(logContent, patterns)
	}
}

// BenchmarkCountErrorsSmallLog benchmarks with a small log (typical single workflow run)
func BenchmarkCountErrorsSmallLog(b *testing.B) {
	logContent := `2024-01-01T10:00:00.123Z [ERROR] Connection failed
2024-01-01T10:00:01.456Z [WARN] Retrying connection
npm ERR! Authentication failed
npm WARN deprecated package@1.0.0
[2024-01-01T10:00:02] stream error: timeout`

	patterns := []ErrorPattern{
		{
			ID:           "copilot-error",
			Pattern:      `(\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{3}Z)\s+\[(ERROR)\]\s+(.+)`,
			LevelGroup:   2,
			MessageGroup: 3,
		},
		{
			ID:           "npm-error",
			Pattern:      `npm (ERR!)(.+)`,
			LevelGroup:   1,
			MessageGroup: 2,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = CountErrorsAndWarningsWithPatterns(logContent, patterns)
	}
}
