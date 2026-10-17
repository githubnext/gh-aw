package pipeline

import (
	"strings"
	"testing"

	"github.com/goccy/go-yaml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func outputPipeline() *Pipeline {
	return &Pipeline{
		Name: "Typed workflow",
		Envelope: map[string]any{
			"on":          map[string]any{"workflow_dispatch": map[string]any{}},
			"permissions": map[string]any{},
		},
		Jobs: []*Job{
			{
				ID: "setup", RunsOn: "ubuntu-latest",
				Steps: []Step{RunStep{
					StepOptions: StepOptions{ID: "produce", Outputs: []OutputDecl{{Name: "value"}}},
					Script:      "printf 'value=ready\\n' >> \"$GITHUB_OUTPUT\"\n",
				}},
			},
			{
				ID: "agent", RunsOn: []string{"self-hosted", "linux"},
				Steps: []Step{ActionStep{
					StepOptions: StepOptions{
						ID: "consume", Name: "Read output",
						Condition: Binary{
							Left:     OutputRef{Job: "setup", Step: "produce", Name: "value"},
							Operator: "==", Right: Literal("ready"),
						},
						Env: map[string]Expression{
							"VALUE": Coalesce(OutputRef{Job: "setup", Step: "produce", Name: "value"}, Literal("fallback")),
							"TOKEN": Secret("API_TOKEN"),
						},
					},
					Uses: "owner/action@0123456789",
					With: map[string]Expression{
						"message": Concat(Literal("value: "), OutputRef{Job: "setup", Step: "produce", Name: "value"}),
					},
				}},
			},
		},
	}
}

func TestEmitRoundTrip(t *testing.T) {
	input := outputPipeline()
	content, err := Emit(input)
	require.NoError(t, err)
	var actual map[string]any
	require.NoError(t, yaml.Unmarshal([]byte(content), &actual))
	var expected map[string]any
	require.NoError(t, yaml.Unmarshal([]byte(`
name: Typed workflow
on:
  workflow_dispatch: {}
permissions: {}
jobs:
  setup:
    runs-on: ubuntu-latest
    outputs:
      aw_7_produce_value: ${{ steps.produce.outputs.value }}
    steps:
      - id: produce
        run: |
          printf 'value=ready\n' >> "$GITHUB_OUTPUT"
  agent:
    runs-on: [self-hosted, linux]
    needs: [setup]
    steps:
      - id: consume
        name: Read output
        uses: owner/action@0123456789
        if: ${{ (needs.setup.outputs.aw_7_produce_value == 'ready') }}
        env:
          VALUE: ${{ (needs.setup.outputs.aw_7_produce_value || 'fallback') }}
          TOKEN: ${{ secrets.API_TOKEN }}
        with:
          message: "${{ format('{0}{1}', 'value: ', needs.setup.outputs.aw_7_produce_value) }}"
`), &expected))
	assert.Equal(t, expected, actual)
	assert.Empty(t, input.Jobs[0].Outputs, "resolution must not mutate the producer")
	assert.Empty(t, input.Jobs[1].Needs, "resolution must not mutate the consumer")
	repeated, err := Emit(input)
	require.NoError(t, err)
	assert.Equal(t, content, repeated)
}

func TestSameJobOutputsAndScopedIDs(t *testing.T) {
	input := outputPipeline()
	input.Jobs[1].Steps = []Step{RunStep{
		StepOptions: StepOptions{ID: "produce"},
		Script:      "echo unrelated",
	}}
	input.Jobs[0].Steps = append(input.Jobs[0].Steps, RunStep{
		StepOptions: StepOptions{
			Env: map[string]Expression{"VALUE": OutputRef{Job: "setup", Step: "produce", Name: "value"}},
		},
		Script: "echo \"$VALUE\"",
	})
	content, err := Emit(input)
	require.NoError(t, err)
	assert.Contains(t, content, "${{ steps.produce.outputs.value }}")
	assert.NotContains(t, content, "needs:")
	assert.NotContains(t, content, "outputs:\n", "same-job output does not need a job binding")
}

func TestResolvePreservesExplicitEdgesAndBindings(t *testing.T) {
	input := outputPipeline()
	input.Jobs[0].Outputs = map[OutputName]Expression{
		"public": OutputRef{Job: "setup", Step: "produce", Name: "value"},
	}
	input.Jobs[1].Needs = []JobID{"setup", "setup"}
	input.Jobs[1].Env = map[string]Expression{"PUBLIC": OutputRef{Job: "setup", Name: "public"}}
	resolved, err := Resolve(input)
	require.NoError(t, err)
	assert.Equal(t, []JobID{"setup"}, resolved.Jobs[1].Needs)
	assert.Len(t, resolved.Jobs[0].Outputs, 1, "reuse explicit output binding instead of generating a second alias")
	rendered, err := RenderExpression(OutputRef{Job: "setup", Step: "produce", Name: "value"}, "agent", resolved, true)
	require.NoError(t, err)
	assert.Equal(t, "${{ needs.setup.outputs.public }}", rendered)
}

