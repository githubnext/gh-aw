package workflow

import (
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/workflow/pipeline"
	"github.com/goccy/go-yaml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPipelineBridgeDerivesAndLowers(t *testing.T) {
	jm := NewJobManager()
	require.NoError(t, jm.AddJob(&Job{
		Name: "setup", RunsOn: "runs-on: ubuntu-latest",
		Steps: []string{"      - name: Existing step\n        run: echo unchanged # keep comment\n"},
		Pipeline: &pipeline.Job{Steps: []pipeline.Step{pipeline.RunStep{
			StepOptions: pipeline.StepOptions{ID: "produce", Outputs: []pipeline.OutputDecl{{Name: "value"}}},
			Script:      "echo 'value=ready' >> \"$GITHUB_OUTPUT\"",
		}}},
	}))
	require.NoError(t, jm.AddJob(&Job{
		Name: "agent", RunsOn: "runs-on: ubuntu-latest",
		Pipeline: &pipeline.Job{Steps: []pipeline.Step{pipeline.ActionStep{
			StepOptions: pipeline.StepOptions{
				Name: "Consume",
				Env: map[string]pipeline.Expression{
					"VALUE": pipeline.OutputRef{Job: "setup", Step: "produce", Name: "value"},
				},
			},
			Uses: "owner/action@sha # v1",
		}}},
	}))
	require.NoError(t, jm.resolvePipeline())
	agent, _ := jm.GetJob("agent")
	assert.Equal(t, []string{"setup"}, agent.Needs)
	setup, _ := jm.GetJob("setup")
	assert.Equal(t, `"${{ steps.produce.outputs.value }}"`, setup.Outputs["aw_7_produce_value"])
	content := renderPipelineBridgeJobs(jm)
	assert.Contains(t, content, "echo unchanged # keep comment")
	assert.Contains(t, content, "${{ needs.setup.outputs.aw_7_produce_value }}")
	assert.Contains(t, content, "uses: owner/action@sha # v1")
	require.NoError(t, jm.resolvePipeline())
	assert.Equal(t, content, renderPipelineBridgeJobs(jm), "lowering is idempotent")
}

func TestPipelineBridgeLegacyOutputAndCondition(t *testing.T) {
	jm := NewJobManager()
	require.NoError(t, jm.AddJob(&Job{Name: "safe_outputs", Outputs: map[string]string{
		"worker": "${{ steps.process.outputs.worker }}",
	}}))
	require.NoError(t, jm.AddJob(&Job{
		Name: "call_worker", Uses: "./.github/workflows/worker.yml",
		Pipeline: &pipeline.Job{Condition: pipeline.Binary{
			Left:     pipeline.OutputRef{Job: "safe_outputs", Name: "worker"},
			Operator: "==", Right: pipeline.Literal("worker"),
		}},
	}))
	require.NoError(t, jm.resolvePipeline())
	job, _ := jm.GetJob("call_worker")
	assert.Equal(t, []string{"safe_outputs"}, job.Needs)
	assert.Equal(t, "${{ (needs.safe_outputs.outputs.worker == 'worker') }}", job.If)
}

func TestPipelineBridgeFailureIsAtomic(t *testing.T) {
	jm := NewJobManager()
	original := &Job{Name: "build", Pipeline: &pipeline.Job{
		Steps: []pipeline.Step{pipeline.RunStep{Script: "echo first"}, pipeline.ActionStep{}},
	}}
	require.NoError(t, jm.AddJob(original))
	require.ErrorContains(t, jm.resolvePipeline(), "nonempty uses")
	current, _ := jm.GetJob("build")
	assert.Same(t, original, current)
	assert.Empty(t, current.Steps)
	assert.NotNil(t, current.Pipeline)
}

func TestPipelineBridgePreservesLegacyYAML(t *testing.T) {
	jm := NewJobManager()
	require.NoError(t, jm.AddJob(&Job{
		Name: "build", RunsOn: "runs-on: ubuntu-latest",
		Steps:   []string{"      - run: echo test # comment\n"},
		Outputs: map[string]string{"value": "${{ steps.build.outputs.value }}"},
	}))
	before := renderPipelineBridgeJobs(jm)
	require.NoError(t, jm.resolvePipeline())
	assert.Equal(t, before, renderPipelineBridgeJobs(jm))
}

