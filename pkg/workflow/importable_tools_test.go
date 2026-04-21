//go:build !integration

package workflow_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/stringutil"

	"github.com/github/gh-aw/pkg/testutil"
	"github.com/github/gh-aw/pkg/workflow"
)

// TestImportPlaywrightTool tests that playwright tool can be imported from a shared workflow
func TestImportPlaywrightTool(t *testing.T) {
	tempDir := testutil.TempDir(t, "test-*")

	// Create a shared workflow with playwright tool
	sharedPath := filepath.Join(tempDir, "shared-playwright.md")
	sharedContent := `---
description: "Shared playwright configuration"
tools:
  playwright:
    version: "v1.41.0"
network:
  allowed:
    - playwright
---

# Shared Playwright Configuration
`
	if err := os.WriteFile(sharedPath, []byte(sharedContent), 0644); err != nil {
		t.Fatalf("Failed to write shared file: %v", err)
	}

	// Create main workflow that imports playwright
	workflowPath := filepath.Join(tempDir, "main-workflow.md")
	workflowContent := `---
on: issues
engine: copilot
imports:
  - shared-playwright.md
permissions:
  contents: read
  issues: read
  pull-requests: read
---

# Main Workflow

Uses imported playwright tool.
`
	if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	// Compile the workflow
	compiler := workflow.NewCompiler()
	if err := compiler.CompileWorkflow(workflowPath); err != nil {
		t.Fatalf("CompileWorkflow failed: %v", err)
	}

	// Read the generated lock file
	lockFilePath := stringutil.MarkdownToLockFile(workflowPath)
	lockFileContent, err := os.ReadFile(lockFilePath)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}

	workflowData := string(lockFileContent)

	// Verify playwright is configured in the MCP config
	if !strings.Contains(workflowData, `"playwright"`) {
		t.Error("Expected compiled workflow to contain playwright tool")
	}

	// Verify playwright Docker image
	if !strings.Contains(workflowData, "mcr.microsoft.com/playwright/mcp") {
		t.Error("Expected compiled workflow to contain playwright Docker image")
	}
}

// TestImportAgenticWorkflowsTool tests that agentic-workflows tool can be imported
func TestImportAgenticWorkflowsTool(t *testing.T) {
	tempDir := testutil.TempDir(t, "test-*")

	// Create a shared workflow with agentic-workflows tool
	sharedPath := filepath.Join(tempDir, "shared-aw.md")
	sharedContent := `---
description: "Shared agentic-workflows configuration"
tools:
  agentic-workflows: true
permissions:
  actions: read
---

# Shared Agentic Workflows Configuration
`
	if err := os.WriteFile(sharedPath, []byte(sharedContent), 0644); err != nil {
		t.Fatalf("Failed to write shared file: %v", err)
	}

	// Create main workflow that imports agentic-workflows
	workflowPath := filepath.Join(tempDir, "main-workflow.md")
	workflowContent := `---
on: issues
engine: copilot
imports:
  - shared-aw.md
permissions:
  actions: read
  contents: read
  issues: read
  pull-requests: read
---

# Main Workflow

Uses imported agentic-workflows tool.
`
	if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	// Compile the workflow
	compiler := workflow.NewCompiler()
	if err := compiler.CompileWorkflow(workflowPath); err != nil {
		t.Fatalf("CompileWorkflow failed: %v", err)
	}

	// Read the generated lock file
	lockFilePath := stringutil.MarkdownToLockFile(workflowPath)
	lockFileContent, err := os.ReadFile(lockFilePath)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}

	workflowData := string(lockFileContent)

	// Verify containerized agenticworkflows server is present (per MCP Gateway Specification v1.0.0)
	// In dev mode, no entrypoint or entrypointArgs (uses container's defaults)
	if strings.Contains(workflowData, `"entrypointArgs"`) {
		t.Error("Did not expect entrypointArgs field in dev mode (uses container's CMD)")
	}

	if strings.Contains(workflowData, `"--cmd"`) {
		t.Error("Did not expect --cmd argument in dev mode")
	}

	// Verify container format is used (not command format)
	// In dev mode, should use locally built image
	if !strings.Contains(workflowData, `"container": "localhost/gh-aw:dev"`) {
		t.Error("Expected compiled workflow to contain localhost/gh-aw:dev container for agentic-workflows in dev mode")
	}

	// Verify NO entrypoint field (uses container's default ENTRYPOINT)
	if strings.Contains(workflowData, `"entrypoint"`) {
		t.Error("Did not expect entrypoint field in dev mode (uses container's ENTRYPOINT)")
	}

	// Verify ${RUNNER_TEMP}/gh-aw is always mounted read-only for security
	if !strings.Contains(workflowData, `${RUNNER_TEMP}/gh-aw:${RUNNER_TEMP}/gh-aw:ro`) {
		t.Error("Expected ${RUNNER_TEMP}/gh-aw to be mounted read-only in AWF for security")
	}

	// Verify DEBUG and GITHUB_TOKEN are present
	if !strings.Contains(workflowData, `"DEBUG": "*"`) {
		t.Error("Expected DEBUG set to literal '*' in env vars")
	}
	if !strings.Contains(workflowData, `"GITHUB_TOKEN"`) {
		t.Error("Expected GITHUB_TOKEN in env vars")
	}

	// Verify working directory args are present
	if !strings.Contains(workflowData, `"args": ["--network", "host", "-w", "\${GITHUB_WORKSPACE}"]`) {
		t.Error("Expected args with network access and working directory")
	}
}

