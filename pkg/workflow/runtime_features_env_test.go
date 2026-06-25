//go:build !integration

package workflow

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/constants"
)

func TestBuildRenderedJobEnv_AddsRuntimeFeaturesForBuiltInJobs(t *testing.T) {
	job := &Job{Name: string(constants.ActivationJobName)}

	env := buildRenderedJobEnv(job)

	if env[runtimeFeaturesEnvVarName] != runtimeFeaturesEnvVarExpression {
		t.Fatalf("%s = %q, want %q", runtimeFeaturesEnvVarName, env[runtimeFeaturesEnvVarName], runtimeFeaturesEnvVarExpression)
	}
}

func TestBuildRenderedJobEnv_DoesNotAddRuntimeFeaturesForCustomJobs(t *testing.T) {
	job := &Job{Name: "custom_job"}

	env := buildRenderedJobEnv(job)

	if len(env) != 0 {
		t.Fatalf("expected no env vars for custom job, got %v", env)
	}
}

func TestBuildRenderedJobEnv_PreservesExistingRuntimeFeaturesOverride(t *testing.T) {
	job := &Job{
		Name: string(constants.AgentJobName),
		Env: map[string]string{
			runtimeFeaturesEnvVarName: `"explicit"`,
			"KEEP_ME":                 `"yes"`,
		},
	}

	env := buildRenderedJobEnv(job)

	if env[runtimeFeaturesEnvVarName] != `"explicit"` {
		t.Fatalf("expected explicit runtime features value to be preserved, got %q", env[runtimeFeaturesEnvVarName])
	}
	if env["KEEP_ME"] != `"yes"` {
		t.Fatalf("expected KEEP_ME env var to be preserved, got %q", env["KEEP_ME"])
	}
}

func TestBuildRenderedJobEnv_DoesNotAddRuntimeFeaturesForUsesJobs(t *testing.T) {
	job := &Job{
		Name: string(constants.AgentJobName),
		Uses: "./.github/workflows/reusable.yml",
	}

	env := buildRenderedJobEnv(job)

	if _, ok := env[runtimeFeaturesEnvVarName]; ok {
		t.Fatalf("expected reusable workflow job to skip %s, got %v", runtimeFeaturesEnvVarName, env)
	}
}

func TestActivationJobIncludesRuntimeFeatureSummaryStep(t *testing.T) {
	compiler := NewCompiler()
	compiler.repoConfigLoaded = true
	compiler.repoConfig = &RepoConfig{}

	job, err := compiler.buildActivationJob(&WorkflowData{}, false, "", "test.lock.yml")
	if err != nil {
		t.Fatalf("buildActivationJob() error = %v", err)
	}

	steps := strings.Join(job.Steps, "\n")
	if !strings.Contains(steps, "name: Log runtime features") {
		t.Fatal("expected activation job to include runtime feature summary step")
	}
	if !strings.Contains(steps, "GH_AW_RUNTIME_FEATURES") {
		t.Fatal("expected runtime feature summary step to reference GH_AW_RUNTIME_FEATURES")
	}
	if !strings.Contains(steps, "if:") {
		t.Fatal("expected runtime feature summary step to use if: condition to skip when var is unset")
	}
	if !strings.Contains(steps, "log_runtime_features_summary.sh") {
		t.Fatal("expected runtime feature summary step to call shared shell script")
	}

	// Verify that the shared shell script itself writes to GITHUB_STEP_SUMMARY, so the
	// behavioral contract is not silently broken by editing the script.
	_, testFile, _, _ := runtime.Caller(0)
	repoRoot := filepath.Join(filepath.Dir(testFile), "..", "..")
	scriptPath := filepath.Join(repoRoot, "actions", "setup", "sh", "log_runtime_features_summary.sh")
	scriptContent, err := os.ReadFile(scriptPath)
	if err != nil {
		t.Fatalf("expected shared shell script to exist at %s: %v", scriptPath, err)
	}
	if !strings.Contains(string(scriptContent), "GITHUB_STEP_SUMMARY") {
		t.Fatal("expected shared shell script to write to GITHUB_STEP_SUMMARY")
	}
}
