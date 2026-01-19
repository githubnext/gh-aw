package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/githubnext/gh-aw/pkg/stringutil"
	"github.com/githubnext/gh-aw/pkg/testutil"
)

func TestAccessLogParsing(t *testing.T) {
	// Create a temporary directory for the test
	tempDir := testutil.TempDir(t, "test-*")

	// Create test access.log content
	testLogContent := `1701234567.123    180 192.168.1.100 TCP_MISS/200 1234 GET http://example.com/api/data - HIER_DIRECT/93.184.216.34 text/html
1701234568.456    250 192.168.1.100 TCP_DENIED/403 0 CONNECT github.com:443 - HIER_NONE/- -
1701234569.789    120 192.168.1.100 TCP_HIT/200 5678 GET http://api.github.com/repos - HIER_DIRECT/140.82.112.6 application/json
1701234570.012    0 192.168.1.100 TCP_DENIED/403 0 GET http://malicious.site/evil - HIER_NONE/- -`

	// Write test log file
	accessLogPath := filepath.Join(tempDir, "access.log")
	err := os.WriteFile(accessLogPath, []byte(testLogContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test access.log: %v", err)
	}

	// Test parsing
	analysis, err := parseSquidAccessLog(accessLogPath, false)
	if err != nil {
		t.Fatalf("Failed to parse access log: %v", err)
	}

	// Verify results
	if analysis.TotalRequests != 4 {
		t.Errorf("Expected 4 total requests, got %d", analysis.TotalRequests)
	}

	if analysis.AllowedCount != 2 {
		t.Errorf("Expected 2 allowed requests, got %d", analysis.AllowedCount)
	}

	if analysis.BlockedCount != 2 {
		t.Errorf("Expected 2 denied requests, got %d", analysis.BlockedCount)
	}

	// Check allowed domains
	expectedAllowed := []string{"api.github.com", "example.com"}
	if len(analysis.AllowedDomains) != len(expectedAllowed) {
		t.Errorf("Expected %d allowed domains, got %d", len(expectedAllowed), len(analysis.AllowedDomains))
	}
}

func TestMultipleAccessLogAnalysis(t *testing.T) {
	// Create a temporary directory for the test
	tempDir := testutil.TempDir(t, "test-*")
	accessLogsDir := filepath.Join(tempDir, "access.log")
	err := os.MkdirAll(accessLogsDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create access.log directory: %v", err)
	}

	// Create test access log content for multiple MCP servers
	fetchLogContent := `1701234567.123    180 192.168.1.100 TCP_MISS/200 1234 GET http://example.com/api/data - HIER_DIRECT/93.184.216.34 text/html
1701234568.456    250 192.168.1.100 TCP_HIT/200 5678 GET http://api.github.com/repos - HIER_DIRECT/140.82.112.6 application/json`

	browserLogContent := `1701234569.789    120 192.168.1.100 TCP_DENIED/403 0 CONNECT github.com:443 - HIER_NONE/- -
1701234570.012    0 192.168.1.100 TCP_DENIED/403 0 GET http://malicious.site/evil - HIER_NONE/- -`

	// Write separate log files for different MCP servers
	fetchLogPath := filepath.Join(accessLogsDir, "access-fetch.log")
	err = os.WriteFile(fetchLogPath, []byte(fetchLogContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test access-fetch.log: %v", err)
	}

	browserLogPath := filepath.Join(accessLogsDir, "access-browser.log")
	err = os.WriteFile(browserLogPath, []byte(browserLogContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test access-browser.log: %v", err)
	}

	// Test analysis of multiple access logs
	analysis, err := analyzeMultipleAccessLogs(accessLogsDir, false)
	if err != nil {
		t.Fatalf("Failed to analyze multiple access logs: %v", err)
	}

	// Verify aggregated results
	if analysis.TotalRequests != 4 {
		t.Errorf("Expected 4 total requests, got %d", analysis.TotalRequests)
	}

	if analysis.AllowedCount != 2 {
		t.Errorf("Expected 2 allowed requests, got %d", analysis.AllowedCount)
	}

	if analysis.BlockedCount != 2 {
		t.Errorf("Expected 2 denied requests, got %d", analysis.BlockedCount)
	}

	// Check allowed domains
	expectedAllowed := []string{"api.github.com", "example.com"}
	if len(analysis.AllowedDomains) != len(expectedAllowed) {
		t.Errorf("Expected %d allowed domains, got %d", len(expectedAllowed), len(analysis.AllowedDomains))
	}

	// Check blocked domains
	expectedDenied := []string{"github.com", "malicious.site"}
	if len(analysis.BlockedDomains) != len(expectedDenied) {
		t.Errorf("Expected %d blocked domains, got %d", len(expectedDenied), len(analysis.BlockedDomains))
	}
}

func TestAnalyzeAccessLogsDirectory(t *testing.T) {
	// Create a temporary directory structure
	tempDir := testutil.TempDir(t, "test-*")

	// Test case 1: Multiple access logs in access-logs subdirectory
	accessLogsDir := filepath.Join(tempDir, "run1", "access.log")
	err := os.MkdirAll(accessLogsDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create access.log directory: %v", err)
	}

	fetchLogContent := `1701234567.123    180 192.168.1.100 TCP_MISS/200 1234 GET http://example.com/api/data - HIER_DIRECT/93.184.216.34 text/html`
	fetchLogPath := filepath.Join(accessLogsDir, "access-fetch.log")
	err = os.WriteFile(fetchLogPath, []byte(fetchLogContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test access-fetch.log: %v", err)
	}

	analysis, err := analyzeAccessLogs(filepath.Join(tempDir, "run1"), false)
	if err != nil {
		t.Fatalf("Failed to analyze access logs: %v", err)
	}

	if analysis == nil {
		t.Fatal("Expected analysis result, got nil")
	}

	if analysis.TotalRequests != 1 {
		t.Errorf("Expected 1 total request, got %d", analysis.TotalRequests)
	}

	// Test case 2: No access logs
	run2Dir := filepath.Join(tempDir, "run2")
	err = os.MkdirAll(run2Dir, 0755)
	if err != nil {
		t.Fatalf("Failed to create run2 directory: %v", err)
	}

	analysis, err = analyzeAccessLogs(run2Dir, false)
	if err != nil {
		t.Fatalf("Failed to analyze no access logs: %v", err)
	}

	if analysis != nil {
		t.Errorf("Expected nil analysis for no access logs, got %+v", analysis)
	}
}

func TestExtractDomainFromURL(t *testing.T) {
	tests := []struct {
		url      string
		expected string
	}{
		{"http://example.com/path", "example.com"},
		{"https://api.github.com/repos", "api.github.com"},
		{"github.com:443", "github.com"},
		{"malicious.site", "malicious.site"},
		{"http://sub.domain.com:8080/path", "sub.domain.com"},
	}

	for _, test := range tests {
		result := stringutil.ExtractDomainFromURL(test.url)
		if result != test.expected {
			t.Errorf("stringutil.ExtractDomainFromURL(%q) = %q, expected %q", test.url, result, test.expected)
		}
	}
}
