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
	compiler := NewCompiler(WithVersion("1.0.0"))

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
	compiler := NewCompiler(WithVersion("1.0.0"))

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
	compiler := NewCompiler(WithVersion("1.0.0"))

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
	compiler := NewCompiler(WithVersion("1.0.0"))

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
	require.NotEqual(t, -1, permIdx, "permissions: should be present in YAML output")
	require.NotEqual(t, -1, usesIdx, "uses: should be present in YAML output")
	assert.Less(t, permIdx, usesIdx, "permissions: should appear before uses: in job YAML")
}

// TestExtractCallWorkflowPermissions_LockYMLPriorityOverYML tests that .lock.yml takes
// priority over .yml when both exist
func TestExtractCallWorkflowPermissions_LockYMLPriorityOverYML(t *testing.T) {
	tmpDir := t.TempDir()
	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
	require.NoError(t, os.MkdirAll(workflowsDir, 0755), "Failed to create workflows directory")

	// .lock.yml has contents: write
	lockContent := `name: Worker Lock
on:
  workflow_call: {}
jobs:
  work:
    permissions:
      contents: write
    runs-on: ubuntu-latest
    steps:
      - run: echo "lock"
`
	// .yml has contents: read (should be ignored when .lock.yml exists)
	ymlContent := `name: Worker YML
on:
  workflow_call: {}
jobs:
  work:
    permissions:
      contents: read
    runs-on: ubuntu-latest
    steps:
      - run: echo "yml"
`
	require.NoError(t, os.WriteFile(filepath.Join(workflowsDir, "worker-priority.lock.yml"), []byte(lockContent), 0644), "Failed to write worker-priority.lock.yml")
	require.NoError(t, os.WriteFile(filepath.Join(workflowsDir, "worker-priority.yml"), []byte(ymlContent), 0644), "Failed to write worker-priority.yml")

	markdownPath := filepath.Join(workflowsDir, "gateway.md")

	perms, err := extractCallWorkflowPermissions("worker-priority", markdownPath)
	require.NoError(t, err, "Should extract permissions without error")
	require.NotNil(t, perms, "Should return non-nil permissions")

	rendered := perms.RenderToYAML()
	// Should use .lock.yml (contents: write), not .yml (contents: read)
	assert.Contains(t, rendered, "contents: write", "Should use .lock.yml permissions, not .yml")
}

