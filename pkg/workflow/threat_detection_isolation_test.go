//go:build !integration

package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/stringutil"

	"github.com/github/gh-aw/pkg/testutil"
)

func TestThreatDetectionIsolation(t *testing.T) {
	compiler := NewCompiler()

	// Create a temporary directory for the test workflow
	tmpDir := testutil.TempDir(t, "test-*")
	workflowPath := filepath.Join(tmpDir, "test-isolation.md")

	workflowContent := `---
on: push
safe-outputs:
  create-issue:
tools:
  github:
    allowed: ["*"]
---
Test workflow`

	// Write the workflow file
	if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	// Compile the workflow
	if err := compiler.CompileWorkflow(workflowPath); err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	// Read the compiled output
	lockFile := stringutil.MarkdownToLockFile(workflowPath)
	result, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read compiled workflow: %v", err)
	}

	yamlStr := string(result)

	// Detection is now a separate detection job - agent job should NOT contain inline detection steps
	agentSection := extractJobSection(yamlStr, "agent")
	if agentSection == "" {
		t.Fatal("Agent job not found in compiled workflow")
	}

	// Test 1: Detection job should exist as a separate job
	detectionSection := extractJobSection(yamlStr, "detection")
	if detectionSection == "" {
		t.Error("Detection job should exist as a separate job")
	}
	if !strings.Contains(detectionSection, "detection_guard") {
		t.Error("Detection job should contain detection_guard step")
	}
	if !strings.Contains(detectionSection, "detection_conclusion") {
		t.Error("Detection job should contain detection_conclusion step")
	}

	// Test 2: Detection engine step should use limited tools (no --allow-all-tools)
	// The detection copilot invocation uses only shell tools for analysis
	if !strings.Contains(detectionSection, "parse_threat_detection_results.cjs") {
		t.Error("Detection job should contain parse_threat_detection_results.cjs for detection")
	}

	// Test 3: Main agent job should still have --allow-tool or --allow-all-tools for the main agent execution
	if !strings.Contains(agentSection, "--allow-tool") && !strings.Contains(agentSection, "--allow-all-tools") {
		t.Error("Main agent job should have --allow-tool or --allow-all-tools arguments")
	}

	// Test 4: Main agent job should have MCP setup
	if !strings.Contains(agentSection, "Start MCP Gateway") {
		t.Error("Main agent job should have MCP setup step")
	}

	// Test 5: A separate detection job should exist
	if !strings.Contains(yamlStr, "  detection:") {
		t.Error("Separate detection job should exist")
	}
}
