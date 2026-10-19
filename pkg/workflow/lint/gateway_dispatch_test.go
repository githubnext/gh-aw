package lint

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/github/gh-aw/pkg/workflow/actions"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGatewayDispatchLiteralsRule(t *testing.T) {
	for _, tt := range []struct {
		name, source string
		paths        []string
	}{
		{"literals", `on: {gateway-dispatch: {repos: ["github/*"], issue: {filters: {labels: [triage]}}, pull-request: {filters: {draft: false}}}}`, nil},
		{"native trigger", `on: issues`, nil},
		{"outside section", `env: {REPO: "${{ github.repository }}"}
on: {github-token: "${{ secrets.PRIVATE_VALUE }}", gateway-dispatch: {issue: {}}}`, nil},
		{"whole section", `on: {gateway-dispatch: "${{ vars.PRIVATE_VALUE }}"}`, []string{"on.gateway-dispatch"}},
		{"trigger key", `on: {gateway-dispatch: {"${{ vars.PRIVATE_VALUE }}": {}}}`, []string{"on.gateway-dispatch property name"}},
		{"nested key", `on: {gateway-dispatch: {issue: {filters: {"${{ vars.PRIVATE_VALUE }}": triage}}}}`, []string{"on.gateway-dispatch.issue.filters property name"}},
		{"nested list", `on: {gateway-dispatch: {issue: {filters: {labels: [triage, "${{ vars.PRIVATE_VALUE }}"]}}}}`, []string{"on.gateway-dispatch.issue.filters.labels[1]"}},
		{"escaped marker", `on: {gateway-dispatch: {issue: {filters: {labels: ["\u0024{{ vars.PRIVATE_VALUE }}"]}}}}`, []string{"on.gateway-dispatch.issue.filters.labels[0]"}},
		{"incomplete marker", `on: {gateway-dispatch: {slash-command: "prefix-${{PRIVATE_VALUE"}}`, []string{"on.gateway-dispatch.slash-command"}},
		{"every finding", `on: {gateway-dispatch: {slash-command: "${{ vars.PRIVATE_VALUE }}", issue: {filters: {labels: ["${{ vars.PRIVATE_VALUE }}", "${{"]}}}}`, []string{
			"on.gateway-dispatch.issue.filters.labels[0]", "on.gateway-dispatch.issue.filters.labels[1]", "on.gateway-dispatch.slash-command",
		}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			for _, strict := range []string{"", "strict: false\n"} {
				input := policyTestInput(t, strict+tt.source)
				input.Document.Markdown = "Prompt expressions remain supported: ${{ toJSON(gateway.events) }}"
				before, err := json.Marshal(input)
				require.NoError(t, err)
				runner, err := Default().Select("gateway-dispatch-literals")
				require.NoError(t, err)
				diagnostics, err := runner.Run(context.Background(), input)
				require.NoError(t, err)
				var paths []string
				for _, diagnostic := range diagnostics {
					paths = append(paths, diagnostic.Path)
					assert.Equal(t, "gateway-dispatch-literals", diagnostic.RuleID)
					assert.Equal(t, Error, diagnostic.Severity)
				}
				assert.Equal(t, tt.paths, paths)
				encoded, err := json.Marshal(diagnostics)
				require.NoError(t, err)
				assert.NotContains(t, string(encoded), "PRIVATE_VALUE")
				after, err := json.Marshal(input)
				require.NoError(t, err)
				assert.Equal(t, before, after, "lint must not mutate declarations")
			}
		})
	}
}

func TestGatewayDispatchLiteralsRuleScopeAndCancellation(t *testing.T) {
	rule := GatewayDispatchLiteralsRule()
	diagnostics, err := rule.Check(context.Background(), Input{Workflow: &actions.Workflow{}})
	require.NoError(t, err)
	assert.Empty(t, diagnostics, "compiled Actions are not authored gateway declarations")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = rule.Check(ctx, policyTestInput(t, "on: {gateway-dispatch: {issue: {}}}"))
	assert.ErrorIs(t, err, context.Canceled)
}
