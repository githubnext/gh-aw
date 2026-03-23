//go:build !integration

package workflow

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDispatchWorkflowMultiDirectoryDiscovery tests that dispatch_workflow can find workflows
// in multiple directories (same directory and .github/workflows)
func TestDispatchWorkflowMultiDirectoryDiscovery(t *testing.T) {
	compiler := NewCompilerWithVersion("1.0.0")

	// Create a temporary directory structure
	tmpDir := t.TempDir()
	awDir := filepath.Join(tmpDir, ".github", "aw")
	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")

	err := os.MkdirAll(awDir, 0755)
	require.NoError(t, err, "Failed to create aw directory")
	err = os.MkdirAll(workflowsDir, 0755)
	require.NoError(t, err, "Failed to create workflows directory")

	// Create a workflow in .github/workflows with workflow_dispatch
	ciWorkflow := `name: CI
on:
  push:
  workflow_dispatch:
    inputs:
      test_mode:
        description: 'Test mode'
        type: choice
        options:
          - unit
          - integration
        required: false
        default: 'unit'
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - run: echo "Running tests"
`
	ciFile := filepath.Join(workflowsDir, "ci.lock.yml")
	err = os.WriteFile(ciFile, []byte(ciWorkflow), 0644)
	require.NoError(t, err, "Failed to write ci workflow")

	// Create a dispatcher workflow in .github/aw that references ci
	dispatcherWorkflow := `---
on: issues
engine: copilot
permissions:
  contents: read
safe-outputs:
  dispatch-workflow:
    workflows:
      - ci
    max: 1
---

# Dispatcher Workflow

This workflow dispatches to ci workflow.
`
	dispatcherFile := filepath.Join(awDir, "dispatcher.md")
	err = os.WriteFile(dispatcherFile, []byte(dispatcherWorkflow), 0644)
	require.NoError(t, err, "Failed to write dispatcher workflow")

	// Change to the aw directory for compilation
	oldDir, err := os.Getwd()
	require.NoError(t, err, "Failed to get current directory")
	err = os.Chdir(awDir)
	require.NoError(t, err, "Failed to change directory")
	defer func() { _ = os.Chdir(oldDir) }()

	// Parse the dispatcher workflow
	workflowData, err := compiler.ParseWorkflowFile("dispatcher.md")
	require.NoError(t, err, "Failed to parse workflow")
	require.NotNil(t, workflowData.SafeOutputs, "SafeOutputs should not be nil")
	require.NotNil(t, workflowData.SafeOutputs.DispatchWorkflow, "DispatchWorkflow should not be nil")

	// Verify dispatch-workflow configuration
	assert.Equal(t, strPtr("1"), workflowData.SafeOutputs.DispatchWorkflow.Max)
	assert.Equal(t, []string{"ci"}, workflowData.SafeOutputs.DispatchWorkflow.Workflows)

	// Validate the workflow - should find ci in .github/workflows
	err = compiler.validateDispatchWorkflow(workflowData, dispatcherFile)
	assert.NoError(t, err, "Validation should succeed - ci workflow should be found in .github/workflows")
}

