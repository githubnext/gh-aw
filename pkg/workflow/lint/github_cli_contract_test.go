package lint

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/github/gh-aw/pkg/workflow/actions"
	"github.com/github/gh-aw/pkg/workflow/aw"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGitHubCLICommandSyntax(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name, script string
		count        int
	}{
		{"observed pager", "gh pr view 42 --no-pager", 1},
		{"global pager before subcommand", "gh --no-pager pr view 42", 1},
		{"observed PR field", "gh pr view 42 --json number,authorAssociation,files", 1},
		{"list field", "gh pr list --json=number,authorAssociation", 1},
		{"both mistakes", "gh pr view 42 --no-pager --json 'number,authorAssociation'", 2},
		{"expression before pager flag", "gh pr view ${{ github.event.pull_request.number }} --no-pager", 1},
		{"multiline expression preserves source lines", "gh pr view ${{\n github.event.pull_request.number\n}} --no-pager\necho done\ngh pr view 42 --no-pager", 2},
		{"non-gh command with expression", "sighthound --sha ${{ github.sha }}", 0},
		{"template value resembles pager flag", "gh pr view 42 --json title --template '--no-pager'", 0},
		{"template value resembles invalid JSON field", "gh pr view 42 --json title --template '--json=authorAssociation'", 0},
		{"literal flag beside template value", "gh pr view 42 --json title --template '--no-pager' --no-pager", 1},
		{"chain", "echo start && gh pr diff 42 --no-pager | head -100", 1},
		{"assignment", "GH_PAGER=cat gh pr view 42 --json number,title", 0},
		{"REST field", "gh api repos/example/project/pulls/42 --jq '.author_association'", 0},
		{"GraphQL field is valid", "gh api graphql -f query='query { repository { authorAssociation } }'", 0},
		{"unsupported field not selected", "gh pr view 42 --json number,title --jq '.authorAssociation'", 0},
		{"quoted command is data", `echo 'gh pr view 42 --no-pager'`, 0},
		{"comment is data", "# gh pr view 42 --no-pager\ngh pr view 42", 0},
		{"other client", "git --no-pager diff", 0},
		{"dynamic JSON selection defers", `gh pr view 42 --json "$FIELDS"`, 0},
		{"dynamic executable defers", `"$CLIENT" pr view 42 --no-pager`, 0},
		{"option terminator", "gh pr view -- --no-pager", 0},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			workflow := &actions.Workflow{Jobs: []*actions.Job{{
				ID: "check", Steps: []actions.Step{actions.RawStep{Fields: map[string]any{"run": test.script, "shell": "bash"}}},
			}}}
			runner, err := New(GitHubCLISyntaxRule())
			require.NoError(t, err)
			findings, err := runner.Run(t.Context(), Input{Workflow: workflow})
			require.NoError(t, err)
			require.Len(t, findings, test.count)
			for _, finding := range findings {
				assert.Equal(t, "jobs.check.steps[0].run", finding.Path)
				assert.Equal(t, Warning, finding.Severity)
				assert.Equal(t, "github-cli-syntax", finding.RuleID)
				if test.name == "multiline expression preserves source lines" && finding.Line == 5 {
					continue
				}
				assert.Equal(t, 1, finding.Line)
			}
		})
	}
}

func TestGitHubCLIExecutionScriptsResolveInheritedShells(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name                               string
		workflowShell, jobShell, stepShell string
		want                               int
	}{
		{"workflow Python default", "python", "", "", 0},
		{"job Bash overrides workflow Python", "python", "bash", "", 1},
		{"job Python overrides workflow Bash", "bash", "python", "", 0},
		{"step Python overrides job Bash", "", "bash", "python", 0},
		{"step Bash overrides job Python", "", "python", "bash", 1},
		{"unspecified shell defaults to Bash", "", "", "", 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			defaults := func(shell string) map[string]any {
				if shell == "" {
					return nil
				}
				return map[string]any{"run": map[string]any{"shell": shell}}
			}
			fields := map[string]any{"run": `print("gh")`}
			if test.stepShell != "" {
				fields["shell"] = test.stepShell
			}
			workflow := &actions.Workflow{
				Defaults: defaults(test.workflowShell),
				Jobs: []*actions.Job{{
					ID: "check", Defaults: defaults(test.jobShell),
					Steps: []actions.Step{actions.RawStep{Fields: fields}},
				}},
			}
			scripts, err := githubCLIExecutionScripts(t.Context(), Input{Workflow: workflow}, false)
			require.NoError(t, err)
			require.Len(t, scripts, test.want)
		})
	}
}

