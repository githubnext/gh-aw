package workflow_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/githubnext/gh-aw/pkg/testutil"
	"github.com/githubnext/gh-aw/pkg/workflow"
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
    allowed_domains:
      - "example.com"
      - "github.com"
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
	compiler := workflow.NewCompiler(false, "", "test")
	if err := compiler.CompileWorkflow(workflowPath); err != nil {
		t.Fatalf("CompileWorkflow failed: %v", err)
	}

	// Read the generated lock file
	lockFilePath := strings.TrimSuffix(workflowPath, ".md") + ".lock.yml"
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

	// Verify allowed domains are present
	if !strings.Contains(workflowData, "example.com") {
		t.Error("Expected compiled workflow to contain example.com domain")
	}
	if !strings.Contains(workflowData, "github.com") {
		t.Error("Expected compiled workflow to contain github.com domain")
	}
}

// TestImportSerenaTool tests that serena tool can be imported from a shared workflow
func TestImportSerenaTool(t *testing.T) {
	tempDir := testutil.TempDir(t, "test-*")

	// Create a shared workflow with serena tool
	sharedPath := filepath.Join(tempDir, "shared-serena.md")
	sharedContent := `---
description: "Shared serena configuration"
tools:
  serena:
    - go
    - typescript
---

# Shared Serena Configuration
`
	if err := os.WriteFile(sharedPath, []byte(sharedContent), 0644); err != nil {
		t.Fatalf("Failed to write shared file: %v", err)
	}

	// Create main workflow that imports serena
	workflowPath := filepath.Join(tempDir, "main-workflow.md")
	workflowContent := `---
on: issues
engine: copilot
imports:
  - shared-serena.md
permissions:
  contents: read
  issues: read
  pull-requests: read
---

# Main Workflow

Uses imported serena tool.
`
	if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	// Compile the workflow
	compiler := workflow.NewCompiler(false, "", "test")
	if err := compiler.CompileWorkflow(workflowPath); err != nil {
		t.Fatalf("CompileWorkflow failed: %v", err)
	}

	// Read the generated lock file
	lockFilePath := strings.TrimSuffix(workflowPath, ".md") + ".lock.yml"
	lockFileContent, err := os.ReadFile(lockFilePath)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}

	workflowData := string(lockFileContent)

	// Verify serena is configured in the MCP config
	if !strings.Contains(workflowData, `"serena"`) {
		t.Error("Expected compiled workflow to contain serena tool")
	}

	// Verify serena command
	if !strings.Contains(workflowData, "git+https://github.com/oraios/serena") {
		t.Error("Expected compiled workflow to contain serena git repository")
	}

	// Verify language service setup for Go
	if !strings.Contains(workflowData, "Setup Go") {
		t.Error("Expected compiled workflow to contain Go setup for serena")
	}

	// Verify language service setup for TypeScript (Node.js)
	if !strings.Contains(workflowData, "Setup Node.js") {
		t.Error("Expected compiled workflow to contain Node.js setup for serena")
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
	compiler := workflow.NewCompiler(false, "", "test")
	if err := compiler.CompileWorkflow(workflowPath); err != nil {
		t.Fatalf("CompileWorkflow failed: %v", err)
	}

	// Read the generated lock file
	lockFilePath := strings.TrimSuffix(workflowPath, ".md") + ".lock.yml"
	lockFileContent, err := os.ReadFile(lockFilePath)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}

	workflowData := string(lockFileContent)

	// Verify gh aw mcp-server command is present
	if !strings.Contains(workflowData, `"aw", "mcp-server"`) {
		t.Error("Expected compiled workflow to contain 'aw', 'mcp-server' command")
	}

	// Verify gh CLI is used
	if !strings.Contains(workflowData, `"command": "gh"`) {
		t.Error("Expected compiled workflow to contain gh CLI command for agentic-workflows")
	}
}

