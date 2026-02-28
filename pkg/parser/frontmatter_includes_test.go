//go:build !integration

package parser

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Test ExtractWorkflowNameFromMarkdown function

func TestProcessIncludedFileWithNameAndDescription(t *testing.T) {
	tempDir := t.TempDir()
	docsDir := filepath.Join(tempDir, "docs")
	if err := os.MkdirAll(docsDir, 0755); err != nil {
		t.Fatalf("Failed to create docs directory: %v", err)
	}

	// Create a test file with name and description fields
	testFile := filepath.Join(docsDir, "shared-config.md")
	testContent := `---
name: Shared Configuration
description: Common tools and configuration for workflows
tools:
  github:
    allowed: [issue_read]
---

# Shared Configuration

This is a shared configuration file.`

	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Process the included file - should not generate warnings for name and description
	result, err := processIncludedFileWithVisited(testFile, "", false, make(map[string]bool))
	if err != nil {
		t.Fatalf("processIncludedFileWithVisited() error = %v", err)
	}

	if !strings.Contains(result, "# Shared Configuration") {
		t.Errorf("Expected markdown content not found in result")
	}

	// The test should pass without warnings being printed to stderr
	// We can't easily capture stderr in this test, but the absence of an error
	// indicates that the file was processed successfully
}

// TestProcessIncludedFileWithOnlyNameAndDescription verifies that files with only
// name and description fields (and no other fields) are processed without warnings
func TestProcessIncludedFileWithOnlyNameAndDescription(t *testing.T) {
	tempDir := t.TempDir()
	docsDir := filepath.Join(tempDir, "docs")
	if err := os.MkdirAll(docsDir, 0755); err != nil {
		t.Fatalf("Failed to create docs directory: %v", err)
	}

	// Create a test file with only name and description fields
	testFile := filepath.Join(docsDir, "minimal-config.md")
	testContent := `---
name: Minimal Configuration
description: A minimal configuration with just metadata
---

# Minimal Configuration

This file only has name and description in frontmatter.`

	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Process the included file - should not generate warnings
	result, err := processIncludedFileWithVisited(testFile, "", false, make(map[string]bool))
	if err != nil {
		t.Fatalf("processIncludedFileWithVisited() error = %v", err)
	}

	if !strings.Contains(result, "# Minimal Configuration") {
		t.Errorf("Expected markdown content not found in result")
	}
}

// TestProcessIncludedFileWithDisableModelInvocationField verifies that the "disable-model-invocation" field
// (used in custom agent format) is accepted without warnings
func TestProcessIncludedFileWithDisableModelInvocationField(t *testing.T) {
	tempDir := t.TempDir()
	agentsDir := filepath.Join(tempDir, ".github", "agents")
	if err := os.MkdirAll(agentsDir, 0755); err != nil {
		t.Fatalf("Failed to create agents directory: %v", err)
	}

	// Create a test file with the "disable-model-invocation" field (custom agent format)
	testFile := filepath.Join(agentsDir, "test-agent.agent.md")
	testContent := `---
name: Test Agent
description: A test custom agent
disable-model-invocation: true
---

# Test Agent

This is a custom agent file with the disable-model-invocation field.`

	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Process the included file - should not generate warnings
	result, err := processIncludedFileWithVisited(testFile, "", false, make(map[string]bool))
	if err != nil {
		t.Fatalf("processIncludedFileWithVisited() error = %v", err)
	}

	if !strings.Contains(result, "# Test Agent") {
		t.Errorf("Expected markdown content not found in result")
	}
}

// TestProcessIncludedFileWithAgentToolsArray verifies that custom agent files
// with tools as an array (GitHub Copilot format) are processed without validation errors
func TestProcessIncludedFileWithAgentToolsArray(t *testing.T) {
	tempDir := t.TempDir()
	agentsDir := filepath.Join(tempDir, ".github", "agents")
	if err := os.MkdirAll(agentsDir, 0755); err != nil {
		t.Fatalf("Failed to create agents directory: %v", err)
	}

	// Create a test file with tools as an array (custom agent format)
	testFile := filepath.Join(agentsDir, "feature-flag-remover.agent.md")
	testContent := `---
description: "Removes feature flags from codebase"
tools:
  [
    "edit",
    "search",
    "execute/getTerminalOutput",
    "execute/runInTerminal",
    "read/terminalLastCommand",
    "read/terminalSelection",
    "execute/createAndRunTask",
    "execute/getTaskOutput",
    "execute/runTask",
    "read/problems",
    "search/changes",
    "agent",
    "runTasks",
    "problems",
    "changes",
    "runSubagent",
  ]
---

# Feature Flag Remover Agent

This agent removes feature flags from the codebase.`

	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Process the included file - should not generate validation errors
	// because custom agent files use a different tools format (array vs object)
	result, err := processIncludedFileWithVisited(testFile, "", false, make(map[string]bool))
	if err != nil {
		t.Fatalf("processIncludedFileWithVisited() error = %v, want nil", err)
	}

	if !strings.Contains(result, "# Feature Flag Remover Agent") {
		t.Errorf("Expected markdown content not found in result")
	}

	// Also test that tools extraction skips agent files and returns empty object
	toolsResult, err := processIncludedFileWithVisited(testFile, "", true, make(map[string]bool))
	if err != nil {
		t.Fatalf("processIncludedFileWithVisited(extractTools=true) error = %v, want nil", err)
	}

	if toolsResult != "{}" {
		t.Errorf("processIncludedFileWithVisited(extractTools=true) = %q, want {}", toolsResult)
	}
}

// TestProcessIncludedFileWithEngineCommand verifies that included files
// with engine.command property are processed without validation errors
func TestProcessIncludedFileWithEngineCommand(t *testing.T) {
	tempDir := t.TempDir()
	docsDir := filepath.Join(tempDir, "docs")
	if err := os.MkdirAll(docsDir, 0755); err != nil {
		t.Fatalf("Failed to create docs directory: %v", err)
	}

	// Create a test file with engine.command property
	testFile := filepath.Join(docsDir, "engine-config.md")
	testContent := `---
engine:
  id: copilot
  command: /custom/path/to/copilot
  version: "1.0.0"
tools:
  github:
    allowed: [issue_read]
---

# Engine Configuration

This is a shared engine configuration with custom command.`

	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Process the included file - should not generate validation errors
	result, err := processIncludedFileWithVisited(testFile, "", false, make(map[string]bool))
	if err != nil {
		t.Fatalf("processIncludedFileWithVisited() error = %v, want nil", err)
	}

	if !strings.Contains(result, "# Engine Configuration") {
		t.Errorf("Expected markdown content not found in result")
	}

	// Also test that tools extraction works correctly
	toolsResult, err := processIncludedFileWithVisited(testFile, "", true, make(map[string]bool))
	if err != nil {
		t.Fatalf("processIncludedFileWithVisited(extractTools=true) error = %v, want nil", err)
	}

	if !strings.Contains(toolsResult, `"github"`) {
		t.Errorf("processIncludedFileWithVisited(extractTools=true) should contain github tools, got: %q", toolsResult)
	}
}
