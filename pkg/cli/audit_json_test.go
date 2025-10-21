package cli

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/githubnext/gh-aw/pkg/workflow"
)

// TestAuditCommandJSONFlag verifies that the audit command's --json flag
// produces valid JSON output that can be parsed back into the AuditData structure
func TestAuditCommandJSONFlag(t *testing.T) {
	// Create a temporary directory with mock audit data
	tempDir := t.TempDir()
	runDir := filepath.Join(tempDir, "run-18668879630")
	if err := os.MkdirAll(runDir, 0755); err != nil {
		t.Fatalf("Failed to create run directory: %v", err)
	}

	// Create mock aw_info.json
	awInfoPath := filepath.Join(runDir, "aw_info.json")
	awInfoContent := `{"engine_id": "copilot", "workflow_name": "test-workflow", "run_id": 18668879630}`
	if err := os.WriteFile(awInfoPath, []byte(awInfoContent), 0644); err != nil {
		t.Fatalf("Failed to create mock aw_info.json: %v", err)
	}

	// Create mock safe_output.jsonl
	safeOutputPath := filepath.Join(runDir, "safe_output.jsonl")
	safeOutputContent := `{"type":"comment","data":"test output"}`
	if err := os.WriteFile(safeOutputPath, []byte(safeOutputContent), 0644); err != nil {
		t.Fatalf("Failed to create mock safe_output.jsonl: %v", err)
	}

	// Create a ProcessedRun with test data
	run := WorkflowRun{
		DatabaseID:    18668879630,
		WorkflowName:  "Test Workflow",
		Status:        "completed",
		Conclusion:    "success",
		CreatedAt:     time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
		StartedAt:     time.Date(2024, 1, 1, 10, 0, 30, 0, time.UTC),
		UpdatedAt:     time.Date(2024, 1, 1, 10, 5, 0, 0, time.UTC),
		Duration:      4*time.Minute + 30*time.Second,
		Event:         "push",
		HeadBranch:    "main",
		URL:           "https://github.com/githubnext/gh-aw/actions/runs/18668879630",
		TokenUsage:    2500,
		EstimatedCost: 0.035,
		Turns:         8,
		ErrorCount:    1,
		WarningCount:  2,
		LogsPath:      runDir,
	}

	metrics := LogMetrics{
		TokenUsage:    2500,
		EstimatedCost: 0.035,
		Turns:         8,
		Errors:        []workflow.LogError{},
		ToolCalls:     []workflow.ToolCallInfo{},
	}

	processedRun := ProcessedRun{
		Run:          run,
		MissingTools: []MissingToolReport{},
		MCPFailures:  []MCPFailureReport{},
	}

	// Build audit data
	auditData := buildAuditData(processedRun, metrics)

	// Test JSON output
	t.Run("JSON output is valid and parseable", func(t *testing.T) {
		// Capture stdout
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		err := renderJSON(auditData)
		w.Close()

		// Read captured output
		var buf strings.Builder
		io.Copy(&buf, r)
		os.Stdout = oldStdout

		if err != nil {
			t.Fatalf("renderJSON failed: %v", err)
		}

		jsonOutput := buf.String()

		// Verify it's valid JSON
		var parsed AuditData
		if err := json.Unmarshal([]byte(jsonOutput), &parsed); err != nil {
			t.Fatalf("Failed to parse JSON output: %v\nOutput: %s", err, jsonOutput)
		}

		// Verify key fields match the original data
		if parsed.Overview.RunID != 18668879630 {
			t.Errorf("Expected run ID 18668879630, got %d", parsed.Overview.RunID)
		}
		if parsed.Overview.WorkflowName != "Test Workflow" {
			t.Errorf("Expected workflow name 'Test Workflow', got '%s'", parsed.Overview.WorkflowName)
		}
		if parsed.Overview.Status != "completed" {
			t.Errorf("Expected status 'completed', got '%s'", parsed.Overview.Status)
		}
		if parsed.Overview.Conclusion != "success" {
			t.Errorf("Expected conclusion 'success', got '%s'", parsed.Overview.Conclusion)
		}
		if parsed.Metrics.TokenUsage != 2500 {
			t.Errorf("Expected token usage 2500, got %d", parsed.Metrics.TokenUsage)
		}
		if parsed.Metrics.EstimatedCost != 0.035 {
			t.Errorf("Expected estimated cost 0.035, got %f", parsed.Metrics.EstimatedCost)
		}
		if parsed.Metrics.ErrorCount != 1 {
			t.Errorf("Expected error count 1, got %d", parsed.Metrics.ErrorCount)
		}
		if parsed.Metrics.WarningCount != 2 {
			t.Errorf("Expected warning count 2, got %d", parsed.Metrics.WarningCount)
		}
	})

	// Test console output
	t.Run("Console output is formatted correctly", func(t *testing.T) {
		// Capture stdout
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		renderConsole(auditData, runDir)
		w.Close()

		// Read captured output
		var buf strings.Builder
		io.Copy(&buf, r)
		os.Stdout = oldStdout

		output := buf.String()

		// Verify key sections are present
		expectedSections := []string{
			"# Workflow Run Audit Report",
			"## Overview",
			"## Metrics",
			"## Downloaded Files",
			"## Logs Location",
		}

		for _, section := range expectedSections {
			if !strings.Contains(output, section) {
				t.Errorf("Console output missing expected section: %s", section)
			}
		}

		// Verify key data is present
		expectedContent := []string{
			"18668879630",   // Run ID
			"Test Workflow", // Workflow name
			"completed",     // Status
			"success",       // Conclusion
			"main",          // Branch
			"2.5k",          // Token usage (formatted)
			"$0.035",        // Cost (formatted)
		}

		for _, content := range expectedContent {
			if !strings.Contains(output, content) {
				t.Errorf("Console output missing expected content: %s", content)
			}
		}

		// Verify it's NOT JSON
		var parsed AuditData
		if err := json.Unmarshal([]byte(output), &parsed); err == nil {
			t.Error("Console output should not be valid JSON")
		}
	})

	// Test that both outputs contain the same data
	t.Run("JSON and console outputs contain equivalent data", func(t *testing.T) {
		// Get JSON output
		oldStdout := os.Stdout
		r1, w1, _ := os.Pipe()
		os.Stdout = w1
		renderJSON(auditData)
		w1.Close()
		var jsonBuf strings.Builder
		io.Copy(&jsonBuf, r1)
		os.Stdout = oldStdout

		var jsonData AuditData
		json.Unmarshal([]byte(jsonBuf.String()), &jsonData)

		// Get console output (we can't parse it directly, but we verify the data matches the source)
		r2, w2, _ := os.Pipe()
		os.Stdout = w2
		renderConsole(auditData, runDir)
		w2.Close()
		var consoleBuf strings.Builder
		io.Copy(&consoleBuf, r2)
		os.Stdout = oldStdout

		// Verify JSON data matches the original auditData
		if jsonData.Overview.RunID != auditData.Overview.RunID {
			t.Error("JSON output has different run ID than original data")
		}
		if jsonData.Metrics.TokenUsage != auditData.Metrics.TokenUsage {
			t.Error("JSON output has different token usage than original data")
		}

		// Verify console output contains the same numeric values
		consoleOutput := consoleBuf.String()
		if !strings.Contains(consoleOutput, "18668879630") {
			t.Error("Console output doesn't contain run ID")
		}
		// Token usage should be formatted as "2.5k" in console
		if !strings.Contains(consoleOutput, "2.5k") {
			t.Error("Console output doesn't contain formatted token usage")
		}
	})
}

