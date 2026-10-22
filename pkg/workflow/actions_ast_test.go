package workflow

import (
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/workflow/actions"
	"github.com/goccy/go-yaml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestActionsASTDerivesAndLowers(t *testing.T) {
	jm := NewJobManager()
	require.NoError(t, jm.AddJob(&Job{
		Name: "setup", RunsOn: "runs-on: ubuntu-latest",
		Steps: []string{"      - name: Existing step\n        run: echo unchanged # keep comment\n"},
		Actions: &actions.Job{Steps: []actions.Step{actions.RunStep{
			StepOptions: actions.StepOptions{ID: "produce", Outputs: []actions.OutputDecl{{Name: "value"}}},
			Run:         "echo 'value=ready' >> \"$GITHUB_OUTPUT\"",
		}}},
	}))
	require.NoError(t, jm.AddJob(&Job{
		Name: "agent", RunsOn: "runs-on: ubuntu-latest",
		Actions: &actions.Job{Steps: []actions.Step{actions.ActionStep{
			StepOptions: actions.StepOptions{
				Name: "Consume",
				Env: map[string]actions.Expression{
					"VALUE": actions.OutputRef{Job: "setup", Step: "produce", Name: "value"},
				},
			},
			Uses: "owner/action@sha # v1",
		}}},
	}))
	require.NoError(t, jm.resolveActions())
	agent, _ := jm.GetJob("agent")
	assert.Equal(t, []string{"setup"}, agent.Needs)
	setup, _ := jm.GetJob("setup")
	assert.Equal(t, `"${{ steps.produce.outputs.value }}"`, setup.Outputs["aw_7_produce_value"])
	content := renderActionsASTJobs(jm)
	assert.Contains(t, content, "echo unchanged # keep comment")
	assert.Contains(t, content, "${{ needs.setup.outputs.aw_7_produce_value }}")
	assert.Contains(t, content, "uses: owner/action@sha # v1")
	require.NoError(t, jm.resolveActions())
	assert.Equal(t, content, renderActionsASTJobs(jm), "lowering is idempotent")
}

func TestActionsASTLegacyOutputAndCondition(t *testing.T) {
	jm := NewJobManager()
	require.NoError(t, jm.AddJob(&Job{Name: "safe_outputs", Outputs: map[string]string{
		"worker": "${{ steps.process.outputs.worker }}",
	}}))
	require.NoError(t, jm.AddJob(&Job{
		Name: "call_worker", Uses: "./.github/workflows/worker.yml",
		Actions: &actions.Job{If: actions.Binary{
			Left:     actions.OutputRef{Job: "safe_outputs", Name: "worker"},
			Operator: "==", Right: actions.Literal("worker"),
		}},
	}))
	require.NoError(t, jm.resolveActions())
	job, _ := jm.GetJob("call_worker")
	assert.Equal(t, []string{"safe_outputs"}, job.Needs)
	assert.Equal(t, "${{ (needs.safe_outputs.outputs.worker == 'worker') }}", job.If)
}

func TestActionsASTSerializesConditionsAsSafeYAMLScalars(t *testing.T) {
	tests := []struct {
		name      string
		condition actions.Expression
	}{
		{
			name: "colon literal",
			condition: actions.Binary{
				Left: actions.Literal("build: release"), Operator: "==",
				Right: actions.Literal("build: release"),
			},
		},
		{
			name: "hash literal",
			condition: actions.Binary{
				Left: actions.Literal("value # comment"), Operator: "==",
				Right: actions.Literal("value # comment"),
			},
		},
		{
			name: "long false condition",
			condition: actions.Binary{
				Left: actions.Binary{
					Left: actions.Literal(strings.Repeat("worker", 22)), Operator: "==",
					Right: actions.Literal("selected"),
				},
				Operator: "&&", Right: actions.Boolean(false),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jm := NewJobManager()
			require.NoError(t, jm.AddJob(&Job{
				Name: "call", Uses: "./worker.yml", SecretsInherit: true,
				Actions: &actions.Job{If: tt.condition},
			}))
			require.NoError(t, jm.resolveActions())
			job, _ := jm.GetJob("call")

			content := renderActionsASTJobs(jm)
			var parsed struct {
				Jobs map[string]struct {
					If string `yaml:"if"`
				} `yaml:"jobs"`
			}
			require.NoError(t, yaml.Unmarshal([]byte(content), &parsed))
			assert.Equal(t, job.If, parsed.Jobs["call"].If)
			assert.NotContains(t, parsed.Jobs["call"].If, "\n")
		})
	}
}

