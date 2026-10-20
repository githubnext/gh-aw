//go:build !integration

package workflow

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/stringutil"
	"github.com/github/gh-aw/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdmissionRuntimeWiring(t *testing.T) {
	t.Setenv("GH_AW_DEFAULT_ADMISSION_URL", "https://compiler-process.invalid/api/admission")
	t.Setenv("GH_AW_ADMISSION_WORKFLOW_HASH", "compiler-process-hash")
	dir := testutil.TempDir(t, "admission-runtime-*")
	file := filepath.Join(dir, "admission.md")
	require.NoError(t, os.WriteFile(file, []byte("---\non:\n  workflow_dispatch:\n  stale-check: false\nsafe-outputs:\n  add-comment:\n    max: 1\n---\nAdmission runtime\n"), 0o644))
	compiler := NewCompiler()
	require.NoError(t, compiler.CompileWorkflow(file))
	content, err := os.ReadFile(stringutil.MarkdownToLockFile(file))
	require.NoError(t, err)
	yaml := string(content)
	for _, fragment := range []string{
		"id: admission", "id-token: write", "check_admission.cjs",
		"if: ${{ vars.GH_AW_DEFAULT_ADMISSION_URL != '' }}",
		"GH_AW_DEFAULT_ADMISSION_URL: ${{ vars.GH_AW_DEFAULT_ADMISSION_URL }}",
		"GH_AW_DEFAULT_ADMISSION_AUDIENCE: ${{ vars.GH_AW_DEFAULT_ADMISSION_AUDIENCE }}",
		"GH_AW_DEFAULT_ADMISSION_ON_OUTAGE: ${{ vars.GH_AW_DEFAULT_ADMISSION_ON_OUTAGE }}",
		"GH_AW_INFO_WORKFLOW_ID: ${{ job.workflow_ref }}",
		"GITHUB_ACTOR: ${{ github.actor }}", "GITHUB_TRIGGERING_ACTOR: ${{ github.triggering_actor }}",
	} {
		assert.Contains(t, yaml, fragment)
	}
	assert.NotContains(t, yaml, "compiler-process.invalid")
	assert.NotContains(t, yaml, "compiler-process-hash")
	hashPattern := regexp.MustCompile(`GH_AW_ADMISSION_WORKFLOW_HASH: "([0-9a-f]{64}):([0-9a-f]{64})"`)
	hashes := hashPattern.FindStringSubmatch(yaml)
	require.Len(t, hashes, 3)
	assert.Contains(t, yaml, `"frontmatter_hash":"`+hashes[1]+`"`)
	assert.Contains(t, yaml, `"body_hash":"`+hashes[2]+`"`)
	assert.Contains(t, yaml, strings.Join([]string{
		"            const path = require('node:path');",
		"            const actionsDir = path.join(process.env.RUNNER_TEMP, 'gh-aw', 'actions');",
		"            const { setupGlobals } = require(path.join(actionsDir, 'setup_globals.cjs'));",
		"            setupGlobals(core, github, context, exec, io, getOctokit);",
		"            const { main } = require(path.join(actionsDir, 'check_admission.cjs'));",
		"            await main(core);",
	}, "\n"))
	assert.Less(t, strings.Index(yaml, "id: admission"), strings.Index(yaml, "id: daily-ai-credits-workflow-guardrail"))
	assert.Contains(t, yaml, admissionDailyAICFallbackIfExpr)
	assert.NotContains(t, yaml, "\n  admission:\n")
	for name, job := range compiler.jobManager.jobs {
		for output := range job.Outputs {
			assert.NotContains(t, output, "admission", "job %s", name)
		}
		if name == "activation" {
			continue
		}
		encoded, err := json.Marshal(job)
		require.NoError(t, err)
		for _, fragment := range []string{"GH_AW_ADMISSION", "steps.admission", "outputs.admission", "admission-aic", "admission_budget.cjs"} {
			assert.NotContains(t, string(encoded), fragment, "job %s", name)
		}
	}
	require.NoError(t, os.WriteFile(file, []byte("---\non:\n  workflow_dispatch:\n  stale-check: false\nsafe-outputs:\n  add-comment:\n    max: 1\n---\nChanged admission prompt\n"), 0o644))
	require.NoError(t, NewCompiler().CompileWorkflow(file))
	changed, err := os.ReadFile(stringutil.MarkdownToLockFile(file))
	require.NoError(t, err)
	changedHashes := hashPattern.FindStringSubmatch(string(changed))
	require.Len(t, changedHashes, 3)
	assert.Equal(t, hashes[1], changedHashes[1])
	assert.NotEqual(t, hashes[2], changedHashes[2], "Prompt changes must invalidate workflow-content caches")
}

func TestAdmissionRejectsCodeGenerationSwitch(t *testing.T) {
	file := filepath.Join(testutil.TempDir(t, "admission-unsupported-gate-*"), "workflow.md")
	for _, value := range []string{"false", "true"} {
		t.Run(value, func(t *testing.T) {
			source := "---\non: workflow_dispatch\nadmission-gate: " + value + "\n---\nTest workflow\n"
			require.NoError(t, os.WriteFile(file, []byte(source), 0o644))
			err := NewCompiler().CompileWorkflow(file)
			require.ErrorContains(t, err, "admission-gate", "admission code generation has no frontmatter switch")
		})
	}
}

func TestAdmissionRepositoryReusableCallerPermissions(t *testing.T) {
	files, err := filepath.Glob("../../.github/workflows/*.yml")
	require.NoError(t, err)
	callers := 0
	for _, file := range files {
		if strings.HasSuffix(file, ".lock.yml") {
			continue
		}
		workflow, err := readWorkflowYAML(file)
		require.NoError(t, err)
		jobs, ok := workflow["jobs"].(map[string]any)
		require.True(t, ok, file)
		for name, value := range jobs {
			job, ok := value.(map[string]any)
			require.True(t, ok, "%s/%s", file, name)
			uses, _ := job["uses"].(string)
			if !strings.HasPrefix(uses, "./.github/workflows/") || !strings.HasSuffix(uses, ".lock.yml") {
				continue
			}
			perms := NewPermissionsParserFromValue(job["permissions"]).ToPermissions()
			level, exists := perms.Get(PermissionIdToken)
			assert.True(t, exists, "%s/%s must permit the callee's activation OIDC", file, name)
			assert.Equal(t, PermissionWrite, level, "%s/%s", file, name)
			callers++
		}
	}
	require.NotZero(t, callers, "must cover the repository's hand-authored reusable callers")
}

func TestAdmissionReusableCallerPermissionsWithStaleWorkerLock(t *testing.T) {
	t.Setenv("GH_AW_WORKFLOWS_DIR", "")
	dir := filepath.Join(testutil.TempDir(t, "admission-reusable-*"), ".github", "workflows")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	worker := filepath.Join(dir, "worker.md")
	require.NoError(t, os.WriteFile(worker, []byte("---\non:\n  workflow_call: {}\n---\nWorker\n"), 0o644))
	for _, staleLock := range []bool{false, true} {
		if staleLock {
			require.NoError(t, os.WriteFile(stringutil.MarkdownToLockFile(worker), []byte("on:\n  workflow_call: {}\njobs:\n  activation:\n    permissions:\n      contents: read\n    runs-on: ubuntu-latest\n    steps:\n      - run: echo old-worker\n"), 0o644))
		}
		imported, err := extractCallWorkflowPermissionImport("worker", filepath.Join(dir, "caller.md"))
		require.NoError(t, err)
		require.NotNil(t, imported)
		assert.Contains(t, imported.permissions.RenderToYAML(), "id-token: write")
	}
}
