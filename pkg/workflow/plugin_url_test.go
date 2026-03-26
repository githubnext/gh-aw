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
		name           string
		input          string
		wantPluginSpec string
		wantOK         bool
	}{
		{
			name:           "full GitHub tree URL",
			input:          "https://github.com/github/copilot-plugins/tree/main/plugins/advanced-security/skills/secret-scanning",
			wantPluginSpec: "github/copilot-plugins:advanced-security/skills/secret-scanning",
			wantOK:         true,
		},
		{
			name:           "shallow plugin path",
			input:          "https://github.com/acme/my-plugins/tree/main/tools/linter",
			wantPluginSpec: "acme/my-plugins:tools/linter",
			wantOK:         true,
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
			name:           "http scheme",
			input:          "http://github.com/org/repo/tree/main/plugins/foo",
			wantPluginSpec: "org/repo:foo",
			wantOK:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pluginSpec, ok := parseGitHubPluginURL(tt.input)
			assert.Equal(t, tt.wantOK, ok, "ok mismatch")
			if tt.wantOK {
				assert.Equal(t, tt.wantPluginSpec, pluginSpec, "pluginSpec mismatch")
			}
		})
	}
}

func TestNormalizePlugins(t *testing.T) {
	tests := []struct {
		name           string
		input          []string
		wantNormalized []string
	}{
		{
			name:           "plain names only",
			input:          []string{"plugin-a", "plugin-b"},
			wantNormalized: []string{"plugin-a", "plugin-b"},
		},
		{
			name: "URL entry is converted to OWNER/REPO:PATH spec",
			input: []string{
				"https://github.com/github/copilot-plugins/tree/main/plugins/advanced-security/skills/secret-scanning",
			},
			wantNormalized: []string{
				"github/copilot-plugins:advanced-security/skills/secret-scanning",
			},
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
				"github/copilot-plugins:advanced-security/skills/secret-scanning",
				"another-plugin",
			},
		},
		{
			name:           "empty input",
			input:          nil,
			wantNormalized: []string{},
		},
		{
			name: "two URLs from same repo",
			input: []string{
				"https://github.com/github/copilot-plugins/tree/main/plugins/advanced-security/skills/secret-scanning",
				"https://github.com/github/copilot-plugins/tree/main/plugins/code-review/skills/review",
			},
			wantNormalized: []string{
				"github/copilot-plugins:advanced-security/skills/secret-scanning",
				"github/copilot-plugins:code-review/skills/review",
			},
		},
		{
			name: "plugin@marketplace spec passes through unchanged",
			input: []string{
				"my-plugin@my-marketplace",
			},
			wantNormalized: []string{
				"my-plugin@my-marketplace",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			normalized := normalizePlugins(tt.input)
			assert.Equal(t, tt.wantNormalized, normalized, "normalized plugins mismatch")
		})
	}
}

// TestGitHubTreeURLConvertedToOwnerRepoPath verifies that a GitHub tree URL
// in imports.plugins is converted to the OWNER/REPO:PATH/TO/PLUGIN format
// accepted by the Copilot CLI ("subdirectory in a repository" spec).
func TestGitHubTreeURLConvertedToOwnerRepoPath(t *testing.T) {
	tmpDir := t.TempDir()
	workflowsDir := tmpDir + "/.github/workflows"
	require.NoError(t, os.MkdirAll(workflowsDir, 0755), "create workflows dir")

	content := `---
name: Test GitHub Tree URL Conversion
on:
  workflow_dispatch:
engine: copilot
imports:
  plugins:
    - https://github.com/github/copilot-plugins/tree/main/plugins/advanced-security/skills/secret-scanning
---
Test GitHub tree URL conversion to OWNER/REPO:PATH format.
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
	// Plugin install uses OWNER/REPO:PATH format
	assert.Contains(t, yamlStr,
		"copilot plugin install github/copilot-plugins:advanced-security/skills/secret-scanning",
		"plugin install step must use OWNER/REPO:PATH format")
}
