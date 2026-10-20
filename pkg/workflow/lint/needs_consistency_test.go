package lint

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/workflow/actions"
	"github.com/github/gh-aw/pkg/workflow/aw"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNeedsConsistencyExpressions(t *testing.T) {
	const producer = "jobs:\n  build:\n    runs-on: ubuntu-latest\n    outputs: {value: public}\n    steps: [{run: echo build}]\n  check:\n    runs-on: ubuntu-latest\n"
	for _, tt := range []struct {
		name, fields, path string
	}{
		{"missing direct edge", "    if: needs.build.result == 'success'\n", "jobs.check.if"},
		{"explicit edge", "    needs: build\n    if: needs.build.result == 'success'\n", ""},
		{"case insensitive", "    needs: build\n    if: NEEDS['BUILD']['RESULT'] == 'success'\n", ""},
		{"literal brackets", "    if: needs['build']['outputs']['value'] != ''\n", "jobs.check.if"},
		{"declared output", "    needs: build\n    env: {VALUE: '${{ needs.build.outputs.value }}'}\n", ""},
		{"unknown output", "    needs: build\n    env: {VALUE: '${{ needs.build.outputs.PRIVATE_VALUE }}'}\n", "jobs.check.env.VALUE"},
		{"unknown job", "    env: {VALUE: '${{ needs.PRIVATE_VALUE.result }}'}\n", "jobs.check.env.VALUE"},
		{"unknown property", "    needs: build\n    if: needs.build.PRIVATE_VALUE == 'success'\n", "jobs.check.if"},
		{"quoted text", "    if: contains('needs.PRIVATE_VALUE.result', 'needs')\n", ""},
		{"ordinary text", "    env: {VALUE: needs.PRIVATE_VALUE.result}\n", ""},
		{"wildcard filter", "    needs: build\n    if: contains(needs.*.result, 'success')\n", ""},
		{"whole context", "    needs: build\n    env: {VALUE: '${{ toJSON(needs) }}'}\n", ""},
		{"computed index", "    needs: build\n    env: {VALUE: '${{ needs[inputs.job].outputs[inputs.output] }}'}\n", ""},
		{"nested computed index", "    env: {VALUE: '${{ needs[needs.build.outputs.value].result }}'}\n", "jobs.check.env.VALUE"},
		{"ordering only", "    needs: build\n", ""},
		{"output wildcard", "    needs: build\n    env: {VALUE: '${{ toJSON(needs.build.outputs.*) }}'}\n", ""},
		{"literal star is not wildcard", "    env: {VALUE: \"${{ needs['*'].result }}\"}\n", "jobs.check.env.VALUE"},
		{"strategy", "    strategy: {matrix: '${{ fromJSON(needs.build.outputs.value) }}'}\n", "jobs.check.strategy.matrix"},
	} {
		for _, target := range []ASTTargets{AWAST, ActionsAST} {
			t.Run(tt.name+"/"+target.String(), func(t *testing.T) {
				input := ruleContractInput(t, producer+tt.fields+"    steps: [{run: echo check}]\n", target, false)
				before, err := json.Marshal(input)
				require.NoError(t, err)
				runner, err := Default().Select("needs-consistency")
				require.NoError(t, err)
				found, err := runner.Run(context.Background(), input)
				require.NoError(t, err)
				if tt.path == "" {
					assert.Empty(t, found)
				} else {
					require.Len(t, found, 1)
					assert.Equal(t, tt.path, found[0].Path)
					assert.Equal(t, Warning, found[0].Severity)
					assert.NotContains(t, found[0].Message, "PRIVATE_VALUE")
				}
				after, err := json.Marshal(input)
				require.NoError(t, err)
				assert.Equal(t, string(before), string(after))
			})
		}
	}
}

func TestNeedsConsistencyStepAndCallFields(t *testing.T) {
	for _, fields := range []string{
		"steps:\n      - if: needs.build.result == 'success'\n        run: echo check",
		"steps:\n      - run: echo '${{ needs.build.outputs.value }}'",
		"steps:\n      - uses: ./local\n        with: {value: '${{ needs.build.outputs.value }}'}",
		"uses: ./reusable.yml\n    with: {value: '${{ needs.build.outputs.value }}'}",
		"uses: ./reusable.yml\n    secrets: {value: '${{ needs.build.outputs.value }}'}",
		"outputs: {value: '${{ needs.build.outputs.value }}'}\n    steps: [{run: echo check}]",
	} {
		for _, target := range []ASTTargets{AWAST, ActionsAST} {
			runner := ""
			if !strings.HasPrefix(fields, "uses:") {
				runner = "    runs-on: ubuntu-latest\n"
			}
			input := ruleContractInput(t, "jobs:\n  build:\n    runs-on: ubuntu-latest\n    outputs: {value: public}\n    steps: [{run: echo build}]\n  check:\n"+runner+"    "+fields+"\n", target, false)
			found, err := NeedsConsistencyRule().Check(context.Background(), input)
			require.NoError(t, err)
			require.Len(t, found, 1, fields)
		}
	}
	// An action input named "if" is not an implicit Actions expression.
	input := ruleContractInput(t, "jobs:\n  check:\n    steps:\n      - uses: ./local\n        with: {if: arbitrary text}\n", AWAST, false)
	found, err := NeedsConsistencyRule().Check(context.Background(), input)
	require.NoError(t, err)
	assert.Empty(t, found)
}

