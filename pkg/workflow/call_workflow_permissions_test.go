//go:build !integration

package workflow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestExtractJobPermissionsFromParsedWorkflow_NoJobs tests empty workflow map
func TestExtractJobPermissionsFromParsedWorkflow_NoJobs(t *testing.T) {
	perms := extractJobPermissionsFromParsedWorkflow(map[string]any{})
	assert.Empty(t, perms.RenderToYAML(), "Should return empty permissions when no jobs present")
}

// TestExtractJobPermissionsFromParsedWorkflow_SingleJob tests a single job with permissions
func TestExtractJobPermissionsFromParsedWorkflow_SingleJob(t *testing.T) {
	workflow := map[string]any{
		"jobs": map[string]any{
			"agent": map[string]any{
				"permissions": map[string]any{
					"contents":      "read",
					"issues":        "read",
					"pull-requests": "read",
					"actions":       "read",
				},
			},
		},
	}

	perms := extractJobPermissionsFromParsedWorkflow(workflow)
	rendered := perms.RenderToYAML()
	assert.Contains(t, rendered, "contents: read", "Should include contents: read")
	assert.Contains(t, rendered, "issues: read", "Should include issues: read")
	assert.Contains(t, rendered, "pull-requests: read", "Should include pull-requests: read")
	assert.Contains(t, rendered, "actions: read", "Should include actions: read")
}

// TestExtractJobPermissionsFromParsedWorkflow_MultipleJobs tests merging permissions from multiple jobs
func TestExtractJobPermissionsFromParsedWorkflow_MultipleJobs(t *testing.T) {
	workflow := map[string]any{
		"jobs": map[string]any{
			"activation": map[string]any{
				"permissions": map[string]any{
					"contents": "read",
				},
			},
			"agent": map[string]any{
				"permissions": map[string]any{
					"actions":       "read",
					"contents":      "read",
					"issues":        "read",
					"pull-requests": "read",
				},
			},
			"safe_outputs": map[string]any{
				"permissions": map[string]any{
					"contents":      "write",
					"issues":        "write",
					"pull-requests": "write",
				},
			},
		},
	}

	perms := extractJobPermissionsFromParsedWorkflow(workflow)
	rendered := perms.RenderToYAML()

	// Write should win over read for contents
	assert.Contains(t, rendered, "contents: write", "Write should take precedence over read for contents")
	assert.Contains(t, rendered, "issues: write", "Write should take precedence for issues")
	assert.Contains(t, rendered, "pull-requests: write", "Write should take precedence for pull-requests")
	assert.Contains(t, rendered, "actions: read", "Should include actions: read from agent job")
}

// TestExtractJobPermissionsFromParsedWorkflow_NoPermissionsOnJobs tests jobs with no permissions block
func TestExtractJobPermissionsFromParsedWorkflow_NoPermissionsOnJobs(t *testing.T) {
	workflow := map[string]any{
		"jobs": map[string]any{
			"build": map[string]any{
				"runs-on": "ubuntu-latest",
			},
		},
	}

	perms := extractJobPermissionsFromParsedWorkflow(workflow)
	assert.Empty(t, perms.RenderToYAML(), "Should return empty when jobs have no permissions")
}

// TestExtractCallWorkflowPermissions_FromLockYML tests extracting permissions from a .lock.yml file
func TestExtractCallWorkflowPermissions_FromLockYML(t *testing.T) {
	tmpDir := t.TempDir()
	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
	require.NoError(t, os.MkdirAll(workflowsDir, 0755), "Failed to create workflows directory")

	workerContent := `name: Worker A
on:
  workflow_call: {}
jobs:
  activation:
    permissions:
      contents: read
    runs-on: ubuntu-latest
    steps:
      - run: echo "activation"
  agent:
    permissions:
      actions: read
      contents: read
      issues: read
      pull-requests: read
    runs-on: ubuntu-latest
    steps:
      - run: echo "agent"
  safe_outputs:
    permissions:
      contents: write
      issues: write
      pull-requests: write
    runs-on: ubuntu-latest
    steps:
      - run: echo "safe_outputs"
`
	require.NoError(t, os.WriteFile(filepath.Join(workflowsDir, "worker-a.lock.yml"), []byte(workerContent), 0644), "Failed to write worker-a.lock.yml")

	markdownPath := filepath.Join(workflowsDir, "gateway.md")

	perms, err := extractCallWorkflowPermissions("worker-a", markdownPath)
	require.NoError(t, err, "Should extract permissions without error")
	require.NotNil(t, perms, "Should return non-nil permissions")

	rendered := perms.RenderToYAML()
	assert.Contains(t, rendered, "contents: write", "Should include contents: write (merged from safe_outputs)")
	assert.Contains(t, rendered, "issues: write", "Should include issues: write")
	assert.Contains(t, rendered, "pull-requests: write", "Should include pull-requests: write")
	assert.Contains(t, rendered, "actions: read", "Should include actions: read from agent")
}

