package lint

import (
	"context"
	"testing"

	"github.com/github/gh-aw/pkg/workflow/actions"
	"github.com/github/gh-aw/pkg/workflow/aw"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func workflowRunScopeInput(t *testing.T, trigger string) Input {
	t.Helper()
	workflow, err := actions.Parse([]byte(trigger + `
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - run: echo test
`))
	require.NoError(t, err)
	return Input{Workflow: workflow}
}

func runWorkflowRunScope(t *testing.T, input Input) []Diagnostic {
	t.Helper()
	runner, err := New(WorkflowRunScopeRule())
	require.NoError(t, err)
	diagnostics, err := runner.Run(context.Background(), input)
	require.NoError(t, err)
	for _, diagnostic := range diagnostics {
		assert.Equal(t, "workflow-run-scope", diagnostic.RuleID)
		assert.NotEmpty(t, diagnostic.Path)
		assert.Zero(t, diagnostic.Line, "structural paths must not masquerade as script line numbers")
	}
	return diagnostics
}

func TestWorkflowRunScopeParsedTriggers(t *testing.T) {
	tests := []struct {
		name    string
		trigger string
		path    string
	}{
		{name: "no trigger", trigger: ""},
		{name: "scalar other trigger", trigger: "on: workflow_dispatch"},
		{name: "other trigger sequence", trigger: "on: [push, workflow_dispatch]"},
		{name: "substring is not event", trigger: "on: [workflow_run_extra, not_workflow_run]"},
		{name: "event name in other config", trigger: "on: {push: {branches: [workflow_run]}}"},
		{name: "workflow_run null", trigger: "on: {workflow_run: null}", path: "on.workflow_run"},
		{name: "workflow_run scalar", trigger: "on: workflow_run", path: "on"},
		{name: "workflow_run sequence", trigger: "on: [push, workflow_run]", path: "on[1]"},
		{name: "workflow_run malformed sequence mapping", trigger: "on: [{workflow_run: {workflows: [Build], branches: [main]}}]", path: "on[0].workflow_run"},
		{name: "workflow_run wrong config", trigger: "on: {workflow_run: [completed]}", path: "on.workflow_run"},
		{name: "workflow_run boolean config", trigger: "on: {workflow_run: false}", path: "on.workflow_run"},
		{name: "missing workflows", trigger: "on: {workflow_run: {branches: [main]}}", path: "on.workflow_run.workflows"},
		{name: "valid workflow list", trigger: "on: {workflow_run: {workflows: [Build, Test], branches: [main]}}"},
		{name: "valid scalar workflow name", trigger: "on: {workflow_run: {workflows: Build, branches: [main]}}"},
		{name: "scoped branch glob", trigger: "on: {workflow_run: {workflows: [Build], branches: ['release/**']}}"},
		{name: "scoped exclusions", trigger: "on: {workflow_run: {workflows: [Build], branches: ['release/**', '!release/beta/**']}}"},
		{name: "ordered universal exclusion and scoped reinclude", trigger: "on: {workflow_run: {workflows: [Build], branches: ['**', '!**', main]}}"},
		{name: "mixed events", trigger: "on: {push: null, workflow_run: {workflows: [Build], branches: [main]}}"},
		{name: "valid YAML alias", trigger: "on: {push: {branches: &trusted [main]}, workflow_run: {workflows: [Build], branches: *trusted}}"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diagnostics := runWorkflowRunScope(t, workflowRunScopeInput(t, tt.trigger))
			if tt.path == "" {
				assert.Empty(t, diagnostics)
				return
			}
			require.Len(t, diagnostics, 1)
			assert.Equal(t, Error, diagnostics[0].Severity)
			assert.Equal(t, tt.path, diagnostics[0].Path)
			assert.Contains(t, diagnostics[0].Message, "workflows")
		})
	}
}

func TestWorkflowRunScopeWorkflowSelection(t *testing.T) {
	for _, workflows := range []string{
		"null", "[]", "''", "'   '", "false", "42", "{}", "[42]", "['', ' ']", "[Build, false]", "[Build, '']",
		"'${{ inputs.workflow }}'", "[Build, '${{ inputs.workflow }}']",
	} {
		t.Run(workflows, func(t *testing.T) {
			input := workflowRunScopeInput(t, "on: {workflow_run: {workflows: "+workflows+", branches: [main]}}")
			input.Document = &aw.Document{Frontmatter: map[string]any{"strict": false}}
			diagnostics := runWorkflowRunScope(t, input)
			require.Len(t, diagnostics, 1)
			assert.Equal(t, Error, diagnostics[0].Severity, "malformed workflow selection is always an error")
			assert.Equal(t, "on.workflow_run.workflows", diagnostics[0].Path)
		})
	}
}

