package cli

import (
	"archive/zip"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/githubnext/gh-aw/pkg/testutil"

	"github.com/githubnext/gh-aw/pkg/cli/fileutil"
	"github.com/githubnext/gh-aw/pkg/console"
	"github.com/githubnext/gh-aw/pkg/workflow"
)

func TestDownloadWorkflowLogs(t *testing.T) {
	t.Skip("Skipping slow network-dependent test")

	// Test the DownloadWorkflowLogs function
	// This should either fail with auth error (if not authenticated)
	// or succeed with no results (if authenticated but no workflows match)
	err := DownloadWorkflowLogs(context.Background(), "", 1, "", "", "./test-logs", "", "", 0, 0, "", false, false, false, false, false, false, false, 0, false, "summary.json", "")

	// If GitHub CLI is authenticated, the function may succeed but find no results
	// If not authenticated, it should return an auth error
	if err != nil {
		// If there's an error, it should be an authentication or workflow-related error
		errMsg := strings.ToLower(err.Error())
		if !strings.Contains(errMsg, "authentication required") &&
			!strings.Contains(errMsg, "failed to list workflow runs") &&
			!strings.Contains(errMsg, "exit status 1") {
			t.Errorf("Expected authentication error, workflow listing error, or no error, got: %v", err)
		}
	}
	// If err is nil, that's also acceptable (authenticated case with no results)

	// Clean up
	os.RemoveAll("./test-logs")
}

func TestFormatNumber(t *testing.T) {
	tests := []struct {
		input    int
		expected string
	}{
		{0, "0"},
		{5, "5"},
		{42, "42"},
		{999, "999"},
		{1000, "1.00k"},
		{1200, "1.20k"},
		{1234, "1.23k"},
		{12000, "12.0k"},
		{12300, "12.3k"},
		{123000, "123k"},
		{999999, "1000k"},
		{1000000, "1.00M"},
		{1200000, "1.20M"},
		{1234567, "1.23M"},
		{12000000, "12.0M"},
		{12300000, "12.3M"},
		{123000000, "123M"},
		{999999999, "1000M"},
		{1000000000, "1.00B"},
		{1200000000, "1.20B"},
		{1234567890, "1.23B"},
		{12000000000, "12.0B"},
		{123000000000, "123B"},
	}

	for _, test := range tests {
		result := console.FormatNumber(test.input)
		if result != test.expected {
			t.Errorf("console.FormatNumber(%d) = %s, expected %s", test.input, result, test.expected)
		}
	}
}

func TestParseLogFileWithoutAwInfo(t *testing.T) {
	// Create a temporary log file
	tmpDir := testutil.TempDir(t, "test-*")
	logFile := filepath.Join(tmpDir, "test.log")

	logContent := `2024-01-15T10:30:00Z Starting workflow execution
2024-01-15T10:30:15Z Claude API request initiated
2024-01-15T10:30:45Z Input tokens: 1250
2024-01-15T10:30:45Z Output tokens: 850
2024-01-15T10:30:45Z Total tokens used: 2100
2024-01-15T10:30:45Z Cost: $0.025
2024-01-15T10:31:30Z Workflow completed successfully`

	err := os.WriteFile(logFile, []byte(logContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test log file: %v", err)
	}

	// Test parseLogFileWithEngine without an engine (simulates no aw_info.json)
	metrics, err := parseLogFileWithEngine(logFile, nil, false, false)
	if err != nil {
		t.Fatalf("parseLogFileWithEngine failed: %v", err)
	}

	// Without aw_info.json, should return empty metrics
	if metrics.TokenUsage != 0 {
		t.Errorf("Expected token usage 0 (no aw_info.json), got %d", metrics.TokenUsage)
	}

	// Check cost - should be 0 without engine-specific parsing
	if metrics.EstimatedCost != 0 {
		t.Errorf("Expected cost 0 (no aw_info.json), got %f", metrics.EstimatedCost)
	}

	// Duration is no longer extracted from logs - using GitHub API timestamps instead
}

func TestExtractJSONMetrics(t *testing.T) {
	tests := []struct {
		name           string
		line           string
		expectedTokens int
		expectedCost   float64
	}{
		{
			name:           "Claude streaming format with usage",
			line:           `{"type": "message_delta", "delta": {"usage": {"input_tokens": 123, "output_tokens": 456}}}`,
			expectedTokens: 579, // 123 + 456
		},
		{
			name:           "Simple token count (timestamp ignored)",
			line:           `{"tokens": 1234, "timestamp": "2024-01-15T10:30:00Z"}`,
			expectedTokens: 1234,
		},
		{
			name:         "Cost information",
			line:         `{"cost": 0.045, "price": 0.01}`,
			expectedCost: 0.045, // Should pick up the first one found
		},
		{
			name:           "Usage object with cost",
			line:           `{"usage": {"total_tokens": 999}, "billing": {"cost": 0.123}}`,
			expectedTokens: 999,
			expectedCost:   0.123,
		},
		{
			name:           "Claude result format with total_cost_usd",
			line:           `{"type": "result", "total_cost_usd": 0.8606770999999999, "usage": {"input_tokens": 126, "output_tokens": 7685}}`,
			expectedTokens: 7811, // 126 + 7685
			expectedCost:   0.8606770999999999,
		},
		{
			name:           "Claude result format with cache tokens",
			line:           `{"type": "result", "total_cost_usd": 0.86, "usage": {"input_tokens": 126, "cache_creation_input_tokens": 100034, "cache_read_input_tokens": 1232098, "output_tokens": 7685}}`,
			expectedTokens: 1339943, // 126 + 100034 + 1232098 + 7685
			expectedCost:   0.86,
		},
		{
			name:           "Not JSON",
			line:           "regular log line with tokens: 123",
			expectedTokens: 0, // Should return zero since it's not JSON
		},
		{
			name:           "Invalid JSON",
			line:           `{"invalid": json}`,
			expectedTokens: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metrics := extractJSONMetrics(tt.line, false)

			if metrics.TokenUsage != tt.expectedTokens {
				t.Errorf("Expected tokens %d, got %d", tt.expectedTokens, metrics.TokenUsage)
			}

			if metrics.EstimatedCost != tt.expectedCost {
				t.Errorf("Expected cost %f, got %f", tt.expectedCost, metrics.EstimatedCost)
			}
		})
	}
}