// TestImportMultipleTools tests importing multiple tools together
func TestImportMultipleTools(t *testing.T) {
	tempDir := testutil.TempDir(t, "test-*")

	// Create a shared workflow with multiple tools
	sharedPath := filepath.Join(tempDir, "shared-all.md")
	sharedContent := `---
description: "Shared configuration with multiple tools"
tools:
  agentic-workflows: true
  playwright:
    version: "v1.41.0"
permissions:
  actions: read
network:
  allowed:
    - playwright
---

# Shared All Tools Configuration
`
	if err := os.WriteFile(sharedPath, []byte(sharedContent), 0644); err != nil {
		t.Fatalf("Failed to write shared file: %v", err)
	}

	// Create main workflow that imports all tools
	workflowPath := filepath.Join(tempDir, "main-workflow.md")
	workflowContent := `---
on: issues
engine: copilot
imports:
  - shared-all.md
permissions:
  actions: read
  contents: read
  issues: read
  pull-requests: read
---

# Main Workflow

Uses all imported tools.
`
	if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	// Compile the workflow
	compiler := workflow.NewCompiler()
	if err := compiler.CompileWorkflow(workflowPath); err != nil {
		t.Fatalf("CompileWorkflow failed: %v", err)
	}

	// Read the generated lock file
	lockFilePath := stringutil.MarkdownToLockFile(workflowPath)
	lockFileContent, err := os.ReadFile(lockFilePath)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}

	workflowData := string(lockFileContent)

	// Verify tools are present
	if !strings.Contains(workflowData, `"playwright"`) {
		t.Error("Expected compiled workflow to contain playwright tool")
	}
	// Per MCP Gateway Specification v1.0.0, agentic-workflows uses containerized format
	if !strings.Contains(workflowData, `"`+constants.AgenticWorkflowsMCPServerID.String()+`"`) {
		t.Error("Expected compiled workflow to contain agentic-workflows tool")
	}

	// Verify specific configurations
	if !strings.Contains(workflowData, "mcr.microsoft.com/playwright/mcp") {
		t.Error("Expected compiled workflow to contain playwright Docker image")
	}
}

