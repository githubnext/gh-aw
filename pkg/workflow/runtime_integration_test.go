package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCompileWorkflowWithRuntimes(t *testing.T) {
	// Create temp directory for test
	tempDir, err := os.MkdirTemp("", "runtime-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create workflow with runtime overrides
	workflowContent := `---
on: push
permissions:
  contents: read
  issues: read
  pull-requests: read
engine: copilot
runtimes:
  node:
    version: "22"
  python:
    version: "3.12"
---

# Test Workflow

Test workflow with runtime overrides.
`
	workflowPath := filepath.Join(tempDir, "test-workflow.md")
	if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	// Compile workflow
	compiler := NewCompiler(false, "", "test")
	workflowData, err := compiler.ParseWorkflowFile(workflowPath)
	if err != nil {
		t.Fatalf("Failed to parse workflow: %v", err)
	}

	// Verify runtimes were extracted
	if workflowData.Runtimes == nil {
		t.Fatal("Expected Runtimes to be non-nil")
	}

	// Check node runtime
	nodeRuntime, ok := workflowData.Runtimes["node"]
	if !ok {
		t.Fatal("Expected 'node' runtime to be present")
	}
	nodeConfig, ok := nodeRuntime.(map[string]any)
	if !ok {
		t.Fatal("Expected node runtime to be a map")
	}
	if nodeConfig["version"] != "22" {
		t.Errorf("Expected node version '22', got '%v'", nodeConfig["version"])
	}

	// Check python runtime
	pythonRuntime, ok := workflowData.Runtimes["python"]
	if !ok {
		t.Fatal("Expected 'python' runtime to be present")
	}
	pythonConfig, ok := pythonRuntime.(map[string]any)
	if !ok {
		t.Fatal("Expected python runtime to be a map")
	}
	if pythonConfig["version"] != "3.12" {
		t.Errorf("Expected python version '3.12', got '%v'", pythonConfig["version"])
	}
}

func TestCompileWorkflowWithRuntimesFromImports(t *testing.T) {
	// Create temp directory for test
	tempDir, err := os.MkdirTemp("", "runtime-import-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create shared directory
	sharedDir := filepath.Join(tempDir, ".github", "workflows", "shared")
	if err := os.MkdirAll(sharedDir, 0755); err != nil {
		t.Fatalf("Failed to create shared directory: %v", err)
	}

	// Create shared workflow with runtime overrides
	sharedContent := `---
runtimes:
  ruby:
    version: "3.2"
  go:
    version: "1.22"
---
`
	sharedPath := filepath.Join(sharedDir, "shared-runtimes.md")
	if err := os.WriteFile(sharedPath, []byte(sharedContent), 0644); err != nil {
		t.Fatalf("Failed to write shared file: %v", err)
	}

	// Create main workflow that imports the shared runtimes
	workflowContent := `---
on: push
permissions:
  contents: read
  issues: read
  pull-requests: read
engine: copilot
imports:
  - shared/shared-runtimes.md
runtimes:
  node:
    version: "22"
---

# Test Workflow

Test workflow with imported runtimes.
`
	workflowPath := filepath.Join(tempDir, ".github", "workflows", "test-workflow.md")
	workflowDir := filepath.Dir(workflowPath)
	if err := os.MkdirAll(workflowDir, 0755); err != nil {
		t.Fatalf("Failed to create workflow directory: %v", err)
	}
	if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	// Compile workflow
	compiler := NewCompiler(false, "", "test")
	workflowData, err := compiler.ParseWorkflowFile(workflowPath)
	if err != nil {
		t.Fatalf("Failed to parse workflow: %v", err)
	}

	// Verify runtimes were merged
	if workflowData.Runtimes == nil {
		t.Fatal("Expected Runtimes to be non-nil")
	}

	// Check node runtime (from main workflow)
	nodeRuntime, ok := workflowData.Runtimes["node"]
	if !ok {
		t.Fatal("Expected 'node' runtime to be present")
	}
	nodeConfig, ok := nodeRuntime.(map[string]any)
	if !ok {
		t.Fatal("Expected node runtime to be a map")
	}
	if nodeConfig["version"] != "22" {
		t.Errorf("Expected node version '22', got '%v'", nodeConfig["version"])
	}

	// Check ruby runtime (from imported workflow)
	rubyRuntime, ok := workflowData.Runtimes["ruby"]
	if !ok {
		t.Fatal("Expected 'ruby' runtime to be present (from import)")
	}
	rubyConfig, ok := rubyRuntime.(map[string]any)
	if !ok {
		t.Fatal("Expected ruby runtime to be a map")
	}
	if rubyConfig["version"] != "3.2" {
		t.Errorf("Expected ruby version '3.2', got '%v'", rubyConfig["version"])
	}

	// Check go runtime (from imported workflow)
	goRuntime, ok := workflowData.Runtimes["go"]
	if !ok {
		t.Fatal("Expected 'go' runtime to be present (from import)")
	}
	goConfig, ok := goRuntime.(map[string]any)
	if !ok {
		t.Fatal("Expected go runtime to be a map")
	}
	if goConfig["version"] != "1.22" {
		t.Errorf("Expected go version '1.22', got '%v'", goConfig["version"])
	}
}

func TestCompileWorkflowWithRuntimesAppliedToSteps(t *testing.T) {
	// Create temp directory for test
	tempDir, err := os.MkdirTemp("", "runtime-steps-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create workflow with custom steps and runtime overrides
	workflowContent := `---
on: push
permissions:
  contents: read
  issues: read
  pull-requests: read
engine: copilot
steps:
  - name: Install dependencies
    run: npm install
runtimes:
  node:
    version: "22"
---

# Test Workflow

Test workflow with runtime overrides applied to steps.
`
	workflowPath := filepath.Join(tempDir, "test-workflow.md")
	if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	// Compile workflow
	compiler := NewCompiler(false, "", "test")
	err = compiler.CompileWorkflow(workflowPath)
	if err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	// Read the generated lock file
	lockPath := strings.TrimSuffix(workflowPath, ".md") + ".lock.yml"
	lockContent, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}

	lockStr := string(lockContent)

	// Verify that Node.js setup step is included with version 22
	if !strings.Contains(lockStr, "actions/setup-node@2028fbc5c25fe9cf00d9f06a71cc4710d4507903") {
		t.Error("Expected setup-node action in lock file")
	}
	if !strings.Contains(lockStr, "node-version: '22'") {
		t.Error("Expected node-version: '22' in lock file")
	}
}

func TestCompileWorkflowWithCustomActionRepo(t *testing.T) {
	// Create temp directory for test
	tempDir, err := os.MkdirTemp("", "runtime-custom-action-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create workflow with custom action-repo and action-version
	workflowContent := `---
on: push
permissions:
  contents: read
  issues: read
  pull-requests: read
engine: copilot
steps:
  - name: Install dependencies
    run: npm install
runtimes:
  node:
    version: "22"
    action-repo: "custom/setup-node"
    action-version: "v5"
---

# Test Workflow

Test workflow with custom setup action.
`
	workflowPath := filepath.Join(tempDir, "test-workflow.md")
	if err := os.WriteFile(workflowPath, []byte(workflowContent), 0644); err != nil {
		t.Fatalf("Failed to write workflow file: %v", err)
	}

	// Compile workflow
	compiler := NewCompiler(false, "", "test")
	err = compiler.CompileWorkflow(workflowPath)
	if err != nil {
		t.Fatalf("Failed to compile workflow: %v", err)
	}

	// Read the generated lock file
	lockPath := strings.TrimSuffix(workflowPath, ".md") + ".lock.yml"
	lockContent, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatalf("Failed to read lock file: %v", err)
	}

	lockStr := string(lockContent)

	// Verify that custom setup action is used
	if !strings.Contains(lockStr, "custom/setup-node@v5") {
		t.Error("Expected custom/setup-node@v5 action in lock file")
	}
	if !strings.Contains(lockStr, "node-version: '22'") {
		t.Error("Expected node-version: '22' in lock file")
	}
}
