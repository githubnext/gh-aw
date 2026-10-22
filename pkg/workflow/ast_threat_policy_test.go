//go:build !integration

package workflow

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/github/gh-aw/pkg/workflow/aw"
	workflowlint "github.com/github/gh-aw/pkg/workflow/lint"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckoutLintSourceToCompiler(t *testing.T) {
	const trigger = "on: {pull_request_target: {acknowledge-risk: true}}\n"
	const allow = "on: {pull_request_target: {acknowledge-risk: true, allowed-checkouts: [{repository: Owner/Repo, ref: main}]}}\n"
	for _, tt := range []struct {
		name, fields string
		unsafe       bool
	}{
		{"disabled", trigger + "checkout: false\n", false},
		{"base default", trigger + "checkout: {}\n", false},
		{"base sha", trigger + "checkout: {ref: '${{ github.event.pull_request.base.sha }}'}\n", false},
		{"allowed literal", allow + "checkout: {repository: Owner/Repo, ref: main}\n", false},
		{"head sha", trigger + "checkout: {ref: '${{ github.event.pull_request.head.sha }}'}\n", true},
		{"base fetch", trigger + "checkout: {fetch: [main]}\n", true},
		{"case mismatch", allow + "checkout: {repository: owner/repo, ref: main}\n", true},
		{"allowed fetch", allow + "checkout: {repository: Owner/Repo, ref: main, fetch: [main]}\n", true},
		{"allowed wiki", allow + "checkout: {repository: Owner/Repo, ref: main, wiki: true}\n", true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			document, err := aw.Parse("---\nengine: copilot\npermissions: {contents: read}\n" + tt.fields + "---\nPublic prompt.\n")
			require.NoError(t, err)
			runner, err := workflowlint.Default().Select("pull-request-target-checkout")
			require.NoError(t, err)
			diagnostics, err := runner.Run(context.Background(), workflowlint.Input{Document: document})
			require.NoError(t, err)
			assert.Equal(t, tt.unsafe, len(diagnostics) > 0)

			compiler := NewCompiler(WithNoEmit(true))
			data, err := compiler.ParseWorkflowDocument(document, filepath.Join(t.TempDir(), "checkout.md"))
			require.NoError(t, err, "fixtures must pass the actual authored schema")
			_, err = compiler.CompileToYAML(data, "checkout.md")
			if tt.unsafe {
				require.ErrorContains(t, err, "extremely insecure")
			} else {
				require.NoError(t, err, "lint remedies must compile")
			}
		})
	}
}

func TestPromptLintCompilerPolicy(t *testing.T) {
	for _, tt := range []struct {
		prompt string
		unsafe bool
	}{
		{"Repository: ${{ github.repository }}", false},
		{"${{\n github.repository\n}}", true},
		{"${{ env.CONFIG.constructor }}", true},
		{"${{ env.CONFIG['__proto__'] }}", true},
	} {
		t.Run(tt.prompt, func(t *testing.T) {
			document, err := aw.Parse(tt.prompt)
			require.NoError(t, err)
			diagnostics, err := workflowlint.PromptExpressionRule().Check(context.Background(), workflowlint.Input{Document: document})
			require.NoError(t, err)
			assert.Equal(t, tt.unsafe, len(diagnostics) > 0)
			assert.Equal(t, tt.unsafe, validateExpressionSafety(tt.prompt) != nil)
		})
	}
}