func TestParseLogFileWithJSON(t *testing.T) {
	// Create a temporary log file with mixed JSON and text format
	tmpDir := testutil.TempDir(t, "test-*")
	logFile := filepath.Join(tmpDir, "test-mixed.log")

	logContent := `2024-01-15T10:30:00Z Starting workflow execution
{"type": "message_start"}
{"type": "content_block_delta", "delta": {"type": "text", "text": "Hello"}}
{"type": "message_delta", "delta": {"usage": {"input_tokens": 150, "output_tokens": 200}}}
Regular log line: tokens: 1000
{"cost": 0.035}
2024-01-15T10:31:30Z Workflow completed successfully`

	err := os.WriteFile(logFile, []byte(logContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test log file: %v", err)
	}

	metrics, err := parseLogFileWithEngine(logFile, nil, false, false)
	if err != nil {
		t.Fatalf("parseLogFileWithEngine failed: %v", err)
	}

	// Without aw_info.json and specific engine, should return empty metrics
	if metrics.TokenUsage != 0 {
		t.Errorf("Expected token usage 0 (no aw_info.json), got %d", metrics.TokenUsage)
	}

	// Should have no cost without engine-specific parsing
	if metrics.EstimatedCost != 0 {
		t.Errorf("Expected cost 0 (no aw_info.json), got %f", metrics.EstimatedCost)
	}

	// Duration is no longer extracted from logs - using GitHub API timestamps instead
}

func TestConvertToInt(t *testing.T) {
	tests := []struct {
		value    any
		expected int
	}{
		{123, 123},
		{int64(456), 456},
		{789.0, 789},
		{"123", 123},
		{"invalid", 0},
		{nil, 0},
	}

	for _, tt := range tests {
		result := workflow.ConvertToInt(tt.value)
		if result != tt.expected {
			t.Errorf("ConvertToInt(%v) = %d, expected %d", tt.value, result, tt.expected)
		}
	}
}

func TestConvertToFloat(t *testing.T) {
	tests := []struct {
		value    any
		expected float64
	}{
		{123.45, 123.45},
		{123, 123.0},
		{int64(456), 456.0},
		{"123.45", 123.45},
		{"invalid", 0.0},
		{nil, 0.0},
	}

	for _, tt := range tests {
		result := workflow.ConvertToFloat(tt.value)
		if result != tt.expected {
			t.Errorf("ConvertToFloat(%v) = %f, expected %f", tt.value, result, tt.expected)
		}
	}
}

func TestDirExists(t *testing.T) {
	tmpDir := testutil.TempDir(t, "test-*")

	// Test existing directory
	if !fileutil.DirExists(tmpDir) {
		t.Errorf("DirExists should return true for existing directory")
	}

	// Test non-existing directory
	nonExistentDir := filepath.Join(tmpDir, "does-not-exist")
	if fileutil.DirExists(nonExistentDir) {
		t.Errorf("DirExists should return false for non-existing directory")
	}

	// Test file vs directory
	testFile := filepath.Join(tmpDir, "testfile")
	err := os.WriteFile(testFile, []byte("test"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	if fileutil.DirExists(testFile) {
		t.Errorf("DirExists should return false for a file")
	}
}

func TestIsDirEmpty(t *testing.T) {
	tmpDir := testutil.TempDir(t, "test-*")

	// Test empty directory
	emptyDir := filepath.Join(tmpDir, "empty")
	err := os.Mkdir(emptyDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create empty directory: %v", err)
	}

	if !fileutil.IsDirEmpty(emptyDir) {
		t.Errorf("IsDirEmpty should return true for empty directory")
	}

	// Test directory with files
	nonEmptyDir := filepath.Join(tmpDir, "nonempty")
	err = os.Mkdir(nonEmptyDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create non-empty directory: %v", err)
	}

	testFile := filepath.Join(nonEmptyDir, "testfile")
	err = os.WriteFile(testFile, []byte("test"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	if fileutil.IsDirEmpty(nonEmptyDir) {
		t.Errorf("IsDirEmpty should return false for directory with files")
	}

	// Test non-existing directory
	nonExistentDir := filepath.Join(tmpDir, "does-not-exist")
	if !fileutil.IsDirEmpty(nonExistentDir) {
		t.Errorf("IsDirEmpty should return true for non-existing directory")
	}
}

func TestErrNoArtifacts(t *testing.T) {
	// Test that ErrNoArtifacts is properly defined and can be used with errors.Is
	err := ErrNoArtifacts
	if !errors.Is(err, ErrNoArtifacts) {
		t.Errorf("errors.Is should return true for ErrNoArtifacts")
	}

	// Test wrapping
	wrappedErr := errors.New("wrapped: " + ErrNoArtifacts.Error())
	if errors.Is(wrappedErr, ErrNoArtifacts) {
		t.Errorf("errors.Is should return false for wrapped error that doesn't use errors.Wrap")
	}
}

func TestListWorkflowRunsWithPagination(t *testing.T) {
	// Test that listWorkflowRunsWithPagination properly adds beforeDate filter
	// Since we can't easily mock the GitHub CLI, we'll test with known auth issues

	// This should fail with authentication error (if not authenticated)
	// or succeed with empty results (if authenticated but no workflows match)
	runs, _, err := listWorkflowRunsWithPagination(ListWorkflowRunsOptions{
		WorkflowName:   "nonexistent-workflow",
		Limit:          5,
		BeforeDate:     "2024-01-01T00:00:00Z",
		ProcessedCount: 0,
		TargetCount:    5,
		Verbose:        false,
	})

	if err != nil {
		// If there's an error, it should be an authentication error or workflow not found
		if !strings.Contains(err.Error(), "authentication required") && !strings.Contains(err.Error(), "failed to list workflow runs") {
			t.Errorf("Expected authentication error or workflow error, got: %v", err)
		}
	} else {
		// If no error, should return empty results for nonexistent workflow
		if len(runs) > 0 {
			t.Errorf("Expected empty results for nonexistent workflow, got %d runs", len(runs))
		}
	}
}

func TestIterativeAlgorithmConstants(t *testing.T) {
	// Test that our constants are reasonable
	if MaxIterations <= 0 {
		t.Errorf("MaxIterations should be positive, got %d", MaxIterations)
	}
	if MaxIterations > 20 {
		t.Errorf("MaxIterations seems too high (%d), could cause performance issues", MaxIterations)
	}

	if BatchSize <= 0 {
		t.Errorf("BatchSize should be positive, got %d", BatchSize)
	}
	if BatchSize > 100 {
		t.Errorf("BatchSize seems too high (%d), might hit GitHub API limits", BatchSize)
	}
}

func TestExtractJSONCost(t *testing.T) {
	tests := []struct {
		name     string
		data     map[string]any
		expected float64
	}{
		{
			name:     "total_cost_usd field",
			data:     map[string]any{"total_cost_usd": 0.8606770999999999},
			expected: 0.8606770999999999,
		},
		{
			name:     "traditional cost field",
			data:     map[string]any{"cost": 1.23},
			expected: 1.23,
		},
		{
			name:     "nested billing cost",
			data:     map[string]any{"billing": map[string]any{"cost": 2.45}},
			expected: 2.45,
		},
		{
			name:     "no cost fields",
			data:     map[string]any{"tokens": 1000},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := workflow.ExtractJSONCost(tt.data)
			if result != tt.expected {
				t.Errorf("ExtractJSONCost() = %f, expected %f", result, tt.expected)
			}
		})
	}
}

func TestParseLogFileWithClaudeResult(t *testing.T) {
	// Create a temporary log file with the exact Claude result format from the issue
	tmpDir := testutil.TempDir(t, "test-*")
	logFile := filepath.Join(tmpDir, "test-claude.log")

	// This is the exact JSON format provided in the issue (compacted to single line)
	claudeResultJSON := `{"type": "result", "subtype": "success", "is_error": false, "duration_ms": 145056, "duration_api_ms": 142970, "num_turns": 66, "result": "**Integration test execution complete. All objectives achieved successfully.** 🎯", "session_id": "d0a2839f-3569-42e9-9ccb-70835de4e760", "total_cost_usd": 0.8606770999999999, "usage": {"input_tokens": 126, "cache_creation_input_tokens": 100034, "cache_read_input_tokens": 1232098, "output_tokens": 7685, "server_tool_use": {"web_search_requests": 0}, "service_tier": "standard"}}`

	logContent := `2024-01-15T10:30:00Z Starting Claude workflow execution
Claude processing request...
` + claudeResultJSON + `
2024-01-15T10:32:30Z Workflow completed successfully`

	err := os.WriteFile(logFile, []byte(logContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test log file: %v", err)
	}

	// Test with Claude engine to parse Claude-specific logs
	claudeEngine := workflow.NewClaudeEngine()
	metrics, err := parseLogFileWithEngine(logFile, claudeEngine, false, false)
	if err != nil {
		t.Fatalf("parseLogFileWithEngine failed: %v", err)
	}

	// Check total token usage includes all token types from Claude
	expectedTokens := 126 + 100034 + 1232098 + 7685 // input + cache_creation + cache_read + output
	if metrics.TokenUsage != expectedTokens {
		t.Errorf("Expected token usage %d, got %d", expectedTokens, metrics.TokenUsage)
	}

	// Check cost extraction from total_cost_usd
	expectedCost := 0.8606770999999999
	if metrics.EstimatedCost != expectedCost {
		t.Errorf("Expected cost %f, got %f", expectedCost, metrics.EstimatedCost)
	}

	// Check turns extraction from num_turns
	expectedTurns := 66
	if metrics.Turns != expectedTurns {
		t.Errorf("Expected turns %d, got %d", expectedTurns, metrics.Turns)
	}

	// Duration is no longer extracted from logs - using GitHub API timestamps instead
}

func TestParseLogFileWithCodexFormat(t *testing.T) {
	// Create a temporary log file with the Codex output format from the issue
	tmpDir := testutil.TempDir(t, "test-*")
	logFile := filepath.Join(tmpDir, "test-codex.log")

	// This is the exact Codex format provided in the issue with thinking sections added
	logContent := `[2025-08-13T00:24:45] Starting Codex workflow execution
[2025-08-13T00:24:50] thinking
I need to analyze the pull request details first.
[2025-08-13T00:24:50] codex

I'm ready to generate a Codex PR summary, but I need the pull request number to fetch its details. Could you please share the PR number (and confirm the repo/owner if it isn't ` + "`githubnext/gh-aw`" + `)?
[2025-08-13T00:24:50] thinking  
Now I need to wait for the user's response.
[2025-08-13T00:24:50] tokens used: 13934
[2025-08-13T00:24:55] Workflow completed successfully`

	err := os.WriteFile(logFile, []byte(logContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test log file: %v", err)
	}

	// Test with Codex engine to parse Codex-specific logs
	codexEngine := workflow.NewCodexEngine()
	metrics, err := parseLogFileWithEngine(logFile, codexEngine, false, false)
	if err != nil {
		t.Fatalf("parseLogFileWithEngine failed: %v", err)
	}

	// Check token usage extraction from Codex format
	expectedTokens := 13934
	if metrics.TokenUsage != expectedTokens {
		t.Errorf("Expected token usage %d, got %d", expectedTokens, metrics.TokenUsage)
	}

	// Check turns extraction from thinking sections
	expectedTurns := 2 // Two thinking sections in the test data
	if metrics.Turns != expectedTurns {
		t.Errorf("Expected turns %d, got %d", expectedTurns, metrics.Turns)
	}

	// Duration is no longer extracted from logs - using GitHub API timestamps instead
}

func TestParseLogFileWithCodexTokenSumming(t *testing.T) {
	// Create a temporary log file with multiple Codex token entries
	tmpDir := testutil.TempDir(t, "test-*")
	logFile := filepath.Join(tmpDir, "test-codex-tokens.log")

	// Simulate the exact Codex format from the issue
	logContent := `  ]
}
[2025-08-13T04:38:03] tokens used: 32169
[2025-08-13T04:38:06] codex
I've posted the PR summary comment with analysis and recommendations. Let me know if you'd like to adjust any details or add further insights!
[2025-08-13T04:38:06] tokens used: 28828
[2025-08-13T04:38:10] Processing complete
[2025-08-13T04:38:15] tokens used: 5000`

	err := os.WriteFile(logFile, []byte(logContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test log file: %v", err)
	}

	// Test with Codex engine
	codexEngine := workflow.NewCodexEngine()
	metrics, err := parseLogFileWithEngine(logFile, codexEngine, false, false)
	if err != nil {
		t.Fatalf("parseLogFileWithEngine failed: %v", err)
	}

	// Check that all token entries are summed
	expectedTokens := 32169 + 28828 + 5000 // 65997
	if metrics.TokenUsage != expectedTokens {
		t.Errorf("Expected summed token usage %d, got %d", expectedTokens, metrics.TokenUsage)
	}
}

func TestParseLogFileWithCodexRustFormat(t *testing.T) {
	// Create a temporary log file with the new Rust-based Codex format
	tmpDir := testutil.TempDir(t, "test-*")
	logFile := filepath.Join(tmpDir, "test-codex-rust.log")

	// This simulates the new Rust format from the Codex engine
	logContent := `2025-01-15T10:30:00.123456Z  INFO codex: Starting codex execution
2025-01-15T10:30:00.234567Z DEBUG codex_core: Initializing MCP servers
2025-01-15T10:30:01.123456Z  INFO codex: Session initialized
thinking
Let me fetch the list of pull requests first to see what we're working with.
2025-01-15T10:30:02.123456Z DEBUG codex_exec: Executing tool call
tool github.list_pull_requests({"state": "closed", "per_page": 5})
2025-01-15T10:30:03.456789Z DEBUG codex_core: Tool execution started
2025-01-15T10:30:04.567890Z  INFO codex: github.list_pull_requests(...) success in 2.1s
thinking
Now I need to get details for each PR to write a comprehensive summary.
2025-01-15T10:30:05.123456Z DEBUG codex_exec: Executing tool call
tool github.get_pull_request({"pull_number": 123})
2025-01-15T10:30:06.234567Z  INFO codex: github.get_pull_request(...) success in 0.8s
2025-01-15T10:30:07.345678Z DEBUG codex_core: Processing response
thinking
I have all the information I need. Let me create a summary issue.
2025-01-15T10:30:08.456789Z DEBUG codex_exec: Executing tool call
tool safe_outputs.create_issue({"title": "PR Summary", "body": "..."})
2025-01-15T10:30:09.567890Z  INFO codex: safe_outputs.create_issue(...) success in 1.2s
2025-01-15T10:30:10.123456Z DEBUG codex_core: Workflow completing
tokens used: 15234
2025-01-15T10:30:10.234567Z  INFO codex: Execution complete`

	err := os.WriteFile(logFile, []byte(logContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test log file: %v", err)
	}

	// Test with Codex engine to parse new Rust format
	codexEngine := workflow.NewCodexEngine()
	metrics, err := parseLogFileWithEngine(logFile, codexEngine, false, false)
	if err != nil {
		t.Fatalf("parseLogFileWithEngine failed: %v", err)
	}

	// Check token usage extraction from Rust format
	expectedTokens := 15234
	if metrics.TokenUsage != expectedTokens {
		t.Errorf("Expected token usage %d, got %d", expectedTokens, metrics.TokenUsage)
	}

	// Check turns extraction from thinking sections (new Rust format uses standalone "thinking" lines)
	expectedTurns := 3 // Three thinking sections in the test data
	if metrics.Turns != expectedTurns {
		t.Errorf("Expected turns %d, got %d", expectedTurns, metrics.Turns)
	}

	// Check tool calls extraction from new Rust format
	expectedToolCount := 3
	if len(metrics.ToolCalls) != expectedToolCount {
		t.Errorf("Expected %d tool calls, got %d", expectedToolCount, len(metrics.ToolCalls))
	}

	// Verify the specific tools were detected
	toolNames := make(map[string]bool)
	for _, tool := range metrics.ToolCalls {
		toolNames[tool.Name] = true
	}

	expectedTools := []string{"github_list_pull_requests", "github_get_pull_request", "safe_outputs_create_issue"}
	for _, expectedTool := range expectedTools {
		if !toolNames[expectedTool] {
			t.Errorf("Expected tool %s not found in tool calls", expectedTool)
		}
	}
}

func TestParseLogFileWithCodexMixedFormats(t *testing.T) {
	// Create a temporary log file with mixed old TypeScript and new Rust formats
	tmpDir := testutil.TempDir(t, "test-*")
	logFile := filepath.Join(tmpDir, "test-codex-mixed.log")

	// Mix both formats to test backward compatibility
	logContent := `[2025-08-13T00:24:45] Starting Codex workflow execution
[2025-08-13T00:24:50] thinking
Old format thinking section
[2025-08-13T00:24:50] tool github.list_repos({"org": "test"})
[2025-08-13T00:24:51] codex
Response from old format
2025-01-15T10:30:00.123456Z  INFO codex: Starting execution
thinking
New Rust format thinking section
tool github.create_issue({"title": "Test", "body": "Body"})
2025-01-15T10:30:05.567890Z  INFO codex: github.create_issue(...) success in 1.2s
[2025-08-13T00:24:52] tokens used: 5000
tokens used: 10000
2025-01-15T10:30:10.234567Z  INFO codex: Execution complete`

	err := os.WriteFile(logFile, []byte(logContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test log file: %v", err)
	}

	// Test with Codex engine to parse mixed formats
	codexEngine := workflow.NewCodexEngine()
	metrics, err := parseLogFileWithEngine(logFile, codexEngine, false, false)
	if err != nil {
		t.Fatalf("parseLogFileWithEngine failed: %v", err)
	}

	// Check token usage is summed from both formats
	expectedTokens := 15000 // 5000 + 10000
	if metrics.TokenUsage != expectedTokens {
		t.Errorf("Expected token usage %d (summed from both formats), got %d", expectedTokens, metrics.TokenUsage)
	}

	// Check turns from both formats
	expectedTurns := 2 // One from old format, one from new format
	if metrics.Turns != expectedTurns {
		t.Errorf("Expected turns %d (from both formats), got %d", expectedTurns, metrics.Turns)
	}

	// Check tool calls from both formats
	expectedToolCount := 2
	if len(metrics.ToolCalls) != expectedToolCount {
		t.Errorf("Expected %d tool calls, got %d", expectedToolCount, len(metrics.ToolCalls))
	}

	// Verify the specific tools were detected from both formats
	toolNames := make(map[string]bool)
	for _, tool := range metrics.ToolCalls {
		toolNames[tool.Name] = true
	}

	expectedTools := []string{"github_list_repos", "github_create_issue"}
	for _, expectedTool := range expectedTools {
		if !toolNames[expectedTool] {
			t.Errorf("Expected tool %s not found in tool calls", expectedTool)
		}
	}
}

func TestParseLogFileWithMixedTokenFormats(t *testing.T) {
	// Create a temporary log file with mixed token formats
	tmpDir := testutil.TempDir(t, "test-*")
	logFile := filepath.Join(tmpDir, "test-mixed-tokens.log")

	// Mix of Codex and non-Codex formats - should prioritize Codex summing
	logContent := `[2025-08-13T04:38:03] tokens used: 1000
tokens: 5000
[2025-08-13T04:38:06] tokens used: 2000
token_count: 10000`

	err := os.WriteFile(logFile, []byte(logContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test log file: %v", err)
	}

	// Get the Codex engine for testing
	registry := workflow.NewEngineRegistry()
	codexEngine, err := registry.GetEngine("codex")
	if err != nil {
		t.Fatalf("Failed to get Codex engine: %v", err)
	}

	metrics, err := parseLogFileWithEngine(logFile, codexEngine, false, false)
	if err != nil {
		t.Fatalf("parseLogFile failed: %v", err)
	}

	// Should sum Codex entries: 1000 + 2000 = 3000 (ignoring non-Codex formats)
	expectedTokens := 1000 + 2000
	if metrics.TokenUsage != expectedTokens {
		t.Errorf("Expected token usage %d (sum of Codex entries), got %d", expectedTokens, metrics.TokenUsage)
	}
}

func TestExtractEngineFromAwInfoNestedDirectory(t *testing.T) {
	tmpDir := testutil.TempDir(t, "test-*")

	// Test Case 1: aw_info.json as a regular file
	awInfoFile := filepath.Join(tmpDir, "aw_info.json")
	awInfoContent := `{
		"engine_id": "claude",
		"engine_name": "Claude",
		"model": "claude-3-sonnet",
		"version": "20240620",
		"workflow_name": "Test Claude",
		"experimental": false,
		"supports_tools_allowlist": true,
		"supports_http_transport": false,
		"run_id": 123456789,
		"run_number": 42,
		"run_attempt": "1",
		"repository": "githubnext/gh-aw",
		"ref": "refs/heads/main",
		"sha": "abc123",
		"actor": "testuser",
		"event_name": "workflow_dispatch",
		"created_at": "2025-08-13T13:36:39.704Z"
	}`

	err := os.WriteFile(awInfoFile, []byte(awInfoContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create aw_info.json file: %v", err)
	}

	// Test regular file extraction
	engine := extractEngineFromAwInfo(awInfoFile, true)
	if engine == nil {
		t.Errorf("Expected to extract engine from regular aw_info.json file, got nil")
	} else if engine.GetID() != "claude" {
		t.Errorf("Expected engine ID 'claude', got '%s'", engine.GetID())
	}

	// Clean up for next test
	os.Remove(awInfoFile)

	// Test Case 2: aw_info.json as a directory containing the actual file
	awInfoDir := filepath.Join(tmpDir, "aw_info.json")
	err = os.Mkdir(awInfoDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create aw_info.json directory: %v", err)
	}

	// Create the nested aw_info.json file inside the directory
	nestedAwInfoFile := filepath.Join(awInfoDir, "aw_info.json")
	awInfoContentCodex := `{
		"engine_id": "codex",
		"engine_name": "Codex",
		"model": "o4-mini",
		"version": "",
		"workflow_name": "Test Codex",
		"experimental": true,
		"supports_tools_allowlist": true,
		"supports_http_transport": false,
		"run_id": 987654321,
		"run_number": 7,
		"run_attempt": "1",
		"repository": "githubnext/gh-aw",
		"ref": "refs/heads/copilot/fix-24",
		"sha": "def456",
		"actor": "testuser2",
		"event_name": "workflow_dispatch",
		"created_at": "2025-08-13T13:36:39.704Z"
	}`

	err = os.WriteFile(nestedAwInfoFile, []byte(awInfoContentCodex), 0644)
	if err != nil {
		t.Fatalf("Failed to create nested aw_info.json file: %v", err)
	}

	// Test directory-based extraction (the main fix)
	engine = extractEngineFromAwInfo(awInfoDir, true)
	if engine == nil {
		t.Errorf("Expected to extract engine from aw_info.json directory, got nil")
	} else if engine.GetID() != "codex" {
		t.Errorf("Expected engine ID 'codex', got '%s'", engine.GetID())
	}

	// Test Case 3: Non-existent aw_info.json should return nil
	nonExistentPath := filepath.Join(tmpDir, "nonexistent", "aw_info.json")
	engine = extractEngineFromAwInfo(nonExistentPath, false)
	if engine != nil {
		t.Errorf("Expected nil for non-existent aw_info.json, got engine: %s", engine.GetID())
	}

	// Test Case 4: Directory without nested aw_info.json should return nil
	emptyDir := filepath.Join(tmpDir, "empty_aw_info.json")
	err = os.Mkdir(emptyDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create empty directory: %v", err)
	}

	engine = extractEngineFromAwInfo(emptyDir, false)
	if engine != nil {
		t.Errorf("Expected nil for directory without nested aw_info.json, got engine: %s", engine.GetID())
	}

	// Test Case 5: Invalid JSON should return nil
	invalidAwInfoFile := filepath.Join(tmpDir, "invalid_aw_info.json")
	invalidContent := `{invalid json content`
	err = os.WriteFile(invalidAwInfoFile, []byte(invalidContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create invalid aw_info.json file: %v", err)
	}

	engine = extractEngineFromAwInfo(invalidAwInfoFile, false)
	if engine != nil {
		t.Errorf("Expected nil for invalid JSON aw_info.json, got engine: %s", engine.GetID())
	}

	// Test Case 6: Missing engine_id should return nil
	missingEngineIDFile := filepath.Join(tmpDir, "missing_engine_id_aw_info.json")
	missingEngineIDContent := `{
		"workflow_name": "Test Workflow",
		"run_id": 123456789
	}`
	err = os.WriteFile(missingEngineIDFile, []byte(missingEngineIDContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create aw_info.json file without engine_id: %v", err)
	}

	engine = extractEngineFromAwInfo(missingEngineIDFile, false)
	if engine != nil {
		t.Errorf("Expected nil for aw_info.json without engine_id, got engine: %s", engine.GetID())
	}
}

func TestParseLogFileWithNonCodexTokensOnly(t *testing.T) {
	// Create a temporary log file with only non-Codex token formats
	tmpDir := testutil.TempDir(t, "test-*")
	logFile := filepath.Join(tmpDir, "test-generic-tokens.log")

	// Only non-Codex formats - should keep maximum behavior
	logContent := `tokens: 5000
token_count: 10000
input_tokens: 2000`

	err := os.WriteFile(logFile, []byte(logContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test log file: %v", err)
	}

	// Without aw_info.json and specific engine, should return empty metrics
	metrics, err := parseLogFileWithEngine(logFile, nil, false, false)
	if err != nil {
		t.Fatalf("parseLogFileWithEngine failed: %v", err)
	}

	// Without engine-specific parsing, should return 0
	if metrics.TokenUsage != 0 {
		t.Errorf("Expected token usage 0 (no aw_info.json), got %d", metrics.TokenUsage)
	}
}

func TestDownloadWorkflowLogsWithEngineFilter(t *testing.T) {
	t.Skip("Skipping slow network-dependent test")

	// Test that the engine filter parameter is properly validated and passed through
	tests := []struct {
		name        string
		engine      string
		expectError bool
		errorText   string
	}{
		{
			name:        "valid claude engine",
			engine:      "claude",
			expectError: false,
		},
		{
			name:        "valid codex engine",
			engine:      "codex",
			expectError: false,
		},
		{
			name:        "valid copilot engine",
			engine:      "copilot",
			expectError: false,
		},
		{
			name:        "empty engine (no filter)",
			engine:      "",
			expectError: false,
		},
		{
			name:        "invalid engine",
			engine:      "gpt",
			expectError: true,
			errorText:   "invalid engine value 'gpt'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This function should validate the engine parameter
			// If invalid, it would exit in the actual command but we can't test that easily
			// So we just test that valid engines don't cause immediate errors
			if !tt.expectError {
				// For valid engines, test that the function can be called without panic
				// It may still fail with auth errors, which is expected
				err := DownloadWorkflowLogs(context.Background(), "", 1, "", "", "./test-logs", tt.engine, "", 0, 0, "", false, false, false, false, false, false, false, 0, false, "summary.json", "")

				// Clean up any created directories
				os.RemoveAll("./test-logs")

				// If there's an error, it should be auth or workflow-related, not parameter validation
				if err != nil {
					errMsg := strings.ToLower(err.Error())
					if strings.Contains(errMsg, "invalid engine") {
						t.Errorf("Got engine validation error for valid engine '%s': %v", tt.engine, err)
					}
				}
			}
		})
	}
}
func TestLogsCommandFlags(t *testing.T) {
	// Test that the logs command has the expected flags including the new engine flag
	cmd := NewLogsCommand()

	// Check that all expected flags are present
	expectedFlags := []string{"count", "start-date", "end-date", "output", "engine", "ref", "before-run-id", "after-run-id"}

	for _, flagName := range expectedFlags {
		flag := cmd.Flags().Lookup(flagName)
		if flag == nil {
			t.Errorf("Expected flag '%s' not found in logs command", flagName)
		}
	}

	// Test ref flag specifically
	refFlag := cmd.Flags().Lookup("ref")
	if refFlag == nil {
		t.Fatal("Ref flag not found")
	}

	if refFlag.Usage != "Filter runs by branch or tag name (e.g., main, v1.0.0)" {
		t.Errorf("Unexpected ref flag usage text: %s", refFlag.Usage)
	}

	if refFlag.DefValue != "" {
		t.Errorf("Expected ref flag default value to be empty, got: %s", refFlag.DefValue)
	}

	// Test before-run-id flag
	beforeRunIDFlag := cmd.Flags().Lookup("before-run-id")
	if beforeRunIDFlag == nil {
		t.Fatal("Before-run-id flag not found")
	}

	if beforeRunIDFlag.Usage != "Filter runs with database ID before this value (exclusive)" {
		t.Errorf("Unexpected before-run-id flag usage text: %s", beforeRunIDFlag.Usage)
	}

	// Test after-run-id flag
	afterRunIDFlag := cmd.Flags().Lookup("after-run-id")
	if afterRunIDFlag == nil {
		t.Fatal("After-run-id flag not found")
	}

	if afterRunIDFlag.Usage != "Filter runs with database ID after this value (exclusive)" {
		t.Errorf("Unexpected after-run-id flag usage text: %s", afterRunIDFlag.Usage)
	}

	// Test engine flag specifically
	engineFlag := cmd.Flags().Lookup("engine")
	if engineFlag == nil {
		t.Fatal("Engine flag not found")
	}

	if engineFlag.Usage != "Filter logs by AI engine (claude, codex, copilot, custom)" {
		t.Errorf("Unexpected engine flag usage text: %s", engineFlag.Usage)
	}

	if engineFlag.DefValue != "" {
		t.Errorf("Expected engine flag default value to be empty, got: %s", engineFlag.DefValue)
	}

	// Test that engine flag has the -e shorthand for consistency with other commands
	if engineFlag.Shorthand != "e" {
		t.Errorf("Expected engine flag shorthand to be 'e', got: %s", engineFlag.Shorthand)
	}
}

func TestFormatFileSize(t *testing.T) {
	tests := []struct {
		size     int64
		expected string
	}{
		{0, "0 B"},
		{100, "100 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},          // 1.5 * 1024
		{1048576, "1.0 MB"},       // 1024 * 1024
		{2097152, "2.0 MB"},       // 2 * 1024 * 1024
		{1073741824, "1.0 GB"},    // 1024^3
		{1099511627776, "1.0 TB"}, // 1024^4
	}

	for _, tt := range tests {
		result := console.FormatFileSize(tt.size)
		if result != tt.expected {
			t.Errorf("console.FormatFileSize(%d) = %q, expected %q", tt.size, result, tt.expected)
		}
	}
}

func TestExtractLogMetricsWithAwOutputFile(t *testing.T) {
	// Create a temporary directory with safe_output.jsonl
	tmpDir := testutil.TempDir(t, "test-*")

	// Create safe_output.jsonl file
	awOutputPath := filepath.Join(tmpDir, "safe_output.jsonl")
	awOutputContent := "This is the agent's output content.\nIt contains multiple lines."
	err := os.WriteFile(awOutputPath, []byte(awOutputContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create safe_output.jsonl: %v", err)
	}

	// Test that extractLogMetrics doesn't fail with safe_output.jsonl present
	metrics, err := extractLogMetrics(tmpDir, false)
	if err != nil {
		t.Fatalf("extractLogMetrics failed: %v", err)
	}

	// Without an engine, should return empty metrics but not error
	if metrics.TokenUsage != 0 {
		t.Errorf("Expected token usage 0 (no engine), got %d", metrics.TokenUsage)
	}

	// Test verbose mode to ensure it detects the file
	// We can't easily test the console output, but we can ensure it doesn't error
	metrics, err = extractLogMetrics(tmpDir, true)
	if err != nil {
		t.Fatalf("extractLogMetrics in verbose mode failed: %v", err)
	}
}

func TestCustomEngineParseLogMetrics(t *testing.T) {
	// Test that custom engine tries both Claude and Codex parsing approaches
	customEngine := workflow.NewCustomEngine()

	// Test Case 1: Claude-style logs (properly formatted as JSON array)
	claudeLogContent := `[{"type": "message", "content": "Starting workflow"}, {"type": "result", "subtype": "success", "is_error": false, "num_turns": 42, "total_cost_usd": 1.5, "usage": {"input_tokens": 1000, "output_tokens": 500}}]`

	metrics := customEngine.ParseLogMetrics(claudeLogContent, false)

	// Should extract turns, tokens, and cost from Claude format
	if metrics.Turns != 42 {
		t.Errorf("Expected turns 42 from Claude-style logs, got %d", metrics.Turns)
	}
	if metrics.TokenUsage != 1500 {
		t.Errorf("Expected token usage 1500 from Claude-style logs, got %d", metrics.TokenUsage)
	}
	if metrics.EstimatedCost != 1.5 {
		t.Errorf("Expected cost 1.5 from Claude-style logs, got %f", metrics.EstimatedCost)
	}

	// Test Case 2: Codex-style logs
	codexLogContent := `[2025-08-13T00:24:45] Starting workflow
[2025-08-13T00:24:50] thinking
I need to analyze the problem.
[2025-08-13T00:24:51] codex
Working on solution.
[2025-08-13T00:24:52] thinking
Now I'll implement the solution.
[2025-08-13T00:24:53] tokens used: 5000
[2025-08-13T00:24:55] Workflow completed`

	metrics = customEngine.ParseLogMetrics(codexLogContent, false)

	// Should extract turns and tokens from Codex format
	if metrics.Turns != 2 {
		t.Errorf("Expected turns 2 from Codex-style logs, got %d", metrics.Turns)
	}
	if metrics.TokenUsage != 5000 {
		t.Errorf("Expected token usage 5000 from Codex-style logs, got %d", metrics.TokenUsage)
	}

	// Test Case 3: Generic logs (fallback - no error detection)
	// Custom engine no longer has its own error detection patterns
	// It relies on Claude/Codex parsers which don't match plain text logs
	genericLogContent := `2025-08-13T00:24:45 Starting workflow
2025-08-13T00:24:50 Processing request
2025-08-13T00:24:52 Warning: Something happened
2025-08-13T00:24:53 Error: Failed to process
2025-08-13T00:24:55 Workflow completed`

	metrics = customEngine.ParseLogMetrics(genericLogContent, false)

	// Should fall back to basic parsing (no metrics extracted)
	if metrics.Turns != 0 {
		t.Errorf("Expected turns 0 from generic logs, got %d", metrics.Turns)
	}
	if metrics.TokenUsage != 0 {
		t.Errorf("Expected token usage 0 from generic logs, got %d", metrics.TokenUsage)
	}
	// Error patterns have been removed - no error/warning counting
}

func TestRunIDFilteringLogic(t *testing.T) {
	// Test the run ID filtering logic in isolation
	testRuns := []WorkflowRun{
		{DatabaseID: 1000, WorkflowName: "Test Workflow"},
		{DatabaseID: 1500, WorkflowName: "Test Workflow"},
		{DatabaseID: 2000, WorkflowName: "Test Workflow"},
		{DatabaseID: 2500, WorkflowName: "Test Workflow"},
		{DatabaseID: 3000, WorkflowName: "Test Workflow"},
	}

	// Test before-run-id filter (exclusive)
	var filteredRuns []WorkflowRun
	beforeRunID := int64(2000)
	for _, run := range testRuns {
		if beforeRunID > 0 && run.DatabaseID >= beforeRunID {
			continue
		}
		filteredRuns = append(filteredRuns, run)
	}

	if len(filteredRuns) != 2 {
		t.Errorf("Expected 2 runs before ID 2000 (exclusive), got %d", len(filteredRuns))
	}
	if filteredRuns[0].DatabaseID != 1000 || filteredRuns[1].DatabaseID != 1500 {
		t.Errorf("Expected runs 1000 and 1500, got %d and %d", filteredRuns[0].DatabaseID, filteredRuns[1].DatabaseID)
	}

	// Test after-run-id filter (exclusive)
	filteredRuns = nil
	afterRunID := int64(2000)
	for _, run := range testRuns {
		if afterRunID > 0 && run.DatabaseID <= afterRunID {
			continue
		}
		filteredRuns = append(filteredRuns, run)
	}

	if len(filteredRuns) != 2 {
		t.Errorf("Expected 2 runs after ID 2000 (exclusive), got %d", len(filteredRuns))
	}
	if filteredRuns[0].DatabaseID != 2500 || filteredRuns[1].DatabaseID != 3000 {
		t.Errorf("Expected runs 2500 and 3000, got %d and %d", filteredRuns[0].DatabaseID, filteredRuns[1].DatabaseID)
	}

	// Test range filter (both before and after)
	filteredRuns = nil
	beforeRunID = int64(2500)
	afterRunID = int64(1000)
	for _, run := range testRuns {
		// Apply before-run-id filter (exclusive)
		if beforeRunID > 0 && run.DatabaseID >= beforeRunID {
			continue
		}
		// Apply after-run-id filter (exclusive)
		if afterRunID > 0 && run.DatabaseID <= afterRunID {
			continue
		}
		filteredRuns = append(filteredRuns, run)
	}

	if len(filteredRuns) != 2 {
		t.Errorf("Expected 2 runs in range (1000, 2500), got %d", len(filteredRuns))
	}
	if filteredRuns[0].DatabaseID != 1500 || filteredRuns[1].DatabaseID != 2000 {
		t.Errorf("Expected runs 1500 and 2000, got %d and %d", filteredRuns[0].DatabaseID, filteredRuns[1].DatabaseID)
	}
}

func TestRefFilteringWithGitHubCLI(t *testing.T) {
	// Test that ref filtering is properly added to GitHub CLI args
	// This is a unit test for the args construction, not a network test

	// Simulate args construction for ref filtering
	args := []string{"run", "list", "--json", "databaseId,number,url,status,conclusion,workflowName,createdAt,startedAt,updatedAt,event,headBranch,headSha,displayTitle"}

	ref := "feature-branch"
	if ref != "" {
		args = append(args, "--branch", ref)
	}

	// Verify that the ref filter was added correctly (uses --branch flag under the hood)
	found := false
	for i, arg := range args {
		if arg == "--branch" && i+1 < len(args) && args[i+1] == "feature-branch" {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("Expected ref filter '--branch feature-branch' not found in args: %v", args)
	}
}

func TestFindAgentLogFile(t *testing.T) {
	// Create a temporary directory structure for testing
	tmpDir := testutil.TempDir(t, "test-*")

	// Test 1: Copilot engine with agent_output directory
	t.Run("Copilot engine uses agent_output", func(t *testing.T) {
		copilotEngine := workflow.NewCopilotEngine()

		// Create agent_output directory with a log file
		agentOutputDir := filepath.Join(tmpDir, "copilot_test", "agent_output")
		err := os.MkdirAll(agentOutputDir, 0755)
		if err != nil {
			t.Fatalf("Failed to create agent_output directory: %v", err)
		}

		logFile := filepath.Join(agentOutputDir, "debug.log")
		err = os.WriteFile(logFile, []byte("test log content"), 0644)
		if err != nil {
			t.Fatalf("Failed to create log file: %v", err)
		}

		// Create agent-stdio.log as well (should be ignored for Copilot)
		stdioLog := filepath.Join(tmpDir, "copilot_test", "agent-stdio.log")
		err = os.WriteFile(stdioLog, []byte("stdio content"), 0644)
		if err != nil {
			t.Fatalf("Failed to create agent-stdio.log: %v", err)
		}

		// Test findAgentLogFile
		found, ok := findAgentLogFile(filepath.Join(tmpDir, "copilot_test"), copilotEngine)
		if !ok {
			t.Errorf("Expected to find agent log file for Copilot engine")
		}

		// Should find the file in agent_output directory
		if !strings.Contains(found, "agent_output") {
			t.Errorf("Expected to find file in agent_output directory, got: %s", found)
		}
	})

	// Test Copilot engine with flattened agent_outputs artifact
	// After flattening, session logs are at sandbox/agent/logs/ in the root
	t.Run("copilot_engine_flattened_location", func(t *testing.T) {
		copilotDir := filepath.Join(tmpDir, "copilot_flattened_test")
		err := os.MkdirAll(copilotDir, 0755)
		if err != nil {
			t.Fatalf("Failed to create test directory: %v", err)
		}

		// Create flattened session logs directory (after flattenAgentOutputsArtifact)
		sessionLogsDir := filepath.Join(copilotDir, "sandbox", "agent", "logs")
		err = os.MkdirAll(sessionLogsDir, 0755)
		if err != nil {
			t.Fatalf("Failed to create flattened session logs directory: %v", err)
		}

		// Create a test session log file
		sessionLog := filepath.Join(sessionLogsDir, "session-test-123.log")
		err = os.WriteFile(sessionLog, []byte("test session log content"), 0644)
		if err != nil {
			t.Fatalf("Failed to create session log file: %v", err)
		}

		copilotEngine := workflow.NewCopilotEngine()

		// Test findAgentLogFile - should find the session log in flattened location
		found, ok := findAgentLogFile(copilotDir, copilotEngine)
		if !ok {
			t.Errorf("Expected to find agent log file for Copilot engine in flattened location")
		}

		// Should find the session log file
		if !strings.HasSuffix(found, "session-test-123.log") {
			t.Errorf("Expected to find session-test-123.log, but found %s", found)
		}

		// Verify the path is correct
		expectedPath := filepath.Join(copilotDir, "sandbox", "agent", "logs", "session-test-123.log")
		if found != expectedPath {
			t.Errorf("Expected path %s, got %s", expectedPath, found)
		}
	})

	// Test Copilot engine with session log directly in run directory (recursive search)
	// This handles cases where artifact structure differs from expected
	t.Run("copilot_engine_recursive_search", func(t *testing.T) {
		copilotDir := filepath.Join(tmpDir, "copilot_recursive_test")
		err := os.MkdirAll(copilotDir, 0755)
		if err != nil {
			t.Fatalf("Failed to create test directory: %v", err)
		}

		// Create session log directly in the run directory
		// This simulates the case where the artifact was flattened differently
		sessionLog := filepath.Join(copilotDir, "session-direct-456.log")
		err = os.WriteFile(sessionLog, []byte("test session log content"), 0644)
		if err != nil {
			t.Fatalf("Failed to create session log file: %v", err)
		}

		copilotEngine := workflow.NewCopilotEngine()

		// Test findAgentLogFile - should find via recursive search
		found, ok := findAgentLogFile(copilotDir, copilotEngine)
		if !ok {
			t.Errorf("Expected to find agent log file via recursive search")
		}

		// Should find the session log file
		if !strings.HasSuffix(found, "session-direct-456.log") {
			t.Errorf("Expected to find session-direct-456.log, but found %s", found)
		}

		// Verify the path is correct
		if found != sessionLog {
			t.Errorf("Expected path %s, got %s", sessionLog, found)
		}
	})

	// Test Copilot engine with process log (new naming convention)
	// Copilot changed from session-*.log to process-*.log
	t.Run("copilot_engine_process_log", func(t *testing.T) {
		copilotDir := filepath.Join(tmpDir, "copilot_process_test")
		err := os.MkdirAll(copilotDir, 0755)
		if err != nil {
			t.Fatalf("Failed to create test directory: %v", err)
		}

		// Create process log directly in the run directory
		// This simulates the new naming convention for Copilot logs
		processLog := filepath.Join(copilotDir, "process-12345.log")
		err = os.WriteFile(processLog, []byte("test process log content"), 0644)
		if err != nil {
			t.Fatalf("Failed to create process log file: %v", err)
		}

		copilotEngine := workflow.NewCopilotEngine()

		// Test findAgentLogFile - should find via recursive search
		found, ok := findAgentLogFile(copilotDir, copilotEngine)
		if !ok {
			t.Errorf("Expected to find agent log file via recursive search")
		}

		// Should find the process log file
		if !strings.HasSuffix(found, "process-12345.log") {
			t.Errorf("Expected to find process-12345.log, but found %s", found)
		}

		// Verify the path is correct
		if found != processLog {
			t.Errorf("Expected path %s, got %s", processLog, found)
		}
	})

	// Test Copilot engine with process log in nested directory
	t.Run("copilot_engine_process_log_nested", func(t *testing.T) {
		copilotDir := filepath.Join(tmpDir, "copilot_process_nested_test")
		err := os.MkdirAll(copilotDir, 0755)
		if err != nil {
			t.Fatalf("Failed to create test directory: %v", err)
		}

		// Create nested directory structure
		processLogsDir := filepath.Join(copilotDir, "sandbox", "agent", "logs")
		err = os.MkdirAll(processLogsDir, 0755)
		if err != nil {
			t.Fatalf("Failed to create process logs directory: %v", err)
		}

		// Create a test process log file
		processLog := filepath.Join(processLogsDir, "process-test-789.log")
		err = os.WriteFile(processLog, []byte("test process log content"), 0644)
		if err != nil {
			t.Fatalf("Failed to create process log file: %v", err)
		}

		copilotEngine := workflow.NewCopilotEngine()

		// Test findAgentLogFile - should find the process log in nested location
		found, ok := findAgentLogFile(copilotDir, copilotEngine)
		if !ok {
			t.Errorf("Expected to find agent log file for Copilot engine in nested location")
		}

		// Should find the process log file
		if !strings.HasSuffix(found, "process-test-789.log") {
			t.Errorf("Expected to find process-test-789.log, but found %s", found)
		}

		// Verify the path is correct
		expectedPath := filepath.Join(copilotDir, "sandbox", "agent", "logs", "process-test-789.log")
		if found != expectedPath {
			t.Errorf("Expected path %s, got %s", expectedPath, found)
		}
	})

	// Test 2: Claude engine with agent-stdio.log
	t.Run("Claude engine uses agent-stdio.log", func(t *testing.T) {
		claudeEngine := workflow.NewClaudeEngine()

		// Create only agent-stdio.log
		claudeDir := filepath.Join(tmpDir, "claude_test")
		err := os.MkdirAll(claudeDir, 0755)
		if err != nil {
			t.Fatalf("Failed to create claude test directory: %v", err)
		}

		stdioLog := filepath.Join(claudeDir, "agent-stdio.log")
		err = os.WriteFile(stdioLog, []byte("stdio content"), 0644)
		if err != nil {
			t.Fatalf("Failed to create agent-stdio.log: %v", err)
		}

		// Test findAgentLogFile
		found, ok := findAgentLogFile(claudeDir, claudeEngine)
		if !ok {
			t.Errorf("Expected to find agent log file for Claude engine")
		}

		// Should find agent-stdio.log
		if !strings.Contains(found, "agent-stdio.log") {
			t.Errorf("Expected to find agent-stdio.log, got: %s", found)
		}
	})

	// Test 3: No logs found
	t.Run("No logs found returns false", func(t *testing.T) {
		emptyDir := filepath.Join(tmpDir, "empty_test")
		err := os.MkdirAll(emptyDir, 0755)
		if err != nil {
			t.Fatalf("Failed to create empty test directory: %v", err)
		}

		claudeEngine := workflow.NewClaudeEngine()
		_, ok := findAgentLogFile(emptyDir, claudeEngine)
		if ok {
			t.Errorf("Expected to not find agent log file in empty directory")
		}
	})

	// Test 4: Codex engine with agent-stdio.log
	t.Run("Codex engine uses agent-stdio.log", func(t *testing.T) {
		codexEngine := workflow.NewCodexEngine()

		// Create only agent-stdio.log
		codexDir := filepath.Join(tmpDir, "codex_test")
		err := os.MkdirAll(codexDir, 0755)
		if err != nil {
			t.Fatalf("Failed to create codex test directory: %v", err)
		}

		stdioLog := filepath.Join(codexDir, "agent-stdio.log")
		err = os.WriteFile(stdioLog, []byte("stdio content"), 0644)
		if err != nil {
			t.Fatalf("Failed to create agent-stdio.log: %v", err)
		}

		// Test findAgentLogFile
		found, ok := findAgentLogFile(codexDir, codexEngine)
		if !ok {
			t.Errorf("Expected to find agent log file for Codex engine")
		}

		// Should find agent-stdio.log
		if !strings.Contains(found, "agent-stdio.log") {
			t.Errorf("Expected to find agent-stdio.log, got: %s", found)
		}
	})
}

func TestUnzipFile(t *testing.T) {
	// Create a temporary directory for the test
	tmpDir := testutil.TempDir(t, "test-*")

	// Create a test zip file
	zipPath := filepath.Join(tmpDir, "test.zip")
	zipFile, err := os.Create(zipPath)
	if err != nil {
		t.Fatalf("Failed to create test zip file: %v", err)
	}

	zipWriter := zip.NewWriter(zipFile)

	// Add a test file to the zip
	testContent := "This is test content for workflow logs"
	writer, err := zipWriter.Create("test-log.txt")
	if err != nil {
		zipFile.Close()
		t.Fatalf("Failed to create file in zip: %v", err)
	}
	_, err = writer.Write([]byte(testContent))
	if err != nil {
		zipFile.Close()
		t.Fatalf("Failed to write content to zip: %v", err)
	}

	// Add a subdirectory with a file
	writer, err = zipWriter.Create("logs/job-1.txt")
	if err != nil {
		zipFile.Close()
		t.Fatalf("Failed to create subdirectory file in zip: %v", err)
	}
	_, err = writer.Write([]byte("Job 1 logs"))
	if err != nil {
		zipFile.Close()
		t.Fatalf("Failed to write subdirectory content to zip: %v", err)
	}

	// Close the zip writer
	err = zipWriter.Close()
	if err != nil {
		zipFile.Close()
		t.Fatalf("Failed to close zip writer: %v", err)
	}
	zipFile.Close()

	// Create a destination directory
	destDir := filepath.Join(tmpDir, "extracted")
	err = os.MkdirAll(destDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create destination directory: %v", err)
	}

	// Test the unzipFile function
	err = unzipFile(zipPath, destDir, false)
	if err != nil {
		t.Fatalf("unzipFile failed: %v", err)
	}

	// Verify the extracted files
	extractedFile := filepath.Join(destDir, "test-log.txt")
	content, err := os.ReadFile(extractedFile)
	if err != nil {
		t.Fatalf("Failed to read extracted file: %v", err)
	}

	if string(content) != testContent {
		t.Errorf("Extracted content mismatch: got %q, want %q", string(content), testContent)
	}

	// Verify subdirectory file
	subdirFile := filepath.Join(destDir, "logs", "job-1.txt")
	content, err = os.ReadFile(subdirFile)
	if err != nil {
		t.Fatalf("Failed to read extracted subdirectory file: %v", err)
	}

	if string(content) != "Job 1 logs" {
		t.Errorf("Extracted subdirectory content mismatch: got %q, want %q", string(content), "Job 1 logs")
	}
}

func TestUnzipFileZipSlipPrevention(t *testing.T) {
	// Create a temporary directory for the test
	tmpDir := testutil.TempDir(t, "test-*")

	// Create a test zip file with a malicious path
	zipPath := filepath.Join(tmpDir, "malicious.zip")
	zipFile, err := os.Create(zipPath)
	if err != nil {
		t.Fatalf("Failed to create test zip file: %v", err)
	}

	zipWriter := zip.NewWriter(zipFile)

	// Try to create a file that escapes the destination directory
	writer, err := zipWriter.Create("../../../etc/passwd")
	if err != nil {
		zipFile.Close()
		t.Fatalf("Failed to create malicious file in zip: %v", err)
	}
	_, err = writer.Write([]byte("malicious content"))
	if err != nil {
		zipFile.Close()
		t.Fatalf("Failed to write malicious content to zip: %v", err)
	}

	err = zipWriter.Close()
	if err != nil {
		zipFile.Close()
		t.Fatalf("Failed to close zip writer: %v", err)
	}
	zipFile.Close()

	// Create a destination directory
	destDir := filepath.Join(tmpDir, "safe-extraction")
	err = os.MkdirAll(destDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create destination directory: %v", err)
	}

	// Test that unzipFile rejects the malicious path
	err = unzipFile(zipPath, destDir, false)
	if err == nil {
		t.Error("Expected unzipFile to reject malicious path, but it succeeded")
	}

	if !strings.Contains(err.Error(), "invalid file path") {
		t.Errorf("Expected error about invalid file path, got: %v", err)
	}
}

func TestDownloadWorkflowRunLogsStructure(t *testing.T) {
	// This test verifies that workflow logs are extracted into a workflow-logs subdirectory
	// Note: This test cannot fully test downloadWorkflowRunLogs without GitHub CLI authentication
	// So we test the directory creation and unzipFile behavior that mimics the workflow

	tmpDir := testutil.TempDir(t, "test-*")

	// Create a mock workflow logs zip file similar to what GitHub API returns
	zipPath := filepath.Join(tmpDir, "workflow-logs.zip")
	zipFile, err := os.Create(zipPath)
	if err != nil {
		t.Fatalf("Failed to create test zip file: %v", err)
	}

	zipWriter := zip.NewWriter(zipFile)

	// Add files that simulate GitHub Actions workflow logs structure
	logFiles := map[string]string{
		"1_job1.txt":        "Job 1 execution logs",
		"2_job2.txt":        "Job 2 execution logs",
		"3_build/build.txt": "Build step logs",
		"4_test/test-1.txt": "Test step 1 logs",
		"4_test/test-2.txt": "Test step 2 logs",
	}

	for filename, content := range logFiles {
		writer, err := zipWriter.Create(filename)
		if err != nil {
			zipFile.Close()
			t.Fatalf("Failed to create file %s in zip: %v", filename, err)
		}
		_, err = writer.Write([]byte(content))
		if err != nil {
			zipFile.Close()
			t.Fatalf("Failed to write content to %s: %v", filename, err)
		}
	}

	err = zipWriter.Close()
	if err != nil {
		zipFile.Close()
		t.Fatalf("Failed to close zip writer: %v", err)
	}
	zipFile.Close()

	// Create a run directory (simulating logs/run-12345)
	runDir := filepath.Join(tmpDir, "run-12345")
	err = os.MkdirAll(runDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create run directory: %v", err)
	}

	// Create some other artifacts in the run directory (to verify they don't get mixed with logs)
	err = os.WriteFile(filepath.Join(runDir, "aw_info.json"), []byte(`{"engine_id": "claude"}`), 0644)
	if err != nil {
		t.Fatalf("Failed to create aw_info.json: %v", err)
	}

	// Create the workflow-logs subdirectory and extract logs there
	workflowLogsDir := filepath.Join(runDir, "workflow-logs")
	err = os.MkdirAll(workflowLogsDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create workflow-logs directory: %v", err)
	}

	// Extract logs into the workflow-logs subdirectory (mimics downloadWorkflowRunLogs behavior)
	err = unzipFile(zipPath, workflowLogsDir, false)
	if err != nil {
		t.Fatalf("Failed to extract logs: %v", err)
	}

	// Verify that workflow-logs directory exists
	if !fileutil.DirExists(workflowLogsDir) {
		t.Error("workflow-logs directory should exist")
	}

	// Verify that log files are in the workflow-logs subdirectory, not in run root
	for filename := range logFiles {
		expectedPath := filepath.Join(workflowLogsDir, filename)
		if !fileutil.FileExists(expectedPath) {
			t.Errorf("Expected log file %s to be in workflow-logs subdirectory", filename)
		}

		// Verify the file is NOT in the run directory root
		wrongPath := filepath.Join(runDir, filename)
		if fileutil.FileExists(wrongPath) {
			t.Errorf("Log file %s should not be in run directory root", filename)
		}
	}

	// Verify that other artifacts remain in the run directory root
	awInfoPath := filepath.Join(runDir, "aw_info.json")
	if !fileutil.FileExists(awInfoPath) {
		t.Error("aw_info.json should remain in run directory root")
	}

	// Verify the content of one of the extracted log files
	testLogPath := filepath.Join(workflowLogsDir, "1_job1.txt")
	content, err := os.ReadFile(testLogPath)
	if err != nil {
		t.Fatalf("Failed to read extracted log file: %v", err)
	}

	expectedContent := "Job 1 execution logs"
	if string(content) != expectedContent {
		t.Errorf("Log file content mismatch: got %q, want %q", string(content), expectedContent)
	}

	// Verify nested directory structure is preserved
	nestedLogPath := filepath.Join(workflowLogsDir, "3_build", "build.txt")
	if !fileutil.FileExists(nestedLogPath) {
		t.Error("Nested log directory structure should be preserved")
	}
}

// TestCountParameterBehavior verifies that the count parameter limits matching results
// not the number of runs fetched when date filters are specified
func TestCountParameterBehavior(t *testing.T) {
	// This test documents the expected behavior:
	// 1. When date filters (startDate/endDate) are specified, fetch ALL runs in that range
	// 2. Apply post-download filters (engine, staged, etc.)
	// 3. Limit final output to 'count' matching runs
	//
	// Without date filters:
	// 1. Fetch runs iteratively until we have 'count' runs with artifacts
	// 2. Apply filters during iteration (old behavior for backward compatibility)

	// Note: This is a documentation test - the actual logic is tested via integration tests
	// that require GitHub CLI authentication and a real repository

	tests := []struct {
		name             string
		startDate        string
		endDate          string
		count            int
		expectedFetchAll bool
	}{
		{
			name:             "with startDate should fetch all in range",
			startDate:        "2024-01-01",
			endDate:          "",
			count:            10,
			expectedFetchAll: true,
		},
		{
			name:             "with endDate should fetch all in range",
			startDate:        "",
			endDate:          "2024-12-31",
			count:            10,
			expectedFetchAll: true,
		},
		{
			name:             "with both dates should fetch all in range",
			startDate:        "2024-01-01",
			endDate:          "2024-12-31",
			count:            10,
			expectedFetchAll: true,
		},
		{
			name:             "without dates should use count as fetch limit",
			startDate:        "",
			endDate:          "",
			count:            10,
			expectedFetchAll: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This documents the logic: when startDate or endDate is set, we fetch all
			fetchAllInRange := tt.startDate != "" || tt.endDate != ""

			if fetchAllInRange != tt.expectedFetchAll {
				t.Errorf("Expected fetchAllInRange=%v for startDate=%q endDate=%q, got %v",
					tt.expectedFetchAll, tt.startDate, tt.endDate, fetchAllInRange)
			}
		})
	}
}
