package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/githubnext/gh-aw/pkg/workflow"
)

func TestExtractRunID(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expected  int64
		shouldErr bool
	}{
		{
			name:      "Numeric run ID",
			input:     "1234567890",
			expected:  1234567890,
			shouldErr: false,
		},
		{
			name:      "Run URL",
			input:     "https://github.com/owner/repo/actions/runs/12345678",
			expected:  12345678,
			shouldErr: false,
		},
		{
			name:      "Job URL",
			input:     "https://github.com/owner/repo/actions/runs/12345678/job/98765432",
			expected:  12345678,
			shouldErr: false,
		},
		{
			name:      "Job URL with attempts",
			input:     "https://github.com/owner/repo/actions/runs/12345678/attempts/2",
			expected:  12345678,
			shouldErr: false,
		},
		{
			name:      "Run URL with trailing slash",
			input:     "https://github.com/owner/repo/actions/runs/12345678/",
			expected:  12345678,
			shouldErr: false,
		},
		{
			name:      "Invalid format",
			input:     "not-a-number",
			expected:  0,
			shouldErr: true,
		},
		{
			name:      "Invalid URL without run ID",
			input:     "https://github.com/owner/repo/actions",
			expected:  0,
			shouldErr: true,
		},
		{
			name:      "Empty string",
			input:     "",
			expected:  0,
			shouldErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := extractRunID(tt.input)

			if tt.shouldErr {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if result != tt.expected {
					t.Errorf("Expected run ID %d, got %d", tt.expected, result)
				}
			}
		})
	}
}