// TestDispatchWorkflowOnlySearchesGithubWorkflows tests that workflows are only
// searched in .github/workflows, not in the same directory as the current workflow
func TestDispatchWorkflowOnlySearchesGithubWorkflows(t *testing.T) {
	tmpDir := t.TempDir()
	awDir := filepath.Join(tmpDir, ".github", "aw")
	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")

	err := os.MkdirAll(awDir, 0755)
	require.NoError(t, err, "Failed to create aw directory")
	err = os.MkdirAll(workflowsDir, 0755)
	require.NoError(t, err, "Failed to create workflows directory")

	// Create a workflow in .github/workflows with workflow_dispatch
	workflowsTestWorkflow := `name: Test (workflows)
on:
  workflow_dispatch:
    inputs:
      env:
        description: 'Environment'
        default: 'staging'
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - run: echo "From workflows"
`
	workflowsTestFile := filepath.Join(workflowsDir, "test.lock.yml")
	err = os.WriteFile(workflowsTestFile, []byte(workflowsTestWorkflow), 0644)
	require.NoError(t, err, "Failed to write workflows test workflow")

	// Create a workflow with the same name in .github/aw (should be ignored)
	awTestWorkflow := `name: Test (aw)
on:
  workflow_dispatch:
    inputs:
      mode:
        description: 'Test mode'
        default: 'fast'
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - run: echo "From aw"
`
	awTestFile := filepath.Join(awDir, "test.lock.yml")
	err = os.WriteFile(awTestFile, []byte(awTestWorkflow), 0644)
	require.NoError(t, err, "Failed to write aw test workflow")

	// Create a dispatcher workflow that references test
	dispatcherWorkflow := `---
on: issues
engine: copilot
permissions:
  contents: read
safe-outputs:
  dispatch-workflow:
    workflows:
      - test
    max: 1
---

# Dispatcher Workflow

This workflow dispatches to test workflow.
`
	dispatcherFile := filepath.Join(awDir, "dispatcher.md")
	err = os.WriteFile(dispatcherFile, []byte(dispatcherWorkflow), 0644)
	require.NoError(t, err, "Failed to write dispatcher workflow")

	// Test that findWorkflowFile finds the one in .github/workflows only (not .github/aw)
	fileResult, err := findWorkflowFile("test", dispatcherFile)
	require.NoError(t, err, "findWorkflowFile should succeed")
	assert.True(t, fileResult.lockExists, "Lock file should exist")

	// Verify it found the workflows version (not aw version)
	assert.Contains(t, fileResult.lockPath, filepath.Join(".github", "workflows", "test.lock.yml"),
		"Should find workflow in .github/workflows only")
	assert.NotContains(t, fileResult.lockPath, filepath.Join(".github", "aw", "test.lock.yml"),
		"Should NOT find workflow in same directory")
}

// TestDispatchWorkflowNotFound tests error handling when workflow is not found
func TestDispatchWorkflowNotFound(t *testing.T) {
	compiler := NewCompilerWithVersion("1.0.0")

	tmpDir := t.TempDir()
	awDir := filepath.Join(tmpDir, ".github", "aw")
	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")

	err := os.MkdirAll(awDir, 0755)
	require.NoError(t, err, "Failed to create aw directory")
	err = os.MkdirAll(workflowsDir, 0755)
	require.NoError(t, err, "Failed to create workflows directory")

	// Create a dispatcher workflow that references a non-existent workflow
	dispatcherWorkflow := `---
on: issues
engine: copilot
permissions:
  contents: read
safe-outputs:
  dispatch-workflow:
    workflows:
      - nonexistent
    max: 1
---

# Dispatcher Workflow

This workflow tries to dispatch to a non-existent workflow.
`
	dispatcherFile := filepath.Join(awDir, "dispatcher.md")
	err = os.WriteFile(dispatcherFile, []byte(dispatcherWorkflow), 0644)
	require.NoError(t, err, "Failed to write dispatcher workflow")

	// Change to the aw directory
	oldDir, err := os.Getwd()
	require.NoError(t, err, "Failed to get current directory")
	err = os.Chdir(awDir)
	require.NoError(t, err, "Failed to change directory")
	defer func() { _ = os.Chdir(oldDir) }()

	// Parse the dispatcher workflow
	workflowData, err := compiler.ParseWorkflowFile("dispatcher.md")
	require.NoError(t, err, "Failed to parse workflow")

	// Validate the workflow - should fail because nonexistent workflow is not found
	err = compiler.validateDispatchWorkflow(workflowData, dispatcherFile)
	require.Error(t, err, "Validation should fail - workflow not found")
	assert.Contains(t, err.Error(), "not found", "Error should mention workflow not found")
	assert.Contains(t, err.Error(), "nonexistent", "Error should mention the workflow name")
}