// TestImportPlaywrightWithCustomArgs tests playwright with custom arguments
func TestImportPlaywrightWithCustomArgs(t *testing.T) {
	tempDir := testutil.TempDir(t, "test-*")

	// Create a shared workflow with playwright tool with custom args
	sharedPath := filepath.Join(tempDir, "shared-playwright-args.md")
	sharedContent := `---
description: "Shared playwright with custom args"
tools:
  playwright:
    version: "v1.41.0"
    args:
      - "--custom-flag"
      - "value"
network:
  allowed:
    - playwright
---

# Shared Playwright with Args
`
	if err := os.WriteFile(sharedPath, []byte(sharedContent), 0644); err != nil {
		t.Fatalf("Failed to write shared file: %v", err)
	}

	// Create main workflow that imports playwright
	workflowPath := filepath.Join(tempDir, "main-workflow.md")
	workflowContent := `---
on: issues
engine: copilot
imports:
  - shared-playwright-args.md
permissions:
  contents: read
  issues: read
  pull-requests: read
---

# Main Workflow

Uses imported playwright with custom args.
`
	if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	// Compile the workflow
	compiler := workflow.NewCompiler()
	if err := compiler.CompileWorkflow(workflowPath); err != nil {
		t.Fatalf("CompileWorkflow failed: %v", err)
	}

	// Read the generated lock file
	lockFilePath := stringutil.MarkdownToLockFile(workflowPath)
	lockFileContent, err := os.ReadFile(lockFilePath)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}

	workflowData := string(lockFileContent)

	// Verify playwright is configured
	if !strings.Contains(workflowData, `"playwright"`) {
		t.Error("Expected compiled workflow to contain playwright tool")
	}

	// Verify custom args are present
	if !strings.Contains(workflowData, "--custom-flag") {
		t.Error("Expected compiled workflow to contain --custom-flag custom argument")
	}
	if !strings.Contains(workflowData, "value") {
		t.Error("Expected compiled workflow to contain custom argument value")
	}
}

// TestImportAgenticWorkflowsRequiresPermissions tests that agentic-workflows tool requires actions:read permission
func TestImportAgenticWorkflowsRequiresPermissions(t *testing.T) {
	tempDir := testutil.TempDir(t, "test-*")

	// Create a shared workflow with agentic-workflows tool
	sharedPath := filepath.Join(tempDir, "shared-aw.md")
	sharedContent := `---
description: "Shared agentic-workflows configuration"
tools:
  agentic-workflows: true
permissions:
  actions: read
---

# Shared Agentic Workflows Configuration
`
	if err := os.WriteFile(sharedPath, []byte(sharedContent), 0644); err != nil {
		t.Fatalf("Failed to write shared file: %v", err)
	}

	// Create main workflow WITHOUT actions:read permission
	workflowPath := filepath.Join(tempDir, "main-workflow.md")
	workflowContent := `---
on: issues
engine: copilot
imports:
  - shared-aw.md
permissions:
  contents: read
  issues: read
  pull-requests: read
---

# Main Workflow

Missing actions:read permission.
`
	if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	// Compile the workflow - should fail due to missing permission
	compiler := workflow.NewCompiler()
	err := compiler.CompileWorkflow(workflowPath)

	if err == nil {
		t.Fatal("Expected CompileWorkflow to fail due to missing actions:read permission")
	}

	// Verify error message mentions permissions
	if !strings.Contains(err.Error(), "actions: read") {
		t.Errorf("Expected error to mention 'actions: read', got: %v", err)
	}
}

