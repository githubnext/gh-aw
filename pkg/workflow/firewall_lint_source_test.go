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

func TestFirewallLintSourceToCompiler(t *testing.T) {
	for _, tt := range []struct {
		name, fields, path, compileError string
	}{
		{
			name:   "accepted URL flags without SSL inspection",
			fields: "strict: false\nsandbox: {agent: {args: [--allow-urls, 'https://example.com/path']}}\n",
			path:   "sandbox.agent.args",
		},
		{
			name:   "accepted equals URL flag without SSL inspection",
			fields: "strict: false\nsandbox: {agent: {args: ['--allow-urls=https://example.com/path']}}\n",
			path:   "sandbox.agent.args",
		},
		{
			name:   "compilable SSL remedy",
			fields: "strict: false\nsandbox: {agent: {args: [--ssl-bump, --allow-urls, 'https://example.com/path']}}\n",
		},
		{
			name:   "compilable SSL remedy after URL filters",
			fields: "strict: false\nsandbox: {agent: {args: [--allow-urls, 'https://example.com/path', --ssl-bump]}}\n",
		},
		{
			name:   "compilable explicit SSL enablement",
			fields: "strict: false\nsandbox: {agent: {args: ['--allow-urls=https://example.com/path', '--ssl-bump=true']}}\n",
		},
		{
			name:   "later SSL disablement remains a lint violation",
			fields: "strict: false\nsandbox: {agent: {args: [--ssl-bump, --allow-urls, 'https://example.com/path', '--ssl-bump=false']}}\n",
			path:   "sandbox.agent.args",
		},
		{
			name:         "legacy missing SSL requires migration",
			fields:       "strict: false\nnetwork: {firewall: {allow-urls: ['https://example.com/path']}}\n",
			path:         "network.firewall",
			compileError: "firewall",
		},
		{
			name:         "legacy SSL true still requires migration",
			fields:       "strict: false\nnetwork: {firewall: {allow-urls: ['https://example.com/path'], ssl-bump: true}}\n",
			path:         "network.firewall",
			compileError: "firewall",
		},
		{
			name:         "SSL does not make strict internal overrides valid",
			fields:       "strict: true\nsandbox: {agent: {args: [--ssl-bump, --allow-urls, 'https://example.com/path']}}\n",
			compileError: "sandbox.agent.args",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			document, err := aw.Parse("---\non: workflow_dispatch\nengine: copilot\npermissions: {contents: read}\n" + tt.fields + "---\nPublic prompt.\n")
			require.NoError(t, err)
			runner, err := workflowlint.Default().Select("firewall-ssl-bump")
			require.NoError(t, err)
			diagnostics, err := runner.Run(context.Background(), workflowlint.Input{Document: document})
			require.NoError(t, err)
			if tt.path == "" {
				assert.Empty(t, diagnostics)
			} else {
				require.Len(t, diagnostics, 1)
				assert.Equal(t, tt.path, diagnostics[0].Path)
				assert.Equal(t, workflowlint.Error, diagnostics[0].Severity)
			}

			compiler := NewCompiler(WithNoEmit(true))
			data, err := compiler.ParseWorkflowDocument(document, filepath.Join(t.TempDir(), "firewall.md"))
			if tt.compileError != "" {
				require.ErrorContains(t, err, tt.compileError)
				return
			}
			require.NoError(t, err, "lint fixtures must pass the actual compiler schema and source validation")
			require.NotNil(t, data.SandboxConfig)
			require.NotNil(t, data.SandboxConfig.Agent)
			require.NotEmpty(t, data.SandboxConfig.Agent.Args)
			compiled, err := compiler.CompileToYAML(data, "firewall.md")
			require.NoError(t, err, "the SSL remedy must compile without skipping validation")
			assert.Contains(t, compiled, "awf --config ")
			for _, arg := range data.SandboxConfig.Agent.Args {
				assert.Contains(t, compiled, arg)
			}
		})
	}
}
