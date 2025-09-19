package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMCPLogsUpload(t *testing.T) {
	// Create a temporary directory for the test
	tmpDir := t.TempDir()

	// Create a test markdown file with Playwright tool configuration
	testMarkdown := `---
on:
  workflow_dispatch:
tools:
  playwright:
    allowed_domains: ["example.com"]
engine: claude
---

# Test MCP Logs Upload

This is a test workflow to validate MCP logs upload generation.

Please navigate to example.com and take a screenshot.
`

	// Write the test file
	mdFile := filepath.Join(tmpDir, "test-mcp-logs.md")
	if err := os.WriteFile(mdFile, []byte(testMarkdown), 0644); err != nil {
		t.Fatalf("Failed to write test markdown file: %v", err)
	}

	// Initialize compiler
	compiler := NewCompiler(false, "", "test-version")

	// Compile the workflow
	err := compiler.CompileWorkflow(mdFile)
	if err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	// Read the generated lock file
	lockFile := filepath.Join(tmpDir, "test-mcp-logs.lock.yml")
	lockContent, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}

	lockContentStr := string(lockContent)

	// Verify Playwright MCP configuration includes output-dir
	if !strings.Contains(lockContentStr, "\"--output-dir\"") {
		t.Error("Expected Playwright MCP configuration to include '--output-dir' argument")
	}

	if !strings.Contains(lockContentStr, "\"/tmp/mcp-logs/playwright\"") {
		t.Error("Expected Playwright MCP configuration to include '/tmp/mcp-logs/playwright' path")
	}

	// Verify MCP logs upload step exists
	if !strings.Contains(lockContentStr, "- name: Upload MCP logs") {
		t.Error("Expected 'Upload MCP logs' step to be in generated workflow")
	}

	// Verify the upload step uses actions/upload-artifact@v4
	if !strings.Contains(lockContentStr, "uses: actions/upload-artifact@v4") {
		t.Error("Expected upload-artifact action to be used for MCP logs upload step")
	}

	// Verify the artifact upload configuration
	if !strings.Contains(lockContentStr, "name: mcp-logs") {
		t.Error("Expected artifact name 'mcp-logs' in upload step")
	}

	if !strings.Contains(lockContentStr, "path: /tmp/mcp-logs/") {
		t.Error("Expected artifact path '/tmp/mcp-logs/' in upload step")
	}

	if !strings.Contains(lockContentStr, "if-no-files-found: ignore") {
		t.Error("Expected 'if-no-files-found: ignore' in upload step")
	}

	// Verify the upload step has 'if: always()' condition
	uploadMCPLogsIndex := strings.Index(lockContentStr, "- name: Upload MCP logs")
	if uploadMCPLogsIndex == -1 {
		t.Fatal("Upload MCP logs step not found")
	}

	// Find the next step after upload MCP logs step
	nextUploadStart := uploadMCPLogsIndex + len("- name: Upload MCP logs")
	uploadStepEnd := strings.Index(lockContentStr[nextUploadStart:], "- name:")
	if uploadStepEnd == -1 {
		uploadStepEnd = len(lockContentStr) - nextUploadStart
	}
	uploadMCPLogsStep := lockContentStr[uploadMCPLogsIndex : nextUploadStart+uploadStepEnd]

	if !strings.Contains(uploadMCPLogsStep, "if: always()") {
		t.Error("Expected upload MCP logs step to have 'if: always()' condition")
	}

	// Verify step ordering: MCP logs upload should be after agentic execution but before agent logs upload
	agenticIndex := strings.Index(lockContentStr, "Execute Claude Code")
	if agenticIndex == -1 {
		// Try alternative agentic step names
		agenticIndex = strings.Index(lockContentStr, "npx @anthropic-ai/claude-code")
		if agenticIndex == -1 {
			agenticIndex = strings.Index(lockContentStr, "uses: githubnext/claude-action")
		}
	}

	uploadAgentLogsIndex := strings.Index(lockContentStr, "Upload agent logs")

	if agenticIndex != -1 && uploadMCPLogsIndex != -1 && uploadAgentLogsIndex != -1 {
		if uploadMCPLogsIndex <= agenticIndex {
			t.Error("MCP logs upload step should appear after agentic execution step")
		}

		if uploadMCPLogsIndex >= uploadAgentLogsIndex {
			t.Error("MCP logs upload step should appear before agent logs upload step")
		}
	}
}

func TestMCPLogsUploadWithoutPlaywright(t *testing.T) {
	// Create a temporary directory for the test
	tmpDir := t.TempDir()

	// Create a test markdown file without Playwright tool configuration
	testMarkdown := `---
on:
  workflow_dispatch:
tools:
  github:
    allowed: [get_repository]
engine: claude
---

# Test Without Playwright

This workflow does not use Playwright but should still have MCP logs upload.
`

	// Write the test file
	mdFile := filepath.Join(tmpDir, "test-no-playwright.md")
	if err := os.WriteFile(mdFile, []byte(testMarkdown), 0644); err != nil {
		t.Fatalf("Failed to write test markdown file: %v", err)
	}

	// Initialize compiler
	compiler := NewCompiler(false, "", "test-version")

	// Compile the workflow
	err := compiler.CompileWorkflow(mdFile)
	if err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	// Read the generated lock file
	lockFile := filepath.Join(tmpDir, "test-no-playwright.lock.yml")
	lockContent, err := os.ReadFile(lockFile)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}

	lockContentStr := string(lockContent)

	// Verify MCP logs upload step EXISTS even when no Playwright is used (always emit)
	if !strings.Contains(lockContentStr, "- name: Upload MCP logs") {
		t.Error("Expected 'Upload MCP logs' step to be present even when Playwright is not used")
	}

	if !strings.Contains(lockContentStr, "name: mcp-logs") {
		t.Error("Expected 'mcp-logs' artifact even when Playwright is not used")
	}

	// Verify the upload step uses actions/upload-artifact@v4
	if !strings.Contains(lockContentStr, "uses: actions/upload-artifact@v4") {
		t.Error("Expected upload-artifact action to be used for MCP logs upload step")
	}

	// Verify the artifact upload configuration
	if !strings.Contains(lockContentStr, "path: /tmp/mcp-logs/") {
		t.Error("Expected artifact path '/tmp/mcp-logs/' in upload step")
	}

	if !strings.Contains(lockContentStr, "if-no-files-found: ignore") {
		t.Error("Expected 'if-no-files-found: ignore' in upload step")
	}
}