// TestImportEditTool tests that edit tool can be imported from a shared workflow
func TestImportEditTool(t *testing.T) {
	tempDir := testutil.TempDir(t, "test-*")

	// Create a shared workflow with edit tool
	sharedPath := filepath.Join(tempDir, "shared-edit.md")
	sharedContent := `---
description: "Shared edit tool configuration"
tools:
  edit:
---

# Shared Edit Tool Configuration
`
	if err := os.WriteFile(sharedPath, []byte(sharedContent), 0644); err != nil {
		t.Fatalf("Failed to write shared file: %v", err)
	}

	// Create main workflow that imports edit tool
	workflowPath := filepath.Join(tempDir, "main-workflow.md")
	workflowContent := `---
on: issues
engine: copilot
imports:
  - shared-edit.md
permissions:
  contents: read
---

# Main Workflow

Uses imported edit tool.
`
	if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	// Compile the workflow
	compiler := workflow.NewCompiler()
	if err := compiler.CompileWorkflow(workflowPath); err != nil {
		t.Fatalf("CompileWorkflow failed: %v", err)
	}

	// Read the generated lock file
	lockFilePath := stringutil.MarkdownToLockFile(workflowPath)
	lockFileContent, err := os.ReadFile(lockFilePath)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}

	workflowData := string(lockFileContent)

	// Verify edit tool functionality is present
	// The edit tool enables --allow-all-paths flag in Copilot
	if !strings.Contains(workflowData, "--allow-all-paths") {
		t.Error("Expected compiled workflow to contain --allow-all-paths flag for edit tool")
	}
}

// TestImportWebFetchTool tests that web-fetch tool can be imported from a shared workflow
func TestImportWebFetchTool(t *testing.T) {
	tempDir := testutil.TempDir(t, "test-*")

	// Create a shared workflow with web-fetch tool
	sharedPath := filepath.Join(tempDir, "shared-web-fetch.md")
	sharedContent := `---
description: "Shared web-fetch tool configuration"
tools:
  web-fetch:
---

# Shared Web Fetch Tool Configuration
`
	if err := os.WriteFile(sharedPath, []byte(sharedContent), 0644); err != nil {
		t.Fatalf("Failed to write shared file: %v", err)
	}

	// Create main workflow that imports web-fetch tool
	workflowPath := filepath.Join(tempDir, "main-workflow.md")
	workflowContent := `---
on: issues
engine: copilot
imports:
  - shared-web-fetch.md
permissions:
  contents: read
---

# Main Workflow

Uses imported web-fetch tool.
`
	if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	// Compile the workflow
	compiler := workflow.NewCompiler()
	if err := compiler.CompileWorkflow(workflowPath); err != nil {
		t.Fatalf("CompileWorkflow failed: %v", err)
	}

	// Read the generated lock file
	lockFilePath := stringutil.MarkdownToLockFile(workflowPath)
	lockFileContent, err := os.ReadFile(lockFilePath)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}

	// Verify compilation succeeded
	// Note: Copilot has built-in web-fetch support, so no explicit MCP configuration is needed
	// The test verifies that the workflow compiles successfully when web-fetch is imported
	_ = lockFileContent // Compilation success is sufficient verification
}

// TestImportWebSearchTool tests that web-search tool can be imported from a shared workflow
func TestImportWebSearchTool(t *testing.T) {
	tempDir := testutil.TempDir(t, "test-*")

	// Create a shared workflow with web-search tool
	sharedPath := filepath.Join(tempDir, "shared-web-search.md")
	sharedContent := `---
description: "Shared web-search tool configuration"
tools:
  web-search:
---

# Shared Web Search Tool Configuration
`
	if err := os.WriteFile(sharedPath, []byte(sharedContent), 0644); err != nil {
		t.Fatalf("Failed to write shared file: %v", err)
	}

	// Create main workflow that imports web-search tool
	// Use Claude engine since Copilot doesn't support web-search
	workflowPath := filepath.Join(tempDir, "main-workflow.md")
	workflowContent := `---
on: issues
engine: claude
imports:
  - shared-web-search.md
permissions:
  contents: read
---

# Main Workflow

Uses imported web-search tool.
`
	if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	// Compile the workflow
	compiler := workflow.NewCompiler()
	if err := compiler.CompileWorkflow(workflowPath); err != nil {
		t.Fatalf("CompileWorkflow failed: %v", err)
	}

	// Read the generated lock file
	lockFilePath := stringutil.MarkdownToLockFile(workflowPath)
	lockFileContent, err := os.ReadFile(lockFilePath)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}

	workflowData := string(lockFileContent)

	// Verify web-search tool is configured
	// For Claude, web-search is a native tool capability
	if !strings.Contains(workflowData, "WebSearch") {
		t.Error("Expected compiled workflow to contain WebSearch tool configuration for Claude")
	}
}

