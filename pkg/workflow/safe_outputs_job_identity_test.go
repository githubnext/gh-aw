//go:build !integration

package workflow

import (
	"strings"
	"testing"

	"github.com/goccy/go-yaml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type publishingIdentityJob struct {
	Env   map[string]string `yaml:"env"`
	Steps []struct {
		Name string            `yaml:"name"`
		Env  map[string]string `yaml:"env"`
	} `yaml:"steps"`
}

func assertPublishingStepIdentity(t *testing.T, job publishingIdentityJob, stepName string) {
	t.Helper()
	assert.NotContains(t, job.Env, "GH_AW_JOB_CHECK_RUN_ID", "job context is unavailable in job-level env")
	for _, step := range job.Steps {
		if step.Name == stepName {
			assert.Equal(t, "${{ job.check_run_id }}", step.Env["GH_AW_JOB_CHECK_RUN_ID"],
				"publishing identity must resolve the runtime job instance, not the YAML job key")
			return
		}
		assert.NotContains(t, step.Env, "GH_AW_JOB_CHECK_RUN_ID")
	}
	t.Fatalf("publishing step %q not found", stepName)
}

func TestSafeOutputsPublishingJobIdentity(t *testing.T) {
	for _, tc := range []struct {
		name     string
		on       string
		strategy string
	}{
		{name: "ordinary", on: "on: workflow_dispatch"},
		{name: "reusable", on: "on:\n  workflow_call:"},
		{
			name:     "matrix",
			on:       "on: workflow_dispatch",
			strategy: "strategy:\n      matrix:\n        target: [first, second]",
		},
		{
			name:     "reusable matrix",
			on:       "on:\n  workflow_call:",
			strategy: "strategy:\n      matrix:\n        target: [first, second]",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			compiler := NewCompiler()
			data := &WorkflowData{
				Name: "Reusable Publisher",
				On:   tc.on,
				SafeOutputs: &SafeOutputsConfig{
					CreateIssues: &CreateIssuesConfig{},
				},
			}
			job, _, err := compiler.buildConsolidatedSafeOutputsJob(data, "agent", "reusable-publisher.md")
			require.NoError(t, err)
			require.NotNil(t, job)
			job.Strategy = tc.strategy
			require.NoError(t, compiler.jobManager.AddJob(job))
			var rendered strings.Builder
			compiler.jobManager.WriteJobsYAML(&rendered)
			var parsed struct {
				Jobs map[string]publishingIdentityJob `yaml:"jobs"`
			}
			require.NoError(t, yaml.Unmarshal([]byte(rendered.String()), &parsed))
			publisher, ok := parsed.Jobs["safe_outputs"]
			require.True(t, ok)
			assertPublishingStepIdentity(t, publisher, "Process Safe Outputs")
			assert.Equal(t, 1, strings.Count(rendered.String(), "GH_AW_JOB_CHECK_RUN_ID:"),
				"only the trusted publishing step should receive the check-run identity")
		})
	}
}

func TestSafeOutputsReplayPublishingJobIdentity(t *testing.T) {
	opts := buildMaintenanceWorkflowYAMLOptions{runsOnValue: "ubuntu-latest"}
	labelJob, err := buildMaintenanceLabelApplySafeOutputsJob(opts, "./actions/setup")
	require.NoError(t, err)
	for _, tc := range []struct {
		name       string
		jobKey     string
		stepName   string
		workflowID string
		content    string
	}{
		{
			name:       "maintenance dispatch",
			jobKey:     "apply_safe_outputs",
			stepName:   "Apply Safe Outputs",
			workflowID: "agentics-maintenance",
			content:    buildMaintenanceApplySafeOutputsJob(opts, "./actions/setup"),
		},
		{
			name:       "maintenance label",
			jobKey:     "label_apply_safe_outputs",
			stepName:   "Apply safe outputs from referenced run",
			workflowID: "agentics-maintenance",
			content:    labelJob,
		},
		{
			name:       "side repository",
			jobKey:     "apply_safe_outputs",
			stepName:   "Apply Safe Outputs",
			workflowID: "agentics-maintenance-owner-target",
			content: "  apply_safe_outputs:\n    steps:\n" + buildApplySafeOutputsStep(sideRepoMaintenanceRenderContext{
				repoSlug: "owner/target",
				token:    "${{ secrets.GITHUB_TOKEN }}",
			}),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var parsed struct {
				Jobs map[string]publishingIdentityJob `yaml:"jobs"`
			}
			require.NoError(t, yaml.Unmarshal([]byte("jobs:\n"+tc.content), &parsed))
			job, ok := parsed.Jobs[tc.jobKey]
			require.True(t, ok)
			assertPublishingStepIdentity(t, job, tc.stepName)
			for _, step := range job.Steps {
				if step.Name == tc.stepName {
					assert.Equal(t, tc.workflowID, step.Env["GH_AW_WORKFLOW_ID"])
				}
			}
			assert.Equal(t, 1, strings.Count(tc.content, "GH_AW_JOB_CHECK_RUN_ID:"))
		})
	}
}

func TestCustomSafeOutputsPublishingJobIdentity(t *testing.T) {
	compiler := NewCompiler()
	data := &WorkflowData{
		Name: "Reusable Custom Publisher",
		On:   "on:\n  workflow_call:",
		SafeOutputs: &SafeOutputsConfig{
			Jobs: map[string]*SafeJobConfig{
				"custom-publisher": {
					Permissions: map[string]string{"issues": "write"},
					Env:         map[string]string{"GH_AW_JOB_CHECK_RUN_ID": "not-a-runtime-identity"},
					Steps: []any{
						map[string]any{
							"name": "Create issue with stable API",
							"uses": "actions/github-script@v8",
							"env":  map[string]any{"GH_AW_JOB_CHECK_RUN_ID": "${{ github.job }}"},
							"with": map[string]any{
								"script": "const { setupGlobals, createIssue } = require(process.env.RUNNER_TEMP + '/gh-aw/actions/index.cjs');\nsetupGlobals(core, github, context, exec, io, getOctokit);\nawait createIssue({ title: 'Custom output', body: 'Trusted publisher' });",
							},
						},
					},
				},
			},
		},
	}
	names, err := compiler.buildSafeJobs(data, false)
	require.NoError(t, err)
	require.Len(t, names, 1)
	job, ok := compiler.jobManager.GetJob(names[0])
	require.True(t, ok)
	assert.NotContains(t, job.Env, "GH_AW_JOB_CHECK_RUN_ID")
	var parsed publishingIdentityJob
	require.NoError(t, yaml.Unmarshal([]byte("steps:\n"+strings.Join(job.Steps, "")), &parsed))
	assertPublishingStepIdentity(t, parsed, "Create issue with stable API")
}