// TestCallWorkflowPermissions_EndToEnd tests full gateway compilation with permissioned workers —
// the generated lock file must include job-level permissions blocks on every call-* job.
func TestCallWorkflowPermissions_EndToEnd(t *testing.T) {
	compiler := NewCompiler(WithVersion("1.0.0"))

	tmpDir := t.TempDir()
	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
	require.NoError(t, os.MkdirAll(workflowsDir, 0755), "Failed to create workflows directory")

	// Worker A: needs read permissions
	workerA := `name: Worker A
on:
  workflow_call:
    inputs:
      payload:
        type: string
        required: false
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
	require.NoError(t, os.WriteFile(filepath.Join(workflowsDir, "worker-a.lock.yml"), []byte(workerA), 0644), "Failed to write worker-a.lock.yml")

	// Worker B: only needs issues: write
	workerB := `name: Worker B
on:
  workflow_call:
    inputs:
      payload:
        type: string
        required: false
jobs:
  work:
    permissions:
      issues: write
    runs-on: ubuntu-latest
    steps:
      - run: echo "work"
`
	require.NoError(t, os.WriteFile(filepath.Join(workflowsDir, "worker-b.lock.yml"), []byte(workerB), 0644), "Failed to write worker-b.lock.yml")

	// Gateway markdown: calls both workers
	gatewayMD := `---
on:
  issues:
    types: [opened]
engine: copilot
permissions:
  contents: read
safe-outputs:
  add-comment:
    max: 1
  call-workflow:
    workflows:
      - worker-a
      - worker-b
    max: 1
---

# Gateway

Analyse the issue and determine which worker to run.
`
	gatewayFile := filepath.Join(workflowsDir, "gateway.md")
	require.NoError(t, os.WriteFile(gatewayFile, []byte(gatewayMD), 0644), "Failed to write gateway.md")

	require.NoError(t, compiler.CompileWorkflow(gatewayFile), "Should compile without error")

	lockFile := gatewayFile[:len(gatewayFile)-len(".md")] + ".lock.yml"
	lockContentBytes, err := os.ReadFile(lockFile)
	require.NoError(t, err, "Should read the generated lock file")
	yamlOutput := string(lockContentBytes)

	// Verify call-worker-a job exists and has permissions
	assert.Contains(t, yamlOutput, "call-worker-a:", "Should contain call-worker-a job")
	assert.Contains(t, yamlOutput, "call-worker-b:", "Should contain call-worker-b job")

	// Both call-* jobs must include a permissions: block
	assert.Contains(t, yamlOutput, "permissions:", "Generated YAML should include at least one permissions block")

	// Locate the call-worker-a section and verify its permissions block
	callAStart := strings.Index(yamlOutput, "call-worker-a:")
	callBStart := strings.Index(yamlOutput, "call-worker-b:")
	require.NotEqual(t, -1, callAStart, "call-worker-a: must appear in generated YAML")
	require.NotEqual(t, -1, callBStart, "call-worker-b: must appear in generated YAML")

	// Extract the YAML section for call-worker-a (up to the next top-level job or end of file)
	var callAEnd int
	if callBStart > callAStart {
		callAEnd = callBStart
	} else {
		callAEnd = len(yamlOutput)
	}
	callASection := yamlOutput[callAStart:callAEnd]
	assert.Contains(t, callASection, "permissions:", "call-worker-a job must have permissions block")
	// Worker A has contents: write (from safe_outputs job — write wins over read)
	assert.Contains(t, callASection, "contents: write", "call-worker-a permissions should include contents: write")

	// Extract the YAML section for call-worker-b (union from its single job)
	callBSection := yamlOutput[callBStart:]
	assert.Contains(t, callBSection, "permissions:", "call-worker-b job must have permissions block")
	assert.Contains(t, callBSection, "issues: write", "call-worker-b permissions should include issues: write")
}

// TestCallWorkflowPermissions_EndToEnd_YMLWorker tests that a worker referenced via a .yml
// file (not .lock.yml) also gets its permissions propagated in the generated call-* job.
func TestCallWorkflowPermissions_EndToEnd_YMLWorker(t *testing.T) {
	compiler := NewCompiler(WithVersion("1.0.0"))

	tmpDir := t.TempDir()
	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
	require.NoError(t, os.MkdirAll(workflowsDir, 0755), "Failed to create workflows directory")

	// Worker delivered as a plain .yml (no .lock.yml counterpart)
	workerYML := `name: Worker YML
on:
  workflow_call:
    inputs:
      payload:
        type: string
        required: false
jobs:
  work:
    permissions:
      contents: read
      pull-requests: write
    runs-on: ubuntu-latest
    steps:
      - run: echo "work"
`
	require.NoError(t, os.WriteFile(filepath.Join(workflowsDir, "worker-plain.yml"), []byte(workerYML), 0644), "Failed to write worker-plain.yml")

	gatewayMD := `---
on:
  issues:
    types: [opened]
engine: copilot
permissions:
  contents: read
safe-outputs:
  add-comment:
    max: 1
  call-workflow:
    workflows:
      - worker-plain
    max: 1
---

# Gateway

Pick the right worker.
`
	gatewayFile := filepath.Join(workflowsDir, "gateway.md")
	require.NoError(t, os.WriteFile(gatewayFile, []byte(gatewayMD), 0644), "Failed to write gateway.md")

	require.NoError(t, compiler.CompileWorkflow(gatewayFile), "Should compile without error")

	lockFile := gatewayFile[:len(gatewayFile)-len(".md")] + ".lock.yml"
	lockContentBytes, err := os.ReadFile(lockFile)
	require.NoError(t, err, "Should read the generated lock file")
	yamlOutput := string(lockContentBytes)

	callStart := strings.Index(yamlOutput, "call-worker-plain:")
	require.NotEqual(t, -1, callStart, "call-worker-plain: must appear in generated YAML")

	callSection := yamlOutput[callStart:]
	assert.Contains(t, callSection, "permissions:", "call-worker-plain job must have permissions block")
	assert.Contains(t, callSection, "contents: read", "Permissions should include contents: read")
	assert.Contains(t, callSection, "pull-requests: write", "Permissions should include pull-requests: write")
}
