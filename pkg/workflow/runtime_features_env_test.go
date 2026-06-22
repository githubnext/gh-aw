//go:build !integration

package workflow

import (
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
	if !strings.Contains(steps, "GH_AW_RUNTIME_FEATURES_IS_SET") {
		t.Fatal("expected runtime feature summary step to distinguish unset from empty values")
	}
	if !strings.Contains(steps, "_Empty string_") {
		t.Fatal("expected runtime feature summary step to render empty values distinctly")
	}
	if !strings.Contains(steps, "$GITHUB_STEP_SUMMARY") {
		t.Fatal("expected runtime feature summary step to write to GITHUB_STEP_SUMMARY")
	}
}