// TestImportTimeoutTool tests that timeout tool setting can be imported from a shared workflow
func TestImportTimeoutTool(t *testing.T) {
	tempDir := testutil.TempDir(t, "test-*")

	// Create a shared workflow with timeout setting
	sharedPath := filepath.Join(tempDir, "shared-timeout.md")
	sharedContent := `---
description: "Shared timeout configuration"
tools:
  timeout: 90
---

# Shared Timeout Configuration
`
	if err := os.WriteFile(sharedPath, []byte(sharedContent), 0644); err != nil {
		t.Fatalf("Failed to write shared file: %v", err)
	}

	// Create main workflow that imports timeout setting
	workflowPath := filepath.Join(tempDir, "main-workflow.md")
	workflowContent := `---
on: issues
engine: copilot
imports:
  - shared-timeout.md
permissions:
  contents: read
---

# Main Workflow

Uses imported timeout setting.
`
	if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	// Compile the workflow
	compiler := workflow.NewCompiler()
	if err := compiler.CompileWorkflow(workflowPath); err != nil {
		t.Fatalf("CompileWorkflow failed: %v", err)
	}

	// Read the generated lock file
	lockFilePath := stringutil.MarkdownToLockFile(workflowPath)
	lockFileContent, err := os.ReadFile(lockFilePath)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}

	workflowData := string(lockFileContent)

	// Verify timeout is configured
	// The timeout setting sets environment variables for MCP and bash tools
	hasTimeout := strings.Contains(workflowData, "MCP_TOOL_TIMEOUT") ||
		strings.Contains(workflowData, "90000") ||
		strings.Contains(workflowData, "GH_AW_TOOL_TIMEOUT")
	if !hasTimeout {
		t.Error("Expected compiled workflow to contain timeout configuration (90 seconds)")
	}
}

// TestImportStartupTimeoutTool tests that startup-timeout tool setting can be imported from a shared workflow
func TestImportStartupTimeoutTool(t *testing.T) {
	tempDir := testutil.TempDir(t, "test-*")

	// Create a shared workflow with startup-timeout setting
	sharedPath := filepath.Join(tempDir, "shared-startup-timeout.md")
	sharedContent := `---
description: "Shared startup-timeout configuration"
tools:
  startup-timeout: 60
---

# Shared Startup Timeout Configuration
`
	if err := os.WriteFile(sharedPath, []byte(sharedContent), 0644); err != nil {
		t.Fatalf("Failed to write shared file: %v", err)
	}

	// Create main workflow that imports startup-timeout setting
	workflowPath := filepath.Join(tempDir, "main-workflow.md")
	workflowContent := `---
on: issues
engine: copilot
imports:
  - shared-startup-timeout.md
permissions:
  contents: read
---

# Main Workflow

Uses imported startup-timeout setting.
`
	if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	// Compile the workflow
	compiler := workflow.NewCompiler()
	if err := compiler.CompileWorkflow(workflowPath); err != nil {
		t.Fatalf("CompileWorkflow failed: %v", err)
	}

	// Read the generated lock file
	lockFilePath := stringutil.MarkdownToLockFile(workflowPath)
	lockFileContent, err := os.ReadFile(lockFilePath)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}

	workflowData := string(lockFileContent)

	// Verify startup-timeout is configured
	// The startup-timeout setting sets environment variables for MCP startup
	hasStartupTimeout := strings.Contains(workflowData, "MCP_TIMEOUT") ||
		strings.Contains(workflowData, "60000") ||
		strings.Contains(workflowData, "GH_AW_STARTUP_TIMEOUT")
	if !hasStartupTimeout {
		t.Error("Expected compiled workflow to contain startup-timeout configuration (60 seconds)")
	}
}

