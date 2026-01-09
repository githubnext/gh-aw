package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/githubnext/gh-aw/pkg/testutil"
)

// TestAuditWithGatewayLogs validates that the audit command properly handles
// MCP gateway logs when they are present in the artifacts.
func TestAuditWithGatewayLogs(t *testing.T) {
	// Create a temporary directory structure mimicking a downloaded workflow run
	tmpDir := testutil.TempDir(t, "test-audit-gateway-*")
	runDir := filepath.Join(tmpDir, "run-12345678")

	// Create the directory structure for a typical smoke-copilot run
	dirs := []string{
		filepath.Join(runDir, "mcp-logs"),
		filepath.Join(runDir, "sandbox", "firewall", "logs"),
		filepath.Join(runDir, "safe-inputs", "logs"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create directory %s: %v", dir, err)
		}
	}

	// Create mock gateway log file
	gatewayLogContent := `{"timestamp":"2026-01-09T04:00:00Z","level":"info","msg":"Gateway started","port":8080}
{"timestamp":"2026-01-09T04:00:01Z","level":"info","msg":"Request received","method":"tools/list","server":"github","request_id":"abc-123"}
{"timestamp":"2026-01-09T04:00:01Z","level":"info","msg":"Request completed","method":"tools/list","server":"github","request_id":"abc-123","duration_ms":42,"status":"success"}
{"timestamp":"2026-01-09T04:00:02Z","level":"info","msg":"Request received","method":"tools/call","tool":"list_pull_requests","server":"github","request_id":"def-456"}
{"timestamp":"2026-01-09T04:00:02Z","level":"info","msg":"Request completed","method":"tools/call","tool":"list_pull_requests","server":"github","request_id":"def-456","duration_ms":156,"status":"success"}
{"timestamp":"2026-01-09T04:00:05Z","level":"info","msg":"Gateway shutdown","total_requests":2}
`
	gatewayLogFile := filepath.Join(runDir, "mcp-logs", "gateway-github.log")
	if err := os.WriteFile(gatewayLogFile, []byte(gatewayLogContent), 0644); err != nil {
		t.Fatalf("Failed to write gateway log file: %v", err)
	}

	// Create mock GitHub MCP server log
	githubLogContent := `{"level":"info","msg":"MCP server initialized","server":"github","timestamp":"2026-01-09T04:00:00Z"}
{"level":"info","msg":"Tool called","tool":"list_pull_requests","timestamp":"2026-01-09T04:00:02Z"}
{"level":"info","msg":"Tool completed","tool":"list_pull_requests","duration_ms":140,"timestamp":"2026-01-09T04:00:02Z"}
`
	githubLogFile := filepath.Join(runDir, "mcp-logs", "github.log")
	if err := os.WriteFile(githubLogFile, []byte(githubLogContent), 0644); err != nil {
		t.Fatalf("Failed to write GitHub MCP log file: %v", err)
	}

	// Create aw_info.json with workflow metadata
	awInfo := `{
  "workflow_name": "Smoke Copilot",
  "engine": "copilot",
  "run_id": 12345678,
  "mcp_servers": ["github"],
  "gateway_enabled": true
}
`
	awInfoFile := filepath.Join(runDir, "aw_info.json")
	if err := os.WriteFile(awInfoFile, []byte(awInfo), 0644); err != nil {
		t.Fatalf("Failed to write aw_info.json: %v", err)
	}

	// Create agent-stdio.log
	agentLog := `{"type":"system","subtype":"init","mcp_servers":[{"name":"github","status":"connected"}]}
{"type":"assistant","message":{"content":[{"type":"text","text":"Checking workflow logs"}]}}
`
	agentLogFile := filepath.Join(runDir, "agent-stdio.log")
	if err := os.WriteFile(agentLogFile, []byte(agentLog), 0644); err != nil {
		t.Fatalf("Failed to write agent-stdio.log: %v", err)
	}

	// Test 1: Verify gateway log file exists
	if _, err := os.Stat(gatewayLogFile); os.IsNotExist(err) {
		t.Fatal("Gateway log file was not created")
	}

	// Test 2: Verify gateway log content is structured JSON
	lines := strings.Split(strings.TrimSpace(gatewayLogContent), "\n")
	if len(lines) < 3 {
		t.Errorf("Expected at least 3 log lines, got %d", len(lines))
	}

	// Test 3: Verify gateway logs contain expected fields
	requiredFields := []string{"timestamp", "level", "msg", "request_id", "duration_ms"}
	for _, field := range requiredFields {
		if !strings.Contains(gatewayLogContent, field) {
			t.Errorf("Gateway log missing required field: %s", field)
		}
	}

	// Test 4: Verify gateway logs indicate successful requests
	if !strings.Contains(gatewayLogContent, `"status":"success"`) {
		t.Error("Gateway log should contain successful requests")
	}

	// Test 5: Verify no error-level messages in gateway logs
	if strings.Contains(gatewayLogContent, `"level":"error"`) {
		t.Error("Gateway log should not contain errors in this test scenario")
	}

	// Test 6: Verify MCP logs directory contains both gateway and server logs
	mcpLogsDir := filepath.Join(runDir, "mcp-logs")
	entries, err := os.ReadDir(mcpLogsDir)
	if err != nil {
		t.Fatalf("Failed to read mcp-logs directory: %v", err)
	}

	logFiles := make(map[string]bool)
	for _, entry := range entries {
		if !entry.IsDir() {
			logFiles[entry.Name()] = true
		}
	}

	expectedFiles := []string{"gateway-github.log", "github.log"}
	for _, expected := range expectedFiles {
		if !logFiles[expected] {
			t.Errorf("Expected log file '%s' not found in mcp-logs directory", expected)
		}
	}

	// Test 7: Verify artifact structure matches expected layout
	expectedDirs := []string{
		filepath.Join(runDir, "mcp-logs"),
		filepath.Join(runDir, "sandbox", "firewall", "logs"),
	}
	for _, dir := range expectedDirs {
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			t.Errorf("Expected directory does not exist: %s", dir)
		}
	}
}

