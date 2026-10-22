package lint

import (
	"context"
	"fmt"
	"testing"

	"github.com/github/gh-aw/pkg/workflow/actions"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGitHubScriptTemplateInjection(t *testing.T) {
	for _, test := range []struct {
		name, uses, script string
		want               bool
	}{
		{"direct", "actions/github-script@v8", "console.log('${{ github.event.issue.body }}')", true},
		{"serialized input is still code", "actions/github-script@v8", "const data = ${{ toJSON(inputs.title) }}", true},
		{"environment alias", "actions/github-script@v8", "const title = '${{ env.TITLE }}'", true},
		{"runtime environment", "actions/github-script@v8", "console.log(process.env.TITLE)", false},
		{"trusted expression", "actions/github-script@v8", "console.log('${{ github.run_id }}')", false},
		{"boolean filter", "actions/github-script@v8", "const present = ${{ contains(inputs.title, 'text') }}", false},
		{"other script input is data", "owner/github-script@v8", "${{ inputs.title }}", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			workflow := &actions.Workflow{
				Env: map[string]actions.Expression{"TITLE": actions.Context("github.event.issue.title")},
				Jobs: []*actions.Job{{ID: "check", RunsOn: "ubuntu-latest", Steps: []actions.Step{
					actions.RawStep{Fields: map[string]any{
						"uses": test.uses, "with": map[string]any{"script": "/* public */\n" + test.script},
					}},
				}}},
			}
			diagnostics := runTemplateInjection(t, workflow)
			if test.want {
				require.Len(t, diagnostics, 1)
				assert.Equal(t, "jobs.check.steps[0].with.script", diagnostics[0].Path)
				assert.Equal(t, 2, diagnostics[0].Line)
				assert.Contains(t, diagnostics[0].Message, "process.env")
			} else {
				assert.Empty(t, diagnostics)
			}
		})
	}
}

func TestGitHubScriptTemplateInjectionFailures(t *testing.T) {
	for _, with := range []any{"PRIVATE_VALUE", map[string]any{"script": []string{"PRIVATE_VALUE"}}} {
		workflow := templateInjectionWorkflow("echo public", false)
		workflow.Jobs[0].Steps[0] = actions.RawStep{Fields: map[string]any{"uses": "actions/github-script@v8", "with": with}}
		_, err := TemplateInjectionRule().Check(context.Background(), Input{Workflow: workflow})
		require.Error(t, err)
		assert.NotContains(t, err.Error(), "PRIVATE_VALUE")
	}
	workflow := templateInjectionWorkflow("echo public", false)
	workflow.Jobs[0].Steps[0] = actions.RawStep{Fields: map[string]any{
		"uses": "actions/github-script@v8",
		"with": map[string]any{"script": "${{ inputs.title }}\n${{ inputs.PRIVATE_VALUE + }}"},
	}}
	diagnostics := runTemplateInjection(t, workflow)
	require.Len(t, diagnostics, 2)
	assert.Equal(t, Error, diagnostics[0].Severity)
	assert.Equal(t, Warning, diagnostics[1].Severity)
	assert.NotContains(t, fmt.Sprint(diagnostics), "PRIVATE_VALUE")
}