// TestDispatchWorkflowWithoutWorkflowDispatchTrigger tests error handling
// when referenced workflow doesn't support workflow_dispatch
func TestDispatchWorkflowWithoutWorkflowDispatchTrigger(t *testing.T) {
	compiler := NewCompilerWithVersion("1.0.0")

	tmpDir := t.TempDir()
	awDir := filepath.Join(tmpDir, ".github", "aw")
	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")

	err := os.MkdirAll(awDir, 0755)
	require.NoError(t, err, "Failed to create aw directory")
	err = os.MkdirAll(workflowsDir, 0755)
	require.NoError(t, err, "Failed to create workflows directory")

	// Create a workflow WITHOUT workflow_dispatch
	ciWorkflow := `name: CI
on:
  push:
  pull_request:
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - run: echo "Running tests"
`
	ciFile := filepath.Join(workflowsDir, "ci.lock.yml")
	err = os.WriteFile(ciFile, []byte(ciWorkflow), 0644)
	require.NoError(t, err, "Failed to write ci workflow")

	// Create a dispatcher workflow that references ci
	dispatcherWorkflow := `---
on: issues
engine: copilot
permissions:
  contents: read
safe-outputs:
  dispatch-workflow:
    workflows:
      - ci
    max: 1
---

# Dispatcher Workflow

This workflow tries to dispatch to ci workflow.
`
	dispatcherFile := filepath.Join(awDir, "dispatcher.md")
	err = os.WriteFile(dispatcherFile, []byte(dispatcherWorkflow), 0644)
	require.NoError(t, err, "Failed to write dispatcher workflow")

	// Change to the aw directory
	oldDir, err := os.Getwd()
	require.NoError(t, err, "Failed to get current directory")
	err = os.Chdir(awDir)
	require.NoError(t, err, "Failed to change directory")
	defer func() { _ = os.Chdir(oldDir) }()

	// Parse the dispatcher workflow
	workflowData, err := compiler.ParseWorkflowFile("dispatcher.md")
	require.NoError(t, err, "Failed to parse workflow")

	// Validate the workflow - should fail because ci doesn't support workflow_dispatch
	err = compiler.validateDispatchWorkflow(workflowData, dispatcherFile)
	require.Error(t, err, "Validation should fail - workflow doesn't support workflow_dispatch")
	assert.Contains(t, err.Error(), "workflow_dispatch", "Error should mention workflow_dispatch")
}

// TestDispatchWorkflowFileExtensionResolution tests that the correct file extension
// (.lock.yml or .yml) is stored in the WorkflowFiles map
func TestDispatchWorkflowFileExtensionResolution(t *testing.T) {
	compiler := NewCompilerWithVersion("1.0.0")

	tmpDir := t.TempDir()
	awDir := filepath.Join(tmpDir, ".github", "aw")
	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")

	err := os.MkdirAll(awDir, 0755)
	require.NoError(t, err, "Failed to create aw directory")
	err = os.MkdirAll(workflowsDir, 0755)
	require.NoError(t, err, "Failed to create workflows directory")

	// Create a .lock.yml workflow (agentic workflow)
	lockWorkflow := `name: Lock Workflow
on:
  workflow_dispatch:
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - run: echo "Lock workflow"
`
	lockFile := filepath.Join(workflowsDir, "lock-test.lock.yml")
	err = os.WriteFile(lockFile, []byte(lockWorkflow), 0644)
	require.NoError(t, err, "Failed to write lock workflow")

	// Create a .yml workflow (standard GitHub Actions)
	ymlWorkflow := `name: YAML Workflow
on:
  workflow_dispatch:
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - run: echo "YAML workflow"
`
	ymlFile := filepath.Join(workflowsDir, "yml-test.yml")
	err = os.WriteFile(ymlFile, []byte(ymlWorkflow), 0644)
	require.NoError(t, err, "Failed to write yml workflow")

	// Create a dispatcher workflow that references both
	dispatcherWorkflow := `---
on: issues
engine: copilot
permissions:
  contents: read
safe-outputs:
  dispatch-workflow:
    workflows:
      - lock-test
      - yml-test
    max: 2
---

# Dispatcher Workflow

This workflow dispatches to different workflow types.
`
	dispatcherFile := filepath.Join(awDir, "dispatcher.md")
	err = os.WriteFile(dispatcherFile, []byte(dispatcherWorkflow), 0644)
	require.NoError(t, err, "Failed to write dispatcher workflow")

	// Change to the aw directory
	oldDir, err := os.Getwd()
	require.NoError(t, err, "Failed to get current directory")
	err = os.Chdir(awDir)
	require.NoError(t, err, "Failed to change directory")
	defer func() { _ = os.Chdir(oldDir) }()

	// Parse and compile the dispatcher workflow
	workflowData, err := compiler.ParseWorkflowFile("dispatcher.md")
	require.NoError(t, err, "Failed to parse workflow")

	// Populate workflow files (this is what the fix does)
	populateDispatchWorkflowFiles(workflowData, dispatcherFile)

	// Verify WorkflowFiles map has correct extensions after populate
	require.NotNil(t, workflowData.SafeOutputs.DispatchWorkflow.WorkflowFiles,
		"WorkflowFiles should be populated after populateDispatchWorkflowFiles")
	assert.Equal(t, ".lock.yml", workflowData.SafeOutputs.DispatchWorkflow.WorkflowFiles["lock-test"],
		"lock-test should use .lock.yml extension")
	assert.Equal(t, ".yml", workflowData.SafeOutputs.DispatchWorkflow.WorkflowFiles["yml-test"],
		"yml-test should use .yml extension")

	// Generate safe outputs config to verify workflow_files is included
	configJSON := generateSafeOutputsConfig(workflowData)
	require.NotEmpty(t, configJSON, "Config JSON should not be empty")

	// Parse config to verify workflow_files is present
	var config map[string]any
	err = json.Unmarshal([]byte(configJSON), &config)
	require.NoError(t, err, "Config JSON should be valid")

	dispatchWorkflowConfig, ok := config["dispatch_workflow"].(map[string]any)
	require.True(t, ok, "dispatch_workflow should be in config")

	workflowFiles, ok := dispatchWorkflowConfig["workflow_files"].(map[string]any)
	require.True(t, ok, "workflow_files should be in dispatch_workflow config")

	assert.Equal(t, ".lock.yml", workflowFiles["lock-test"],
		"lock-test extension should be in workflow_files")
	assert.Equal(t, ".yml", workflowFiles["yml-test"],
		"yml-test extension should be in workflow_files")
}

