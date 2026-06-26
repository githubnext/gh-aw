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

func TestExtractCallWorkflowPermissionImport_MDWithoutPermissionsReturnsNil(t *testing.T) {
	tmpDir := t.TempDir()
	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
	require.NoError(t, os.MkdirAll(workflowsDir, 0755), "Failed to create workflows directory")

	mdContent := `---
on:
  workflow_call: {}
engine: copilot
---

# Worker Without Permissions
`
	require.NoError(t, os.WriteFile(filepath.Join(workflowsDir, "worker-no-perms.md"), []byte(mdContent), 0644), "Failed to write worker-no-perms.md")

	markdownPath := filepath.Join(workflowsDir, "gateway.md")

	imported, err := extractCallWorkflowPermissionImport("worker-no-perms", markdownPath)
	require.NoError(t, err, "Should not error when markdown worker has no permissions")
	assert.Nil(t, imported, "Should treat markdown workers with no permissions like other missing-import cases")
}

func TestExtractCallWorkflowPermissionImport_TracksReviewSource(t *testing.T) {
	t.Setenv("GH_AW_WORKFLOWS_DIR", "")

	tmpDir := t.TempDir()
	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
	require.NoError(t, os.MkdirAll(workflowsDir, 0755), "Failed to create workflows directory")

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
	require.NoError(t, os.WriteFile(filepath.Join(workflowsDir, "worker-review.lock.yml"), []byte(lockContent), 0644), "Failed to write worker-review.lock.yml")

	markdownPath := filepath.Join(workflowsDir, "gateway.md")

	imported, err := extractCallWorkflowPermissionImport("worker-review", markdownPath)
	require.NoError(t, err, "Should extract imported permissions without error")
	require.NotNil(t, imported, "Should return import metadata")
	require.NotNil(t, imported.permissions, "Should include permissions")
	assert.Equal(t, workflowSourceKindLock, imported.sourceKind, "Should track lock workflow source kind")
	assert.Equal(t, "./.github/workflows/worker-review.lock.yml", renderWorkflowReviewPath(imported.sourcePath),
		"Should render a repo-relative review path for help comments")
}

func TestBuildCallWorkflowPermissionsComment_NilInputs(t *testing.T) {
	assert.Empty(t, buildCallWorkflowPermissionsComment("worker", nil), "Nil import should not emit a comment")
	assert.Empty(t, buildCallWorkflowPermissionsComment("worker", &callWorkflowPermissionImport{}), "Nil permissions should not emit a comment")
	assert.Empty(t, buildCallWorkflowPermissionsComment("worker", &callWorkflowPermissionImport{
		permissions: NewPermissions(),
		sourceKind:  workflowSourceKindLock,
	}), "Empty permissions should not emit a comment")
}

// TestBuildCallWorkflowJobs_SetsPermissionsFromLockYML tests that call-workflow jobs
// carry the union of caller + worker permissions when a .lock.yml worker file is present.
// When the caller already covers all of the worker's needs, the effective permissions
// equal the caller's declared permissions.
func TestBuildCallWorkflowJobs_SetsPermissionsFromLockYML(t *testing.T) {
	t.Setenv("GH_AW_WORKFLOWS_DIR", "")

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
		// Caller declares its own envelope; the caller already covers all of the worker's
		// needs so the effective permissions equal the caller's declared permissions.
		Permissions: "permissions:\n  contents: write\n  issues: write\n  pull-requests: write",
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
	assert.Contains(t, job.PermissionsComment,
		`Imported from called workflow "worker-docs" because GitHub requires the caller job to grant permissions requested by reusable workflow jobs.`,
		"Job should explain why worker permissions are imported")
	assert.Contains(t, job.PermissionsComment,
		"Review the called workflow's job-level permissions in ./.github/workflows/worker-docs.lock.yml.",
		"Job should point reviewers to the compiled worker workflow")
	assert.NotEmpty(t, job.Permissions, "Job should have permissions set")
	assert.Contains(t, job.Permissions, "contents: write", "Permissions should include contents: write")
	assert.Contains(t, job.Permissions, "issues: write", "Permissions should include issues: write")
	assert.Contains(t, job.Permissions, "pull-requests: write", "Permissions should include pull-requests: write")
}

