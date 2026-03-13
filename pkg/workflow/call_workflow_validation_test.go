//go:build !integration

package workflow

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestValidateCallWorkflow_EmptyList tests that empty workflow list returns an error
func TestValidateCallWorkflow_EmptyList(t *testing.T) {
	compiler := NewCompilerWithVersion("1.0.0")

	tmpDir := t.TempDir()
	awDir := filepath.Join(tmpDir, ".github", "aw")
	err := os.MkdirAll(awDir, 0755)
	require.NoError(t, err, "Failed to create aw directory")

	gatewayFile := filepath.Join(awDir, "gateway.md")

	workflowData := &WorkflowData{
		SafeOutputs: &SafeOutputsConfig{
			CallWorkflow: &CallWorkflowConfig{
				BaseSafeOutputConfig: BaseSafeOutputConfig{
					Max: strPtr("1"),
				},
				Workflows: []string{},
			},
		},
	}

	err = compiler.validateCallWorkflow(workflowData, gatewayFile)
	require.Error(t, err, "Validation should fail for empty workflows list")
	assert.Contains(t, err.Error(), "must specify at least one workflow", "Should mention the requirement")
}

// TestValidateCallWorkflow_NoConfig tests that nil config passes validation
func TestValidateCallWorkflow_NoConfig(t *testing.T) {
	compiler := NewCompilerWithVersion("1.0.0")

	workflowData := &WorkflowData{
		SafeOutputs: &SafeOutputsConfig{},
	}

	err := compiler.validateCallWorkflow(workflowData, "gateway.md")
	assert.NoError(t, err, "Should pass when call-workflow is not configured")
}

// TestValidateCallWorkflow_SelfReference tests that self-reference is rejected
func TestValidateCallWorkflow_SelfReference(t *testing.T) {
	compiler := NewCompilerWithVersion("1.0.0")

	tmpDir := t.TempDir()
	awDir := filepath.Join(tmpDir, ".github", "aw")
	err := os.MkdirAll(awDir, 0755)
	require.NoError(t, err, "Failed to create aw directory")

	gatewayFile := filepath.Join(awDir, "gateway.md")

	workflowData := &WorkflowData{
		SafeOutputs: &SafeOutputsConfig{
			CallWorkflow: &CallWorkflowConfig{
				BaseSafeOutputConfig: BaseSafeOutputConfig{
					Max: strPtr("1"),
				},
				Workflows: []string{"gateway"}, // Self-reference
			},
		},
	}

	err = compiler.validateCallWorkflow(workflowData, gatewayFile)
	require.Error(t, err, "Self-reference should fail validation")
	assert.Contains(t, err.Error(), "self-reference not allowed", "Should mention self-reference")
	assert.Contains(t, err.Error(), "gateway", "Should mention the workflow name")
}

// TestValidateCallWorkflow_WorkflowNotFound tests that a missing workflow fails validation
func TestValidateCallWorkflow_WorkflowNotFound(t *testing.T) {
	compiler := NewCompilerWithVersion("1.0.0")

	tmpDir := t.TempDir()
	awDir := filepath.Join(tmpDir, ".github", "aw")
	err := os.MkdirAll(awDir, 0755)
	require.NoError(t, err, "Failed to create aw directory")

	gatewayFile := filepath.Join(awDir, "gateway.md")

	workflowData := &WorkflowData{
		SafeOutputs: &SafeOutputsConfig{
			CallWorkflow: &CallWorkflowConfig{
				BaseSafeOutputConfig: BaseSafeOutputConfig{
					Max: strPtr("1"),
				},
				Workflows: []string{"nonexistent-worker"},
			},
		},
	}

	err = compiler.validateCallWorkflow(workflowData, gatewayFile)
	require.Error(t, err, "Missing workflow should fail validation")
	assert.Contains(t, err.Error(), "not found", "Should mention workflow not found")
	assert.Contains(t, err.Error(), "nonexistent-worker", "Should mention the workflow name")
}

