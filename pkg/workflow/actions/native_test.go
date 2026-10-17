package actions

import (
	"math"
	"testing"

	"github.com/goccy/go-yaml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNativeWorkflowAndRunnerProperties(t *testing.T) {
	input := outputWorkflow()
	input.RunName = "${{ github.workflow }}: ${{ github.ref_name }}"
	input.Env = map[string]Expression{"GLOBAL": Variable("GLOBAL")}
	input.Defaults = map[string]any{"run": map[string]any{"shell": "bash"}}
	input.Concurrency = map[string]any{"group": "${{ github.workflow }}", "cancel-in-progress": true}
	job := input.Jobs[1]
	job.Permissions = map[string]any{"contents": "read"}
	job.Strategy = map[string]any{"matrix": map[string]any{"os": []string{"ubuntu-latest"}}}
	job.Environment = "production"
	job.Defaults = map[string]any{"run": map[string]any{"working-directory": "./src"}}
	job.Concurrency = "deployment"
	job.Container = "node:22"
	job.Services = map[string]any{"redis": map[string]any{"image": "redis:7"}}
	job.TimeoutMinutes = 15
	job.ContinueOnError = Context("matrix.experimental")
	job.Steps = []Step{RunStep{
		StepOptions: StepOptions{ContinueOnError: Boolean(false), TimeoutMinutes: 5},
		Run:         "echo \"$VALUE\"", Shell: "bash", WorkingDirectory: "./src",
	}}
	fields := emittedWorkflow(t, input)
	assert.Equal(t, input.RunName, fields["run-name"])
	assert.Equal(t, map[string]any{"GLOBAL": "${{ vars.GLOBAL }}"}, fields["env"])
	assert.Equal(t, input.Defaults, fields["defaults"])
	assert.Equal(t, input.Concurrency, fields["concurrency"])
	actual := fields["jobs"].(map[string]any)["agent"].(map[string]any)
	assert.Equal(t, job.Permissions, actual["permissions"])
	assert.Equal(t, map[string]any{"matrix": map[string]any{"os": []any{"ubuntu-latest"}}}, actual["strategy"])
	assert.Equal(t, job.Environment, actual["environment"])
	assert.Equal(t, job.Defaults, actual["defaults"])
	assert.Equal(t, job.Concurrency, actual["concurrency"])
	assert.Equal(t, job.Container, actual["container"])
	assert.Equal(t, job.Services, actual["services"])
	assert.EqualValues(t, 15, actual["timeout-minutes"])
	assert.Equal(t, "${{ matrix.experimental }}", actual["continue-on-error"])
	step := actual["steps"].([]any)[0].(map[string]any)
	assert.Equal(t, "${{ false }}", step["continue-on-error"])
	assert.EqualValues(t, 5, step["timeout-minutes"])
	assert.Equal(t, "bash", step["shell"])
	assert.Equal(t, "./src", step["working-directory"])
}

func TestReusableWorkflowInputsSecretsAndOutputs(t *testing.T) {
	input := outputWorkflow()
	call := &Job{
		ID: "call", Uses: "./.github/workflows/worker.yml",
		With: map[string]Expression{
			"message": OutputRef{Job: "setup", Step: "produce", Name: "value"},
			"enabled": Boolean(false), "count": Number(3),
		},
		Secrets:     map[string]Expression{"token": Secret("API_TOKEN")},
		Permissions: map[string]any{"contents": "read"},
		CallOutputs: []OutputDecl{{Name: "result"}},
	}
	input.Jobs = append(input.Jobs[:1], call, &Job{
		ID: "consume", RunsOn: "ubuntu-latest",
		Steps: []Step{RunStep{
			StepOptions: StepOptions{Env: map[string]Expression{"RESULT": OutputRef{Job: "call", Name: "result"}}},
			Run:         "echo \"$RESULT\"",
		}},
	})
	fields := emittedWorkflow(t, input)
	jobs := fields["jobs"].(map[string]any)
	caller := jobs["call"].(map[string]any)
	assert.Equal(t, []any{"setup"}, caller["needs"])
	with := caller["with"].(map[string]any)
	assert.Equal(t, "${{ needs.setup.outputs.aw_7_produce_value }}", with["message"])
	assert.Equal(t, false, with["enabled"])
	assert.EqualValues(t, 3, with["count"])
	assert.Equal(t, map[string]any{"token": "${{ secrets.API_TOKEN }}"}, caller["secrets"])
	assert.NotContains(t, caller, "outputs")
	assert.NotContains(t, caller, "runs-on")
	consumer := jobs["consume"].(map[string]any)
	assert.Equal(t, []any{"call"}, consumer["needs"])
	step := consumer["steps"].([]any)[0].(map[string]any)
	assert.Equal(t, map[string]any{"RESULT": "${{ needs.call.outputs.result }}"}, step["env"])
	assert.Empty(t, call.Needs, "resolution must not mutate reusable-workflow callers")
	call.Secrets = nil
	call.SecretsInherit = true
	fields = emittedWorkflow(t, input)
	assert.Equal(t, "inherit", fields["jobs"].(map[string]any)["call"].(map[string]any)["secrets"])
}

