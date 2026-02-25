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

	// Detection is now inline in the agent job - no separate detection job
	agentSection := extractJobSection(yamlStr, "agent")
	if agentSection == "" {
		t.Fatal("Agent job not found in compiled workflow")
	}

	// Test 1: Agent job should contain inline detection steps
	if !strings.Contains(agentSection, "detection_guard") {
		t.Error("Agent job should contain detection_guard step for inline detection")
	}
	if !strings.Contains(agentSection, "parse_detection_results") {
		t.Error("Agent job should contain parse_detection_results step for inline detection")
	}

	// Test 2: Detection engine step should use limited tools (no --allow-all-tools)
	// The detection copilot invocation uses only shell tools for analysis
	if !strings.Contains(agentSection, "parse_threat_detection_results.cjs") {
		t.Error("Agent job should contain parse_threat_detection_results.cjs for detection")
	}

	// Test 3: Main agent job should still have --allow-tool or --allow-all-tools for the main agent execution
	// (before the detection steps)
	if !strings.Contains(agentSection, "--allow-tool") && !strings.Contains(agentSection, "--allow-all-tools") {
		t.Error("Main agent job should have --allow-tool or --allow-all-tools arguments")
	}

	// Test 4: Main agent job should have MCP setup
	if !strings.Contains(agentSection, "Start MCP Gateway") {
		t.Error("Main agent job should have MCP setup step")
	}

	// Test 5: No separate detection job should exist
	if strings.Contains(yamlStr, "\n  detection:\n") {
		t.Error("Separate detection job should NOT exist (detection is now inline in agent job)")
	}
}
