package workflow

import (
	"strings"
	"testing"
)

func TestParseClaudeLogNewFormat(t *testing.T) {
	// Test with the new format that includes debug entries and JSON
	newFormatLog := `npm warn exec The following package was not found and will be installed: @anthropic-ai/claude-code@1.0.115
[DEBUG] Watching for changes in setting files /tmp/gh-aw/.claude/settings.json...
[ERROR] Failed to save config with lock: Error: ENOENT: no such file or directory, lstat '/home/runner/.claude.json'
[DEBUG] Writing to temp file: /home/runner/.claude.json.tmp.2123.1757985980850
[DEBUG] Temp file written successfully, size: 103 bytes
[DEBUG] Renaming /home/runner/.claude.json.tmp.2123.1757985980850 to /home/runner/.claude.json
[DEBUG] File /home/runner/.claude.json written atomically
{"type":"system","subtype":"init","cwd":"/home/runner/work/gh-aw/gh-aw","session_id":"15b818fc-d93c-45e7-b7f2-89bad9ba54f7","tools":["Task","Bash","Read"],"model":"claude-sonnet-4-20250514"}
[DEBUG] Stream started - received first chunk
{"type":"assistant","message":{"content":[{"type":"text","text":"I'll help you with this task."},{"type":"tool_use","id":"tool_123","name":"Bash","input":{"command":"echo 'Hello World'"}}]}}
[DEBUG] Stream ended
{"type":"user","message":{"content":[{"type":"tool_result","tool_use_id":"tool_123","content":"Hello World"}]}}
{"type":"result","total_cost_usd":0.0015,"usage":{"input_tokens":150,"output_tokens":50},"num_turns":1}`

	engine := NewClaudeEngine()

	metrics := engine.ParseLogMetrics(newFormatLog, true)

	// Verify that metrics were extracted correctly
	if metrics.TokenUsage != 200 {
		t.Errorf("Expected token usage 200 (150+50), got %d", metrics.TokenUsage)
	}
	if metrics.EstimatedCost != 0.0015 {
		t.Errorf("Expected cost 0.0015, got %f", metrics.EstimatedCost)
	}
	if metrics.Turns != 1 {
		t.Errorf("Expected 1 turn, got %d", metrics.Turns)
	}

	// Check that error count includes the [ERROR] line
	errorCount := CountErrors(metrics.Errors)
	if errorCount == 0 {
		t.Errorf("Expected at least 1 error (from [ERROR] line), got %d", errorCount)
	}
}

func TestParseClaudeLogNewFormatJSScript(t *testing.T) {
	// Test the JavaScript parser with the new format
	script := GetLogParserScript("parse_claude_log")
	if script == "" {
		t.Skip("parse_claude_log script not available")
	}

	// Test with new format log
	newFormatLog := `[DEBUG] Starting Claude Code CLI
{"type":"system","subtype":"init","session_id":"test-123","tools":["Bash","Read"],"model":"claude-sonnet-4-20250514"}
[DEBUG] Processing user prompt
{"type":"assistant","message":{"content":[{"type":"text","text":"I'll help you with this task."},{"type":"tool_use","id":"tool_123","name":"Bash","input":{"command":"echo 'Hello World'"}}]}}
[DEBUG] Executing bash command
{"type":"user","message":{"content":[{"type":"tool_result","tool_use_id":"tool_123","content":"Hello World"}]}}
[DEBUG] Workflow completed successfully
{"type":"result","total_cost_usd":0.0015,"usage":{"input_tokens":150,"output_tokens":50},"num_turns":1}`

	result, err := runJSLogParser(script, newFormatLog)
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
	if !strings.Contains(result, "echo 'Hello World'") {
		t.Error("Expected new format Claude log output to contain the bash command")
	}
	if !strings.Contains(result, "Total Cost") {
		t.Error("Expected new format Claude log output to contain cost information")
	}
	if !strings.Contains(result, "test-123") {
		t.Error("Expected new format Claude log output to contain session ID")
	}
}