func TestWorkflowRunScopeBranchSelection(t *testing.T) {
	tests := []struct {
		name   string
		fields string
	}{
		{name: "missing branches"},
		{name: "null branches", fields: "branches: null"},
		{name: "empty branches", fields: "branches: []"},
		{name: "blank branch", fields: "branches: [' ']"},
		{name: "scalar branch", fields: "branches: main"},
		{name: "boolean branch", fields: "branches: false"},
		{name: "mapping branch", fields: "branches: {name: main}"},
		{name: "mixed branch types", fields: "branches: [main, false]"},
		{name: "mixed blank branch", fields: "branches: [main, '']"},
		{name: "expression branch", fields: "branches: ['${{ inputs.branch }}']"},
		{name: "single wildcard", fields: "branches: ['*']"},
		{name: "double wildcard", fields: "branches: ['**']"},
		{name: "wildcard path", fields: "branches: ['**/*']"},
		{name: "wildcard with question", fields: "branches: ['**?**']"},
		{name: "broad positive before scoped positive", fields: "branches: ['**', main]"},
		{name: "broad positive after scoped positive", fields: "branches: [main, '**']"},
		{name: "positive canceled by exact exclusion", fields: "branches: [main, '!main']"},
		{name: "positive canceled by scoped exclusion", fields: "branches: ['release/**', '!release/**']"},
		{name: "negative only", fields: "branches: ['!untrusted/**']"},
		{name: "negative wildcard only", fields: "branches: ['!**']"},
		{name: "broad positive with exclusions", fields: "branches: ['**', '!untrusted/**']"},
		{name: "broad reinclude after universal exclusion", fields: "branches: [main, '!**', '**']"},
		{name: "branches-ignore alone", fields: "branches-ignore: [untrusted]"},
		{name: "branches-ignore wildcard", fields: "branches-ignore: ['**']"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := workflowRunScopeInput(t, "on: {workflow_run: {workflows: [Build], "+tt.fields+"}}")
			diagnostics := runWorkflowRunScope(t, input)
			require.Len(t, diagnostics, 1)
			assert.Equal(t, Warning, diagnostics[0].Severity)
			assert.Equal(t, "on.workflow_run.branches", diagnostics[0].Path)
			assert.Contains(t, diagnostics[0].Message, "branch scope")
		})
	}
}

func TestWorkflowRunScopeStrictPolicy(t *testing.T) {
	tests := []struct {
		name     string
		document *aw.Document
		severity Severity
	}{
		{name: "plain Actions", severity: Warning},
		{name: "AW default", document: &aw.Document{}, severity: Error},
		{name: "AW strict true", document: &aw.Document{Frontmatter: map[string]any{"strict": true}}, severity: Error},
		{name: "AW strict false", document: &aw.Document{Frontmatter: map[string]any{"strict": false}}, severity: Warning},
		{name: "AW malformed strict string", document: &aw.Document{Frontmatter: map[string]any{"strict": "false"}}, severity: Error},
		{name: "AW malformed strict null", document: &aw.Document{Frontmatter: map[string]any{"strict": nil}}, severity: Error},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := workflowRunScopeInput(t, "on: {workflow_run: {workflows: [Build]}}")
			input.Document = tt.document
			diagnostics := runWorkflowRunScope(t, input)
			require.Len(t, diagnostics, 1)
			assert.Equal(t, tt.severity, diagnostics[0].Severity)
		})
	}
}

func TestWorkflowRunScopeParsedAWPolicy(t *testing.T) {
	for _, tt := range []struct {
		source   string
		severity Severity
	}{
		{source: "---\non: workflow_dispatch\n---\nPrompt", severity: Error},
		{source: "---\nstrict: true\n---\nPrompt", severity: Error},
		{source: "---\nstrict: false\n---\nPrompt", severity: Warning},
	} {
		document, err := aw.Parse(tt.source)
		require.NoError(t, err)
		input := workflowRunScopeInput(t, "on: {workflow_run: {workflows: [Build]}}")
		input.Document = document
		diagnostics := runWorkflowRunScope(t, input)
		require.Len(t, diagnostics, 1)
		assert.Equal(t, tt.severity, diagnostics[0].Severity)
	}
}

func TestWorkflowRunScopeExclusiveBranchFilters(t *testing.T) {
	input := workflowRunScopeInput(t, "on: {workflow_run: {workflows: [Build], branches: [main], branches-ignore: [dev]}}")
	diagnostics := runWorkflowRunScope(t, input)
	require.Len(t, diagnostics, 1)
	assert.Equal(t, Error, diagnostics[0].Severity)
	assert.Equal(t, "on.workflow_run.branches-ignore", diagnostics[0].Path)
	assert.Contains(t, diagnostics[0].Message, "cannot combine")
}