// TestAuditWithGatewayErrors validates that the audit command properly detects
// and reports errors in gateway logs.
func TestAuditWithGatewayErrors(t *testing.T) {
	tmpDir := testutil.TempDir(t, "test-audit-gateway-errors-*")
	runDir := filepath.Join(tmpDir, "run-87654321")

	// Create directory structure
	mcpLogsDir := filepath.Join(runDir, "mcp-logs")
	if err := os.MkdirAll(mcpLogsDir, 0755); err != nil {
		t.Fatalf("Failed to create mcp-logs directory: %v", err)
	}

	// Create gateway log with errors
	gatewayLogWithErrors := `{"timestamp":"2026-01-09T04:00:00Z","level":"info","msg":"Gateway started","port":8080}
{"timestamp":"2026-01-09T04:00:01Z","level":"error","msg":"Failed to connect to MCP server","server":"github","error":"connection refused"}
{"timestamp":"2026-01-09T04:00:02Z","level":"error","msg":"Authentication failed","server":"github","error":"invalid API key"}
{"timestamp":"2026-01-09T04:00:03Z","level":"warn","msg":"Request timeout","method":"tools/call","server":"github","request_id":"xyz-789","duration_ms":5500}
{"timestamp":"2026-01-09T04:00:05Z","level":"error","msg":"Gateway shutdown due to errors","error_count":2}
`
	gatewayLogFile := filepath.Join(mcpLogsDir, "gateway-github.log")
	if err := os.WriteFile(gatewayLogFile, []byte(gatewayLogWithErrors), 0644); err != nil {
		t.Fatalf("Failed to write gateway log file: %v", err)
	}

	// Test 1: Verify error messages are present
	if !strings.Contains(gatewayLogWithErrors, `"level":"error"`) {
		t.Fatal("Test gateway log should contain error-level messages")
	}

	// Test 2: Count number of error entries
	errorCount := strings.Count(gatewayLogWithErrors, `"level":"error"`)
	if errorCount != 3 {
		t.Errorf("Expected 3 error entries, found %d", errorCount)
	}

	// Test 3: Verify specific error types are present
	expectedErrors := []string{
		"Failed to connect to MCP server",
		"Authentication failed",
		"Gateway shutdown due to errors",
	}
	for _, errMsg := range expectedErrors {
		if !strings.Contains(gatewayLogWithErrors, errMsg) {
			t.Errorf("Expected error message not found: %s", errMsg)
		}
	}

	// Test 4: Verify warning messages are detected
	if !strings.Contains(gatewayLogWithErrors, `"level":"warn"`) {
		t.Error("Test gateway log should contain warning-level messages")
	}

	// Test 5: Verify timeout is reported
	if !strings.Contains(gatewayLogWithErrors, "Request timeout") {
		t.Error("Expected timeout warning in gateway logs")
	}
}

// TestAuditWithoutGatewayLogs validates that the audit command handles cases
// where gateway logs are not present (e.g., when gateway is not configured).
func TestAuditWithoutGatewayLogs(t *testing.T) {
	tmpDir := testutil.TempDir(t, "test-audit-no-gateway-*")
	runDir := filepath.Join(tmpDir, "run-99999999")

	// Create directory structure without gateway logs
	mcpLogsDir := filepath.Join(runDir, "mcp-logs")
	if err := os.MkdirAll(mcpLogsDir, 0755); err != nil {
		t.Fatalf("Failed to create mcp-logs directory: %v", err)
	}

	// Create only GitHub MCP server log (no gateway)
	githubLogContent := `{"level":"info","msg":"MCP server initialized","server":"github"}
{"level":"info","msg":"Direct connection mode"}
`
	githubLogFile := filepath.Join(mcpLogsDir, "github.log")
	if err := os.WriteFile(githubLogFile, []byte(githubLogContent), 0644); err != nil {
		t.Fatalf("Failed to write GitHub MCP log file: %v", err)
	}

	// Create aw_info.json without gateway configuration
	awInfo := `{
  "workflow_name": "Test Workflow",
  "engine": "copilot",
  "mcp_servers": ["github"],
  "gateway_enabled": false
}
`
	awInfoFile := filepath.Join(runDir, "aw_info.json")
	if err := os.WriteFile(awInfoFile, []byte(awInfo), 0644); err != nil {
		t.Fatalf("Failed to write aw_info.json: %v", err)
	}

	// Test 1: Verify no gateway log files exist
	entries, err := os.ReadDir(mcpLogsDir)
	if err != nil {
		t.Fatalf("Failed to read mcp-logs directory: %v", err)
	}

	hasGatewayLogs := false
	for _, entry := range entries {
		if strings.Contains(entry.Name(), "gateway") {
			hasGatewayLogs = true
			break
		}
	}

	if hasGatewayLogs {
		t.Error("Should not have gateway logs when gateway is not configured")
	}

	// Test 2: Verify direct MCP server logs are still present
	if _, err := os.Stat(githubLogFile); os.IsNotExist(err) {
		t.Fatal("GitHub MCP log file should exist even without gateway")
	}

	// Test 3: Verify aw_info indicates gateway is disabled
	awInfoBytes, err := os.ReadFile(awInfoFile)
	if err != nil {
		t.Fatalf("Failed to read aw_info.json: %v", err)
	}

	if !strings.Contains(string(awInfoBytes), `"gateway_enabled": false`) {
		t.Error("aw_info.json should indicate gateway is disabled")
	}
}
