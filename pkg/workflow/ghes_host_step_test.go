//go:build !integration

package workflow

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateGHESHostConfigurationStep(t *testing.T) {
	step := generateGHESHostConfigurationStep()

	assert.Contains(t, step, "Configure GH_HOST for enterprise compatibility", "step should have the expected name")
	assert.Contains(t, step, "id: ghes-host-config", "step should have the step ID ghes-host-config")
	assert.Contains(t, step, "shell: bash", "step should explicitly set shell to bash for Windows runner compatibility")
	assert.Contains(t, step, "GITHUB_SERVER_URL", "step should reference GITHUB_SERVER_URL")
	assert.Contains(t, step, "GH_HOST=", "step should set GH_HOST")
	assert.Contains(t, step, "GITHUB_OUTPUT", "step should write to GITHUB_OUTPUT for step-scoped propagation")
	assert.NotContains(t, step, "GITHUB_ENV", "step should not write to GITHUB_ENV (avoids github-env security finding)")
	assert.Contains(t, step, "${GITHUB_SERVER_URL#https://}", "step should strip https:// prefix")
	assert.Contains(t, step, "${GH_HOST#http://}", "step should also strip http:// prefix")

	// Verify it's valid YAML indentation (6 spaces for step-level)
	for line := range strings.SplitSeq(step, "\n") {
		if line == "" {
			continue
		}
		assert.True(t, strings.HasPrefix(line, "      "), "each line should be indented at step level (6 spaces): %q", line)
	}
}

func TestGHESHostStepConstants(t *testing.T) {
	assert.Equal(t, "ghes-host-config", GHESHostStepID, "step ID constant should match expected value")
	assert.Equal(t, "${{ steps.ghes-host-config.outputs.GH_HOST }}", GHESHostOutputExpr,
		"output expression should reference the correct step ID and output name")
}

func TestGHESHostStepInCustomJobs(t *testing.T) {
	compiler := &Compiler{
		jobManager: NewJobManager(),
		actionMode: ActionModeRelease,
	}

	data := &WorkflowData{
		Name: "test-workflow",
		Jobs: map[string]any{
			"custom-job": map[string]any{
				"runs-on": "ubuntu-latest",
				"steps": []any{
					map[string]any{
						"name": "My custom step",
						"run":  "echo hello",
					},
				},
			},
		},
	}

	err := compiler.buildCustomJobs(data, false)
	require.NoError(t, err, "should build custom jobs without error")

	job, exists := compiler.jobManager.jobs["custom-job"]
	assert.True(t, exists, "custom-job should exist")

	// First step should be the GH_HOST configuration
	assert.Greater(t, len(job.Steps), 1, "should have at least 2 steps (GH_HOST config + custom)")
	assert.Contains(t, job.Steps[0], "Configure GH_HOST for enterprise compatibility",
		"first step should be the GH_HOST configuration step")

	// Second step should be the user's custom step with GH_HOST env injected
	assert.Contains(t, job.Steps[1], "My custom step",
		"second step should be the user's custom step")
	assert.Contains(t, job.Steps[1], "GH_HOST",
		"user step should have GH_HOST env var injected from ghes-host-config output")
	assert.Contains(t, job.Steps[1], GHESHostOutputExpr,
		"user step should reference GH_HOST from ghes-host-config step output")
}

func TestGHESHostStepNotInReusableWorkflowJobs(t *testing.T) {
	compiler := &Compiler{
		jobManager: NewJobManager(),
		actionMode: ActionModeRelease,
	}

	data := &WorkflowData{
		Name: "test-workflow",
		Jobs: map[string]any{
			"reusable-job": map[string]any{
				"uses": "./.github/workflows/reusable.yml",
			},
		},
	}

	err := compiler.buildCustomJobs(data, false)
	require.NoError(t, err, "should build reusable workflow jobs without error")

	job, exists := compiler.jobManager.jobs["reusable-job"]
	assert.True(t, exists, "reusable-job should exist")

	// Reusable workflow jobs should have no steps (they use `uses:`)
	assert.Empty(t, job.Steps, "reusable workflow jobs should have no steps")
}

func TestInjectGHESHostEnv(t *testing.T) {
	t.Run("injects GH_HOST into step with no existing env", func(t *testing.T) {
		step := &WorkflowStep{
			Name: "test step",
			Run:  "echo hello",
		}
		injectGHESHostEnv(step)
		require.NotNil(t, step.Env, "env map should be initialized")
		assert.Equal(t, GHESHostOutputExpr, step.Env["GH_HOST"],
			"GH_HOST should reference the ghes-host-config step output")
	})

	t.Run("preserves existing env vars when injecting GH_HOST", func(t *testing.T) {
		step := &WorkflowStep{
			Name: "test step",
			Run:  "echo hello",
			Env: map[string]string{
				"MY_VAR": "my-value",
			},
		}
		injectGHESHostEnv(step)
		assert.Equal(t, GHESHostOutputExpr, step.Env["GH_HOST"],
			"GH_HOST should be injected")
		assert.Equal(t, "my-value", step.Env["MY_VAR"],
			"existing env vars should be preserved")
	})
}