// TestExtractCallWorkflowPermissions_FromYML tests extracting permissions from a .yml file
func TestExtractCallWorkflowPermissions_FromYML(t *testing.T) {
	tmpDir := t.TempDir()
	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
	require.NoError(t, os.MkdirAll(workflowsDir, 0755), "Failed to create workflows directory")

	workerContent := `name: Worker B
on:
  workflow_call: {}
jobs:
  work:
    permissions:
      contents: read
      issues: write
    runs-on: ubuntu-latest
    steps:
      - run: echo "work"
`
	require.NoError(t, os.WriteFile(filepath.Join(workflowsDir, "worker-b.yml"), []byte(workerContent), 0644), "Failed to write worker-b.yml")

	markdownPath := filepath.Join(workflowsDir, "gateway.md")

	perms, err := extractCallWorkflowPermissions("worker-b", markdownPath)
	require.NoError(t, err, "Should extract permissions without error")
	require.NotNil(t, perms, "Should return non-nil permissions")

	rendered := perms.RenderToYAML()
	assert.Contains(t, rendered, "contents: read", "Should include contents: read")
	assert.Contains(t, rendered, "issues: write", "Should include issues: write")
}

// TestExtractCallWorkflowPermissions_FromMD tests extracting permissions from a .md source file
func TestExtractCallWorkflowPermissions_FromMD(t *testing.T) {
	tmpDir := t.TempDir()
	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
	require.NoError(t, os.MkdirAll(workflowsDir, 0755), "Failed to create workflows directory")

	// A same-batch .md source with frontmatter permissions
	mdContent := `---
on:
  workflow_call: {}
engine: copilot
permissions:
  contents: read
  issues: write
  pull-requests: write
---

# Worker C
`
	require.NoError(t, os.WriteFile(filepath.Join(workflowsDir, "worker-c.md"), []byte(mdContent), 0644), "Failed to write worker-c.md")

	markdownPath := filepath.Join(workflowsDir, "gateway.md")

	perms, err := extractCallWorkflowPermissions("worker-c", markdownPath)
	require.NoError(t, err, "Should extract permissions from .md without error")
	require.NotNil(t, perms, "Should return non-nil permissions")

	rendered := perms.RenderToYAML()
	assert.Contains(t, rendered, "contents: read", "Should include contents: read from frontmatter")
	assert.Contains(t, rendered, "issues: write", "Should include issues: write from frontmatter")
	assert.Contains(t, rendered, "pull-requests: write", "Should include pull-requests: write from frontmatter")
}

// TestExtractCallWorkflowPermissions_FileNotFound tests that nil is returned when no file exists
func TestExtractCallWorkflowPermissions_FileNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
	require.NoError(t, os.MkdirAll(workflowsDir, 0755), "Failed to create workflows directory")

	markdownPath := filepath.Join(workflowsDir, "gateway.md")

	perms, err := extractCallWorkflowPermissions("nonexistent-worker", markdownPath)
	require.NoError(t, err, "Should not error when file not found")
	assert.Nil(t, perms, "Should return nil when no file exists")
}

