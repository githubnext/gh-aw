package workflow

import (
	"os"
	"testing"
)

func TestClaudeEngine_ParseLogMetrics_WithToolInvocations(t *testing.T) {
	// Read test log file
	logContent, err := os.ReadFile("/tmp/test-logs/claude-test.log")
	if err != nil {
		// Skip test if file doesn't exist (may happen in CI)
		t.Skip("Test log file not found, skipping integration test")
	}

	engine := NewClaudeEngine()
	metrics := engine.ParseLogMetrics(string(logContent), true)

	// Verify basic metrics
	if metrics.TokenUsage != 200 { // 150 + 50
		t.Errorf("Expected token usage 200, got %d", metrics.TokenUsage)
	}

	if metrics.EstimatedCost != 0.0015 {
		t.Errorf("Expected cost 0.0015, got %f", metrics.EstimatedCost)
	}

	// Verify tool invocation statistics
	if len(metrics.ToolInvocations) == 0 {
		t.Fatal("Expected tool invocations to be parsed, got none")
	}

	// Check Bash tool invocation
	bashStats, exists := metrics.ToolInvocations["Bash"]
	if !exists {
		t.Fatal("Expected Bash tool invocation, not found")
	}

	if bashStats.Count != 1 {
		t.Errorf("Expected Bash count 1, got %d", bashStats.Count)
	}

	if bashStats.SuccessCount != 1 {
		t.Errorf("Expected Bash success count 1, got %d", bashStats.SuccessCount)
	}

	if bashStats.ErrorCount != 0 {
		t.Errorf("Expected Bash error count 0, got %d", bashStats.ErrorCount)
	}

	// Check MCP tool invocation
	mcpStats, exists := metrics.ToolInvocations["mcp__github__search_issues"]
	if !exists {
		t.Fatal("Expected mcp__github__search_issues tool invocation, not found")
	}

	if mcpStats.Count != 1 {
		t.Errorf("Expected MCP tool count 1, got %d", mcpStats.Count)
	}

	if mcpStats.SuccessCount != 1 {
		t.Errorf("Expected MCP tool success count 1, got %d", mcpStats.SuccessCount)
	}

	if mcpStats.ErrorCount != 0 {
		t.Errorf("Expected MCP tool error count 0, got %d", mcpStats.ErrorCount)
	}

	// Check output sizes (should be > 0 since we have content)
	if bashStats.TotalOutputSize <= 0 {
		t.Errorf("Expected Bash output size > 0, got %d", bashStats.TotalOutputSize)
	}

	if mcpStats.TotalOutputSize <= 0 {
		t.Errorf("Expected MCP tool output size > 0, got %d", mcpStats.TotalOutputSize)
	}

	t.Logf("Successfully parsed %d tool invocations", len(metrics.ToolInvocations))
	for toolName, stats := range metrics.ToolInvocations {
		t.Logf("Tool: %s, Count: %d, Success: %d, Error: %d, OutputSize: %d",
			toolName, stats.Count, stats.SuccessCount, stats.ErrorCount, stats.TotalOutputSize)
	}
}

func TestCodexEngine_ParseLogMetrics_WithToolInvocations(t *testing.T) {
	// Read test log file
	logContent, err := os.ReadFile("/tmp/test-logs/codex-test.log")
	if err != nil {
		// Skip test if file doesn't exist (may happen in CI)
		t.Skip("Test log file not found, skipping integration test")
	}

	engine := NewCodexEngine()
	metrics := engine.ParseLogMetrics(string(logContent), true)

	// Verify basic metrics
	if metrics.TokenUsage != 250 {
		t.Errorf("Expected token usage 250, got %d", metrics.TokenUsage)
	}

	// Verify tool invocation statistics
	if len(metrics.ToolInvocations) == 0 {
		t.Fatal("Expected tool invocations to be parsed, got none")
	}

	// Check get_current_time tool invocation
	timeStats, exists := metrics.ToolInvocations["get_current_time"]
	if !exists {
		t.Fatal("Expected get_current_time tool invocation, not found")
	}

	if timeStats.Count != 1 {
		t.Errorf("Expected get_current_time count 1, got %d", timeStats.Count)
	}

	if timeStats.SuccessCount != 1 {
		t.Errorf("Expected get_current_time success count 1, got %d", timeStats.SuccessCount)
	}

	if timeStats.ErrorCount != 0 {
		t.Errorf("Expected get_current_time error count 0, got %d", timeStats.ErrorCount)
	}

	// Check Bash exec command
	bashStats, exists := metrics.ToolInvocations["Bash"]
	if !exists {
		t.Fatal("Expected Bash tool invocation, not found")
	}

	if bashStats.Count != 1 {
		t.Errorf("Expected Bash count 1, got %d", bashStats.Count)
	}

	if bashStats.SuccessCount != 1 {
		t.Errorf("Expected Bash success count 1, got %d", bashStats.SuccessCount)
	}

	if bashStats.ErrorCount != 0 {
		t.Errorf("Expected Bash error count 0, got %d", bashStats.ErrorCount)
	}

	// Verify the duration was extracted (should be > 0)
	if timeStats.TotalDuration <= 0 {
		t.Errorf("Expected get_current_time duration > 0, got %v", timeStats.TotalDuration)
	}

	if bashStats.TotalDuration <= 0 {
		t.Errorf("Expected Bash duration > 0, got %v", bashStats.TotalDuration)
	}

	t.Logf("Successfully parsed %d tool invocations", len(metrics.ToolInvocations))
	for toolName, stats := range metrics.ToolInvocations {
		t.Logf("Tool: %s, Count: %d, Success: %d, Error: %d, OutputSize: %d, Duration: %v",
			toolName, stats.Count, stats.SuccessCount, stats.ErrorCount, stats.TotalOutputSize, stats.TotalDuration)
	}
}