func TestIsPermissionError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "Nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "Authentication required error",
			err:      fmt.Errorf("authentication required"),
			expected: true,
		},
		{
			name:     "Exit status 4 error",
			err:      fmt.Errorf("exit status 4"),
			expected: true,
		},
		{
			name:     "GitHub CLI authentication error",
			err:      fmt.Errorf("GitHub CLI authentication required"),
			expected: true,
		},
		{
			name:     "Permission denied error",
			err:      fmt.Errorf("permission denied"),
			expected: true,
		},
		{
			name:     "GH_TOKEN error",
			err:      fmt.Errorf("GH_TOKEN environment variable not set"),
			expected: true,
		},
		{
			name:     "Other error",
			err:      fmt.Errorf("some other error"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isPermissionError(tt.err)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestGenerateAuditReport(t *testing.T) {
	// Create test data
	run := WorkflowRun{
		DatabaseID:    123456,
		WorkflowName:  "Test Workflow",
		Status:        "completed",
		Conclusion:    "success",
		CreatedAt:     time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
		StartedAt:     time.Date(2024, 1, 1, 10, 0, 30, 0, time.UTC),
		UpdatedAt:     time.Date(2024, 1, 1, 10, 5, 0, 0, time.UTC),
		Duration:      4*time.Minute + 30*time.Second,
		Event:         "push",
		HeadBranch:    "main",
		URL:           "https://github.com/org/repo/actions/runs/123456",
		TokenUsage:    1500,
		EstimatedCost: 0.025,
		Turns:         5,
		ErrorCount:    0,
		WarningCount:  1,
		LogsPath:      "/tmp/gh-aw/test-logs",
	}

	metrics := LogMetrics{
		TokenUsage:    1500,
		EstimatedCost: 0.025,
		Turns:         5,
		Errors: []workflow.LogError{
			{
				File:    "/tmp/gh-aw/logs/agent.log",
				Line:    42,
				Type:    "warning",
				Message: "Example warning message",
			},
		},
		ToolCalls: []workflow.ToolCallInfo{
			{
				Name:          "github_get_issue",
				CallCount:     3,
				MaxInputSize:  256,
				MaxOutputSize: 1024,
				MaxDuration:   2 * time.Second,
			},
			{
				Name:          "bash_echo",
				CallCount:     2,
				MaxInputSize:  128,
				MaxOutputSize: 512,
				MaxDuration:   1 * time.Second,
			},
		},
	}

	missingTools := []MissingToolReport{
		{
			Tool:         "missing_tool",
			Reason:       "Tool not available",
			Alternatives: "use alternative_tool instead",
			Timestamp:    "2024-01-01T10:00:00Z",
		},
	}

	mcpFailures := []MCPFailureReport{
		{
			ServerName: "test-server",
			Status:     "failed",
		},
	}

	processedRun := ProcessedRun{
		Run:          run,
		MissingTools: missingTools,
		MCPFailures:  mcpFailures,
	}

	// Generate report
	report := generateAuditReport(processedRun, metrics)

	// Verify report contains expected sections
	expectedSections := []string{
		"# Workflow Run Audit Report",
		"## Overview",
		"## Metrics",
		"## MCP Tool Usage",
		"## MCP Server Failures",
		"## Missing Tools",
		"## Available Artifacts",
	}

	for _, section := range expectedSections {
		if !strings.Contains(report, section) {
			t.Errorf("Report missing expected section: %s", section)
		}
	}

	// Verify report contains specific data
	expectedContent := []string{
		"123456",        // Run ID
		"Test Workflow", // Workflow name
		"success",       // Conclusion
		"main",          // Branch
		"0.025",         // Estimated cost
		"5",             // Turns
		"missing_tool",  // Missing tool
		"test-server",   // MCP failure
	}

	for _, content := range expectedContent {
		if !strings.Contains(report, content) {
			t.Errorf("Report missing expected content: %s", content)
		}
	}

	// Token usage should be present (formatted as 1.5k or similar)
	if !strings.Contains(report, "1.5k") && !strings.Contains(report, "1500") && !strings.Contains(report, "Token Usage") {
		t.Errorf("Report missing token usage (should be 1.5k or 1500)\nReport:\n%s", report)
	}
}

func TestGenerateAuditReportMinimal(t *testing.T) {
	// Test with minimal data (no errors, no tools, etc.)
	run := WorkflowRun{
		DatabaseID:   789,
		WorkflowName: "Minimal Workflow",
		Status:       "in_progress",
		CreatedAt:    time.Now(),
		Event:        "workflow_dispatch",
		HeadBranch:   "feature",
		URL:          "https://github.com/org/repo/actions/runs/789",
		LogsPath:     "/tmp/gh-aw/minimal-logs",
	}

	metrics := LogMetrics{}

	processedRun := ProcessedRun{
		Run: run,
	}

	// Generate report
	report := generateAuditReport(processedRun, metrics)

	// Verify report contains basic sections even with minimal data
	expectedSections := []string{
		"# Workflow Run Audit Report",
		"## Overview",
		"## Metrics",
		"## Available Artifacts",
	}

	for _, section := range expectedSections {
		if !strings.Contains(report, section) {
			t.Errorf("Minimal report missing expected section: %s", section)
		}
	}

	// Verify it doesn't contain sections that should be omitted when empty
	unexpectedSections := []string{
		"## MCP Server Failures",
		"## Missing Tools",
		"## Issue Summary",
	}

	for _, section := range unexpectedSections {
		if strings.Contains(report, section) {
			t.Errorf("Minimal report should not contain section: %s", section)
		}
	}
}

func TestGenerateAuditReportWithErrors(t *testing.T) {
	// Test with errors to verify issue summary
	run := WorkflowRun{
		DatabaseID:   999,
		WorkflowName: "Error Workflow",
		Status:       "completed",
		Conclusion:   "failure",
		CreatedAt:    time.Now(),
		Event:        "push",
		HeadBranch:   "main",
		URL:          "https://github.com/org/repo/actions/runs/999",
		ErrorCount:   3,
		WarningCount: 2,
		LogsPath:     "/tmp/gh-aw/error-logs",
	}

	metrics := LogMetrics{
		Errors: []workflow.LogError{
			{
				File:    "/tmp/gh-aw/error-logs/agent.log",
				Line:    10,
				Type:    "error",
				Message: "Failed to initialize tool",
			},
			{
				File:    "/tmp/gh-aw/error-logs/agent.log",
				Line:    15,
				Type:    "error",
				Message: "Connection timeout",
			},
			{
				File:    "/tmp/gh-aw/error-logs/agent.log",
				Line:    102,
				Type:    "error",
				Message: "Permission denied",
			},
			{
				File:    "/tmp/gh-aw/error-logs/agent.log",
				Line:    20,
				Type:    "warning",
				Message: "Deprecated API usage",
			},
			{
				File:    "/tmp/gh-aw/error-logs/agent.log",
				Line:    156,
				Type:    "warning",
				Message: "Resource limit approaching",
			},
		},
	}

	processedRun := ProcessedRun{
		Run: run,
	}

	// Generate report
	report := generateAuditReport(processedRun, metrics)

	// Verify issue summary is present
	if !strings.Contains(report, "## Issue Summary") {
		t.Error("Report should contain Issue Summary when errors are present")
	}

	// Verify error counts are mentioned
	if !strings.Contains(report, "3 error(s)") {
		t.Error("Report should mention error count")
	}
	if !strings.Contains(report, "2 warning(s)") {
		t.Error("Report should mention warning count")
	}

	// Verify individual errors are displayed
	if !strings.Contains(report, "### Errors and Warnings") {
		t.Error("Report should contain 'Errors and Warnings' section")
	}
	if !strings.Contains(report, "Failed to initialize tool") {
		t.Error("Report should contain first error message")
	}
	if !strings.Contains(report, "Connection timeout") {
		t.Error("Report should contain second error message")
	}
	if !strings.Contains(report, "Deprecated API usage") {
		t.Error("Report should contain warning message")
	}

	// Verify the format includes file and line information
	if !strings.Contains(report, "agent.log:10:") {
		t.Error("Report should contain file:line format for first error")
	}
}

func TestGenerateAuditReportArtifacts(t *testing.T) {
	// Create temporary directory with test artifacts
	tmpDir := t.TempDir()

	// Create test artifact files
	artifacts := []string{
		"aw_info.json",
		"safe_output.jsonl",
		"aw.patch",
		"agent_output.json",
	}

	for _, artifact := range artifacts {
		if err := os.WriteFile(filepath.Join(tmpDir, artifact), []byte("test"), 0644); err != nil {
			t.Fatalf("Failed to create test artifact %s: %v", artifact, err)
		}
	}

	run := WorkflowRun{
		DatabaseID:   555,
		WorkflowName: "Artifact Test",
		Status:       "completed",
		Conclusion:   "success",
		CreatedAt:    time.Now(),
		Event:        "push",
		HeadBranch:   "main",
		URL:          "https://github.com/org/repo/actions/runs/555",
		LogsPath:     tmpDir,
	}

	metrics := LogMetrics{}

	processedRun := ProcessedRun{
		Run: run,
	}

	// Generate report
	report := generateAuditReport(processedRun, metrics)

	// Verify all artifacts are listed
	expectedArtifacts := []string{
		"aw_info.json",
		"safe_output.jsonl",
		"aw.patch",
		"agent_output.json",
	}

	for _, artifact := range expectedArtifacts {
		if !strings.Contains(report, artifact) {
			t.Errorf("Report should list artifact: %s", artifact)
		}
	}
}

func TestBuildAuditData(t *testing.T) {
	// Create test data
	run := WorkflowRun{
		DatabaseID:    123456,
		WorkflowName:  "Test Workflow",
		Status:        "completed",
		Conclusion:    "success",
		CreatedAt:     time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
		StartedAt:     time.Date(2024, 1, 1, 10, 0, 30, 0, time.UTC),
		UpdatedAt:     time.Date(2024, 1, 1, 10, 5, 0, 0, time.UTC),
		Duration:      4*time.Minute + 30*time.Second,
		Event:         "push",
		HeadBranch:    "main",
		URL:           "https://github.com/org/repo/actions/runs/123456",
		TokenUsage:    1500,
		EstimatedCost: 0.025,
		Turns:         5,
		ErrorCount:    2,
		WarningCount:  1,
		LogsPath:      t.TempDir(),
	}

	metrics := LogMetrics{
		TokenUsage:    1500,
		EstimatedCost: 0.025,
		Turns:         5,
		Errors: []workflow.LogError{
			{
				File:    "/tmp/gh-aw/logs/agent.log",
				Line:    42,
				Type:    "warning",
				Message: "Example warning message",
			},
			{
				File:    "/tmp/gh-aw/logs/agent.log",
				Line:    50,
				Type:    "error",
				Message: "Example error message",
			},
			{
				File:    "/tmp/gh-aw/logs/agent.log",
				Line:    60,
				Type:    "error",
				Message: "Another error message",
			},
		},
		ToolCalls: []workflow.ToolCallInfo{
			{
				Name:          "github_get_issue",
				CallCount:     3,
				MaxInputSize:  256,
				MaxOutputSize: 1024,
				MaxDuration:   2 * time.Second,
			},
		},
	}

	missingTools := []MissingToolReport{
		{
			Tool:         "missing_tool",
			Reason:       "Tool not available",
			Alternatives: "use alternative_tool instead",
			Timestamp:    "2024-01-01T10:00:00Z",
		},
	}

	mcpFailures := []MCPFailureReport{
		{
			ServerName: "test-server",
			Status:     "failed",
		},
	}

	processedRun := ProcessedRun{
		Run:          run,
		MissingTools: missingTools,
		MCPFailures:  mcpFailures,
	}

	// Build audit data
	auditData := buildAuditData(processedRun, metrics)

	// Verify overview
	if auditData.Overview.RunID != 123456 {
		t.Errorf("Expected run ID 123456, got %d", auditData.Overview.RunID)
	}
	if auditData.Overview.WorkflowName != "Test Workflow" {
		t.Errorf("Expected workflow name 'Test Workflow', got %s", auditData.Overview.WorkflowName)
	}
	if auditData.Overview.Status != "completed" {
		t.Errorf("Expected status 'completed', got %s", auditData.Overview.Status)
	}

	// Verify metrics
	if auditData.Metrics.TokenUsage != 1500 {
		t.Errorf("Expected token usage 1500, got %d", auditData.Metrics.TokenUsage)
	}
	if auditData.Metrics.EstimatedCost != 0.025 {
		t.Errorf("Expected estimated cost 0.025, got %f", auditData.Metrics.EstimatedCost)
	}
	if auditData.Metrics.ErrorCount != 2 {
		t.Errorf("Expected error count 2, got %d", auditData.Metrics.ErrorCount)
	}
	if auditData.Metrics.WarningCount != 1 {
		t.Errorf("Expected warning count 1, got %d", auditData.Metrics.WarningCount)
	}

	// Verify errors and warnings are properly split
	if len(auditData.Errors) != 2 {
		t.Errorf("Expected 2 errors, got %d", len(auditData.Errors))
	}
	if len(auditData.Warnings) != 1 {
		t.Errorf("Expected 1 warning, got %d", len(auditData.Warnings))
	}

	// Verify tool usage
	if len(auditData.ToolUsage) != 1 {
		t.Errorf("Expected 1 tool usage entry, got %d", len(auditData.ToolUsage))
	}

	// Verify missing tools
	if len(auditData.MissingTools) != 1 {
		t.Errorf("Expected 1 missing tool, got %d", len(auditData.MissingTools))
	}

	// Verify MCP failures
	if len(auditData.MCPFailures) != 1 {
		t.Errorf("Expected 1 MCP failure, got %d", len(auditData.MCPFailures))
	}
}

func TestDescribeFile(t *testing.T) {
	tests := []struct {
		filename    string
		description string
	}{
		{"aw_info.json", "Engine configuration and workflow metadata"},
		{"safe_output.jsonl", "Safe outputs from workflow execution"},
		{"agent_output.json", "Validated safe outputs"},
		{"aw.patch", "Git patch of changes made during execution"},
		{"agent-stdio.log", "Agent standard output/error logs"},
		{"log.md", "Human-readable agent session summary"},
		{"random.log", "Log file"},
		{"unknown.txt", ""},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			result := describeFile(tt.filename)
			if result != tt.description {
				t.Errorf("Expected description '%s', got '%s'", tt.description, result)
			}
		})
	}
}

