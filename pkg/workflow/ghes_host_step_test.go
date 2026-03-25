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
	assert.Contains(t, step, "GITHUB_ENV", "step should write to GITHUB_ENV so all subsequent steps inherit GH_HOST")
	assert.NotContains(t, step, "GITHUB_OUTPUT", "step should not write to GITHUB_OUTPUT (GITHUB_ENV makes it available to all steps)")
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

	// Second step should be the user's custom step
	assert.Contains(t, job.Steps[1], "My custom step",
		"second step should be the user's custom step")
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