// TestDispatchWorkflowValidationWithoutAgenticWorkflowsTool tests that dispatch-workflow
// validation runs even when the agentic-workflows tool is not present
func TestDispatchWorkflowValidationWithoutAgenticWorkflowsTool(t *testing.T) {
	compiler := NewCompilerWithVersion("1.0.0")

	tmpDir := t.TempDir()
	awDir := filepath.Join(tmpDir, ".github", "aw")
	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")

	err := os.MkdirAll(awDir, 0755)
	require.NoError(t, err, "Failed to create aw directory")
	err = os.MkdirAll(workflowsDir, 0755)
	require.NoError(t, err, "Failed to create workflows directory")

	// Create a dispatcher workflow WITHOUT the agentic-workflows tool
	// This workflow references a non-existent workflow
	dispatcherWorkflow := `---
on: issues
engine: copilot
permissions:
  contents: read
safe-outputs:
  dispatch-workflow:
    workflows:
      - nonexistent
    max: 1
---

# Dispatcher Workflow

This workflow tries to dispatch to a non-existent workflow.
No agentic-workflows tool is present.
`
	dispatcherFile := filepath.Join(awDir, "dispatcher.md")
	err = os.WriteFile(dispatcherFile, []byte(dispatcherWorkflow), 0644)
	require.NoError(t, err, "Failed to write dispatcher workflow")

	// Change to the aw directory
	oldDir, err := os.Getwd()
	require.NoError(t, err, "Failed to get current directory")
	err = os.Chdir(tmpDir)
	require.NoError(t, err, "Failed to change directory")
	defer func() { _ = os.Chdir(oldDir) }()

	// Compile the workflow - should fail with validation error
	err = compiler.CompileWorkflow(dispatcherFile)

	// Check that compilation failed due to validation
	require.Error(t, err, "Compilation should fail for non-existent workflow")
	assert.Contains(t, err.Error(), "dispatch-workflow validation failed",
		"Should fail with dispatch-workflow validation error")
	assert.Contains(t, err.Error(), "not found",
		"Error should mention workflow not found")
	assert.Contains(t, err.Error(), "nonexistent",
		"Error should mention the workflow name")
}

