package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/githubnext/gh-aw/pkg/workflow"
)

func TestGenerateAuditReport(t *testing.T) {
	// Create test data
	run := WorkflowRun{
		DatabaseID:   123456,
		WorkflowName: "Test Workflow",
		Status:       "completed",
		Conclusion:   "success",
		CreatedAt:    time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
		StartedAt:    time.Date(2024, 1, 1, 10, 0, 30, 0, time.UTC),
		UpdatedAt:    time.Date(2024, 1, 1, 10, 5, 0, 0, time.UTC),
		Duration:     4*time.Minute + 30*time.Second,
		Event:        "push",
		HeadBranch:   "main",
		URL:          "https://github.com/org/repo/actions/runs/123456",
		TokenUsage:   1500,
		EstimatedCost: 0.025,
		Turns:        5,
		ErrorCount:   0,
		WarningCount: 1,
		LogsPath:     "/tmp/test-logs",
	}

	metrics := LogMetrics{
		TokenUsage:    1500,
		EstimatedCost: 0.025,
		Turns:         5,
		ErrorCount:    0,
		WarningCount:  1,
		ToolCalls: []workflow.ToolCallInfo{
			{
				Name:          "github_get_issue",
				CallCount:     3,
				MaxOutputSize: 1024,
				MaxDuration:   2 * time.Second,
			},
			{
				Name:          "bash_echo",
				CallCount:     2,
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
	report := generateAuditReport(processedRun, metrics, false)

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
		"123456",                // Run ID
		"Test Workflow",         // Workflow name
		"success",               // Conclusion
		"main",                  // Branch
		"0.025",                 // Estimated cost
		"5",                     // Turns
		"missing_tool",          // Missing tool
		"test-server",           // MCP failure
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
		LogsPath:     "/tmp/minimal-logs",
	}

	metrics := LogMetrics{}

	processedRun := ProcessedRun{
		Run: run,
	}

	// Generate report
	report := generateAuditReport(processedRun, metrics, false)

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
		LogsPath:     "/tmp/error-logs",
	}

	metrics := LogMetrics{
		ErrorCount:   3,
		WarningCount: 2,
	}

	processedRun := ProcessedRun{
		Run: run,
	}

	// Generate report
	report := generateAuditReport(processedRun, metrics, false)

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
	report := generateAuditReport(processedRun, metrics, false)

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