func TestGitHubCLIAuthoredPromptBoundaries(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name, prompt string
		count        int
	}{
		{"literal imperative", "Run `gh pr view 42 --no-pager`.", 1},
		{"negative wording after positive command", "Run `gh pr view 42 --no-pager` if the response is not cached.", 1},
		{"inline command with example repository", "Run `gh pr view 42 --no-pager --repo example/project`.", 1},
		{"list imperative", "- Run `gh pr view 42 --json authorAssociation`.", 1},
		{"unquoted imperative", "Run gh pr diff 42 --no-pager.", 1},
		{"unquoted command with example repository", "Run gh pr view 42 --no-pager --repo example/project.", 1},
		{"shell fence", "Run the following:\n\n```bash\necho start\ngh pr view 42 --no-pager\n```", 1},
		{"tilde fence", "Please run the following:\n~~~sh\ngh pr view 42 --json authorAssociation\n~~~", 1},
		{"negated instruction", "Do not run `gh pr view 42 --no-pager`.", 0},
		{"negative fence", "Run no examples:\n```bash\ngh pr view 42 --no-pager\n```", 0},
		{"quoted instruction", "> Run `gh pr view 42 --no-pager`.", 0},
		{"ordinary prose", "Some clients use gh pr view 42 --no-pager.", 0},
		{"example fence", "Example:\n```bash\ngh pr view 42 --no-pager\n```", 0},
		{"non-shell fence", "Run the following:\n```json\n\"gh pr view 42 --no-pager\"\n```", 0},
		{"HTML comment", "<!--\nRun `gh pr view 42 --no-pager`.\n-->", 0},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			runner, err := New(GitHubCLISyntaxRule())
			require.NoError(t, err)
			findings, err := runner.Run(t.Context(), Input{Document: &aw.Document{Markdown: test.prompt}})
			require.NoError(t, err)
			require.Len(t, findings, test.count)
			for _, finding := range findings {
				assert.Equal(t, "prompt", finding.Path)
				if test.name == "shell fence" {
					assert.Equal(t, 5, finding.Line)
				}
			}
		})
	}
}

func TestGitHubProxyDiffOnlyChecksAgentRoute(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name, mode, prompt string
		count              int
	}{
		{"observed proxy path", "gh-proxy", "Run `gh pr diff 42`.", 1},
		{"inline command with example repository", "gh-proxy", "Run `gh pr diff 42 --repo example/project`.", 1},
		{"unquoted command with example repository", "gh-proxy", "Run gh pr diff 42 --repo example/project.", 1},
		{"normal host client", "local", "Run `gh pr diff 42`.", 0},
		{"unknown imported mode", "", "Run `gh pr diff 42`.", 0},
		{"native metadata command", "gh-proxy", "Run `gh pr view 42 --json number,title`.", 0},
		{"local snapshot", "gh-proxy", "Read the supplied diff.patch using native file readers.", 0},
		{"denied example", "gh-proxy", "Never run `gh pr diff 42`.", 0},
		{"file paths not raw diff", "gh-proxy", "Run `gh pr diff 42 --name-only`.", 0},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			document := &aw.Document{
				Frontmatter: map[string]any{
					"tools": map[string]any{"github": map[string]any{"mode": test.mode}},
					"steps": []any{map[string]any{"run": "gh pr diff 42"}},
				},
				Markdown: test.prompt,
			}
			runner, err := New(GitHubProxyDiffRule())
			require.NoError(t, err)
			findings, err := runner.Run(t.Context(), Input{Document: document})
			require.NoError(t, err)
			require.Len(t, findings, test.count, "host pre-agent steps must not inherit proxy restrictions")
		})
	}
}

func TestGitHubCLIWarningsPreserveCoverageAndValues(t *testing.T) {
	t.Parallel()
	runner, err := New(GitHubCLISyntaxRule(), GitHubProxyDiffRule())
	require.NoError(t, err)
	document := &aw.Document{Frontmatter: map[string]any{
		"steps": []any{map[string]any{"run": "gh pr view PRIVATE_VALUE --no-pager"}, map[string]any{"run": "gh pr view 'unterminated"}},
	}}
	findings, err := runner.Run(t.Context(), Input{Document: document})
	require.NoError(t, err)
	require.Len(t, findings, 2)
	data, err := json.Marshal(findings)
	require.NoError(t, err)
	assert.NotContains(t, string(data), "PRIVATE_VALUE")
	assert.Contains(t, string(data), "Incomplete GitHub CLI syntax analysis")
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err = runner.Run(ctx, Input{Document: document})
	require.ErrorIs(t, err, context.Canceled)
}