func TestNativeJobExpressionReferencesDeriveDependencies(t *testing.T) {
	ref := OutputRef{Job: "setup", Step: "produce", Name: "value"}
	for _, field := range []string{"with", "secrets", "continue-on-error", "step-continue-on-error"} {
		t.Run(field, func(t *testing.T) {
			input := outputWorkflow()
			job := input.Jobs[1]
			job.Steps = []Step{RunStep{Run: "echo test"}}
			switch field {
			case "with":
				job.Uses, job.RunsOn, job.Steps = "./.github/workflows/worker.yml", nil, nil
				job.With = map[string]Expression{"value": ref}
			case "secrets":
				job.Uses, job.RunsOn, job.Steps = "./.github/workflows/worker.yml", nil, nil
				job.Secrets = map[string]Expression{"value": ref}
			case "continue-on-error":
				job.ContinueOnError = ref
			case "step-continue-on-error":
				job.Steps = []Step{RunStep{StepOptions: StepOptions{ContinueOnError: ref}, Run: "echo test"}}
			}
			resolved, err := Resolve(input)
			require.NoError(t, err)
			assert.Equal(t, []JobID{"setup"}, resolved.Jobs[1].Needs)
			_, err = Emit(input)
			require.NoError(t, err)
		})
	}
}

func TestNativeConditionsKeepLiteralExpressions(t *testing.T) {
	input := &Workflow{Jobs: []*Job{{
		ID: "build", RunsOn: "ubuntu-latest", If: Literal("false"),
		Steps: []Step{RunStep{
			StepOptions: StepOptions{If: Literal("ready")},
			Run:         "echo test",
		}},
	}}}

	job := emittedWorkflow(t, input)["jobs"].(map[string]any)["build"].(map[string]any)
	assert.Equal(t, "${{ 'false' }}", job["if"])
	step := job["steps"].([]any)[0].(map[string]any)
	assert.Equal(t, "${{ 'ready' }}", step["if"])
}

func TestContextOperandsPreserveBinaryGrouping(t *testing.T) {
	tests := []struct {
		name     string
		expr     Expression
		expected string
	}{
		{
			name:     "left",
			expr:     Binary{Left: Context("true || false"), Operator: "&&", Right: Boolean(false)},
			expected: "((true || false) && false)",
		},
		{
			name:     "right",
			expr:     Binary{Left: Boolean(false), Operator: "||", Right: Context("true && false")},
			expected: "(false || (true && false))",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value, err := RenderExpression(tt.expr, "build", &Workflow{}, false)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, value)
		})
	}
}

