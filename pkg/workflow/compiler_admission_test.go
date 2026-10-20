//go:build !integration

package workflow

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/stringutil"
	"github.com/github/gh-aw/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdmissionRuntimeWiring(t *testing.T) {
	t.Setenv("GH_AW_DEFAULT_ADMISSION_URL", "https://compiler-process.invalid/api/admission")
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
