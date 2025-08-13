package cli

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/githubnext/gh-aw/pkg/workflow"
)

func TestDownloadWorkflowLogs(t *testing.T) {
	// Test the DownloadWorkflowLogs function
	// This should either fail with auth error (if not authenticated)
	// or succeed with no results (if authenticated but no workflows match)
	err := DownloadWorkflowLogs("", 1, "", "", "./test-logs", false)

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

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		duration time.Duration
		expected string
	}{
		{30 * time.Second, "30s"},
		{90 * time.Second, "1.5m"},
		{2 * time.Hour, "2.0h"},
		{45 * time.Minute, "45.0m"},
	}

	for _, tt := range tests {
		result := formatDuration(tt.duration)
		if result != tt.expected {
			t.Errorf("formatDuration(%v) = %q, expected %q", tt.duration, result, tt.expected)
		}
	}
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
		result := formatNumber(test.input)
		if result != test.expected {
			t.Errorf("formatNumber(%d) = %s, expected %s", test.input, result, test.expected)
		}
	}
}

func TestParseLogFileWithoutAwInfo(t *testing.T) {
	// Create a temporary log file
	tmpDir := t.TempDir()
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
	metrics, err := parseLogFileWithEngine(logFile, nil, false)
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
	tmpDir := t.TempDir()
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

	metrics, err := parseLogFileWithEngine(logFile, nil, false)
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
		value    interface{}
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
		value    interface{}
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
	tmpDir := t.TempDir()

	// Test existing directory
	if !dirExists(tmpDir) {
		t.Errorf("dirExists should return true for existing directory")
	}

	// Test non-existing directory
	nonExistentDir := filepath.Join(tmpDir, "does-not-exist")
	if dirExists(nonExistentDir) {
		t.Errorf("dirExists should return false for non-existing directory")
	}

	// Test file vs directory
	testFile := filepath.Join(tmpDir, "testfile")
	err := os.WriteFile(testFile, []byte("test"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	if dirExists(testFile) {
		t.Errorf("dirExists should return false for a file")
	}
}

func TestIsDirEmpty(t *testing.T) {
	tmpDir := t.TempDir()

	// Test empty directory
	emptyDir := filepath.Join(tmpDir, "empty")
	err := os.Mkdir(emptyDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create empty directory: %v", err)
	}

	if !isDirEmpty(emptyDir) {
		t.Errorf("isDirEmpty should return true for empty directory")
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

	if isDirEmpty(nonEmptyDir) {
		t.Errorf("isDirEmpty should return false for directory with files")
	}

	// Test non-existing directory
	nonExistentDir := filepath.Join(tmpDir, "does-not-exist")
	if !isDirEmpty(nonExistentDir) {
		t.Errorf("isDirEmpty should return true for non-existing directory")
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
	runs, err := listWorkflowRunsWithPagination("nonexistent-workflow", 5, "", "", "2024-01-01T00:00:00Z", false)

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
		data     map[string]interface{}
		expected float64
	}{
		{
			name:     "total_cost_usd field",
			data:     map[string]interface{}{"total_cost_usd": 0.8606770999999999},
			expected: 0.8606770999999999,
		},
		{
			name:     "traditional cost field",
			data:     map[string]interface{}{"cost": 1.23},
			expected: 1.23,
		},
		{
			name:     "nested billing cost",
			data:     map[string]interface{}{"billing": map[string]interface{}{"cost": 2.45}},
			expected: 2.45,
		},
		{
			name:     "no cost fields",
			data:     map[string]interface{}{"tokens": 1000},
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
	tmpDir := t.TempDir()
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
	metrics, err := parseLogFileWithEngine(logFile, claudeEngine, false)
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

	// Duration is no longer extracted from logs - using GitHub API timestamps instead
}

func TestParseLogFileWithCodexFormat(t *testing.T) {
	// Create a temporary log file with the Codex output format from the issue
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "test-codex.log")

	// This is the exact Codex format provided in the issue
	logContent := `[2025-08-13T00:24:45] Starting Codex workflow execution
[2025-08-13T00:24:50] codex

I'm ready to generate a Codex PR summary, but I need the pull request number to fetch its details. Could you please share the PR number (and confirm the repo/owner if it isn't ` + "`githubnext/gh-aw`" + `)?
[2025-08-13T00:24:50] tokens used: 13934
[2025-08-13T00:24:55] Workflow completed successfully`

	err := os.WriteFile(logFile, []byte(logContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test log file: %v", err)
	}

	// Test with Codex engine to parse Codex-specific logs
	codexEngine := workflow.NewCodexEngine()
	metrics, err := parseLogFileWithEngine(logFile, codexEngine, false)
	if err != nil {
		t.Fatalf("parseLogFileWithEngine failed: %v", err)
	}

	// Check token usage extraction from Codex format
	expectedTokens := 13934
	if metrics.TokenUsage != expectedTokens {
		t.Errorf("Expected token usage %d, got %d", expectedTokens, metrics.TokenUsage)
	}

	// Duration is no longer extracted from logs - using GitHub API timestamps instead
}

func TestParseLogFileWithCodexTokenSumming(t *testing.T) {
	// Create a temporary log file with multiple Codex token entries
	tmpDir := t.TempDir()
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

	// Get the Codex engine for testing
	registry := workflow.NewEngineRegistry()
	codexEngine, err := registry.GetEngine("codex")
	if err != nil {
		t.Fatalf("Failed to get Codex engine: %v", err)
	}

	metrics, err := parseLogFileWithEngine(logFile, codexEngine, false)
	if err != nil {
		t.Fatalf("parseLogFile failed: %v", err)
	}

	// Should sum all Codex token entries: 32169 + 28828 + 5000 = 65997
	expectedTokens := 32169 + 28828 + 5000
	if metrics.TokenUsage != expectedTokens {
		t.Errorf("Expected token usage %d (sum of all Codex entries), got %d", expectedTokens, metrics.TokenUsage)
	}
}

func TestParseLogFileWithMixedTokenFormats(t *testing.T) {
	// Create a temporary log file with mixed token formats
	tmpDir := t.TempDir()
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

	metrics, err := parseLogFileWithEngine(logFile, codexEngine, false)
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
	tmpDir := t.TempDir()

	// Test Case 1: aw_info.json as a regular file
	awInfoFile := filepath.Join(tmpDir, "aw_info.json")
	awInfoContent := `{
		"engine_id": "claude",
		"engine_name": "Claude",
		"model": "claude-3-sonnet",
		"version": "20240620",
		"workflow_name": "Test Claude",
		"experimental": false,
		"supports_tools_whitelist": true,
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
		"supports_tools_whitelist": true,
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
	tmpDir := t.TempDir()
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
	metrics, err := parseLogFileWithEngine(logFile, nil, false)
	if err != nil {
		t.Fatalf("parseLogFileWithEngine failed: %v", err)
	}

	// Without engine-specific parsing, should return 0
	if metrics.TokenUsage != 0 {
		t.Errorf("Expected token usage 0 (no aw_info.json), got %d", metrics.TokenUsage)
	}
}