func TestRejectsNonActionsAndInvalidNativeProperties(t *testing.T) {
	tests := []struct {
		name   string
		change func(*Workflow)
		want   string
	}{
		{"runner inputs", func(w *Workflow) { w.Jobs[0].With = map[string]Expression{"input": Literal("value")} }, "require a reusable"},
		{"runner secrets", func(w *Workflow) { w.Jobs[0].SecretsInherit = true }, "require a reusable"},
		{"runner call outputs", func(w *Workflow) { w.Jobs[0].CallOutputs = []OutputDecl{{Name: "value"}} }, "require a reusable"},
		{"runner missing runs-on", func(w *Workflow) { w.Jobs[0].RunsOn = nil }, "requires runs-on"},
		{"runner missing steps", func(w *Workflow) { w.Jobs[1].Steps = nil }, "requires steps"},
		{"job timeout", func(w *Workflow) { w.Jobs[0].TimeoutMinutes = -1 }, "cannot be negative"},
		{"workflow env outputs", func(w *Workflow) {
			w.Env = map[string]Expression{"VALUE": OutputRef{Job: "setup", Step: "produce", Name: "value"}}
		}, "workflow env cannot reference"},
		{"duplicate call output", func(w *Workflow) {
			w.Jobs = append(w.Jobs, &Job{ID: "call", Uses: "./.github/workflows/worker.yml",
				CallOutputs: []OutputDecl{{Name: "value"}, {Name: "value"}}})
		}, "duplicate reusable workflow output"},
		{"secrets inherit conflict", func(w *Workflow) {
			w.Jobs = append(w.Jobs, &Job{ID: "call", Uses: "./.github/workflows/worker.yml",
				SecretsInherit: true, Secrets: map[string]Expression{"token": Secret("TOKEN")}})
		}, "cannot combine inherited and explicit secrets"},
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

func TestRejectsNonActionsRawSteps(t *testing.T) {
	tests := []struct {
		fields map[string]any
		want   string
	}{
		{map[string]any{"task": "Bash@3"}, "unsupported GitHub Actions step property"},
		{map[string]any{"bash": "echo test"}, "unsupported GitHub Actions step property"},
		{map[string]any{"script": "echo test"}, "unsupported GitHub Actions step property"},
		{map[string]any{"run": "echo test", "displayName": "Test"}, "unsupported GitHub Actions step property"},
		{map[string]any{"name": "Test"}, "exactly one of run or uses"},
		{map[string]any{"run": "echo test", "uses": "owner/action@sha"}, "exactly one of run or uses"},
		{map[string]any{"run": 123}, "nonempty string"},
		{map[string]any{"uses": ""}, "nonempty string"},
		{map[string]any{"run": "echo test", "with": map[string]any{}}, "run step cannot declare with"},
		{map[string]any{"uses": "owner/action@sha", "shell": "bash"}, "uses step cannot declare shell"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			_, err := LowerStep(RawStep{Fields: tt.fields}, "build", &Workflow{})
			require.ErrorContains(t, err, tt.want)
		})
	}
}

func TestActionsNumberAndTruthiness(t *testing.T) {
	value, err := RenderExpression(Or(Number(0), Number(2.5)), "build", &Workflow{}, true)
	require.NoError(t, err)
	assert.Equal(t, "${{ (0 || 2.5) }}", value)
	for _, value := range []float64{math.NaN(), math.Inf(1), math.Inf(-1)} {
		_, err := RenderExpression(Number(value), "build", &Workflow{}, true)
		require.ErrorContains(t, err, "must be finite")
	}
}

func TestRejectsNonActionsExpressionFunctions(t *testing.T) {
	for _, name := range []string{"coalesce", "variables", "eq", "and", "succeeded"} {
		t.Run(name, func(t *testing.T) {
			_, err := RenderExpression(Call{Name: name}, "build", &Workflow{}, true)
			require.ErrorContains(t, err, "unsupported GitHub Actions expression function")
		})
	}
	for _, name := range []string{"contains", "startsWith", "endsWith", "format", "join",
		"toJSON", "fromJSON", "hashFiles", "success", "always", "cancelled", "failure", "ALWAYS"} {
		t.Run(name, func(t *testing.T) {
			value, err := RenderExpression(Call{Name: name}, "build", &Workflow{}, true)
			require.NoError(t, err)
			assert.Equal(t, "${{ "+name+"() }}", value)
		})
	}
}

func emittedWorkflow(t *testing.T, input *Workflow) map[string]any {
	t.Helper()
	content, err := Emit(input)
	require.NoError(t, err)
	var fields map[string]any
	require.NoError(t, yaml.Unmarshal([]byte(content), &fields))
	return fields
}