// TestBuildCallWorkflowJobs_SetsPermissionsFromLockYML tests that call-workflow jobs
// include permissions extracted from the worker's .lock.yml file
func TestBuildCallWorkflowJobs_SetsPermissionsFromLockYML(t *testing.T) {
	compiler := NewCompilerWithVersion("1.0.0")

	tmpDir := t.TempDir()
	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
	require.NoError(t, os.MkdirAll(workflowsDir, 0755), "Failed to create workflows directory")

	// Create worker with permissions
	workerContent := `name: Worker
on:
  workflow_call: {}
jobs:
  agent:
    permissions:
      contents: read
      issues: read
      pull-requests: read
    runs-on: ubuntu-latest
    steps:
      - run: echo "agent"
  safe_outputs:
    permissions:
      contents: write
      issues: write
      pull-requests: write
    runs-on: ubuntu-latest
    steps:
      - run: echo "safe_outputs"
`
	require.NoError(t, os.WriteFile(filepath.Join(workflowsDir, "worker-docs.lock.yml"), []byte(workerContent), 0644), "Failed to write worker-docs.lock.yml")

	markdownPath := filepath.Join(workflowsDir, "gateway.md")

	workflowData := &WorkflowData{
		SafeOutputs: &SafeOutputsConfig{
			CallWorkflow: &CallWorkflowConfig{
				BaseSafeOutputConfig: BaseSafeOutputConfig{Max: strPtr("1")},
				Workflows:            []string{"worker-docs"},
				WorkflowFiles: map[string]string{
					"worker-docs": "./.github/workflows/worker-docs.lock.yml",
				},
			},
		},
	}

	jobNames, err := compiler.buildCallWorkflowJobs(workflowData, markdownPath)
	require.NoError(t, err, "Should build call-workflow jobs without error")
	assert.Equal(t, []string{"call-worker-docs"}, jobNames, "Should generate the job")

	job, exists := compiler.jobManager.GetJob("call-worker-docs")
	require.True(t, exists, "Job should exist in job manager")
	assert.NotEmpty(t, job.Permissions, "Job should have permissions set")
	assert.Contains(t, job.Permissions, "contents: write", "Permissions should include contents: write")
	assert.Contains(t, job.Permissions, "issues: write", "Permissions should include issues: write")
	assert.Contains(t, job.Permissions, "pull-requests: write", "Permissions should include pull-requests: write")
}

// TestBuildCallWorkflowJobs_SetsPermissionsFromMD tests that call-workflow jobs
// include permissions from .md frontmatter for same-batch compilation targets
func TestBuildCallWorkflowJobs_SetsPermissionsFromMD(t *testing.T) {
	compiler := NewCompilerWithVersion("1.0.0")

	tmpDir := t.TempDir()
	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
	require.NoError(t, os.MkdirAll(workflowsDir, 0755), "Failed to create workflows directory")

	// Create same-batch .md worker (no .lock.yml exists yet)
	mdContent := `---
on:
  workflow_call: {}
engine: copilot
permissions:
  contents: read
  issues: write
---

# Worker E
`
	require.NoError(t, os.WriteFile(filepath.Join(workflowsDir, "worker-e.md"), []byte(mdContent), 0644), "Failed to write worker-e.md")

	markdownPath := filepath.Join(workflowsDir, "gateway.md")

	workflowData := &WorkflowData{
		SafeOutputs: &SafeOutputsConfig{
			CallWorkflow: &CallWorkflowConfig{
				BaseSafeOutputConfig: BaseSafeOutputConfig{Max: strPtr("1")},
				Workflows:            []string{"worker-e"},
				WorkflowFiles: map[string]string{
					"worker-e": "./.github/workflows/worker-e.lock.yml",
				},
			},
		},
	}

	jobNames, err := compiler.buildCallWorkflowJobs(workflowData, markdownPath)
	require.NoError(t, err, "Should build call-workflow jobs without error")
	assert.Equal(t, []string{"call-worker-e"}, jobNames, "Should generate the job")

	job, exists := compiler.jobManager.GetJob("call-worker-e")
	require.True(t, exists, "Job should exist in job manager")
	assert.NotEmpty(t, job.Permissions, "Job should have permissions from .md frontmatter")
	assert.Contains(t, job.Permissions, "contents: read", "Permissions should include contents: read")
	assert.Contains(t, job.Permissions, "issues: write", "Permissions should include issues: write")
}

