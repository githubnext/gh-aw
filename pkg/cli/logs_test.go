package cli

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDownloadWorkflowLogs(t *testing.T) {
	// Test the DownloadWorkflowLogs function
	// This should either fail with auth error (if not authenticated)
	// or succeed with no results (if authenticated but no workflows match)
	err := DownloadWorkflowLogs("", 1, "", "", "./test-logs", false)

	// If GitHub CLI is authenticated, the function may succeed but find no results
	// If not authenticated, it should return an auth error
	if err != nil {
		// If there's an error, it should be an authentication error
		if !strings.Contains(err.Error(), "authentication required") {
			t.Errorf("Expected authentication error or no error, got: %v", err)
		}
	}
	// If err is nil, that's also acceptable (authenticated case with no results)

	// Clean up
	os.RemoveAll("./test-logs")
}

func TestExtractTokenUsage(t *testing.T) {
	tests := []struct {
		line     string
		expected int
	}{
		{"tokens: 1234", 1234},
		{"token_count: 567", 567},
		{"input_tokens: 890", 890},
		{"Total tokens used: 999", 999},
		{"tokens used: 13934", 13934},                       // Codex format
		{"[2025-08-13T00:24:50] tokens used: 13934", 13934}, // Codex format with timestamp
		{"no token info here", 0},
		{"tokens: invalid", 0},
	}

	for _, tt := range tests {
		result := extractTokenUsage(tt.line)
		if result != tt.expected {
			t.Errorf("extractTokenUsage(%q) = %d, expected %d", tt.line, result, tt.expected)
		}
	}
}

func TestExtractCost(t *testing.T) {
	tests := []struct {
		line     string
		expected float64
	}{
		{"cost: $1.23", 1.23},
		{"price: 0.45", 0.45},
		{"Total cost: $99.99", 99.99},
		{"$5.67 spent", 5.67},
		{"no cost info here", 0},
		{"cost: invalid", 0},
	}

	for _, tt := range tests {
		result := extractCost(tt.line)
		if result != tt.expected {
			t.Errorf("extractCost(%q) = %f, expected %f", tt.line, result, tt.expected)
		}
	}
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

func TestParseLogFile(t *testing.T) {
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

	metrics, err := parseLogFile(logFile, false)
	if err != nil {
		t.Fatalf("parseLogFile failed: %v", err)
	}

	// Check token usage (should pick up the highest individual value: 2100)
	if metrics.TokenUsage != 2100 {
		t.Errorf("Expected token usage 2100, got %d", metrics.TokenUsage)
	}

	// Check cost
	if metrics.EstimatedCost != 0.025 {
		t.Errorf("Expected cost 0.025, got %f", metrics.EstimatedCost)
	}

	// Check duration (90 seconds between start and end)
	expectedDuration := 90 * time.Second
	if metrics.Duration != expectedDuration {
		t.Errorf("Expected duration %v, got %v", expectedDuration, metrics.Duration)
	}
}

func TestExtractJSONMetrics(t *testing.T) {
	tests := []struct {
		name            string
		line            string
		expectedTokens  int
		expectedCost    float64
		expectTimestamp bool
	}{
		{
			name:           "Claude streaming format with usage",
			line:           `{"type": "message_delta", "delta": {"usage": {"input_tokens": 123, "output_tokens": 456}}}`,
			expectedTokens: 579, // 123 + 456
		},
		{
			name:            "Simple token count",
			line:            `{"tokens": 1234, "timestamp": "2024-01-15T10:30:00Z"}`,
			expectedTokens:  1234,
			expectTimestamp: true,
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

			if tt.expectTimestamp && metrics.Timestamp.IsZero() {
				t.Error("Expected timestamp to be parsed, but got zero value")
			}
		})
	}
}

