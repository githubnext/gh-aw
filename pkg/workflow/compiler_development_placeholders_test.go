//go:build !integration

package workflow

import (
	"io"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/workflow/actions"
	"github.com/github/gh-aw/pkg/workflow/compilerenv"
	"github.com/goccy/go-yaml"
	"github.com/rhysd/actionlint"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDryRunPlaceholderReusableOutputContract(t *testing.T) {
	for _, dryRun := range []bool{false, true} {
		t.Run(map[bool]string{false: "normal", true: "dry-run"}[dryRun], func(t *testing.T) {
			c := NewCompiler()
			c.SetDryRun(dryRun)
			c.repoConfigLoaded = true
			producer := &Job{
				Name: "prepare", Uses: "owner/repo/.github/workflows/prepare.yml@main",
				SecretsInherit: true,
			}
			consumer := &Job{
				Name: "consumer", RunsOn: "runs-on: ubuntu-latest", Needs: []string{"prepare"},
				Steps: []string{"      - run: echo '${{ needs.prepare.outputs.value }}'\n"},
				Env: map[string]string{
					"BRACKET_OUTPUT": "${{ needs['prepare']['outputs']['bracket-value'] }}",
				},
				Actions: &actions.Job{
					Env: map[string]actions.Expression{
						"TYPED_OUTPUT": actions.OutputRef{Job: "prepare", Name: "typed"},
					},
				},
			}
			require.NoError(t, c.jobManager.AddJob(producer))
			require.NoError(t, c.jobManager.AddJob(consumer))
			data := &WorkflowData{
				On:   "workflow_call:\n  outputs:\n    result:\n      value: ${{ jobs.prepare.outputs.workflow_result }}\n",
				Jobs: map[string]any{"prepare": nil},
			}
			c.disableDryRunCustomJobs(data)
			if !dryRun {
				assert.NotEmpty(t, producer.Uses)
				assert.Nil(t, producer.Outputs)
				assert.True(t, producer.SecretsInherit)
				return
			}
			assert.Equal(t, map[string]string{
				"value": "${{ '' }}", "bracket-value": "${{ '' }}", "typed": "${{ '' }}",
				"workflow_result": "${{ '' }}",
			}, producer.Outputs)
			assert.Empty(t, producer.Uses)
			assert.False(t, producer.SecretsInherit)
			require.NoError(t, c.jobManager.resolveActions(), "typed consumers must retain a declared producer output")
			var rendered strings.Builder
			c.jobManager.WriteJobsYAML(&rendered)
			_, err := actions.Parse([]byte("on: workflow_dispatch\n" + rendered.String()))
			require.NoError(t, err)
			linter, err := actionlint.NewLinter(io.Discard, &actionlint.LinterOptions{})
			require.NoError(t, err)
			diagnostics, err := linter.Lint("workflow.yml", []byte("on: workflow_dispatch\n"+rendered.String()), nil)
			require.NoError(t, err)
			assert.Empty(t, diagnostics)
		})
	}
}

func TestDryRunPlaceholderPreservesTypedDeclarations(t *testing.T) {
	job := &Job{
		Name: "prepare", Needs: []string{"activation"},
		Outputs: map[string]string{"legacy": "${{ steps.run.outputs.value }}"},
		Actions: &actions.Job{
			Needs:       []actions.JobID{"activation", "other"},
			Outputs:     map[actions.OutputName]actions.Expression{"typed": actions.Secret("TOKEN")},
			CallOutputs: []actions.OutputDecl{{Name: "callee"}},
		},
	}
	makeDryRunJobInert(job, "runs-on: self-hosted")
	assert.Equal(t, []string{"activation", "other"}, job.Needs)
	assert.Equal(t, map[string]string{
		"legacy": "${{ '' }}", "typed": "${{ '' }}", "callee": "${{ '' }}",
	}, job.Outputs)
	assert.Nil(t, job.Actions)
	assert.Equal(t, "permissions: {}", job.Permissions)
}

func TestDryRunPlaceholderStaticRunner(t *testing.T) {
	for _, test := range []struct {
		name, runner, workflowRunner, slimRunner, safeRunner, want string
	}{
		{"static", "runs-on: [self-hosted, linux]", "", "", "", "runs-on: [self-hosted, linux]"},
		{"matrix", "runs-on: ${{ matrix.runner }}", "runs-on: [self-hosted, linux]", "", "", "runs-on: [self-hosted, linux]"},
		{"slim", "runs-on: [self-hosted, '${{ matrix.os }}']", "", "runs-on: self-hosted", "", "runs-on: self-hosted"},
		{"safe-output", "", "", "", "runs-on:\n  group: custom-pool", "runs-on:\n      group: custom-pool"},
		{"default", "runs-on: ${{ matrix.runner }}", "", "", "", "runs-on: ubuntu-slim"},
	} {
		t.Run(test.name, func(t *testing.T) {
			c := NewCompiler()
			c.repoConfigLoaded = true
			t.Setenv(compilerenv.RequireSelfHostedRunners, "")
			data := &WorkflowData{RunsOn: test.workflowRunner, RunsOnSlim: test.slimRunner}
			if test.safeRunner != "" {
				data.SafeOutputs = &SafeOutputsConfig{RunsOn: test.safeRunner}
			}
			assert.Equal(t, test.want, c.dryRunPlaceholderRunsOn(data, &Job{RunsOn: test.runner}))
		})
	}
}

func TestDryRunPlaceholderSelfHostedPolicy(t *testing.T) {
	for _, policy := range []string{"flag", "env", "repo"} {
		t.Run(policy, func(t *testing.T) {
			t.Setenv(compilerenv.RequireSelfHostedRunners, "")
			c := NewCompiler()
			c.SetDryRun(true)
			c.repoConfigLoaded = true
			switch policy {
			case "flag":
				c.SetRequireSelfHostedRunners(true)
			case "env":
				t.Setenv(compilerenv.RequireSelfHostedRunners, "true")
			case "repo":
				c.repoConfig = &RepoConfig{RequireSelfHostedRunners: true}
			}

			data := &WorkflowData{
				Jobs: map[string]any{"prepare": nil},
				On:   "workflow_call:\n  outputs:\n    result:\n      value: ${{ jobs.call-worker.outputs.result }}\n",
				SafeOutputs: &SafeOutputsConfig{CallWorkflow: &CallWorkflowConfig{
					Workflows: []string{"worker"},
				}},
			}
			custom := &Job{Name: "prepare", RunsOn: "runs-on: ${{ matrix.runner }}", Strategy: "strategy:\n  matrix:\n    runner: [self-hosted]"}
			require.NoError(t, c.jobManager.AddJob(custom))
			managed := c.buildCallWorkflowJob(data, "worker", "")
			assert.Equal(t, map[string]string{"result": "${{ '' }}"}, managed.Outputs)
			require.NoError(t, c.jobManager.AddJob(managed))
			c.disableDryRunCustomJobs(data)
			for _, name := range []string{"prepare", "call-worker"} {
				job, ok := c.jobManager.GetJob(name)
				require.True(t, ok)
				assert.Equal(t, "runs-on: self-hosted", job.RunsOn)
				assert.Empty(t, job.Strategy)
				assert.Empty(t, job.Uses)
				assert.Empty(t, job.Secrets)
				assert.False(t, job.SecretsInherit)
				assert.Equal(t, []string{"safe_outputs"}, managed.Needs)
				var runner map[string]any
				require.NoError(t, yaml.Unmarshal([]byte(job.RunsOn), &runner))
				require.NoError(t, validateSelfHostedRunners(map[string]any{
					"jobs": map[string]any{name: runner},
				}))
			}
		})
	}
}

func TestDryRunPlaceholderCompilationSelfHosted(t *testing.T) {
	const source = `---
on: workflow_dispatch
engine: copilot
permissions:
  contents: read
runs-on: self-hosted
runs-on-slim: self-hosted
jobs:
  prepare:
    uses: owner/repo/.github/workflows/prepare.yml@main
    secrets: inherit
  cleanup:
    needs: [prepare]
    runs-on: ${{ matrix.runner }}
    strategy:
      matrix:
        runner: [self-hosted]
    steps:
      - run: echo unsafe
---
Analyze the repository using ${{ needs.prepare.outputs.value }}.
`
	for _, policy := range []string{"flag", "env", "repo"} {
		t.Run(policy, func(t *testing.T) {
			t.Setenv(compilerenv.RequireSelfHostedRunners, "")
			c := NewCompiler(WithNoEmit(true), WithSkipValidation(true))
			c.SetDryRun(true)
			c.repoConfigLoaded = true
			c.repoConfig = &RepoConfig{}
			switch policy {
			case "flag":
				c.SetRequireSelfHostedRunners(true)
			case "env":
				t.Setenv(compilerenv.RequireSelfHostedRunners, "true")
			case "repo":
				c.repoConfig = &RepoConfig{RequireSelfHostedRunners: true}
			}
			data, err := c.ParseWorkflowString(source, "placeholder.md")
			require.NoError(t, err)
			compiled, err := c.CompileToYAML(data, "placeholder.md")
			require.NoError(t, err)
			var workflow map[string]any
			require.NoError(t, yaml.Unmarshal([]byte(compiled), &workflow))
			require.NoError(t, validateSelfHostedRunners(workflow))
			jobs := workflow["jobs"].(map[string]any)
			prepare := jobs["prepare"].(map[string]any)
			assert.Equal(t, map[string]any{"value": "${{ '' }}"}, prepare["outputs"])
			assert.NotContains(t, compiled, "echo unsafe")
			assert.NotContains(t, compiled, "secrets: inherit")
			assert.NotContains(t, compiled, "${{ matrix.runner }}")
			assert.NotContains(t, compiled, "owner/repo/.github/workflows/prepare.yml")
		})
	}
}
