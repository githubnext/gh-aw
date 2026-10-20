//go:build !integration

package workflow

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/constants"
	"github.com/github/gh-aw/pkg/stringutil"
	"github.com/github/gh-aw/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdmissionRuntimeBudgetSelection(t *testing.T) {
	for _, tc := range []struct {
		name, result, allowance, expected string
		component, invalid                bool
		local                             int64
		evals                             bool
	}{
		{"approved overrides local", "admitted", "111", "111", false, false, 777, false},
		{"approved higher allowance", "admitted", "999", "999", false, false, 777, false},
		{"approved overrides unlimited", "admitted", "111", "111", false, false, -1, false},
		{"disabled retains local", "", "", "777", false, false, 777, false},
		{"disabled retains unlimited", "", "", "-1", false, false, -1, false},
		{"ignored ignores stale allowance", "ignored", "111", "777", false, false, 777, false},
		{"optional absent retains component", "admitted", "", "777", true, false, 777, false},
		{"primary absent blocks", "admitted", "", "", false, true, 777, false},
		{"invalid allowance blocks", "admitted", "12oops", "", false, true, 777, false},
		{"overflow blocks", "admitted", "2147483648", "", false, true, 777, false},
		{"shell payload blocks", "admitted", "$(echo unsafe)", "", false, true, 777, false},
		{"detection selects allowance", "admitted", "111", "111", true, false, 777, false},
		{"evals selects allowance", "admitted", "999", "999", false, false, 777, true},
		{"evals missing optional retains policy", "admitted", "", "777", false, false, 777, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			data := &WorkflowData{EngineConfig: &EngineConfig{MaxAICredits: tc.local}, IsDetectionRun: tc.component, IsEvalsRun: tc.evals}
			script, config := buildMaxAICreditsExport(AWFCommandConfig{WorkflowData: data}, `{"apiProxy":{"maxRuns":1,"maxAiCredits":777}}`)
			assert.Equal(t, 1, strings.Count(config, `"maxAiCredits"`))
			assert.Contains(t, config, `"maxAiCredits":${GH_AW_MAX_AI_CREDITS}`)
			resultFile := filepath.Join(t.TempDir(), "budget")
			cmd := exec.Command("bash", "-c", script+"\nprintf '%s' "+buildAWFConfigPrintfArg(config, true)+" > \"$BUDGET_RESULT\"")
			cmd.Env = append(os.Environ(), "GH_AW_ADMISSION_RESULT="+tc.result, "GH_AW_ADMISSION_MAX_AI_CREDITS="+tc.allowance, "GH_AW_MAX_AI_CREDITS=222", "BUDGET_RESULT="+resultFile)
			output, err := cmd.CombinedOutput()
			if tc.invalid {
				require.Error(t, err)
				assert.Contains(t, string(output), "ERR_ADMISSION")
				return
			}
			require.NoError(t, err, "%s", output)
			value, err := os.ReadFile(resultFile)
			require.NoError(t, err)
			assert.JSONEq(t, `{"apiProxy":{"maxRuns":1,"maxAiCredits":`+tc.expected+`}}`, string(value))
			for _, withheld := range []string{"777", "111", "999", "222"} {
				assert.NotContains(t, string(output), withheld)
			}
		})
	}
}

func TestAdmissionComponentBudgetOutputs(t *testing.T) {
	for _, tc := range []struct {
		name, output     string
		detection, evals bool
	}{
		{"agent", "admission-max-ai-credits", false, false},
		{"detection", "admission-max-ai-credits-threat-detection", true, false},
		{"evals", "admission-max-ai-credits-evals", false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			env := map[string]string{}
			applyDefaultMaxAICreditsEnvToMap(env, &WorkflowData{EngineConfig: &EngineConfig{MaxAICredits: 777}, IsDetectionRun: tc.detection, IsEvalsRun: tc.evals})
			assert.Equal(t, "${{ needs.activation.outputs."+tc.output+" }}", env["GH_AW_ADMISSION_MAX_AI_CREDITS"])
			assert.Equal(t, "${{ needs.activation.outputs.admission-result }}", env["GH_AW_ADMISSION_RESULT"])
		})
	}
}

