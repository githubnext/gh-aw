package lint

import (
	"context"
	"fmt"
	"testing"

	"github.com/github/gh-aw/pkg/workflow/actions"
	"github.com/github/gh-aw/pkg/workflow/aw"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuiltInRulesOnAWAST(t *testing.T) {
	for _, tt := range []struct {
		id, frontmatter, markdown, path string
	}{
		{"action-pin", "post-steps:\n  - uses: actions/checkout@v4", "", "post-steps[0].uses"},
		{"action-pin", "safe-jobs:\n  publish:\n    steps:\n      - uses: actions/checkout@v4", "", "safe-jobs.publish.steps[0].uses"},
		{"action-pin", "jobs:\n  reuse:\n    uses: owner/repo/.github/workflows/test.yml@main", "", "jobs.reuse.uses"},
		{"job-timeout", "timeout-minutes: 0", "", "timeout-minutes"},
		{"job-timeout", "jobs:\n  test:\n    timeout-minutes: '${{ inputs.timeout }}'", "", "jobs.test.timeout-minutes"},
		{"template-injection", "steps:\n  - run: echo '${{ github.event.issue.title }}'", "", "steps[0].run"},
		{"workflow-run-scope", "on:\n  workflow_run:\n    workflows: [CI]", "", "on.workflow_run.branches"},
		{"permission-scope", "permissions: write-all", "", "permissions"},
		{"permission-scope", "safe-jobs:\n  publish:\n    permissions:\n      id-token: read", "", "safe-jobs.publish.permissions.id-token"},
		{"version-check", "check-for-updates: false", "", "check-for-updates"},
		{"conditional-import", "imports:\n  - path: shared.md\n    if: false", "", "imports[0].if"},
		{"private-to-public", "tools:\n  github:\n    private-to-public-flows: allow", "", "tools.github.private-to-public-flows"},
		{"install-scripts", "runtimes:\n  node:\n    run-install-scripts: true", "", "runtimes.node.run-install-scripts"},
		{"allowed-labels", "safe-outputs:\n  create-issue:\n    allowed-labels: ['*']", "", "safe-outputs.create-issue.allowed-labels"},
		{"network-scope", "network:\n  allowed: ['*']", "", "network.allowed"},
		{"push-target", "safe-outputs:\n  push-to-pull-request-branch:\n    target: '*'", "", "safe-outputs.push-to-pull-request-branch.target"},
		{"secret-leak", "env:\n  TOKEN: '${{ secrets.TOKEN }}'\nsteps:\n  - shell: bash\n    run: echo \"$TOKEN\"", "", "steps[0].run"},
		{"secret-leak", "env:\n  TOKEN: '${{ secrets.TOKEN }}'", "${{ env.TOKEN }}", "markdown"},
		{"secret-leak", "post-steps:\n  - shell: bash\n    run: echo '${{ secrets.TOKEN }}'", "", "post-steps[0].run"},
	} {
		t.Run(tt.id+"/"+tt.path, func(t *testing.T) {
			document, err := aw.Parse("---\n" + tt.frontmatter + "\n---\n" + tt.markdown)
			require.NoError(t, err)
			before, err := aw.Emit(document)
			require.NoError(t, err)
			runner, err := Default().Select(tt.id)
			require.NoError(t, err)
			diagnostics, err := runner.Run(context.Background(), Input{Document: document})
			require.NoError(t, err)
			var paths []string
			for _, diagnostic := range diagnostics {
				if diagnostic.Path != "workflow" {
					paths = append(paths, diagnostic.Path)
				}
			}
			assert.Contains(t, paths, tt.path)
			after, err := aw.Emit(document)
			require.NoError(t, err)
			assert.Equal(t, before, after, "AST rules must leave the document unchanged")
		})
	}
}

func TestASTRuleApplicability(t *testing.T) {
	called := make(map[string]bool)
	makeRule := func(id string, targets ASTTargets) Rule {
		return Rule{
			ID: id, Description: id, Severity: Warning, Targets: targets,
			Check: func(context.Context, Input) ([]Diagnostic, error) { called[id] = true; return nil, nil },
		}
	}
	runner, err := New(makeRule("legacy", 0), makeRule("aw-only", AWAST), makeRule("both", ActionsAST|AWAST))
	require.NoError(t, err)
	_, err = runner.Run(context.Background(), Input{Document: &aw.Document{}})
	require.NoError(t, err)
	assert.Equal(t, map[string]bool{"aw-only": true, "both": true}, called)
	clear(called)
	_, err = runner.Run(context.Background(), Input{Workflow: &actions.Workflow{}})
	require.NoError(t, err)
	assert.Equal(t, map[string]bool{"legacy": true, "both": true}, called)
	_, err = New(makeRule("invalid-target", ASTTargets(128)))
	require.Error(t, err)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = runner.Run(ctx, Input{Document: &aw.Document{}})
	require.ErrorIs(t, err, context.Canceled)
}

func TestAWASTPartialDiagnostics(t *testing.T) {
	document, err := aw.Parse("---\nsteps:\n  - uses: actions/checkout@v4\n  - VERY_SENSITIVE_SCALAR\njobs:\n  bad: VERY_SENSITIVE_SCALAR\n---\n")
	require.NoError(t, err)
	runner, err := Default().Select("action-pin")
	require.NoError(t, err)
	diagnostics, err := runner.Run(context.Background(), Input{Document: document})
	require.Error(t, err)
	require.Len(t, diagnostics, 1)
	assert.Equal(t, "steps[0].uses", diagnostics[0].Path)
	assert.NotContains(t, err.Error(), "VERY_SENSITIVE_SCALAR")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = aw.ExecutionBlocks(ctx, document)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestActionPinMalformedAuthoredScopes(t *testing.T) {
	for _, scope := range []struct {
		fields string
		path   string
	}{
		{"steps: [{uses: %s}]", "steps[0].uses"},
		{"post-steps: [{uses: %s}]", "post-steps[0].uses"},
		{"safe-outputs: {steps: [{uses: %s}]}", "safe-outputs.steps[0].uses"},
		{"safe-outputs: {post-steps: [{uses: %s}]}", "safe-outputs.post-steps[0].uses"},
		{"jobs: {check: {steps: [{uses: %s}]}}", "jobs.check.steps[0].uses"},
		{"safe-jobs: {check: {steps: [{uses: %s}]}}", "safe-jobs.check.steps[0].uses"},
		{"safe-outputs: {jobs: {check: {steps: [{uses: %s}]}}}", "safe-outputs.jobs.check.steps[0].uses"},
		{"jobs: {check: {uses: %s}}", "jobs.check.uses"},
	} {
		for _, pin := range []string{`"./action\u2028PRIVATE_PIN"`, `"actions/checkout@${{ secrets.PRIVATE_PIN }}"`, `"actions/checkout@"`, `"docker://alpine@sha256:PRIVATE_PIN"`} {
			t.Run(scope.path+pin, func(t *testing.T) {
				for _, strict := range []bool{false, true} {
					source := fmt.Sprintf("---\nstrict: %t\n"+scope.fields+"\n---\n", strict, pin)
					document, err := aw.Parse(source)
					require.NoError(t, err)
					diagnostics := integrityTestRun(t, ActionPinRule(), Input{Document: document})
					require.Len(t, diagnostics, 1)
					assert.Equal(t, Error, diagnostics[0].Severity)
					assert.Equal(t, scope.path, diagnostics[0].Path)
					assert.NotContains(t, diagnostics[0].Message, "PRIVATE_PIN")
				}
			})
		}
	}
}
