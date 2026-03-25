//go:build !integration

package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
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
			wantPluginID:    "plugins/advanced-security/skills/secret-scanning",
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
			wantPluginID:    "plugins/foo",
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
			wantNormalized:           []string{"plugins/advanced-security/skills/secret-scanning"},
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
				"plugins/advanced-security/skills/secret-scanning",
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
				"plugins/advanced-security/skills/secret-scanning",
				"plugins/code-review/skills/review",
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