func TestAdmissionSessionOutputs(t *testing.T) {
	for _, daily := range []bool{false, true} {
		step := strings.Join(buildUsageActivityGenerationStep(daily, false, func(action string) string { return action }), "")
		for env, output := range map[string]string{
			"GH_AW_ADMISSION_RESULT":                          "admission-result",
			"GH_AW_ADMISSION_MAX_AI_CREDITS":                  "admission-max-ai-credits",
			"GH_AW_ADMISSION_MAX_AI_CREDITS_THREAT_DETECTION": "admission-max-ai-credits-threat-detection",
			"GH_AW_ADMISSION_MAX_AI_CREDITS_EVALS":            "admission-max-ai-credits-evals",
		} {
			assert.Contains(t, step, env+": ${{ needs.activation.outputs."+output+" }}")
		}
	}
}

func TestAdmissionComponentExecutionWiring(t *testing.T) {
	for _, engine := range []string{"copilot", "claude", "codex", "pi"} {
		t.Run(engine, func(t *testing.T) {
			data := &WorkflowData{
				AI:           engine,
				EngineConfig: &EngineConfig{ID: engine, MaxAICredits: 777},
				SafeOutputs:  &SafeOutputsConfig{ThreatDetection: &ThreatDetectionConfig{}},
			}
			compiler := NewCompiler()
			evals := strings.Join(compiler.buildEvalsEngineSteps(data), "")
			assert.Contains(t, evals, "GH_AW_ADMISSION_MAX_AI_CREDITS: ${{ needs.activation.outputs.admission-max-ai-credits-evals }}")
			assert.NotContains(t, evals, "needs.activation.outputs.admission-max-ai-credits-threat-detection")
			assert.Contains(t, evals, `GH_AW_MAX_AI_CREDITS="$GH_AW_ADMISSION_MAX_AI_CREDITS"`)
			for _, external := range []bool{false, true} {
				data.Features = map[string]any{string(constants.GHAWDetectionFeatureFlag): external}
				detection := strings.Join(compiler.buildDetectionJobSteps(data), "")
				assert.Contains(t, detection, "GH_AW_ADMISSION_MAX_AI_CREDITS: ${{ needs.activation.outputs.admission-max-ai-credits-threat-detection }}")
				assert.NotContains(t, detection, "needs.activation.outputs.admission-max-ai-credits-evals")
				assert.Contains(t, detection, `GH_AW_MAX_AI_CREDITS="$GH_AW_ADMISSION_MAX_AI_CREDITS"`)
			}
		})
	}
}

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
		"id: admission", "id-token: write", "activation_admission.cjs",
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
		"            await require(require('node:path').join(process.env.RUNNER_TEMP, 'gh-aw', 'actions', 'activation_admission.cjs')).main(core, github, context, exec, io, getOctokit);",
	}, "\n"))
	assert.Contains(t, yaml, admissionDailyAICFallbackIfExpr)
	assert.Contains(t, yaml, "\n  activation_admission:\n")
	assert.NotContains(t, yaml, "\n  admission:\n")
	admissionJob := compiler.jobManager.jobs[activationAdmissionJobName]
	require.NotNil(t, admissionJob)
	assert.Contains(t, admissionJob.Permissions, "id-token: write")
	assert.Contains(t, admissionJob.If, "vars.GH_AW_DEFAULT_ADMISSION_URL != ''")
	assert.NotContains(t, compiler.jobManager.jobs["activation"].Permissions, "id-token: write")
	assert.Contains(t, compiler.jobManager.jobs["activation"].Needs, activationAdmissionJobName)
	for name, job := range compiler.jobManager.jobs {
		if name == "activation" {
			assert.Equal(t, "${{ needs.activation_admission.outputs.max-ai-credits }}", job.Outputs["admission-max-ai-credits"])
			continue
		}
		if name == activationAdmissionJobName {
			continue
		}
		encoded, err := json.Marshal(job)
		require.NoError(t, err)
		for _, fragment := range []string{"steps.admission", "admission-aic", "admission_budget.cjs", "GH_AW_ADMISSION_WORKFLOW_HASH"} {
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