func TestParseLogFileWithJSON(t *testing.T) {
	// Create a temporary log file with mixed JSON and text format
	tmpDir := t.TempDir()
	logFile := filepath.Join(tmpDir, "test-mixed.log")

	logContent := `2024-01-15T10:30:00Z Starting workflow execution
{"type": "message_start", "timestamp": "2024-01-15T10:30:15Z"}
{"type": "content_block_delta", "delta": {"type": "text", "text": "Hello"}}
{"type": "message_delta", "delta": {"usage": {"input_tokens": 150, "output_tokens": 200}}}
Regular log line: tokens: 1000
{"cost": 0.035, "timestamp": "2024-01-15T10:31:00Z"}
2024-01-15T10:31:30Z Workflow completed successfully`

	err := os.WriteFile(logFile, []byte(logContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test log file: %v", err)
	}

	metrics, err := parseLogFile(logFile, false)
	if err != nil {
		t.Fatalf("parseLogFile failed: %v", err)
	}

	// Should pick up the highest token usage (1000 from text vs 350 from JSON)
	if metrics.TokenUsage != 1000 {
		t.Errorf("Expected token usage 1000, got %d", metrics.TokenUsage)
	}

	// Should accumulate cost from JSON
	if metrics.EstimatedCost != 0.035 {
		t.Errorf("Expected cost 0.035, got %f", metrics.EstimatedCost)
	}

	// Check duration (90 seconds between start and end)
	expectedDuration := 90 * time.Second
	if metrics.Duration != expectedDuration {
		t.Errorf("Expected duration %v, got %v", expectedDuration, metrics.Duration)
	}
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
		result := convertToInt(tt.value)
		if result != tt.expected {
			t.Errorf("convertToInt(%v) = %d, expected %d", tt.value, result, tt.expected)
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
		result := convertToFloat(tt.value)
		if result != tt.expected {
			t.Errorf("convertToFloat(%v) = %f, expected %f", tt.value, result, tt.expected)
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
			result := extractJSONCost(tt.data)
			if result != tt.expected {
				t.Errorf("extractJSONCost() = %f, expected %f", result, tt.expected)
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

	metrics, err := parseLogFile(logFile, false)
	if err != nil {
		t.Fatalf("parseLogFile failed: %v", err)
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

	// Check duration (150 seconds between start and end)
	expectedDuration := 150 * time.Second
	if metrics.Duration != expectedDuration {
		t.Errorf("Expected duration %v, got %v", expectedDuration, metrics.Duration)
	}
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

	metrics, err := parseLogFile(logFile, false)
	if err != nil {
		t.Fatalf("parseLogFile failed: %v", err)
	}

	// Check token usage extraction from Codex format
	expectedTokens := 13934
	if metrics.TokenUsage != expectedTokens {
		t.Errorf("Expected token usage %d, got %d", expectedTokens, metrics.TokenUsage)
	}

	// Check duration (10 seconds between start and end)
	expectedDuration := 10 * time.Second
	if metrics.Duration != expectedDuration {
		t.Errorf("Expected duration %v, got %v", expectedDuration, metrics.Duration)
	}
}

func TestExtractTokenUsageCodexPatterns(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		expected int
	}{
		{
			name:     "Codex basic format",
			line:     "tokens used: 13934",
			expected: 13934,
		},
		{
			name:     "Codex format with timestamp",
			line:     "[2025-08-13T00:24:50] tokens used: 13934",
			expected: 13934,
		},
		{
			name:     "Codex format with different timestamp",
			line:     "[2024-12-01T15:30:45] tokens used: 5678",
			expected: 5678,
		},
		{
			name:     "Codex format mixed with other text",
			line:     "Processing completed. tokens used: 999 - Summary generated",
			expected: 999,
		},
		{
			name:     "Standard format still works",
			line:     "tokens: 1234",
			expected: 1234,
		},
		{
			name:     "Total tokens used format",
			line:     "total tokens used: 4567",
			expected: 4567,
		},
		{
			name:     "No token info",
			line:     "[2025-08-13T00:24:50] codex processing",
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractTokenUsage(tt.line)
			if result != tt.expected {
				t.Errorf("extractTokenUsage(%q) = %d, expected %d", tt.line, result, tt.expected)
			}
		})
	}
}
