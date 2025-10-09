package workflow

import (
	"os"
	"strings"
	"testing"
)

func TestParseClaudeLogNewFormatFile(t *testing.T) {
	// Test the new format from file
	content, err := os.ReadFile("test_data/sample_claude_log.txt")
	if err != nil {
		t.Fatalf("Failed to read test file: %v", err)
	}

	engine := NewClaudeEngine()
	metrics := engine.ParseLogMetrics(string(content), true)

	// Verify parsing worked correctly
	errorCount := CountErrors(metrics.Errors)
	t.Logf("Parsed metrics: Tokens=%d, Cost=%.6f, Turns=%d, Errors=%d",
		metrics.TokenUsage, metrics.EstimatedCost, metrics.Turns, errorCount)

	// Should extract the correct final result metrics
	if metrics.TokenUsage == 0 {
		t.Error("Expected non-zero token usage")
	}
	if metrics.EstimatedCost == 0 {
		t.Error("Expected non-zero cost")
	}
	if metrics.Turns == 0 {
		t.Error("Expected non-zero turns")
	}

	// Should count the [ERROR] line in the debug logs
	if errorCount == 0 {
		t.Error("Expected at least one error from debug logs")
	}
}

func TestParseClaudeLogNewFormatJSScriptFromFile(t *testing.T) {
	// Test the JavaScript parser with the new format
	script := GetLogParserScript("parse_claude_log")
	if script == "" {
		t.Skip("parse_claude_log script not available")
	}

	content, err := os.ReadFile("test_data/sample_claude_log.txt")
	if err != nil {
		t.Fatalf("Failed to read test file: %v", err)
	}

	result, err := runJSLogParser(script, string(content))
	if err != nil {
		t.Fatalf("Failed to parse new format Claude log: %v", err)
	}

	// Verify essential sections are present
	if !strings.Contains(result, "🚀 Initialization") {
		t.Error("Expected new format Claude log output to contain Initialization section")
	}
	if !strings.Contains(result, "🤖 Commands and Tools") {
		t.Error("Expected new format Claude log output to contain Commands and Tools section")
	}
	if !strings.Contains(result, "Total Cost") {
		t.Error("Expected new format Claude log output to contain cost information")
	}
	if !strings.Contains(result, "29d324d8-1a92-43c6-8740-babc2875a1d6") {
		t.Error("Expected new format Claude log output to contain session ID")
	}
	if !strings.Contains(result, "safe_outputs::missing-tool") {
		t.Error("Expected new format Claude log output to contain MCP tool call")
	}

	t.Logf("JavaScript parser output looks correct with proper sections")
}