// TestValidateCallWorkflow_WorkflowWithoutWorkflowCall tests that a workflow missing
// workflow_call trigger fails validation
func TestValidateCallWorkflow_WorkflowWithoutWorkflowCall(t *testing.T) {
	compiler := NewCompilerWithVersion("1.0.0")

	tmpDir := t.TempDir()
	awDir := filepath.Join(tmpDir, ".github", "aw")
	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
	err := os.MkdirAll(awDir, 0755)
	require.NoError(t, err, "Failed to create aw directory")
	err = os.MkdirAll(workflowsDir, 0755)
	require.NoError(t, err, "Failed to create workflows directory")

	// Create a worker without workflow_call trigger
	workerContent := `name: Worker
on:
  push:
  workflow_dispatch:
jobs:
  work:
    runs-on: ubuntu-latest
    steps:
      - run: echo "Working"
`
	err = os.WriteFile(filepath.Join(workflowsDir, "worker-a.lock.yml"), []byte(workerContent), 0644)
	require.NoError(t, err, "Failed to write worker file")

	gatewayFile := filepath.Join(awDir, "gateway.md")

	workflowData := &WorkflowData{
		SafeOutputs: &SafeOutputsConfig{
			CallWorkflow: &CallWorkflowConfig{
				BaseSafeOutputConfig: BaseSafeOutputConfig{
					Max: strPtr("1"),
				},
				Workflows: []string{"worker-a"},
			},
		},
	}

	err = compiler.validateCallWorkflow(workflowData, gatewayFile)
	require.Error(t, err, "Worker without workflow_call should fail validation")
	assert.Contains(t, err.Error(), "workflow_call", "Should mention workflow_call trigger")
	assert.Contains(t, err.Error(), "worker-a", "Should mention the workflow name")
}

// TestValidateCallWorkflow_ValidWorkflow tests that a valid worker passes validation
func TestValidateCallWorkflow_ValidWorkflow(t *testing.T) {
	compiler := NewCompilerWithVersion("1.0.0")

	tmpDir := t.TempDir()
	awDir := filepath.Join(tmpDir, ".github", "aw")
	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
	err := os.MkdirAll(awDir, 0755)
	require.NoError(t, err, "Failed to create aw directory")
	err = os.MkdirAll(workflowsDir, 0755)
	require.NoError(t, err, "Failed to create workflows directory")

	// Create a valid worker with workflow_call trigger
	workerContent := `name: Worker A
on:
  workflow_call:
    inputs:
      payload:
        type: string
        required: false
jobs:
  work:
    runs-on: ubuntu-latest
    steps:
      - run: echo "Working"
`
	err = os.WriteFile(filepath.Join(workflowsDir, "worker-a.lock.yml"), []byte(workerContent), 0644)
	require.NoError(t, err, "Failed to write worker file")

	gatewayFile := filepath.Join(awDir, "gateway.md")

	workflowData := &WorkflowData{
		SafeOutputs: &SafeOutputsConfig{
			CallWorkflow: &CallWorkflowConfig{
				BaseSafeOutputConfig: BaseSafeOutputConfig{
					Max: strPtr("1"),
				},
				Workflows: []string{"worker-a"},
			},
		},
	}

	err = compiler.validateCallWorkflow(workflowData, gatewayFile)
	assert.NoError(t, err, "Valid worker with workflow_call should pass validation")
}

// TestValidateCallWorkflow_MDSourceWithWorkflowCall tests that a .md source with workflow_call
// trigger passes validation (same-batch compilation target)
func TestValidateCallWorkflow_MDSourceWithWorkflowCall(t *testing.T) {
	compiler := NewCompilerWithVersion("1.0.0")

	tmpDir := t.TempDir()
	awDir := filepath.Join(tmpDir, ".github", "aw")
	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
	err := os.MkdirAll(awDir, 0755)
	require.NoError(t, err, "Failed to create aw directory")
	err = os.MkdirAll(workflowsDir, 0755)
	require.NoError(t, err, "Failed to create workflows directory")

	// Create a worker .md source with workflow_call trigger
	workerMD := `---
on:
  workflow_call:
    inputs:
      payload:
        type: string
engine: copilot
---

# Worker A

This is a worker workflow.
`
	err = os.WriteFile(filepath.Join(workflowsDir, "worker-a.md"), []byte(workerMD), 0644)
	require.NoError(t, err, "Failed to write worker .md file")

	gatewayFile := filepath.Join(awDir, "gateway.md")

	workflowData := &WorkflowData{
		SafeOutputs: &SafeOutputsConfig{
			CallWorkflow: &CallWorkflowConfig{
				BaseSafeOutputConfig: BaseSafeOutputConfig{
					Max: strPtr("1"),
				},
				Workflows: []string{"worker-a"},
			},
		},
	}

	err = compiler.validateCallWorkflow(workflowData, gatewayFile)
	assert.NoError(t, err, ".md source with workflow_call should pass validation")
}
