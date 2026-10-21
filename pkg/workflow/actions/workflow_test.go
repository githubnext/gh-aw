package actions

import (
	"strings"
	"testing"

	"github.com/goccy/go-yaml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func outputWorkflow() *Workflow {
	return &Workflow{
		Name:        "Typed workflow",
		On:          map[string]any{"workflow_dispatch": map[string]any{}},
		Permissions: map[string]any{},
		Jobs: []*Job{
			{
				ID: "setup", RunsOn: "ubuntu-latest",
				Steps: []Step{RunStep{
					StepOptions: StepOptions{ID: "produce", Outputs: []OutputDecl{{Name: "value"}}},
					Run:         "printf 'value=ready\\n' >> \"$GITHUB_OUTPUT\"\n",
				}},
			},
			{
				ID: "agent", RunsOn: []string{"self-hosted", "linux"},
				Steps: []Step{ActionStep{
					StepOptions: StepOptions{
						ID: "consume", Name: "Read output",
						If: Binary{
							Left:     OutputRef{Job: "setup", Step: "produce", Name: "value"},
							Operator: "==", Right: Literal("ready"),
						},
						Env: map[string]Expression{
							"VALUE": Or(OutputRef{Job: "setup", Step: "produce", Name: "value"}, Literal("fallback")),
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
	input := outputWorkflow()
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

func TestRunsOnRoundTrip(t *testing.T) {
	tests := []struct {
		name   string
		source string
		value  any
	}{
		{"scalar", "ubuntu-latest", "ubuntu-latest"},
		{"single quoted scalar", "'ubuntu-latest'", "ubuntu-latest"},
		{"double quoted scalar", `"ubuntu-latest"`, "ubuntu-latest"},
		{"quoted special characters", "'runner: linux # large'", "runner: linux # large"},
		{"expression", "${{ matrix.runner }}", "${{ matrix.runner }}"},
		{"quoted expression", `"${{ inputs.runner }}"`, "${{ inputs.runner }}"},
		{"array expression", `${{ fromJSON('["self-hosted", "linux"]') }}`, `${{ fromJSON('["self-hosted", "linux"]') }}`},
		{"folded expression", ">-\n      ${{ inputs.runner ||\n          'ubuntu-latest' }}", "${{ inputs.runner ||\n    'ubuntu-latest' }}"},
		{"single label", "[self-hosted]", []any{"self-hosted"}},
		{"single block label", "\n      - self-hosted", []any{"self-hosted"}},
		{"flow labels", "[self-hosted, linux, x64, something]", []any{"self-hosted", "linux", "x64", "something"}},
		{"multiline flow labels", "[\n      self-hosted,\n      linux,\n    ]", []any{"self-hosted", "linux"}},
		{"block labels", "\n      - self-hosted\n      - linux\n      - x64", []any{"self-hosted", "linux", "x64"}},
		{"indentless labels", "\n    - self-hosted\n    - linux", []any{"self-hosted", "linux"}},
		{"group", "{group: my-runners}", map[string]any{"group": "my-runners"}},
		{"block group", "\n      group: my-runners", map[string]any{"group": "my-runners"}},
		{"label only", "{labels: linux}", map[string]any{"labels": "linux"}},
		{"labels only", "{labels: [self-hosted, linux]}", map[string]any{"labels": []any{"self-hosted", "linux"}}},
		{"group and label", "{group: my-runners, labels: linux}", map[string]any{"group": "my-runners", "labels": "linux"}},
		{"flow group and labels", "{group: my-runners, labels: [linux, x64]}", map[string]any{"group": "my-runners", "labels": []any{"linux", "x64"}}},
		{"group and labels", "\n      group: my-runners\n      labels:\n        - linux\n        - x64", map[string]any{"group": "my-runners", "labels": []any{"linux", "x64"}}},
		{"group and indentless labels", "\n      group: my-runners\n      labels:\n      - linux\n      - x64", map[string]any{"group": "my-runners", "labels": []any{"linux", "x64"}}},
		{"group expression", "{group: '${{ inputs.group }}'}", map[string]any{"group": "${{ inputs.group }}"}},
		{"labels expression", "{labels: '${{ matrix.labels }}'}", map[string]any{"labels": "${{ matrix.labels }}"}},
		{"group and label expressions", "{group: '${{ inputs.group }}', labels: [self-hosted, '${{ matrix.os }}']}", map[string]any{"group": "${{ inputs.group }}", "labels": []any{"self-hosted", "${{ matrix.os }}"}}},
		{"expression label", "[self-hosted, '${{ matrix.os }}']", []any{"self-hosted", "${{ matrix.os }}"}},
		{"anchored scalar", "&runner ubuntu-latest", "ubuntu-latest"},
		{"anchored labels", "&runners [self-hosted, linux]", []any{"self-hosted", "linux"}},
		{"anchored group", "&runners {group: my-runners, labels: [linux, x64]}", map[string]any{"group": "my-runners", "labels": []any{"linux", "x64"}}},
		{"aliased label", "[&label self-hosted, *label]", []any{"self-hosted", "self-hosted"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source := "# Runner selection\njobs:\n  build:\n    runs-on: " + tt.source + "\n    steps:\n      - run: echo test\n"
			input, err := Parse([]byte(source))
			require.NoError(t, err)
			assert.Equal(t, tt.value, input.Jobs[0].RunsOn)
			emitted, err := Emit(input)
			require.NoError(t, err)
			assert.Equal(t, source, emitted, "unchanged runners must preserve source bytes")

			input.Name = "Edited"
			emitted, err = Emit(input)
			require.NoError(t, err)
			reparsed, err := Parse([]byte(emitted))
			require.NoError(t, err)
			assert.Equal(t, tt.value, reparsed.Jobs[0].RunsOn, "unrelated edits must preserve runners")

			constructed := &Workflow{Jobs: []*Job{{
				ID: "build", RunsOn: tt.value, Steps: []Step{RunStep{Run: "echo test"}},
			}}}
			emitted, err = Emit(constructed)
			require.NoError(t, err)
			reparsed, err = Parse([]byte(emitted))
			require.NoError(t, err)
			assert.Equal(t, tt.value, reparsed.Jobs[0].RunsOn, "new ASTs must preserve runner types and values")

			for _, target := range tests {
				t.Run("edit to "+target.name, func(t *testing.T) {
					input.Jobs[0].RunsOn = target.value
					emitted, err := Emit(input)
					require.NoError(t, err)
					reparsed, err := Parse([]byte(emitted))
					require.NoError(t, err)
					assert.Equal(t, target.value, reparsed.Jobs[0].RunsOn)
					again, err := Emit(reparsed)
					require.NoError(t, err)
					assert.Equal(t, emitted, again, "edited runner emission must be stable")
				})
			}
		})
	}
}

func TestRunsOnAliasRoundTrip(t *testing.T) {
	for _, runner := range []string{"ubuntu-latest", "[self-hosted, linux]", "{group: my-runners, labels: [linux, x64]}"} {
		t.Run(runner, func(t *testing.T) {
			source := "jobs:\n  build:\n    runs-on: &runners " + runner +
				"\n    steps: [{run: echo build}]\n  test:\n    runs-on: *runners\n    steps: [{run: echo test}]\n"
			input, err := Parse([]byte(source))
			require.NoError(t, err)
			expected := input.Jobs[1].RunsOn
			emitted, err := Emit(input)
			require.NoError(t, err)
			assert.Equal(t, source, emitted)

			input.Jobs[0].RunsOn = "windows-latest"
			emitted, err = Emit(input)
			require.NoError(t, err)
			reparsed, err := Parse([]byte(emitted))
			require.NoError(t, err)
			assert.Equal(t, "windows-latest", reparsed.Jobs[0].RunsOn)
			assert.Equal(t, expected, reparsed.Jobs[1].RunsOn, "editing an anchor must not change another job's runners")
		})
	}
}

func TestSameJobOutputsAndScopedIDs(t *testing.T) {
	input := outputWorkflow()
	input.Jobs[1].Steps = []Step{RunStep{
		StepOptions: StepOptions{ID: "produce"},
		Run:         "echo unrelated",
	}}
	input.Jobs[0].Steps = append(input.Jobs[0].Steps, RunStep{
		StepOptions: StepOptions{
			Env: map[string]Expression{"VALUE": OutputRef{Job: "setup", Step: "produce", Name: "value"}},
		},
		Run: "echo \"$VALUE\"",
	})
	content, err := Emit(input)
	require.NoError(t, err)
	assert.Contains(t, content, "${{ steps.produce.outputs.value }}")
	assert.NotContains(t, content, "needs:")
	assert.NotContains(t, content, "outputs:\n", "same-job output does not need a job binding")
}

func TestResolvePreservesExplicitEdgesAndBindings(t *testing.T) {
	input := outputWorkflow()
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
		change func(*Workflow)
		want   string
	}{
		{"duplicate job", func(p *Workflow) { p.Jobs = append(p.Jobs, p.Jobs[0]) }, "duplicate workflow job"},
		{"invalid job", func(p *Workflow) { p.Jobs[0].ID = "bad.job" }, "invalid job id"},
		{"nil job", func(p *Workflow) { p.Jobs = append(p.Jobs, nil) }, "job cannot be nil"},
		{"nil step", func(p *Workflow) { p.Jobs[0].Steps = append(p.Jobs[0].Steps, nil) }, "step cannot be nil"},
		{"duplicate step", func(p *Workflow) { p.Jobs[0].Steps = append(p.Jobs[0].Steps, p.Jobs[0].Steps[0]) }, "duplicate workflow step"},
		{"duplicate raw step", func(p *Workflow) {
			p.Jobs[0].Steps = append(p.Jobs[0].Steps, RawStep{Fields: map[string]any{"id": "produce", "run": "echo test"}})
		}, "duplicate workflow step"},
		{"invalid raw step id", func(p *Workflow) {
			p.Jobs[0].Steps = append(p.Jobs[0].Steps, RawStep{Fields: map[string]any{"id": 1, "run": "echo test"}})
		}, "id must be a string"},
		{"unknown needs", func(p *Workflow) { p.Jobs[0].Needs = []JobID{"missing"} }, "non-existent job"},
		{"explicit cycle", func(p *Workflow) { p.Jobs[0].Needs = []JobID{"agent"} }, "cycle detected"},
		{"self cycle", func(p *Workflow) { p.Jobs[0].Needs = []JobID{"setup"} }, "cycle detected"},
		{"missing producer", func(p *Workflow) { p.Jobs = p.Jobs[1:] }, "unknown producer job"},
		{"unknown step output", func(p *Workflow) {
			p.Jobs[1].Env = map[string]Expression{"VALUE": OutputRef{Job: "setup", Step: "produce", Name: "missing"}}
		}, "undeclared step output"},
		{"unknown job output", func(p *Workflow) {
			p.Jobs[1].Env = map[string]Expression{"VALUE": OutputRef{Job: "setup", Name: "missing"}}
		}, "unknown output"},
		{"same job output", func(p *Workflow) {
			p.Jobs[0].Outputs = map[OutputName]Expression{"public": Literal("value")}
			p.Jobs[0].Env = map[string]Expression{"VALUE": OutputRef{Job: "setup", Name: "public"}}
		}, "cannot consume its own job output"},
		{"anonymous output", func(p *Workflow) {
			p.Jobs[0].Steps = []Step{RunStep{StepOptions: StepOptions{Outputs: []OutputDecl{{Name: "value"}}}, Run: "echo test"}}
		}, "anonymous step"},
		{"duplicate output", func(p *Workflow) {
			step := p.Jobs[0].Steps[0].(RunStep)
			step.Outputs = append(step.Outputs, step.Outputs[0])
			p.Jobs[0].Steps[0] = step
		}, "duplicate output"},
		{"forward reference", func(p *Workflow) {
			p.Jobs[0].Steps = append([]Step{RunStep{
				StepOptions: StepOptions{Env: map[string]Expression{"VALUE": OutputRef{Job: "setup", Step: "produce", Name: "value"}}},
				Run:         "echo \"$VALUE\"",
			}}, p.Jobs[0].Steps...)
		}, "before it runs"},
		{"self step reference", func(p *Workflow) {
			step := p.Jobs[0].Steps[0].(RunStep)
			step.Env = map[string]Expression{"VALUE": OutputRef{Job: "setup", Step: "produce", Name: "value"}}
			p.Jobs[0].Steps[0] = step
		}, "before it runs"},
		{"job condition reads local step", func(p *Workflow) {
			p.Jobs[0].If = OutputRef{Job: "setup", Step: "produce", Name: "value"}
		}, "before it runs"},
		{"binding collision", func(p *Workflow) {
			p.Jobs[0].Outputs = map[OutputName]Expression{"aw_7_produce_value": Literal("other")}
		}, "conflicts with an explicit output"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := outputWorkflow()
			tt.change(input)
			_, err := Resolve(input)
			require.ErrorContains(t, err, tt.want)
		})
	}
	_, err := Resolve(nil)
	require.ErrorContains(t, err, "cannot be nil")
}

func TestNestedReferencesDeriveEdges(t *testing.T) {
	input := outputWorkflow()
	input.Jobs[1].Steps = nil
	input.Jobs[1].If = Not{Value: Binary{
		Left: Call{Name: "contains", Args: []Expression{
			Concat(Variable("PREFIX"), OutputRef{Job: "setup", Step: "produce", Name: "value"}),
			Literal("don't"),
		}},
		Operator: "==", Right: Boolean(false),
	}}
	resolved, err := Resolve(input)
	require.NoError(t, err)
	assert.Equal(t, []JobID{"setup"}, resolved.Jobs[1].Needs)
	value, err := RenderExpression(input.Jobs[1].If, "agent", resolved, true)
	require.NoError(t, err)
	assert.Contains(t, value, "'don''t'")
	assert.Contains(t, value, "vars.PREFIX")
	assert.Contains(t, value, "needs.setup.outputs.aw_7_produce_value")
}

func TestLowerRejectsInvalidShapes(t *testing.T) {
	tests := []struct {
		name   string
		change func(*Workflow)
		want   string
	}{
		{"reusable", func(p *Workflow) { p.Jobs[0].Uses = "./.github/workflows/worker.yml" }, "cannot declare"},
		{"empty run", func(p *Workflow) { p.Jobs[1].Steps = []Step{RunStep{}} }, "nonempty script"},
		{"run interpolation", func(p *Workflow) {
			p.Jobs[1].Steps = []Step{RunStep{Run: `echo "${{ github.event.issue.title }}"`}}
		}, "pass Actions expressions through env"},
		{"empty action", func(p *Workflow) { p.Jobs[1].Steps = []Step{ActionStep{}} }, "nonempty uses"},
		{"empty raw", func(p *Workflow) { p.Jobs[1].Steps = []Step{RawStep{}} }, "cannot be empty"},
		{"negative timeout", func(p *Workflow) {
			p.Jobs[1].Steps = []Step{RunStep{StepOptions: StepOptions{TimeoutMinutes: -1}, Run: "echo test"}}
		}, "cannot be negative"},
		{"raw multiline expression", func(p *Workflow) { p.Jobs[1].If = Context("always()\nfoo") }, "single-line"},
		{"raw delimiters", func(p *Workflow) { p.Jobs[1].If = Context("${{ always() }}") }, "exclude expression delimiters"},
		{"invalid operator", func(p *Workflow) {
			p.Jobs[1].If = Binary{Left: Boolean(true), Operator: ";", Right: Boolean(false)}
		}, "unsupported workflow expression operator"},
		{"nil expression", func(p *Workflow) { p.Jobs[1].Env = map[string]Expression{"VALUE": nil} }, "cannot be nil"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := outputWorkflow()
			tt.change(input)
			_, err := Emit(input)
			require.ErrorContains(t, err, tt.want)
		})
	}
}

func TestRawStepPreservesNativeFields(t *testing.T) {
	input := &Workflow{Name: "raw", Jobs: []*Job{{
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

func TestResolveValidatesIdentifierGrammar(t *testing.T) {
	workflowWithIdentifiers := func(jobID JobID, stepID StepID, outputName OutputName) *Workflow {
		return &Workflow{Jobs: []*Job{{
			ID: jobID, RunsOn: "ubuntu-latest",
			Steps: []Step{RunStep{
				StepOptions: StepOptions{ID: stepID, Outputs: []OutputDecl{{Name: outputName}}},
				Run:         "echo test",
			}},
		}}}
	}
	for _, value := range []string{"agent", "_step", "safe-outputs", "output_1"} {
		_, err := Resolve(workflowWithIdentifiers(JobID(value), StepID(value), OutputName(value)))
		require.NoError(t, err)
	}
	for _, value := range []string{"", "1job", "job.name", "job name", "job\ninjected", "job:"} {
		t.Run("job id "+value, func(t *testing.T) {
			_, err := Resolve(workflowWithIdentifiers(JobID(value), "step", "output"))
			require.ErrorContains(t, err, "invalid job id")
		})
		if value != "" {
			t.Run("step id "+value, func(t *testing.T) {
				_, err := Resolve(workflowWithIdentifiers("job", StepID(value), "output"))
				require.ErrorContains(t, err, "invalid step id")
			})
		}
		t.Run("output name "+value, func(t *testing.T) {
			_, err := Resolve(workflowWithIdentifiers("job", "step", OutputName(value)))
			require.ErrorContains(t, err, "invalid output name")
		})
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