func TestNeedsConsistencyAuthoredGraph(t *testing.T) {
	for _, tt := range []struct {
		name, source, path string
	}{
		{"unknown", "jobs: {check: {needs: missing}}", "jobs.check.needs"},
		{"self", "jobs: {check: {needs: check}}", "jobs.check.needs"},
		{"cycle", "jobs: {a: {needs: b}, b: {needs: a}}", "jobs.b.needs"},
		{"transitive", "jobs: {a: {}, b: {needs: a}, c: {needs: b, if: \"needs.a.result == 'success'\"}}", "jobs.c.if"},
		{"builtin target", "jobs: {check: {needs: agent}}", ""},
		{"builtin missing edge", "jobs: {check: {if: \"needs.agent.result == 'success'\"}}", "jobs.check.if"},
		{"default activation edge", "jobs: {check: {if: \"needs.activation.result == 'success'\"}}", ""},
		{"explicit needs replaces default", "jobs: {build: {}, check: {needs: build, if: \"needs.activation.result == 'success'\"}}", "jobs.check.if"},
		{"imported target", "imports: [shared.md]\njobs: {check: {needs: imported}}", ""},
		{"imported edge", "imports: [shared.md]\njobs: {check: {if: \"needs.imported.result == 'success'\"}}", ""},
		{"safe job normalization", "safe-jobs: {first-job: {}, second-job: {needs: first-job, if: \"needs.first_job.result == 'success'\"}}", ""},
		{"safe output jobs", "safe-outputs: {jobs: {a: {}, b: {needs: a, if: \"needs.a.result == 'success'\"}}}", ""},
		{"safe job cycle", "safe-jobs: {a: {needs: b}, b: {needs: a}}", "safe-jobs.b.needs"},
		{"safe job defaults", "safe-jobs: {check: {if: \"needs.agent.result == 'success'\"}}", ""},
		{"root and prompt defaults", "env: {VALUE: '${{ needs.build.outputs.value }}'}", ""},
	} {
		t.Run(tt.name, func(t *testing.T) {
			input := ruleContractInput(t, tt.source+"\n", AWAST, false)
			found, err := NeedsConsistencyRule().Check(context.Background(), input)
			require.NoError(t, err)
			if tt.path == "" {
				assert.Empty(t, found)
			} else {
				require.Len(t, found, 1)
				assert.Equal(t, tt.path, found[0].Path)
			}
		})
	}
}

func TestNeedsConsistencyCompiledPrecedenceAndTypedReferences(t *testing.T) {
	doc, err := aw.Parse("---\njobs: {check: {needs: missing}}\n---\n")
	require.NoError(t, err)
	wf, err := actions.Parse([]byte("jobs:\n  build:\n    uses: ./reusable.yml\n  check:\n    runs-on: ubuntu-latest\n    needs: build\n    env: {VALUE: '${{ needs.build.outputs.external }}'}\n    steps: [{run: echo check}]\n"))
	require.NoError(t, err)
	found, err := NeedsConsistencyRule().Check(context.Background(), Input{Document: doc, Workflow: wf})
	require.NoError(t, err)
	assert.Empty(t, found, "compiled graph wins and external reusable outputs are unknown")

	wf = &actions.Workflow{Jobs: []*actions.Job{
		{ID: "build", RunsOn: "ubuntu-latest", Outputs: map[actions.OutputName]actions.Expression{"value": actions.Literal("public")},
			Steps: []actions.Step{actions.RawStep{Fields: map[string]any{"run": "echo build"}}}},
		{ID: "check", RunsOn: "ubuntu-latest", Env: map[string]actions.Expression{
			"VALUE": actions.OutputRef{Job: "build", Name: "value"},
		}, Steps: []actions.Step{actions.RawStep{Fields: map[string]any{"run": "echo check"}}}},
	}}
	found, err = NeedsConsistencyRule().Check(context.Background(), Input{Workflow: wf})
	require.NoError(t, err)
	assert.Empty(t, found, "typed references derive needs during lowering")
	assert.Empty(t, wf.Jobs[1].Needs, "lowering must not mutate the input")
}

func TestNeedsConsistencyInvalidAndCancelled(t *testing.T) {
	for _, source := range []string{
		"jobs: invalid", "jobs: {check: invalid}", "jobs: {check: {needs: [123]}}",
		"jobs: {check: {needs: '${{ inputs.PRIVATE_VALUE }}'}}",
		"jobs: {check: {if: 'needs.PRIVATE_VALUE &&'}}",
		"jobs: {check: {env: {VALUE: '${{ needs.PRIVATE_VALUE'}}}",
	} {
		input := ruleContractInput(t, source+"\n", AWAST, false)
		_, err := NeedsConsistencyRule().Check(context.Background(), input)
		require.Error(t, err)
		assert.NotContains(t, err.Error(), "PRIVATE_VALUE")
	}
	_, err := NeedsConsistencyRule().Check(context.Background(), Input{})
	require.Error(t, err)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = NeedsConsistencyRule().Check(ctx, Input{Workflow: &actions.Workflow{}})
	require.ErrorIs(t, err, context.Canceled)
}

func TestNeedsConsistencyWorkflowScope(t *testing.T) {
	input := ruleContractInput(t, "env: {VALUE: '${{ needs.build.outputs.value }}'}\njobs:\n  build:\n    runs-on: ubuntu-latest\n    outputs: {value: public}\n    steps: [{run: echo build}]\n", ActionsAST, false)
	found, err := NeedsConsistencyRule().Check(context.Background(), input)
	require.NoError(t, err)
	require.Len(t, found, 1)
	assert.Equal(t, "env.VALUE", found[0].Path)
}