func TestPipelineBridgeQuotesValuesAndCollectsSecrets(t *testing.T) {
	jm := NewJobManager()
	require.NoError(t, jm.AddJob(&Job{Name: "build", Pipeline: &pipeline.Job{
		Env: map[string]pipeline.Expression{
			"MULTILINE": pipeline.Literal("one\ntwo\n"),
			"BOOLEAN":   pipeline.Literal("true"),
			"TOKEN":     pipeline.Secret("API_TOKEN"),
			"MESSAGE":   pipeline.Concat(pipeline.Literal("key: value # not a comment"), pipeline.Variable("NAME")),
		},
		Outputs: map[pipeline.OutputName]pipeline.Expression{
			"message": pipeline.Literal("key: value\nand more"),
		},
	}}))
	require.NoError(t, jm.resolvePipeline())
	content := renderPipelineBridgeJobs(jm)
	var parsed struct {
		Jobs map[string]struct {
			Env     map[string]string `yaml:"env"`
			Outputs map[string]string `yaml:"outputs"`
		} `yaml:"jobs"`
	}
	require.NoError(t, yaml.Unmarshal([]byte(content), &parsed))
	require.Contains(t, parsed.Jobs, "build")
	assert.Equal(t, "one\ntwo\n", parsed.Jobs["build"].Env["MULTILINE"])
	assert.Equal(t, "true", parsed.Jobs["build"].Env["BOOLEAN"])
	assert.Equal(t, "${{ secrets.API_TOKEN }}", parsed.Jobs["build"].Env["TOKEN"])
	assert.Equal(t, "key: value\nand more", parsed.Jobs["build"].Outputs["message"])
	assert.Contains(t, CollectSecretReferences(content), "API_TOKEN")
}

func TestPipelineBridgeRejectsConflictingContributions(t *testing.T) {
	tests := []struct {
		name string
		job  *Job
		want string
	}{
		{"id", &Job{Name: "build", Pipeline: &pipeline.Job{ID: "other"}}, "does not match"},
		{"native field", &Job{Name: "build", Pipeline: &pipeline.Job{RunsOn: "ubuntu-latest"}}, "enclosing job"},
		{"condition", &Job{Name: "build", If: "always()", Pipeline: &pipeline.Job{Condition: pipeline.Boolean(true)}}, "both a legacy and typed job condition"},
		{"env", &Job{Name: "build", Env: map[string]string{"VALUE": "existing"}, Pipeline: &pipeline.Job{
			Env: map[string]pipeline.Expression{"VALUE": pipeline.Literal("typed")},
		}}, "both a legacy and typed environment"},
		{"output", &Job{Name: "build", Outputs: map[string]string{"value": "existing"}, Pipeline: &pipeline.Job{
			Outputs: map[pipeline.OutputName]pipeline.Expression{"value": pipeline.Literal("typed")},
		}}, "both legacy and typed output"},
		{"reusable env", &Job{Name: "build", Uses: "./worker.yml", Pipeline: &pipeline.Job{
			Env: map[string]pipeline.Expression{"VALUE": pipeline.Literal("typed")},
		}}, "cannot declare typed steps or env"},
		{"reusable step", &Job{Name: "build", Uses: "./worker.yml", Pipeline: &pipeline.Job{
			Steps: []pipeline.Step{pipeline.RunStep{Script: "echo test"}},
		}}, "cannot declare typed steps or env"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jm := NewJobManager()
			require.NoError(t, jm.AddJob(tt.job))
			require.ErrorContains(t, jm.resolvePipeline(), tt.want)
		})
	}
}

func renderPipelineBridgeJobs(jm *JobManager) string {
	var b strings.Builder
	jm.WriteJobsYAML(&b)
	return b.String()
}