// TestBuildCallWorkflowJobs_SetsPermissionsFromMD tests that call-workflow jobs carry the
// union of caller + worker permissions even when the worker is a same-batch .md compilation
// target. When caller and worker declare the same permissions, the effective permissions
// equal the caller's declared permissions.
func TestBuildCallWorkflowJobs_SetsPermissionsFromMD(t *testing.T) {
	t.Setenv("GH_AW_WORKFLOWS_DIR", "")

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
		// Caller and worker declare the same permissions, so the effective permissions equal this.
		Permissions: "permissions:\n  contents: read\n  issues: write",
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
	assert.Contains(t, job.PermissionsComment,
		"Review the called workflow's frontmatter permissions in ./.github/workflows/worker-e.md.",
		"Job should point reviewers to the markdown worker when no compiled file exists yet")
	assert.NotEmpty(t, job.Permissions, "Job should have permissions")
	assert.Contains(t, job.Permissions, "contents: read", "Permissions should include contents: read")
	assert.Contains(t, job.Permissions, "issues: write", "Permissions should include issues: write")
}

// TestBuildCallWorkflowJobs_WorkerPermissionsElevateCallerPermissions tests the core fix:
// when the caller's declared permissions are insufficient for the worker, the compiler
// automatically promotes the call-workflow job's permissions to the union of both.
// This prevents GitHub's startup_failure when a reusable workflow job requests a
// permission level greater than the caller grants.
func TestBuildCallWorkflowJobs_WorkerPermissionsElevateCallerPermissions(t *testing.T) {
	compiler := NewCompiler(WithVersion("1.0.0"))

	tmpDir := t.TempDir()
	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
	require.NoError(t, os.MkdirAll(workflowsDir, 0755), "Failed to create workflows directory")

	// Worker needs issues: write and pull-requests: write (typical agentic workflow).
	workerContent := `name: Worker
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
      contents: read
      issues: write
      pull-requests: write
    runs-on: ubuntu-latest
    steps:
      - run: echo "agent"
`
	require.NoError(t, os.WriteFile(filepath.Join(workflowsDir, "worker-g.lock.yml"), []byte(workerContent), 0644), "Failed to write worker-g.lock.yml")

	markdownPath := filepath.Join(workflowsDir, "gateway.md")

	workflowData := &WorkflowData{
		// Caller declares only contents: read and pull-requests: read — insufficient for
		// the worker which needs issues: write and pull-requests: write.
		Permissions: "permissions:\n  contents: read\n  pull-requests: read",
		SafeOutputs: &SafeOutputsConfig{
			CallWorkflow: &CallWorkflowConfig{
				BaseSafeOutputConfig: BaseSafeOutputConfig{Max: strPtr("1")},
				Workflows:            []string{"worker-g"},
				WorkflowFiles: map[string]string{
					"worker-g": "./.github/workflows/worker-g.lock.yml",
				},
			},
		},
	}

	jobNames, err := compiler.buildCallWorkflowJobs(workflowData, markdownPath)
	require.NoError(t, err, "Should build call-workflow jobs without error")
	assert.Equal(t, []string{"call-worker-g"}, jobNames, "Should generate the job")

	job, exists := compiler.jobManager.GetJob("call-worker-g")
	require.True(t, exists, "Job should exist in job manager")
	assert.NotEmpty(t, job.Permissions, "Job should have permissions")
	// The call-* job must carry the union of caller + worker permissions so that GitHub
	// does not reject the call at startup:
	assert.Contains(t, job.Permissions, "contents: read", "Should include contents: read from caller")
	assert.Contains(t, job.Permissions, "issues: write", "Should include issues: write elevated from worker")
	assert.Contains(t, job.Permissions, "pull-requests: write", "Should include pull-requests: write elevated from worker (overrides caller's read)")
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

// TestCallWorkflowJobYAMLOutput_WithPermissions tests that the YAML output of a
// call-workflow job includes the union of the caller's declared permissions and the
// worker's required permissions. When the caller already covers the worker's needs,
// the effective permissions equal the caller's declared permissions.
func TestCallWorkflowJobYAMLOutput_WithPermissions(t *testing.T) {
	compiler := NewCompiler(WithVersion("1.0.0"))

	tmpDir := t.TempDir()
	workflowsDir := filepath.Join(tmpDir, ".github", "workflows")
	require.NoError(t, os.MkdirAll(workflowsDir, 0755), "Failed to create workflows directory")

	// Worker requires contents: write and issues: write. The caller declares a
	// broader envelope below; the call-* job must reflect the CALLER's permissions.
	workerContent := `name: Worker
on:
  workflow_call: {}
jobs:
  agent:
    permissions:
      contents: write
      issues: write
    runs-on: ubuntu-latest
    steps:
      - run: echo "agent"
`
	require.NoError(t, os.WriteFile(filepath.Join(workflowsDir, "worker-a.lock.yml"), []byte(workerContent), 0644), "Failed to write worker-a.lock.yml")

	markdownPath := filepath.Join(workflowsDir, "gateway.md")

	workflowData := &WorkflowData{
		// Caller's declared permissions — caller already covers the worker's needs so the
		// effective permissions equal the caller's declared permissions.
		Permissions: "permissions:\n  contents: write\n  issues: write\n  pull-requests: write",
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

	var yamlBuf strings.Builder
	compiler.jobManager.WriteJobsYAML(&yamlBuf)
	yamlOutput := yamlBuf.String()

	assert.Contains(t, yamlOutput, "uses: ./.github/workflows/worker-a.lock.yml", "Should contain uses directive")
	assert.Contains(t, yamlOutput, "secrets: inherit", "Should inherit secrets")
	assert.Contains(t, yamlOutput, "permissions:", "Should include permissions block")
	assert.Contains(t, yamlOutput,
		`# Imported from called workflow "worker-a" because GitHub requires the caller job to grant permissions requested by reusable workflow jobs.`,
		"Rendered YAML should explain imported workflow_call permissions")
	assert.Contains(t, yamlOutput,
		"# Review the called workflow's job-level permissions in ./.github/workflows/worker-a.lock.yml.",
		"Rendered YAML should point to the worker workflow for review")
	// The call-* job gets the union of caller + worker permissions. Since the caller
	// already covers all of the worker's needs, the effective permissions equal the
	// caller's declared permissions.
	assert.Contains(t, yamlOutput, "contents: write", "Should include contents: write")
	assert.Contains(t, yamlOutput, "issues: write", "Should include issues: write")
	assert.Contains(t, yamlOutput, "pull-requests: write", "Should include pull-requests: write")

	// Verify permissions appear before uses in the YAML (job-level ordering)
	permIdx := strings.Index(yamlOutput, "permissions:")
	usesIdx := strings.Index(yamlOutput, "uses:")
	require.NotEqual(t, -1, permIdx, "permissions: should be present in YAML output")
	require.NotEqual(t, -1, usesIdx, "uses: should be present in YAML output")
	assert.Less(t, permIdx, usesIdx, "permissions: should appear before uses: in job YAML")
}

// TestExtractCallWorkflowPermissions_LockYMLPriorityOverYML tests that .lock.yml takes
// priority over .yml when both exist

// TestCallWorkflowPermissions_EndToEnd tests full gateway compilation with permissioned workers —
// every call-* job must carry the union of the caller's declared permissions and the worker's
// required permissions, so the call job always grants enough scope for the worker to run.
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
	// The call-* job carries the union of the caller's declared permissions (contents: read)
	// and the worker's required permissions (contents: write, issues: write, pull-requests: write,
	// actions: read). The worker's broader write scope wins over the caller's read for contents.
	assert.Contains(t, callASection, "contents: write", "call-worker-a permissions should include worker's contents: write")
	assert.Contains(t, callASection, "issues: write", "call-worker-a permissions should include worker's issues: write")
	assert.Contains(t, callASection, "pull-requests: write", "call-worker-a permissions should include worker's pull-requests: write")
	assert.Contains(t, callASection, "actions: read", "call-worker-a permissions should include worker's actions: read")

	// Extract the YAML section for call-worker-b (bounded to just this job, since later
	// framework jobs such as conclusion legitimately carry issues: write).
	callBSection := yamlOutput[callBStart:]
	if convIdx := strings.Index(callBSection, "\n  conclusion:"); convIdx != -1 {
		callBSection = callBSection[:convIdx]
	}
	assert.Contains(t, callBSection, "permissions:", "call-worker-b job must have permissions block")
	// call-worker-b union: caller's contents: read + worker's issues: write.
	assert.Contains(t, callBSection, "contents: read", "call-worker-b permissions should include caller's contents: read")
	assert.Contains(t, callBSection, "issues: write", "call-worker-b permissions should include worker's issues: write")
}

// TestCallWorkflowPermissions_EndToEnd_YMLWorker tests that when a worker is referenced via a
// .yml file (not .lock.yml), the generated call-* job carries the union of the caller's declared
// permissions and the worker's required permissions.
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
	if convIdx := strings.Index(callSection, "\n  conclusion:"); convIdx != -1 {
		callSection = callSection[:convIdx]
	}
	assert.Contains(t, callSection, "permissions:", "call-worker-plain job must have permissions block")
	// The call-* job carries the union of the caller's declared permissions (contents: read) and
	// the worker's requirements (contents: read, pull-requests: write).
	assert.Contains(t, callSection, "contents: read", "Permissions should include caller's contents: read")
	assert.Contains(t, callSection, "pull-requests: write", "Permissions should include worker's pull-requests: write")
}