func TestResolveInvalidGraphs(t *testing.T) {
	tests := []struct {
		name   string
		change func(*Pipeline)
		want   string
	}{
		{"duplicate job", func(p *Pipeline) { p.Jobs = append(p.Jobs, p.Jobs[0]) }, "duplicate pipeline job"},
		{"invalid job", func(p *Pipeline) { p.Jobs[0].ID = "bad.job" }, "invalid job id"},
		{"nil job", func(p *Pipeline) { p.Jobs = append(p.Jobs, nil) }, "job cannot be nil"},
		{"nil step", func(p *Pipeline) { p.Jobs[0].Steps = append(p.Jobs[0].Steps, nil) }, "step cannot be nil"},
		{"duplicate step", func(p *Pipeline) { p.Jobs[0].Steps = append(p.Jobs[0].Steps, p.Jobs[0].Steps[0]) }, "duplicate pipeline step"},
		{"duplicate raw step", func(p *Pipeline) {
			p.Jobs[0].Steps = append(p.Jobs[0].Steps, RawStep{Fields: map[string]any{"id": "produce", "run": "echo test"}})
		}, "duplicate pipeline step"},
		{"invalid raw step id", func(p *Pipeline) {
			p.Jobs[0].Steps = append(p.Jobs[0].Steps, RawStep{Fields: map[string]any{"id": 1, "run": "echo test"}})
		}, "id must be a string"},
		{"unknown needs", func(p *Pipeline) { p.Jobs[0].Needs = []JobID{"missing"} }, "non-existent job"},
		{"explicit cycle", func(p *Pipeline) { p.Jobs[0].Needs = []JobID{"agent"} }, "cycle detected"},
		{"self cycle", func(p *Pipeline) { p.Jobs[0].Needs = []JobID{"setup"} }, "cycle detected"},
		{"missing producer", func(p *Pipeline) { p.Jobs = p.Jobs[1:] }, "unknown producer job"},
		{"unknown step output", func(p *Pipeline) {
			p.Jobs[1].Env = map[string]Expression{"VALUE": OutputRef{Job: "setup", Step: "produce", Name: "missing"}}
		}, "undeclared step output"},
		{"unknown job output", func(p *Pipeline) {
			p.Jobs[1].Env = map[string]Expression{"VALUE": OutputRef{Job: "setup", Name: "missing"}}
		}, "unknown output"},
		{"same job output", func(p *Pipeline) {
			p.Jobs[0].Outputs = map[OutputName]Expression{"public": Literal("value")}
			p.Jobs[0].Env = map[string]Expression{"VALUE": OutputRef{Job: "setup", Name: "public"}}
		}, "cannot consume its own job output"},
		{"anonymous output", func(p *Pipeline) {
			p.Jobs[0].Steps = []Step{RunStep{StepOptions: StepOptions{Outputs: []OutputDecl{{Name: "value"}}}, Script: "echo test"}}
		}, "anonymous step"},
		{"duplicate output", func(p *Pipeline) {
			step := p.Jobs[0].Steps[0].(RunStep)
			step.Outputs = append(step.Outputs, step.Outputs[0])
			p.Jobs[0].Steps[0] = step
		}, "duplicate output"},
		{"forward reference", func(p *Pipeline) {
			p.Jobs[0].Steps = append([]Step{RunStep{
				StepOptions: StepOptions{Env: map[string]Expression{"VALUE": OutputRef{Job: "setup", Step: "produce", Name: "value"}}},
				Script:      "echo \"$VALUE\"",
			}}, p.Jobs[0].Steps...)
		}, "before it runs"},
		{"self step reference", func(p *Pipeline) {
			step := p.Jobs[0].Steps[0].(RunStep)
			step.Env = map[string]Expression{"VALUE": OutputRef{Job: "setup", Step: "produce", Name: "value"}}
			p.Jobs[0].Steps[0] = step
		}, "before it runs"},
		{"job condition reads local step", func(p *Pipeline) {
			p.Jobs[0].Condition = OutputRef{Job: "setup", Step: "produce", Name: "value"}
		}, "before it runs"},
		{"binding collision", func(p *Pipeline) {
			p.Jobs[0].Outputs = map[OutputName]Expression{"aw_7_produce_value": Literal("other")}
		}, "conflicts with an explicit output"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := outputPipeline()
			tt.change(input)
			_, err := Resolve(input)
			require.ErrorContains(t, err, tt.want)
		})
	}
	_, err := Resolve(nil)
	require.ErrorContains(t, err, "cannot be nil")
}

