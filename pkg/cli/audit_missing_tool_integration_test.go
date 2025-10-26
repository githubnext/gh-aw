package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/githubnext/gh-aw/pkg/constants"
)

// TestAuditCommandWithMissingToolDetection tests the complete audit flow with missing tool detection
// This test validates the scenario from workflow run 18810355435 where the agent reported
// a missing tool for GitHub API access.
func TestAuditCommandWithMissingToolDetection(t *testing.T) {
	// Create a temporary directory structure mimicking workflow run 18810355435
	tmpDir := t.TempDir()
	runDir := filepath.Join(tmpDir, "run-18810355435")
	err := os.MkdirAll(runDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	// Create agent_output.json with the actual missing tool data from the workflow run
	agentOutputContent := `{
  "items": [
    {
      "type": "missing_tool",
      "tool": "GitHub API read access for workflows and artifacts",
      "reason": "Need to query GitHub API to list workflows, workflow runs, and artifacts data for generating the artifacts usage report",
      "alternatives": "Could use gh CLI with appropriate permissions or authenticated API token",
      "timestamp": "2025-10-26T00:19:46.916Z"
    }
  ],
  "errors": []
}`

	agentOutputPath := filepath.Join(runDir, constants.AgentOutputArtifactName)
	err = os.WriteFile(agentOutputPath, []byte(agentOutputContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write agent_output.json: %v", err)
	}

	// Create aw_info.json
	awInfoContent := `{
  "engine_id": "copilot",
  "workflow_name": "Artifacts Summary",
  "staged": false
}`
	awInfoPath := filepath.Join(runDir, "aw_info.json")
	err = os.WriteFile(awInfoPath, []byte(awInfoContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write aw_info.json: %v", err)
	}

	// Create test run with metadata matching the actual workflow run
	testRun := WorkflowRun{
		DatabaseID:   18810355435,
		WorkflowName: "Artifacts Summary",
		Status:       "completed",
		Conclusion:   "success",
		CreatedAt:    time.Date(2025, 10, 26, 0, 17, 1, 0, time.UTC),
		StartedAt:    time.Date(2025, 10, 26, 0, 17, 1, 0, time.UTC),
		UpdatedAt:    time.Date(2025, 10, 26, 0, 19, 48, 0, time.UTC),
		Event:        "workflow_dispatch",
		HeadBranch:   "copilot/fix-awf-compiler-arguments",
		URL:          "https://github.com/githubnext/gh-aw/actions/runs/18810355435",
		ErrorCount:   0,
		WarningCount: 0,
		LogsPath:     runDir,
	}

	// Extract missing tools using the same function the audit command uses
	missingTools, err := extractMissingToolsFromRun(runDir, testRun, false)
	if err != nil {
		t.Fatalf("Error extracting missing tools: %v", err)
	}

	// Verify that the missing tool was detected
	if len(missingTools) != 1 {
		t.Errorf("Expected 1 missing tool to be detected, got %d", len(missingTools))
		return
	}

	tool := missingTools[0]

	// Verify the tool name matches
	expectedTool := "GitHub API read access for workflows and artifacts"
	if tool.Tool != expectedTool {
		t.Errorf("Expected tool '%s', got '%s'", expectedTool, tool.Tool)
	}

	// Verify the reason is present and meaningful
	if !strings.Contains(tool.Reason, "GitHub API") {
		t.Errorf("Expected reason to mention 'GitHub API', got '%s'", tool.Reason)
	}

	// Verify alternatives are suggested
	if !strings.Contains(tool.Alternatives, "gh CLI") {
		t.Errorf("Expected alternatives to mention 'gh CLI', got '%s'", tool.Alternatives)
	}

	// Verify workflow metadata is populated
	if tool.WorkflowName != testRun.WorkflowName {
		t.Errorf("Expected workflow name '%s', got '%s'", testRun.WorkflowName, tool.WorkflowName)
	}

	if tool.RunID != testRun.DatabaseID {
		t.Errorf("Expected run ID %d, got %d", testRun.DatabaseID, tool.RunID)
	}

	// Build audit data to verify it includes the missing tool
	metrics := LogMetrics{}
	processedRun := ProcessedRun{
		Run:          testRun,
		MissingTools: missingTools,
	}

	auditData := buildAuditData(processedRun, metrics)

	// Verify audit data includes the missing tool
	if len(auditData.MissingTools) != 1 {
		t.Errorf("Expected audit data to include 1 missing tool, got %d", len(auditData.MissingTools))
	}

	// Generate the audit report to ensure it includes the missing tool section
	report := generateAuditReport(processedRun, metrics)

	// Verify the report contains the missing tools section
	if !strings.Contains(report, "## Missing Tools") {
		t.Error("Audit report should contain 'Missing Tools' section when tools are missing")
	}

	// Verify the specific tool is mentioned in the report
	if !strings.Contains(report, expectedTool) {
		t.Errorf("Audit report should mention the missing tool '%s'", expectedTool)
	}

	// Verify the reason is in the report
	if !strings.Contains(report, "query GitHub API") {
		t.Error("Audit report should include the reason for the missing tool")
	}

	// Test JSON output format as well
	jsonData, err := json.Marshal(auditData)
	if err != nil {
		t.Fatalf("Failed to marshal audit data to JSON: %v", err)
	}

	// Verify JSON includes the missing tool
	if !strings.Contains(string(jsonData), expectedTool) {
		t.Error("JSON output should include the missing tool")
	}
}

// TestAuditCommandWithNoMissingTools verifies that the audit command
// correctly handles runs that don't have any missing tools
func TestAuditCommandWithNoMissingTools(t *testing.T) {
	tmpDir := t.TempDir()
	runDir := filepath.Join(tmpDir, "run-12345")
	err := os.MkdirAll(runDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	// Create agent_output.json with no missing tools
	agentOutputContent := `{
  "items": [
    {
      "type": "create-discussion",
      "title": "Test Discussion",
      "body": "Test content"
    }
  ],
  "errors": []
}`

	agentOutputPath := filepath.Join(runDir, constants.AgentOutputArtifactName)
	err = os.WriteFile(agentOutputPath, []byte(agentOutputContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write agent_output.json: %v", err)
	}

	testRun := WorkflowRun{
		DatabaseID:   12345,
		WorkflowName: "Test Workflow",
		LogsPath:     runDir,
	}

	// Extract missing tools - should find none
	missingTools, err := extractMissingToolsFromRun(runDir, testRun, false)
	if err != nil {
		t.Fatalf("Error extracting missing tools: %v", err)
	}

	if len(missingTools) != 0 {
		t.Errorf("Expected 0 missing tools, got %d", len(missingTools))
	}

	// Generate report and verify it doesn't include missing tools section
	metrics := LogMetrics{}
	processedRun := ProcessedRun{
		Run:          testRun,
		MissingTools: missingTools,
	}

	report := generateAuditReport(processedRun, metrics)

	// When there are no missing tools, the section should not appear
	if strings.Contains(report, "## Missing Tools") {
		t.Error("Audit report should not contain 'Missing Tools' section when no tools are missing")
	}
}

// TestAuditCommandWithMultipleMissingTools tests the audit command with multiple missing tools
func TestAuditCommandWithMultipleMissingTools(t *testing.T) {
	tmpDir := t.TempDir()
	runDir := filepath.Join(tmpDir, "run-67890")
	err := os.MkdirAll(runDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	// Create agent_output.json with multiple missing tools
	agentOutputContent := `{
  "items": [
    {
      "type": "missing_tool",
      "tool": "Docker CLI",
      "reason": "Need to build and push container images",
      "alternatives": "Use GitHub Actions Docker buildx action"
    },
    {
      "type": "missing_tool",
      "tool": "Kubernetes kubectl",
      "reason": "Need to deploy to Kubernetes cluster",
      "alternatives": "Use kubectl GitHub Action"
    },
    {
      "type": "create-issue",
      "title": "Other output",
      "body": "Not a missing tool"
    }
  ],
  "errors": []
}`

	agentOutputPath := filepath.Join(runDir, constants.AgentOutputArtifactName)
	err = os.WriteFile(agentOutputPath, []byte(agentOutputContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write agent_output.json: %v", err)
	}

	testRun := WorkflowRun{
		DatabaseID:   67890,
		WorkflowName: "Deployment Workflow",
		LogsPath:     runDir,
	}

	// Extract missing tools
	missingTools, err := extractMissingToolsFromRun(runDir, testRun, false)
	if err != nil {
		t.Fatalf("Error extracting missing tools: %v", err)
	}

	// Verify both missing tools were detected
	if len(missingTools) != 2 {
		t.Errorf("Expected 2 missing tools, got %d", len(missingTools))
	}

	// Verify first tool
	if len(missingTools) >= 1 {
		if missingTools[0].Tool != "Docker CLI" {
			t.Errorf("Expected first tool 'Docker CLI', got '%s'", missingTools[0].Tool)
		}
	}

	// Verify second tool
	if len(missingTools) >= 2 {
		if missingTools[1].Tool != "Kubernetes kubectl" {
			t.Errorf("Expected second tool 'Kubernetes kubectl', got '%s'", missingTools[1].Tool)
		}
	}

	// Generate report and verify both tools are mentioned
	metrics := LogMetrics{}
	processedRun := ProcessedRun{
		Run:          testRun,
		MissingTools: missingTools,
	}

	report := generateAuditReport(processedRun, metrics)

	if !strings.Contains(report, "Docker CLI") {
		t.Error("Audit report should mention 'Docker CLI'")
	}

	if !strings.Contains(report, "Kubernetes kubectl") {
		t.Error("Audit report should mention 'Kubernetes kubectl'")
	}
}
