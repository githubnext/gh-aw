package lint

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/github/gh-aw/pkg/workflow/actions"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPullRequestTargetCheckoutPolicy(t *testing.T) {
	const trigger = "on: {pull_request_target: {acknowledge-risk: true}}\n"
	const allow = "on: {pull_request_target: {allowed-checkouts: [{repository: Owner/Repo, ref: main}]}}\n"
	for _, tt := range []struct {
		name, source string
		paths        []string
	}{
		{"omission is not disablement", trigger, []string{"checkout"}},
		{"explicit disablement", trigger + "checkout: false", nil},
		{"base default ref", trigger + "checkout: {}", nil},
		{"base repository", trigger + "checkout: {repository: '${{ github.repository }}'}", nil},
		{"base sha", trigger + "checkout: {ref: '${{github.event.pull_request.base.sha}}'}", nil},
		{"base branch", trigger + "checkout: {ref: '${{ github.event.pull_request.base.ref }}'}", nil},
		{"head sha", trigger + "checkout: {ref: '${{ github.event.pull_request.head.sha }}'}", []string{"checkout"}},
		{"mixed base expression", trigger + "checkout: {ref: '${{ github.event.pull_request.base.sha || github.event.pull_request.head.sha }}'}", []string{"checkout"}},
		{"literal base ref not inferred", trigger + "checkout: {ref: main}", []string{"checkout"}},
		{"empty checkout list", trigger + "checkout: []", []string{"checkout"}},
		{"every checkout", trigger + "checkout: [{}, {ref: '${{ github.event.pull_request.head.sha }}'}, {repository: other/repo, ref: main}]", []string{"checkout[1]", "checkout[2]"}},
		{"base fetch scalar", trigger + "checkout: {fetch: main}", []string{"checkout"}},
		{"base fetch list", trigger + "checkout: {ref: '${{ github.event.pull_request.base.sha }}', fetch: [main]}", []string{"checkout"}},
		{"empty fetch list", trigger + "checkout: {fetch: []}", nil},
		{"exact allowed pair", allow + "checkout: {repository: Owner/Repo, ref: main}", nil},
		{"repository alias", allow + "checkout: {repo: Owner/Repo, ref: main}", nil},
		{"case sensitive repository", allow + "checkout: {repository: owner/repo, ref: main}", []string{"checkout"}},
		{"case sensitive ref", allow + "checkout: {repository: Owner/Repo, ref: Main}", []string{"checkout"}},
		{"external missing ref", allow + "checkout: {repository: Owner/Repo}", []string{"checkout"}},
		{"allowed pair fetch", allow + "checkout: {repository: Owner/Repo, ref: main, fetch: [other]}", []string{"checkout"}},
		{"allowed pair wiki", allow + "checkout: {repository: Owner/Repo, ref: main, wiki: true}", []string{"checkout"}},
		{"unlisted pair", allow + "checkout: {repository: Owner/Repo, ref: release}", []string{"checkout"}},
		{"expression allowlist not trusted", "on: {pull_request_target: {allowed-checkouts: [{repository: Owner/Repo, ref: '${{ inputs.ref }}'}]}}\ncheckout: {repository: Owner/Repo, ref: '${{ inputs.ref }}'}", []string{"checkout"}},
		{"pull ref allowlist not trusted", "on: {pull_request_target: {allowed-checkouts: [{repository: Owner/Repo, ref: refs/pull/1/head}]}}\ncheckout: {repository: Owner/Repo, ref: refs/pull/1/head}", []string{"checkout"}},
		{"wildcard allowlist not trusted", "on: {pull_request_target: {allowed-checkouts: [{repository: Owner/Repo, ref: 'release/*'}]}}\ncheckout: {repository: Owner/Repo, ref: 'release/*'}", []string{"checkout"}},
		{"imported checkout unknown", "imports: [shared.md]\n" + trigger, nil},
		{"imported allowlist unknown", "imports: [shared.md]\n" + trigger + "checkout: {repository: Owner/Repo, ref: main}", nil},
		{"imported external ref unknown", "imports: [shared.md]\n" + trigger + "checkout: {repository: Owner/Repo}", nil},
		{"imported literal repository unknown", "imports: [shared.md]\n" + trigger + "checkout: {ref: main}", nil},
		{"imports cannot allow fetch", "imports: [shared.md]\n" + trigger + "checkout: {fetch: [main]}", []string{"checkout"}},
		{"imports cannot allow head expression", "imports: [shared.md]\n" + trigger + "checkout: {ref: '${{ github.event.pull_request.head.sha }}'}", []string{"checkout"}},
		{"imports cannot allow wildcard", "imports: [shared.md]\n" + allow + "checkout: {repository: Owner/Repo, ref: '*'}", []string{"checkout"}},
		{"suppression is roles only", "features: {dangerously-allow-roles-read-and-pull-request-target: true}\n" + trigger + "checkout: {fetch: [main]}", []string{"checkout"}},
		{"scalar trigger", "on: pull_request_target", []string{"checkout"}},
		{"list trigger", "on: [push, pull_request_target]", []string{"checkout"}},
		{"exact trigger name", "on: {pull_request_target_staging: null}", nil},
		{"other triggers", "on: [push, pull_request]\ncheckout: {ref: '${{ github.event.pull_request.head.sha }}'}", nil},
	} {
		t.Run(tt.name, func(t *testing.T) {
			for _, strict := range []string{"", "strict: false\n"} {
				input := policyTestInput(t, strict+tt.source)
				before, err := json.Marshal(input)
				require.NoError(t, err)
				runner, err := Default().Select("pull-request-target-checkout")
				require.NoError(t, err)
				diagnostics, err := runner.Run(context.Background(), input)
				require.NoError(t, err)
				var paths []string
				for _, diagnostic := range diagnostics {
					paths = append(paths, diagnostic.Path)
					severity := Error
					if strict != "" {
						severity = Warning
					}
					assert.Equal(t, severity, diagnostic.Severity)
					assert.Equal(t, "CTR-008", diagnostic.ThreatRuleID)
				}
				assert.Equal(t, tt.paths, paths)
				after, err := json.Marshal(input)
				require.NoError(t, err)
				assert.Equal(t, before, after)
			}
		})
	}
}