// TestBuildCallWorkflowJobs_NoPermissionsWhenWorkerHasNone tests that call-workflow
// jobs omit the permissions block when the worker's jobs have no permissions
func TestBuildCallWorkflowJobs_NoPermissionsWhenWorkerHasNone(t *testing.T) {
	compiler := NewCompilerWithVersion("1.0.0")

	tmpDir := t.TempDir()
	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
	require.NoError(t, os.MkdirAll(workflowsDir, 0755), "Failed to create workflows directory")

	// Worker with no job-level permissions
	workerContent := `name: Worker F
on:
  workflow_call: {}
jobs:
  work:
    runs-on: ubuntu-latest
    steps:
      - run: echo "hello"
`
	require.NoError(t, os.WriteFile(filepath.Join(workflowsDir, "worker-f.lock.yml"), []byte(workerContent), 0644), "Failed to write worker-f.lock.yml")

	markdownPath := filepath.Join(workflowsDir, "gateway.md")

	workflowData := &WorkflowData{
		SafeOutputs: &SafeOutputsConfig{
			CallWorkflow: &CallWorkflowConfig{
				BaseSafeOutputConfig: BaseSafeOutputConfig{Max: strPtr("1")},
				Workflows:            []string{"worker-f"},
				WorkflowFiles: map[string]string{
					"worker-f": "./.github/workflows/worker-f.lock.yml",
				},
			},
		},
	}

	jobNames, err := compiler.buildCallWorkflowJobs(workflowData, markdownPath)
	require.NoError(t, err, "Should build call-workflow jobs without error")
	assert.Equal(t, []string{"call-worker-f"}, jobNames, "Should generate the job")

	job, exists := compiler.jobManager.GetJob("call-worker-f")
	require.True(t, exists, "Job should exist in job manager")
	assert.Empty(t, job.Permissions, "Job should have no permissions when worker has none")
}

// TestCallWorkflowJobYAMLOutput_WithPermissions tests the YAML output of a call-workflow
// job includes the permissions block derived from the worker's .lock.yml
func TestCallWorkflowJobYAMLOutput_WithPermissions(t *testing.T) {
	compiler := NewCompilerWithVersion("1.0.0")

	tmpDir := t.TempDir()
	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
	require.NoError(t, os.MkdirAll(workflowsDir, 0755), "Failed to create workflows directory")

	workerContent := `name: Worker
on:
  workflow_call: {}
jobs:
  agent:
    permissions:
      contents: read
      issues: read
    runs-on: ubuntu-latest
    steps:
      - run: echo "agent"
`
	require.NoError(t, os.WriteFile(filepath.Join(workflowsDir, "worker-a.lock.yml"), []byte(workerContent), 0644), "Failed to write worker-a.lock.yml")

	markdownPath := filepath.Join(workflowsDir, "gateway.md")

	workflowData := &WorkflowData{
		SafeOutputs: &SafeOutputsConfig{
			CallWorkflow: &CallWorkflowConfig{
				BaseSafeOutputConfig: BaseSafeOutputConfig{Max: strPtr("1")},
				Workflows:            []string{"worker-a"},
				WorkflowFiles: map[string]string{
					"worker-a": "./.github/workflows/worker-a.lock.yml",
				},
			},
		},
	}

	_, err := compiler.buildCallWorkflowJobs(workflowData, markdownPath)
	require.NoError(t, err, "Should build jobs without error")

	yamlOutput := compiler.jobManager.RenderToYAML()

	assert.Contains(t, yamlOutput, "uses: ./.github/workflows/worker-a.lock.yml", "Should contain uses directive")
	assert.Contains(t, yamlOutput, "secrets: inherit", "Should inherit secrets")
	assert.Contains(t, yamlOutput, "permissions:", "Should include permissions block")
	assert.Contains(t, yamlOutput, "contents: read", "Should include contents: read")
	assert.Contains(t, yamlOutput, "issues: read", "Should include issues: read")

	// Verify permissions appear before uses in the YAML (job-level ordering)
	permIdx := strings.Index(yamlOutput, "permissions:")
	usesIdx := strings.Index(yamlOutput, "uses:")
	assert.Less(t, permIdx, usesIdx, "permissions: should appear before uses: in job YAML")
}
