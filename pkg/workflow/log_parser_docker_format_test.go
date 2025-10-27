package workflow

import (
	"strings"
	"testing"
)

func TestParseClaudeLogDockerPullFormat(t *testing.T) {
	// Test with the complex format that includes docker pull output
	dockerPullLog := `npm warn exec The following package was not found and will be installed: @anthropic-ai/claude-code@1.0.115
[DEBUG] Watching for changes in setting files /tmp/gh-aw/.claude/settings.json...
[ERROR] Failed to save config with lock: Error: ENOENT: no such file or directory, lstat '/home/runner/.claude.json'
[ERROR] MCP server "github" Server stderr: Unable to find image 'ghcr.io/github/github-mcp-server:v0.20.0' locally
[DEBUG] Shell snapshot created successfully (242917 bytes)
[ERROR] MCP server "github" Server stderr: v0.20.0: Pulling from github/github-mcp-server
[ERROR] MCP server "github" Server stderr: 35d697fe2738: Pulling fs layer
[ERROR] MCP server "github" Server stderr: bfb59b82a9b6: Pulling fs layer
4eff9a62d888: Pulling fs layer
62de241dac5f: Pulling fs layer
a62778643d56: Pulling fs layer
[ERROR] MCP server "github" Server stderr: bfb59b82a9b6: Verifying Checksum
bfb59b82a9b6: Download complete
[ERROR] MCP server "github" Server stderr: 4eff9a62d888: Verifying Checksum
{"type":"system","subtype":"init","cwd":"/home/runner/work/gh-aw/gh-aw","session_id":"test-123","tools":["Bash","Read"],"model":"claude-sonnet-4-20250514"}
{"type":"assistant","message":{"content":[{"type":"text","text":"I'll help you with this task."},{"type":"tool_use","id":"tool_123","name":"Bash","input":{"command":"echo 'Hello World'"}}]}}
{"type":"user","message":{"content":[{"type":"tool_result","tool_use_id":"tool_123","content":"Hello World"}]}}
{"type":"result","total_cost_usd":0.0015,"usage":{"input_tokens":150,"output_tokens":50},"num_turns":1}`

	engine := NewClaudeEngine()

	// Test parsing with the complex docker format
	metrics := engine.ParseLogMetrics(dockerPullLog, true)

	// Should still extract the correct metrics
	if metrics.TokenUsage != 200 {
		t.Errorf("Expected token usage 200 (150+50), got %d", metrics.TokenUsage)
	}
	if metrics.EstimatedCost != 0.0015 {
		t.Errorf("Expected cost 0.0015, got %f", metrics.EstimatedCost)
	}
	if metrics.Turns != 1 {
		t.Errorf("Expected 1 turn, got %d", metrics.Turns)
	}

	// Should count all the error lines including MCP server stderr and the initial [ERROR]
	errorCount := CountErrors(metrics.Errors)
	if errorCount < 5 {
		t.Errorf("Expected at least 5 errors from various error lines, got %d", errorCount)
	}

	t.Logf("Successfully parsed complex docker log with %d errors, %d tokens, cost $%.6f, %d turns",
		errorCount, metrics.TokenUsage, metrics.EstimatedCost, metrics.Turns)
}

func TestParseClaudeLogDockerPullFormatJS(t *testing.T) {
	// Test the JavaScript parser with complex docker pull format
	script := GetLogParserScript("parse_claude_log")
	if script == "" {
		t.Skip("parse_claude_log script not available")
	}

	dockerPullLog := `[DEBUG] Starting Claude
[ERROR] MCP server "github" Server stderr: Unable to find image 'ghcr.io/github/github-mcp-server:v0.20.0' locally
[ERROR] MCP server "github" Server stderr: v0.20.0: Pulling from github/github-mcp-server
4eff9a62d888: Pulling fs layer
62de241dac5f: Pulling fs layer
{"type":"system","subtype":"init","session_id":"test-123","tools":["Bash","Read"],"model":"claude-sonnet-4-20250514"}
{"type":"assistant","message":{"content":[{"type":"text","text":"Working on it."},{"type":"tool_use","id":"tool_456","name":"Bash","input":{"command":"ls -la"}}]}}
{"type":"user","message":{"content":[{"type":"tool_result","tool_use_id":"tool_456","content":"total 0"}]}}
{"type":"result","total_cost_usd":0.002,"usage":{"input_tokens":100,"output_tokens":40},"num_turns":1}`

	result, err := runJSLogParser(script, dockerPullLog)
	if err != nil {
		t.Fatalf("Failed to parse docker pull format Claude log: %v", err)
	}

	// Verify parsing worked correctly despite docker pull lines
	if !strings.Contains(result, "🚀 Initialization") {
		t.Error("Expected docker pull format Claude log output to contain Initialization section")
	}
	if !strings.Contains(result, "ls -la") {
		t.Error("Expected docker pull format Claude log output to contain the bash command")
	}
	if !strings.Contains(result, "test-123") {
		t.Error("Expected docker pull format Claude log output to contain session ID")
	}

	t.Logf("JavaScript parser correctly handled docker pull format")
}