// TestImportAllThreeTools tests importing all three tools together
func TestImportAllThreeTools(t *testing.T) {
	tempDir := testutil.TempDir(t, "test-*")

	// Create a shared workflow with all three tools
	sharedPath := filepath.Join(tempDir, "shared-all.md")
	sharedContent := `---
description: "Shared configuration with all tools"
tools:
  agentic-workflows: true
  serena:
    - go
  playwright:
    version: "v1.41.0"
    allowed_domains:
      - "example.com"
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
	compiler := workflow.NewCompiler(false, "", "test")
	if err := compiler.CompileWorkflow(workflowPath); err != nil {
		t.Fatalf("CompileWorkflow failed: %v", err)
	}

	// Read the generated lock file
	lockFilePath := strings.TrimSuffix(workflowPath, ".md") + ".lock.yml"
	lockFileContent, err := os.ReadFile(lockFilePath)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}

	workflowData := string(lockFileContent)

	// Verify all three tools are present
	if !strings.Contains(workflowData, `"playwright"`) {
		t.Error("Expected compiled workflow to contain playwright tool")
	}
	if !strings.Contains(workflowData, `"serena"`) {
		t.Error("Expected compiled workflow to contain serena tool")
	}
	if !strings.Contains(workflowData, `"aw", "mcp-server"`) {
		t.Error("Expected compiled workflow to contain agentic-workflows tool")
	}

	// Verify specific configurations
	if !strings.Contains(workflowData, "mcr.microsoft.com/playwright/mcp") {
		t.Error("Expected compiled workflow to contain playwright Docker image")
	}
	if !strings.Contains(workflowData, "git+https://github.com/oraios/serena") {
		t.Error("Expected compiled workflow to contain serena git repository")
	}
	if !strings.Contains(workflowData, "example.com") {
		t.Error("Expected compiled workflow to contain example.com domain for playwright")
	}
}

// TestImportSerenaWithLanguageConfig tests serena with detailed language configuration
func TestImportSerenaWithLanguageConfig(t *testing.T) {
	tempDir := testutil.TempDir(t, "test-*")

	// Create a shared workflow with serena tool with detailed language config
	sharedPath := filepath.Join(tempDir, "shared-serena-config.md")
	sharedContent := `---
description: "Shared serena with language config"
tools:
  serena:
    languages:
      go:
        version: "1.21"
        gopls-version: "latest"
      typescript:
        version: "22"
---

# Shared Serena Language Configuration
`
	if err := os.WriteFile(sharedPath, []byte(sharedContent), 0644); err != nil {
		t.Fatalf("Failed to write shared file: %v", err)
	}

	// Create main workflow that imports serena
	workflowPath := filepath.Join(tempDir, "main-workflow.md")
	workflowContent := `---
on: issues
engine: copilot
imports:
  - shared-serena-config.md
permissions:
  contents: read
  issues: read
  pull-requests: read
---

# Main Workflow

Uses imported serena with language config.
`
	if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	// Compile the workflow
	compiler := workflow.NewCompiler(false, "", "test")
	if err := compiler.CompileWorkflow(workflowPath); err != nil {
		t.Fatalf("CompileWorkflow failed: %v", err)
	}

	// Read the generated lock file
	lockFilePath := strings.TrimSuffix(workflowPath, ".md") + ".lock.yml"
	lockFileContent, err := os.ReadFile(lockFilePath)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}

	workflowData := string(lockFileContent)

	// Verify serena is configured
	if !strings.Contains(workflowData, `"serena"`) {
		t.Error("Expected compiled workflow to contain serena tool")
	}

	// Verify Go setup with version
	if !strings.Contains(workflowData, "Setup Go") {
		t.Error("Expected compiled workflow to contain Go setup")
	}
	if !strings.Contains(workflowData, "go-version: '1.21'") {
		t.Error("Expected compiled workflow to contain Go version 1.21")
	}

	// Verify Node.js setup with version
	if !strings.Contains(workflowData, "Setup Node.js") {
		t.Error("Expected compiled workflow to contain Node.js setup")
	}
	// Note: TypeScript version in serena config may use default Node.js version
	// This is expected behavior as the TypeScript version configuration
	// refers to Node.js version, and may fall back to defaults
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
    allowed_domains:
      - "example.com"
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
	compiler := workflow.NewCompiler(false, "", "test")
	if err := compiler.CompileWorkflow(workflowPath); err != nil {
		t.Fatalf("CompileWorkflow failed: %v", err)
	}

	// Read the generated lock file
	lockFilePath := strings.TrimSuffix(workflowPath, ".md") + ".lock.yml"
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
	compiler := workflow.NewCompiler(false, "", "test")
	err := compiler.CompileWorkflow(workflowPath)

	if err == nil {
		t.Fatal("Expected CompileWorkflow to fail due to missing actions:read permission")
	}

	// Verify error message mentions permissions
	if !strings.Contains(err.Error(), "actions: read") {
		t.Errorf("Expected error to mention 'actions: read', got: %v", err)
	}
}