// TestAuditCommandJSONFlagRealWorld tests the JSON flag with a more realistic scenario
func TestAuditCommandJSONFlagRealWorld(t *testing.T) {
	// Create a temporary directory with mock audit data that simulates a real workflow run
	tempDir := t.TempDir()
	runDir := filepath.Join(tempDir, "run-123456789")
	if err := os.MkdirAll(runDir, 0755); err != nil {
		t.Fatalf("Failed to create run directory: %v", err)
	}

	// Create comprehensive mock artifacts
	artifacts := map[string]string{
		"aw_info.json":      `{"engine_id": "claude", "workflow_name": "ci-test", "run_id": 123456789}`,
		"safe_output.jsonl": `{"type":"issue","title":"Test Issue","body":"Test body"}`,
		"aw.patch":          "diff --git a/test.txt b/test.txt\n--- a/test.txt\n+++ b/test.txt\n@@ -1 +1 @@\n-old\n+new",
	}

	for filename, content := range artifacts {
		path := filepath.Join(runDir, filename)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create %s: %v", filename, err)
		}
	}

	// Create test data with errors, warnings, and tool usage
	run := WorkflowRun{
		DatabaseID:    123456789,
		WorkflowName:  "CI Test Workflow",
		Status:        "completed",
		Conclusion:    "failure",
		CreatedAt:     time.Now().Add(-1 * time.Hour),
		StartedAt:     time.Now().Add(-55 * time.Minute),
		UpdatedAt:     time.Now().Add(-5 * time.Minute),
		Duration:      50 * time.Minute,
		Event:         "pull_request",
		HeadBranch:    "feature/test",
		URL:           "https://github.com/org/repo/actions/runs/123456789",
		TokenUsage:    15000,
		EstimatedCost: 0.125,
		Turns:         15,
		ErrorCount:    3,
		WarningCount:  5,
		LogsPath:      runDir,
	}

	metrics := LogMetrics{
		TokenUsage:    15000,
		EstimatedCost: 0.125,
		Turns:         15,
		Errors: []workflow.LogError{
			{File: "agent.log", Line: 42, Type: "error", Message: "Failed to execute command"},
			{File: "agent.log", Line: 100, Type: "error", Message: "Network timeout"},
			{File: "agent.log", Line: 150, Type: "error", Message: "Permission denied"},
			{File: "agent.log", Line: 80, Type: "warning", Message: "Deprecated API"},
			{File: "agent.log", Line: 120, Type: "warning", Message: "Slow response time"},
		},
		ToolCalls: []workflow.ToolCallInfo{
			{Name: "github_get_issue", CallCount: 10, MaxInputSize: 1024, MaxOutputSize: 4096, MaxDuration: 2 * time.Second},
			{Name: "bash_run", CallCount: 25, MaxInputSize: 512, MaxOutputSize: 2048, MaxDuration: 5 * time.Second},
		},
	}

	processedRun := ProcessedRun{
		Run: run,
		MissingTools: []MissingToolReport{
			{Tool: "docker", Reason: "Not available in environment", Alternatives: "Use podman instead"},
		},
		MCPFailures: []MCPFailureReport{
			{ServerName: "custom-mcp-server", Status: "connection refused"},
		},
	}

	auditData := buildAuditData(processedRun, metrics)

	// Verify JSON output includes all sections
	t.Run("JSON output includes all data sections", func(t *testing.T) {
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		err := renderJSON(auditData)
		w.Close()

		var buf strings.Builder
		io.Copy(&buf, r)
		os.Stdout = oldStdout

		if err != nil {
			t.Fatalf("renderJSON failed: %v", err)
		}

		var parsed AuditData
		if err := json.Unmarshal([]byte(buf.String()), &parsed); err != nil {
			t.Fatalf("Failed to parse JSON: %v", err)
		}

		// Verify all sections have data
		if len(parsed.DownloadedFiles) == 0 {
			t.Error("Expected downloaded files in JSON output")
		}
		if len(parsed.MissingTools) == 0 {
			t.Error("Expected missing tools in JSON output")
		}
		if len(parsed.MCPFailures) == 0 {
			t.Error("Expected MCP failures in JSON output")
		}
		if len(parsed.Errors) == 0 {
			t.Error("Expected errors in JSON output")
		}
		if len(parsed.Warnings) == 0 {
			t.Error("Expected warnings in JSON output")
		}
		if len(parsed.ToolUsage) == 0 {
			t.Error("Expected tool usage in JSON output")
		}
	})
}