func TestPullRequestTargetCheckoutFailures(t *testing.T) {
	for _, fields := range []string{
		"checkout: PRIVATE_VALUE",
		"checkout: true",
		"checkout: [null]",
		"checkout: {ref: [PRIVATE_VALUE]}",
		"checkout: {repo: 42}",
		"checkout: {repo: Owner/Repo, repository: Other/Repo}",
		"checkout: {fetch: 42}",
		"checkout: {fetch: [PRIVATE_VALUE, 42]}",
		"checkout: {fetch: ''}",
		"checkout: {wiki: PRIVATE_VALUE}",
		"imports: PRIVATE_VALUE",
		"strict: PRIVATE_VALUE",
		"on: {pull_request_target: PRIVATE_VALUE}",
		"on: {pull_request_target: {allowed-checkouts: PRIVATE_VALUE}}",
		"on: {pull_request_target: {allowed-checkouts: [PRIVATE_VALUE]}}",
		"on: {pull_request_target: {allowed-checkouts: [{repository: PRIVATE_VALUE}]}}",
	} {
		t.Run(fields, func(t *testing.T) {
			source := fields
			if !strings.HasPrefix(fields, "on:") {
				source = "on: pull_request_target\n" + fields
			}
			input := policyTestInput(t, source)
			_, err := PullRequestTargetCheckoutRule().Check(context.Background(), input)
			require.Error(t, err)
			assert.NotContains(t, err.Error(), "PRIVATE_VALUE")
		})
	}
}

func TestPullRequestTargetCheckoutPartialFindings(t *testing.T) {
	input := policyTestInput(t, "on: pull_request_target\ncheckout: [{ref: '${{ github.event.pull_request.head.sha }}'}, null, {fetch: main}]")
	diagnostics, err := PullRequestTargetCheckoutRule().Check(context.Background(), input)
	require.Error(t, err)
	require.Len(t, diagnostics, 2)
	ctx := &integrityCancelContext{Context: context.Background(), remaining: 5}
	_, err = PullRequestTargetCheckoutRule().Check(ctx, input)
	require.ErrorIs(t, err, context.Canceled)
}

func TestNewThreatRulesRequireAuthoredEvidence(t *testing.T) {
	for _, rule := range []Rule{PullRequestTargetCheckoutRule(), PromptExpressionRule()} {
		runner, err := New(rule)
		require.NoError(t, err)
		diagnostics, err := runner.Run(context.Background(), Input{Workflow: &actions.Workflow{On: "pull_request_target"}})
		require.NoError(t, err)
		assert.Empty(t, diagnostics)
		_, err = rule.Check(context.Background(), Input{})
		require.Error(t, err)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		_, err = rule.Check(ctx, policyTestInput(t, "on: pull_request_target"))
		assert.ErrorIs(t, err, context.Canceled)
	}
}
