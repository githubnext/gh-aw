//go:build !integration

package workflow

import (
	"strings"
	"testing"
)

func TestCopilotSessionFileCopyStep(t *testing.T) {
	engine := NewCopilotEngine()
	workflowData := &WorkflowData{
		Name: "test-workflow",
	}

	// Get the firewall logs collection step (which now includes session file copy)
	steps := engine.GetFirewallLogsCollectionStep(workflowData)

	// Should have at least one step (session file copy)
	if len(steps) == 0 {
		t.Fatal("Expected at least one step for session file copy")
	}

	// Check that the step contains session file copy logic
	stepContent := strings.Join([]string(steps[0]), "\n")

	// Verify step name
	if !strings.Contains(stepContent, "Copy Copilot session state files to logs") {
		t.Error("Expected step name to contain 'Copy Copilot session state files to logs'")
	}

	// Verify if: always() condition
	if !strings.Contains(stepContent, "if: always()") {
		t.Error("Expected step to have 'if: always()' condition")
	}

	// Verify continue-on-error
	if !strings.Contains(stepContent, "continue-on-error: true") {
		t.Error("Expected step to have 'continue-on-error: true'")
	}

	// Verify source directory reference
	if !strings.Contains(stepContent, "$HOME/.copilot/session-state") {
		t.Error("Expected step to reference '$HOME/.copilot/session-state' directory")
	}

	// Verify target directory reference
	if !strings.Contains(stepContent, "/tmp/gh-aw/sandbox/agent/logs") {
		t.Error("Expected step to reference '/tmp/gh-aw/sandbox/agent/logs' directory")
	}

	// Verify recursive find for .jsonl files in session subdirectories
	if !strings.Contains(stepContent, "find") {
		t.Error("Expected step to use 'find' for recursive .jsonl file discovery")
	}
	if !strings.Contains(stepContent, "*.jsonl") {
		t.Error("Expected step to search for *.jsonl files")
	}

	// Verify session ID is preserved in the output filename
	if !strings.Contains(stepContent, "session_id=$(basename") {
		t.Error("Expected step to extract session_id from parent directory")
	}
	if !strings.Contains(stepContent, "${filename}-${session_id}.jsonl") {
		t.Error("Expected step to include session_id in output filename to avoid collisions")
	}

	// Verify cp command
	if !strings.Contains(stepContent, "cp -v") {
		t.Error("Expected step to use 'cp -v' command")
	}
}