func TestRenderJSON(t *testing.T) {
	// Create test audit data
	auditData := AuditData{
		Overview: OverviewData{
			RunID:        123456,
			WorkflowName: "Test Workflow",
			Status:       "completed",
			Conclusion:   "success",
			Event:        "push",
			Branch:       "main",
			URL:          "https://github.com/org/repo/actions/runs/123456",
		},
		Metrics: MetricsData{
			TokenUsage:    1500,
			EstimatedCost: 0.025,
			Turns:         5,
			ErrorCount:    1,
			WarningCount:  1,
		},
		Jobs: []JobData{
			{
				Name:       "test-job",
				Status:     "completed",
				Conclusion: "success",
				Duration:   "2m30s",
			},
		},
		DownloadedFiles: []FileInfo{
			{
				Path:          "aw_info.json",
				Size:          1024,
				SizeFormatted: "1.0 KB",
				Description:   "Engine configuration and workflow metadata",
				IsDirectory:   false,
			},
		},
		MissingTools: []MissingToolReport{
			{
				Tool:   "missing_tool",
				Reason: "Tool not available",
			},
		},
		Errors: []ErrorInfo{
			{
				File:    "agent.log",
				Line:    42,
				Type:    "error",
				Message: "Test error",
			},
		},
		Warnings: []ErrorInfo{
			{
				File:    "agent.log",
				Line:    50,
				Type:    "warning",
				Message: "Test warning",
			},
		},
	}

	// Render to JSON
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := renderJSON(auditData)
	w.Close()

	// Read the output
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
		t.Fatalf("Failed to parse JSON output: %v", err)
	}

	// Verify key fields
	if parsed.Overview.RunID != 123456 {
		t.Errorf("Expected run ID 123456, got %d", parsed.Overview.RunID)
	}
	if parsed.Metrics.TokenUsage != 1500 {
		t.Errorf("Expected token usage 1500, got %d", parsed.Metrics.TokenUsage)
	}
	if len(parsed.Jobs) != 1 {
		t.Errorf("Expected 1 job, got %d", len(parsed.Jobs))
	}
	if len(parsed.DownloadedFiles) != 1 {
		t.Errorf("Expected 1 downloaded file, got %d", len(parsed.DownloadedFiles))
	}
	if len(parsed.MissingTools) != 1 {
		t.Errorf("Expected 1 missing tool, got %d", len(parsed.MissingTools))
	}
	if len(parsed.Errors) != 1 {
		t.Errorf("Expected 1 error, got %d", len(parsed.Errors))
	}
	if len(parsed.Warnings) != 1 {
		t.Errorf("Expected 1 warning, got %d", len(parsed.Warnings))
	}
}