func TestWorkflowRunScopeConstructedAST(t *testing.T) {
	for _, on := range []any{
		map[string]any{"workflow_run": map[string]any{"workflows": []string{"Build"}, "branches": []string{"main"}}},
		map[string]any{"workflow_run": map[string]any{"workflows": []any{"Build"}, "branches": []any{"main"}}},
		map[string]any{"workflow_run": map[string]any{"workflows": "Build", "branches": []string{"release/**"}}},
		[]string{"push", "workflow_dispatch"},
		[]any{"push", "workflow_dispatch"},
		"workflow_dispatch",
		nil,
	} {
		assert.Empty(t, runWorkflowRunScope(t, Input{Workflow: &actions.Workflow{On: on}}))
	}
	for _, on := range []any{
		map[string]any{"workflow_run": nil},
		map[string]any{"workflow_run": map[string]any{"workflows": []string{}, "branches": []string{"main"}}},
		map[string]any{"workflow_run": map[string]any{"workflows": []string{"Build", ""}, "branches": []string{"main"}}},
		[]string{"push", "workflow_run"},
		[]any{"push", "workflow_run"},
		"workflow_run",
	} {
		diagnostics := runWorkflowRunScope(t, Input{Workflow: &actions.Workflow{On: on}})
		require.Len(t, diagnostics, 1)
		assert.Equal(t, Error, diagnostics[0].Severity)
	}
	for _, branches := range [][]string{
		nil, {}, {""}, {"!untrusted/**"}, {"**"}, {"main", "**"},
	} {
		input := Input{Workflow: &actions.Workflow{On: map[string]any{
			"workflow_run": map[string]any{"workflows": []string{"Build"}, "branches": branches},
		}}}
		diagnostics := runWorkflowRunScope(t, input)
		require.Len(t, diagnostics, 1)
		assert.Equal(t, Warning, diagnostics[0].Severity)
		assert.Equal(t, "on.workflow_run.branches", diagnostics[0].Path)
	}
}

func TestWorkflowRunScopeReadOnlyValueFreeAndDeterministic(t *testing.T) {
	input := workflowRunScopeInput(t, "on: {workflow_run: {workflows: [PRIVATE_UPSTREAM_NAME], branches: ['PRIVATE_BRANCH', '**']}}")
	before, err := actions.Emit(input.Workflow)
	require.NoError(t, err)
	first := runWorkflowRunScope(t, input)
	require.Len(t, first, 1)
	assert.NotContains(t, first[0].Message, "PRIVATE")
	assert.Equal(t, first, runWorkflowRunScope(t, input))
	after, err := actions.Emit(input.Workflow)
	require.NoError(t, err)
	assert.Equal(t, before, after)
}

func TestWorkflowRunScopeCanceledAndMissingAST(t *testing.T) {
	rule := WorkflowRunScopeRule()
	input := workflowRunScopeInput(t, "on: {workflow_run: {workflows: [Build]}}")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	diagnostics, err := rule.Check(ctx, input)
	require.ErrorIs(t, err, context.Canceled)
	assert.Empty(t, diagnostics)
	_, err = rule.Check(context.Background(), Input{})
	require.ErrorContains(t, err, "requires an Actions or AW AST")
}

type workflowRunCancelContext struct {
	context.Context
	remaining int
}

func (ctx *workflowRunCancelContext) Err() error {
	ctx.remaining--
	if ctx.remaining <= 0 {
		return context.Canceled
	}
	return nil
}

func TestWorkflowRunScopeCancellationDuringTraversal(t *testing.T) {
	for _, on := range []any{
		[]string{"push", "workflow_dispatch", "workflow_run"},
		[]any{"push", "workflow_dispatch", "workflow_run"},
		map[string]any{"workflow_run": map[string]any{
			"workflows": []string{"Build", "Test", "Publish"},
		}},
		map[string]any{"workflow_run": map[string]any{
			"workflows": []any{"Build", "Test", "Publish"},
		}},
		map[string]any{"workflow_run": map[string]any{
			"workflows": "Build", "branches": []string{"main", "release/**"},
		}},
	} {
		ctx := &workflowRunCancelContext{Context: context.Background(), remaining: 3}
		diagnostics, err := WorkflowRunScopeRule().Check(ctx, Input{Workflow: &actions.Workflow{On: on}})
		require.ErrorIs(t, err, context.Canceled)
		assert.Empty(t, diagnostics)
	}
}