// TestDispatchWorkflowMultipleErrors tests that multiple validation errors are aggregated
func TestDispatchWorkflowMultipleErrors(t *testing.T) {
	compiler := NewCompilerWithVersion("1.0.0")
	compiler.failFast = false // Enable error aggregation

	tmpDir := t.TempDir()
	awDir := filepath.Join(tmpDir, ".github", "aw")
	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")

	err := os.MkdirAll(awDir, 0755)
	require.NoError(t, err, "Failed to create aw directory")
	err = os.MkdirAll(workflowsDir, 0755)
	require.NoError(t, err, "Failed to create workflows directory")

	// Create a workflow WITHOUT workflow_dispatch
	ciWorkflow := `name: CI
on:
  push:
  pull_request:
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - run: echo "Running tests"
`
	ciFile := filepath.Join(workflowsDir, "ci.lock.yml")
	err = os.WriteFile(ciFile, []byte(ciWorkflow), 0644)
	require.NoError(t, err, "Failed to write ci workflow")

	// Create dispatcher workflow that references multiple problematic workflows
	dispatcherWorkflow := `---
on: issues
engine: copilot
permissions:
  contents: read
safe-outputs:
  dispatch-workflow:
    workflows:
      - dispatcher  # Self-reference
      - ci          # Missing workflow_dispatch
      - nonexistent # Not found
    max: 3
---

# Dispatcher Workflow

This workflow has multiple validation errors.
`
	dispatcherFile := filepath.Join(awDir, "dispatcher.md")
	err = os.WriteFile(dispatcherFile, []byte(dispatcherWorkflow), 0644)
	require.NoError(t, err, "Failed to write dispatcher workflow")

	// Change to the aw directory
	oldDir, err := os.Getwd()
	require.NoError(t, err, "Failed to get current directory")
	err = os.Chdir(awDir)
	require.NoError(t, err, "Failed to change directory")
	defer func() { _ = os.Chdir(oldDir) }()

	// Parse the dispatcher workflow
	workflowData, err := compiler.ParseWorkflowFile("dispatcher.md")
	require.NoError(t, err, "Failed to parse workflow")

	// Validate the workflow - should return multiple errors
	err = compiler.validateDispatchWorkflow(workflowData, dispatcherFile)
	require.Error(t, err, "Validation should fail with multiple errors")

	// Check that all three errors are present in the aggregated error
	errStr := err.Error()
	assert.Contains(t, errStr, "Found 3 dispatch-workflow errors:", "Should have error count header")
	assert.Contains(t, errStr, "self-reference", "Should contain self-reference error")
	assert.Contains(t, errStr, "dispatcher", "Should mention dispatcher workflow")
	assert.Contains(t, errStr, "workflow_dispatch", "Should contain workflow_dispatch error")
	assert.Contains(t, errStr, "ci", "Should mention ci workflow")
	assert.Contains(t, errStr, "not found", "Should contain not found error")
	assert.Contains(t, errStr, "nonexistent", "Should mention nonexistent workflow")
}

// TestDispatchWorkflowMultipleErrorsFailFast tests fail-fast mode stops at first error
func TestDispatchWorkflowMultipleErrorsFailFast(t *testing.T) {
	compiler := NewCompilerWithVersion("1.0.0")
	compiler.failFast = true // Enable fail-fast mode

	tmpDir := t.TempDir()
	awDir := filepath.Join(tmpDir, ".github", "aw")
	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")

	err := os.MkdirAll(awDir, 0755)
	require.NoError(t, err, "Failed to create aw directory")
	err = os.MkdirAll(workflowsDir, 0755)
	require.NoError(t, err, "Failed to create workflows directory")

	// Create dispatcher workflow with multiple errors
	dispatcherWorkflow := `---
on: issues
engine: copilot
permissions:
  contents: read
safe-outputs:
  dispatch-workflow:
    workflows:
      - dispatcher  # Self-reference (first error)
      - nonexistent # Not found (second error)
    max: 2
---

# Dispatcher Workflow
`
	dispatcherFile := filepath.Join(awDir, "dispatcher.md")
	err = os.WriteFile(dispatcherFile, []byte(dispatcherWorkflow), 0644)
	require.NoError(t, err, "Failed to write dispatcher workflow")

	// Change to the aw directory
	oldDir, err := os.Getwd()
	require.NoError(t, err, "Failed to get current directory")
	err = os.Chdir(awDir)
	require.NoError(t, err, "Failed to change directory")
	defer func() { _ = os.Chdir(oldDir) }()

	// Parse the dispatcher workflow
	workflowData, err := compiler.ParseWorkflowFile("dispatcher.md")
	require.NoError(t, err, "Failed to parse workflow")

	// Validate the workflow - should fail fast with first error only
	err = compiler.validateDispatchWorkflow(workflowData, dispatcherFile)
	require.Error(t, err, "Validation should fail")

	// In fail-fast mode, only the first error should be returned
	errStr := err.Error()
	assert.Contains(t, errStr, "self-reference", "Should contain first error")
	assert.NotContains(t, errStr, "Found 2", "Should not have multiple error header in fail-fast mode")
}

// TestInjectAwContextIntoOnYAML_NoWorkflowDispatch verifies that injectAwContextIntoOnYAML is
// a no-op when there is no workflow_dispatch trigger.
func TestInjectAwContextIntoOnYAML_NoWorkflowDispatch(t *testing.T) {
	onYAML := `"on":
  push:`
	result := injectAwContextIntoOnYAML(onYAML)
	assert.YAMLEq(t, onYAML, result, "Should return unchanged YAML when workflow_dispatch is absent")
	assert.NotContains(t, result, "aw_context", "aw_context should not appear without workflow_dispatch")
}