// TestImportMultipleNeutralTools tests importing multiple neutral tools together
func TestImportMultipleNeutralTools(t *testing.T) {
	tempDir := testutil.TempDir(t, "test-*")

	// Create a shared workflow with multiple neutral tools
	sharedPath := filepath.Join(tempDir, "shared-neutral-tools.md")
	sharedContent := `---
description: "Shared configuration with multiple neutral tools"
tools:
  edit:
  web-fetch:
  safety-prompt: true
  timeout: 120
  startup-timeout: 90
---

# Shared Neutral Tools Configuration
`
	if err := os.WriteFile(sharedPath, []byte(sharedContent), 0644); err != nil {
		t.Fatalf("Failed to write shared file: %v", err)
	}

	// Create main workflow that imports all neutral tools
	workflowPath := filepath.Join(tempDir, "main-workflow.md")
	workflowContent := `---
on: issues
engine: copilot
imports:
  - shared-neutral-tools.md
permissions:
  contents: read
---

# Main Workflow

Uses all imported neutral tools.
`
	if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	// Compile the workflow
	compiler := workflow.NewCompiler()
	if err := compiler.CompileWorkflow(workflowPath); err != nil {
		t.Fatalf("CompileWorkflow failed: %v", err)
	}

	// Read the generated lock file
	lockFilePath := stringutil.MarkdownToLockFile(workflowPath)
	lockFileContent, err := os.ReadFile(lockFilePath)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}

	workflowData := string(lockFileContent)

	// Verify edit tool is present (--allow-all-paths flag)
	if !strings.Contains(workflowData, "--allow-all-paths") {
		t.Error("Expected compiled workflow to contain --allow-all-paths flag for edit tool")
	}

	// Note: web-fetch has built-in Copilot support, so no explicit MCP configuration is needed
	// The test verifies that web-fetch compiles successfully when imported

	// Verify timeout is configured (120 seconds)
	hasTimeout := strings.Contains(workflowData, "120000") ||
		strings.Contains(workflowData, "MCP_TOOL_TIMEOUT") ||
		strings.Contains(workflowData, "GH_AW_TOOL_TIMEOUT")
	if !hasTimeout {
		t.Error("Expected compiled workflow to contain timeout configuration (120 seconds)")
	}

	// Verify startup-timeout is configured (90 seconds)
	hasStartupTimeout := strings.Contains(workflowData, "90000") ||
		strings.Contains(workflowData, "MCP_TIMEOUT") ||
		strings.Contains(workflowData, "GH_AW_STARTUP_TIMEOUT")
	if !hasStartupTimeout {
		t.Error("Expected compiled workflow to contain startup-timeout configuration (90 seconds)")
	}
}

// TestBashMainWorkflowOverridesImportBashList verifies that bash: true in the main workflow
// (unrestricted bash) takes precedence over a specific bash command list defined in an import.
// This prevents imports from accidentally restricting the main workflow's bash permissions.
func TestBashMainWorkflowOverridesImportBashList(t *testing.T) {
	tempDir := testutil.TempDir(t, "test-*")

	// Create a shared workflow with a restricted bash tool list
	sharedPath := filepath.Join(tempDir, "bash-restricted.md")
	sharedContent := `---
description: "Shared workflow that restricts bash to specific commands"
tools:
  bash:
    - "ls"
    - "cat"
---

# Restricted Bash Import
`
	if err := os.WriteFile(sharedPath, []byte(sharedContent), 0644); err != nil {
		t.Fatalf("Failed to write shared file: %v", err)
	}

	// Main workflow with bash: true (unrestricted), importing the restricted shared file
	workflowPath := filepath.Join(tempDir, "my-workflow.md")
	workflowContent := `---
on: issues
engine: copilot
strict: false
permissions:
  contents: read
  issues: read
tools:
  bash: true
sandbox:
  agent: false
imports:
  - bash-restricted.md
---

# My Workflow

This workflow needs unrestricted bash.
`
	if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	compiler := workflow.NewCompiler()
	if err := compiler.CompileWorkflow(workflowPath); err != nil {
		t.Fatalf("CompileWorkflow failed: %v", err)
	}

	lockFilePath := stringutil.MarkdownToLockFile(workflowPath)
	lockFileContent, err := os.ReadFile(lockFilePath)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}

	workflowData := string(lockFileContent)

	// The compiled output must NOT contain shell(ls) or shell(cat) — those would indicate
	// the import's restricted list won over the main workflow's bash: true.
	restrictedCmds := []string{"shell(ls)", "shell(cat)"}
	for _, cmd := range restrictedCmds {
		if strings.Contains(workflowData, cmd) {
			t.Errorf("Expected compiled workflow to NOT contain %q (bash: true in main workflow should override import's restricted bash list)", cmd)
		}
	}

	// bash: true gets converted to unrestricted bash (["*"]), which produces --allow-all-tools.
	if !strings.Contains(workflowData, "--allow-all-tools") {
		t.Error("Expected compiled workflow to contain '--allow-all-tools' (unrestricted bash from bash: true in main workflow)")
	}
}

