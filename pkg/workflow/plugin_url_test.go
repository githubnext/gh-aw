//go:build !integration

package workflow

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseGitHubPluginURL(t *testing.T) {
	tests := []struct {
		name            string
		input           string
		wantMarketplace string
		wantPluginID    string
		wantOK          bool
	}{
		{
			name:            "full GitHub tree URL",
			input:           "https://github.com/github/copilot-plugins/tree/main/plugins/advanced-security/skills/secret-scanning",
			wantMarketplace: "https://github.com/github/copilot-plugins",
			wantPluginID:    "advanced-security/skills/secret-scanning",
			wantOK:          true,
		},
		{
			name:            "shallow plugin path",
			input:           "https://github.com/acme/my-plugins/tree/main/tools/linter",
			wantMarketplace: "https://github.com/acme/my-plugins",
			wantPluginID:    "tools/linter",
			wantOK:          true,
		},
		{
			name:   "plain plugin name – not a URL",
			input:  "my-extension",
			wantOK: false,
		},
		{
			name:   "repo root URL without tree path",
			input:  "https://github.com/github/copilot-plugins",
			wantOK: false,
		},
		{
			name:   "GitHub URL missing plugin path segment",
			input:  "https://github.com/github/copilot-plugins/tree/main",
			wantOK: false,
		},
		{
			name:   "non-GitHub HTTPS URL",
			input:  "https://example.com/some/plugin",
			wantOK: false,
		},
		{
			name:   "empty string",
			input:  "",
			wantOK: false,
		},
		{
			name:            "http scheme",
			input:           "http://github.com/org/repo/tree/main/plugins/foo",
			wantMarketplace: "http://github.com/org/repo",
			wantPluginID:    "foo",
			wantOK:          true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			marketplace, pluginID, ok := parseGitHubPluginURL(tt.input)
			assert.Equal(t, tt.wantOK, ok, "ok mismatch")
			if tt.wantOK {
				assert.Equal(t, tt.wantMarketplace, marketplace, "marketplace mismatch")
				assert.Equal(t, tt.wantPluginID, pluginID, "pluginID mismatch")
			}
		})
	}
}

func TestNormalizePlugins(t *testing.T) {
	tests := []struct {
		name                     string
		input                    []string
		wantNormalized           []string
		wantInferredMarketplaces []string
	}{
		{
			name:                     "plain names only",
			input:                    []string{"plugin-a", "plugin-b"},
			wantNormalized:           []string{"plugin-a", "plugin-b"},
			wantInferredMarketplaces: nil,
		},
		{
			name: "URL entry is replaced by plugin path; marketplace inferred",
			input: []string{
				"https://github.com/github/copilot-plugins/tree/main/plugins/advanced-security/skills/secret-scanning",
			},
			wantNormalized:           []string{"advanced-security/skills/secret-scanning"},
			wantInferredMarketplaces: []string{"https://github.com/github/copilot-plugins"},
		},
		{
			name: "mix of plain names and URLs",
			input: []string{
				"my-extension",
				"https://github.com/github/copilot-plugins/tree/main/plugins/advanced-security/skills/secret-scanning",
				"another-plugin",
			},
			wantNormalized: []string{
				"my-extension",
				"advanced-security/skills/secret-scanning",
				"another-plugin",
			},
			wantInferredMarketplaces: []string{"https://github.com/github/copilot-plugins"},
		},
		{
			name:                     "empty input",
			input:                    nil,
			wantNormalized:           nil,
			wantInferredMarketplaces: nil,
		},
		{
			name: "two URLs from same repo produce two inferred marketplace entries (dedup happens upstream)",
			input: []string{
				"https://github.com/github/copilot-plugins/tree/main/plugins/advanced-security/skills/secret-scanning",
				"https://github.com/github/copilot-plugins/tree/main/plugins/code-review/skills/review",
			},
			wantNormalized: []string{
				"advanced-security/skills/secret-scanning",
				"code-review/skills/review",
			},
			wantInferredMarketplaces: []string{
				"https://github.com/github/copilot-plugins",
				"https://github.com/github/copilot-plugins",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			normalized, inferred := normalizePlugins(tt.input)
			assert.Equal(t, tt.wantNormalized, normalized, "normalized plugins mismatch")
			assert.Equal(t, tt.wantInferredMarketplaces, inferred, "inferred marketplaces mismatch")
		})
	}
}

// TestCopilotEngineGetBuiltinMarketplaces verifies that the Copilot engine
// declares the correct built-in marketplace URLs.
func TestCopilotEngineGetBuiltinMarketplaces(t *testing.T) {
	engine := &CopilotEngine{}
	builtins := engine.GetBuiltinMarketplaces()
	require.NotEmpty(t, builtins, "Copilot engine should declare at least one built-in marketplace")
	assert.Contains(t, builtins, "https://github.com/github/copilot-plugins",
		"Copilot CLI pre-registers github/copilot-plugins")
}

// TestClaudeEngineGetBuiltinMarketplaces verifies that the Claude engine
// declares no built-in marketplaces.
func TestClaudeEngineGetBuiltinMarketplaces(t *testing.T) {
	engine := &ClaudeEngine{}
	builtins := engine.GetBuiltinMarketplaces()
	assert.Empty(t, builtins, "Claude engine should have no built-in marketplaces")
}

// TestBuiltinMarketplacesFilteredFromCompilationOutput verifies that a
// marketplace URL inferred from a plugin URL is not emitted as a setup step
// when it matches a built-in marketplace for the engine.
func TestBuiltinMarketplacesFilteredFromCompilationOutput(t *testing.T) {
	// The secret-scanning skill lives in copilot-plugins, which is a built-in
	// marketplace.  The generated lock file must NOT contain a
	// "copilot plugin marketplace add https://github.com/github/copilot-plugins"
	// step, but MUST contain the plugin install step.
	tmpDir := t.TempDir()
	workflowsDir := tmpDir + "/.github/workflows"
	require.NoError(t, os.MkdirAll(workflowsDir, 0755), "create workflows dir")

	content := `---
name: Test Builtin Marketplace Filter
on:
  workflow_dispatch:
engine: copilot
imports:
  plugins:
    - https://github.com/github/copilot-plugins/tree/main/plugins/advanced-security/skills/secret-scanning
---
Test builtin marketplace filtering.
`
	workflowFile := workflowsDir + "/test-workflow.md"
	require.NoError(t, os.WriteFile(workflowFile, []byte(content), 0600), "write workflow")

	orig, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(orig) //nolint:errcheck
	require.NoError(t, os.Chdir(tmpDir))

	compiler := NewCompiler()
	require.NoError(t, compiler.CompileWorkflow(workflowFile), "compile workflow")

	lockFile := strings.Replace(workflowFile, ".md", ".lock.yml", 1)
	lockYAML, err := os.ReadFile(lockFile)
	require.NoError(t, err)

	yamlStr := string(lockYAML)
	assert.NotContains(t, yamlStr,
		"copilot plugin marketplace add https://github.com/github/copilot-plugins",
		"built-in marketplace must not be re-registered")
	assert.Contains(t, yamlStr,
		"copilot plugin install advanced-security/skills/secret-scanning",
		"plugin install step must still be emitted")
}