// TestInjectAwContextIntoOnYAML_BareWorkflowDispatch verifies that aw_context is injected
// into a bare workflow_dispatch trigger (no existing inputs).
func TestInjectAwContextIntoOnYAML_BareWorkflowDispatch(t *testing.T) {
	onYAML := `"on":
  workflow_dispatch:`
	result := injectAwContextIntoOnYAML(onYAML)
	assert.Contains(t, result, "aw_context:", "aw_context input should be injected")
	assert.Contains(t, result, "workflow_dispatch:", "workflow_dispatch section should still be present")
	assert.Contains(t, result, "type: string", "aw_context type should be string")
	assert.Contains(t, result, "required: false", "aw_context should not be required")
}

// TestInjectAwContextIntoOnYAML_ExistingInputs verifies that aw_context is appended without
// disturbing existing workflow_dispatch inputs.
func TestInjectAwContextIntoOnYAML_ExistingInputs(t *testing.T) {
	onYAML := `"on":
  workflow_dispatch:
    inputs:
      environment:
        description: Deployment environment
        required: true
        type: string`
	result := injectAwContextIntoOnYAML(onYAML)
	assert.Contains(t, result, "environment:", "Existing 'environment' input should be preserved")
	assert.Contains(t, result, "aw_context:", "aw_context should be added")
}

// TestInjectAwContextIntoOnYAML_Idempotent verifies that calling injectAwContextIntoOnYAML twice
// does not duplicate the aw_context entry.
func TestInjectAwContextIntoOnYAML_Idempotent(t *testing.T) {
	onYAML := `"on":
  workflow_dispatch:`
	once := injectAwContextIntoOnYAML(onYAML)
	twice := injectAwContextIntoOnYAML(once)
	assert.Equal(t, once, twice, "Second injection should be a no-op")
	assert.Equal(t, 1, strings.Count(twice, "aw_context:"), "aw_context should appear exactly once")
}

// TestInjectAwContextIntoOnYAML_WithOtherTriggers verifies that aw_context is injected
// even when other triggers are present alongside workflow_dispatch.
func TestInjectAwContextIntoOnYAML_WithOtherTriggers(t *testing.T) {
	onYAML := `"on":
  pull_request:
    types:
    - labeled
  workflow_dispatch:
    inputs:
      item_number:
        description: The number of the issue
        required: false
        default: ""
        type: string`
	result := injectAwContextIntoOnYAML(onYAML)
	assert.Contains(t, result, "pull_request:", "pull_request trigger should be preserved")
	assert.Contains(t, result, "item_number:", "item_number input should be preserved")
	assert.Contains(t, result, "aw_context:", "aw_context should be added alongside item_number")
}

// TestInjectAwContextIntoOnYAML_CompiledOutput verifies that a workflow with workflow_dispatch
// trigger produces aw_context in the compiled lock file on section.
func TestInjectAwContextIntoOnYAML_CompiledOutput(t *testing.T) {
	compiler := NewCompilerWithVersion("1.0.0")

	tmpDir := t.TempDir()
	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
	require.NoError(t, os.MkdirAll(workflowsDir, 0755))

	workflowMD := `---
on:
  workflow_dispatch:
    inputs:
      env:
        description: "Target environment"
        required: false
        type: string
engine: copilot
permissions:
  contents: read
---

# Test Workflow
Run tests.
`
	mdFile := filepath.Join(workflowsDir, "test.md")
	require.NoError(t, os.WriteFile(mdFile, []byte(workflowMD), 0644))

	err := compiler.CompileWorkflow(mdFile)
	require.NoError(t, err, "Compilation should succeed")

	lockFile := filepath.Join(workflowsDir, "test.lock.yml")
	content, err := os.ReadFile(lockFile)
	require.NoError(t, err, "Lock file should be generated")

	lockStr := string(content)
	assert.Contains(t, lockStr, "aw_context:", "Compiled output should include aw_context input")
	assert.Contains(t, lockStr, "Internal", "aw_context description should mention internal usage")
	// Existing user input should still be present
	assert.Contains(t, lockStr, "env:", "Existing user input should be preserved in compiled output")
}