// TestCacheMemoryMapOverridesBoolInImportChain verifies that when a parent import sets
// tools.cache-memory: {key: "specific-key"}, and a further-nested (child) import sets
// tools.cache-memory: true, the parent import's specific key wins.
// This preserves custom cache isolation across workflow runs.
func TestCacheMemoryMapOverridesBoolInImportChain(t *testing.T) {
	tempDir := testutil.TempDir(t, "test-*")

	// Create a deeply nested import that sets cache-memory: true (generic)
	childPath := filepath.Join(tempDir, "generic-cache.md")
	childContent := `---
description: "A utility import that enables cache-memory generically"
tools:
  cache-memory: true
---

# Generic Cache Utility
`
	if err := os.WriteFile(childPath, []byte(childContent), 0644); err != nil {
		t.Fatalf("Failed to write child shared file: %v", err)
	}

	// Create a parent import that sets a specific cache-memory key and imports the child
	parentPath := filepath.Join(tempDir, "specific-cache.md")
	parentContent := `---
description: "Parent import with a specific cache-memory key"
tools:
  cache-memory:
    key: my-specific-cache-key
imports:
  - generic-cache.md
---

# Specific Cache Parent
`
	if err := os.WriteFile(parentPath, []byte(parentContent), 0644); err != nil {
		t.Fatalf("Failed to write parent shared file: %v", err)
	}

	// Main workflow with no cache-memory of its own, importing the parent
	workflowPath := filepath.Join(tempDir, "my-workflow.md")
	workflowContent := `---
on: issues
engine: copilot
strict: false
permissions:
  contents: read
  issues: read
sandbox:
  agent: false
imports:
  - specific-cache.md
---

# My Workflow

This workflow relies on the parent import's cache-memory key.
`
	if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	compiler := workflow.NewCompiler()
	if err := compiler.CompileWorkflow(workflowPath); err != nil {
		t.Fatalf("CompileWorkflow failed: %v", err)
	}

	lockFilePath := stringutil.MarkdownToLockFile(workflowPath)
	lockFileContent, err := os.ReadFile(lockFilePath)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}

	workflowData := string(lockFileContent)

	// The specific key from the parent import must appear in the compiled cache key.
	if !strings.Contains(workflowData, "my-specific-cache-key") {
		t.Error("Expected compiled workflow to contain 'my-specific-cache-key' in the cache key (parent import's specific key should win over child's generic cache-memory: true)")
	}

	// The generic default key pattern (using GH_AW_WORKFLOW_ID_SANITIZED) must NOT be
	// used as the cache key suffix, which would indicate cache-memory: true won.
	// Note: GH_AW_WORKFLOW_ID_SANITIZED is also set as an env var; we look for it in the
	// context of a "key:" line to distinguish the two.
	if strings.Contains(workflowData, "key: memory-none-nopolicy-${{ env.GH_AW_WORKFLOW_ID_SANITIZED }}") {
		t.Error("Expected compiled workflow cache key to NOT use the default GH_AW_WORKFLOW_ID_SANITIZED pattern (parent import's specific key should win)")
	}
}