func TestNestedReferencesDeriveEdges(t *testing.T) {
	input := outputPipeline()
	input.Jobs[1].Steps = nil
	input.Jobs[1].Condition = Not{Value: Binary{
		Left: Call{Name: "contains", Args: []Expression{
			Concat(Variable("PREFIX"), OutputRef{Job: "setup", Step: "produce", Name: "value"}),
			Literal("don't"),
		}},
		Operator: "==", Right: Boolean(false),
	}}
	resolved, err := Resolve(input)
	require.NoError(t, err)
	assert.Equal(t, []JobID{"setup"}, resolved.Jobs[1].Needs)
	value, err := RenderExpression(input.Jobs[1].Condition, "agent", resolved, true)
	require.NoError(t, err)
	assert.Contains(t, value, "'don''t'")
	assert.Contains(t, value, "vars.PREFIX")
	assert.Contains(t, value, "needs.setup.outputs.aw_7_produce_value")
}

func TestLowerRejectsInvalidShapes(t *testing.T) {
	tests := []struct {
		name   string
		change func(*Pipeline)
		want   string
	}{
		{"envelope", func(p *Pipeline) { p.Envelope["jobs"] = "override" }, "cannot override"},
		{"job field", func(p *Pipeline) { p.Jobs[0].Fields = map[string]any{"steps": "override"} }, "cannot override"},
		{"reusable", func(p *Pipeline) { p.Jobs[0].Fields = map[string]any{"uses": "./worker.yml"} }, "cannot declare"},
		{"empty run", func(p *Pipeline) { p.Jobs[1].Steps = []Step{RunStep{}} }, "nonempty script"},
		{"run interpolation", func(p *Pipeline) {
			p.Jobs[1].Steps = []Step{RunStep{Script: `echo "${{ github.event.issue.title }}"`}}
		}, "pass Actions expressions through env"},
		{"empty action", func(p *Pipeline) { p.Jobs[1].Steps = []Step{ActionStep{}} }, "nonempty uses"},
		{"empty raw", func(p *Pipeline) { p.Jobs[1].Steps = []Step{RawStep{}} }, "cannot be empty"},
		{"negative timeout", func(p *Pipeline) {
			p.Jobs[1].Steps = []Step{RunStep{StepOptions: StepOptions{TimeoutMinutes: -1}, Script: "echo test"}}
		}, "cannot be negative"},
		{"raw multiline expression", func(p *Pipeline) { p.Jobs[1].Condition = Context("always()\nfoo") }, "single-line"},
		{"raw delimiters", func(p *Pipeline) { p.Jobs[1].Condition = Context("${{ always() }}") }, "exclude expression delimiters"},
		{"invalid operator", func(p *Pipeline) {
			p.Jobs[1].Condition = Binary{Left: Boolean(true), Operator: ";", Right: Boolean(false)}
		}, "unsupported pipeline expression operator"},
		{"nil expression", func(p *Pipeline) { p.Jobs[1].Env = map[string]Expression{"VALUE": nil} }, "cannot be nil"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := outputPipeline()
			tt.change(input)
			_, err := Emit(input)
			require.ErrorContains(t, err, tt.want)
		})
	}
}

func TestRawStepPreservesNativeFields(t *testing.T) {
	input := &Pipeline{Name: "raw", Jobs: []*Job{{
		ID: "build", RunsOn: "ubuntu-latest",
		Steps: []Step{RawStep{Fields: map[string]any{
			"uses":            "owner/action@sha",
			"with":            map[string]any{"number": 3, "flag": true},
			"timeout-minutes": "${{ fromJSON(vars.TIMEOUT) }}",
		}}},
	}}}
	content, err := Emit(input)
	require.NoError(t, err)
	assert.Contains(t, content, "number: 3")
	assert.Contains(t, content, "flag: true")
	assert.Contains(t, content, "${{ fromJSON(vars.TIMEOUT) }}")
}

func TestIdentifierGrammar(t *testing.T) {
	for _, value := range []string{"agent", "_step", "safe-outputs", "output_1"} {
		_, err := NewJobID(value)
		require.NoError(t, err)
		_, err = NewStepID(value)
		require.NoError(t, err)
		_, err = NewOutputName(value)
		require.NoError(t, err)
	}
	for _, value := range []string{"", "1job", "job.name", "job name", "job\ninjected", "job:"} {
		_, err := NewJobID(value)
		require.Error(t, err)
		_, err = NewStepID(value)
		require.Error(t, err)
		_, err = NewOutputName(value)
		require.Error(t, err)
	}
}

func TestDistinctOutputAliases(t *testing.T) {
	first := &Job{ID: "producer", Outputs: make(map[OutputName]Expression)}
	a, err := exposeOutput(first, OutputRef{Job: "producer", Step: "a_b", Name: "c"})
	require.NoError(t, err)
	b, err := exposeOutput(first, OutputRef{Job: "producer", Step: "a", Name: "b_c"})
	require.NoError(t, err)
	assert.NotEqual(t, a, b)
	assert.True(t, strings.HasPrefix(string(a), "aw_"))
}