func TestActionsASTFailureIsAtomic(t *testing.T) {
	jm := NewJobManager()
	original := &Job{Name: "build", Actions: &actions.Job{
		Steps: []actions.Step{actions.RunStep{Run: "echo first"}, actions.ActionStep{}},
	}}
	require.NoError(t, jm.AddJob(original))
	require.ErrorContains(t, jm.resolveActions(), "nonempty uses")
	current, _ := jm.GetJob("build")
	assert.Same(t, original, current)
	assert.Empty(t, current.Steps)
	assert.NotNil(t, current.Actions)
}

func TestActionsASTPreservesLegacyYAML(t *testing.T) {
	jm := NewJobManager()
	require.NoError(t, jm.AddJob(&Job{
		Name: "build", RunsOn: "runs-on: ubuntu-latest",
		Steps:   []string{"      - run: echo test # comment\n"},
		Outputs: map[string]string{"value": "${{ steps.build.outputs.value }}"},
	}))
	before := renderActionsASTJobs(jm)
	require.NoError(t, jm.resolveActions())
	assert.Equal(t, before, renderActionsASTJobs(jm))
}

func TestActionsASTQuotesValuesAndCollectsSecrets(t *testing.T) {
	jm := NewJobManager()
	require.NoError(t, jm.AddJob(&Job{Name: "build", RunsOn: "runs-on: ubuntu-latest", Actions: &actions.Job{
		Env: map[string]actions.Expression{
			"MULTILINE": actions.Literal("one\ntwo\n"),
			"BOOLEAN":   actions.Literal("true"),
			"TOKEN":     actions.Secret("API_TOKEN"),
			"MESSAGE":   actions.Concat(actions.Literal("key: value # not a comment"), actions.Variable("NAME")),
		},
		Outputs: map[actions.OutputName]actions.Expression{
			"message": actions.Literal("key: value\nand more"),
		},
		Steps: []actions.Step{actions.RunStep{Run: "echo test"}},
	}}))
	require.NoError(t, jm.resolveActions())
	content := renderActionsASTJobs(jm)
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

func TestActionsASTRejectsConflictingContributions(t *testing.T) {
	tests := []struct {
		name string
		job  *Job
		want string
	}{
		{"id", &Job{Name: "build", Actions: &actions.Job{ID: "other"}}, "does not match"},
		{"native field", &Job{Name: "build", Actions: &actions.Job{RunsOn: "ubuntu-latest"}}, "enclosing job"},
		{"condition", &Job{Name: "build", If: "always()", Actions: &actions.Job{If: actions.Boolean(true)}}, "both a legacy and typed job condition"},
		{"env", &Job{Name: "build", Env: map[string]string{"VALUE": "existing"}, Actions: &actions.Job{
			Env: map[string]actions.Expression{"VALUE": actions.Literal("typed")},
		}}, "both a legacy and typed environment"},
		{"output", &Job{Name: "build", Outputs: map[string]string{"value": "existing"}, Actions: &actions.Job{
			Outputs: map[actions.OutputName]actions.Expression{"value": actions.Literal("typed")},
		}}, "both legacy and typed output"},
		{"reusable env", &Job{Name: "build", Uses: "./worker.yml", Actions: &actions.Job{
			Env: map[string]actions.Expression{"VALUE": actions.Literal("typed")},
		}}, "cannot declare typed steps, env, or outputs"},
		{"reusable step", &Job{Name: "build", Uses: "./worker.yml", Actions: &actions.Job{
			Steps: []actions.Step{actions.RunStep{Run: "echo test"}},
		}}, "cannot declare typed steps, env, or outputs"},
		{"reusable output", &Job{Name: "build", Uses: "./worker.yml", Actions: &actions.Job{
			Outputs: map[actions.OutputName]actions.Expression{"result": actions.Literal("ready")},
		}}, "cannot declare typed steps, env, or outputs"},
		{"runner missing runs-on", &Job{Name: "build", Actions: &actions.Job{
			Steps: []actions.Step{actions.RunStep{Run: "echo test"}},
		}}, "runner job requires runs-on"},
		{"reusable with runs-on", &Job{Name: "build", Uses: "./worker.yml", RunsOn: "runs-on: ubuntu-latest",
			Actions: &actions.Job{If: actions.Boolean(true)}}, "reusable workflow job cannot declare runner properties"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jm := NewJobManager()
			require.NoError(t, jm.AddJob(tt.job))
			require.ErrorContains(t, jm.resolveActions(), tt.want)
		})
	}
}

func renderActionsASTJobs(jm *JobManager) string {
	var b strings.Builder
	jm.WriteJobsYAML(&b)
	return b.String()
}