// TestGitHubToolFalseOverridesImport verifies that tools.github: false in the main workflow
// takes precedence over tools.github.mode: remote (or any github tool config) from an import.
func TestGitHubToolFalseOverridesImport(t *testing.T) {
	tempDir := testutil.TempDir(t, "test-*")

	// Create a shared workflow that enables the GitHub MCP server
	sharedPath := filepath.Join(tempDir, "side-repository.md")
	sharedContent := `---
description: "Shared side-repo configuration with GitHub MCP"
tools:
  github:
    mode: remote
---

# Shared Side Repository Configuration
`
	if err := os.WriteFile(sharedPath, []byte(sharedContent), 0644); err != nil {
		t.Fatalf("Failed to write shared file: %v", err)
	}

	// Create main workflow that explicitly disables github tools, but imports the shared file
	workflowPath := filepath.Join(tempDir, "my-workflow.md")
	workflowContent := `---
on: issues
engine: copilot
strict: false
permissions:
  contents: read
  issues: read
tools:
  github: false
sandbox:
  agent: false
imports:
  - side-repository.md
---

# My Workflow

This workflow explicitly opts out of GitHub MCP tools.
`
	if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	compiler := workflow.NewCompiler()
	if err := compiler.CompileWorkflow(workflowPath); err != nil {
		t.Fatalf("CompileWorkflow failed: %v", err)
	}

	lockFilePath := stringutil.MarkdownToLockFile(workflowPath)
	lockFileContent, err := os.ReadFile(lockFilePath)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}

	workflowData := string(lockFileContent)

	// The GitHub MCP server should NOT appear in the compiled output because
	// tools.github: false in the main workflow must override the import.
	githubMCPIndicators := []string{
		"X-MCP-Toolsets",
		"api.githubcopilot.com/mcp/",
		"ghcr.io/github/github-mcp-server",
	}
	for _, indicator := range githubMCPIndicators {
		if strings.Contains(workflowData, indicator) {
			t.Errorf("Expected compiled workflow to NOT contain GitHub MCP server indicator %q (github MCP server should be disabled by tools.github: false)", indicator)
		}
	}
}

// TestGitHubModeTopLevelOverridesImport verifies that tools.github.mode in the main
// workflow overrides imported tools.github.mode values.
func TestGitHubModeTopLevelOverridesImport(t *testing.T) {
	tempDir := testutil.TempDir(t, "test-*")

	sharedPath := filepath.Join(tempDir, "shared.md")
	sharedContent := `---
tools:
  github:
    mode: gh-proxy
---
`
	if err := os.WriteFile(sharedPath, []byte(sharedContent), 0644); err != nil {
		t.Fatalf("Failed to write shared file: %v", err)
	}

	workflowPath := filepath.Join(tempDir, "main.md")
	workflowContent := `---
on: issues
engine: copilot
strict: false
permissions:
  contents: read
  issues: read
tools:
  github:
    mode: local
imports:
  - shared.md
---

# Main
`
	if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	compiler := workflow.NewCompiler()
	if err := compiler.CompileWorkflow(workflowPath); err != nil {
		t.Fatalf("CompileWorkflow failed: %v", err)
	}

	lockFilePath := stringutil.MarkdownToLockFile(workflowPath)
	lockFileContent, err := os.ReadFile(lockFilePath)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}

	workflowData := string(lockFileContent)
	if strings.Contains(workflowData, "cli_proxy_prompt.md") {
		t.Error("Expected top-level mode:local to win over imported mode:gh-proxy (cli_proxy_prompt.md should not be present)")
	}
	if !strings.Contains(workflowData, "github_mcp_tools_prompt.md") {
		t.Error("Expected GitHub MCP prompt guidance when top-level mode is local")
	}
}
